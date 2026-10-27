package billing

import (
	"errors"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func billingTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.InternetPackage{}, &domain.Subscription{},
		&domain.Invoice{}, &domain.InvoiceItem{}, &domain.Payment{},
		&domain.BillingEvent{}, &domain.RadiusUser{},
		&domain.DocumentSequence{},
	))
	return db
}

func TestGenerateMonthlyInvoicesIsIdempotent(t *testing.T) {
	db := billingTestDB(t)
	loc, err := time.LoadLocation("Asia/Jakarta")
	require.NoError(t, err)
	now := time.Date(2026, time.October, 1, 0, 0, 0, 0, loc)
	customer := domain.Customer{Name: "Customer One", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	pkg := domain.InternetPackage{Code: "HOME-10", Name: "Home 10 Mbps", Price: 150000, RadiusProfileID: 1, BillingCycle: "monthly", Status: "active"}
	require.NoError(t, db.Create(&pkg).Error)
	sub := domain.Subscription{CustomerID: customer.ID, PackageID: pkg.ID, Status: domain.SubscriptionActive,
		StartDate: now.AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)

	created, err := GenerateMonthlyInvoices(db, now, 10)
	require.NoError(t, err)
	require.Equal(t, 1, created)
	created, err = GenerateMonthlyInvoices(db, now.Add(time.Hour), 10)
	require.NoError(t, err)
	require.Zero(t, created)

	var invoice domain.Invoice
	require.NoError(t, db.Where("subscription_id = ?", sub.ID).First(&invoice).Error)
	require.Equal(t, "INV-202610-000001", invoice.InvoiceNo)
	require.Equal(t, int64(150000), invoice.Total)
	require.Equal(t, int64(150000), invoice.Balance)
	require.True(t, time.Date(2026, time.October, 11, 0, 0, 0, 0, loc).Equal(invoice.DueDate))
	var item domain.InvoiceItem
	require.NoError(t, db.Where("invoice_id = ?", invoice.ID).First(&item).Error)
	require.Equal(t, int64(150000), item.UnitPrice)
	require.Equal(t, "Home 10 Mbps", item.Description)

	created, err = GenerateMonthlyInvoices(db, now.AddDate(0, 1, 0), 10)
	require.NoError(t, err)
	require.Equal(t, 1, created)
	var november domain.Invoice
	require.NoError(t, db.Where("subscription_id = ? AND invoice_date = ?", sub.ID, now.AddDate(0, 1, 0)).First(&november).Error)
	require.Equal(t, "INV-202611-000001", november.InvoiceNo)
}

func TestRecordPaymentPartialAndAutoReactivation(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Customer Two", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "subscriber-2", Password: "secret", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-000002", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: domain.SuspensionBillingOverdue,
		StartDate: now.AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{InvoiceNo: "INV-202610-000002", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, 0, -14), DueDate: now.AddDate(0, 0, -4), PeriodStart: now.AddDate(0, 0, -14),
		PeriodEnd: now.AddDate(0, 0, 16), Subtotal: 350000, Total: 350000, Balance: 350000, Status: domain.InvoiceOverdue}
	require.NoError(t, db.Create(&invoice).Error)

	first := domain.Payment{InvoiceID: invoice.ID, Amount: 100000, Method: "cash"}
	require.NoError(t, RecordPayment(db, &first, now, true))
	require.Equal(t, "PAY-202610-000001", first.PaymentNo)
	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, domain.InvoicePartial, updated.Status)
	require.Equal(t, int64(250000), updated.Balance)

	tooMuch := domain.Payment{InvoiceID: invoice.ID, Amount: 250001, Method: "cash"}
	require.ErrorIs(t, RecordPayment(db, &tooMuch, now, true), ErrInvalidPayment)

	last := domain.Payment{InvoiceID: invoice.ID, Amount: 250000, Method: "bank_transfer"}
	require.NoError(t, RecordPayment(db, &last, now.Add(time.Minute), true))
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, domain.InvoicePaid, updated.Status)
	require.Zero(t, updated.Balance)
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionActive, sub.Status)
	require.Empty(t, sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "enabled", user.Status)

	novemberInvoice := domain.Invoice{InvoiceNo: "INV-202611-000099", CustomerID: customer.ID, Total: 1000, Balance: 1000, Status: domain.InvoiceIssued, InvoiceDate: now.AddDate(0, 1, 0), DueDate: now.AddDate(0, 1, 10)}
	require.NoError(t, db.Create(&novemberInvoice).Error)
	novemberPayment := domain.Payment{InvoiceID: novemberInvoice.ID, Amount: 1000, Method: "cash"}
	require.NoError(t, RecordPayment(db, &novemberPayment, now.AddDate(0, 1, 0), false))
	require.Equal(t, "PAY-202611-000001", novemberPayment.PaymentNo)
}

