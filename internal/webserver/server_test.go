package webserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func resolvedClientIP(t *testing.T, extractor echo.IPExtractor, remoteAddr, forwardedFor string) string {
	t.Helper()
	e := echo.New()
	e.IPExtractor = extractor
	req := httptest.NewRequest(http.MethodGet, "/realip", nil)
	req.RemoteAddr = remoteAddr
	if forwardedFor != "" {
		req.Header.Set(echo.HeaderXForwardedFor, forwardedFor)
	}
	return e.NewContext(req, httptest.NewRecorder()).RealIP()
}

func TestTrustedProxyIPExtractor(t *testing.T) {
	t.Run("forwarded header ignored without trusted proxies", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor(""), "203.0.113.10:3456", "198.51.100.7")
		assert.Equal(t, "203.0.113.10", ip)
	})

	t.Run("forwarded header honored from trusted CIDR", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("203.0.113.0/24"), "203.0.113.10:3456", "198.51.100.7")
		assert.Equal(t, "198.51.100.7", ip)
	})

	t.Run("forwarded header ignored from an untrusted peer", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("203.0.113.0/24"), "192.0.2.20:3456", "198.51.100.7")
		assert.Equal(t, "192.0.2.20", ip)
	})

	t.Run("bare address is accepted as a single-host range", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("203.0.113.10"), "203.0.113.10:3456", "198.51.100.7")
		assert.Equal(t, "198.51.100.7", ip)
	})

	t.Run("invalid entries are skipped", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("not-an-ip, 203.0.113.0/24"), "203.0.113.10:3456", "198.51.100.7")
		assert.Equal(t, "198.51.100.7", ip)
	})
}

func TestParseTrustedProxyRanges(t *testing.T) {
	ranges := parseTrustedProxyRanges(" 10.0.0.1 , 2001:db8::/32 ,, bad ")
	assert.Len(t, ranges, 2)
	assert.Equal(t, "10.0.0.1/32", ranges[0].String())
	assert.Equal(t, "2001:db8::/32", ranges[1].String())
}
