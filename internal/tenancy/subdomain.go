package tenancy

import (
	"strings"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SubdomainResolver extracts the tenant from the request subdomain and injects
// tenant context into the request. For example, "emil.e-isp.id" resolves to the
// tenant with subdomain "emil".
func SubdomainResolver(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			host := c.Request().Host // e.g., "emil.e-isp.id" or "e-isp.id"

			// Extract subdomain
			subdomain := extractSubdomain(host)

			// Skip if no subdomain (e.g., e-isp.id root)
			if subdomain == "" {
				return next(c)
			}

			// Lookup tenant by subdomain
			var tenant domain.Tenant
			if err := db.Where("subdomain = ? AND status = ?", subdomain, "active").First(&tenant).Error; err != nil {
				// Fallback: try matching by slug for backward compatibility
				if err := db.Where("slug = ? AND status = ?", subdomain, "active").First(&tenant).Error; err != nil {
					// Tenant not found or not active - continue without tenant context
					// The request will be handled by login page or other public endpoints
					return next(c)
				}
			}

			// Set tenant context
			c.Set("tenant_id", tenant.ID)
			c.Set("tenant", tenant)
			c.SetRequest(c.Request().WithContext(
				WithTenantID(c.Request().Context(), tenant.ID),
			))

			return next(c)
		}
	}
}

// extractSubdomain parses the host and returns the subdomain part.
// For "emil.e-isp.id" it returns "emil".
// For "e-isp.id" or "localhost" it returns "".
func extractSubdomain(host string) string {
	// Remove port if present
	host = strings.Split(host, ":")[0]

	// Split by dots
	parts := strings.Split(host, ".")

	// Handle e-isp.id domain (or any 2-part TLD)
	if len(parts) >= 3 && strings.HasSuffix(host, "e-isp.id") {
		return parts[0] // "emil" from "emil.e-isp.id"
	}

	// Handle localhost or single-part hosts
	return ""
}
