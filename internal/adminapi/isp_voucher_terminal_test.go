package adminapi

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// voucherRevokeFixture seeds one active/used voucher with a linked, enabled
// RADIUS credential. The handler is driven through the real revokeVoucher.
type voucherRevokeFixture struct {
	t *testing.T

	db      *gorm.DB
	appCtx  app.AppContext
	user    domain.RadiusUser
	voucher domain.HotspotVoucher
}

func newVoucherRevokeFixture(t *testing.T, status string) *voucherRevokeFixture {
	t.Helper()
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}))
	appCtx := setupTestApp(t, db)
	user := domain.RadiusUser{
		ID: common.UUIDint64(), Username: "HOT-TERMINAL", Password: "HOT-TERMINAL",
		Status: common.ENABLED, ExpireTime: time.Now().Add(time.Hour),
	}
	require.NoError(t, db.Create(&user).Error)
	voucher := domain.HotspotVoucher{
		BatchID: 1, PackageID: 1, RadiusUserID: user.ID,
		Code: user.Username, Password: user.Password,
		ValiditySeconds: 3600, Status: status,
	}
	require.NoError(t, db.Create(&voucher).Error)
	return &voucherRevokeFixture{t: t, db: db, appCtx: appCtx, user: user, voucher: voucher}
}

func (f *voucherRevokeFixture) revoke() (int, voucherRevokeBody) {
	f.t.Helper()
	c, rec := newVoucherRevokeCtx(f.t, f.db, f.appCtx, f.voucher.ID)
	require.NoError(f.t, revokeVoucher(c))
	var body voucherRevokeBody
	require.NoError(f.t, json.Unmarshal(rec.Body.Bytes(), &body))
	return rec.Code, body
}

func (f *voucherRevokeFixture) stored() (domain.HotspotVoucher, domain.RadiusUser) {
	f.t.Helper()
	var voucher domain.HotspotVoucher
	require.NoError(f.t, f.db.First(&voucher, f.voucher.ID).Error)
	var user domain.RadiusUser
	require.NoError(f.t, f.db.First(&user, f.user.ID).Error)
	return voucher, user
}

// TestRevokeVoucherConcurrentRequestsConvergeOnOneTerminalState fires the real
// handler from several goroutines. The voucher must end revoked exactly once,
// every response must be a 200 (fresh revoke or idempotent replay), and the
// credential must stay disabled afterwards.
func TestRevokeVoucherConcurrentRequestsConvergeOnOneTerminalState(t *testing.T) {
	f := newVoucherRevokeFixture(t, "active")
	sqlDB, err := f.db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	const workers = 8
	type result struct {
		code int
		body voucherRevokeBody
		err  error
	}
	results := make(chan result, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	var fresh int32
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			c, rec := newVoucherRevokeCtx(t, f.db, f.appCtx, f.voucher.ID)
			err := revokeVoucher(c)
			res := result{code: rec.Code, err: err}
			if err == nil {
				_ = json.Unmarshal(rec.Body.Bytes(), &res.body)
			}
			if !res.body.Data.AlreadyRevoked {
				atomic.AddInt32(&fresh, 1)
			}
			results <- res
		}()
	}
	close(start)
	wg.Wait()
	close(results)

	for res := range results {
		require.NoError(t, res.err)
		require.Equal(t, http.StatusOK, res.code)
		assert.Equal(t, "revoked", res.body.Data.Status)
	}
	assert.EqualValues(t, 1, atomic.LoadInt32(&fresh), "exactly one request performs the revoke")

	voucher, user := f.stored()
	assert.Equal(t, "revoked", voucher.Status)
	assert.Equal(t, common.DISABLED, user.Status)

	code, body := f.revoke()
	assert.Equal(t, http.StatusOK, code)
	assert.True(t, body.Data.AlreadyRevoked, "a later retry must not perform a second revoke")
	voucher, user = f.stored()
	assert.Equal(t, "revoked", voucher.Status)
	assert.Equal(t, common.DISABLED, user.Status)
}

