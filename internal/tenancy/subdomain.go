package tenancy

import (
	"net"
	"strings"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SubdomainResolver extracts a tenant from the managed e-isp.id subdomain or
// from a custom hostname whose DNS ownership challenge has been verified.
func SubdomainResolver(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			host := requestHostname(c.Request().Host)
			if host == "" {
				return next(c)
			}

			var tenant domain.Tenant
			if subdomain := extractSubdomain(host); subdomain != "" {
				if err := db.Where("(subdomain = ? OR slug = ?) AND status = ?", subdomain, subdomain, "active").First(&tenant).Error; err != nil {
					return next(c)
				}
			} else {
				var mapping domain.TenantDomain
				if err := db.Where("hostname = ? AND verified_at IS NOT NULL", host).First(&mapping).Error; err != nil {
					return next(c)
				}
				if err := db.Where("id = ? AND status = ?", mapping.TenantID, "active").First(&tenant).Error; err != nil {
					return next(c)
				}
			}

			c.Set("tenant_id", tenant.ID)
			c.Set("tenant", tenant)
			c.SetRequest(c.Request().WithContext(WithTenantID(c.Request().Context(), tenant.ID)))
			return next(c)
		}
	}
}

func requestHostname(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	host = strings.TrimSuffix(host, ".")
	return strings.Trim(host, "[]")
}

// extractSubdomain parses the tenant part of a managed e-isp.id hostname.
// For "emil.e-isp.id" it returns "emil"; for the platform root it returns "".
func extractSubdomain(host string) string {
	host = requestHostname(host)
	parts := strings.Split(host, ".")
	if len(parts) == 3 && parts[1] == "e-isp" && parts[2] == "id" {
		if parts[0] != "" && parts[0] != "www" {
			return parts[0]
		}
	}
	return ""
}
