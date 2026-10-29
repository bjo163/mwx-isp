// Package revenue audits the exceptions a completed billing ledger can still
// contain: periods that were never invoiced, invoices whose status machine lost
// the grace clock, suspensions that left a RADIUS account usable, and payment
// rows that disagree with the invoice they were booked against.
//
// It is an ECO-009 research prototype. It only reads, no product code imports
// it, and every finding names the billing or RADIUS rule that makes it an
// exception, because a revenue claim without a rule is not evidence. Findings
// that have no figure derivable from stored rows say so instead of estimating.
package revenue

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// Source files the rules are read from. A finding quotes one of them so a
// reviewer can re-derive the rule instead of trusting this package.
const (
	sourceBilling = "internal/billing/service.go"
	sourceRadius  = "internal/radiusd/radius.go"
)

// radiusStatusDisabled is the account state a billing suspension writes.
const radiusStatusDisabled = "disabled"

// maxRenderedPeriods caps the missing billing dates quoted in one finding so a
// long gap stays a single readable line.
const maxRenderedPeriods = 6

// Rule is one documented billing or access rule that a finding traces to.
type Rule struct {
	// ID names the rule in the audit report.
	ID string
	// Function is the Go function that implements the rule.
	Function string
	// Condition is the guard that function enforces, as stored in the source.
	Condition string
	// Source is the repository file holding the function.
	Source string
}

// String renders the rule as "id @ function (source)".
func (r Rule) String() string {
	return fmt.Sprintf("%s @ %s (%s)", r.ID, r.Function, r.Source)
}

// The rules this audit applies. Each one is a guard the product already
// enforces; an exception is data that violates the guard, not a new opinion
// about what billing should do.
var (
	ruleGenerateMonthly = Rule{"billing.generate-monthly", "generateMonthlyInvoices",
		"status=active AND 1<=billing_day<=28 AND now.Day()>=billing_day AND start_date<=billing date AND no invoice for (subscription, period_start)", sourceBilling}
	rulePeriodUnique = Rule{"billing.one-invoice-per-period", "generateMonthlyInvoices",
		"the unique index on (tenant_id, subscription_id, period_start) admits one invoice per subscription per period", sourceBilling}
	ruleProcessOverdue = Rule{"billing.process-overdue", "ProcessOverdueInvoices",
		"balance>0 AND status IN (issued,partial,overdue) AND dayAfter(now, due_date) sets status=overdue", sourceBilling}
	ruleSuspendOverdue = Rule{"billing.suspend-overdue", "SuspendOverdueSubscriptions",
		"status=overdue AND dayAfter(now, due_date+grace_days) suspends an active subscription with suspension_reason=billing_overdue", sourceBilling}
	ruleDisableRadiusUser = Rule{"billing.radius-user-disable", "SuspendOverdueSubscriptions",
		"radius_user_id>0 sets radius_user.status=disabled in the suspension transaction", sourceBilling}
	ruleNoDisconnect = Rule{"billing.suspend-no-disconnect", "SuspendOverdueSubscriptions",
		"the function does not send Disconnect-Requests; callers must disconnect matching online sessions", sourceBilling}
	ruleAutoReactivate = Rule{"billing.auto-reactivate", "RecordPayment",
		"a payment that clears the invoice reactivates a billing_overdue suspension when no other invoice of that subscription has balance>0 AND status IN (issued,partial,overdue) AND due_date before today", sourceBilling}
	ruleLedger = Rule{"billing.ledger", "RecordPayment",
		"paid_amount grows by the payment and balance shrinks by the same amount", sourceBilling}
	ruleStatusFromBalance = Rule{"billing.status-from-balance", "RecordPayment",
		"status=paid only when balance reaches zero, otherwise partial", sourceBilling}
	ruleVoidClosed = Rule{"billing.void-is-closed", "RecordPayment",
		"status=void rejects a payment, so a void invoice must not carry money", sourceBilling}
	ruleOnlineMeansDelivery = Rule{"radius.online-means-delivery", "GetNetRadiusOnlineFromRequest",
		"a radius_online row is built from an Accounting-Start packet, so it means the NAS is carrying traffic for that username", sourceRadius}
)

