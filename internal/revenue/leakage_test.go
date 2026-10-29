package revenue

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// auditNow is the clock every fixture is read against: late in a month whose
// billing day for the fixtures has already passed.
var auditNow = time.Date(2026, time.October, 29, 9, 0, 0, 0, time.UTC)

const (
	fixtureTenant = int64(1)
	fixturePrice  = int64(100000)
	// fixtureMonths counts the invoices the clean fixture stores: January to
	// October 2026, each billed on the 5th.
	fixtureMonths = 10
)

func monthStart(month time.Month) time.Time {
	return time.Date(2026, month, 5, 0, 0, 0, 0, time.UTC)
}

// cleanLedger is the negative fixture. One active subscription billed on the 5th
// of every month since January, every invoice paid in full with a matching
// payment row, one enabled RADIUS account with one live session. It violates no
// rule, so the audit must stay silent about it.
func cleanLedger() Input {
	pkg := domain.InternetPackage{
		ID: 1, TenantID: fixtureTenant, Code: "BRONZE", Name: "Bronze 10M",
		Price: fixturePrice, BillingCycle: "monthly", Status: "active",
	}
	sub := domain.Subscription{
		ID: 10, TenantID: fixtureTenant, SubscriptionNo: "SUB-0001", CustomerID: 5,
		PackageID: 1, RadiusUserID: 100, Status: domain.SubscriptionActive,
		StartDate: monthStart(time.January), BillingDay: 5, GraceDays: 3,
	}
	user := domain.RadiusUser{ID: 100, TenantID: fixtureTenant, Username: "sub-0001", Status: "enabled"}
	session := domain.RadiusOnline{
		ID: 1, TenantID: fixtureTenant, Username: "sub-0001", AcctSessionId: "SES-0001",
		AcctStartTime: auditNow.Add(-time.Hour), AcctSessionTime: 3600,
	}

	var invoices []domain.Invoice
	var payments []domain.Payment
	for month := 0; month < fixtureMonths; month++ {
		period := monthStart(time.January).AddDate(0, month, 0)
		invoice := domain.Invoice{
			ID: int64(100 + month), TenantID: fixtureTenant, InvoiceNo: fmt.Sprintf("INV-2026%02d-000001", month+1),
			CustomerID: 5, SubscriptionID: 10, InvoiceDate: period, DueDate: period.AddDate(0, 0, 3),
			PeriodStart: period, PeriodEnd: period.AddDate(0, 1, -1),
			Subtotal: fixturePrice, Total: fixturePrice, PaidAmount: fixturePrice,
			Balance: 0, Status: domain.InvoicePaid,
		}
		invoices = append(invoices, invoice)
		payments = append(payments, domain.Payment{
			ID: int64(200 + month), TenantID: fixtureTenant,
			PaymentNo: fmt.Sprintf("PAY-2026%02d-000001", month+1), CustomerID: 5,
			InvoiceID: invoice.ID, Amount: fixturePrice, Method: "cash",
			PaidAt: period.AddDate(0, 0, 1), Status: domain.PaymentReceived,
		})
	}
	return Input{
		Subscriptions: []domain.Subscription{sub},
		Packages:      []domain.InternetPackage{pkg},
		Invoices:      invoices,
		Payments:      payments,
		Users:         []domain.RadiusUser{user},
		Online:        []domain.RadiusOnline{session},
	}
}

// oneMonthFixture is the same tenant one month old: started and billed on
// October 5th with the invoice paid in full. Because the subscription has only
// reached one billing day, one mutation produces one finding.
func oneMonthFixture() Input {
	input := cleanLedger()
	input.Subscriptions[0].StartDate = monthStart(time.October)
	input.Invoices = input.Invoices[fixtureMonths-1:]
	input.Payments = input.Payments[fixtureMonths-1:]
	return input
}

// unpaidInvoice turns the fixture's October invoice into an unpaid receivable
// whose due date is the given number of days in the past.
func unpaidInvoice(input *Input, status string, daysPastDue int) {
	invoice := &input.Invoices[0]
	invoice.Status = status
	invoice.PaidAmount = 0
	invoice.Balance = invoice.Total
	invoice.DueDate = auditNow.AddDate(0, 0, -daysPastDue)
	input.Payments = nil
}

