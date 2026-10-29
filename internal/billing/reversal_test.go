package billing

import (
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var (
	testCorrection = PaymentCorrection{
		Operator:   "operator.a",
		OperatorIP: "192.0.2.10",
		Reason:     "wrong customer invoice",
	}
	testClock = time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC)
)

// reversalFixture creates an open invoice due at testClock+dueOffset and
// returns it with the clock the reversal uses.
func reversalFixture(t *testing.T, total int64, status string, dueOffset time.Duration) (*gorm.DB, domain.Invoice, time.Time) {
	t.Helper()
	db := billingTestDB(t)
	now := testClock
	invoice := domain.Invoice{
		InvoiceNo:      "INV-REV-1",
		CustomerID:     1,
		SubscriptionID: 1,
		InvoiceDate:    now.AddDate(0, 0, -5),
		DueDate:        now.Add(dueOffset),
		PeriodStart:    now.AddDate(0, 0, -5),
		PeriodEnd:      now,
		Subtotal:       total,
		Total:          total,
		Balance:        total,
		Status:         status,
	}
	require.NoError(t, db.Create(&invoice).Error)
	return db, invoice, now
}

func recordReversalPayment(t *testing.T, db *gorm.DB, invoiceID, amount int64, now time.Time) domain.Payment {
	t.Helper()
	payment := domain.Payment{InvoiceID: invoiceID, Amount: amount, Method: "cash"}
	require.NoError(t, RecordPayment(db, &payment, now, false))
	return payment
}

func loadInvoice(t *testing.T, db *gorm.DB, id int64) domain.Invoice {
	t.Helper()
	var invoice domain.Invoice
	require.NoError(t, db.First(&invoice, id).Error)
	return invoice
}

func countCorrectionEvents(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).Where("type = ?", "payment_reversed").Count(&count).Error)
	return count
}

func countOperatorLogs(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&domain.SysOprLog{}).Count(&count).Error)
	return count
}

// TestReversePaymentRestoresInvoiceAcrossPaymentStates is the paid/partial
// matrix: reversing a payment must restore exactly the balance it settled and
// derive a status the invoice can legitimately hold, never leaving it paid.
func TestReversePaymentRestoresInvoiceAcrossPaymentStates(t *testing.T) {
	t.Run("full payment on a future-due invoice returns to issued", func(t *testing.T) {
		db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
		payment := recordReversalPayment(t, db, invoice.ID, 150000, now)
		require.Equal(t, domain.InvoicePaid, loadInvoice(t, db, invoice.ID).Status)

		reversed, updated, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, domain.PaymentReversed, reversed.Status)
		require.Equal(t, domain.InvoiceIssued, updated.Status)
		require.EqualValues(t, 0, updated.PaidAmount)
		require.EqualValues(t, 150000, updated.Balance)

		stored := loadInvoice(t, db, invoice.ID)
		require.Equal(t, domain.InvoiceIssued, stored.Status)
		require.EqualValues(t, 150000, stored.Balance)

		var trail domain.BillingEvent
		require.NoError(t, db.Where("type = ?", "payment_reversed").First(&trail).Error)
		require.Equal(t, invoice.ID, trail.InvoiceID)
		require.Contains(t, trail.Description, payment.PaymentNo)
		require.Contains(t, trail.Description, testCorrection.Operator)
		require.Contains(t, trail.Description, testCorrection.Reason)

		var log domain.SysOprLog
		require.NoError(t, db.First(&log).Error)
		require.Equal(t, testCorrection.Operator, log.OprName)
		require.Equal(t, testCorrection.OperatorIP, log.OprIp)
		require.Equal(t, "payment.reverse", log.OptAction)
		require.Contains(t, log.OptDesc, invoice.InvoiceNo)
		require.Contains(t, log.OptDesc, testCorrection.Reason)
	})

	t.Run("full payment on an overdue invoice returns to overdue", func(t *testing.T) {
		db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, -48*time.Hour)
		payment := recordReversalPayment(t, db, invoice.ID, 150000, now)

		_, updated, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, domain.InvoiceOverdue, updated.Status)
		require.EqualValues(t, 150000, updated.Balance)
	})

	t.Run("reversing the last of two payments leaves the invoice partial", func(t *testing.T) {
		db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
		recordReversalPayment(t, db, invoice.ID, 60000, now)
		last := recordReversalPayment(t, db, invoice.ID, 90000, now)
		require.Equal(t, domain.InvoicePaid, loadInvoice(t, db, invoice.ID).Status)

		_, updated, err := ReversePayment(db, last.ID, testCorrection, now.Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, domain.InvoicePartial, updated.Status)
		require.EqualValues(t, 60000, updated.PaidAmount)
		require.EqualValues(t, 90000, updated.Balance)
	})

	t.Run("reversing the only partial payment returns to issued", func(t *testing.T) {
		db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
		payment := recordReversalPayment(t, db, invoice.ID, 50000, now)
		require.Equal(t, domain.InvoicePartial, loadInvoice(t, db, invoice.ID).Status)

		_, updated, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
		require.NoError(t, err)
		require.Equal(t, domain.InvoiceIssued, updated.Status)
		require.EqualValues(t, 0, updated.PaidAmount)
		require.EqualValues(t, 150000, updated.Balance)
	})
}