// Rules returns every rule this audit can cite, in the order the billing
// lifecycle applies them.
func Rules() []Rule {
	return []Rule{
		ruleGenerateMonthly, rulePeriodUnique, ruleProcessOverdue, ruleSuspendOverdue,
		ruleDisableRadiusUser, ruleNoDisconnect, ruleAutoReactivate, ruleLedger,
		ruleStatusFromBalance, ruleVoidClosed, ruleOnlineMeansDelivery,
	}
}

// Finding kinds. A kind is the shape of the exception; the rule states why it
// is one.
const (
	// KindUnbilledPeriod means an active subscription passed its billing day
	// without an invoice, so the period was never charged.
	KindUnbilledPeriod = "unbilled_period"
	// KindInvalidBillingDay means the billing day can never satisfy the
	// generator guard, so an active subscription is unbillable.
	KindInvalidBillingDay = "invalid_billing_day"
	// KindDuplicatePeriod means two invoices claim the same subscription
	// period, which is a double charge rather than revenue.
	KindDuplicatePeriod = "duplicate_period"
	// KindStaleOverdueStatus means an invoice is past its due date but was
	// never marked overdue, so the grace clock never started.
	KindStaleOverdueStatus = "stale_overdue_status"
	// KindGraceExceededActive means the grace period elapsed while the
	// subscription stayed active, so service continued on an unpaid balance.
	KindGraceExceededActive = "grace_exceeded_still_active"
	// KindStatusBalanceConflict means the invoice status contradicts its
	// balance, so the receivable a report reads is not the one the ledger holds.
	KindStatusBalanceConflict = "status_balance_conflict"
	// KindLedgerMismatch means the invoice money state disagrees with its
	// payment rows.
	KindLedgerMismatch = "ledger_mismatch"
	// KindPaymentOnVoidInvoice means money is recorded against a void invoice,
	// which is a refund liability rather than revenue.
	KindPaymentOnVoidInvoice = "payment_on_void_invoice"
	// KindEnabledUserAfterSuspend means a billing suspension left the linked
	// RADIUS account usable, so authentication is still possible.
	KindEnabledUserAfterSuspend = "enabled_radius_user_after_suspend"
	// KindOnlineAfterSuspend means the NAS still carries a session for an
	// account suspended for billing.
	KindOnlineAfterSuspend = "online_after_billing_suspend"
	// KindStuckReactivation means a billing suspension remains although no
	// overdue invoice is outstanding.
	KindStuckReactivation = "stuck_reactivation"
)

// Finding is one exception with its evidence.
type Finding struct {
	// TenantID is the tenant the audited rows belong to.
	TenantID int64
	// Kind is one of the Kind* constants.
	Kind string
	// Rule is the billing or access rule that makes this an exception.
	Rule Rule
	// Subject identifies the row or account the exception belongs to.
	Subject string
	// Evidence quotes the stored values the finding was derived from.
	Evidence []string
	// Exposure is the IDR at stake, zero whenever no figure is derivable.
	Exposure int64
	// Quantified reports whether Exposure comes from stored rows.
	Quantified bool
	// Note states the limit of the claim.
	Note string
}

// String renders the finding as one deterministic line.
func (f Finding) String() string {
	quantified := "not quantified"
	if f.Quantified {
		quantified = fmt.Sprintf("IDR %d", f.Exposure)
	}
	return fmt.Sprintf("tenant=%d %s %s [%s] exposure=%s evidence=%s",
		f.TenantID, f.Kind, f.Subject, f.Rule.ID, quantified, strings.Join(f.Evidence, "; "))
}

