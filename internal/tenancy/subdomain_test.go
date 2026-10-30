package tenancy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSubdomainResolverRequiresVerifiedCustomDomain(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.TenantDomain{}))
	active := domain.Tenant{ID: 7, Name: "Tenant Seven", Slug: "tenant-seven", Status: "active"}
	require.NoError(t, db.Create(&active).Error)
	verifiedAt := time.Now().UTC()
	require.NoError(t, db.Create(&domain.TenantDomain{
		TenantID: active.ID, Hostname: "isp.example.com",
		VerificationToken: "verified-token", VerifiedAt: &verifiedAt,
	}).Error)
	require.NoError(t, db.Create(&domain.TenantDomain{
		TenantID: active.ID, Hostname: "pending.example.com",
		VerificationToken: "pending-token",
	}).Error)

	for _, test := range []struct {
		name string
		host string
		want int64
	}{
		{name: "verified custom domain resolves", host: "isp.example.com", want: active.ID},
		{name: "unverified custom domain does not resolve", host: "pending.example.com", want: 0},
		{name: "unknown custom domain does not resolve", host: "unknown.example.com", want: 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/api/v1/tenant-website/public/current", nil)
			req.Host = test.host
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			handler := SubdomainResolver(db)(func(c echo.Context) error {
				id, _ := c.Get("tenant_id").(int64)
				if requestID, ok := TenantID(c.Request().Context()); ok {
					require.Equal(t, id, requestID)
				} else {
					require.Zero(t, id)
				}
				if test.want > 0 {
					require.Equal(t, test.want, id)
				} else {
					require.Zero(t, id)
				}
				return c.NoContent(http.StatusNoContent)
			})
			require.NoError(t, handler(c))
			require.Equal(t, http.StatusNoContent, rec.Code)
		})
	}
}

func TestSubdomainResolverKeepsManagedSubdomains(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.TenantDomain{}))
	tenant := domain.Tenant{ID: 8, Name: "Managed Tenant", Slug: "managed", Subdomain: "managed", Status: "active"}
	require.NoError(t, db.Create(&tenant).Error)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	req.Host = "managed.e-isp.id:8443"
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := SubdomainResolver(db)(func(c echo.Context) error {
		id, ok := c.Get("tenant_id").(int64)
		require.True(t, ok)
		require.Equal(t, tenant.ID, id)
		return c.NoContent(http.StatusNoContent)
	})
	require.NoError(t, handler(c))
	require.Equal(t, http.StatusNoContent, rec.Code)
}
