package suspendtrace

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var traceDBSeq atomic.Int64

func traceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:suspendtrace-%d?mode=memory&cache=shared", traceDBSeq.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.Subscription{}, &domain.Invoice{},
		&domain.BillingEvent{}, &domain.RadiusUser{},
	))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

var traceNow = time.Date(2026, time.October, 29, 10, 0, 0, 0, time.UTC)

type traceFixture struct {
	db      *gorm.DB
	sub     domain.Subscription
	invoice domain.Invoice
	user    domain.RadiusUser
}

// suspendedFixture records the rows billing.SuspendOverdueSubscriptions writes:
// the subscription reason, the disabled linked user, and the suspension event.
func suspendedFixture(t *testing.T, decision time.Time) traceFixture {
	t.Helper()
	db := traceTestDB(t)

	user := domain.RadiusUser{Username: "trace-user", Status: "disabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{
		SubscriptionNo: "SUB-TRACE-1", CustomerID: 1, PackageID: 1, RadiusUserID: user.ID,
		Status: domain.SubscriptionSuspended, SuspensionReason: domain.SuspensionBillingOverdue,
		StartDate: traceNow.AddDate(0, -2, 0), BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, db.Create(&sub).Error)

	invoice := domain.Invoice{
		InvoiceNo: "INV-TRACE-1", CustomerID: 1, SubscriptionID: sub.ID,
		InvoiceDate: traceNow.AddDate(0, 0, -20), DueDate: traceNow.AddDate(0, 0, -11),
		PeriodStart: traceNow.AddDate(0, -1, 0), PeriodEnd: traceNow.AddDate(0, 0, -11),
		Subtotal: 150000, Total: 150000, Balance: 150000, Status: domain.InvoiceOverdue,
	}
	require.NoError(t, db.Create(&invoice).Error)

	require.NoError(t, db.Create(&domain.BillingEvent{
		CustomerID: 1, SubscriptionID: sub.ID, InvoiceID: invoice.ID,
		Type: "invoice_overdue", Description: invoice.InvoiceNo, CreatedAt: traceNow.AddDate(0, 0, -11),
	}).Error)
	require.NoError(t, db.Create(&domain.BillingEvent{
		CustomerID: 1, SubscriptionID: sub.ID, InvoiceID: invoice.ID,
		Type: "subscription_suspended", Description: domain.SuspensionBillingOverdue, CreatedAt: decision,
	}).Error)

	return traceFixture{db: db, sub: sub, invoice: invoice, user: user}
}

func stepByRule(t *testing.T, trace Trace, rule string) Step {
	t.Helper()
	for _, step := range trace.Steps {
		if step.Rule == rule {
			return step
		}
	}
	require.Failf(t, "step not found", "rule %q missing from %+v", rule, trace.Steps)
	return Step{}
}

func TestExplainReconstructsBillingSuspension(t *testing.T) {
	f := suspendedFixture(t, traceNow.AddDate(0, 0, -8))

	trace, err := Explain(f.db, f.sub.ID, traceNow)
	require.NoError(t, err)

	assert.Equal(t, domain.SubscriptionSuspended, trace.Status)
	assert.Equal(t, domain.SuspensionBillingOverdue, trace.SuspensionReason)
	assert.Contains(t, trace.Conclusion, "unpaid overdue invoice")

	assert.Equal(t, Matched, stepByRule(t, trace, "subscription state").Status)

	invoiceStep := stepByRule(t, trace, "overdue invoice")
	assert.Equal(t, Matched, invoiceStep.Status)
	assert.Contains(t, invoiceStep.Detail, "INV-TRACE-1")
	assert.Contains(t, invoiceStep.Detail, "balance=150000")
	assert.Contains(t, invoiceStep.Detail, "threshold=2026-10-21")
	assert.Equal(t, fmt.Sprintf("isp_invoice#%d", f.invoice.ID), invoiceStep.Evidence)

	// The service disables the linked user in the same transaction.
	radiusStep := stepByRule(t, trace, "radius user")
	assert.Equal(t, Matched, radiusStep.Status)
	assert.Contains(t, radiusStep.Detail, "status=disabled")

	eventStep := stepByRule(t, trace, "billing events")
	assert.Equal(t, Matched, eventStep.Status)
	assert.Contains(t, eventStep.Detail, "invoice_overdue@")
	assert.Contains(t, eventStep.Detail, "subscription_suspended@")
	assert.Contains(t, eventStep.Detail, fmt.Sprintf("invoice#%d", f.invoice.ID))

	// The decision inputs that were never persisted are named, not guessed.
	require.Len(t, trace.Gaps, 1)
	assert.Contains(t, trace.Gaps[0], "grace period in force at decision time is not stored")
}