// TestRevokeVoucherPastValidityWindowStaysTerminal covers an expiry that has
// passed but has not been marked yet. Revocation wins and the lazy expiry
// marker (which only touches `used` rows) must never revive or re-expire it.
func TestRevokeVoucherPastValidityWindowStaysTerminal(t *testing.T) {
	f := newVoucherRevokeFixture(t, "used")
	past := time.Now().Add(-time.Minute)
	require.NoError(t, f.db.Model(&domain.HotspotVoucher{}).Where("id = ?", f.voucher.ID).
		Updates(map[string]any{"first_login_at": past, "expires_at": past}).Error)

	code, body := f.revoke()
	require.Equal(t, http.StatusOK, code)
	assert.False(t, body.Data.AlreadyRevoked)

	voucher, user := f.stored()
	assert.Equal(t, "revoked", voucher.Status)
	assert.Equal(t, common.DISABLED, user.Status)

	// Same conditional update as markExpiredHotspotVoucher.
	require.NoError(t, f.db.Model(&domain.HotspotVoucher{}).
		Where("code = ? AND status = ? AND expires_at IS NOT NULL AND expires_at <= ?",
			voucher.Code, "used", time.Now()).
		Update("status", "expired").Error)
	voucher, _ = f.stored()
	assert.Equal(t, "revoked", voucher.Status, "the expiry marker must not touch a revoked voucher")

	code, body = f.revoke()
	assert.Equal(t, http.StatusOK, code)
	assert.True(t, body.Data.AlreadyRevoked)
}

// TestRevokeVoucherFallbackDisablesOnlyMatchingCredential covers vouchers
// generated before RadiusUserID was persisted. The fallback must match both the
// voucher username and password, so a shared password or a rotated credential
// never disables an unrelated account.
func TestRevokeVoucherFallbackDisablesOnlyMatchingCredential(t *testing.T) {
	db := setupTestDB(t)
	require.NoError(t, db.AutoMigrate(&domain.HotspotBatch{}, &domain.HotspotVoucher{}))
	appCtx := setupTestApp(t, db)

	matching := domain.RadiusUser{
		ID: common.UUIDint64(), Username: "HOT-LINK", Password: "shared-pass",
		Status: common.ENABLED, ExpireTime: time.Now().Add(time.Hour),
	}
	otherUsername := domain.RadiusUser{
		ID: common.UUIDint64(), Username: "OTHER-LINK", Password: "shared-pass",
		Status: common.ENABLED, ExpireTime: time.Now().Add(time.Hour),
	}
	rotated := domain.RadiusUser{
		ID: common.UUIDint64(), Username: "HOT-ROTATED", Password: "changed-pass",
		Status: common.ENABLED, ExpireTime: time.Now().Add(time.Hour),
	}
	for _, user := range []*domain.RadiusUser{&matching, &otherUsername, &rotated} {
		require.NoError(t, db.Create(user).Error)
	}

	linked := domain.HotspotVoucher{
		BatchID: 1, PackageID: 1, RadiusUserID: 0,
		Code: matching.Username, Password: "shared-pass",
		ValiditySeconds: 3600, Status: "active",
	}
	require.NoError(t, db.Create(&linked).Error)
	changed := domain.HotspotVoucher{
		BatchID: 1, PackageID: 1, RadiusUserID: 0,
		Code: rotated.Username, Password: "shared-pass",
		ValiditySeconds: 3600, Status: "active",
	}
	require.NoError(t, db.Create(&changed).Error)

	revoke := func(id int64) int {
		c, rec := newVoucherRevokeCtx(t, db, appCtx, id)
		require.NoError(t, revokeVoucher(c))
		return rec.Code
	}

	require.Equal(t, http.StatusOK, revoke(linked.ID))
	require.NoError(t, db.First(&matching, matching.ID).Error)
	assert.Equal(t, common.DISABLED, matching.Status, "the linked credential must be disabled")
	require.NoError(t, db.First(&otherUsername, otherUsername.ID).Error)
	assert.Equal(t, common.ENABLED, otherUsername.Status, "a shared password on another username must be left alone")
	require.NoError(t, db.First(&rotated, rotated.ID).Error)
	assert.Equal(t, common.ENABLED, rotated.Status, "an unrelated account must be left alone")

	require.Equal(t, http.StatusOK, revoke(changed.ID))
	require.NoError(t, db.First(&rotated, rotated.ID).Error)
	assert.Equal(t, common.ENABLED, rotated.Status,
		"a voucher whose password no longer matches must not disable the account")
	var stored domain.HotspotVoucher
	require.NoError(t, db.First(&stored, changed.ID).Error)
	assert.Equal(t, "revoked", stored.Status, "the voucher is still terminal even when no credential matched")
}
