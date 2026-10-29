// Package replay reconstructs billing and RADIUS session state from the rows
// MWX-ISP already stores, so an operator can reproduce how an invoice or a
// session reached its current state without re-running the original actions.
//
// This is the bounded FUT-022 prototype: no application package imports it, it
// never writes, and it reads only rows that already exist in the tenant
// database. The replay is deterministic for the same input rows, so a
// disagreement between the replayed state and the stored row is a concrete
// reproduction step rather than an opinion.
//
// Two evidence sets are replayed:
//
//   - Billing: the ordered isp_billing_event trail plus the payment rows of one
//     invoice. Folding payment_received and payment_reversed events in trail
//     order yields the paid amount, balance and status the invoice should hold.
//   - RADIUS: radius_accounting rows, the live radius_online row, queued
//     radius_session_action rows and radius_session_action_audit rows for one
//     Acct-Session-Id. Replaying them yields the session lifetime and the
//     actions that were supposed to end it.
package replay

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
)

// Divergence kinds reported by a billing replay.
const (
	// DivergencePaidAmount reports that the stored paid amount differs from the
	// amount replayed from the event trail.
	DivergencePaidAmount = "paid_amount_mismatch"
	// DivergenceBalance reports that the stored balance differs from total minus
	// the replayed paid amount.
	DivergenceBalance = "balance_mismatch"
	// DivergenceStatus reports that the stored invoice state (paid, partial,
	// unpaid) differs from the state implied by the replayed balance. The
	// issued/overdue distinction depends on scheduler timing and is not compared.
	DivergenceStatus = "status_mismatch"
	// DivergenceMissingEvent reports a payment row whose state has no matching
	// event in the trail.
	DivergenceMissingEvent = "missing_payment_event"
	// DivergenceUnknownPayment reports an event that names a payment number with
	// no payment row.
	DivergenceUnknownPayment = "unknown_payment_event"
	// DivergenceReversalOrder reports a reversal event that precedes the
	// matching payment_received event, or a payment reversed while still marked
	// received in its row.
	DivergenceReversalOrder = "reversal_without_payment"
	// DivergenceDuplicateEvent reports two payment_received events for one
	// payment.
	DivergenceDuplicateEvent = "duplicate_payment_event"
	// DivergenceNegativePaid reports a replayed paid amount below zero.
	DivergenceNegativePaid = "negative_paid_amount"
	// DivergenceEventOrder reports an event whose timestamp predates the event
	// before it.
	DivergenceEventOrder = "event_order"
)

// Divergence kinds reported by a RADIUS session replay.
const (
	// DivergenceStopBeforeStart reports a row whose stop time precedes its start
	// time, which a NAS clock step or a malformed packet can produce.
	DivergenceStopBeforeStart = "stop_before_start"
	// DivergenceSessionTime reports a final counter that disagrees with the
	// start/stop span beyond the tolerance below.
	DivergenceSessionTime = "session_time_mismatch"
	// DivergenceCounterRegression reports accounting counters that decrease
	// between rows of the same Acct-Session-Id.
	DivergenceCounterRegression = "counter_regression"
	// DivergenceSessionOverlap reports a row that starts before an earlier row
	// for the same Acct-Session-Id stopped, or while that row is still open.
	DivergenceSessionOverlap = "overlapping_session_lifetime"
	// DivergenceOrphanOpen reports an open accounting row whose online row is
	// gone, which means the Accounting-Stop never reached the server.
	DivergenceOrphanOpen = "orphan_open_session"
	// DivergenceClosedWithOnline reports a stopped accounting row that still has
	// an online row.
	DivergenceClosedWithOnline = "closed_accounting_with_online_row"
	// DivergenceOnlineWithoutAccounting reports an online row with no accounting
	// row at all, which means the Accounting-Start row was never persisted or
	// has been pruned.
	DivergenceOnlineWithoutAccounting = "online_without_accounting"
	// DivergenceStaleOpen reports a session that has stayed open past
	// maxOpenSession.
	DivergenceStaleOpen = "stale_open_session"
	// DivergenceDisconnectPending reports a queued disconnect that has not
	// succeeded while the session is still online.
	DivergenceDisconnectPending = "disconnect_not_delivered"
	// DivergenceDisconnectNoEffect reports a disconnect that reported success
	// while the session is still online.
	DivergenceDisconnectNoEffect = "disconnect_did_not_end_session"
)

const (
	// sessionTimeTolerance absorbs whole-second rounding and packet transit
	// between the NAS-reported session time and stop minus start.
	sessionTimeTolerance = 90 * time.Second
	// maxOpenSession is how long a row may stay open before a replay reports it
	// as stale rather than as a live session.
	maxOpenSession = 30 * 24 * time.Hour
)

