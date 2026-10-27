package adminapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"

	"github.com/bjo163/mwx-isp/internal/tenancy"
)

func runRateLimitedRequest(t *testing.T, mw echo.MiddlewareFunc, remoteAddr, forwardedFor string, tenantID int64) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	// Mirror the production server, which always installs an extractor; echo's
	// legacy fallback would otherwise trust X-Forwarded-For from any peer.
	e.IPExtractor = echo.ExtractIPDirect()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/public/vouchers/check", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set(echo.HeaderXForwardedFor, forwardedFor)
	}
	if tenantID > 0 {
		req = req.WithContext(tenancy.WithTenantID(req.Context(), tenantID))
	}
	rec := httptest.NewRecorder()
	handler := mw(func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})
	require.NoError(t, handler(e.NewContext(req, rec)))
	return rec
}

func TestPublicRateLimitBurstReturns429WithRetryAfter(t *testing.T) {
	mw := publicRateLimit{name: "test-burst", limit: rate.Every(time.Hour), burst: 2, expiresIn: time.Minute}.middleware()

	for i := 0; i < 2; i++ {
		rec := runRateLimitedRequest(t, mw, "203.0.113.10:1000", "", 1)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d is within the burst", i+1)
	}

	rec := runRateLimitedRequest(t, mw, "203.0.113.10:1000", "", 1)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3600", rec.Header().Get(echo.HeaderRetryAfter))
	assert.Contains(t, rec.Body.String(), "RATE_LIMITED")
}

func TestPublicRateLimitBucketsByTenantAndIP(t *testing.T) {
	mw := publicRateLimit{name: "test-tenant", limit: rate.Every(time.Hour), burst: 1, expiresIn: time.Minute}.middleware()

	require.Equal(t, http.StatusOK, runRateLimitedRequest(t, mw, "203.0.113.10:1000", "", 1).Code)
	require.Equal(t, http.StatusTooManyRequests, runRateLimitedRequest(t, mw, "203.0.113.10:1000", "", 1).Code)

	// The same address serving a different tenant keeps its own budget.
	assert.Equal(t, http.StatusOK, runRateLimitedRequest(t, mw, "203.0.113.10:1000", "", 2).Code)
	// A different address in the throttled tenant also keeps its own budget.
	assert.Equal(t, http.StatusOK, runRateLimitedRequest(t, mw, "203.0.113.11:1000", "", 1).Code)
}

func TestPublicRateLimitIgnoresSpoofedForwardedFor(t *testing.T) {
	mw := publicRateLimit{name: "test-spoof", limit: rate.Every(time.Hour), burst: 1, expiresIn: time.Minute}.middleware()

	require.Equal(t, http.StatusOK, runRateLimitedRequest(t, mw, "203.0.113.10:1000", "198.51.100.1", 1).Code)
	// The forwarding header changes but the socket peer does not, and without
	// a trusted proxy echo resolves the peer address, so the bucket is shared.
	assert.Equal(t, http.StatusTooManyRequests, runRateLimitedRequest(t, mw, "203.0.113.10:1000", "198.51.100.2", 1).Code)
}

func TestLoginRateLimiterSetsRetryAfter(t *testing.T) {
	mw := loginRateLimiter()
	for i := 0; i < 5; i++ {
		require.Equal(t, http.StatusOK, runRateLimitedRequest(t, mw, "203.0.113.20:1000", "", 0).Code)
	}
	rec := runRateLimitedRequest(t, mw, "203.0.113.20:1000", "", 0)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Equal(t, "3", rec.Header().Get(echo.HeaderRetryAfter))
}

func TestRetryAfterSeconds(t *testing.T) {
	assert.Equal(t, 3, retryAfterSeconds(rate.Every(3*time.Second)))
	assert.Equal(t, 2, retryAfterSeconds(rate.Every(2*time.Second)))
	assert.Equal(t, 1, retryAfterSeconds(rate.Every(100*time.Millisecond)))
	assert.Equal(t, 60, retryAfterSeconds(0))
}
