package voucherquota

import (
	"context"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type fixture struct {
	db      *gorm.DB
	ctx     context.Context
	user    domain.RadiusUser
	voucher domain.HotspotVoucher
	now     time.Time
}

func newFixture(t *testing.T, quota int64) *fixture {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(&domain.RadiusUser{}, &domain.HotspotVoucher{}, &domain.VoucherUsageSession{}))
	ctx := tenancy.WithTenantID(context.Background(), 1)
	user := domain.RadiusUser{Username: "VOUCHER-1", Status: "enabled"}
	require.NoError(t, db.WithContext(ctx).Create(&user).Error)
	voucher := domain.HotspotVoucher{
		TenantID: 1, RadiusUserID: user.ID, Code: user.Username,
		QuotaBytes: quota, Status: "used", UsedBytes: 0,
		FirstLoginAt: timePtr(time.Now().Add(-time.Hour)), ExpiresAt: timePtr(time.Now().Add(time.Hour)),
	}
	require.NoError(t, db.WithContext(ctx).Create(&voucher).Error)
	return &fixture{db: db, ctx: ctx, user: user, voucher: voucher, now: time.Now()}
}

func timePtr(v time.Time) *time.Time { return &v }

func (f *fixture) apply(t *testing.T, status int, session string, in, out int64, seconds int) Result {
	t.Helper()
	result, err := ApplyAccounting(f.ctx, f.db, f.user.Username, session, status, in, out, seconds, f.now.Add(time.Duration(seconds)*time.Second))
	require.NoError(t, err)
	return result
}

func (f *fixture) reload(t *testing.T) (domain.HotspotVoucher, domain.RadiusUser) {
	t.Helper()
	var voucher domain.HotspotVoucher
	var user domain.RadiusUser
	require.NoError(t, f.db.WithContext(f.ctx).First(&voucher, f.voucher.ID).Error)
	require.NoError(t, f.db.WithContext(f.ctx).First(&user, f.user.ID).Error)
	return voucher, user
}

func TestVoucherQuotaCountersAreIdempotentAndAccumulateDeltas(t *testing.T) {
	f := newFixture(t, 10000)
	f.apply(t, StatusStart, "session-1", 0, 0, 0)
	f.apply(t, StatusInterim, "session-1", 100, 50, 10)
	f.apply(t, StatusInterim, "session-1", 100, 50, 10) // retransmission
	f.apply(t, StatusInterim, "session-1", 200, 150, 20)
	f.apply(t, StatusStop, "session-1", 250, 200, 30)
	// Reused NAS session id after an ordinary Stop begins a new history row.
	f.apply(t, StatusStart, "session-1", 0, 0, 0)
	f.apply(t, StatusStop, "session-1", 50, 25, 10)

	voucher, _ := f.reload(t)
	require.EqualValues(t, 525, voucher.UsedBytes)
	var sessions int64
	require.NoError(t, f.db.WithContext(f.ctx).Model(&domain.VoucherUsageSession{}).Count(&sessions).Error)
	require.EqualValues(t, 2, sessions)
}

func TestVoucherQuotaHandlesStopBeforeStartWithoutDoubleCountingLateStart(t *testing.T) {
	f := newFixture(t, 10000)
	f.apply(t, StatusStop, "session-race", 150, 50, 20)
	f.apply(t, StatusStart, "session-race", 0, 0, 0)
	f.apply(t, StatusStart, "session-race", 0, 0, 0) // retransmitted late Start
	voucher, _ := f.reload(t)
	require.EqualValues(t, 200, voucher.UsedBytes)
	var sessions int64
	require.NoError(t, f.db.WithContext(f.ctx).Model(&domain.VoucherUsageSession{}).Count(&sessions).Error)
	require.EqualValues(t, 1, sessions)
}

func TestVoucherQuotaIgnoresOutOfOrderAndCountsCounterReset(t *testing.T) {
	f := newFixture(t, 10000)
	f.apply(t, StatusStart, "session-reset", 100, 100, 10)
	f.apply(t, StatusInterim, "session-reset", 150, 150, 20) // +100
	f.apply(t, StatusInterim, "session-reset", 300, 300, 15) // stale, ignored
	f.apply(t, StatusInterim, "session-reset", 10, 20, 30)   // reset, +30
	voucher, _ := f.reload(t)
	require.EqualValues(t, 330, voucher.UsedBytes)
}

func TestVoucherQuotaDisablesCredentialAndReportsExhaustion(t *testing.T) {
	f := newFixture(t, 300)
	f.apply(t, StatusStart, "session-quota", 100, 0, 0)
	result := f.apply(t, StatusInterim, "session-quota", 250, 100, 20)
	require.True(t, result.Exhausted)
	require.True(t, result.NewlyExhausted)
	voucher, user := f.reload(t)
	require.Equal(t, "exhausted", voucher.Status)
	require.EqualValues(t, 350, voucher.UsedBytes)
	require.Equal(t, "disabled", user.Status)
	// Further packets after exhaustion cannot increase the quota ledger.
	again := f.apply(t, StatusStop, "session-quota", 900, 900, 30)
	require.True(t, again.Exhausted)
	voucher, _ = f.reload(t)
	require.EqualValues(t, 350, voucher.UsedBytes)
}

func TestVoucherWithoutQuotaRemainsUnlimited(t *testing.T) {
	f := newFixture(t, 0)
	f.apply(t, StatusStart, "unlimited", 0, 0, 0)
	f.apply(t, StatusInterim, "unlimited", 500, 400, 20)
	voucher, _ := f.reload(t)
	require.EqualValues(t, 0, voucher.UsedBytes)
	require.Equal(t, "used", voucher.Status)
}
