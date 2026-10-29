package replay

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var replayDBSeq atomic.Int64

func replayTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:replay-%d?mode=memory&cache=shared", replayDBSeq.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.Invoice{}, &domain.Payment{}, &domain.BillingEvent{},
		&domain.RadiusAccounting{}, &domain.RadiusOnline{},
		&domain.RadiusSessionAction{}, &domain.RadiusSessionActionAudit{},
	))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

var replayNow = time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC)

func replayInvoice(paid, total int64, status string, due time.Time) domain.Invoice {
	return domain.Invoice{
		InvoiceNo:  "INV-REPLAY-1",
		Total:      total,
		PaidAmount: paid,
		Balance:    total - paid,
		Status:     status,
		DueDate:    due,
	}
}

func replayPayment(number string, amount int64, status string, paidAt time.Time) domain.Payment {
	return domain.Payment{PaymentNo: number, Amount: amount, Status: status, PaidAt: paidAt}
}

func replayEvent(id int64, kind, description string, at time.Time) domain.BillingEvent {
	return domain.BillingEvent{ID: id, Type: kind, Description: description, CreatedAt: at}
}

func divergenceKinds(divergences []Divergence) []string {
	kinds := make([]string, 0, len(divergences))
	for _, divergence := range divergences {
		kinds = append(kinds, divergence.Kind)
	}
	return kinds
}

func TestReplayInvoiceRebuildsLedgerFromTrail(t *testing.T) {
	invoice := replayInvoice(50000, 150000, domain.InvoicePartial, replayNow.AddDate(0, 0, 10))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", 50000, domain.PaymentReceived, replayNow.AddDate(0, 0, -10)),
		replayPayment("PAY-202610-000002", 100000, domain.PaymentReversed, replayNow.AddDate(0, 0, -5)),
	}
	events := []domain.BillingEvent{
		replayEvent(1, "invoice_generated", invoice.InvoiceNo, replayNow.AddDate(0, 0, -30)),
		replayEvent(2, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
		replayEvent(3, "payment_received", "PAY-202610-000002", replayNow.AddDate(0, 0, -5)),
		replayEvent(4, "payment_reversed", "PAY-202610-000002 reversed by alice: wrong invoice", replayNow.AddDate(0, 0, -2)),
	}

	state, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.Equal(t, int64(50000), state.Paid)
	assert.Equal(t, int64(100000), state.Balance)
	assert.Equal(t, domain.InvoicePartial, state.Status)
	assert.Equal(t, 2, state.Payments)
	assert.Equal(t, 4, state.Events)
	assert.Empty(t, divergences, "%v", divergences)
}

func TestReplayInvoiceDetectsTamperedLedger(t *testing.T) {
	invoice := replayInvoice(150000, 150000, domain.InvoicePaid, replayNow.AddDate(0, 0, -15))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", 50000, domain.PaymentReceived, replayNow.AddDate(0, 0, -10)),
	}
	events := []domain.BillingEvent{
		replayEvent(1, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
	}

	state, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.Equal(t, int64(50000), state.Paid)
	assert.ElementsMatch(t,
		[]string{DivergencePaidAmount, DivergenceBalance, DivergenceStatus},
		divergenceKinds(divergences), "%v", divergences)
}

func TestReplayInvoiceDetectsMissingReversalEvent(t *testing.T) {
	invoice := replayInvoice(0, 150000, domain.InvoiceIssued, replayNow.AddDate(0, 0, 10))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", 50000, domain.PaymentReversed, replayNow.AddDate(0, 0, -10)),
	}
	events := []domain.BillingEvent{
		replayEvent(1, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
	}

	state, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.Equal(t, int64(50000), state.Paid, "the trail still shows the payment as received")
	assert.Contains(t, divergenceKinds(divergences), DivergenceMissingEvent, "%v", divergences)
	assert.Contains(t, divergenceKinds(divergences), DivergencePaidAmount, "%v", divergences)
}

func TestReplayInvoiceDetectsReversalWithoutPayment(t *testing.T) {
	invoice := replayInvoice(0, 150000, domain.InvoiceIssued, replayNow.AddDate(0, 0, 10))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", 50000, domain.PaymentReversed, replayNow.AddDate(0, 0, -10)),
	}
	events := []domain.BillingEvent{
		replayEvent(1, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
		replayEvent(2, "payment_reversed", "PAY-202610-000001 reversed by alice: first correction", replayNow.AddDate(0, 0, -3)),
		replayEvent(3, "payment_reversed", "PAY-202610-000001 reversed by alice: second correction", replayNow.AddDate(0, 0, -2)),
	}

	_, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.Equal(t, []string{DivergenceReversalOrder}, divergenceKinds(divergences), "%v", divergences)
}