// Totals summarize an audit report.
type Totals struct {
	// Findings counts the reported exceptions.
	Findings int
	// Exposure sums the quantified findings only. It is money at stake, not
	// money proven lost: each contributing kind states its own basis.
	Exposure int64
	// Unquantified counts findings that deliberately state no money figure.
	Unquantified int
	// ByKind counts findings per exception kind.
	ByKind map[string]int
}

// TenantTotals is one tenant's share of a report.
type TenantTotals struct {
	// TenantID is the tenant the findings belong to.
	TenantID int64
	// Findings counts that tenant's exceptions.
	Findings int
	// Exposure sums that tenant's quantified findings.
	Exposure int64
}

// Report is one audit run.
type Report struct {
	// Findings are ordered by tenant, then kind, then subject.
	Findings []Finding
	// Totals summarizes Findings across all tenants.
	Totals Totals
	// ByTenant splits Totals per tenant, ordered by tenant ID.
	ByTenant []TenantTotals
	// Gaps names what the ledger cannot tell an auditor, so a clean report is
	// not read as proof that nothing is wrong.
	Gaps []string
}

// Input is the ledger slice an audit reads. Every field is a plain projection of
// stored rows, so the same rules apply to a synthetic fixture and to a
// sanitized sample database.
type Input struct {
	// Subscriptions holds the accounts being billed.
	Subscriptions []domain.Subscription
	// Packages holds the priced plan each subscription references.
	Packages []domain.InternetPackage
	// Invoices holds the charges.
	Invoices []domain.Invoice
	// Payments holds the receipts.
	Payments []domain.Payment
	// Users holds the RADIUS accounts linked to subscriptions.
	Users []domain.RadiusUser
	// Online holds the sessions the NAS currently reports.
	Online []domain.RadiusOnline
}

// gaps names what no ledger slice can tell an auditor, so a clean report is not
// read as proof that nothing is wrong.
var gaps = []string{
	"a missing invoice row has no cause in the ledger: the same evidence covers a skipped subscription and a scheduled job that never ran",
	"prices are read from the package row as it stands today, so an unbilled period is valued at the current price, not the price of that month",
	"an invoice whose subscription_id matches no subscription row is audited as money only, never as a service exception",
	"a subscription with an empty start date has no billing history to compare, so it is never counted as unbilled",
	"a radius_online row is current only until its Accounting-Stop arrives, so a stale row and a live session cannot be told apart here",
	"internal/replay owns the event trail: this audit reads stored money state, not the events that produced it",
}

// Audit applies the billing rules to one ledger slice. It never writes, and it
// never estimates beyond the values it was given. Rows are partitioned by tenant
// first, so a finding can only ever quote rows of one tenant.
func Audit(input Input, now time.Time) Report {
	perTenant := map[int64]*Input{}
	tenant := func(tenantID int64) *Input {
		if in, ok := perTenant[tenantID]; ok {
			return in
		}
		in := &Input{}
		perTenant[tenantID] = in
		return in
	}
	for _, row := range input.Subscriptions {
		in := tenant(row.TenantID)
		in.Subscriptions = append(in.Subscriptions, row)
	}
	for _, row := range input.Packages {
		in := tenant(row.TenantID)
		in.Packages = append(in.Packages, row)
	}
	for _, row := range input.Invoices {
		in := tenant(row.TenantID)
		in.Invoices = append(in.Invoices, row)
	}
	for _, row := range input.Payments {
		in := tenant(row.TenantID)
		in.Payments = append(in.Payments, row)
	}
	for _, row := range input.Users {
		in := tenant(row.TenantID)
		in.Users = append(in.Users, row)
	}
	for _, row := range input.Online {
		in := tenant(row.TenantID)
		in.Online = append(in.Online, row)
	}

	tenantIDs := make([]int64, 0, len(perTenant))
	for tenantID := range perTenant {
		tenantIDs = append(tenantIDs, tenantID)
	}
	sort.Slice(tenantIDs, func(i, j int) bool { return tenantIDs[i] < tenantIDs[j] })

	var report Report
	report.Gaps = append(report.Gaps, gaps...)
	for _, tenantID := range tenantIDs {
		findings := auditTenant(*perTenant[tenantID], now)
		totals := TenantTotals{TenantID: tenantID, Findings: len(findings)}
		for _, finding := range findings {
			finding.TenantID = tenantID
			report.Findings = append(report.Findings, finding)
			if finding.Quantified {
				totals.Exposure += finding.Exposure
			}
		}
		report.ByTenant = append(report.ByTenant, totals)
	}

	sort.SliceStable(report.Findings, func(i, j int) bool {
		if report.Findings[i].TenantID != report.Findings[j].TenantID {
			return report.Findings[i].TenantID < report.Findings[j].TenantID
		}
		if report.Findings[i].Kind != report.Findings[j].Kind {
			return report.Findings[i].Kind < report.Findings[j].Kind
		}
		return report.Findings[i].Subject < report.Findings[j].Subject
	})

	report.Totals.Findings = len(report.Findings)
	report.Totals.ByKind = map[string]int{}
	for _, finding := range report.Findings {
		report.Totals.ByKind[finding.Kind]++
		if !finding.Quantified {
			report.Totals.Unquantified++
			continue
		}
		report.Totals.Exposure += finding.Exposure
	}
	return report
}

