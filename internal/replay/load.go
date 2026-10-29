package replay

import (
	"sort"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"gorm.io/gorm"
)

// BillingReport summarizes one billing replay run.
type BillingReport struct {
	// Invoices, Payments and Events count the rows that were replayed.
	Invoices int
	Payments int
	Events   int
	// Divergences lists every disagreement between the stored rows and the
	// replayed ledger, ordered by invoice then kind.
	Divergences []Divergence
	// Gaps lists what the evidence cannot tell an operator.
	Gaps []string
}

// ReplayBilling replays every invoice that appears in the tenant's billing
// event trail and returns the divergences it finds.
//
// Only invoices with at least one event are replayed: without a trail there is
// nothing to replay, and a missing trail is itself the gap. The function is
// read-only.
func ReplayBilling(db *gorm.DB, now time.Time) (BillingReport, error) {
	report := BillingReport{
		Gaps: []string{
			"event rows carry no amounts, so payment amounts are read from the payment rows; an edited payment row is not detectable from the trail",
			"void invoices carry no event, so their ledger is not replayed and their balance is not compared",
			"automatic job runs record neither the operator nor the clock, so a replay cannot tell a missed run from a late one",
		},
	}

	var invoiceIDs []int64
	if err := db.Model(&domain.BillingEvent{}).
		Where("invoice_id > 0").
		Distinct().
		Order("invoice_id").
		Pluck("invoice_id", &invoiceIDs).Error; err != nil {
		return report, err
	}
	if len(invoiceIDs) == 0 {
		return report, nil
	}

	var invoices []domain.Invoice
	if err := db.Where("id IN ?", invoiceIDs).Order("id").Find(&invoices).Error; err != nil {
		return report, err
	}
	var payments []domain.Payment
	if err := db.Where("invoice_id IN ?", invoiceIDs).Order("paid_at, id").Find(&payments).Error; err != nil {
		return report, err
	}
	var events []domain.BillingEvent
	if err := db.Where("invoice_id IN ?", invoiceIDs).Order("created_at, id").Find(&events).Error; err != nil {
		return report, err
	}

	paymentsByInvoice := make(map[int64][]domain.Payment, len(invoices))
	for _, payment := range payments {
		paymentsByInvoice[payment.InvoiceID] = append(paymentsByInvoice[payment.InvoiceID], payment)
	}
	eventsByInvoice := make(map[int64][]domain.BillingEvent, len(invoices))
	for _, event := range events {
		eventsByInvoice[event.InvoiceID] = append(eventsByInvoice[event.InvoiceID], event)
	}

	report.Invoices = len(invoices)
	report.Payments = len(payments)
	report.Events = len(events)
	for _, invoice := range invoices {
		_, divergences := ReplayInvoice(invoice, paymentsByInvoice[invoice.ID], eventsByInvoice[invoice.ID], now)
		report.Divergences = append(report.Divergences, divergences...)
	}
	sortDivergences(report.Divergences)
	return report, nil
}

// RadiusReport summarizes one RADIUS session replay run.
type RadiusReport struct {
	// Sessions, Rows and Audits count the evidence that was replayed.
	Sessions int
	Rows     int
	Audits   int
	// Divergences lists every contradiction found per Acct-Session-Id, ordered
	// by session then kind.
	Divergences []Divergence
	// Gaps lists what the evidence cannot tell an operator.
	Gaps []string
}

// ReplayRadius replays every Acct-Session-Id that still has an accounting row,
// a live online row or a queued session action, and returns the contradictions
// it finds.
//
// The function is read-only and does not contact a NAS.
func ReplayRadius(db *gorm.DB, now time.Time) (RadiusReport, error) {
	report := RadiusReport{
		Gaps: []string{
			"Interim-Update packets are not stored as rows, so intra-session counter history cannot be replayed",
			"acct_start_time and acct_stop_time come from the NAS; clock skew between NAS and server is not corrected",
			"session actions and audits are retention-pruned, so an absent action is not evidence that none was attempted",
			"a reused Acct-Session-Id cannot be separated from an earlier session that used the same identifier",
		},
	}

	var rows []domain.RadiusAccounting
	if err := db.Order("acct_start_time, id").Find(&rows).Error; err != nil {
		return report, err
	}
	var online []domain.RadiusOnline
	if err := db.Order("acct_session_id").Find(&online).Error; err != nil {
		return report, err
	}
	var actions []domain.RadiusSessionAction
	if err := db.Order("acct_session_id, id").Find(&actions).Error; err != nil {
		return report, err
	}
	var audits []domain.RadiusSessionActionAudit
	if err := db.Order("acct_session_id, triggered_at, id").Find(&audits).Error; err != nil {
		return report, err
	}

	sessions := make(map[string]struct{})
	rowsBySession := make(map[string][]domain.RadiusAccounting)
	for _, row := range rows {
		if row.AcctSessionId == "" {
			continue
		}
		rowsBySession[row.AcctSessionId] = append(rowsBySession[row.AcctSessionId], row)
		sessions[row.AcctSessionId] = struct{}{}
	}
	onlineBySession := make(map[string]*domain.RadiusOnline, len(online))
	for i := range online {
		if online[i].AcctSessionId == "" {
			continue
		}
		onlineBySession[online[i].AcctSessionId] = &online[i]
		sessions[online[i].AcctSessionId] = struct{}{}
	}
	actionsBySession := make(map[string][]domain.RadiusSessionAction)
	for _, action := range actions {
		if action.AcctSessionID == "" {
			continue
		}
		actionsBySession[action.AcctSessionID] = append(actionsBySession[action.AcctSessionID], action)
		sessions[action.AcctSessionID] = struct{}{}
	}
	auditsBySession := make(map[string][]domain.RadiusSessionActionAudit)
	for _, audit := range audits {
		if audit.AcctSessionID == "" {
			continue
		}
		auditsBySession[audit.AcctSessionID] = append(auditsBySession[audit.AcctSessionID], audit)
	}

	ids := make([]string, 0, len(sessions))
	for id := range sessions {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	report.Rows = len(rows)
	report.Audits = len(audits)
	for _, id := range ids {
		_, divergences := ReplaySession(id, rowsBySession[id], onlineBySession[id], actionsBySession[id], auditsBySession[id], now)
		report.Sessions++
		report.Divergences = append(report.Divergences, divergences...)
	}
	sortDivergences(report.Divergences)
	return report, nil
}