func findKind(report Report, kind string) []Finding {
	var out []Finding
	for _, finding := range report.Findings {
		if finding.Kind == kind {
			out = append(out, finding)
		}
	}
	return out
}

func assertKind(t *testing.T, report Report, kind string) Finding {
	t.Helper()
	matches := findKind(report, kind)
	require.Len(t, matches, 1, "expected exactly one %s finding, got:\n%s", kind, renderFindings(report))
	return matches[0]
}

func renderFindings(report Report) string {
	var out strings.Builder
	for _, finding := range report.Findings {
		out.WriteString(finding.String())
		out.WriteString("\n")
	}
	return out.String()
}

func TestCleanLedgerProducesNoFindings(t *testing.T) {
	report := Audit(cleanLedger(), auditNow)
	assert.Empty(t, report.Findings, "the clean fixture must not be reported as leakage:\n%s", renderFindings(report))
	assert.Zero(t, report.Totals.Exposure)
	assert.Equal(t, []TenantTotals{{TenantID: fixtureTenant}}, report.ByTenant)
	assert.NotEmpty(t, report.Gaps, "a clean report still states what it cannot see")
}

func TestUnbilledPeriodsAreQuantifiedWithThePackagePrice(t *testing.T) {
	input := cleanLedger()
	// September and October were never invoiced.
	input.Invoices = input.Invoices[:fixtureMonths-2]
	input.Payments = input.Payments[:fixtureMonths-2]

	finding := assertKind(t, Audit(input, auditNow), KindUnbilledPeriod)
	assert.Equal(t, "SUB-0001", finding.Subject)
	assert.Equal(t, ruleGenerateMonthly, finding.Rule)
	assert.True(t, finding.Quantified)
	assert.Equal(t, fixturePrice*2, finding.Exposure)
	evidence := strings.Join(finding.Evidence, " ")
	assert.Contains(t, evidence, "2026-09-05")
	assert.Contains(t, evidence, "2026-10-05")
	assert.Contains(t, evidence, "package BRONZE price=IDR 100000")
}

func TestCurrentMonthIsOnlyExpectedOnceTheBillingDayPassed(t *testing.T) {
	input := cleanLedger()
	input.Invoices = input.Invoices[:fixtureMonths-1]
	input.Payments = input.Payments[:fixtureMonths-1]

	// On the 4th the October invoice is not yet due, so nothing is missing.
	assert.Empty(t, Audit(input, monthStart(time.October).AddDate(0, 0, -1)).Findings)

	report := Audit(input, monthStart(time.October))
	assertKind(t, report, KindUnbilledPeriod)
}

func TestBillingDayOutsideTheGuardIsReportedWithoutAFigure(t *testing.T) {
	for _, day := range []int{0, 29, 31} {
		input := cleanLedger()
		input.Subscriptions[0].BillingDay = day
		// The generator would never have billed this subscription, so no invoice
		// exists at all.
		input.Invoices = nil
		input.Payments = nil

		report := Audit(input, auditNow)
		finding := assertKind(t, report, KindInvalidBillingDay)
		assert.False(t, finding.Quantified)
		assert.Zero(t, finding.Exposure, "a blocked guard is not a proven receivable")
		assert.Empty(t, findKind(report, KindUnbilledPeriod),
			"an unbillable day must not also be counted as missing periods")
		assert.Contains(t, strings.Join(finding.Evidence, " "), fmt.Sprintf("billing_day=%d", day))
	}
}

func TestPastDueInvoiceThatNeverReachedOverdue(t *testing.T) {
	input := oneMonthFixture()
	unpaidInvoice(&input, domain.InvoiceIssued, 1)

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindStaleOverdueStatus)
	assert.Equal(t, ruleProcessOverdue, finding.Rule)
	assert.False(t, finding.Quantified, "the receivable is counted once, by the grace check")
	assert.Empty(t, findKind(report, KindGraceExceededActive), "one day past a three day grace is not yet leakage")
	assert.Len(t, report.Findings, 1)
}