// index is the lookup set one tenant's audit runs against.
type index struct {
	invoices       []domain.Invoice
	packagesByID   map[int64]domain.InternetPackage
	usersByID      map[int64]domain.RadiusUser
	subscriptions  []domain.Subscription
	subByID        map[int64]domain.Subscription
	invoicesBySub  map[int64][]domain.Invoice
	receivedByID   map[int64]int64
	paidRowsByID   map[int64]int
	sessionsByUser map[string][]domain.RadiusOnline
}

// auditTenant reads one tenant's rows and returns its findings. Row order in
// the input never matters: every slice is sorted before it is indexed, so the
// same rows always produce the same findings.
func auditTenant(input Input, now time.Time) []Finding {
	idx := index{
		subscriptions:  sortedSubscriptions(input.Subscriptions),
		invoices:       sortedInvoices(input.Invoices),
		packagesByID:   map[int64]domain.InternetPackage{},
		usersByID:      map[int64]domain.RadiusUser{},
		subByID:        map[int64]domain.Subscription{},
		invoicesBySub:  map[int64][]domain.Invoice{},
		receivedByID:   map[int64]int64{},
		paidRowsByID:   map[int64]int{},
		sessionsByUser: map[string][]domain.RadiusOnline{},
	}
	for _, pkg := range input.Packages {
		idx.packagesByID[pkg.ID] = pkg
	}
	for _, user := range sortedUsers(input.Users) {
		idx.usersByID[user.ID] = user
	}
	for _, sub := range idx.subscriptions {
		idx.subByID[sub.ID] = sub
	}
	for _, invoice := range idx.invoices {
		idx.invoicesBySub[invoice.SubscriptionID] = append(idx.invoicesBySub[invoice.SubscriptionID], invoice)
	}
	for _, payment := range input.Payments {
		if payment.Status != domain.PaymentReceived {
			continue
		}
		idx.receivedByID[payment.InvoiceID] += payment.Amount
		idx.paidRowsByID[payment.InvoiceID]++
	}
	for _, session := range sortedSessions(input.Online) {
		idx.sessionsByUser[session.Username] = append(idx.sessionsByUser[session.Username], session)
	}

	var findings []Finding
	findings = append(findings, auditLedger(idx)...)
	findings = append(findings, auditInvoiceClock(idx, now)...)
	findings = append(findings, auditSubscriptions(idx, now)...)
	findings = append(findings, auditAccessState(idx)...)
	return findings
}

