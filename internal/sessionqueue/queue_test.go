package sessionqueue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:sessionqueue-%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(
		&domain.RadiusOnline{}, &domain.RadiusUser{},
		&domain.RadiusSessionAction{}, &domain.RadiusSessionActionAudit{},
	))
	return db
}

// seedSession creates a disabled subscriber with one online session for the
// tenant, mirroring the state billing suspension leaves behind.
func seedSession(t *testing.T, db *gorm.DB, tenantID int64, username string) domain.RadiusOnline {
	t.Helper()
	tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
	user := domain.RadiusUser{Username: username, Password: "secret", Status: "disabled"}
	require.NoError(t, tenantDB.Create(&user).Error)
	session := domain.RadiusOnline{Username: username, AcctSessionId: "sess-" + username, NasAddr: "192.0.2.10"}
	require.NoError(t, tenantDB.Create(&session).Error)
	return session
}

func tenantDB(db *gorm.DB, tenantID int64) *gorm.DB {
	return db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
}

func loadAction(t *testing.T, db *gorm.DB, tenantID, id int64) domain.RadiusSessionAction {
	t.Helper()
	var action domain.RadiusSessionAction
	require.NoError(t, tenantDB(db, tenantID).Where("id = ?", id).First(&action).Error)
	return action
}

func successSender(calls *int) Sender {
	return func(context.Context, domain.RadiusOnline) Outcome {
		*calls++
		return Outcome{Success: true, Target: "192.0.2.10:3799", ResponseCode: "Disconnect-ACK", Attempts: 1, RTTMillis: 5}
	}
}

func failingSender(calls *int) Sender {
	return func(context.Context, domain.RadiusOnline) Outcome {
		*calls++
		return Outcome{Target: "192.0.2.10:3799", ResponseCode: "Disconnect-NAK", ErrorCause: 402, ErrorCauseText: "Session Context Not Found", Error: "NAS rejected the request", Attempts: 2, RTTMillis: 40}
	}
}

func TestBackoff(t *testing.T) {
	cases := map[int]time.Duration{
		0:  time.Minute,
		1:  time.Minute,
		2:  2 * time.Minute,
		3:  4 * time.Minute,
		8:  128 * time.Minute,
		10: 6 * time.Hour,
		30: 6 * time.Hour,
	}
	for attempt, want := range cases {
		assert.Equal(t, want, Backoff(attempt), "attempt %d", attempt)
	}
}

func TestEnqueueDisconnectIsIdempotentPerTenant(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "alice")

	created, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)
	assert.True(t, created)

	created, err = EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, created, "a repeated scheduler run must not enqueue a duplicate")

	// The same session row cannot belong to another tenant, but the idempotency
	// index itself must be tenant-scoped so tenants never collide on a key.
	foreign := session
	foreign.TenantID = 20
	created, err = EnqueueDisconnect(tenantDB(db, 20), foreign, ReasonBillingSuspension, now)
	require.NoError(t, err)
	assert.True(t, created)

	var stored []domain.RadiusSessionAction
	require.NoError(t, db.Order("tenant_id").Find(&stored).Error)
	require.Len(t, stored, 2)
	assert.Equal(t, int64(10), stored[0].TenantID)
	assert.Equal(t, int64(20), stored[1].TenantID)
	assert.Equal(t, StatusPending, stored[0].Status)
	assert.Equal(t, DefaultMaxAttempts, stored[0].MaxAttempts)
	assert.Equal(t, "alice", stored[0].Username)
	assert.Equal(t, "sess-alice", stored[0].AcctSessionID)
}

func TestEnqueueDisconnectValidatesInput(t *testing.T) {
	db := testDB(t)
	now := time.Now()
	_, err := EnqueueDisconnect(db, domain.RadiusOnline{TenantID: 10}, ReasonBillingSuspension, now)
	require.Error(t, err)
	_, err = EnqueueDisconnect(db, domain.RadiusOnline{ID: 1}, ReasonBillingSuspension, now)
	require.Error(t, err)
	_, err = EnqueueDisconnect(db, domain.RadiusOnline{ID: 1, TenantID: 10}, "  ", now)
	require.Error(t, err)
}