func TestGraceElapsedWhileTheSubscriptionStaysActive(t *testing.T) {
	for _, status := range []string{domain.InvoiceOverdue, domain.InvoiceIssued} {
		input := oneMonthFixture()
		unpaidInvoice(&input, status, 10)

		report := Audit(input, auditNow)
		finding := assertKind(t, report, KindGraceExceededActive)
		assert.Equal(t, input.Invoices[0].InvoiceNo, finding.Subject)
		assert.True(t, finding.Quantified)
		assert.Equal(t, fixturePrice, finding.Exposure)
		if status == domain.InvoiceOverdue {
			assert.Equal(t, ruleSuspendOverdue, finding.Rule)
			assert.Empty(t, findKind(report, KindStaleOverdueStatus))
			assert.Len(t, report.Findings, 1)
			continue
		}
		assert.Equal(t, ruleProcessOverdue, finding.Rule, "the status never advanced, so that is the rule that failed")
		assert.Len(t, findKind(report, KindStaleOverdueStatus), 1)
		assert.Equal(t, fixturePrice, report.Totals.Exposure, "one balance is counted once whatever the status path")
	}
}

func TestGraceStillRunningIsNotLeakage(t *testing.T) {
	input := oneMonthFixture()
	unpaidInvoice(&input, domain.InvoiceOverdue, 2)

	report := Audit(input, auditNow)
	assert.Empty(t, report.Findings, "two days past due inside a three day grace is not an exception:\n%s", renderFindings(report))
}

func TestLedgerThatDisagreesWithItsPaymentRows(t *testing.T) {
	input := oneMonthFixture()
	invoice := &input.Invoices[0]
	invoice.Status = domain.InvoicePartial
	invoice.PaidAmount = fixturePrice / 2
	invoice.Balance = fixturePrice / 2
	invoice.DueDate = auditNow.AddDate(0, 0, 5)
	input.Payments = []domain.Payment{{
		ID: 901, TenantID: fixtureTenant, PaymentNo: "PAY-DOUBLE", CustomerID: 5,
		InvoiceID: invoice.ID, Amount: fixturePrice, Method: "transfer",
		PaidAt: auditNow, Status: domain.PaymentReceived,
	}}

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindLedgerMismatch)
	assert.Equal(t, ruleLedger, finding.Rule)
	assert.True(t, finding.Quantified)
	assert.Equal(t, fixturePrice/2, finding.Exposure)
	assert.Contains(t, strings.Join(finding.Evidence, " "), "sum to 100000")
	assert.Len(t, report.Findings, 1)
}

func TestReversedPaymentIsNotCountedAsMoneyReceived(t *testing.T) {
	input := oneMonthFixture()
	input.Payments = []domain.Payment{
		{
			ID: 902, TenantID: fixtureTenant, PaymentNo: "PAY-KEPT", CustomerID: 5,
			InvoiceID: input.Invoices[0].ID, Amount: fixturePrice, Method: "cash",
			PaidAt: auditNow, Status: domain.PaymentReceived,
		},
		{
			ID: 903, TenantID: fixtureTenant, PaymentNo: "PAY-REV", CustomerID: 5,
			InvoiceID: input.Invoices[0].ID, Amount: fixturePrice, Method: "cash",
			PaidAt: auditNow, Status: domain.PaymentReversed,
		},
	}

	report := Audit(input, auditNow)
	assert.Empty(t, report.Findings, "a reversed payment is not money the invoice holds:\n%s", renderFindings(report))
}

func TestStatusThatContradictsItsBalance(t *testing.T) {
	input := oneMonthFixture()
	invoice := &input.Invoices[0]
	invoice.Status = domain.InvoicePaid
	invoice.PaidAmount = fixturePrice / 2
	invoice.Balance = fixturePrice / 2
	invoice.DueDate = auditNow.AddDate(0, 0, 5)
	input.Payments = []domain.Payment{{
		ID: 903, TenantID: fixtureTenant, PaymentNo: "PAY-HALF", CustomerID: 5,
		InvoiceID: invoice.ID, Amount: fixturePrice / 2, Method: "cash",
		PaidAt: auditNow, Status: domain.PaymentReceived,
	}}

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindStatusBalanceConflict)
	assert.Equal(t, ruleStatusFromBalance, finding.Rule)
	assert.True(t, finding.Quantified)
	assert.Equal(t, fixturePrice/2, finding.Exposure)
	assert.Empty(t, findKind(report, KindLedgerMismatch), "the money rows agree, only the status lies")
}