// auditLedger checks the money state of stored invoices against their payment
// rows and against the one-invoice-per-period guard.
func auditLedger(idx index) []Finding {
	var findings []Finding
	seenPeriod := map[[4]int64]domain.Invoice{}
	for _, invoice := range idx.invoices {
		received := idx.receivedByID[invoice.ID]
		rows := idx.paidRowsByID[invoice.ID]

		if invoice.PaidAmount != received || invoice.Balance != invoice.Total-received {
			findings = append(findings, Finding{
				Kind:    KindLedgerMismatch,
				Rule:    ruleLedger,
				Subject: invoice.InvoiceNo,
				Evidence: []string{
					fmt.Sprintf("stored paid_amount=%d balance=%d total=%d", invoice.PaidAmount, invoice.Balance, invoice.Total),
					fmt.Sprintf("%d received payment row(s) sum to %d", rows, received),
				},
				Exposure:   absInt64(invoice.PaidAmount - received),
				Quantified: true,
				Note:       "the disagreement is the figure at stake; internal/replay names the event trail that produced it",
			})
		}
		if invoice.Status == domain.InvoiceVoid && (invoice.PaidAmount > 0 || received > 0) {
			liability := invoice.PaidAmount
			if received > liability {
				liability = received
			}
			findings = append(findings, Finding{
				Kind:    KindPaymentOnVoidInvoice,
				Rule:    ruleVoidClosed,
				Subject: invoice.InvoiceNo,
				Evidence: []string{
					fmt.Sprintf("status=void paid_amount=%d balance=%d", invoice.PaidAmount, invoice.Balance),
					fmt.Sprintf("%d received payment row(s) sum to %d", rows, received),
				},
				Exposure:   liability,
				Quantified: liability > 0,
				Note:       "a void invoice collects nothing, so the money on it is a refund liability; no product path sets status=void today, so this row came from a legacy or repaired ledger",
			})
		}
		if statusBalanceConflict(invoice) {
			findings = append(findings, Finding{
				Kind:    KindStatusBalanceConflict,
				Rule:    ruleStatusFromBalance,
				Subject: invoice.InvoiceNo,
				Evidence: []string{
					fmt.Sprintf("status=%s balance=%d total=%d paid_amount=%d", invoice.Status, invoice.Balance, invoice.Total, invoice.PaidAmount),
				},
				Exposure:   absInt64(invoice.Balance),
				Quantified: invoice.Balance != 0,
				Note:       "reports read the status, the ledger reads the balance; the two disagree, so either the receivable or the collection is overstated",
			})
		}

		key := periodKey(invoice.PeriodStart, invoice.SubscriptionID)
		if first, ok := seenPeriod[key]; ok {
			findings = append(findings, Finding{
				Kind:    KindDuplicatePeriod,
				Rule:    rulePeriodUnique,
				Subject: invoice.InvoiceNo,
				Evidence: []string{
					fmt.Sprintf("period_start=%s already billed by %s", invoice.PeriodStart.Format("2006-01-02"), first.InvoiceNo),
					fmt.Sprintf("this row charges total=%d balance=%d", invoice.Total, invoice.Balance),
				},
				Exposure:   invoice.Total,
				Quantified: invoice.Total > 0,
				Note:       "the current unique index rejects this row, so it predates the index or was written by a repair; the duplicate charge is a refund liability",
			})
		} else {
			seenPeriod[key] = invoice
		}
	}
	return findings
}

// statusBalanceConflict reports an invoice whose status and balance contradict
// the rule RecordPayment uses to set them.
func statusBalanceConflict(invoice domain.Invoice) bool {
	if invoice.Status == domain.InvoicePaid {
		return invoice.Balance != 0
	}
	switch invoice.Status {
	case domain.InvoiceIssued, domain.InvoicePartial, domain.InvoiceOverdue:
		return invoice.Balance <= 0
	default:
		return false
	}
}