func TestReplayInvoiceDetectsEventOrderAndUnknownPayment(t *testing.T) {
	invoice := replayInvoice(50000, 150000, domain.InvoicePartial, replayNow.AddDate(0, 0, 10))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", 50000, domain.PaymentReceived, replayNow.AddDate(0, 0, -10)),
	}
	events := []domain.BillingEvent{
		replayEvent(2, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
		replayEvent(1, "payment_received", "PAY-202610-000009", replayNow.AddDate(0, 0, -5)),
	}

	_, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.ElementsMatch(t,
		[]string{DivergenceEventOrder, DivergenceUnknownPayment},
		divergenceKinds(divergences), "%v", divergences)
}

func TestReplayInvoiceDetectsNegativeLedger(t *testing.T) {
	invoice := replayInvoice(-5000, 150000, domain.InvoiceIssued, replayNow.AddDate(0, 0, 10))
	payments := []domain.Payment{
		replayPayment("PAY-202610-000001", -5000, domain.PaymentReceived, replayNow.AddDate(0, 0, -10)),
	}
	events := []domain.BillingEvent{
		replayEvent(1, "payment_received", "PAY-202610-000001", replayNow.AddDate(0, 0, -10)),
	}

	_, divergences := ReplayInvoice(invoice, payments, events, replayNow)

	assert.Contains(t, divergenceKinds(divergences), DivergenceNegativePaid, "%v", divergences)
}

func closedRow(id int64, sessionID string, start time.Time, elapsed time.Duration) domain.RadiusAccounting {
	return domain.RadiusAccounting{
		ID:                id,
		AcctSessionId:     sessionID,
		Username:          "sub-1",
		AcctStartTime:     start,
		AcctStopTime:      start.Add(elapsed),
		AcctSessionTime:   int(elapsed.Seconds()),
		AcctInputTotal:    1000,
		AcctOutputTotal:   2000,
		AcctInputPackets:  10,
		AcctOutputPackets: 20,
	}
}

func TestReplaySessionRebuildsClosedLifetime(t *testing.T) {
	start := replayNow.AddDate(0, 0, -3)
	state, divergences := ReplaySession("sess-1", []domain.RadiusAccounting{closedRow(1, "sess-1", start, 2*time.Hour)}, nil, nil, nil, replayNow)

	assert.False(t, state.Open)
	assert.Equal(t, start, state.Start)
	assert.Equal(t, start.Add(2*time.Hour), state.Stop)
	assert.Equal(t, int64(1000), state.InputOctets)
	assert.Empty(t, divergences, "%v", divergences)
}

func TestReplaySessionDetectsImpossibleLifetime(t *testing.T) {
	start := replayNow.AddDate(0, 0, -2)
	row := closedRow(1, "sess-1", start, 5*time.Minute)
	row.AcctStopTime = start.Add(-5 * time.Minute)

	_, divergences := ReplaySession("sess-1", []domain.RadiusAccounting{row}, nil, nil, nil, replayNow)
	assert.Contains(t, divergenceKinds(divergences), DivergenceStopBeforeStart, "%v", divergences)

	mismatch := closedRow(2, "sess-1", start, 30*time.Minute)
	mismatch.AcctSessionTime = 300
	_, divergences = ReplaySession("sess-1", []domain.RadiusAccounting{mismatch}, nil, nil, nil, replayNow)
	assert.Contains(t, divergenceKinds(divergences), DivergenceSessionTime, "%v", divergences)
}

func TestReplaySessionDetectsOrphanAndStaleOpenSession(t *testing.T) {
	start := replayNow.AddDate(0, 0, -40)
	row := closedRow(1, "sess-1", start, 0)
	row.AcctStopTime = time.Time{}
	row.AcctSessionTime = 0

	state, divergences := ReplaySession("sess-1", []domain.RadiusAccounting{row}, nil, nil, nil, replayNow)

	assert.True(t, state.Open)
	assert.ElementsMatch(t,
		[]string{DivergenceOrphanOpen, DivergenceStaleOpen},
		divergenceKinds(divergences), "%v", divergences)
}

func TestReplaySessionDetectsUndeliveredDisconnect(t *testing.T) {
	start := replayNow.Add(-2 * time.Hour)
	row := closedRow(1, "sess-1", start, 0)
	row.AcctStopTime = time.Time{}
	online := &domain.RadiusOnline{AcctSessionId: "sess-1", Username: "sub-1"}
	actions := []domain.RadiusSessionAction{{
		AcctSessionID: "sess-1", Action: sessionqueue.ActionDisconnect,
		Reason: sessionqueue.ReasonBillingSuspension, Status: sessionqueue.StatusFailed, Attempts: 3,
	}}
	audits := []domain.RadiusSessionActionAudit{{
		AcctSessionID: "sess-1", Action: sessionqueue.ActionDisconnect,
		Success: true, TriggeredAt: replayNow.Add(-time.Hour),
	}}

	_, divergences := ReplaySession("sess-1", []domain.RadiusAccounting{row}, online, actions, audits, replayNow)

	assert.ElementsMatch(t,
		[]string{DivergenceDisconnectPending, DivergenceDisconnectNoEffect},
		divergenceKinds(divergences), "%v", divergences)
}

