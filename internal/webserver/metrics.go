package webserver

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/pkg/metrics"
	"github.com/labstack/echo/v4"
)

// metricsTokenEnvVar holds the bearer token required to scrape /metrics. The
// endpoint is disabled (404) while it is unset, so a public admin port does not
// expose RADIUS counters by accident.
const metricsTokenEnvVar = "TOUGHRADIUS_METRICS_TOKEN"

// metricsContentType is the Prometheus text exposition format (version 0.0.4).
const metricsContentType = "text/plain; version=0.0.4; charset=utf-8"

func registerMetricsRoute(e *echo.Echo) {
	e.GET("/metrics", metricsHandler)
}

// metricsHandler serves the in-memory RADIUS counters and system gauges in the
// Prometheus text exposition format. Scrapers authenticate with the token from
// TOUGHRADIUS_METRICS_TOKEN as a bearer credential; without it the scrape fails
// closed even when the endpoint is enabled.
func metricsHandler(c echo.Context) error {
	token := strings.TrimSpace(os.Getenv(metricsTokenEnvVar))
	if token == "" {
		return echo.ErrNotFound
	}

	auth := c.Request().Header.Get(echo.HeaderAuthorization)
	const bearerPrefix = "bearer "
	if len(auth) < len(bearerPrefix) || !strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
		return metricsUnauthorized(c)
	}
	provided := strings.TrimSpace(auth[len(bearerPrefix):])
	if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
		return metricsUnauthorized(c)
	}

	return c.Blob(http.StatusOK, metricsContentType, []byte(prometheusMetricsText()))
}

func metricsUnauthorized(c echo.Context) error {
	c.Response().Header().Set(echo.HeaderWWWAuthenticate, `Bearer realm="metrics"`)
	return c.NoContent(http.StatusUnauthorized)
}

// prometheusMetricsText renders every known counter and gauge with a TYPE line.
// Names are emitted as-is because operators already know the historical
// radus_* names from the handbook and logs.
func prometheusMetricsText() string {
	var b strings.Builder

	counters := app.GetAllRadiusMetrics()
	if store := metrics.GetStore(); store != nil {
		for name, value := range store.GetAllCounters() {
			if _, ok := counters[name]; !ok {
				counters[name] = value
			}
		}
	}
	names := sortedKeys(counters)
	for _, name := range names {
		fmt.Fprintf(&b, "# TYPE %s counter\n%s %d\n", name, name, counters[name])
	}

	if store := metrics.GetStore(); store != nil {
		gauges := store.GetAllGauges()
		for _, name := range sortedKeys(gauges) {
			fmt.Fprintf(&b, "# TYPE %s gauge\n%s %d\n", name, name, gauges[name])
		}
	}

	return b.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for name := range m {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	return keys
}