// auditInvoiceClock checks the due-date and grace guards that move an invoice
// to overdue and an unpaid subscription to suspended.
func auditInvoiceClock(idx index, now time.Time) []Finding {
	var findings []Finding
	for _, invoice := range idx.invoices {
		if invoice.Balance <= 0 {
			continue
		}
		pastDue := dayAfter(now, invoice.DueDate)
		if pastDue && (invoice.Status == domain.InvoiceIssued || invoice.Status == domain.InvoicePartial) {
			findings = append(findings, Finding{
				Kind:    KindStaleOverdueStatus,
				Rule:    ruleProcessOverdue,
				Subject: invoice.InvoiceNo,
				Evidence: []string{
					fmt.Sprintf("status=%s balance=%d due_date=%s now=%s", invoice.Status, invoice.Balance,
						invoice.DueDate.Format(time.RFC3339), now.Format(time.RFC3339)),
				},
				Note: "no figure here: the receivable is counted once by grace_exceeded_still_active once the grace clock is read",
			})
		}
		sub, ok := idx.subByID[invoice.SubscriptionID]
		if !ok || sub.Status != domain.SubscriptionActive || !unpaidStatus(invoice.Status) {
			continue
		}
		graceEnd := invoice.DueDate.AddDate(0, 0, sub.GraceDays)
		if !pastDue || !dayAfter(now, graceEnd) {
			continue
		}
		rule := ruleSuspendOverdue
		note := "the subscription kept service after grace, so this balance is collectible but was not collected"
		if invoice.Status != domain.InvoiceOverdue {
			rule = ruleProcessOverdue
			note = "the status never reached overdue, so the suspend guard never saw this invoice; the balance is still the collectible amount"
		}
		findings = append(findings, Finding{
			Kind:    KindGraceExceededActive,
			Rule:    rule,
			Subject: invoice.InvoiceNo,
			Evidence: []string{
				fmt.Sprintf("subscription=%s status=active grace_days=%d due_date=%s now=%s",
					subjectOf(sub, invoice.SubscriptionID), sub.GraceDays,
					invoice.DueDate.Format("2006-01-02"), now.Format("2006-01-02")),
				fmt.Sprintf("invoice status=%s balance=%d", invoice.Status, invoice.Balance),
			},
			Exposure:   invoice.Balance,
			Quantified: true,
			Note:       note,
		})
	}
	return findings
}

// auditSubscriptions checks each active subscription against the invoice
// generator's guards.
func auditSubscriptions(idx index, now time.Time) []Finding {
	var findings []Finding
	startOfToday := startOfDay(now)
	for _, sub := range idx.subscriptions {
		if sub.Status != domain.SubscriptionActive {
			continue
		}
		rows := idx.invoicesBySub[sub.ID]
		pkg, hasPackage := idx.packagesByID[sub.PackageID]

		if sub.BillingDay < 1 || sub.BillingDay > 28 {
			findings = append(findings, Finding{
				Kind:    KindInvalidBillingDay,
				Rule:    ruleGenerateMonthly,
				Subject: subjectOf(sub, sub.ID),
				Evidence: []string{
					fmt.Sprintf("billing_day=%d start_date=%s status=active", sub.BillingDay, sub.StartDate.Format("2006-01-02")),
					packagePrice(pkg, hasPackage),
				},
				Note: "no figure: the guard blocks generation every month, but nothing in the ledger records a charge that was due, so the count of lost periods is not derivable",
			})
			continue
		}

		missing := missingBillingPeriods(sub, rows, now)
		if len(missing) > 0 {
			exposure := int64(0)
			if hasPackage {
				exposure = pkg.Price * int64(len(missing))
			}
			findings = append(findings, Finding{
				Kind:    KindUnbilledPeriod,
				Rule:    ruleGenerateMonthly,
				Subject: subjectOf(sub, sub.ID),
				Evidence: append(renderPeriods(missing),
					fmt.Sprintf("%d invoice row(s) present for this subscription", len(rows)),
					packagePrice(pkg, hasPackage)),
				Exposure:   exposure,
				Quantified: hasPackage && pkg.Price > 0,
				Note:       "the figure uses the package price stored today, not the price at each period; a manually created invoice for a different period_start does not satisfy this guard but does collect money, and one absent month can mean the scheduled job did not run rather than this subscription being skipped",
			})
		}
	}

	for _, sub := range idx.subscriptions {
		if sub.Status != domain.SubscriptionSuspended || sub.SuspensionReason != domain.SuspensionBillingOverdue {
			continue
		}
		outstanding := 0
		for _, invoice := range idx.invoicesBySub[sub.ID] {
			if unpaidStatus(invoice.Status) && invoice.Balance > 0 && invoice.DueDate.Before(startOfToday) {
				outstanding++
			}
		}
		if outstanding > 0 {
			continue
		}
		findings = append(findings, Finding{
			Kind:    KindStuckReactivation,
			Rule:    ruleAutoReactivate,
			Subject: subjectOf(sub, sub.ID),
			Evidence: []string{
				fmt.Sprintf("status=suspended suspension_reason=%s", sub.SuspensionReason),
				fmt.Sprintf("%d invoice row(s), none with balance>0 and due_date before %s",
					len(idx.invoicesBySub[sub.ID]), startOfToday.Format("2006-01-02")),
			},
			Note: "no figure: the customer is paying and still suspended, so the loss is a support cost, not a receivable",
		})
	}
	return findings
}

