package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupNOCImpactDB(t *testing.T) (*gorm.DB, app.AppContext) {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{},
		&domain.Subscription{},
		&domain.ODP{},
		&domain.NetMonitorTarget{},
		&domain.NetMonitorIncident{},
		&domain.TroubleTicket{},
	))
	return db, setupTestApp(t, db)
}

func newNOCImpactCtx(t *testing.T, db *gorm.DB, appCtx app.AppContext, path string, tenantID int64) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if tenantID > 0 {
		req = req.WithContext(tenancy.WithTenantID(req.Context(), tenantID))
	}
	rec := httptest.NewRecorder()
	c := CreateTestContext(setupTestEcho(), db, req, rec, appCtx)
	if tenantID > 0 {
		c.Set("tenant_id", tenantID)
	}
	return c, rec
}

func decodeNOCImpact(t *testing.T, rec *httptest.ResponseRecorder) nocImpactResponse {
	t.Helper()
	var resp struct {
		Data nocImpactResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Data
}

func seedNOCImpactSubscriber(t *testing.T, db *gorm.DB, tenantID, customerID, subscriptionID, userID int64, customerName, username, status string, odpID int64, odpCode string, odpPort int) {
	t.Helper()
	customer := domain.Customer{
		ID: customerID, TenantID: tenantID, CustomerNo: fmt.Sprintf("C-%d", customerID),
		Name: customerName, ODPID: odpID, ODPCode: odpCode, ODPPort: odpPort, Status: "active",
	}
	require.NoError(t, db.Where("id = ?", customerID).FirstOrCreate(&customer).Error)
	user := domain.RadiusUser{ID: userID, TenantID: tenantID, Username: username, Status: "enabled"}
	require.NoError(t, db.Where("id = ?", userID).FirstOrCreate(&user).Error)
	require.NoError(t, db.Create(&domain.Subscription{
		ID: subscriptionID, TenantID: tenantID, SubscriptionNo: fmt.Sprintf("S-%d", subscriptionID),
		CustomerID: customerID, PackageID: 1, RadiusUserID: userID, Status: status,
		StartDate: time.Now().UTC().Add(-30 * 24 * time.Hour),
	}).Error)
}

func seedNOCImpactSession(t *testing.T, db *gorm.DB, tenantID int64, username, nasAddr, sessionID string, lastUpdate time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: tenantID, Username: username, NasAddr: nasAddr,
		AcctSessionId: sessionID, AcctStartTime: lastUpdate.Add(-time.Hour), LastUpdate: lastUpdate,
	}).Error)
}

func seedNOCImpactAccounting(t *testing.T, db *gorm.DB, tenantID int64, username, nasAddr, sessionID string, lastUpdate time.Time) {
	t.Helper()
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		TenantID: tenantID, Username: username, NasAddr: nasAddr,
		AcctSessionId: sessionID, AcctStartTime: lastUpdate.Add(-2 * time.Hour), LastUpdate: lastUpdate,
	}).Error)
}

