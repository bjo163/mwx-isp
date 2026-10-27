//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// itNocImpact mirrors the JSON projection of the read-only impact endpoint.
type itNocImpact struct {
	Scope struct {
		Type     string `json:"type"`
		ID       string `json:"id"`
		Code     string `json:"code"`
		TargetID string `json:"target_id"`
		NasID    string `json:"nas_id"`
		NasName  string `json:"nas_name"`
	} `json:"scope"`
	State      string `json:"state"`
	Confidence string `json:"confidence"`
	Monitoring struct {
		Monitored    bool   `json:"monitored"`
		Stale        bool   `json:"stale"`
		Status       string `json:"status"`
		OpenIncident *struct {
			Summary string `json:"summary"`
		} `json:"open_incident"`
	} `json:"monitoring"`
	Counts struct {
		Customers      int64 `json:"customers"`
		Subscriptions  int64 `json:"subscriptions"`
		OnlineSessions int64 `json:"online_sessions"`
		OpenTickets    int64 `json:"open_tickets"`
	} `json:"counts"`
	Impacted []struct {
		CustomerID string     `json:"customer_id"`
		Username   string     `json:"username"`
		Online     bool       `json:"online"`
		LastSeenAt *time.Time `json:"last_seen_at"`
	} `json:"impacted"`
	Pagination struct {
		Total int64 `json:"total"`
	} `json:"pagination"`
}

func TestPostgresNetworkImpactODPAndNAS(t *testing.T) {
	client := newAPIClient(t)
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	now := time.Now().UTC()

	profileID := seedProfile(t, "it-impact-profile-"+suffix)
	odp := domain.ODP{ID: common.UUIDint64(), Code: "ODP-IT-" + suffix, Name: "Impact ODP " + suffix, TotalPorts: 8}
	require.NoError(t, db.Create(&odp).Error)
	customer := domain.Customer{
		ID: common.UUIDint64(), CustomerNo: "C-IMP-" + suffix, Name: "Impact Customer " + suffix,
		ODPID: odp.ID, ODPCode: odp.Code, ODPPort: 2, Status: "active",
	}
	require.NoError(t, db.Create(&customer).Error)
	user := domain.RadiusUser{
		ID: common.UUIDint64(), ProfileId: profileID, Username: "impact-" + suffix,
		Password: "not-a-real-password", Status: common.ENABLED, ExpireTime: now.AddDate(1, 0, 0),
	}
	require.NoError(t, db.Create(&user).Error)
	subscription := domain.Subscription{
		SubscriptionNo: "SUB-IMP-" + suffix, CustomerID: customer.ID, PackageID: 1,
		RadiusUserID: user.ID, Status: "active", StartDate: now,
	}
	require.NoError(t, db.Create(&subscription).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		TenantID: domain.DefaultTenantID, Username: user.Username, NasAddr: "198.51.100.77",
		AcctSessionId: "impact-session-" + suffix, AcctStartTime: now.Add(-time.Hour), LastUpdate: now.Add(-time.Minute),
	}).Error)

	status, body := client.get(t, "/api/v1/network/impact?odp_id="+strconv.FormatInt(odp.ID, 10))
	require.Equalf(t, http.StatusOK, status, "impact body: %s", body)
	var odpResponse struct {
		Data itNocImpact `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &odpResponse))
	assert.Equal(t, "odp", odpResponse.Data.Scope.Type)
	assert.Equal(t, "confirmed", odpResponse.Data.Confidence)
	assert.Equal(t, "not_monitored", odpResponse.Data.State)
	assert.Equal(t, int64(1), odpResponse.Data.Pagination.Total)
	assert.Equal(t, int64(1), odpResponse.Data.Counts.Customers)
	assert.Equal(t, int64(1), odpResponse.Data.Counts.OnlineSessions)
	require.Len(t, odpResponse.Data.Impacted, 1)
	assert.Equal(t, strconv.FormatInt(customer.ID, 10), odpResponse.Data.Impacted[0].CustomerID)
	assert.True(t, odpResponse.Data.Impacted[0].Online)
	require.NotNil(t, odpResponse.Data.Impacted[0].LastSeenAt)

	nas := domain.NetNas{ID: common.UUIDint64(), Name: "NAS-IMP-" + suffix, Ipaddr: "198.51.100.77", Status: "enabled"}
	require.NoError(t, db.Create(&nas).Error)
	target := domain.NetMonitorTarget{
		ID: common.UUIDint64(), Name: "impact-target-" + suffix, Kind: "nas", Address: nas.Ipaddr,
		ProbeType: "icmp", IntervalSeconds: 60, TimeoutMilliseconds: 1500, FailureThreshold: 2,
		Enabled: true, LastStatus: "down", LastCheckedAt: &now,
	}
	require.NoError(t, db.Create(&target).Error)
	require.NoError(t, db.Create(&domain.NetMonitorIncident{
		ID: common.UUIDint64(), TargetID: target.ID, State: "firing",
		Summary: "it impact incident " + suffix, StartedAt: now.Add(-5 * time.Minute),
	}).Error)

	status, body = client.get(t, "/api/v1/network/impact?nas_id="+strconv.FormatInt(nas.ID, 10))
	require.Equalf(t, http.StatusOK, status, "impact body: %s", body)
	var nasResponse struct {
		Data itNocImpact `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &nasResponse))
	assert.Equal(t, "nas", nasResponse.Data.Scope.Type)
	assert.Equal(t, strconv.FormatInt(target.ID, 10), nasResponse.Data.Scope.TargetID)
	assert.Equal(t, "confirmed_outage", nasResponse.Data.State)
	assert.Equal(t, "inferred", nasResponse.Data.Confidence)
	assert.False(t, nasResponse.Data.Monitoring.Stale)
	require.NotNil(t, nasResponse.Data.Monitoring.OpenIncident)
	assert.Equal(t, int64(1), nasResponse.Data.Pagination.Total)
	require.Len(t, nasResponse.Data.Impacted, 1)
	assert.Equal(t, fmt.Sprintf("impact-%s", suffix), nasResponse.Data.Impacted[0].Username)
	assert.True(t, nasResponse.Data.Impacted[0].Online)
}