func TestMoneyBookedOnAVoidInvoice(t *testing.T) {
	input := oneMonthFixture()
	input.Invoices[0].Status = domain.InvoiceVoid
	input.Invoices[0].Balance = 0

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindPaymentOnVoidInvoice)
	assert.Equal(t, ruleVoidClosed, finding.Rule)
	assert.True(t, finding.Quantified)
	assert.Equal(t, fixturePrice, finding.Exposure, "the collected money is a refund liability")
	assert.Contains(t, finding.Note, "void")
}

func TestDuplicateInvoiceForOnePeriod(t *testing.T) {
	input := oneMonthFixture()
	first := input.Invoices[0]
	second := first
	second.ID = 999
	second.InvoiceNo = "INV-202610-000002"
	second.Status = domain.InvoiceIssued
	second.PaidAmount = 0
	second.Balance = fixturePrice
	second.DueDate = auditNow.AddDate(0, 0, 5)
	input.Invoices = append(input.Invoices, second)

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindDuplicatePeriod)
	assert.Equal(t, rulePeriodUnique, finding.Rule)
	assert.Equal(t, second.InvoiceNo, finding.Subject, "the later row is the duplicate")
	assert.Equal(t, fixturePrice, finding.Exposure)
	assert.Contains(t, strings.Join(finding.Evidence, " "), first.InvoiceNo)
	assert.Len(t, report.Findings, 1)
}

func TestBillingSuspensionThatLeftTheAccountUsable(t *testing.T) {
	input := oneMonthFixture()
	input.Online = nil
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = domain.SuspensionBillingOverdue
	unpaidInvoice(&input, domain.InvoiceOverdue, 10)

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindEnabledUserAfterSuspend)
	assert.Equal(t, ruleDisableRadiusUser, finding.Rule)
	assert.Equal(t, "SUB-0001", finding.Subject)
	assert.False(t, finding.Quantified)
	assert.Contains(t, strings.Join(finding.Evidence, " "), "status=enabled")
	assert.Empty(t, findKind(report, KindGraceExceededActive),
		"the subscription is no longer active, so the exception is the access state, not the balance")
}

func TestSessionStillCarriedForABillingSuspendedAccount(t *testing.T) {
	input := oneMonthFixture()
	input.Users[0].Status = "disabled"
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = domain.SuspensionBillingOverdue
	unpaidInvoice(&input, domain.InvoiceOverdue, 10)

	report := Audit(input, auditNow)
	finding := assertKind(t, report, KindOnlineAfterSuspend)
	assert.Equal(t, ruleNoDisconnect, finding.Rule)
	assert.False(t, finding.Quantified)
	evidence := strings.Join(finding.Evidence, " ")
	assert.Contains(t, evidence, "SES-0001")
	assert.Contains(t, evidence, ruleOnlineMeansDelivery.ID)
	assert.Empty(t, findKind(report, KindEnabledUserAfterSuspend), "the account is disabled, only the session is left open")
}

func TestSettledSubscriptionStillSuspendedForBilling(t *testing.T) {
	input := cleanLedger()
	input.Users = nil
	input.Online = nil
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = domain.SuspensionBillingOverdue

	finding := assertKind(t, Audit(input, auditNow), KindStuckReactivation)
	assert.Equal(t, ruleAutoReactivate, finding.Rule)
	assert.False(t, finding.Quantified)
	assert.Equal(t, "SUB-0001", finding.Subject)
}