// Divergence is one disagreement between stored rows and replayed state.
type Divergence struct {
	// Subject is the invoice number or Acct-Session-Id the finding belongs to.
	Subject string
	// Kind is one of the Divergence* constants.
	Kind string
	// Detail explains the concrete values that disagree.
	Detail string
}

// String formats the finding for a support note or test failure.
func (d Divergence) String() string {
	return fmt.Sprintf("%s: %s: %s", d.Subject, d.Kind, d.Detail)
}

// InvoiceState is an invoice ledger replayed from its event trail.
type InvoiceState struct {
	// InvoiceNo identifies the invoice.
	InvoiceNo string
	// Total is the invoice total taken from the invoice row.
	Total int64
	// Paid is the sum of received payments minus reversed payments in trail
	// order.
	Paid int64
	// Balance is Total minus Paid.
	Balance int64
	// Status is the state implied by the replayed paid amount and due date.
	Status string
	// Payments and Events count the evidence that was replayed.
	Payments int
	Events   int
}

// ReplayInvoice folds the billing events of one invoice in trail order and
// compares the result with the stored invoice row.
//
// Payment amounts are read from the payment rows by payment number: the event
// trail records which payment moved and in which direction, while the payment
// row is the durable record of how much. Events for other invoice types
// (invoice_generated, invoice_overdue, subscription_suspended and so on) are
// counted but do not move money.
//
// A void invoice is not compared, because voiding has no event and its stored
// balance is not maintained by the payment trail.
func ReplayInvoice(invoice domain.Invoice, payments []domain.Payment, events []domain.BillingEvent, now time.Time) (InvoiceState, []Divergence) {
	state := InvoiceState{
		InvoiceNo: invoice.InvoiceNo,
		Total:     invoice.Total,
		Payments:  len(payments),
		Events:    len(events),
	}
	var divergences []Divergence
	add := func(kind, detail string) {
		divergences = append(divergences, Divergence{Subject: invoice.InvoiceNo, Kind: kind, Detail: detail})
	}

	byNumber := make(map[string]domain.Payment, len(payments))
	for _, payment := range payments {
		byNumber[payment.PaymentNo] = payment
	}

	ordered := make([]domain.BillingEvent, len(events))
	copy(ordered, events)
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
		}
		return ordered[i].ID < ordered[j].ID
	})

	received := make(map[string]bool, len(payments))
	reversed := make(map[string]bool, len(payments))
	for i, event := range ordered {
		// IDs are time-ordered serials. A decrease means the row was inserted
		// after a later timestamp, which a clock step or a backdated import
		// produces.
		if i > 0 && event.ID < ordered[i-1].ID {
			add(DivergenceEventOrder, fmt.Sprintf("%s event %d at %s follows event %d at %s",
				event.Type, event.ID, event.CreatedAt.Format(time.RFC3339),
				ordered[i-1].ID, ordered[i-1].CreatedAt.Format(time.RFC3339)))
		}
		number := eventReference(event.Description)
		switch event.Type {
		case "payment_received":
			payment, ok := byNumber[number]
			if !ok {
				add(DivergenceUnknownPayment, fmt.Sprintf("payment_received event references %q, which has no payment row", number))
				continue
			}
			if received[number] {
				add(DivergenceDuplicateEvent, fmt.Sprintf("%s is recorded as received more than once", number))
				continue
			}
			received[number] = true
			state.Paid += payment.Amount
		case "payment_reversed":
			payment, ok := byNumber[number]
			if !ok {
				add(DivergenceUnknownPayment, fmt.Sprintf("payment_reversed event references %q, which has no payment row", number))
				continue
			}
			if !received[number] || reversed[number] {
				add(DivergenceReversalOrder, fmt.Sprintf("%s is reversed without an unreversed payment_received event", number))
				continue
			}
			reversed[number] = true
			state.Paid -= payment.Amount
		}
	}

	state.Balance = state.Total - state.Paid
	state.Status = deriveInvoiceStatus(state.Paid, state.Total, invoice.DueDate, now)
	if state.Paid < 0 {
		add(DivergenceNegativePaid, fmt.Sprintf("the event trail leaves a paid amount of %d", state.Paid))
	}

	for _, payment := range payments {
		if payment.Status == domain.PaymentReceived && !received[payment.PaymentNo] {
			add(DivergenceMissingEvent, fmt.Sprintf("%s is received but has no payment_received event", payment.PaymentNo))
		}
		if payment.Status == domain.PaymentReversed && !reversed[payment.PaymentNo] {
			add(DivergenceMissingEvent, fmt.Sprintf("%s is reversed but has no payment_reversed event", payment.PaymentNo))
		}
		if payment.Status == domain.PaymentReceived && reversed[payment.PaymentNo] {
			add(DivergenceReversalOrder, fmt.Sprintf("%s has a reversal event but its row is still marked received", payment.PaymentNo))
		}
	}

	if invoice.Status != domain.InvoiceVoid {
		if invoice.PaidAmount != state.Paid {
			add(DivergencePaidAmount, fmt.Sprintf("stored paid amount is %d, the trail replays %d", invoice.PaidAmount, state.Paid))
		}
		if invoice.Balance != state.Balance {
			add(DivergenceBalance, fmt.Sprintf("stored balance is %d, total minus replayed paid is %d", invoice.Balance, state.Balance))
		}
		if stored, derived := statusClass(invoice.Status), statusClass(state.Status); stored != derived {
			add(DivergenceStatus, fmt.Sprintf("stored status is %s (%s), the replayed balance implies %s (%s)",
				invoice.Status, stored, state.Status, derived))
		}
	}

	sortDivergences(divergences)
	return state, divergences
}

