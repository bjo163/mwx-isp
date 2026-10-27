package radiusd

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// newVoucherTerminalDB returns an isolated SQLite database whose connection pool
// is capped at one connection. SQLite upgrades read transactions to write locks
// and reports a deadlock instead of waiting, so concurrent callers are
// serialized here; the terminal-state guarantee itself comes from the
// conditional UPDATE in activateHotspotVoucher, which PostgreSQL exercises with
// real row locking in test/integration.
func newVoucherTerminalDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:radius-voucher-terminal-%d?mode=memory&cache=shared", common.UUIDint64())), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(&domain.HotspotVoucher{}, &domain.RadiusUser{}))
	return db
}

// TestHotspotVoucherTerminalStatesNeverReactivate drives concurrent first-login
// activation against expired and revoked vouchers. Every caller must receive an
// expiry in the past, and the terminal row, its first-login timestamp, and the
// linked account must stay untouched.
func TestHotspotVoucherTerminalStatesNeverReactivate(t *testing.T) {
	now := time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		status    string
		expiresAt *time.Time
	}{
		{"expired", "expired", timePtr(now.Add(-time.Minute))},
		{"revoked with expiry", "revoked", timePtr(now.Add(time.Hour))},
		{"revoked without expiry", "revoked", nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db := newVoucherTerminalDB(t)
			user := domain.RadiusUser{
				ID: 801, TenantID: 1, Username: "HOT-TERMINAL",
				Status: "enabled", ExpireTime: now.AddDate(1, 0, 0),
			}
			require.NoError(t, db.Create(&user).Error)
			voucher := domain.HotspotVoucher{
				ID: 802, TenantID: 1, Code: user.Username, Status: tc.status,
				ValiditySeconds: 3600, ExpiresAt: tc.expiresAt,
			}
			require.NoError(t, db.Create(&voucher).Error)

			const workers = 8
			type result struct {
				expiry time.Time
				found  bool
				err    error
			}
			results := make(chan result, workers)
			start := make(chan struct{})
			var wg sync.WaitGroup
			for i := 0; i < workers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					candidate := &domain.RadiusUser{ID: user.ID, TenantID: 1, Username: user.Username, Status: "enabled"}
					expiry, found, err := activateHotspotVoucher(db, 1, candidate, now)
					results <- result{expiry: expiry, found: found, err: err}
				}()
			}
			close(start)
			wg.Wait()
			close(results)

			for res := range results {
				require.NoError(t, res.err)
				require.True(t, res.found)
				assert.False(t, res.expiry.After(now),
					"a terminal voucher must never hand out a live validity window, got %v", res.expiry)
			}

			var stored domain.HotspotVoucher
			require.NoError(t, db.Where("tenant_id = ? AND id = ?", 1, voucher.ID).First(&stored).Error)
			assert.Equal(t, tc.status, stored.Status, "the terminal status must not change")
			assert.Nil(t, stored.FirstLoginAt, "a terminal voucher must not record a first login")

			var storedUser domain.RadiusUser
			require.NoError(t, db.Where("tenant_id = ? AND id = ?", 1, user.ID).First(&storedUser).Error)
			assert.True(t, storedUser.ExpireTime.Equal(user.ExpireTime),
				"activation must not extend the terminal voucher's credential")
		})
	}
}

// TestHotspotVoucherRevokedAfterFirstLoginStaysTerminal models the interleaving
// login -> revoke -> login: once the operator revoke wins, a late
// re-authentication sees the revoked row and is rejected without extending the
// window or re-enabling the account.
func TestHotspotVoucherRevokedAfterFirstLoginStaysTerminal(t *testing.T) {
	db := newVoucherTerminalDB(t)
	now := time.Date(2026, time.October, 5, 12, 0, 0, 0, time.UTC)
	user := domain.RadiusUser{
		ID: 901, TenantID: 1, Username: "HOT-REVOKED-MID", Status: "enabled",
		ExpireTime: now.AddDate(1, 0, 0),
	}
	require.NoError(t, db.Create(&user).Error)
	voucher := domain.HotspotVoucher{
		ID: 902, TenantID: 1, Code: user.Username, Status: "active", ValiditySeconds: 3600,
	}
	require.NoError(t, db.Create(&voucher).Error)

	expiry, found, err := activateHotspotVoucher(db, 1, &user, now)
	require.NoError(t, err)
	require.True(t, found)
	require.True(t, expiry.Equal(now.Add(time.Hour)))

	// The operator revoke handler performs the same conditional transition and
	// credential disable inside one transaction.
	revoke := db.Model(&domain.HotspotVoucher{}).
		Where("id = ? AND status IN ?", voucher.ID, []string{"active", "used"}).
		Update("status", "revoked")
	require.NoError(t, revoke.Error)
	require.EqualValues(t, 1, revoke.RowsAffected)
	require.NoError(t, db.Model(&domain.RadiusUser{}).Where("id = ?", user.ID).
		Update("status", "disabled").Error)

	var storedUser domain.RadiusUser
	require.NoError(t, db.Where("tenant_id = ? AND id = ?", 1, user.ID).First(&storedUser).Error)
	late := now.Add(30 * time.Minute)
	lateExpiry, found, err := activateHotspotVoucher(db, 1, &storedUser, late)
	require.NoError(t, err)
	require.True(t, found)
	assert.False(t, lateExpiry.After(late), "a post-revoke login must be rejected")

	var stored domain.HotspotVoucher
	require.NoError(t, db.Where("tenant_id = ? AND id = ?", 1, voucher.ID).First(&stored).Error)
	assert.Equal(t, "revoked", stored.Status, "a revoke must survive a late login")
	require.NotNil(t, stored.FirstLoginAt)
	assert.True(t, stored.FirstLoginAt.Equal(now), "the original first login must be preserved")
	require.NotNil(t, stored.ExpiresAt)
	assert.True(t, stored.ExpiresAt.Equal(now.Add(time.Hour)), "revocation must not extend the window")

	require.NoError(t, db.Where("tenant_id = ? AND id = ?", 1, user.ID).First(&storedUser).Error)
	assert.Equal(t, "disabled", storedUser.Status, "the credential must stay disabled")
	assert.True(t, storedUser.ExpireTime.Equal(now.Add(time.Hour)), "the credential expiry must not move")
}