func TestRecordPaymentDoesNotReactivateManualSuspension(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Manual Suspension", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "manual-user", Password: "secret", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-MANUAL", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: "manual", StartDate: now, BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{InvoiceNo: "INV-MANUAL", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now, DueDate: now.AddDate(0, 0, 10), PeriodStart: now, PeriodEnd: now.AddDate(0, 1, -1),
		Total: 1000, Balance: 1000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	payment := domain.Payment{InvoiceID: invoice.ID, Amount: 1000, Method: "manual"}
	require.NoError(t, RecordPayment(db, &payment, now, true))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, "manual", sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "disabled", user.Status)
}

// TestManualSuspensionSurvivesSchedulerAndPayment covers the ENG-032 invariant:
// the billing scheduler and a later payment may finish a billing suspension,
// but they must not rewrite or clear an operator's manual suspension.
func TestManualSuspensionSurvivesSchedulerAndPayment(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Manual Stays", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "manual-stays", Password: "secret", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-MANUAL-2", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: "manual",
		StartDate: now.AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{InvoiceNo: "INV-MANUAL-2", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, 0, -20), DueDate: now.AddDate(0, 0, -10),
		PeriodStart: now.AddDate(0, 0, -20), PeriodEnd: now.AddDate(0, 0, 10),
		Total: 50000, Balance: 50000, Status: domain.InvoiceOverdue}
	require.NoError(t, db.Create(&invoice).Error)

	require.NoError(t, SuspendOverdueSubscriptions(db, now))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, "manual", sub.SuspensionReason)
	var suspensions int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_suspended").Count(&suspensions).Error)
	require.Zero(t, suspensions, "the scheduler must not add a billing suspension event to a manual suspension")

	payment := domain.Payment{InvoiceID: invoice.ID, Amount: 50000, Method: "manual"}
	require.NoError(t, RecordPayment(db, &payment, now, true))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, "manual", sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "disabled", user.Status)
	var reactivated int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_reactivated").Count(&reactivated).Error)
	require.Zero(t, reactivated, "a payment must not reactivate a manually suspended subscription")
}

// TestRecordPaymentKeepsSuspensionUntilEveryOverdueInvoiceIsPaid covers the
// multi-invoice policy: settling one overdue invoice must not restore access
// while another overdue invoice for the same subscription still has a balance.
func TestRecordPaymentKeepsSuspensionUntilEveryOverdueInvoiceIsPaid(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Multiple Invoices", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "multi-invoice", Password: "secret", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-MULTI", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: domain.SuspensionBillingOverdue,
		StartDate: now.AddDate(0, -2, 0), BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	first := domain.Invoice{InvoiceNo: "INV-MULTI-1", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, -2, 0), DueDate: now.AddDate(0, -10, 0),
		PeriodStart: now.AddDate(0, -2, 0), PeriodEnd: now.AddDate(0, -1, 0),
		Total: 300000, Balance: 300000, Status: domain.InvoiceOverdue}
	second := domain.Invoice{InvoiceNo: "INV-MULTI-2", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, -1, 0), DueDate: now.AddDate(0, -4, 0),
		PeriodStart: now.AddDate(0, -1, 0), PeriodEnd: now,
		Total: 300000, Balance: 300000, Status: domain.InvoiceOverdue}
	require.NoError(t, db.Create(&first).Error)
	require.NoError(t, db.Create(&second).Error)

	firstPayment := domain.Payment{InvoiceID: first.ID, Amount: 300000, Method: "cash"}
	require.NoError(t, RecordPayment(db, &firstPayment, now, true))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status, "another overdue invoice still blocks reactivation")
	require.Equal(t, domain.SuspensionBillingOverdue, sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "disabled", user.Status)
	var reactivated int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_reactivated").Count(&reactivated).Error)
	require.Zero(t, reactivated)

	secondPayment := domain.Payment{InvoiceID: second.ID, Amount: 300000, Method: "cash"}
	require.NoError(t, RecordPayment(db, &secondPayment, now.Add(time.Minute), true))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionActive, sub.Status)
	require.Empty(t, sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "enabled", user.Status)
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_reactivated").Count(&reactivated).Error)
	require.EqualValues(t, 1, reactivated)
}