func TestNetworkImpactODPAggregatesDeduplicatesAndPaginates(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	now := time.Now().UTC()

	require.NoError(t, db.Create(&domain.ODP{ID: 10, TenantID: 1, Code: "ODP-KNG-001", Name: "Kuningan 1", Zone: "Kuningan", TotalPorts: 16}).Error)
	seedNOCImpactSubscriber(t, db, 1, 100, 300, 200, "Alice", "alice", "active", 10, "ODP-KNG-001", 1)
	seedNOCImpactSubscriber(t, db, 1, 101, 301, 201, "Bob", "bob", "suspended", 10, "ODP-KNG-001", 2)
	seedNOCImpactSubscriber(t, db, 1, 102, 302, 202, "Carol", "carol", "active", 10, "ODP-KNG-001", 3)
	seedNOCImpactSubscriber(t, db, 1, 100, 303, 203, "Alice", "alice2", "pending", 10, "ODP-KNG-001", 1)

	// Two simultaneous sessions for the same username must not duplicate rows.
	seedNOCImpactSession(t, db, 1, "alice", "192.0.2.10", "alice-1", now.Add(-2*time.Minute))
	seedNOCImpactSession(t, db, 1, "alice", "192.0.2.10", "alice-2", now.Add(-1*time.Minute))
	seedNOCImpactSession(t, db, 1, "bob", "192.0.2.99", "bob-1", now.Add(-3*time.Minute))
	seedNOCImpactAccounting(t, db, 1, "carol", "192.0.2.10", "carol-old", now.Add(-2*time.Hour))

	require.NoError(t, db.Create(&domain.TroubleTicket{ID: 400, TenantID: 1, TicketNo: "T-1", CustomerID: 101, Subject: "no internet", Status: "open"}).Error)
	require.NoError(t, db.Create(&domain.TroubleTicket{ID: 401, TenantID: 1, TicketNo: "T-2", CustomerID: 102, Subject: "slow", Status: "resolved"}).Error)

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=10&page=1&pageSize=2", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, "odp", data.Scope.Type)
	assert.Equal(t, "ODP-KNG-001", data.Scope.Code)
	assert.Equal(t, nocImpactStateNotMonitored, data.State)
	assert.Equal(t, nocImpactConfidenceConfirmed, data.Confidence)
	assert.Equal(t, int64(3), data.Counts.Customers)
	assert.Equal(t, int64(4), data.Counts.Subscriptions)
	assert.Equal(t, int64(3), data.Counts.OnlineSessions)
	assert.Equal(t, int64(1), data.Counts.OpenTickets)
	assert.Equal(t, int64(4), data.Pagination.Total)
	assert.Len(t, data.Impacted, 2)

	first := data.Impacted[0]
	assert.Equal(t, "100", first.CustomerID)
	assert.Equal(t, "300", first.SubscriptionID)
	assert.Equal(t, "alice", first.Username)
	assert.True(t, first.Online)
	require.NotNil(t, first.LastSeenAt)

	second := data.Impacted[1]
	assert.Equal(t, "303", second.SubscriptionID)
	assert.False(t, second.Online)

	// A page with no live sessions must never assert that subscribers are offline.
	assert.NotEqual(t, nocImpactStateOutage, data.State)
}

func TestNetworkImpactODPFallsBackToCachedCode(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	require.NoError(t, db.Create(&domain.ODP{ID: 11, TenantID: 1, Code: "ODP-BSD-002", Name: "BSD 2"}).Error)
	seedNOCImpactSubscriber(t, db, 1, 110, 310, 210, "Dewi", "dewi", "active", 0, "ODP-BSD-002", 4)

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=11", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, nocImpactConfidenceInferred, data.Confidence)
	assert.Equal(t, int64(1), data.Pagination.Total)
	assert.Equal(t, "310", data.Impacted[0].SubscriptionID)
	require.NotEmpty(t, data.Warnings)
}

func TestNetworkImpactODPWithoutMappingIsNotConfirmedClear(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	require.NoError(t, db.Create(&domain.ODP{ID: 12, TenantID: 1, Code: "ODP-EMPTY-003", Name: "Empty"}).Error)

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=12", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, nocImpactConfidenceUnknown, data.Confidence)
	assert.Zero(t, data.Pagination.Total)
	assert.Empty(t, data.Impacted)
	require.NotEmpty(t, data.Warnings)
}

