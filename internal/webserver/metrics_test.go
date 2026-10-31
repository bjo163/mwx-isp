package webserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/pkg/metrics"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func scrapeMetrics(t *testing.T, authorization string) *httptest.ResponseRecorder {
	t.Helper()

	e := echo.New()
	registerMetricsRoute(e)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	if authorization != "" {
		req.Header.Set(echo.HeaderAuthorization, authorization)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// TestMetricsEndpointIsDisabledWithoutToken pins the fail-closed default: the
// admin port is commonly published, so an installation that never configured a
// scrape token must not expose any counters at all.
func TestMetricsEndpointIsDisabledWithoutToken(t *testing.T) {
	t.Setenv(metricsTokenEnvVar, "")

	rec := scrapeMetrics(t, "")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestMetricsEndpointRequiresBearerToken(t *testing.T) {
	t.Setenv(metricsTokenEnvVar, "s3cret-scrape-token")

	for name, authorization := range map[string]string{
		"no credential":    "",
		"wrong token":      "Bearer not-the-token",
		"wrong scheme":     "Basic s3cret-scrape-token",
		"bare token":       "s3cret-scrape-token",
		"empty credential": "Bearer ",
	} {
		t.Run(name, func(t *testing.T) {
			rec := scrapeMetrics(t, authorization)
			require.Equal(t, http.StatusUnauthorized, rec.Code)
			assert.Equal(t, `Bearer realm="metrics"`, rec.Header().Get(echo.HeaderWWWAuthenticate))
			assert.Empty(t, rec.Body.String(), "an unauthorized scrape must not leak any metric")
		})
	}
}

func TestMetricsEndpointRendersPrometheusText(t *testing.T) {
	t.Setenv(metricsTokenEnvVar, "s3cret-scrape-token")
	require.NoError(t, metrics.InitMetrics(""))

	app.IncRadiusMetric(app.MetricsRadiusAccept)
	app.IncRadiusMetric(app.MetricsRadiusAccept)
	metrics.SetGauge("system_cpuuse", 37)

	rec := scrapeMetrics(t, "Bearer s3cret-scrape-token")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, metricsContentType, rec.Header().Get(echo.HeaderContentType))

	body := rec.Body.String()
	assert.Contains(t, body, "# TYPE radus_accept counter\nradus_accept 2\n")
	assert.Contains(t, body, "# TYPE system_cpuuse gauge\nsystem_cpuuse 37\n")
	// Every documented RADIUS counter is present, including untouched ones, so a
	// dashboard query never sees a missing series.
	for _, name := range []string{
		app.MetricsRadiusOline, app.MetricsRadiusOffline, app.MetricsRadiusAccounting,
		app.MetricsRadiusRejectPasswdError, app.MetricsRadiusRejectLdapError,
		app.MetricsRadiusAcctDropNas, app.MetricsRadiusRadsecSaturated,
	} {
		assert.Contains(t, body, "# TYPE "+name+" counter\n"+name+" ")
	}
	// The token is never echoed back into the payload.
	assert.NotContains(t, body, "s3cret-scrape-token")
}

// TestMetricsEndpointAcceptsCaseInsensitiveScheme matches the RFC 6750 grammar,
// where the scheme is case-insensitive; a Prometheus exporter that sends
// "bearer" must still scrape.
func TestMetricsEndpointAcceptsCaseInsensitiveScheme(t *testing.T) {
	t.Setenv(metricsTokenEnvVar, "s3cret-scrape-token")

	rec := scrapeMetrics(t, "bearer s3cret-scrape-token")
	assert.Equal(t, http.StatusOK, rec.Code)
}

// TestMetricsEndpointTrimsSurroundingWhitespace keeps a token pasted from a
// .env file with trailing spaces working, without weakening the comparison.
func TestMetricsEndpointTrimsSurroundingWhitespace(t *testing.T) {
	t.Setenv(metricsTokenEnvVar, "  s3cret-scrape-token  ")

	rec := scrapeMetrics(t, "Bearer s3cret-scrape-token")
	assert.Equal(t, http.StatusOK, rec.Code)
}