func TestSuspensionWithAnOutstandingInvoiceIsNotStuck(t *testing.T) {
	input := oneMonthFixture()
	input.Users = nil
	input.Online = nil
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = domain.SuspensionBillingOverdue
	sub.RadiusUserID = 0
	unpaidInvoice(&input, domain.InvoiceOverdue, 10)

	report := Audit(input, auditNow)
	assert.Empty(t, findKind(report, KindStuckReactivation),
		"an unpaid invoice is still outstanding, so the payment that clears it has not happened")
	assert.Empty(t, report.Findings, renderFindings(report))
}

func TestManualSuspensionIsOutsideTheBillingRules(t *testing.T) {
	input := cleanLedger()
	input.Online = nil
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = "operator_request"

	report := Audit(input, auditNow)
	assert.Empty(t, report.Findings, "a manual suspension is never cleared by a payment:\n%s", renderFindings(report))
}

func TestFindingsNeverCrossATenant(t *testing.T) {
	other := cleanLedger()
	for i := range other.Subscriptions {
		other.Subscriptions[i].TenantID = 2
	}
	for i := range other.Packages {
		other.Packages[i].TenantID = 2
	}
	for i := range other.Invoices {
		other.Invoices[i].TenantID = 2
	}
	for i := range other.Payments {
		other.Payments[i].TenantID = 2
	}
	for i := range other.Users {
		other.Users[i].TenantID = 2
	}
	for i := range other.Online {
		other.Online[i].TenantID = 2
	}
	// Tenant 2 never received its October invoice. Both tenants hold the same
	// subscription ID, invoice numbers and username, so a shared index would
	// read tenant 1's rows as tenant 2's cover.
	other.Invoices = other.Invoices[:fixtureMonths-1]
	other.Payments = other.Payments[:fixtureMonths-1]

	tenants := cleanLedger()
	tenants.Subscriptions = append(tenants.Subscriptions, other.Subscriptions...)
	tenants.Packages = append(tenants.Packages, other.Packages...)
	tenants.Invoices = append(tenants.Invoices, other.Invoices...)
	tenants.Payments = append(tenants.Payments, other.Payments...)
	tenants.Users = append(tenants.Users, other.Users...)
	tenants.Online = append(tenants.Online, other.Online...)

	report := Audit(tenants, auditNow)
	require.Len(t, report.Findings, 1, renderFindings(report))
	assert.Equal(t, int64(2), report.Findings[0].TenantID, "only the tenant with the missing period is reported")
	assert.Equal(t, KindUnbilledPeriod, report.Findings[0].Kind)
	assert.Equal(t, []TenantTotals{
		{TenantID: 1, Findings: 0, Exposure: 0},
		{TenantID: 2, Findings: 1, Exposure: fixturePrice},
	}, report.ByTenant)
}

func TestReportIsIndependentOfRowOrder(t *testing.T) {
	input := cleanLedger()
	sub := &input.Subscriptions[0]
	sub.Status = domain.SubscriptionSuspended
	sub.SuspensionReason = domain.SuspensionBillingOverdue
	input.Payments = nil
	for i := range input.Invoices {
		input.Invoices[i].Status = domain.InvoiceIssued
		input.Invoices[i].PaidAmount = 0
		input.Invoices[i].Balance = fixturePrice
		input.Invoices[i].DueDate = auditNow.AddDate(0, 0, -10)
	}

	forward := Audit(input, auditNow)
	reversed := input
	reversed.Subscriptions = reverseSubscriptions(input.Subscriptions)
	reversed.Invoices = reverseInvoices(input.Invoices)
	shuffled := Audit(reversed, auditNow)
	require.NotEmpty(t, forward.Findings)
	assert.True(t, reflect.DeepEqual(forward, shuffled), "row order must not change the report:\n%s---\n%s",
		renderFindings(forward), renderFindings(shuffled))
}

func TestFindingsAreOrderedByTenantKindSubject(t *testing.T) {
	report := Audit(leakageShowcase(), auditNow)
	require.NotEmpty(t, report.Findings)
	for i := 1; i < len(report.Findings); i++ {
		previous, current := report.Findings[i-1], report.Findings[i]
		switch {
		case previous.TenantID != current.TenantID:
			assert.Less(t, previous.TenantID, current.TenantID)
		case previous.Kind != current.Kind:
			assert.Less(t, previous.Kind, current.Kind)
		default:
			assert.LessOrEqual(t, previous.Subject, current.Subject)
		}
	}
}