// TestReversePaymentRejectsVoidInvoice covers the void row of the matrix: a
// void invoice has no collectible balance, so correcting a payment against it
// would create an unreconciled credit rather than restore a debt.
func TestReversePaymentRejectsVoidInvoice(t *testing.T) {
	db, invoice, now := reversalFixture(t, 150000, domain.InvoiceVoid, 48*time.Hour)
	payment := domain.Payment{
		PaymentNo: "PAY-VOID-1", InvoiceID: invoice.ID, CustomerID: 1,
		Amount: 150000, Method: "cash", PaidAt: now, Status: domain.PaymentReceived,
	}
	require.NoError(t, db.Create(&payment).Error)

	_, _, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
	require.ErrorIs(t, err, ErrInvoiceVoided)

	var stored domain.Payment
	require.NoError(t, db.First(&stored, payment.ID).Error)
	require.Equal(t, domain.PaymentReceived, stored.Status, "a rejected correction must not touch the payment")
	require.Equal(t, domain.InvoiceVoid, loadInvoice(t, db, invoice.ID).Status)
	require.Zero(t, countCorrectionEvents(t, db))
	require.Zero(t, countOperatorLogs(t, db))
}

// TestReversePaymentIsSingleShot pins idempotence: one correction per payment,
// with exactly one audit trail entry.
func TestReversePaymentIsSingleShot(t *testing.T) {
	db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
	payment := recordReversalPayment(t, db, invoice.ID, 150000, now)

	_, _, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
	require.NoError(t, err)

	_, _, err = ReversePayment(db, payment.ID, testCorrection, now.Add(2*time.Hour))
	require.ErrorIs(t, err, ErrPaymentNotReversible)
	require.EqualValues(t, 1, countCorrectionEvents(t, db))
	require.EqualValues(t, 1, countOperatorLogs(t, db))
}

// TestReversePaymentRequiresOperatorReason keeps the audit trail substantive:
// an empty, short, or oversized reason and a missing operator are all rejected
// before any row changes.
func TestReversePaymentRequiresOperatorReason(t *testing.T) {
	db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
	payment := recordReversalPayment(t, db, invoice.ID, 150000, now)

	for name, correction := range map[string]PaymentCorrection{
		"empty reason": {Operator: "operator.a", Reason: "   "},
		"short reason": {Operator: "operator.a", Reason: "typo"},
		"long reason":  {Operator: "operator.a", Reason: strings.Repeat("x", 201)},
		"no operator":  {Reason: testCorrection.Reason},
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := ReversePayment(db, payment.ID, correction, now.Add(time.Hour))
			require.Error(t, err)
		})
	}

	var stored domain.Payment
	require.NoError(t, db.First(&stored, payment.ID).Error)
	require.Equal(t, domain.PaymentReceived, stored.Status)
	require.Equal(t, domain.InvoicePaid, loadInvoice(t, db, invoice.ID).Status)
	require.Zero(t, countCorrectionEvents(t, db))
	require.Zero(t, countOperatorLogs(t, db))
}

// TestReversePaymentRejectsUnreceivedPayment covers the payment-side states
// that are not money in the ledger.
func TestReversePaymentRejectsUnreceivedPayment(t *testing.T) {
	db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
	payment := domain.Payment{
		PaymentNo: "PAY-PENDING-1", InvoiceID: invoice.ID, CustomerID: 1,
		Amount: 150000, Method: "cash", PaidAt: now, Status: domain.PaymentPending,
	}
	require.NoError(t, db.Create(&payment).Error)

	_, _, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
	require.ErrorIs(t, err, ErrPaymentNotReversible)
	require.Zero(t, countCorrectionEvents(t, db))
}

// TestReversePaymentRejectsLedgerMismatch refuses to invent a balance when the
// stored payment and invoice amounts disagree.
func TestReversePaymentRejectsLedgerMismatch(t *testing.T) {
	db, invoice, now := reversalFixture(t, 150000, domain.InvoiceIssued, 48*time.Hour)
	payment := domain.Payment{
		PaymentNo: "PAY-MISMATCH-1", InvoiceID: invoice.ID, CustomerID: 1,
		Amount: 150000, Method: "cash", PaidAt: now, Status: domain.PaymentReceived,
	}
	require.NoError(t, db.Create(&payment).Error)

	_, _, err := ReversePayment(db, payment.ID, testCorrection, now.Add(time.Hour))
	require.Error(t, err)
	require.NotErrorIs(t, err, ErrPaymentNotReversible)
	require.Equal(t, domain.InvoiceIssued, loadInvoice(t, db, invoice.ID).Status)
	require.Zero(t, countCorrectionEvents(t, db))
}