func TestProcessDueSendsAndWritesSystemAudit(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "alice")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	calls := 0
	summary, err := ProcessDue(context.Background(), db, 10, now, successSender(&calls), 10)
	require.NoError(t, err)
	assert.Equal(t, Summary{Attempted: 1, Succeeded: 1}, summary)
	assert.Equal(t, 1, calls)

	var action domain.RadiusSessionAction
	require.NoError(t, db.WithContext(tenancy.WithTenantID(context.Background(), 10)).First(&action).Error)
	assert.Equal(t, StatusSucceeded, action.Status)
	assert.Equal(t, 1, action.Attempts)
	assert.NotNil(t, action.SucceededAt)
	assert.True(t, action.NextRetryAt.IsZero(), "terminal actions have no retry time")
	assert.Equal(t, "Disconnect-ACK", action.LastResponseCode)
	assert.Empty(t, action.ClaimToken)

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Find(&audits).Error)
	require.Len(t, audits, 1)
	assert.Equal(t, SystemActor.Name, audits[0].OperatorName)
	assert.True(t, audits[0].Success)
	assert.Equal(t, int64(10), audits[0].TenantID)
	assert.Equal(t, "alice", audits[0].Username)

	// A second pass must not resend a finished action.
	summary, err = ProcessDue(context.Background(), db, 10, now.Add(time.Hour), successSender(&calls), 10)
	require.NoError(t, err)
	assert.Zero(t, summary.Attempted)
	assert.Equal(t, 1, calls)
}

func TestProcessDueBacksOffThenFailsTerminally(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "bob")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	calls := 0
	send := failingSender(&calls)
	summary, err := ProcessDue(context.Background(), db, 10, now, send, 10)
	require.NoError(t, err)
	assert.Equal(t, Summary{Attempted: 1, Retried: 1}, summary)

	var action domain.RadiusSessionAction
	require.NoError(t, tenantDB(db, 10).First(&action).Error)
	assert.Equal(t, StatusPending, action.Status)
	assert.Equal(t, now.Add(time.Minute), action.NextRetryAt)
	assert.Equal(t, "Disconnect-NAK", action.LastResponseCode)
	assert.Equal(t, 402, action.LastErrorCause)
	assert.Equal(t, "NAS rejected the request", action.LastError)

	// Not yet due.
	summary, err = ProcessDue(context.Background(), db, 10, now.Add(30*time.Second), send, 10)
	require.NoError(t, err)
	assert.Zero(t, summary.Attempted)

	for attempt := 2; attempt <= DefaultMaxAttempts; attempt++ {
		now = now.Add(Backoff(attempt - 1))
		summary, err = ProcessDue(context.Background(), db, 10, now, send, 10)
		require.NoError(t, err)
		require.Equal(t, 1, summary.Attempted, "attempt %d", attempt)
	}

	reloaded := loadAction(t, db, 10, action.ID)
	assert.Equal(t, StatusFailed, reloaded.Status)
	assert.Equal(t, DefaultMaxAttempts, reloaded.Attempts)
	assert.True(t, reloaded.NextRetryAt.IsZero())

	summary, err = ProcessDue(context.Background(), db, 10, now.Add(24*time.Hour), send, 10)
	require.NoError(t, err)
	assert.Zero(t, summary.Attempted, "exhausted actions are terminal until manually retried")
	assert.Equal(t, DefaultMaxAttempts, calls)

	var audits int64
	require.NoError(t, db.Model(&domain.RadiusSessionActionAudit{}).Count(&audits).Error)
	assert.Equal(t, int64(DefaultMaxAttempts), audits)
}

