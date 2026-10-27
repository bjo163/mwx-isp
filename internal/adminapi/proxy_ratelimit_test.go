package adminapi

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loginBurst mirrors the burst configured in loginRateLimiter.
const loginBurst = 5

func loginAttempt(t *testing.T, handler http.Handler, remoteAddr, forwardedFor string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login",
		strings.NewReader(`{"username":"ghost","password":"not-the-password"}`))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set(echo.HeaderXForwardedFor, forwardedFor)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

// TestLoginRateLimitIsNotEvadedBySpoofedForwardingHeaders drives the real
// server with a private peer that is not a declared proxy. Echo trusts
// private peers implicitly, so without an explicit opt-out the caller could
// rotate X-Forwarded-For and get a fresh bucket per attempt.
func TestLoginRateLimitIsNotEvadedBySpoofedForwardingHeaders(t *testing.T) {
	t.Setenv("TOUGHRADIUS_TRUSTED_PROXIES", "35.200.0.1")
	handler, _, _ := newRoleMatrixServer(t)

	const peer = "192.168.1.50:5555"
	for i := 0; i < loginBurst; i++ {
		rec := loginAttempt(t, handler, peer, "198.51.100."+strconv.Itoa(i+1))
		require.NotEqualf(t, http.StatusTooManyRequests, rec.Code,
			"attempt %d is within the burst", i+1)
	}

	rec := loginAttempt(t, handler, peer, "198.51.100.200")
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.Contains(t, rec.Body.String(), "RATE_LIMITED")
}

// TestLoginRateLimitBucketsForwardedClientBehindTrustedProxy covers the
// normal deployment: the peer is a listed proxy, so the forwarded client is
// the bucket key and each client keeps its own budget.
func TestLoginRateLimitBucketsForwardedClientBehindTrustedProxy(t *testing.T) {
	t.Setenv("TOUGHRADIUS_TRUSTED_PROXIES", "35.200.0.1")
	handler, _, _ := newRoleMatrixServer(t)

	const proxyPeer = "35.200.0.1:443"
	for i := 0; i < loginBurst; i++ {
		require.NotEqual(t, http.StatusTooManyRequests,
			loginAttempt(t, handler, proxyPeer, "198.51.100.10").Code)
	}
	assert.Equal(t, http.StatusTooManyRequests,
		loginAttempt(t, handler, proxyPeer, "198.51.100.10").Code)

	// A noisy client behind the same proxy must not exhaust anyone else.
	assert.NotEqual(t, http.StatusTooManyRequests,
		loginAttempt(t, handler, proxyPeer, "198.51.100.11").Code)
}
