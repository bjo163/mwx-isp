package adminapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/require"
)

func TestIPAMPoolValidation(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.IPAMPool{}, &domain.RadiusOnline{}))

	create := func(payload string) (int, []byte) {
		t.Helper()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/network/ipam/pools", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		require.NoError(t, createIPAMPool(c))
		return rec.Code, rec.Body.Bytes()
	}
	created := func(t *testing.T, body []byte) domain.IPAMPool {
		t.Helper()
		var pool domain.IPAMPool
		decodeMatrixData(t, body, &pool)
		return pool
	}

	status, body := create(`{"name":"Host bits","cidr":"100.64.0.5/22"}`)
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "INVALID_CIDR")

	status, body = create(`{"name":"CGNAT","cidr":"100.64.0.0/22","gateway":"100.64.0.1","pool_type":"cgnat"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	cgnat := created(t, body)
	require.Equal(t, int64(1022), cgnat.TotalIPs, "IPv4 /22 reserves network and broadcast")
	require.Equal(t, "1022 addresses", cgnat.CapacityLabel)
	require.Equal(t, 4, cgnat.IPVersion)

	// RFC 3021 point-to-point links have no reserved network/broadcast address.
	status, body = create(`{"name":"P2P","cidr":"10.0.0.0/31","gateway":"10.0.0.0"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	require.Equal(t, int64(2), created(t, body).TotalIPs)

	status, body = create(`{"name":"Single host","cidr":"10.0.1.7/32"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	require.Equal(t, int64(1), created(t, body).TotalIPs)

	status, body = create(`{"name":"IPv6 small","cidr":"2001:db8:1::/126","gateway":"2001:db8:1::1"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	require.Equal(t, int64(4), created(t, body).TotalIPs)

	// A /48 exceeds int64; report a label instead of a fabricated number.
	status, body = create(`{"name":"IPv6 ISP","cidr":"2001:db8::/48"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	ispIPv6 := created(t, body)
	require.Zero(t, ispIPv6.TotalIPs)
	require.Equal(t, "2^80 addresses", ispIPv6.CapacityLabel)
	require.Equal(t, 6, ispIPv6.IPVersion)

	status, body = create(`{"name":"Overlap","cidr":"100.64.2.0/24"}`)
	require.Equalf(t, http.StatusConflict, status, "%s", body)
	require.Contains(t, string(body), "CIDR_OVERLAP")

	status, body = create(`{"name":"Outside","cidr":"100.68.0.0/24","gateway":"100.69.0.1"}`)
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "INVALID_GATEWAY")

	status, body = create(`{"name":"Wrong family","cidr":"2001:db8:2::/64","gateway":"10.0.0.1"}`)
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "INVALID_GATEWAY")

	status, body = create(`{"name":"Broadcast gateway","cidr":"10.1.0.0/24","gateway":"10.1.0.255"}`)
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "INVALID_GATEWAY")

	status, body = create(`{"name":"Wrong version","cidr":"10.2.0.0/24","ip_version":6}`)
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "IP_VERSION_MISMATCH")

	// Renumbering a pool must not strand an active lease outside the new CIDR.
	status, body = create(`{"name":"Customer pool","cidr":"100.67.0.0/24","gateway":"100.67.0.1"}`)
	require.Equalf(t, http.StatusOK, status, "%s", body)
	pool := created(t, body)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: 1, Username: "leased", AcctSessionId: "ipam-lease", FramedIpaddr: "100.67.0.55",
	}).Error)

	req := httptest.NewRequest(http.MethodPut,
		fmt.Sprintf("/api/v1/network/ipam/pools/%d", pool.ID),
		strings.NewReader(`{"name":"Customer pool","cidr":"100.69.0.0/24","gateway":"100.69.0.1"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	c.SetParamNames("id")
	c.SetParamValues(fmt.Sprintf("%d", pool.ID))
	require.NoError(t, updateIPAMPool(c))
	require.Equalf(t, http.StatusConflict, rec.Code, "%s", rec.Body.String())
	require.Contains(t, rec.Body.String(), "POOL_HAS_ACTIVE_LEASES")
}
