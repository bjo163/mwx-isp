package adminapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestManualSuspendSurvivesFullPayment drives the operator suspend action and a
// full manual payment through the real handlers: the payment must settle the
// invoice without clearing the manual suspension or re-enabling the account.
func TestManualSuspendSurvivesFullPayment(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.Subscription{}, &domain.Invoice{}, &domain.Payment{},
		&domain.BillingEvent{}, &domain.DocumentSequence{},
	))
	customer := domain.Customer{TenantID: domain.DefaultTenantID, CustomerNo: "CUS-MANUAL", Name: "Manual Customer", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{TenantID: domain.DefaultTenantID, Username: "manual-e2e", Password: "secret", Status: "enabled"}
	require.NoError(t, db.Create(&user).Error)
	sub := domain.Subscription{
		TenantID: domain.DefaultTenantID, SubscriptionNo: "SUB-MANUAL-E2E",
		CustomerID: customer.ID, RadiusUserID: user.ID, Status: domain.SubscriptionActive,
	}
	require.NoError(t, db.Create(&sub).Error)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-MANUAL-E2E", CustomerID: customer.ID,
		SubscriptionID: sub.ID, Total: 100000, Balance: 100000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)

	newContext := func(method, path, body string) (echo.Context, *httptest.ResponseRecorder) {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.Set("tenant_id", domain.DefaultTenantID)
		return c, rec
	}

	suspendCtx, suspendRec := newContext(http.MethodPost, fmt.Sprintf("/api/v1/isp/subscriptions/%d/suspend", sub.ID), "")
	suspendCtx.SetParamNames("id", "action")
	suspendCtx.SetParamValues(fmt.Sprint(sub.ID), "suspend")
	require.NoError(t, subscriptionAction(suspendCtx))
	require.Equal(t, http.StatusOK, suspendRec.Code, suspendRec.Body.String())

	require.NoError(t, db.First(&sub, sub.ID).Error)
	require.Equal(t, domain.SubscriptionSuspended, sub.Status)
	require.Equal(t, "manual", sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	require.Equal(t, "disabled", user.Status)

	paymentCtx, paymentRec := newContext(http.MethodPost, "/api/v1/isp/payments",
		fmt.Sprintf(`{"invoice_id":"%d","amount":100000,"method":"cash"}`, invoice.ID))
	require.NoError(t, createPayment(paymentCtx))
	require.Equal(t, http.StatusCreated, paymentRec.Code, paymentRec.Body.String())

	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	assert.Equal(t, domain.InvoicePaid, updated.Status)

	require.NoError(t, db.First(&sub, sub.ID).Error)
	assert.Equal(t, domain.SubscriptionSuspended, sub.Status)
	assert.Equal(t, "manual", sub.SuspensionReason)
	require.NoError(t, db.First(&user, user.ID).Error)
	assert.Equal(t, "disabled", user.Status)

	var reactivated int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("subscription_id = ? AND type = ?", sub.ID, "subscription_reactivated").Count(&reactivated).Error)
	assert.Zero(t, reactivated)
}