// deriveInvoiceStatus returns the invoice state implied by a replayed paid
// amount. The overdue check compares against the start of today so a reversal on
// the due date itself does not immediately mark the invoice overdue, matching
// billing.GenerateMonthlyInvoices and billing.ReversePayment.
func deriveInvoiceStatus(paid, total int64, due, now time.Time) string {
	if paid <= 0 {
		startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		if due.Before(startOfToday) {
			return domain.InvoiceOverdue
		}
		return domain.InvoiceIssued
	}
	if paid >= total {
		return domain.InvoicePaid
	}
	return domain.InvoicePartial
}

// statusClass collapses statuses whose difference is scheduler timing rather
// than ledger state.
func statusClass(status string) string {
	switch status {
	case domain.InvoicePaid:
		return "paid"
	case domain.InvoicePartial:
		return "partial"
	case domain.InvoiceVoid:
		return "void"
	default:
		return "unpaid"
	}
}

// eventReference returns the payment number carried in an event description.
// payment_received descriptions are the payment number; payment_reversed
// descriptions start with it and continue with the operator and reason.
func eventReference(description string) string {
	fields := strings.Fields(description)
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

// SessionState is the lifecycle of one Acct-Session-Id replayed from its rows.
type SessionState struct {
	// AcctSessionID identifies the NAS session.
	AcctSessionID string
	// Username is the first non-empty username across the rows.
	Username string
	// Start is the earliest NAS-reported start time.
	Start time.Time
	// Stop is the latest NAS-reported stop time.
	Stop time.Time
	// Open reports whether the session is considered live: an online row exists
	// or the newest accounting row has no stop time.
	Open bool
	// InputOctets, OutputOctets and SessionTime are the counters of the newest
	// row.
	InputOctets  int64
	OutputOctets int64
	SessionTime  int
	// Rows, Actions and Audits count the evidence that was replayed.
	Rows    int
	Actions int
	Audits  int
}

// ReplaySession orders the accounting rows of one Acct-Session-Id by NAS start
// time and re-derives the session lifetime, then checks the live row, the queued
// disconnect actions and the operator audit trail for contradictions.
//
// Timestamps come from the NAS and are not corrected for clock skew, but an
// internal contradiction between them (stop before start, a stop that predates
// an earlier row's stop) is still reproducible evidence. Counters are compared
// across rows because Interim-Update packets are not stored as separate rows.
func ReplaySession(sessionID string, rows []domain.RadiusAccounting, online *domain.RadiusOnline, actions []domain.RadiusSessionAction, audits []domain.RadiusSessionActionAudit, now time.Time) (SessionState, []Divergence) {
	state := SessionState{
		AcctSessionID: sessionID,
		Rows:          len(rows),
		Actions:       len(actions),
		Audits:        len(audits),
	}
	var divergences []Divergence
	add := func(kind, detail string) {
		divergences = append(divergences, Divergence{Subject: sessionID, Kind: kind, Detail: detail})
	}

	ordered := make([]domain.RadiusAccounting, len(rows))
	copy(ordered, rows)
	sort.SliceStable(ordered, func(i, j int) bool {
		left, right := rowTime(ordered[i]), rowTime(ordered[j])
		if !left.Equal(right) {
			return left.Before(right)
		}
		return ordered[i].ID < ordered[j].ID
	})

	for i, row := range ordered {
		if state.Username == "" {
			state.Username = row.Username
		}
		if !row.AcctStartTime.IsZero() {
			if state.Start.IsZero() {
				state.Start = row.AcctStartTime
			}
			if !row.AcctStopTime.IsZero() {
				if row.AcctStopTime.Before(row.AcctStartTime) {
					add(DivergenceStopBeforeStart, fmt.Sprintf("row %d stops at %s before it starts at %s",
						row.ID, row.AcctStopTime.Format(time.RFC3339), row.AcctStartTime.Format(time.RFC3339)))
				} else {
					elapsed := row.AcctStopTime.Sub(row.AcctStartTime)
					reported := time.Duration(row.AcctSessionTime) * time.Second
					if delta := elapsed - reported; delta > sessionTimeTolerance || delta < -sessionTimeTolerance {
						add(DivergenceSessionTime, fmt.Sprintf("row %d reports %s of session time but start and stop span %s",
							row.ID, reported, elapsed.Round(time.Second)))
					}
				}
			}
		}
		if !row.AcctStopTime.IsZero() && row.AcctStopTime.After(state.Stop) {
			state.Stop = row.AcctStopTime
		}
		if i > 0 {
			previous := ordered[i-1]
			if row.AcctInputTotal < previous.AcctInputTotal || row.AcctOutputTotal < previous.AcctOutputTotal {
				add(DivergenceCounterRegression, fmt.Sprintf("row %d counters (%d in, %d out) are lower than row %d (%d in, %d out)",
					row.ID, row.AcctInputTotal, row.AcctOutputTotal,
					previous.ID, previous.AcctInputTotal, previous.AcctOutputTotal))
			}
			if previous.AcctStopTime.IsZero() {
				add(DivergenceSessionOverlap, fmt.Sprintf("row %d starts at %s while row %d is still open",
					row.ID, rowTime(row).Format(time.RFC3339), previous.ID))
			} else if !row.AcctStartTime.IsZero() && row.AcctStartTime.Before(previous.AcctStopTime) {
				add(DivergenceSessionOverlap, fmt.Sprintf("row %d starts at %s before row %d stopped at %s",
					row.ID, row.AcctStartTime.Format(time.RFC3339), previous.ID, previous.AcctStopTime.Format(time.RFC3339)))
			}
		}
		state.InputOctets = row.AcctInputTotal
		state.OutputOctets = row.AcctOutputTotal
		state.SessionTime = row.AcctSessionTime
	}

	newestOpen := len(ordered) > 0 && ordered[len(ordered)-1].AcctStopTime.IsZero()
	state.Open = online != nil || newestOpen
	if len(ordered) == 0 && online != nil {
		add(DivergenceOnlineWithoutAccounting, fmt.Sprintf("the online row for %s has no accounting row", online.Username))
	}
	if len(ordered) > 0 && newestOpen && online == nil {
		add(DivergenceOrphanOpen, "the newest accounting row has no stop time and no online row exists, so the Accounting-Stop never reached the server")
	}
	if len(ordered) > 0 && !newestOpen && online != nil {
		add(DivergenceClosedWithOnline, fmt.Sprintf("the newest accounting row stopped at %s but an online row still exists",
			ordered[len(ordered)-1].AcctStopTime.Format(time.RFC3339)))
	}
	if state.Open && !state.Start.IsZero() && now.Sub(state.Start) > maxOpenSession {
		add(DivergenceStaleOpen, fmt.Sprintf("the session has been open since %s (%s)",
			state.Start.Format(time.RFC3339), now.Sub(state.Start).Round(time.Hour)))
	}

	live := online != nil
	for _, action := range actions {
		switch action.Status {
		case sessionqueue.StatusSucceeded:
			if live {
				add(DivergenceDisconnectNoEffect, fmt.Sprintf("the %s action (%s) succeeded but the session is still online",
					action.Action, action.Reason))
			}
		case sessionqueue.StatusPending, sessionqueue.StatusRunning, sessionqueue.StatusFailed:
			if live {
				add(DivergenceDisconnectPending, fmt.Sprintf("the %s action (%s) is %s after %d attempt(s) while the session is still online",
					action.Action, action.Reason, action.Status, action.Attempts))
			}
		}
	}

	latestAudit := make(map[string]domain.RadiusSessionActionAudit, len(audits))
	for _, audit := range audits {
		if previous, ok := latestAudit[audit.Action]; !ok || audit.TriggeredAt.After(previous.TriggeredAt) {
			latestAudit[audit.Action] = audit
		}
	}
	for action, audit := range latestAudit {
		if audit.Success && live {
			add(DivergenceDisconnectNoEffect, fmt.Sprintf("the last %s audit at %s reported success but the session is still online",
				action, audit.TriggeredAt.Format(time.RFC3339)))
		}
	}

	sortDivergences(divergences)
	return state, divergences
}

// rowTime orders accounting rows by the NAS-reported start time, falling back
// to the last update for rows that never carried a start time.
func rowTime(row domain.RadiusAccounting) time.Time {
	if !row.AcctStartTime.IsZero() {
		return row.AcctStartTime
	}
	return row.LastUpdate
}

func sortDivergences(list []Divergence) {
	sort.SliceStable(list, func(i, j int) bool {
		if list[i].Subject != list[j].Subject {
			return list[i].Subject < list[j].Subject
		}
		if list[i].Kind != list[j].Kind {
			return list[i].Kind < list[j].Kind
		}
		return list[i].Detail < list[j].Detail
	})
}