func TestProcessDueCancelsWhenAccessNoLongerNeedsChange(t *testing.T) {
	t.Run("session gone", func(t *testing.T) {
		db := testDB(t)
		now := time.Now().UTC()
		session := seedSession(t, db, 10, "carol")
		_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
		require.NoError(t, err)
		require.NoError(t, tenantDB(db, 10).Delete(&domain.RadiusOnline{}, session.ID).Error)

		calls := 0
		summary, err := ProcessDue(context.Background(), db, 10, now, successSender(&calls), 10)
		require.NoError(t, err)
		assert.Equal(t, Summary{Attempted: 1, Canceled: 1}, summary)
		assert.Zero(t, calls)

		var action domain.RadiusSessionAction
		require.NoError(t, tenantDB(db, 10).First(&action).Error)
		assert.Equal(t, StatusCanceled, action.Status)
		assert.Contains(t, action.LastError, "no longer online")
	})

	t.Run("subscriber re-enabled", func(t *testing.T) {
		db := testDB(t)
		now := time.Now().UTC()
		session := seedSession(t, db, 10, "dave")
		_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
		require.NoError(t, err)
		require.NoError(t, tenantDB(db, 10).Model(&domain.RadiusUser{}).Where("username = ?", "dave").Update("status", "enabled").Error)

		calls := 0
		summary, err := ProcessDue(context.Background(), db, 10, now, successSender(&calls), 10)
		require.NoError(t, err)
		assert.Equal(t, Summary{Attempted: 1, Canceled: 1}, summary)
		assert.Zero(t, calls)

		var action domain.RadiusSessionAction
		require.NoError(t, tenantDB(db, 10).First(&action).Error)
		assert.Equal(t, StatusCanceled, action.Status)
		assert.Contains(t, action.LastError, "access was restored")
	})
}

func TestProcessDueIsTenantScoped(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	sessionA := seedSession(t, db, 10, "erin")
	sessionB := seedSession(t, db, 20, "frank")
	_, err := EnqueueDisconnect(tenantDB(db, 10), sessionA, ReasonBillingSuspension, now)
	require.NoError(t, err)
	_, err = EnqueueDisconnect(tenantDB(db, 20), sessionB, ReasonBillingSuspension, now)
	require.NoError(t, err)

	calls := 0
	summary, err := ProcessDue(context.Background(), db, 10, now, successSender(&calls), 10)
	require.NoError(t, err)
	assert.Equal(t, Summary{Attempted: 1, Succeeded: 1}, summary)
	assert.Equal(t, 1, calls)

	var tenantB domain.RadiusSessionAction
	require.NoError(t, tenantDB(db, 20).First(&tenantB).Error)
	assert.Equal(t, StatusPending, tenantB.Status, "another tenant's action must not be claimed")

	require.Error(t, func() error {
		_, err := ProcessDue(context.Background(), db, 0, now, successSender(&calls), 10)
		return err
	}())
}

func TestProcessDueWithoutSenderIsNoOp(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	session := seedSession(t, db, 10, "gina")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	summary, err := ProcessDue(context.Background(), db, 10, now, nil, 10)
	require.NoError(t, err)
	assert.Zero(t, summary.Attempted)
	assert.Equal(t, StatusPending, loadAction(t, db, 10, 1).Status)
}

func TestProcessDueRecoversStaleRunningAction(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "hugo")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	// Simulate a scheduler that claimed the action and then crashed.
	action, err := claim(tenantDB(db, 10), 10, 1, []string{StatusPending}, now.Add(-staleRunningAfter-time.Minute), false)
	require.NoError(t, err)
	claimedAt := now.Add(-staleRunningAfter - time.Minute)
	require.NoError(t, tenantDB(db, 10).Model(&domain.RadiusSessionAction{}).Where("id = ?", action.ID).
		Update("updated_at", claimedAt).Error)

	calls := 0
	summary, err := ProcessDue(context.Background(), db, 10, now, successSender(&calls), 10)
	require.NoError(t, err)
	assert.Equal(t, Summary{Attempted: 1, Succeeded: 1}, summary)

	reloaded := loadAction(t, db, 10, action.ID)
	assert.Equal(t, StatusSucceeded, reloaded.Status)
	assert.Equal(t, 2, reloaded.Attempts, "the crashed claim still consumed one attempt")
}

