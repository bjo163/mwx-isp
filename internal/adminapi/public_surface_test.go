package adminapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// anonymousRequest sends a request with no Authorization header at all, which is
// what a visitor to /home or /portal issues from the browser.
func anonymousRequest(t *testing.T, handler http.Handler, method, path string, payload any) (int, []byte) {
	t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.Bytes()
}

// The landing page (/home) and the customer portal (/portal) render without a
// session, but the catalogue, bill lookup, voucher check and installation
// intake APIs they advertise are not in the JWT skip list. They must answer 401
// anonymously so those pages can state honestly that a staff login is required,
// and so nobody repairs the public copy by opening customer self-service, which
// TR-N002 keeps out of scope.
func TestLandingSurfaceEndpointsFailClosedAnonymously(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)

	routes := []struct {
		name    string
		method  string
		path    string
		payload any
	}{
		{"package catalogue", http.MethodGet, "/api/v1/public/packages", nil},
		{"new installation intake", http.MethodPost, "/api/v1/public/register", map[string]string{
			"name": "Anonymous Visitor", "phone": "081234560000", "address": "Jl. Uji No. 1",
		}},
		{"voucher validity check", http.MethodGet, "/api/v1/public/vouchers/check?code=VCR-ANON", nil},
		{"customer bill lookup", http.MethodGet, "/api/v1/portal/lookup?q=CUST-ANON", nil},
	}

	for _, route := range routes {
		t.Run(route.name+" is refused without a token", func(t *testing.T) {
			status, body := anonymousRequest(t, handler, route.method, route.path, route.payload)
			assert.Equalf(t, http.StatusUnauthorized, status, "%s -> %s", route.path, body)
		})
	}

	t.Run("the refused intake stored nothing", func(t *testing.T) {
		var customers, tickets int64
		require.NoError(t, db.Model(&domain.Customer{}).Count(&customers).Error)
		require.NoError(t, db.Model(&domain.TroubleTicket{}).Count(&tickets).Error)
		assert.Zero(t, customers, "anonymous intake must not create a customer")
		assert.Zero(t, tickets, "anonymous intake must not create an installation ticket")
	})

	t.Run("an authenticated operator does get the catalogue", func(t *testing.T) {
		profile := domain.RadiusProfile{TenantID: domain.DefaultTenantID, Name: "surface-profile", Status: "enabled"}
		require.NoError(t, db.Create(&profile).Error)
		pkg := domain.InternetPackage{
			TenantID: domain.DefaultTenantID, Code: "PKG-SURFACE", Name: "Surface Package",
			Price: 150000, RadiusProfileID: profile.ID, Status: "active",
		}
		require.NoError(t, db.Create(&pkg).Error)

		token := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9151, LevelOperator, "surface-operator"))
		status, body := matrixRequest(t, handler, http.MethodGet, "/api/v1/public/packages", token, nil)
		require.Equalf(t, http.StatusOK, status, "%s", body)

		var packages []map[string]any
		decodeMatrixData(t, body, &packages)
		assert.Len(t, packages, 1, "the operator sees the catalogue the visitor is withheld")
	})
}