func TestExplainShowsWhenTheOverdueInvoiceNoLongerExplainsTheSuspension(t *testing.T) {
	f := suspendedFixture(t, traceNow.AddDate(0, 0, -8))
	require.NoError(t, f.db.Model(&domain.Invoice{}).Where("id = ?", f.invoice.ID).
		Updates(map[string]any{"balance": 0, "status": domain.InvoicePaid}).Error)

	trace, err := Explain(f.db, f.sub.ID, traceNow)
	require.NoError(t, err)

	assert.Equal(t, NotMatched, stepByRule(t, trace, "overdue invoice").Status)
	assert.Contains(t, strings.Join(trace.Gaps, "\n"), "predates a payment or a status change")
}

func TestExplainDoesNotClaimAManualSuspensionCameFromTheBillingRule(t *testing.T) {
	f := suspendedFixture(t, traceNow.AddDate(0, 0, -8))
	require.NoError(t, f.db.Model(&domain.Subscription{}).Where("id = ?", f.sub.ID).
		Updates(map[string]any{"suspension_reason": ""}).Error)

	trace, err := Explain(f.db, f.sub.ID, traceNow)
	require.NoError(t, err)

	assert.Contains(t, trace.Conclusion, "does not identify the billing rule")
	assert.Contains(t, strings.Join(trace.Gaps, "\n"), "records neither the operator nor a reason")
	// The evidence is still shown, but it does not become the explanation.
	assert.Equal(t, Matched, stepByRule(t, trace, "overdue invoice").Status)
}

func TestExplainReportsAnActiveSubscription(t *testing.T) {
	db := traceTestDB(t)
	sub := domain.Subscription{
		SubscriptionNo: "SUB-TRACE-2", CustomerID: 1, PackageID: 1,
		Status: domain.SubscriptionActive, StartDate: traceNow.AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, db.Create(&sub).Error)

	trace, err := Explain(db, sub.ID, traceNow)
	require.NoError(t, err)

	assert.Contains(t, trace.Conclusion, "not suspended")
	assert.Equal(t, NotApplicable, stepByRule(t, trace, "subscription state").Status)
	assert.Equal(t, NotMatched, stepByRule(t, trace, "overdue invoice").Status)
	assert.Equal(t, NotApplicable, stepByRule(t, trace, "radius user").Status)
	assert.Equal(t, Unknown, stepByRule(t, trace, "billing events").Status)
}

func TestExplainFlagsAReEnabledRadiusUser(t *testing.T) {
	f := suspendedFixture(t, traceNow.AddDate(0, 0, -8))
	require.NoError(t, f.db.Model(&domain.RadiusUser{}).Where("id = ?", f.user.ID).
		Update("status", "active").Error)

	trace, err := Explain(f.db, f.sub.ID, traceNow)
	require.NoError(t, err)

	radiusStep := stepByRule(t, trace, "radius user")
	assert.Equal(t, NotMatched, radiusStep.Status)
	assert.Contains(t, radiusStep.Detail, "expects the linked user to be disabled")
}

func TestExplainRejectsAnUnknownSubscription(t *testing.T) {
	db := traceTestDB(t)
	_, err := Explain(db, 404, traceNow)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
}
