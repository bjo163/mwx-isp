package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"layeh.com/radius"
	"layeh.com/radius/rfc3576"
)

func newSessionActionQueueCtx(t *testing.T, db *gorm.DB, appCtx app.AppContext, method, target, idParam string, tenantID int64) (echo.Context, *httptest.ResponseRecorder) {
	t.Helper()
	e := setupTestEcho()
	req := httptest.NewRequest(method, target, nil)
	if tenantID > 0 {
		req = req.WithContext(tenancy.WithTenantID(req.Context(), tenantID))
	}
	rec := httptest.NewRecorder()
	c := CreateTestContext(e, db, req, rec, appCtx)
	if idParam != "" {
		c.SetParamNames("id")
		c.SetParamValues(idParam)
	}
	return c, rec
}

func decodeRetryResponse(t *testing.T, rec *httptest.ResponseRecorder) (domain.RadiusSessionAction, sessionActionOutcome) {
	t.Helper()
	var resp struct {
		Data struct {
			Action  domain.RadiusSessionAction `json:"action"`
			Outcome sessionActionOutcome       `json:"outcome"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	return resp.Data.Action, resp.Data.Outcome
}

func queueAction(t *testing.T, db *gorm.DB, sessionID int64) domain.RadiusSessionAction {
	t.Helper()
	var session domain.RadiusOnline
	require.NoError(t, db.First(&session, sessionID).Error)
	_, err := sessionqueue.EnqueueDisconnect(db, session, sessionqueue.ReasonBillingSuspension, time.Now())
	require.NoError(t, err)
	var action domain.RadiusSessionAction
	require.NoError(t, db.Where("session_id = ?", sessionID).First(&action).Error)
	return action
}

func TestListSessionActionsIsTenantScopedAndFiltered(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)

	now := time.Now().UTC()
	for _, row := range []domain.RadiusSessionAction{
		{TenantID: 1, IdempotencyKey: "k1", SessionID: 1, Username: "alice", NasAddr: "192.0.2.1", Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension, Status: sessionqueue.StatusPending, MaxAttempts: 8, NextRetryAt: now},
		{TenantID: 1, IdempotencyKey: "k2", SessionID: 2, Username: "bob", NasAddr: "192.0.2.1", Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension, Status: sessionqueue.StatusFailed, MaxAttempts: 8},
		{TenantID: 2, IdempotencyKey: "k3", SessionID: 3, Username: "carol", NasAddr: "192.0.2.2", Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension, Status: sessionqueue.StatusPending, MaxAttempts: 8, NextRetryAt: now},
	} {
		require.NoError(t, db.Create(&row).Error)
	}

	c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodGet, "/api/v1/session-actions?status=pending", "", 1)
	require.NoError(t, ListSessionActions(c))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []domain.RadiusSessionAction `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1, "other tenants' actions must never be listed")
	assert.Equal(t, "alice", resp.Data[0].Username)
	assert.Equal(t, int64(1), resp.Meta.Total)

	c, rec = newSessionActionQueueCtx(t, db, appCtx, http.MethodGet, "/api/v1/session-actions?status=unknown", "", 1)
	require.NoError(t, ListSessionActions(c))
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "INVALID_STATUS", decodeErr(t, rec).Error)
}

func TestRetrySessionActionSendsDisconnectAndAuditsOperator(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	nas := newFakeCoANAS(t, coaTestSecret, radius.CodeDisconnectACK, 0)
	sessionID := seedCoASession(t, db, nas.port(t))
	action := queueAction(t, db, sessionID)

	savedResolver := testOperatorResolver
	testOperatorResolver = func(echo.Context) (*domain.SysOpr, bool) {
		return &domain.SysOpr{ID: 9, Username: "noc-1", Level: "admin", Status: "enabled", TenantID: 1}, true
	}
	t.Cleanup(func() { testOperatorResolver = savedResolver })

	c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodPost, "/api/v1/session-actions/1/retry", "1", 1)
	require.NoError(t, RetrySessionAction(c))
	require.Equal(t, http.StatusOK, rec.Code)

	updated, outcome := decodeRetryResponse(t, rec)
	assert.Equal(t, sessionqueue.StatusSucceeded, updated.Status)
	assert.True(t, outcome.Success)
	assert.Equal(t, "Disconnect-ACK", outcome.ResponseCode)
	assert.Equal(t, 1, outcome.Attempts)

	received := nas.snapshot()
	require.Len(t, received, 1)
	assert.Equal(t, radius.CodeDisconnectRequest, received[0].code)
	assert.Equal(t, action.AcctSessionID, received[0].acctSessionID)

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Find(&audits).Error)
	require.Len(t, audits, 1)
	assert.Equal(t, "noc-1", audits[0].OperatorName)
	assert.Equal(t, int64(1), audits[0].TenantID)
	assert.True(t, audits[0].Success)
}