// auditAccessState checks the RADIUS side of a billing suspension: the account
// status written in the suspension transaction and the sessions that suspension
// leaves connected.
func auditAccessState(idx index) []Finding {
	var findings []Finding
	for _, sub := range idx.subscriptions {
		if sub.Status != domain.SubscriptionSuspended || sub.SuspensionReason != domain.SuspensionBillingOverdue {
			continue
		}
		if sub.RadiusUserID <= 0 {
			continue
		}
		user, ok := idx.usersByID[sub.RadiusUserID]
		if !ok {
			continue
		}
		if user.Status != radiusStatusDisabled {
			findings = append(findings, Finding{
				Kind:    KindEnabledUserAfterSuspend,
				Rule:    ruleDisableRadiusUser,
				Subject: subjectOf(sub, sub.ID),
				Evidence: []string{
					fmt.Sprintf("radius_user=%s status=%s", user.Username, user.Status),
					fmt.Sprintf("subscription suspended with reason=%s", sub.SuspensionReason),
				},
				Note: "no figure: authentication is still possible for an account the ledger has stopped paying for, but the volume it consumes is not stored on the subscription",
			})
		}
		sessions := idx.sessionsByUser[user.Username]
		if len(sessions) == 0 {
			continue
		}
		first := sessions[0]
		seconds := 0
		for _, session := range sessions {
			seconds += session.AcctSessionTime
		}
		findings = append(findings, Finding{
			Kind:    KindOnlineAfterSuspend,
			Rule:    ruleNoDisconnect,
			Subject: subjectOf(sub, sub.ID),
			Evidence: []string{
				fmt.Sprintf("%d radius_online row(s) for %s", len(sessions), user.Username),
				fmt.Sprintf("session %s started %s reports %d second(s) of service",
					sessionID(first), first.AcctStartTime.Format(time.RFC3339), seconds),
				fmt.Sprintf("basis: %s", ruleOnlineMeansDelivery),
			},
			Note: "no figure: the price of those seconds is not stored on the session, and a row whose Accounting-Stop never arrived looks identical to a live session",
		})
	}
	return findings
}

// missingBillingPeriods lists the billing dates an active subscription should
// already have an invoice for, using the generator's own guards: the month must
// have reached its billing day, the billing date must not precede the start
// date, and one invoice per period_start satisfies the period.
func missingBillingPeriods(sub domain.Subscription, invoices []domain.Invoice, now time.Time) []time.Time {
	if sub.StartDate.IsZero() {
		return nil
	}
	have := map[[4]int64]bool{}
	for _, invoice := range invoices {
		have[periodKey(invoice.PeriodStart, sub.ID)] = true
	}
	year, month, _ := startOfDay(sub.StartDate).Date()
	last := startOfDay(now)
	var missing []time.Time
	for {
		billingDate := time.Date(year, month, sub.BillingDay, 0, 0, 0, 0, last.Location())
		if dayAfter(billingDate, last) {
			break
		}
		if !dayAfter(sub.StartDate, billingDate) && !have[periodKey(billingDate, sub.ID)] {
			missing = append(missing, billingDate)
		}
		if !billingDate.Before(last) {
			break
		}
		year, month = addMonth(year, month)
	}
	return missing
}