func seedNOCImpactNASFixture(t *testing.T, db *gorm.DB, lastChecked time.Time, lastStatus string) (targetID, nasID int64) {
	t.Helper()
	require.NoError(t, db.Create(&domain.NetNas{ID: 20, TenantID: 1, Name: "NAS-KTR", Ipaddr: "192.0.2.10", Status: "enabled"}).Error)
	require.NoError(t, db.Create(&domain.NetMonitorTarget{
		ID: 40, TenantID: 1, Name: "NAS KTR", Kind: "nas", Address: "192.0.2.10",
		ProbeType: "icmp", IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2,
		Enabled: true, LastStatus: lastStatus, LastCheckedAt: &lastChecked,
	}).Error)
	require.NoError(t, db.Create(&domain.NetMonitorIncident{
		ID: 50, TenantID: 1, TargetID: 40, State: "firing", Summary: "NAS KTR unreachable",
		StartedAt: lastChecked.Add(-10 * time.Minute),
	}).Error)

	seedNOCImpactSubscriber(t, db, 1, 120, 320, 220, "Dewi", "dewi", "active", 0, "", 0)
	seedNOCImpactSubscriber(t, db, 1, 121, 321, 221, "Eko", "eko", "active", 0, "", 0)
	// Not part of any observed activity on the NAS: must not be listed.
	seedNOCImpactSubscriber(t, db, 1, 122, 322, 222, "Farah", "farah", "active", 0, "", 0)

	seedNOCImpactSession(t, db, 1, "dewi", "192.0.2.10", "dewi-live", lastChecked.Add(-time.Minute))
	seedNOCImpactAccounting(t, db, 1, "eko", "192.0.2.10", "eko-recent", lastChecked.Add(-time.Hour))
	// Outside the lookback window.
	seedNOCImpactAccounting(t, db, 1, "farah", "192.0.2.10", "farah-stale", lastChecked.Add(-30*time.Hour))
	return 40, 20
}

func TestNetworkImpactTargetConfirmedOutageUsesNASMapping(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	now := time.Now().UTC()
	seedNOCImpactNASFixture(t, db, now.Add(-30*time.Second), "down")

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?target_id=40", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, "target", data.Scope.Type)
	assert.Equal(t, "20", data.Scope.NasID)
	assert.Equal(t, nocImpactStateOutage, data.State)
	assert.Equal(t, nocImpactConfidenceInferred, data.Confidence)
	assert.False(t, data.Monitoring.Stale)
	require.NotNil(t, data.Monitoring.OpenIncident)
	assert.Equal(t, "NAS KTR unreachable", data.Monitoring.OpenIncident.Summary)
	assert.Equal(t, int64(2), data.Pagination.Total)
	assert.Equal(t, int64(2), data.Counts.Customers)
	assert.Equal(t, int64(1), data.Counts.OnlineSessions)

	online := map[string]bool{}
	for _, row := range data.Impacted {
		online[row.Username] = row.Online
	}
	assert.True(t, online["dewi"])
	// A subscriber without a live session is not marked offline, only not online.
	assert.False(t, online["eko"])
}

func TestNetworkImpactStaleDownIsNotAConfirmedOutage(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	now := time.Now().UTC()
	seedNOCImpactNASFixture(t, db, now.Add(-2*time.Hour), "down")

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?target_id=40", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, nocImpactStateLastDownStale, data.State)
	assert.True(t, data.Monitoring.Stale)
	assert.Equal(t, nocImpactConfidenceInferred, data.Confidence)
	assert.Equal(t, int64(2), data.Pagination.Total)
	require.NotEmpty(t, data.Warnings)
}

func TestNetworkImpactTargetWithoutNASMapping(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	now := time.Now().UTC()
	require.NoError(t, db.Create(&domain.NetMonitorTarget{
		ID: 41, TenantID: 1, Name: "Edge Router", Kind: "router", Address: "198.51.100.9",
		ProbeType: "icmp", IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2,
		Enabled: true, LastStatus: "up", LastCheckedAt: &now,
	}).Error)

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?target_id=41", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, nocImpactStateOperational, data.State)
	assert.Equal(t, nocImpactConfidenceUnknown, data.Confidence)
	assert.Zero(t, data.Pagination.Total)
	assert.Empty(t, data.Impacted)
	require.NotEmpty(t, data.Warnings)
}

func TestNetworkImpactNASSelectorUsesMonitoringAndSessions(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	now := time.Now().UTC()
	seedNOCImpactNASFixture(t, db, now.Add(-20*time.Second), "up")

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?nas_id=20", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, "nas", data.Scope.Type)
	assert.Equal(t, "40", data.Scope.TargetID)
	assert.Equal(t, nocImpactStateOperational, data.State)
	assert.Equal(t, int64(2), data.Pagination.Total)
}