func TestClaimFencing(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	session := seedSession(t, db, 10, "iris")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	stale, err := claim(tenantDB(db, 10), 10, 1, []string{StatusPending}, now, false)
	require.NoError(t, err)

	_, err = claim(tenantDB(db, 10), 10, 1, []string{StatusPending}, now, false)
	require.ErrorIs(t, err, errNotClaimed, "a second worker cannot claim a running action")

	current, err := claim(tenantDB(db, 10), 10, 1, []string{StatusRunning}, now, false)
	require.NoError(t, err)
	require.NotEqual(t, stale.ClaimToken, current.ClaimToken)

	err = finalize(tenantDB(db, 10), stale, now, StatusSucceeded, Outcome{Success: true}, "")
	require.ErrorIs(t, err, errNotClaimed, "a stale worker must not finalize an action reclaimed by another worker")

	require.NoError(t, finalize(tenantDB(db, 10), current, now, StatusSucceeded, Outcome{Success: true}, ""))
	assert.Equal(t, StatusSucceeded, loadAction(t, db, 10, 1).Status)
}

func TestRetryNowResetsBudgetAndAuditsOperator(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "jane")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	calls := 0
	_, err = ProcessDue(context.Background(), db, 10, now, failingSender(&calls), 10)
	require.NoError(t, err)

	operator := Actor{ID: 7, Name: "noc-1", IP: "198.51.100.7"}
	action, outcome, err := RetryNow(context.Background(), db, 10, 1, now.Add(time.Minute), successSender(&calls), operator)
	require.NoError(t, err)
	assert.True(t, outcome.Success)
	assert.Equal(t, StatusSucceeded, action.Status)
	assert.Equal(t, 1, action.Attempts, "a manual retry restores the automatic attempt budget")
	assert.Equal(t, 1, action.ManualRetries)

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Order("id").Find(&audits).Error)
	require.Len(t, audits, 2)
	last := audits[1]
	assert.Equal(t, operator.Name, last.OperatorName)
	assert.Equal(t, operator.ID, last.OperatorID)
	assert.Equal(t, operator.IP, last.OperatorIP)
	assert.True(t, last.Success)
}

func TestRetryNowRejectsUnknownRunningAndTerminalActions(t *testing.T) {
	db := testDB(t)
	now := time.Now().UTC()
	session := seedSession(t, db, 10, "kate")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)

	calls := 0
	send := successSender(&calls)
	operator := Actor{ID: 1, Name: "noc"}

	_, _, err = RetryNow(context.Background(), db, 20, 1, now, send, operator)
	require.ErrorIs(t, err, ErrNotFound, "actions outside the tenant are invisible")

	_, err = claim(tenantDB(db, 10), 10, 1, []string{StatusPending}, now, false)
	require.NoError(t, err)
	_, _, err = RetryNow(context.Background(), db, 10, 1, now, send, operator)
	require.ErrorIs(t, err, ErrRunning)

	running := loadAction(t, db, 10, 1)
	require.NoError(t, finalize(tenantDB(db, 10), &running, now, StatusSucceeded, Outcome{Success: true}, ""))
	_, _, err = RetryNow(context.Background(), db, 10, 1, now, send, operator)
	require.ErrorIs(t, err, ErrNotRetryable)

	_, _, err = RetryNow(context.Background(), db, 10, 1, now, nil, operator)
	require.ErrorIs(t, err, errNoSender)
}

func TestRetryNowRecoversFailedAction(t *testing.T) {
	db := testDB(t)
	now := time.Date(2026, time.October, 27, 12, 0, 0, 0, time.UTC)
	session := seedSession(t, db, 10, "liam")
	_, err := EnqueueDisconnect(tenantDB(db, 10), session, ReasonBillingSuspension, now)
	require.NoError(t, err)
	require.NoError(t, tenantDB(db, 10).Model(&domain.RadiusSessionAction{}).Where("id = ?", 1).
		Updates(map[string]any{"status": StatusFailed, "attempts": DefaultMaxAttempts}).Error)

	calls := 0
	action, outcome, err := RetryNow(context.Background(), db, 10, 1, now, successSender(&calls), Actor{ID: 2, Name: "noc-2"})
	require.NoError(t, err)
	assert.True(t, outcome.Success)
	assert.Equal(t, StatusSucceeded, action.Status)
	assert.Equal(t, 1, action.Attempts)
	assert.Equal(t, 1, action.ManualRetries)
	assert.Equal(t, 1, calls)
}
