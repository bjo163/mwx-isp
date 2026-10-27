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

// TestPrivilegeDowngradeInvalidatesLiveToken proves ENG-017/TR-F033: a
// downgrade or revocation performed through the real mutation endpoints
// (updateOperator / deleteOperator) bumps TenantMembership.TokenVersion via the
// SysOpr.AfterUpdate callback, so a token issued before the change is rejected
// immediately on the next request -- no waiting for JWT expiry.
//
// The existing TestTenantContextMiddlewareRejectsRevokedMembership only covers a
// direct DB update against the middleware; this test exercises the public
// operator-management API end-to-end instead.
func TestPrivilegeDowngradeInvalidatesLiveToken(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)

	// Caller is a super admin; the target is an admin whose live token we will
	// invalidate in three separate ways.
	superToken := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9301, LevelSuper, "matrix-super"))
	target := seedMatrixOperator(t, db, 9302, LevelAdmin, "matrix-target")
	targetToken := matrixToken(t, appCtx, target)

	// Baseline: the target's token is live and has admin privileges.
	status, body := matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/customers", targetToken,
		map[string]string{"name": "Pre-downgrade Customer"})
	require.Equalf(t, http.StatusCreated, status, "%s", body)

	// 1) Downgrade the target from admin to operator via the public API.
	status, body = matrixRequest(t, handler, http.MethodPut,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), superToken,
		map[string]string{"level": LevelOperator})
	require.Equalf(t, http.StatusOK, status, "%s", body)

	// The old token must now be rejected on a real mutation endpoint. The
	// downgrade bumped TenantMembership.TokenVersion, so the stale token's
	// membership_version claim no longer matches and resolveOperatorFromContext
	// refuses it; RequireLevel surfaces that as a uniform 401.
	status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/customers", targetToken,
		map[string]string{"name": "Should Fail"})
	assert.Equalf(t, http.StatusUnauthorized, status, "%s", body)

	// Confirm the bump actually happened on the membership row.
	var membershipAfterDowngrade domain.TenantMembership
	require.NoError(t, db.Where("operator_id = ?", target.ID).First(&membershipAfterDowngrade).Error)
	assert.Greater(t, membershipAfterDowngrade.TokenVersion, target.MembershipVersion,
		"downgrade must bump the membership token version so the old token is rejected")

	// 2) Disable the target account via the public API.
	status, body = matrixRequest(t, handler, http.MethodPut,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), superToken,
		map[string]string{"status": common.DISABLED})
	require.Equalf(t, http.StatusOK, status, "%s", body)

	// Confirm the disable persisted through the membership row.
	var membership domain.TenantMembership
	require.NoError(t, db.Where("operator_id = ?", target.ID).First(&membership).Error)
	assert.Equal(t, common.DISABLED, membership.Status)

	// 3) Delete the target operator via the public API.
	status, body = matrixRequest(t, handler, http.MethodDelete,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), superToken, nil)
	require.Equalf(t, http.StatusOK, status, "%s", body)

	// The operator row and its membership grant must both be gone.
	var count int64
	require.NoError(t, db.Model(&domain.SysOpr{}).Where("id = ?", target.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)
	require.NoError(t, db.Model(&domain.TenantMembership{}).Where("operator_id = ?", target.ID).Count(&count).Error)
	assert.Equal(t, int64(0), count)

	// 4) Boundary guard: the deleted operator must not resolve for detail or
	// update, so the downgrade cannot be re-applied or recovered through a
	// stale path.
	status, body = matrixRequest(t, handler, http.MethodGet,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), superToken, nil)
	assert.Equalf(t, http.StatusNotFound, status, "%s", body)

	status, body = matrixRequest(t, handler, http.MethodPut,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), superToken,
		map[string]string{"level": LevelOperator})
	assert.Equalf(t, http.StatusNotFound, status, "%s", body)

	// 5) Cross-tenant boundary: a token issued for tenant B must never resolve
	// rows that belong to tenant A, and must never write into tenant A's data.
	// This is the second half of the acceptance criterion -- the old role
	// loses access AND cross-tenant access is 404/403 with no side effect.
	const otherTenantID int64 = 2
	require.NoError(t, db.Create(&domain.Tenant{
		ID: otherTenantID, Name: "Other ISP", Slug: "other", Kind: "isp", Status: "active",
	}).Error)
	otherOperator := domain.SysOpr{
		ID: 9303, TenantID: otherTenantID, Username: "other-target",
		Password: "unused-in-this-test", Level: LevelAdmin, Status: "enabled",
	}
	// SysOpr.AfterCreate provisions the membership grant (token version 1).
	require.NoError(t, db.Create(&otherOperator).Error)
	var otherMembership domain.TenantMembership
	require.NoError(t, db.Where("operator_id = ?", otherOperator.ID).First(&otherMembership).Error)
	otherOperator.MembershipVersion = otherMembership.TokenVersion
	otherToken := matrixToken(t, appCtx, otherOperator)

	// The operator in tenant B may legitimately create a customer in tenant B.
	status, body = matrixRequest(t, handler, http.MethodPost, "/api/v1/isp/customers", otherToken,
		map[string]string{"name": "Tenant B Customer"})
	require.Equalf(t, http.StatusCreated, status, "%s", body)

	// But it must never see tenant A's customer: a cross-tenant detail lookup
	// must 404 rather than leak the row.
	var tenantACustomer domain.Customer
	require.NoError(t, db.Where("tenant_id = ?", domain.DefaultTenantID).First(&tenantACustomer).Error)
	status, body = matrixRequest(t, handler, http.MethodGet,
		fmt.Sprintf("/api/v1/isp/customers/%d", tenantACustomer.ID), otherToken, nil)
	assert.Equalf(t, http.StatusNotFound, status, "%s", body)

	// And it must never modify tenant A's customer: a cross-tenant update must
	// 404 and leave the original row untouched.
	status, body = matrixRequest(t, handler, http.MethodPut,
		fmt.Sprintf("/api/v1/isp/customers/%d", tenantACustomer.ID), otherToken,
		map[string]string{"name": "Cross Tenant Overwrite"})
	assert.Equalf(t, http.StatusNotFound, status, "%s", body)
	var persisted domain.Customer
	require.NoError(t, db.First(&persisted, tenantACustomer.ID).Error)
	assert.Equal(t, tenantACustomer.Name, persisted.Name,
		"cross-tenant update must not modify the row")

	// A cross-tenant operator detail lookup must 404 for the caller.
	status, body = matrixRequest(t, handler, http.MethodGet,
		fmt.Sprintf("/api/v1/system/operators/%d", target.ID), otherToken, nil)
	assert.Equalf(t, http.StatusNotFound, status, "%s", body)
}