// TestSuspendOverdueSkipsManualSuspension proves the automatic billing pass never
// rewrites an operator's manual suspension reason or duplicates its event.
func TestSuspendOverdueSkipsManualSuspension(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Manual And Overdue", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "manual-overdue", Password: "secret", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-MANUAL-OVERDUE", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: "manual", StartDate: now, BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{InvoiceNo: "INV-MANUAL-OVERDUE", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, 0, -14), DueDate: now.AddDate(0, 0, -10),
		PeriodStart: now.AddDate(0, 0, -14), PeriodEnd: now.AddDate(0, 0, 16),
		Total: 2000, Balance: 2000, Status: domain.InvoiceOverdue}
	require.NoError(t, db.Create(&invoice).Error)

	require.NoError(t, SuspendOverdueSubscriptions(db, now))
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, "manual", sub.SuspensionReason)
	var suspensions int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_suspended").Count(&suspensions).Error)
	require.Zero(t, suspensions)
}

func TestProcessOverdueAndSuspendAfterGrace(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 12, 0, 0, 0, time.UTC)
	customer := domain.Customer{Name: "Overdue Customer", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{Username: "overdue-user", Password: "secret", Status: "enabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-OVERDUE", CustomerID: customer.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionActive, StartDate: now.AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{InvoiceNo: "INV-OVERDUE", CustomerID: customer.ID, SubscriptionID: sub.ID,
		InvoiceDate: now.AddDate(0, 0, -14), DueDate: time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC),
		PeriodStart: now.AddDate(0, 0, -14), PeriodEnd: now.AddDate(0, 0, 16), Total: 2000, Balance: 2000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	require.NoError(t, ProcessOverdueInvoices(db, now))
	require.NoError(t, SuspendOverdueSubscriptions(db, now))
	require.NoError(t, db.First(&invoice, invoice.ID).Error)
	require.Equal(t, domain.InvoiceOverdue, invoice.Status)
	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, domain.SuspensionBillingOverdue, sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "disabled", user.Status)

	// A billing-suspended subscription with a remaining balance cannot be
	// suspended a second time or have its original event duplicated.
	require.NoError(t, SuspendOverdueSubscriptions(db, now.Add(time.Hour)))
	var suspensions int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).Where("subscription_id = ? AND type = ?", sub.ID, "subscription_suspended").Count(&suspensions).Error)
	require.EqualValues(t, 1, suspensions)
}

