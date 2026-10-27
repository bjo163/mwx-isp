package adminapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/webserver"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// newRoleMatrixServer boots the real router (including the JWT and tenant
// middleware) in-process so the assertions cover the registered guards, not a
// copy of them.
func newRoleMatrixServer(t *testing.T) (http.Handler, *gorm.DB, app.AppContext) {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{},
		&domain.InternetPackage{},
		&domain.Subscription{},
		&domain.Invoice{},
		&domain.Payment{},
		&domain.BillingEvent{},
		&domain.TroubleTicket{},
		&domain.HotspotBatch{},
		&domain.HotspotVoucher{},
		&domain.DocumentSequence{},
		&domain.NotificationSettings{},
		&domain.NotificationOutbox{},
	))
	appCtx := setupTestApp(t, db)
	webserver.Init(appCtx)
	Init(appCtx)
	return webserver.Handler(), db, appCtx
}

func seedMatrixOperator(t *testing.T, db *gorm.DB, id int64, level, username string) domain.SysOpr {
	t.Helper()
	operator := domain.SysOpr{
		ID: id, TenantID: domain.DefaultTenantID, Username: username,
		Password: "unused-in-this-test", Level: level, Status: "enabled", MembershipVersion: 1,
	}
	// SysOpr.AfterCreate provisions the membership grant (token version 1).
	require.NoError(t, db.Create(&operator).Error)
	return operator
}

func matrixToken(t *testing.T, appCtx app.AppContext, operator domain.SysOpr) string {
	t.Helper()
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":                fmt.Sprintf("%d", operator.ID),
		"tenant_id":          fmt.Sprintf("%d", operator.TenantID),
		"membership_version": operator.MembershipVersion,
		"username":           operator.Username,
		"role":               operator.Level,
		"exp":                now.Add(time.Hour).Unix(),
		"iat":                now.Unix(),
		"nbf":                now.Add(-time.Minute).Unix(),
		"iss":                "mwx-isp",
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(appCtx.Config().Web.Secret))
	require.NoError(t, err)
	return signed
}

func matrixRequest(t *testing.T, handler http.Handler, method, path, token string, payload any) (int, []byte) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

func decodeMatrixData(t *testing.T, body []byte, target any) {
	t.Helper()
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	require.NoErrorf(t, json.Unmarshal(body, &envelope), "response is not JSON: %s", body)
	require.NoError(t, json.Unmarshal(envelope.Data, target))
}

func seedMatrixInvoice(t *testing.T, db *gorm.DB, label string) (domain.Customer, domain.Invoice) {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Second)
	// The invoice uniqueness key includes (tenant, subscription, period start);
	// give each fixture its own period so they can share the empty subscription.
	periodStart := now.Add(-time.Duration(len(label)) * time.Minute)
	customer := domain.Customer{
		ID: common.UUIDint64(), TenantID: domain.DefaultTenantID,
		CustomerNo: "C-MX-" + label, Name: "Matrix " + label, Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&customer).Error)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-MX-" + label, CustomerID: customer.ID,
		InvoiceDate: now, DueDate: now.Add(72 * time.Hour), PeriodStart: periodStart, PeriodEnd: periodStart.Add(30 * 24 * time.Hour),
		Subtotal: 100000, Total: 100000, Balance: 100000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)
	return customer, invoice
}

