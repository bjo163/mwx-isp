//go:build integration

package integration

import (
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPostgresFUPIsolatesTenantsWithDuplicateUsernames proves the FUP scheduler
// attributes traffic, audit events and throttled rates to the owning tenant even
// when two organizations use the same RADIUS username. It also verifies that an
// open accounting row (mirrored by the live session) is not counted twice.
func TestPostgresFUPIsolatesTenantsWithDuplicateUsernames(t *testing.T) {
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	username := "fup-shared-" + suffix

	type fixture struct {
		tenantID int64
		sub      domain.Subscription
		user     domain.RadiusUser
	}
	fixtures := make(map[string]fixture, 2)

	for _, scenario := range []struct {
		key     string
		traffic int64
	}{
		{key: "over", traffic: 2 << 30},
		{key: "under", traffic: 100 << 20},
	} {
		tenant := domain.Tenant{
			Name: "FUP tenant " + scenario.key, Slug: "it-fup-" + scenario.key + "-" + suffix,
			Kind: "isp", Status: "active",
		}
		require.NoError(t, db.Create(&tenant).Error)
		tenantDB := db.WithContext(tenancy.WithTenantID(t.Context(), tenant.ID))

		profile := domain.RadiusProfile{
			ID: common.UUIDint64(), Name: "FUP profile " + scenario.key + " " + suffix,
			Status: common.ENABLED, UpRate: 2048, DownRate: 4096,
		}
		require.NoError(t, tenantDB.Create(&profile).Error)
		pkg := domain.InternetPackage{
			Code: "FUP-" + scenario.key + "-" + suffix, Name: "FUP package " + scenario.key,
			Price: 100000, RadiusProfileID: profile.ID, BillingCycle: "monthly", Status: "active",
			FupLimitGB: 1, FupRateUp: 256, FupRateDown: 512,
		}
		require.NoError(t, tenantDB.Create(&pkg).Error)
		user := domain.RadiusUser{
			ID: common.UUIDint64(), ProfileId: profile.ID, Username: username,
			Password: "secret", Status: common.ENABLED, UpRate: 2048, DownRate: 4096,
		}
		require.NoError(t, tenantDB.Create(&user).Error)
		customer := domain.Customer{
			ID: common.UUIDint64(), CustomerNo: "FUP-" + scenario.key + "-" + suffix,
			Name: "FUP customer " + scenario.key, Status: domain.CustomerActive,
		}
		require.NoError(t, tenantDB.Create(&customer).Error)
		sub := domain.Subscription{
			ID: common.UUIDint64(), SubscriptionNo: "FUP-" + scenario.key + "-" + suffix,
			CustomerID: customer.ID, PackageID: pkg.ID, RadiusUserID: user.ID,
			Status: domain.SubscriptionActive, StartDate: time.Now().AddDate(0, -1, 0), BillingDay: 1,
		}
		require.NoError(t, tenantDB.Create(&sub).Error)

		now := time.Now()
		require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{
			ID: common.UUIDint64(), Username: username, AcctSessionId: "closed-" + scenario.key + "-" + suffix,
			AcctInputTotal: scenario.traffic / 2, AcctOutputTotal: scenario.traffic / 2,
			AcctStartTime: now.Add(-2 * time.Hour), AcctStopTime: now.Add(-time.Hour),
		}).Error)
		if scenario.key == "under" {
			// Open accounting mirrors a live session and must not be double counted.
			require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{
				ID: common.UUIDint64(), Username: username, AcctSessionId: "open-" + suffix,
				AcctInputTotal: 10 << 30, AcctOutputTotal: 10 << 30,
				AcctStartTime: now.Add(-time.Hour),
			}).Error)
		}
		fixtures[scenario.key] = fixture{tenantID: tenant.ID, sub: sub, user: user}
	}

	h.appCtx.SchedFUPCheckTask()

	for key, wantTriggered := range map[string]bool{"over": true, "under": false} {
		f := fixtures[key]
		tenantDB := db.WithContext(tenancy.WithTenantID(t.Context(), f.tenantID))

		var sub domain.Subscription
		require.NoError(t, tenantDB.First(&sub, f.sub.ID).Error)
		require.Equal(t, wantTriggered, sub.FupTriggered, "%s tenant FUP state", key)

		var user domain.RadiusUser
		require.NoError(t, tenantDB.First(&user, f.user.ID).Error)
		if wantTriggered {
			require.Equal(t, 256, user.UpRate)
			require.Equal(t, 512, user.DownRate)
		} else {
			require.Equal(t, 2048, user.UpRate)
			require.Equal(t, 4096, user.DownRate)
		}

		var events []domain.BillingEvent
		require.NoError(t, tenantDB.Where("subscription_id = ?", f.sub.ID).Find(&events).Error)
		if wantTriggered {
			require.Len(t, events, 1)
			assert.Equal(t, "fup_throttled", events[0].Type)
			assert.Equal(t, f.tenantID, events[0].TenantID)
		} else {
			assert.Empty(t, events, "the under-quota tenant must not emit FUP events")
		}
	}
}