func TestEveryFindingCitesARuleEvidenceAndLimit(t *testing.T) {
	report := Audit(leakageShowcase(), auditNow)
	require.NotEmpty(t, report.Findings)
	for _, finding := range report.Findings {
		assert.NotEmpty(t, finding.Kind)
		assert.NotEmpty(t, finding.Rule.ID, "a finding must name the rule it violates: %s", finding.String())
		assert.NotEmpty(t, finding.Rule.Function, finding.String())
		assert.NotEmpty(t, finding.Rule.Condition, finding.String())
		assert.NotEmpty(t, finding.Subject)
		assert.NotEmpty(t, finding.Evidence, "a finding without stored values is a claim: %s", finding.String())
		assert.NotEmpty(t, finding.Note, "every claim states its limit: %s", finding.String())
		if finding.Quantified {
			assert.Positive(t, finding.Exposure, "a quantified finding must name a non-zero figure: %s", finding.String())
		} else {
			assert.Zero(t, finding.Exposure, "an unquantified finding must not carry a figure: %s", finding.String())
		}
	}
}

func TestAuditCoversTheKindsTheShowcaseBuilds(t *testing.T) {
	report := Audit(leakageShowcase(), auditNow)
	for _, kind := range []string{
		KindUnbilledPeriod, KindInvalidBillingDay, KindStaleOverdueStatus, KindGraceExceededActive,
		KindEnabledUserAfterSuspend, KindOnlineAfterSuspend, KindPaymentOnVoidInvoice, KindStuckReactivation,
	} {
		assert.NotEmpty(t, findKind(report, kind), "the showcase ledger must produce a %s finding:\n%s",
			kind, renderFindings(report))
	}
	assert.Positive(t, report.Totals.Unquantified, "the catalog deliberately leaves kinds unquantified")
	assert.Equal(t, report.Totals.Findings, len(report.Findings))
}

func TestEveryRuleIsTraceableToRealSource(t *testing.T) {
	root := repoRoot(t)
	for _, rule := range Rules() {
		require.NotEmpty(t, rule.ID, "a rule needs a name an auditor can quote")
		require.NotEmpty(t, rule.Condition, "%s needs the guard it enforces", rule.ID)
		body := readRepoFile(t, root, rule.Source)
		assert.Contains(t, body, "func "+rule.Function+"(", "%s claims %s implements it", rule.ID, rule.Function)
	}
}

func TestRenderMarkdownNamesTheRuleBehindEveryFinding(t *testing.T) {
	report := Audit(leakageShowcase(), auditNow)
	rendered := RenderMarkdown(report)
	assert.Contains(t, rendered, "## Findings per kind")
	assert.Contains(t, rendered, "## Rules cited")
	assert.Contains(t, rendered, "## What the ledger cannot answer")
	for _, rule := range citedRules(report) {
		assert.Contains(t, rendered, rule.ID)
		assert.Contains(t, rendered, rule.Function)
	}
	for _, gap := range report.Gaps {
		assert.Contains(t, rendered, gap)
	}
	assert.Equal(t, rendered, RenderMarkdown(report), "the same report renders the same bytes")
}

func TestRenderMarkdownEscapesPipesInStoredValues(t *testing.T) {
	input := oneMonthFixture()
	input.Invoices[0].InvoiceNo = "INV|2026|10"
	input.Invoices[0].Status = domain.InvoiceVoid
	input.Invoices[0].Balance = 0

	rendered := RenderMarkdown(Audit(input, auditNow))
	assert.Contains(t, rendered, `INV\|2026\|10`)
	assert.NotContains(t, rendered, "| INV|2026|10 |")
}

func TestFindingStringQuotesTheRuleAndTheFigure(t *testing.T) {
	report := Audit(leakageShowcase(), auditNow)
	quantified := assertKind(t, report, KindGraceExceededActive)
	line := quantified.String()
	assert.Contains(t, line, "tenant=1")
	assert.Contains(t, line, "["+quantified.Rule.ID+"]")
	assert.Contains(t, line, "exposure=IDR 100000")

	unquantified := assertKind(t, report, KindStaleOverdueStatus)
	assert.Contains(t, unquantified.String(), "not quantified")
}