func TestReplaySessionDetectsOnlineWithoutAccounting(t *testing.T) {
	online := &domain.RadiusOnline{AcctSessionId: "sess-1", Username: "sub-1"}

	_, divergences := ReplaySession("sess-1", nil, online, nil, nil, replayNow)

	assert.Equal(t, []string{DivergenceOnlineWithoutAccounting}, divergenceKinds(divergences), "%v", divergences)
}

func TestReplaySessionDetectsCounterRegressionAndOverlap(t *testing.T) {
	start := replayNow.AddDate(0, 0, -1)
	first := closedRow(1, "sess-1", start, time.Hour)
	second := closedRow(2, "sess-1", start.Add(30*time.Minute), time.Hour)
	second.AcctInputTotal = 500
	second.AcctOutputTotal = 400

	_, divergences := ReplaySession("sess-1", []domain.RadiusAccounting{first, second}, nil, nil, nil, replayNow)

	assert.ElementsMatch(t,
		[]string{DivergenceCounterRegression, DivergenceSessionOverlap},
		divergenceKinds(divergences), "%v", divergences)
}

func TestReplayBillingReadsStoredLedger(t *testing.T) {
	db := replayTestDB(t)

	invoice := domain.Invoice{
		InvoiceNo: "INV-DB-1", CustomerID: 1, SubscriptionID: 1,
		InvoiceDate: replayNow.AddDate(0, 0, -20), DueDate: replayNow.AddDate(0, 0, 10),
		PeriodStart: replayNow.AddDate(0, 0, -20), PeriodEnd: replayNow.AddDate(0, 0, -1),
		Subtotal: 150000, Total: 150000, PaidAmount: 50000, Balance: 100000,
		Status: domain.InvoicePartial,
	}
	require.NoError(t, db.Create(&invoice).Error)
	require.NoError(t, db.Create(&domain.Payment{
		PaymentNo: "PAY-DB-1", CustomerID: 1, InvoiceID: invoice.ID, Amount: 50000,
		Method: "cash", PaidAt: replayNow.AddDate(0, 0, -10), Status: domain.PaymentReceived,
	}).Error)
	for _, event := range []domain.BillingEvent{
		{CustomerID: 1, SubscriptionID: 1, InvoiceID: invoice.ID, Type: "invoice_generated", Description: invoice.InvoiceNo, CreatedAt: replayNow.AddDate(0, 0, -20)},
		{CustomerID: 1, SubscriptionID: 1, InvoiceID: invoice.ID, Type: "payment_received", Description: "PAY-DB-1", CreatedAt: replayNow.AddDate(0, 0, -10)},
	} {
		require.NoError(t, db.Create(&event).Error)
	}

	report, err := ReplayBilling(db, replayNow)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Invoices)
	assert.Equal(t, 1, report.Payments)
	assert.Equal(t, 2, report.Events)
	assert.Empty(t, report.Divergences, "%v", report.Divergences)
	assert.NotEmpty(t, report.Gaps)

	require.NoError(t, db.Model(&domain.Invoice{}).Where("id = ?", invoice.ID).
		Updates(map[string]interface{}{"paid_amount": 150000, "balance": 0, "status": domain.InvoicePaid}).Error)

	report, err = ReplayBilling(db, replayNow)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{DivergencePaidAmount, DivergenceBalance, DivergenceStatus},
		divergenceKinds(report.Divergences), "%v", report.Divergences)
}

func TestReplayRadiusReadsStoredSession(t *testing.T) {
	db := replayTestDB(t)

	start := replayNow.Add(-3 * time.Hour)
	row := closedRow(1, "sess-db", start, time.Hour)
	require.NoError(t, db.Create(&row).Error)
	require.NoError(t, db.Create(&domain.RadiusSessionAction{
		IdempotencyKey: "replay-db", AcctSessionID: "sess-db", Username: "sub-1",
		Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension,
		Status: sessionqueue.StatusSucceeded,
	}).Error)

	report, err := ReplayRadius(db, replayNow)
	require.NoError(t, err)
	assert.Equal(t, 1, report.Sessions)
	assert.Equal(t, 1, report.Rows)
	assert.Empty(t, report.Divergences, "%v", report.Divergences)
	assert.NotEmpty(t, report.Gaps)

	require.NoError(t, db.Create(&domain.RadiusOnline{
		AcctSessionId: "sess-db", Username: "sub-1", NasAddr: "192.0.2.1",
	}).Error)

	report, err = ReplayRadius(db, replayNow)
	require.NoError(t, err)
	assert.ElementsMatch(t,
		[]string{DivergenceClosedWithOnline, DivergenceDisconnectNoEffect},
		divergenceKinds(report.Divergences), "%v", report.Divergences)
}

func TestReplayEmptyDatabaseReportsNothing(t *testing.T) {
	db := replayTestDB(t)

	billing, err := ReplayBilling(db, replayNow)
	require.NoError(t, err)
	assert.Zero(t, billing.Invoices)
	assert.Empty(t, billing.Divergences)

	radius, err := ReplayRadius(db, replayNow)
	require.NoError(t, err)
	assert.Zero(t, radius.Sessions)
	assert.Empty(t, radius.Divergences)
}
