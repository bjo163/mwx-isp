package adminapi

import (
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
)

func requireTargetCode(t *testing.T, err error, code string) {
	t.Helper()
	var targetErr *diagnosticTargetError
	require.ErrorAs(t, err, &targetErr)
	require.Equal(t, code, targetErr.Code, "refusal message: %s", targetErr.Message)
}

func TestParseProbeTargetAcceptsExplicitAddresses(t *testing.T) {
	cases := []struct {
		raw     string
		address string
		port    string
	}{
		{"192.168.88.1", "192.168.88.1", "80"},
		{"192.168.88.1:8080", "192.168.88.1", "8080"},
		{"8.8.8.8:53", "8.8.8.8", "53"},
		{"2001:db8::1", "2001:db8::1", "80"},
		{"[2001:db8::1]:443", "2001:db8::1", "443"},
	}
	for _, tc := range cases {
		target, err := parseProbeTarget(tc.raw)
		require.NoErrorf(t, err, "%s", tc.raw)
		require.Equal(t, tc.address, target.Address, "%s", tc.raw)
		require.Equal(t, tc.port, target.Port, "%s", tc.raw)
	}
}

func TestParseProbeTargetRejectsMalformedAndUnsafeInputs(t *testing.T) {
	cases := []struct {
		raw  string
		code string
	}{
		{"   ", "HOST_REQUIRED"},
		// A name would have to be resolved, and could resolve elsewhere after
		// validation; the probe endpoint never resolves.
		{"router.office.local", "TARGET_NOT_IP"},
		{"router.office.local:80", "TARGET_NOT_IP"},
		{"192.168.1.1:0", "INVALID_PORT"},
		{"192.168.1.1:65536", "INVALID_PORT"},
		{"192.168.1.1:http", "INVALID_PORT"},
		{"0.0.0.0", "TARGET_NOT_ALLOWED"},
		{"127.0.0.1", "TARGET_NOT_ALLOWED"},
		{"[::1]:1816", "TARGET_NOT_ALLOWED"},
		{"169.254.169.254", "TARGET_NOT_ALLOWED"},
		{"[fe80::1]", "TARGET_NOT_ALLOWED"},
		{"224.0.0.1", "TARGET_NOT_ALLOWED"},
	}
	for _, tc := range cases {
		_, err := parseProbeTarget(tc.raw)
		requireTargetCode(t, err, tc.code)
	}
}

func TestAuthorizeProbeTargetRequiresRegistrationForPrivateRanges(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.IPAMPool{}, &domain.NetMonitorTarget{}))

	require.NoError(t, authorizeProbeTarget(db, net.ParseIP("8.8.8.8")),
		"public unicast targets are the normal connectivity test")

	requireTargetCode(t, authorizeProbeTarget(db, net.ParseIP("192.168.10.5")), "TARGET_NOT_REGISTERED")

	require.NoError(t, db.Create(&domain.IPAMPool{
		TenantID: domain.DefaultTenantID, Name: "Office", CIDR: "192.168.10.0/24", IPVersion: 4,
	}).Error)
	require.NoError(t, authorizeProbeTarget(db, net.ParseIP("192.168.10.5")))
	requireTargetCode(t, authorizeProbeTarget(db, net.ParseIP("192.168.11.5")), "TARGET_NOT_REGISTERED")

	// CGNAT is shared address space, not private: the same registration rule applies.
	require.NoError(t, db.Create(&domain.IPAMPool{
		TenantID: domain.DefaultTenantID, Name: "CGNAT", CIDR: "100.64.0.0/22", IPVersion: 4,
	}).Error)
	require.NoError(t, authorizeProbeTarget(db, net.ParseIP("100.64.1.20")))
	requireTargetCode(t, authorizeProbeTarget(db, net.ParseIP("100.68.1.20")), "TARGET_NOT_REGISTERED")

	require.NoError(t, db.Create(&domain.NetNas{
		TenantID: domain.DefaultTenantID, Name: "Core NAS", Identifier: "core-nas", Ipaddr: "192.168.20.1",
	}).Error)
	require.NoError(t, authorizeProbeTarget(db, net.ParseIP("192.168.20.1")))

	require.NoError(t, db.Create(&domain.NetMonitorTarget{
		TenantID: domain.DefaultTenantID, Name: "Edge router", Kind: "router", Address: "192.168.30.1",
		ProbeType: "icmp", IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2,
	}).Error)
	require.NoError(t, authorizeProbeTarget(db, net.ParseIP("192.168.30.1")))
}

func TestMonitorTargetRejectsLinkLocalAddress(t *testing.T) {
	input := monitorTargetInput{
		Name: "Edge", Address: "169.254.169.254", ProbeType: "icmp",
		IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2,
	}
	require.ErrorContains(t, validateMonitorInput(&input, false), "link-local")

	input.Address = "fe80::1"
	require.ErrorContains(t, validateMonitorInput(&input, false), "link-local")

	// Loopback is refused for monitor targets too: it aims the periodic probe at
	// the daemon's own listeners, and a local service is reachable through the
	// host's LAN address, which is part of the registered inventory.
	input.Address = "127.0.0.1"
	require.ErrorContains(t, validateMonitorInput(&input, false), "loopback")
}

func TestLivePingRejectsUnsafeTargets(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.AutoMigrate(&domain.IPAMPool{}, &domain.NetMonitorTarget{}))
	token := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9401, LevelAdmin, "diag-admin"))

	cases := []struct {
		host   string
		status int
		code   string
	}{
		{"router.office.local", http.StatusBadRequest, "TARGET_NOT_IP"},
		{"169.254.169.254", http.StatusBadRequest, "TARGET_NOT_ALLOWED"},
		{"[::1]:1816", http.StatusBadRequest, "TARGET_NOT_ALLOWED"},
		{"192.168.50.10", http.StatusForbidden, "TARGET_NOT_REGISTERED"},
	}
	for _, tc := range cases {
		status, body := matrixRequest(t, handler, http.MethodPost,
			"/api/v1/network/diagnostics/ping", token, map[string]any{"host": tc.host, "count": 1})
		require.Equalf(t, tc.status, status, "%s: %s", tc.host, body)
		require.Containsf(t, string(body), tc.code, "%s: %s", tc.host, body)
	}
}

func TestLivePingIsRateLimited(t *testing.T) {
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.AutoMigrate(&domain.IPAMPool{}, &domain.NetMonitorTarget{}))
	token := matrixToken(t, appCtx, seedMatrixOperator(t, db, 9402, LevelAdmin, "diag-limited"))

	for i := 0; i < 5; i++ {
		status, _ := matrixRequest(t, handler, http.MethodPost,
			"/api/v1/network/diagnostics/ping", token, map[string]any{"host": "router.office.local"})
		require.Equal(t, http.StatusBadRequest, status, "attempt %d is within the burst", i+1)
	}
	status, body := matrixRequest(t, handler, http.MethodPost,
		"/api/v1/network/diagnostics/ping", token, map[string]any{"host": "router.office.local"})
	require.Equal(t, http.StatusTooManyRequests, status)
	require.Contains(t, string(body), "RATE_LIMITED")
}
