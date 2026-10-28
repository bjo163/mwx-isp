package networkmonitor

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestCheckTargetAddress(t *testing.T) {
	tests := []struct {
		name    string
		address string
		allowed bool
	}{
		{"public v4", "8.8.8.8", true},
		{"public v6", "2001:4860:4860::8888", true},
		{"rfc1918 class a", "10.20.30.40", true},
		{"rfc1918 class b", "172.16.5.4", true},
		{"rfc1918 class c", "192.168.1.1", true},
		{"unique local v6", "fd00::10", true},
		{"padded address", " 10.0.0.1 ", true},
		{"hostname", "core-router", false},
		{"hostname with domain", "core-router.lan.example", false},
		{"url", "http://10.0.0.1", false},
		{"address with port", "10.0.0.1:161", false},
		{"empty", "", false},
		{"unspecified v4", "0.0.0.0", false},
		{"this network", "0.10.0.1", false},
		{"loopback v4", "127.0.0.1", false},
		{"loopback v6", "::1", false},
		{"ipv4-mapped loopback", "::ffff:127.0.0.1", false},
		{"link-local v4", "169.254.169.254", false},
		{"link-local v6", "fe80::1", false},
		{"link-local v6 with zone", "fe80::1%eth0", false},
		{"multicast v4", "224.0.0.1", false},
		{"multicast v6", "ff02::1", false},
		{"limited broadcast", "255.255.255.255", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckTargetAddress(tc.address)
			if tc.allowed {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
		})
	}
}

// TestMonitorCheckTargetRefusesStoredLoopback proves the egress re-check: a row
// written before the policy existed must not be dialed, and refusing it must not
// fabricate an outage sample.
func TestMonitorCheckTargetRefusesStoredLoopback(t *testing.T) {
	db := monitorTestDB(t)
	target := domain.NetMonitorTarget{
		Name: "legacy-local", Kind: "server", Address: "127.0.0.1", ProbeType: "tcp", Port: 3000,
		IntervalSeconds: 60, TimeoutMilliseconds: 1000, FailureThreshold: 1, Enabled: true, LastStatus: "up",
	}
	require.NoError(t, db.Create(&target).Error)
	probe := &fakeProbe{results: []ProbeResult{{Reachable: true}}}
	monitor := New(db, "application-secret", probe)

	err := monitor.CheckTarget(context.Background(), target.ID, time.Now())
	require.Error(t, err)
	require.Contains(t, err.Error(), "not allowed")
	require.Zero(t, probe.Calls(), "a refused address must never reach the probe")

	var samples int64
	require.NoError(t, db.Model(&domain.NetMonitorSample{}).Count(&samples).Error)
	require.Zero(t, samples)

	var stored domain.NetMonitorTarget
	require.NoError(t, db.First(&stored, target.ID).Error)
	require.Equal(t, "up", stored.LastStatus, "a policy refusal is not an outage")
}

// TestHTTPProbeDoesNotFollowRedirects pins the redirect policy: the Location
// header points at an address the operator never validated, so the probe must
// stop at the first response instead of dialing it.
func TestHTTPProbeDoesNotFollowRedirects(t *testing.T) {
	client := newProbeHTTPClient(time.Second)
	req, err := http.NewRequest(http.MethodGet, "http://10.0.0.1/", nil)
	require.NoError(t, err)

	err = client.CheckRedirect(req, nil)
	require.True(t, errors.Is(err, http.ErrUseLastResponse))
}
