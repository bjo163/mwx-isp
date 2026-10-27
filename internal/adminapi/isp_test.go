package adminapi

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestPackageCodeIsGeneratedAndStableOnEdit(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.InternetPackage{}, &domain.Subscription{}, &domain.Invoice{}))
	profile := domain.RadiusProfile{Name: "Base profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)

	create := func(name string) domain.InternetPackage {
		body := fmt.Sprintf(`{"name":%q,"price":100000,"radius_profile_id":"%d"}`, name, profile.ID)
		req := httptest.NewRequest(http.MethodPost, "/isp/packages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		require.NoError(t, createPackage(c))
		require.Equal(t, http.StatusCreated, rec.Code)
		var response struct {
			Data domain.InternetPackage `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		return response.Data
	}

	first := create("Home 100")
	second := create("Home 200")
	require.Equal(t, "PKG-000001", first.Code)
	require.Equal(t, "PKG-000002", second.Code)

	body := fmt.Sprintf(`{"name":"Updated package","price":120000,"radius_profile_id":"%d"}`, profile.ID)
	req := httptest.NewRequest(http.MethodPut, "/isp/packages/1", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id")
	c.SetParamValues("1")
	require.NoError(t, updatePackage(c))

	var updated domain.InternetPackage
	require.NoError(t, db.First(&updated, first.ID).Error)
	require.Equal(t, "PKG-000001", updated.Code)
	require.Equal(t, "Updated package", updated.Name)
}

func TestListPublicPackagesDoesNotInventOffersWhenCatalogIsEmpty(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.InternetPackage{}))

	req := httptest.NewRequest(http.MethodGet, "/public/packages", nil)
	rec := httptest.NewRecorder()
	ctx := CreateTestContext(e, db, req, rec, appCtx)

	require.NoError(t, listPublicPackages(ctx))
	require.Equal(t, http.StatusOK, rec.Code)
	require.JSONEq(t, `{"data":[]}`, rec.Body.String())
}

func TestLookupCustomerPortal(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.InternetPackage{}, &domain.Subscription{}, &domain.Invoice{}))

	cust := domain.Customer{
		CustomerNo: "CUST-0001",
		Name:       "Budi Santoso",
		Phone:      "08123456789",
		IdentityNo: "3201010101010001",
		Status:     "active",
	}
	require.NoError(t, db.Create(&cust).Error)

	pkg := domain.InternetPackage{
		Code:  "PKG-001",
		Name:  "50 Mbps Home",
		Price: 250000,
	}
	require.NoError(t, db.Create(&pkg).Error)

	sub := domain.Subscription{
		CustomerID: cust.ID,
		PackageID:  pkg.ID,
		Status:     domain.SubscriptionActive,
	}
	require.NoError(t, db.Create(&sub).Error)

	inv := domain.Invoice{
		CustomerID:     cust.ID,
		SubscriptionID: sub.ID,
		InvoiceNo:      "INV-2026-0001",
		Total:          250000,
		Balance:        250000,
		Status:         domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&inv).Error)

	// 1. Query by customer_no
	req := httptest.NewRequest(http.MethodGet, "/portal/lookup?q=CUST-0001", nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, lookupCustomerPortal(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			CustomerNo         string `json:"customer_no"`
			Name               string `json:"name"`
			Status             string `json:"status"`
			PackageName        string `json:"package_name"`
			PackagePrice       int64  `json:"package_price"`
			SubscriptionStatus string `json:"subscription_status"`
			Outstanding        int64  `json:"outstanding"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, "CUST-0001", resp.Data.CustomerNo)
	require.Equal(t, "Budi Santoso", resp.Data.Name)
	require.Equal(t, "50 Mbps Home", resp.Data.PackageName)
	require.Equal(t, int64(250000), resp.Data.PackagePrice)
	require.Equal(t, int64(250000), resp.Data.Outstanding)

	// 2. Query by phone
	req2 := httptest.NewRequest(http.MethodGet, "/portal/lookup?q=08123456789", nil)
	rec2 := httptest.NewRecorder()
	c2 := CreateTestContext(e, db, req2, rec2, appCtx)
	require.NoError(t, lookupCustomerPortal(c2))
	require.Equal(t, http.StatusOK, rec2.Code)

	// 3. Query non-existent
	req3 := httptest.NewRequest(http.MethodGet, "/portal/lookup?q=UNKNOWN", nil)
	rec3 := httptest.NewRecorder()
	c3 := CreateTestContext(e, db, req3, rec3, appCtx)
	require.NoError(t, lookupCustomerPortal(c3))
	require.Equal(t, http.StatusNotFound, rec3.Code)
}

func TestSubscriptionDisconnectAction(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.Subscription{}, &domain.RadiusUser{}, &domain.RadiusOnline{}, &domain.NetNas{}))

	user := domain.RadiusUser{
		Username: "subuser01",
		Status:   "enabled",
	}
	require.NoError(t, db.Create(&user).Error)

	sub := domain.Subscription{
		CustomerID:   1,
		RadiusUserID: user.ID,
		Status:       domain.SubscriptionActive,
	}
	require.NoError(t, db.Create(&sub).Error)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/isp/subscriptions/%d/disconnect", sub.ID), nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id", "action")
	c.SetParamValues(fmt.Sprintf("%d", sub.ID), "disconnect")

	require.NoError(t, subscriptionAction(c))
	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Data struct {
			DisconnectedSessions int `json:"disconnected_sessions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Data.DisconnectedSessions)
}

// TestSubscriptionSuspendQueuesDurableDisconnect proves an operator suspension
// no longer fails with a NAS error after the database state has already changed:
// the Disconnect-Request is recorded in the durable queue and either converged
// immediately (ACK) or retried later (timeout). Repeating the suspension must not
// create a second action for the same online session.
func TestSubscriptionSuspendQueuesDurableDisconnect(t *testing.T) {
	for _, tc := range []struct {
		name       string
		replyCode  radius.Code // 0 drops the request, forcing a client timeout
		wantStatus string
	}{
		{"nas acknowledges", radius.CodeDisconnectACK, sessionqueue.StatusSucceeded},
		{"nas timeout stays pending", 0, sessionqueue.StatusPending},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, e, appCtx := CreateTestAppContext(t)
			require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.Subscription{}, &domain.BillingEvent{}))

			restore := sessionCoAService
			sessionCoAService = func() *radiusd.CoAService {
				return radiusd.NewCoAService(nil,
					radiusd.WithCoATimeout(120*time.Millisecond),
					radiusd.WithCoARetries(0))
			}
			t.Cleanup(func() { sessionCoAService = restore })

			nas := newFakeCoANAS(t, coaTestSecret, tc.replyCode, 0)
			host, portText, err := net.SplitHostPort(nas.addr)
			require.NoError(t, err)
			port, err := strconv.Atoi(portText)
			require.NoError(t, err)
			require.NoError(t, db.Create(&domain.NetNas{
				Name: "manual-nas", Identifier: "manual-nas", Ipaddr: host,
				Secret: coaTestSecret, CoaPort: port, Status: "enabled",
			}).Error)

			user := domain.RadiusUser{Username: "manual-suspend-user", Password: "secret", Status: "enabled"}
			require.NoError(t, db.Create(&user).Error)
			customer := domain.Customer{Name: "Manual Suspend", Status: domain.CustomerActive}
			require.NoError(t, db.Create(&customer).Error)
			sub := domain.Subscription{
				CustomerID: customer.ID, PackageID: 1, RadiusUserID: user.ID,
				Status: domain.SubscriptionActive, StartDate: time.Now(),
			}
			require.NoError(t, db.Create(&sub).Error)
			session := domain.RadiusOnline{
				Username: user.Username, NasId: "manual-nas", NasAddr: host,
				AcctSessionId: "manual-suspend-session", AcctStartTime: time.Now(), LastUpdate: time.Now(),
			}
			require.NoError(t, db.Create(&session).Error)

			suspend := func() *httptest.ResponseRecorder {
				req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/isp/subscriptions/%d/suspend", sub.ID), nil)
				rec := httptest.NewRecorder()
				c := CreateTestContext(e, db, req, rec, appCtx)
				c.SetParamNames("id", "action")
				c.SetParamValues(fmt.Sprintf("%d", sub.ID), "suspend")
				require.NoError(t, subscriptionAction(c))
				return rec
			}

			rec := suspend()
			require.Equalf(t, http.StatusOK, rec.Code,
				"a failed NAS exchange must not fail an already-committed suspension: %s", rec.Body.String())

			require.NoError(t, db.First(&sub, sub.ID).Error)
			require.Equal(t, domain.SubscriptionSuspended, sub.Status)
			require.Equal(t, "manual", sub.SuspensionReason)
			require.NoError(t, db.First(&user, user.ID).Error)
			require.Equal(t, "disabled", user.Status)

			var action domain.RadiusSessionAction
			require.NoError(t, db.Where("session_id = ?", session.ID).First(&action).Error)
			require.Equal(t, sessionqueue.ReasonManualSuspension, action.Reason)
			require.Equal(t, tc.wantStatus, action.Status)
			require.EqualValues(t, 1, action.Attempts)

			var audits int64
			require.NoError(t, db.Model(&domain.RadiusSessionActionAudit{}).
				Where("session_id = ?", session.ID).Count(&audits).Error)
			require.EqualValues(t, 1, audits, "every attempt is audited exactly once")

			rec = suspend()
			require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
			var actions int64
			require.NoError(t, db.Model(&domain.RadiusSessionAction{}).
				Where("session_id = ?", session.ID).Count(&actions).Error)
			require.EqualValues(t, 1, actions, "repeating the suspension must not queue a duplicate action")
		})
	}
}

func TestSendInvoiceWhatsAppValidation(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.Invoice{}))

	// 1. Non-existent invoice
	req := httptest.NewRequest(http.MethodPost, "/isp/invoices/999/send-whatsapp", nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id")
	c.SetParamValues("999")
	require.NoError(t, sendInvoiceWhatsApp(c))
	require.Equal(t, http.StatusNotFound, rec.Code)

	// 2. Invoice with customer that has no phone
	cust := domain.Customer{Name: "No Phone Customer", Phone: ""}
	require.NoError(t, db.Create(&cust).Error)
	inv := domain.Invoice{CustomerID: cust.ID, InvoiceNo: "INV-001", Total: 100000}
	require.NoError(t, db.Create(&inv).Error)

	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/isp/invoices/%d/send-whatsapp", inv.ID), nil)
	rec2 := httptest.NewRecorder()
	c2 := CreateTestContext(e, db, req2, rec2, appCtx)
	c2.SetParamNames("id")
	c2.SetParamValues(fmt.Sprintf("%d", inv.ID))
	require.NoError(t, sendInvoiceWhatsApp(c2))
	require.Equal(t, http.StatusBadRequest, rec2.Code)
}

func TestUnconfiguredPaymentEndpointsFailClosed(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Invoice{}, &domain.Payment{}))

	inv := domain.Invoice{InvoiceNo: "INV-202610-009999", Total: 250000, Balance: 250000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&inv).Error)

	tests := []struct {
		name       string
		path       string
		body       string
		statusCode int
		handler    func(echo.Context) error
	}{
		{name: "unsigned webhook", path: "/portal/payments/webhook", body: `{"invoice_no":"INV-202610-009999","amount":250000,"status":"paid"}`, statusCode: http.StatusServiceUnavailable, handler: handlePaymentWebhook},
		{name: "simulated payment", path: fmt.Sprintf("/portal/invoices/%d/simulate-pay", inv.ID), body: `{"method":"qris_instant","reference":"SIM-123"}`, statusCode: http.StatusGone, handler: simulateInvoicePayment},
		{name: "unconfigured payment channel", path: fmt.Sprintf("/portal/invoices/%d/payment-channel", inv.ID), statusCode: http.StatusServiceUnavailable, handler: getInvoicePaymentChannel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			ctx := CreateTestContext(e, db, req, rec, appCtx)
			if tt.name != "unsigned webhook" {
				ctx.SetParamNames("id")
				ctx.SetParamValues(fmt.Sprintf("%d", inv.ID))
			}
			require.NoError(t, tt.handler(ctx))
			require.Equal(t, tt.statusCode, rec.Code)

			var current domain.Invoice
			require.NoError(t, db.First(&current, inv.ID).Error)
			require.Equal(t, domain.InvoiceIssued, current.Status)
			require.Equal(t, int64(250000), current.Balance)
			require.Zero(t, current.PaidAmount)

			var paymentCount int64
			require.NoError(t, db.Model(&domain.Payment{}).Count(&paymentCount).Error)
			require.Zero(t, paymentCount)
		})
	}
}
func TestFUPApplyAndReset(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.InternetPackage{}, &domain.Subscription{}, &domain.RadiusProfile{}, &domain.RadiusUser{}, &domain.BillingEvent{}))

	profile := domain.RadiusProfile{Name: "50M Profile", UpRate: 50000, DownRate: 50000, Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)

	pkg := domain.InternetPackage{
		Name:            "Giga 50",
		Price:           300000,
		RadiusProfileID: profile.ID,
		FupLimitGB:      500,
		FupRateDown:     2048,
		FupRateUp:       1024,
		Status:          "active",
	}
	require.NoError(t, db.Create(&pkg).Error)

	cust := domain.Customer{Name: "Eko Pratama", CustomerNo: "CUST-002", Status: "active"}
	require.NoError(t, db.Create(&cust).Error)

	rUser := domain.RadiusUser{Username: "eko50", UpRate: 50000, DownRate: 50000, Status: "enabled"}
	require.NoError(t, db.Create(&rUser).Error)

	sub := domain.Subscription{
		CustomerID:   cust.ID,
		PackageID:    pkg.ID,
		RadiusUserID: rUser.ID,
		Status:       domain.SubscriptionActive,
	}
	require.NoError(t, db.Create(&sub).Error)

	// 1. Apply FUP
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/isp/subscriptions/%d/apply-fup", sub.ID), nil)
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", sub.ID))

	require.NoError(t, applySubscriptionFUP(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var checkSub domain.Subscription
	require.NoError(t, db.First(&checkSub, sub.ID).Error)
	require.True(t, checkSub.FupTriggered)

	var checkUser domain.RadiusUser
	require.NoError(t, db.First(&checkUser, rUser.ID).Error)
	require.Equal(t, 1024, checkUser.UpRate)
	require.Equal(t, 2048, checkUser.DownRate)

	// 2. Reset FUP
	req2 := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/isp/subscriptions/%d/reset-fup", sub.ID), nil)
	rec2 := httptest.NewRecorder()
	c2 := CreateTestContext(e, db, req2, rec2, appCtx)
	c2.SetParamNames("id")
	c2.SetParamValues(fmt.Sprintf("%d", sub.ID))

	require.NoError(t, resetSubscriptionFUP(c2))
	require.Equal(t, http.StatusOK, rec2.Code)

	require.NoError(t, db.First(&checkSub, sub.ID).Error)
	require.False(t, checkSub.FupTriggered)

	require.NoError(t, db.First(&checkUser, rUser.ID).Error)
	require.Equal(t, 50000, checkUser.UpRate)
	require.Equal(t, 50000, checkUser.DownRate)
}

func TestInvoicePaymentChannelAndSimulatePayFailClosed(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.Invoice{}, &domain.Payment{}))

	customer := domain.Customer{Name: "Test Customer", CustomerNo: "CUST-0099", Status: "active"}
	require.NoError(t, db.Create(&customer).Error)
	invoice := domain.Invoice{CustomerID: customer.ID, InvoiceNo: "INV-202610-008888", Total: 250000, Balance: 250000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/portal/invoices/%d/payment-channel", invoice.ID), nil)
	rec := httptest.NewRecorder()
	ctx := CreateTestContext(e, db, req, rec, appCtx)
	ctx.SetParamNames("id")
	ctx.SetParamValues(fmt.Sprintf("%d", invoice.ID))
	require.NoError(t, getInvoicePaymentChannel(ctx))
	require.Equal(t, http.StatusServiceUnavailable, rec.Code)

	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/portal/invoices/%d/simulate-pay", invoice.ID), strings.NewReader(`{"method":"qris_instant"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	ctx = CreateTestContext(e, db, req, rec, appCtx)
	ctx.SetParamNames("id")
	ctx.SetParamValues(fmt.Sprintf("%d", invoice.ID))
	require.NoError(t, simulateInvoicePayment(ctx))
	require.Equal(t, http.StatusGone, rec.Code)

	var persisted domain.Invoice
	require.NoError(t, db.First(&persisted, invoice.ID).Error)
	require.Equal(t, domain.InvoiceIssued, persisted.Status)
	require.Equal(t, int64(250000), persisted.Balance)
	require.Zero(t, persisted.PaidAmount)

	var payments int64
	require.NoError(t, db.Model(&domain.Payment{}).Count(&payments).Error)
	require.Zero(t, payments)
}

func TestCustomerODPPortValidation(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.Customer{}, &domain.ODP{}))

	odp := domain.ODP{
		Code:       "ODP-TEST-01",
		Name:       "ODP Test 1",
		TotalPorts: 8,
		Status:     "active",
	}
	require.NoError(t, db.Create(&odp).Error)

	// 1. Assign Customer 1 to Port 1 (Success)
	body1 := fmt.Sprintf(`{"name":"Pelanggan Satu","odp_id":"%d","odp_port":1}`, odp.ID)
	req1 := httptest.NewRequest(http.MethodPost, "/isp/customers", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	rec1 := httptest.NewRecorder()
	c1 := CreateTestContext(e, db, req1, rec1, appCtx)
	require.NoError(t, createCustomer(c1))
	require.Equal(t, http.StatusCreated, rec1.Code)

	var cust1 struct {
		Data domain.Customer `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec1.Body.Bytes(), &cust1))
	require.Equal(t, 1, cust1.Data.ODPPort)
	require.Equal(t, "ODP-TEST-01", cust1.Data.ODPCode)

	// 2. Assign Customer 2 to same Port 1 (Conflict 409)
	bodyConflict := fmt.Sprintf(`{"name":"Pelanggan Dua","odp_id":"%d","odp_port":1}`, odp.ID)
	reqConflict := httptest.NewRequest(http.MethodPost, "/isp/customers", strings.NewReader(bodyConflict))
	reqConflict.Header.Set("Content-Type", "application/json")
	recConflict := httptest.NewRecorder()
	cConflict := CreateTestContext(e, db, reqConflict, recConflict, appCtx)
	require.NoError(t, createCustomer(cConflict))
	require.Equal(t, http.StatusConflict, recConflict.Code)

	// 3. Assign Customer 2 to port exceeding total_ports (Bad Request 400)
	bodyExceed := fmt.Sprintf(`{"name":"Pelanggan Dua","odp_id":"%d","odp_port":9}`, odp.ID)
	reqExceed := httptest.NewRequest(http.MethodPost, "/isp/customers", strings.NewReader(bodyExceed))
	reqExceed.Header.Set("Content-Type", "application/json")
	recExceed := httptest.NewRecorder()
	cExceed := CreateTestContext(e, db, reqExceed, recExceed, appCtx)
	require.NoError(t, createCustomer(cExceed))
	require.Equal(t, http.StatusBadRequest, recExceed.Code)

	// 4. Update Customer 1 to Port 2 (Success)
	updateBody := fmt.Sprintf(`{"name":"Pelanggan Satu Updated","odp_id":"%d","odp_port":2}`, odp.ID)
	reqUpdate := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/isp/customers/%d", cust1.Data.ID), strings.NewReader(updateBody))
	reqUpdate.Header.Set("Content-Type", "application/json")
	recUpdate := httptest.NewRecorder()
	cUpdate := CreateTestContext(e, db, reqUpdate, recUpdate, appCtx)
	cUpdate.SetParamNames("id")
	cUpdate.SetParamValues(fmt.Sprintf("%d", cust1.Data.ID))
	require.NoError(t, updateCustomer(cUpdate))
	require.Equal(t, http.StatusOK, recUpdate.Code)

	// 5. Now Port 1 is free, Customer 2 can take Port 1 (Success)
	req2 := httptest.NewRequest(http.MethodPost, "/isp/customers", strings.NewReader(body1))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	c2 := CreateTestContext(e, db, req2, rec2, appCtx)
	require.NoError(t, createCustomer(c2))
	require.Equal(t, http.StatusCreated, rec2.Code)

	// 6. Moving Customer 1 onto a port held by another live customer conflicts.
	reqMove := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/isp/customers/%d", cust1.Data.ID),
		strings.NewReader(body1))
	reqMove.Header.Set("Content-Type", "application/json")
	recMove := httptest.NewRecorder()
	cMove := CreateTestContext(e, db, reqMove, recMove, appCtx)
	cMove.SetParamNames("id")
	cMove.SetParamValues(fmt.Sprintf("%d", cust1.Data.ID))
	require.NoError(t, updateCustomer(cMove))
	require.Equal(t, http.StatusConflict, recMove.Code)

	// 7. Keeping the customer on its own port must not conflict with itself.
	reqSelf := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/isp/customers/%d", cust1.Data.ID),
		strings.NewReader(updateBody))
	reqSelf.Header.Set("Content-Type", "application/json")
	recSelf := httptest.NewRecorder()
	cSelf := CreateTestContext(e, db, reqSelf, recSelf, appCtx)
	cSelf.SetParamNames("id")
	cSelf.SetParamValues(fmt.Sprintf("%d", cust1.Data.ID))
	require.NoError(t, updateCustomer(cSelf))
	require.Equal(t, http.StatusOK, recSelf.Code)
}

// TestODPShrinkGuard verifies an ODP cannot be shrunk below a port that is
// still assigned to a live customer.
func TestODPShrinkGuard(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.ODP{}, &domain.Customer{}))

	odp := domain.ODP{Code: "ODP-SHRINK", Name: "ODP Shrink", TotalPorts: 16, Status: "active"}
	require.NoError(t, db.Create(&odp).Error)
	require.NoError(t, db.Create(&domain.Customer{
		CustomerNo: "C-SHRINK", Name: "Port Occupant", ODPID: odp.ID, ODPPort: 12, Status: domain.CustomerActive,
	}).Error)

	update := func(payload string) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(http.MethodPut,
			fmt.Sprintf("/api/v1/network/odp/%d", odp.ID), strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.SetParamNames("id")
		c.SetParamValues(fmt.Sprintf("%d", odp.ID))
		require.NoError(t, updateODP(c))
		return rec
	}

	rec := update(`{"total_ports":8}`)
	require.Equal(t, http.StatusConflict, rec.Code)
	require.Contains(t, rec.Body.String(), "port 12")

	var stored domain.ODP
	require.NoError(t, db.First(&stored, odp.ID).Error)
	require.Equal(t, 16, stored.TotalPorts)

	rec = update(`{"total_ports":12}`)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, db.First(&stored, odp.ID).Error)
	require.Equal(t, 12, stored.TotalPorts)
}