// leakageShowcase holds at least one exception of every kind that a single
// tenant ledger can produce, so the report the memo prints covers the catalog.
func leakageShowcase() Input {
	input := oneMonthFixture()
	unpaidInvoice(&input, domain.InvoiceIssued, 10)
	input.Subscriptions = append(input.Subscriptions,
		domain.Subscription{
			ID: 11, TenantID: fixtureTenant, SubscriptionNo: "SUB-0002", CustomerID: 6,
			PackageID: 1, RadiusUserID: 101, Status: domain.SubscriptionActive,
			StartDate: monthStart(time.January), BillingDay: 29, GraceDays: 3,
		},
		domain.Subscription{
			ID: 12, TenantID: fixtureTenant, SubscriptionNo: "SUB-0003", CustomerID: 7,
			PackageID: 1, RadiusUserID: 102, Status: domain.SubscriptionActive,
			StartDate: monthStart(time.September), BillingDay: 5, GraceDays: 3,
		},
		domain.Subscription{
			ID: 13, TenantID: fixtureTenant, SubscriptionNo: "SUB-0004", CustomerID: 8,
			PackageID: 1, RadiusUserID: 103, Status: domain.SubscriptionSuspended,
			StartDate: monthStart(time.June), BillingDay: 5, GraceDays: 3,
			SuspensionReason: domain.SuspensionBillingOverdue,
		},
	)
	input.Users = append(input.Users,
		domain.RadiusUser{ID: 101, TenantID: fixtureTenant, Username: "sub-0002", Status: "enabled"},
		domain.RadiusUser{ID: 102, TenantID: fixtureTenant, Username: "sub-0003", Status: "enabled"},
		domain.RadiusUser{ID: 103, TenantID: fixtureTenant, Username: "sub-0004", Status: "enabled"},
	)
	input.Online = append(input.Online, domain.RadiusOnline{
		ID: 2, TenantID: fixtureTenant, Username: "sub-0004", AcctSessionId: "SES-0004",
		AcctStartTime: auditNow.Add(-2 * time.Hour), AcctSessionTime: 7200,
	})
	// The suspended account's only invoice was voided after it was paid, which
	// leaves a refund liability and no outstanding receivable.
	input.Invoices = append(input.Invoices, domain.Invoice{
		ID: 500, TenantID: fixtureTenant, InvoiceNo: "INV-202606-000009", CustomerID: 8,
		SubscriptionID: 13, InvoiceDate: monthStart(time.June), DueDate: monthStart(time.June).AddDate(0, 0, 3),
		PeriodStart: monthStart(time.June), PeriodEnd: monthStart(time.July).AddDate(0, 0, -1),
		Subtotal: fixturePrice, Total: fixturePrice, PaidAmount: fixturePrice, Balance: 0,
		Status: domain.InvoiceVoid,
	})
	input.Payments = append(input.Payments, domain.Payment{
		ID: 600, TenantID: fixtureTenant, PaymentNo: "PAY-202606-000009", CustomerID: 8,
		InvoiceID: 500, Amount: fixturePrice, Method: "cash",
		PaidAt: monthStart(time.June).AddDate(0, 0, 1), Status: domain.PaymentReceived,
	})
	return input
}

func reverseSubscriptions(rows []domain.Subscription) []domain.Subscription {
	out := append([]domain.Subscription(nil), rows...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func reverseInvoices(rows []domain.Invoice) []domain.Invoice {
	out := append([]domain.Invoice(nil), rows...)
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

// repoRoot walks up from this test file to the module root, so the traceability
// check reads the same sources the rules quote.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, parent, dir, "no go.mod above internal/revenue")
		dir = parent
	}
}

func readRepoFile(t *testing.T, root, relative string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relative))) //nolint:gosec // G304: repository paths quoted by the rule set
	require.NoError(t, err, "rule source %s is not in the repository", relative)
	return string(body)
}
