package adminapi

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"github.com/bjo163/mwx-isp/internal/tenancy"
)

// publicRegistrationBodyLimit caps the registration payload. The handler reads
// a small JSON object, so a larger body is rejected before it is parsed.
const publicRegistrationBodyLimit = "64K"

// publicRateLimit is one abuse-control bucket for a public-facing route. The
// bucket key combines the route name, the scoped tenant, and the resolved
// client IP, so one noisy tenant or address cannot exhaust another party's
// budget. The client IP comes from echo's IPExtractor, which honors
// X-Forwarded-For only for proxies listed in TOUGHRADIUS_TRUSTED_PROXIES (see
// internal/webserver).
type publicRateLimit struct {
	name      string
	limit     rate.Limit
	burst     int
	expiresIn time.Duration
}

func (p publicRateLimit) middleware() echo.MiddlewareFunc {
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{
		Rate:      p.limit,
		Burst:     p.burst,
		ExpiresIn: p.expiresIn,
	})
	return middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: store,
		IdentifierExtractor: func(c echo.Context) (string, error) {
			tenantID := int64(0)
			if scopedID, ok := tenancy.TenantID(c.Request().Context()); ok {
				tenantID = scopedID
			}
			return fmt.Sprintf("%s|tenant:%d|ip:%s", p.name, tenantID, c.RealIP()), nil
		},
		ErrorHandler: func(c echo.Context, err error) error {
			return fail(c, http.StatusInternalServerError, "RATE_LIMIT_ERROR", "Unable to evaluate rate limit", nil)
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			setRetryAfter(c, p.limit)
			return fail(c, http.StatusTooManyRequests, "RATE_LIMITED",
				"Too many requests, please try again later", nil)
		},
	})
}

// setRetryAfter reports how many seconds a throttled caller should wait before
// at least one token is available again.
func setRetryAfter(c echo.Context, limit rate.Limit) {
	c.Response().Header().Set(echo.HeaderRetryAfter, strconv.Itoa(retryAfterSeconds(limit)))
}

func retryAfterSeconds(limit rate.Limit) int {
	if limit <= 0 {
		return 60
	}
	seconds := int(math.Ceil(1 / float64(limit)))
	if seconds < 1 {
		return 1
	}
	return seconds
}

func portalLookupRateLimiter() echo.MiddlewareFunc {
	return publicRateLimit{
		name:      "portal_lookup",
		limit:     rate.Every(2 * time.Second),
		burst:     10,
		expiresIn: 3 * time.Minute,
	}.middleware()
}

func portalPaymentChannelRateLimiter() echo.MiddlewareFunc {
	return publicRateLimit{
		name:      "portal_payment_channel",
		limit:     rate.Every(2 * time.Second),
		burst:     10,
		expiresIn: 3 * time.Minute,
	}.middleware()
}

func voucherCheckRateLimiter() echo.MiddlewareFunc {
	return publicRateLimit{
		name:      "voucher_check",
		limit:     rate.Every(2 * time.Second),
		burst:     10,
		expiresIn: 3 * time.Minute,
	}.middleware()
}

func publicRegistrationRateLimiter() echo.MiddlewareFunc {
	return publicRateLimit{
		name:      "public_register",
		limit:     rate.Every(20 * time.Second),
		burst:     3,
		expiresIn: 10 * time.Minute,
	}.middleware()
}

// diagnosticsRateLimiter bounds the administrator reachability probe. The
// endpoint is admin-only and its targets are already restricted, but a probe
// interval below a couple of seconds turns it into a sweep tool.
func diagnosticsRateLimiter() echo.MiddlewareFunc {
	return publicRateLimit{
		name:      "network_diagnostics_ping",
		limit:     rate.Every(2 * time.Second),
		burst:     5,
		expiresIn: 3 * time.Minute,
	}.middleware()
}