func TestNetworkImpactIsTenantScoped(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Name: "Other ISP", Slug: "other", Kind: "isp", Status: "active"}).Error)
	require.NoError(t, db.Create(&domain.ODP{ID: 60, TenantID: 2, Code: "ODP-OTHER-1", Name: "Other"}).Error)
	seedNOCImpactSubscriber(t, db, 2, 160, 360, 260, "Other Customer", "other-user", "active", 60, "ODP-OTHER-1", 1)
	require.NoError(t, db.Create(&domain.ODP{ID: 61, TenantID: 1, Code: "ODP-MINE-1", Name: "Mine"}).Error)
	seedNOCImpactSubscriber(t, db, 1, 161, 361, 261, "My Customer", "my-user", "active", 61, "ODP-MINE-1", 1)
	// Cross-tenant data corruption must not leak into tenant 1's impact list.
	seedNOCImpactSubscriber(t, db, 2, 162, 362, 262, "Leaked", "leaked-user", "active", 61, "ODP-MINE-1", 2)
	require.NoError(t, db.Create(&domain.NetMonitorTarget{
		ID: 42, TenantID: 2, Name: "Other target", Kind: "router", Address: "203.0.113.7",
		ProbeType: "icmp", IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2, Enabled: true,
	}).Error)

	c1, rec1 := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=60", 1)
	require.NoError(t, getNetworkImpact(c1))
	assert.Equal(t, http.StatusNotFound, rec1.Code)

	c2, rec2 := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=60", 2)
	require.NoError(t, getNetworkImpact(c2))
	require.Equal(t, http.StatusOK, rec2.Code)
	assert.Equal(t, int64(1), decodeNOCImpact(t, rec2).Pagination.Total)

	c3, rec3 := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=61", 1)
	require.NoError(t, getNetworkImpact(c3))
	require.Equal(t, http.StatusOK, rec3.Code)
	data := decodeNOCImpact(t, rec3)
	assert.Equal(t, int64(1), data.Pagination.Total)
	assert.Equal(t, "161", data.Impacted[0].CustomerID)

	c4, rec4 := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?target_id=42", 1)
	require.NoError(t, getNetworkImpact(c4))
	assert.Equal(t, http.StatusNotFound, rec4.Code)
}

func TestNetworkImpactValidatesSelector(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)

	for _, path := range []string{
		"/api/v1/network/impact",
		"/api/v1/network/impact?target_id=1&odp_id=2",
		"/api/v1/network/impact?nas_id=abc",
		"/api/v1/network/impact?odp_id=0",
	} {
		c, rec := newNOCImpactCtx(t, db, appCtx, path, 1)
		require.NoError(t, getNetworkImpact(c), path)
		assert.Equal(t, http.StatusBadRequest, rec.Code, path)
	}

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=999", 1)
	require.NoError(t, getNetworkImpact(c))
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNetworkImpactLargeSubscriberSetPaginates(t *testing.T) {
	db, appCtx := setupNOCImpactDB(t)
	require.NoError(t, db.Create(&domain.ODP{ID: 70, TenantID: 1, Code: "ODP-BIG-1", Name: "Big"}).Error)
	for i := 0; i < 120; i++ {
		seedNOCImpactSubscriber(t, db, 1, int64(700+i), int64(1700+i), int64(2700+i),
			fmt.Sprintf("Customer %03d", i), fmt.Sprintf("user%03d", i), "active", 70, "ODP-BIG-1", (i%16)+1)
	}

	c, rec := newNOCImpactCtx(t, db, appCtx, "/api/v1/network/impact?odp_id=70&pageSize=100", 1)
	require.NoError(t, getNetworkImpact(c))
	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeNOCImpact(t, rec)

	assert.Equal(t, int64(120), data.Pagination.Total)
	assert.Equal(t, int64(120), data.Counts.Customers)
	assert.Len(t, data.Impacted, 100)
}
