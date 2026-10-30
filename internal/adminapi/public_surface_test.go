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

// The package catalogue is a bounded, read-only public surface. Customer bill
// lookup, voucher check and installation intake remain authenticated and fail
// closed; TR-F034 does not open customer self-service.
func TestLandingSurfaceEndpointsFailClosedAnonymously(t *testing.T) {
	handler, db, _ := newRoleMatrixServer(t)

	routes := []struct {
		name    string
		method  string
		path    string
		payload any
	}{
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

	t.Run("anonymous visitors receive only the public read-only package catalogue", func(t *testing.T) {
		profile := domain.RadiusProfile{TenantID: domain.DefaultTenantID, Name: "surface-profile", UpRate: 100000, DownRate: 100000, Status: "enabled"}
		require.NoError(t, db.Create(&profile).Error)
		pkg := domain.InternetPackage{
			TenantID: domain.DefaultTenantID, Code: "PKG-SURFACE", Name: "Operational Package",
			Price: 150000, RadiusProfileID: profile.ID, Status: "active",
		}
		require.NoError(t, db.Create(&pkg).Error)
		require.NoError(t, db.Create(&domain.TenantPackagePresentation{
			TenantID: domain.DefaultTenantID, PackageID: pkg.ID, DisplayName: "Family Fibre 100",
			Category: "Home", Badge: "Popular", PublicVisible: true,
		}).Error)

		status, body := anonymousRequest(t, handler, http.MethodGet, "/api/v1/public/packages", nil)
		require.Equalf(t, http.StatusOK, status, "%s", body)
		var public struct { Data []map[string]any `json:"data"` }
		require.NoError(t, json.Unmarshal(body, &public))
		require.Len(t, public.Data, 1)
		assert.Equal(t, "Family Fibre 100", public.Data[0]["name"])
		assert.Equal(t, "Home", public.Data[0]["category"])
		assert.NotContains(t, public.Data[0], "customer_id")
		assert.NotContains(t, public.Data[0], "support_email")
	})
}