// addMonth returns the next month, wrapping the year.
func addMonth(year int, month time.Month) (int, time.Month) {
	if month == time.December {
		return year + 1, time.January
	}
	return year, month + 1
}

// renderPeriods quotes up to maxRenderedPeriods billing dates, then the count
// of the remainder.
func renderPeriods(periods []time.Time) []string {
	quoted := periods
	suffix := []string(nil)
	if len(periods) > maxRenderedPeriods {
		quoted = periods[:maxRenderedPeriods]
		suffix = []string{fmt.Sprintf("and %d more period(s)", len(periods)-maxRenderedPeriods)}
	}
	evidence := make([]string, 0, len(quoted)+len(suffix)+1)
	for _, period := range quoted {
		evidence = append(evidence, "no invoice for period_start "+period.Format("2006-01-02"))
	}
	evidence = append(evidence, suffix...)
	return evidence
}

// packagePrice states the charge the missing periods should have produced.
func packagePrice(pkg domain.InternetPackage, present bool) string {
	if !present {
		return "package row missing - no price to compare"
	}
	return fmt.Sprintf("package %s price=IDR %d cycle=%s", pkg.Code, pkg.Price, pkg.BillingCycle)
}

// unpaidStatus reports whether an invoice status can still hold a receivable.
func unpaidStatus(status string) bool {
	switch status {
	case domain.InvoiceIssued, domain.InvoicePartial, domain.InvoiceOverdue:
		return true
	default:
		return false
	}
}

// subjectOf names a subscription for a finding, falling back to its ID when the
// fixture left the human-readable number empty.
func subjectOf(sub domain.Subscription, id int64) string {
	if sub.SubscriptionNo != "" {
		return sub.SubscriptionNo
	}
	return fmt.Sprintf("subscription#%d", id)
}

// sessionID names a session for evidence.
func sessionID(session domain.RadiusOnline) string {
	if session.AcctSessionId != "" {
		return session.AcctSessionId
	}
	return fmt.Sprintf("session#%d", session.ID)
}

// periodKey is the generator's uniqueness key: one subscription, one period
// start day.
func periodKey(periodStart time.Time, subscriptionID int64) [4]int64 {
	year, month, day := periodStart.Date()
	return [4]int64{subscriptionID, int64(year), int64(month), int64(day)}
}

// dayAfter mirrors billing's dateAfter: it compares calendar dates, so a
// same-day difference in clock time never changes a guard.
func dayAfter(a, b time.Time) bool {
	return startOfDay(a).After(startOfDay(b))
}

// startOfDay truncates a timestamp to midnight in its own location.
func startOfDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}

// absInt64 returns the magnitude of a money difference.
func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func sortedSubscriptions(rows []domain.Subscription) []domain.Subscription {
	out := append([]domain.Subscription(nil), rows...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedInvoices(rows []domain.Invoice) []domain.Invoice {
	out := append([]domain.Invoice(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].InvoiceNo != out[j].InvoiceNo {
			return out[i].InvoiceNo < out[j].InvoiceNo
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func sortedUsers(rows []domain.RadiusUser) []domain.RadiusUser {
	out := append([]domain.RadiusUser(nil), rows...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func sortedSessions(rows []domain.RadiusOnline) []domain.RadiusOnline {
	out := append([]domain.RadiusOnline(nil), rows...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].AcctSessionId != out[j].AcctSessionId {
			return out[i].AcctSessionId < out[j].AcctSessionId
		}
		return out[i].ID < out[j].ID
	})
	return out
}