func TestMutationRouteRoleMatrix(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)

	profile := domain.RadiusProfile{TenantID: domain.DefaultTenantID, Name: "matrix-profile", Status: "enabled"}
	require.NoError(t, db.Create(&profile).Error)
	pkg := domain.InternetPackage{
		TenantID: domain.DefaultTenantID, Code: "PKG-MATRIX", Name: "Matrix Package",
		Price: 100000, RadiusProfileID: profile.ID, Status: "active",
	}
	require.NoError(t, db.Create(&pkg).Error)

	adminToken := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9101, LevelAdmin, "matrix-admin"))
	operatorToken := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9102, LevelOperator, "matrix-operator"))

	t.Run("operator may read tenant data", func(t *testing.T) {
		status, body := matrixRequest(t, handler, http.MethodGet, "/api/v1/isp/customers", operatorToken, nil)
		assert.Equalf(t, http.StatusOK, status, "%s", body)
	})

	t.Run("customer mutation is admin only", func(t *testing.T) {
		status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/customers", operatorToken,
			map[string]string{"name": "Operator Attempt"})
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/customers", adminToken,
			map[string]string{"name": "Admin Customer"})
		assert.Equalf(t, http.StatusCreated, status, "%s", body)
	})

	t.Run("payment is admin only and has no operator side door", func(t *testing.T) {
		_, invoice := seedMatrixInvoice(t, db, "payment")
		payload := map[string]any{"invoice_id": fmt.Sprint(invoice.ID), "amount": int64(25000), "method": "cash"}

		status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/payments", operatorToken, payload)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/payments", adminToken, payload)
		assert.Equalf(t, http.StatusCreated, status, "%s", body)

		// The simulated payment route and the provider webhook are closed for
		// every role, so neither can be used to book money around the guard.
		status, body = matrixRequest(t, handler, http.MethodPost,
			fmt.Sprintf("/api/v1/portal/invoices/%d/simulate-pay", invoice.ID), operatorToken, nil)
		assert.Equalf(t, http.StatusGone, status, "%s", body)
		status, body = matrixRequest(t, handler, http.MethodPost,
			fmt.Sprintf("/api/v1/portal/invoices/%d/simulate-pay", invoice.ID), adminToken, nil)
		assert.Equalf(t, http.StatusGone, status, "%s", body)
		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/portal/payments/webhook", adminToken, nil)
		assert.Equalf(t, http.StatusServiceUnavailable, status, "%s", body)

		var payments int64
		require.NoError(t, db.Model(&domain.Payment{}).Where("invoice_id = ?", invoice.ID).Count(&payments).Error)
		assert.Equal(t, int64(1), payments)
	})

	t.Run("radius user mutation is admin only", func(t *testing.T) {
		payload := map[string]string{"profile_id": fmt.Sprint(profile.ID), "username": "matrix-user", "password": "secret123"}
		status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/users", operatorToken, payload)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/users", adminToken, payload)
		require.Equalf(t, http.StatusOK, status, "%s", body)
		var user domain.RadiusUser
		decodeMatrixData(t, body, &user)

		status, body = matrixRequest(t, handler, http.MethodPut, fmt.Sprintf("/api/v1/users/%d", user.ID), operatorToken, payload)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)
		status, body = matrixRequest(t, handler, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", user.ID), operatorToken, nil)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)
	})

	t.Run("operator may run the ticket workflow but not dispatch", func(t *testing.T) {
		status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/tickets", operatorToken, map[string]string{
			"subject": "Operator created ticket", "assigned_technician": "Tech A", "technician_phone": "081200000001",
		})
		require.Equalf(t, http.StatusCreated, status, "%s", body)
		var ticket domain.TroubleTicket
		decodeMatrixData(t, body, &ticket)

		status, body = matrixRequest(t, handler, http.MethodPut, fmt.Sprintf("/api/v1/isp/tickets/%d", ticket.ID), operatorToken,
			map[string]string{
				"subject": "Operator updated ticket", "status": "open",
				"assigned_technician": "Tech A", "technician_phone": "081200000001",
			})
		assert.Equalf(t, http.StatusOK, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost,
			fmt.Sprintf("/api/v1/isp/tickets/%d/dispatch", ticket.ID), operatorToken, nil)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost,
			fmt.Sprintf("/api/v1/isp/tickets/%d/dispatch", ticket.ID), adminToken, nil)
		// No WhatsApp recipient is configured in this fixture, so an honest
		// dispatch reports that nothing was queued instead of a fake success.
		require.Equalf(t, http.StatusConflict, status, "%s", body)
		assert.Contains(t, string(body), "NOTIFICATION_NOT_QUEUED")
	})

	t.Run("voucher generation is admin only", func(t *testing.T) {
		payload := map[string]any{"name": "Matrix Batch", "package_id": fmt.Sprint(pkg.ID), "quantity": 2, "price": int64(5000)}
		status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/vouchers/generate", operatorToken, payload)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/vouchers/generate", adminToken, payload)
		assert.Equalf(t, http.StatusOK, status, "%s", body)
	})

	t.Run("customer-facing sends and subscription actions are admin only", func(t *testing.T) {
		_, invoice := seedMatrixInvoice(t, db, "send")
		status, body := matrixRequest(t, handler, http.MethodPost,
			fmt.Sprintf("/api/v1/isp/invoices/%d/send-whatsapp", invoice.ID), operatorToken, nil)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/subscriptions/1/apply-fup", operatorToken, nil)
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)

		status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/packages", operatorToken, map[string]any{
			"name": "Operator Package", "price": int64(1000), "radius_profile_id": fmt.Sprint(profile.ID),
		})
		assert.Equalf(t, http.StatusForbidden, status, "%s", body)
	})
}
