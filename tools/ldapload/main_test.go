package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPercentileUsesNearestRank(t *testing.T) {
	samples := sortedCopy([]time.Duration{9 * time.Millisecond, 1 * time.Millisecond, 5 * time.Millisecond, 3 * time.Millisecond})
	assert.Equal(t, time.Duration(0), percentile(nil, 0.5), "an empty sample set has no quantile")
	assert.Equal(t, 1*time.Millisecond, percentile(samples, 0))
	assert.Equal(t, 3*time.Millisecond, percentile(samples, 0.50))
	assert.Equal(t, 9*time.Millisecond, percentile(samples, 0.95))
	assert.Equal(t, 9*time.Millisecond, percentile(samples, 1))
	assert.Equal(t, 9*time.Millisecond, percentile(samples, 2), "a quantile above 1 clamps to the worst sample")
}

func TestSummarizeCountsErrorsAndKeepsSamples(t *testing.T) {
	samples := []time.Duration{2 * time.Millisecond, 4 * time.Millisecond, 6 * time.Millisecond}
	stat := summarize("verify", samples, 3)
	assert.Equal(t, 3, stat.errors)
	assert.Equal(t, 4*time.Millisecond, stat.p50, "the median of three samples is the middle one")
	assert.Equal(t, 6*time.Millisecond, stat.p99)
	assert.Equal(t, 4*time.Millisecond, stat.mean)
	assert.Equal(t, 2*time.Millisecond, samples[0], "summarize must not reorder the caller's samples")

	empty := summarize("verify", nil, 5)
	assert.Equal(t, time.Duration(0), empty.mean)
	assert.Equal(t, 5, empty.errors)
}

func TestFormatHelpers(t *testing.T) {
	assert.Equal(t, "750us", duration(750*time.Microsecond))
	assert.Equal(t, "1.25ms", duration(1250*time.Microsecond))
	assert.Equal(t, "2.00s", duration(2*time.Second))
	assert.Equal(t, "33%", relative(time.Millisecond, 3*time.Millisecond))
	assert.Equal(t, "n/a", relative(time.Millisecond, 0), "a zero-cost verify has no meaningful share")
}

func TestConcurrencyListSet(t *testing.T) {
	var list concurrencyList
	require.NoError(t, list.Set("1, 10,50"))
	assert.Equal(t, concurrencyList{1, 10, 50}, list)
	assert.Equal(t, "1,10,50", list.String())

	assert.Error(t, list.Set("10,abc"), "a non-numeric level is rejected")
	assert.Error(t, list.Set("0"), "a zero level is rejected")
	assert.Error(t, list.Set(""), "an empty list is rejected")
}

func TestParseFlagsValidatesModes(t *testing.T) {
	opts, err := parseFlags([]string{"-bind-mode", "search", "-requests", "5"})
	require.NoError(t, err)
	assert.Equal(t, "search", opts.bindMode)
	assert.Equal(t, 5, opts.requests)
	assert.Equal(t, concurrencyList{1, 10, 50}, opts.concurrencies, "the default concurrency ladder is kept")

	_, err = parseFlags([]string{"-bind-mode", "anonymous"})
	assert.ErrorContains(t, err, "-bind-mode must be template or search")

	_, err = parseFlags([]string{"-requests", "0"})
	assert.Error(t, err, "a zero-request run would produce no evidence")
}

func TestConfigMirrorsFlags(t *testing.T) {
	opts, err := parseFlags([]string{
		"-url", "ldaps://dc.example.com:636", "-bind-mode", "search",
		"-base-dn", "dc=example,dc=com", "-user-filter", "(sAMAccountName=%s)",
		"-search-bind-dn", "cn=svc,dc=example,dc=com", "-search-bind-password", "svc-password",
		"-starttls", "-tls-skip-verify", "-timeout", "7s",
	})
	require.NoError(t, err)

	cfg := opts.config()
	require.NoError(t, cfg.Validate())
	assert.Equal(t, "ldaps://dc.example.com:636", cfg.ServerURL)
	assert.True(t, cfg.StartTLS)
	assert.True(t, cfg.TLSSkipVerify)
	assert.Equal(t, 7*time.Second, cfg.Timeout)
	assert.Equal(t, "(sAMAccountName=%s)", cfg.UserFilter)
}
