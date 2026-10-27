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

// TestTrustedProxyIPExtractorRejectsImplicitPrivateTrust pins the implicit
// trust defaults of echo's X-Forwarded-For extractor. Echo would trust
// loopback, link-local, and private peers out of the box, which lets a caller
// reaching the process from a LAN address rotate the forwarding header and
// change the address the rate limiters bucket on.
func TestTrustedProxyIPExtractorRejectsImplicitPrivateTrust(t *testing.T) {
	const trusted = "203.0.113.0/24"

	t.Run("private peer cannot spoof the forwarding header", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor(trusted), "192.168.1.50:3456", "198.51.100.7")
		assert.Equal(t, "192.168.1.50", ip)
	})

	t.Run("link-local peer cannot spoof the forwarding header", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor(trusted), "169.254.10.20:3456", "198.51.100.7")
		assert.Equal(t, "169.254.10.20", ip)
	})

	t.Run("loopback proxy stays trusted without being listed", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor(trusted), "127.0.0.1:3456", "198.51.100.7")
		assert.Equal(t, "198.51.100.7", ip)
	})
}

func TestTrustedProxyIPExtractorMultiHop(t *testing.T) {
	t.Run("listed private proxies are walked through to the client", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("10.0.0.0/8"), "10.1.2.3:3456", "203.0.113.55, 10.2.3.4")
		assert.Equal(t, "203.0.113.55", ip)
	})

	t.Run("an unlisted hop is returned instead of the client", func(t *testing.T) {
		// A private hop counts as untrusted unless it is listed, so the walk
		// stops there and every client behind it shares one bucket.
		ip := resolvedClientIP(t, trustedProxyIPExtractor("10.0.0.0/8"), "10.1.2.3:3456", "203.0.113.55, 192.168.5.6")
		assert.Equal(t, "192.168.5.6", ip)
	})

	t.Run("malformed forwarding header falls back to the peer", func(t *testing.T) {
		ip := resolvedClientIP(t, trustedProxyIPExtractor("10.0.0.0/8"), "10.1.2.3:3456", "not-an-ip")
		assert.Equal(t, "10.1.2.3", ip)
	})
}

func TestParseTrustedProxyRanges(t *testing.T) {
	ranges := parseTrustedProxyRanges(" 10.0.0.1 , 2001:db8::/32 ,, bad ")
	assert.Len(t, ranges, 2)
	assert.Equal(t, "10.0.0.1/32", ranges[0].String())
	assert.Equal(t, "2001:db8::/32", ranges[1].String())
}