func TestRetrySessionActionReportsNAKWithoutTerminalFailure(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	nas := newFakeCoANAS(t, coaTestSecret, radius.CodeDisconnectNAK, rfc3576.ErrorCause_Value_SessionContextNotFound)
	sessionID := seedCoASession(t, db, nas.port(t))
	queueAction(t, db, sessionID)

	c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodPost, "/api/v1/session-actions/1/retry", "1", 1)
	require.NoError(t, RetrySessionAction(c))
	require.Equal(t, http.StatusOK, rec.Code)

	updated, outcome := decodeRetryResponse(t, rec)
	assert.Equal(t, sessionqueue.StatusPending, updated.Status, "a NAK remains retryable until the budget is exhausted")
	assert.False(t, outcome.Success)
	assert.Equal(t, "Disconnect-NAK", outcome.ResponseCode)
	assert.Equal(t, int(rfc3576.ErrorCause_Value_SessionContextNotFound), outcome.ErrorCause)
	assert.False(t, updated.NextRetryAt.IsZero())

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Find(&audits).Error)
	require.Len(t, audits, 1)
	assert.False(t, audits[0].Success)
	assert.Equal(t, int(rfc3576.ErrorCause_Value_SessionContextNotFound), audits[0].ErrorCause)
}

func TestRetrySessionActionRecoversFailedAction(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	nas := newFakeCoANAS(t, coaTestSecret, radius.CodeDisconnectACK, 0)
	sessionID := seedCoASession(t, db, nas.port(t))
	action := queueAction(t, db, sessionID)
	require.NoError(t, db.Model(&domain.RadiusSessionAction{}).Where("id = ?", action.ID).
		Updates(map[string]any{"status": sessionqueue.StatusFailed, "attempts": sessionqueue.DefaultMaxAttempts}).Error)

	c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodPost, "/api/v1/session-actions/1/retry", "1", 1)
	require.NoError(t, RetrySessionAction(c))
	require.Equal(t, http.StatusOK, rec.Code)

	updated, outcome := decodeRetryResponse(t, rec)
	assert.True(t, outcome.Success)
	assert.Equal(t, sessionqueue.StatusSucceeded, updated.Status)
	assert.Equal(t, 1, updated.Attempts)
	assert.Equal(t, 1, updated.ManualRetries)
}

func TestRetrySessionActionRejectsForeignRunningAndTerminalActions(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	now := time.Now().UTC()
	foreign := domain.RadiusSessionAction{
		TenantID: 2, IdempotencyKey: "foreign", SessionID: 1, Username: "alice",
		Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension,
		Status: sessionqueue.StatusPending, MaxAttempts: 8, NextRetryAt: now,
	}
	require.NoError(t, db.Create(&foreign).Error)
	succeeded := domain.RadiusSessionAction{
		TenantID: 1, IdempotencyKey: "done", SessionID: 2, Username: "bob",
		Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension,
		Status: sessionqueue.StatusSucceeded, MaxAttempts: 8,
	}
	require.NoError(t, db.Create(&succeeded).Error)
	running := domain.RadiusSessionAction{
		TenantID: 1, IdempotencyKey: "running", SessionID: 3, Username: "carol",
		Action: sessionqueue.ActionDisconnect, Reason: sessionqueue.ReasonBillingSuspension,
		Status: sessionqueue.StatusRunning, MaxAttempts: 8,
	}
	require.NoError(t, db.Create(&running).Error)

	cases := []struct {
		id     string
		status int
		code   string
	}{
		{id: "1", status: http.StatusNotFound, code: "NOT_FOUND"},
		{id: "2", status: http.StatusConflict, code: "NOT_RETRYABLE"},
		{id: "3", status: http.StatusConflict, code: "ACTION_RUNNING"},
		{id: "abc", status: http.StatusBadRequest, code: "INVALID_ID"},
	}
	for _, tc := range cases {
		c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodPost, "/api/v1/session-actions/"+tc.id+"/retry", tc.id, 1)
		require.NoError(t, RetrySessionAction(c))
		assert.Equal(t, tc.status, rec.Code, tc.id)
		assert.Equal(t, tc.code, decodeErr(t, rec).Error, tc.id)
	}
}

func TestRetrySessionActionCancelsWhenSubscriberWasRestored(t *testing.T) {
	db := setupTestDB(t)
	appCtx := setupTestApp(t, db)
	nas := newFakeCoANAS(t, coaTestSecret, radius.CodeDisconnectACK, 0)
	sessionID := seedCoASession(t, db, nas.port(t))
	queueAction(t, db, sessionID)
	require.NoError(t, db.Create(&domain.RadiusUser{Username: "alice", Password: "secret", Status: "enabled"}).Error)

	c, rec := newSessionActionQueueCtx(t, db, appCtx, http.MethodPost, "/api/v1/session-actions/1/retry", "1", 1)
	require.NoError(t, RetrySessionAction(c))
	require.Equal(t, http.StatusOK, rec.Code)

	updated, outcome := decodeRetryResponse(t, rec)
	assert.False(t, outcome.Success)
	assert.Equal(t, sessionqueue.StatusCanceled, updated.Status)
	assert.Contains(t, updated.LastError, "access was restored")
	assert.Empty(t, nas.snapshot(), "a restored subscriber must not be disconnected")
}