func TestRecordPaymentIdempotentRetryDoesNotDoubleBook(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	invoice := domain.Invoice{InvoiceNo: "INV-IDEM-1", Total: 100000, Balance: 100000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	keyOf := func(key string) *string { return &key }
	first := domain.Payment{InvoiceID: invoice.ID, Amount: 40000, Method: "cash", IdempotencyKey: keyOf("retry-1")}
	require.NoError(t, RecordPayment(db, &first, now, false))

	retry := domain.Payment{InvoiceID: invoice.ID, Amount: 40000, Method: "cash", IdempotencyKey: keyOf("retry-1")}
	require.ErrorIs(t, RecordPayment(db, &retry, now.Add(time.Minute), false), ErrDuplicatePayment)

	conflict := domain.Payment{InvoiceID: invoice.ID, Amount: 50000, Method: "cash", IdempotencyKey: keyOf("retry-1")}
	require.ErrorIs(t, RecordPayment(db, &conflict, now.Add(2*time.Minute), false), ErrIdempotencyConflict)

	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, int64(40000), updated.PaidAmount)
	require.Equal(t, int64(60000), updated.Balance)
	require.Equal(t, domain.InvoicePartial, updated.Status)

	var payments, events int64
	require.NoError(t, db.Model(&domain.Payment{}).Count(&payments).Error)
	require.EqualValues(t, 1, payments)
	require.NoError(t, db.Model(&domain.BillingEvent{}).Where("type = ?", "payment_received").Count(&events).Error)
	require.EqualValues(t, 1, events)

	// A different key is a different payment even with identical details.
	second := domain.Payment{InvoiceID: invoice.ID, Amount: 40000, Method: "cash", IdempotencyKey: keyOf("retry-2")}
	require.NoError(t, RecordPayment(db, &second, now.Add(3*time.Minute), false))
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, int64(80000), updated.PaidAmount)
	require.Equal(t, int64(20000), updated.Balance)
}

func TestRecordPaymentRetryAfterInvoicePaidReportsDuplicate(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	invoice := domain.Invoice{InvoiceNo: "INV-IDEM-2", Total: 75000, Balance: 75000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	key := "retry-closed"
	full := domain.Payment{InvoiceID: invoice.ID, Amount: 75000, Method: "bank_transfer", Reference: "TRX-9", IdempotencyKey: &key}
	require.NoError(t, RecordPayment(db, &full, now, false))

	retry := domain.Payment{InvoiceID: invoice.ID, Amount: 75000, Method: "bank_transfer", Reference: "TRX-9", IdempotencyKey: &key}
	err := RecordPayment(db, &retry, now.Add(time.Minute), false)
	require.ErrorIs(t, err, ErrDuplicatePayment)
	require.False(t, errors.Is(err, ErrInvoiceClosed), "a replay must not be reported as a closed invoice")

	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, domain.InvoicePaid, updated.Status)
	require.Equal(t, int64(75000), updated.PaidAmount)
}

func TestRecordPaymentWithoutKeyRecordsSeparateIdenticalPayments(t *testing.T) {
	db := billingTestDB(t)
	now := time.Date(2026, time.October, 15, 10, 0, 0, 0, time.UTC)
	invoice := domain.Invoice{InvoiceNo: "INV-IDEM-3", Total: 100000, Balance: 100000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	for i := 0; i < 2; i++ {
		payment := domain.Payment{InvoiceID: invoice.ID, Amount: 30000, Method: "cash", Reference: "counter-cash"}
		require.NoError(t, RecordPayment(db, &payment, now.Add(time.Duration(i)*time.Minute), false))
	}

	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	require.Equal(t, int64(60000), updated.PaidAmount)
	var payments int64
	require.NoError(t, db.Model(&domain.Payment{}).Count(&payments).Error)
	require.EqualValues(t, 2, payments)
}

func TestRecordPaymentRejectsClosedInvoice(t *testing.T) {
	db := billingTestDB(t)
	invoice := domain.Invoice{InvoiceNo: "INV-CLOSED", Total: 100, Balance: 0, PaidAmount: 100, Status: domain.InvoicePaid}
	require.NoError(t, db.Create(&invoice).Error)
	payment := domain.Payment{InvoiceID: invoice.ID, Amount: 1, Method: "cash"}
	err := RecordPayment(db, &payment, time.Now(), true)
	require.True(t, errors.Is(err, ErrInvoiceClosed))
	var count int64
	require.NoError(t, db.Model(&domain.Payment{}).Count(&count).Error)
	require.Zero(t, count)
}
