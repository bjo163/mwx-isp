//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc3576"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// itSessionAction mirrors the JSON projection of domain.RadiusSessionAction.
// The integration package duplicates the shape because it exercises the API
// strictly over HTTP, as a real client would.
type itSessionAction struct {
	ID               int64     `json:"id,string"`
	Username         string    `json:"username"`
	AcctSessionID    string    `json:"acct_session_id"`
	NasAddr          string    `json:"nas_addr"`
	Action           string    `json:"action"`
	Reason           string    `json:"reason"`
	Status           string    `json:"status"`
	Attempts         int       `json:"attempts"`
	MaxAttempts      int       `json:"max_attempts"`
	ManualRetries    int       `json:"manual_retries"`
	NextRetryAt      time.Time `json:"next_retry_at"`
	LastResponseCode string    `json:"last_response_code"`
	LastErrorCause   int       `json:"last_error_cause"`
	LastError        string    `json:"last_error"`
	LastTimedOut     bool      `json:"last_timed_out"`
}

// itRetryResponse is the payload of POST /session-actions/:id/retry.
type itRetryResponse struct {
	Action  itSessionAction `json:"action"`
	Outcome struct {
		Success        bool   `json:"success"`
		ResponseCode   string `json:"response_code"`
		ErrorCause     int    `json:"error_cause"`
		ErrorCauseText string `json:"error_cause_text"`
		Attempts       int    `json:"attempts"`
		RTTMillis      int64  `json:"rtt_ms"`
		TimedOut       bool   `json:"timed_out"`
		Error          string `json:"error"`
	} `json:"outcome"`
}

// itSeedSubscriber creates the RADIUS account a queued Disconnect targets. A
// billing suspension disables the account before enqueueing the action, so the
// default fixture is disabled; tests that model a re-enabled subscriber flip the
// status afterwards.
func itSeedSubscriber(t *testing.T, username string) domain.RadiusUser {
	t.Helper()
	profileID := seedProfile(t, "it-action-profile-"+uniqueSuffix())
	user := domain.RadiusUser{
		ID:         common.UUIDint64(),
		ProfileId:  profileID,
		Username:   username,
		Password:   "not-a-real-password",
		Status:     common.DISABLED,
		ExpireTime: time.Now().AddDate(1, 0, 0),
	}
	require.NoError(t, h.appCtx.DB().Create(&user).Error)
	t.Cleanup(func() { _ = h.appCtx.DB().Delete(&domain.RadiusUser{}, user.ID).Error })
	return user
}

// itEnqueueDisconnect reloads the session so its persisted tenant ID is present,
// then records the billing suspension action exactly as the scheduler does.
func itEnqueueDisconnect(t *testing.T, sessionID int64) domain.RadiusOnline {
	t.Helper()
	var session domain.RadiusOnline
	require.NoError(t, h.appCtx.DB().First(&session, sessionID).Error)
	inserted, err := sessionqueue.EnqueueDisconnect(h.appCtx.DB(), session, sessionqueue.ReasonBillingSuspension, time.Now())
	require.NoError(t, err)
	require.True(t, inserted, "first enqueue for a session must insert an action")
	t.Cleanup(func() {
		_ = h.appCtx.DB().Where("acct_session_id = ?", session.AcctSessionId).Delete(&domain.RadiusSessionAction{}).Error
		_ = h.appCtx.DB().Where("acct_session_id = ?", session.AcctSessionId).Delete(&domain.RadiusSessionActionAudit{}).Error
	})
	return session
}

// itLoadAction reads the single queue row for a session without tenant scoping,
// mirroring how an operator views the platform-wide queue from the database.
func itLoadAction(t *testing.T, acctSessionID string) domain.RadiusSessionAction {
	t.Helper()
	var actions []domain.RadiusSessionAction
	require.NoError(t, h.appCtx.DB().Where("acct_session_id = ?", acctSessionID).Find(&actions).Error)
	require.Len(t, actions, 1, "tenants and repeated enqueues must never duplicate a billing action")
	return actions[0]
}

func itActionAudits(t *testing.T, acctSessionID string) []domain.RadiusSessionActionAudit {
	t.Helper()
	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, h.appCtx.DB().Where("acct_session_id = ?", acctSessionID).Order("id ASC").Find(&audits).Error)
	return audits
}

// itSessionActionSender mirrors the production scheduler wiring in main.go: it
// resolves the NAS by the session's NAS address and performs one synchronous
// RFC 5176 Disconnect-Request. A short timeout keeps the timeout test fast.
func itSessionActionSender() sessionqueue.Sender {
	svc := radiusd.NewCoAService(nil,
		radiusd.WithCoATimeout(300*time.Millisecond),
		radiusd.WithCoARetries(1))
	return func(ctx context.Context, session domain.RadiusOnline) sessionqueue.Outcome {
		var nas domain.NetNas
		if err := h.appCtx.DB().WithContext(ctx).Where("ipaddr = ?", session.NasAddr).First(&nas).Error; err != nil {
			return sessionqueue.Outcome{Target: session.NasAddr,
				Error: fmt.Sprintf("resolve NAS for session %d: %v", session.ID, err)}
		}
		result, err := svc.Disconnect(ctx, radiusd.CoATargetFromNas(&nas), radiusd.SessionIdentityFromOnline(&session))
		if err != nil {
			return sessionqueue.Outcome{Target: nas.Ipaddr, Error: err.Error()}
		}
		return sessionqueue.Outcome{
			Success:        result.Success,
			Target:         result.Target,
			ResponseCode:   result.ResponseCode,
			ErrorCause:     result.ErrorCause,
			ErrorCauseText: result.ErrorCauseText,
			Error:          result.Err,
			TimedOut:       result.TimedOut,
			Attempts:       result.Attempts,
			RTTMillis:      result.RTT.Milliseconds(),
		}
	}
}

func itDecodeRetry(t *testing.T, body []byte) itRetryResponse {
	t.Helper()
	var resp itRetryResponse
	unwrapData(t, body, &resp)
	return resp
}

// TestPostgresSessionActionQueueAckRetryAndAudit drives the full billing
// suspension convergence path on PostgreSQL: the action is enqueued once (a
// repeated scheduler run inserts no duplicate), an operator retries it over
// HTTP, the NAS receives a real signed RFC 5176 Disconnect-Request, and the
// attempt is audited against the operator and the tenant.
func TestPostgresSessionActionQueueAckRetryAndAudit(t *testing.T) {
	const secret = "it-action-secret-ack"
	nas := newITFakeNAS(t, secret, radius.CodeDisconnectACK, 0)
	sessionID, username, acctSessionID := seedCoAOnlineSession(t, secret, nas.port(t))
	itSeedSubscriber(t, username)
	session := itEnqueueDisconnect(t, sessionID)

	inserted, err := sessionqueue.EnqueueDisconnect(h.appCtx.DB(), session, sessionqueue.ReasonBillingSuspension, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.False(t, inserted, "a repeated billing run must not enqueue a duplicate action")

	c := newAPIClient(t)
	status, body := c.post(t, "/api/v1/session-actions/"+fmt.Sprint(itLoadAction(t, acctSessionID).ID)+"/retry", nil)
	require.Equalf(t, http.StatusOK, status, "retry body: %s", body)
	resp := itDecodeRetry(t, body)
	require.True(t, resp.Outcome.Success, "Disconnect-ACK must report success: %s", body)
	assert.Equal(t, "succeeded", resp.Action.Status)
	assert.Equal(t, 1, resp.Action.Attempts)
	assert.Equal(t, 1, resp.Action.ManualRetries)
	assert.Equal(t, "Disconnect-ACK", resp.Outcome.ResponseCode)
	assert.False(t, resp.Outcome.TimedOut)
	assert.Equal(t, 1, nas.requestCount(), "the NAS should receive exactly one request")
	assert.Equal(t, 0, nas.badAuthCount(), "Disconnect-Request must carry a valid Message-Authenticator (RFC 5176 §3.4)")

	action := itLoadAction(t, acctSessionID)
	assert.Equal(t, sessionqueue.StatusSucceeded, action.Status)
	assert.Equal(t, domain.DefaultTenantID, action.TenantID)
	audits := itActionAudits(t, acctSessionID)
	require.Len(t, audits, 1)
	assert.Equal(t, h.adminUser, audits[0].OperatorName)
	assert.True(t, audits[0].Success)
	assert.Equal(t, "Disconnect-ACK", audits[0].ResponseCode)
	assert.Equal(t, acctSessionID, audits[0].AcctSessionID)

	// The queue list is tenant-scoped and filterable, which is what the UI panel
	// uses to surface still-pending work.
	status, body = c.get(t, "/api/v1/session-actions?status=succeeded&username="+username)
	require.Equalf(t, http.StatusOK, status, "list body: %s", body)
	var page struct {
		Data []itSessionAction `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoErrorf(t, json.Unmarshal(body, &page), "list body: %s", body)
	assert.Equal(t, 1, page.Meta.Total)
	require.Len(t, page.Data, 1)
	assert.Equal(t, acctSessionID, page.Data[0].AcctSessionID)
}

// TestPostgresSessionActionQueueNakKeepsActionPending proves a NAS rejection is
// recorded with its Error-Cause, stays retryable within the automatic budget,
// and reaches the audit trail.
func TestPostgresSessionActionQueueNakKeepsActionPending(t *testing.T) {
	const secret = "it-action-secret-nak"
	nas := newITFakeNAS(t, secret, radius.CodeDisconnectNAK, rfc3576.ErrorCause_Value_SessionContextNotFound)
	sessionID, username, acctSessionID := seedCoAOnlineSession(t, secret, nas.port(t))
	itSeedSubscriber(t, username)
	itEnqueueDisconnect(t, sessionID)
	action := itLoadAction(t, acctSessionID)

	c := newAPIClient(t)
	status, body := c.post(t, "/api/v1/session-actions/"+fmt.Sprint(action.ID)+"/retry", nil)
	require.Equalf(t, http.StatusOK, status, "a NAK is a completed exchange, not an API error: %s", body)
	resp := itDecodeRetry(t, body)
	assert.False(t, resp.Outcome.Success)
	assert.Equal(t, "Disconnect-NAK", resp.Outcome.ResponseCode)
	assert.Equal(t, int(rfc3576.ErrorCause_Value_SessionContextNotFound), resp.Outcome.ErrorCause)
	assert.NotEmpty(t, resp.Outcome.ErrorCauseText)
	assert.Equal(t, sessionqueue.StatusPending, resp.Action.Status, "the action must remain retryable")
	assert.Equal(t, 1, resp.Action.Attempts)
	assert.True(t, resp.Action.NextRetryAt.After(time.Now()), "a pending action must be scheduled for a future retry")

	audits := itActionAudits(t, acctSessionID)
	require.Len(t, audits, 1)
	assert.False(t, audits[0].Success)
	assert.Equal(t, int(rfc3576.ErrorCause_Value_SessionContextNotFound), audits[0].ErrorCause)
	assert.Equal(t, h.adminUser, audits[0].OperatorName)
}

// TestPostgresSessionActionQueueTimeoutConvergesWhenNasReturns proves the
// scheduler path: an unreachable NAS leaves the action pending with a bounded
// backoff (and no duplicate row on the next billing run), and the automatic
// retry succeeds once the NAS comes back, audited as the system actor.
func TestPostgresSessionActionQueueTimeoutConvergesWhenNasReturns(t *testing.T) {
	const secret = "it-action-secret-offline"
	nas := newITFakeNAS(t, secret, 0, 0) // drop requests => client timeout
	sessionID, username, acctSessionID := seedCoAOnlineSession(t, secret, nas.port(t))
	itSeedSubscriber(t, username)
	session := itEnqueueDisconnect(t, sessionID)

	now := time.Now()
	summary, err := sessionqueue.ProcessDue(t.Context(), h.appCtx.DB(), domain.DefaultTenantID, now, itSessionActionSender(), 100)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Attempted)
	assert.Equal(t, 1, summary.Retried, "the offline NAS must leave the action pending")
	assert.GreaterOrEqual(t, nas.requestCount(), 1, "the scheduler attempted the Disconnect-Request")

	pending := itLoadAction(t, acctSessionID)
	assert.Equal(t, sessionqueue.StatusPending, pending.Status)
	assert.Equal(t, 1, pending.Attempts)
	assert.True(t, pending.LastTimedOut)
	assert.WithinDuration(t, now.Add(sessionqueue.Backoff(1)), pending.NextRetryAt, 5*time.Second)

	// The next billing run must observe the same action instead of creating a
	// second one for the still-suspended session.
	inserted, err := sessionqueue.EnqueueDisconnect(h.appCtx.DB(), session, sessionqueue.ReasonBillingSuspension, now.Add(time.Hour))
	require.NoError(t, err)
	assert.False(t, inserted)
	assert.Len(t, itActionAudits(t, acctSessionID), 1, "each automatic attempt writes one audit row")

	nas.setReplyCode(radius.CodeDisconnectACK)
	summary, err = sessionqueue.ProcessDue(t.Context(), h.appCtx.DB(), domain.DefaultTenantID, now.Add(2*time.Minute), itSessionActionSender(), 100)
	require.NoError(t, err)
	assert.Equal(t, 1, summary.Succeeded)

	converged := itLoadAction(t, acctSessionID)
	assert.Equal(t, sessionqueue.StatusSucceeded, converged.Status)
	assert.Equal(t, 2, converged.Attempts, "the successful attempt follows the failed one on the same row")
	assert.Equal(t, 0, converged.ManualRetries)

	audits := itActionAudits(t, acctSessionID)
	require.Len(t, audits, 2)
	assert.False(t, audits[0].Success)
	assert.True(t, audits[0].TimedOut)
	assert.Equal(t, sessionqueue.SystemActor.Name, audits[0].OperatorName)
	assert.True(t, audits[1].Success)
	assert.Equal(t, sessionqueue.SystemActor.Name, audits[1].OperatorName)
}

// TestPostgresSessionActionQueueConcurrentRetrySendsOnce proves the claim fence:
// two simultaneous operator retries result in exactly one Disconnect-Request on
// the wire, one 200 response and one 409 conflict.
func TestPostgresSessionActionQueueConcurrentRetrySendsOnce(t *testing.T) {
	const secret = "it-action-secret-concurrent"
	nas := newITFakeNAS(t, secret, radius.CodeDisconnectACK, 0)
	nas.setReplyDelay(400 * time.Millisecond) // widen the running window
	sessionID, username, acctSessionID := seedCoAOnlineSession(t, secret, nas.port(t))
	itSeedSubscriber(t, username)
	itEnqueueDisconnect(t, sessionID)
	action := itLoadAction(t, acctSessionID)

	c := newAPIClient(t)
	path := c.base + "/api/v1/session-actions/" + fmt.Sprint(action.ID) + "/retry"
	start := make(chan struct{})
	statuses := make(chan int, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			req, err := http.NewRequest(http.MethodPost, path, nil)
			if err != nil {
				errs <- err
				return
			}
			req.Header.Set("Authorization", "Bearer "+c.token)
			resp, err := c.http.Do(req)
			if err != nil {
				errs <- err
				return
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			statuses <- resp.StatusCode
		}()
	}
	close(start)
	wg.Wait()
	close(statuses)
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}

	var got []int
	for status := range statuses {
		got = append(got, status)
	}
	require.Len(t, got, 2)
	assert.ElementsMatch(t, []int{http.StatusOK, http.StatusConflict}, got,
		"exactly one retry may win the claim; the other must conflict")
	assert.Equal(t, 1, nas.requestCount(), "the loser must not transmit a second Disconnect-Request")

	succeeded := itLoadAction(t, acctSessionID)
	assert.Equal(t, sessionqueue.StatusSucceeded, succeeded.Status)
	assert.Equal(t, 1, succeeded.Attempts)
	assert.Equal(t, 1, succeeded.ManualRetries)
	assert.Len(t, itActionAudits(t, acctSessionID), 1, "only the winning attempt writes an audit row")
}

// TestPostgresSessionActionQueueCanceledWhenSubscriberRestored proves a queued
// Disconnect does not terminate a session whose subscriber was re-enabled before
// the action ran.
func TestPostgresSessionActionQueueCanceledWhenSubscriberRestored(t *testing.T) {
	const secret = "it-action-secret-restored"
	nas := newITFakeNAS(t, secret, radius.CodeDisconnectACK, 0)
	sessionID, username, acctSessionID := seedCoAOnlineSession(t, secret, nas.port(t))
	user := itSeedSubscriber(t, username)
	itEnqueueDisconnect(t, sessionID)

	require.NoError(t, h.appCtx.DB().Model(&domain.RadiusUser{}).Where("id = ?", user.ID).
		Update("status", common.ENABLED).Error)

	action := itLoadAction(t, acctSessionID)
	c := newAPIClient(t)
	status, body := c.post(t, "/api/v1/session-actions/"+fmt.Sprint(action.ID)+"/retry", nil)
	require.Equalf(t, http.StatusOK, status, "canceling is a successful, audited decision: %s", body)
	resp := itDecodeRetry(t, body)
	assert.False(t, resp.Outcome.Success)
	assert.Equal(t, sessionqueue.StatusCanceled, resp.Action.Status)
	assert.Contains(t, resp.Action.LastError, "restored")
	assert.Equal(t, 0, nas.requestCount(), "a restored subscriber must not be disconnected")
	assert.Empty(t, itActionAudits(t, acctSessionID), "a canceled action never reached the NAS, so it has no attempt audit")

	status, body = c.post(t, "/api/v1/session-actions/"+fmt.Sprint(action.ID)+"/retry", nil)
	assert.Equalf(t, http.StatusConflict, status, "a canceled action is terminal: %s", body)
}

// TestPostgresSessionActionQueueIsTenantScoped proves queue rows, retries and
// idempotency keys are isolated per tenant on PostgreSQL: the default-tenant
// admin cannot see or retry another tenant's action, and the unique key allows
// the same logical action under two tenants.
func TestPostgresSessionActionQueueIsTenantScoped(t *testing.T) {
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	tenantA := domain.Tenant{Name: "Queue tenant A " + suffix, Slug: "it-queue-a-" + suffix, Kind: "isp", Status: "active"}
	tenantB := domain.Tenant{Name: "Queue tenant B " + suffix, Slug: "it-queue-b-" + suffix, Kind: "isp", Status: "active"}
	require.NoError(t, db.Create(&tenantA).Error)
	require.NoError(t, db.Create(&tenantB).Error)

	// The same session ID and reason produce the same idempotency key; two
	// tenants must still own one row each.
	base := domain.RadiusOnline{
		ID:            common.UUIDint64(),
		Username:      "it-queue-tenant-" + suffix,
		NasAddr:       "192.0.2.44",
		AcctSessionId: "it-queue-tenant-sess-" + suffix,
	}
	sessionA := base
	sessionA.TenantID = tenantA.ID
	sessionB := base
	sessionB.TenantID = tenantB.ID

	inserted, err := sessionqueue.EnqueueDisconnect(db, sessionA, sessionqueue.ReasonBillingSuspension, time.Now())
	require.NoError(t, err)
	require.True(t, inserted)
	inserted, err = sessionqueue.EnqueueDisconnect(db, sessionB, sessionqueue.ReasonBillingSuspension, time.Now())
	require.NoError(t, err)
	require.True(t, inserted, "the idempotency key must be unique per tenant, not globally")
	inserted, err = sessionqueue.EnqueueDisconnect(db, sessionA, sessionqueue.ReasonBillingSuspension, time.Now())
	require.NoError(t, err)
	assert.False(t, inserted)

	var rows []domain.RadiusSessionAction
	require.NoError(t, db.Where("acct_session_id = ?", base.AcctSessionId).Order("tenant_id ASC").Find(&rows).Error)
	require.Len(t, rows, 2)
	t.Cleanup(func() {
		_ = db.Where("acct_session_id = ?", base.AcctSessionId).Delete(&domain.RadiusSessionAction{}).Error
	})

	// The shared platform admin token acts inside the default tenant, so both
	// foreign rows must be invisible and unretryable over HTTP.
	c := newAPIClient(t)
	status, body := c.get(t, "/api/v1/session-actions?username="+base.Username)
	require.Equalf(t, http.StatusOK, status, "list body: %s", body)
	var page struct {
		Data []itSessionAction `json:"data"`
		Meta struct {
			Total int `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(body, &page))
	assert.Zero(t, page.Meta.Total, "another tenant's queue must not be listed")
	assert.Empty(t, page.Data)

	for _, row := range rows {
		status, body = c.post(t, "/api/v1/session-actions/"+fmt.Sprint(row.ID)+"/retry", nil)
		assert.Equalf(t, http.StatusNotFound, status, "cross-tenant retry must be hidden: %s", body)
	}

	// The tenant-scoped database handle sees only its own row.
	var tenantARows []domain.RadiusSessionAction
	require.NoError(t, db.WithContext(tenancy.WithTenantID(t.Context(), tenantA.ID)).
		Where("acct_session_id = ?", base.AcctSessionId).Find(&tenantARows).Error)
	require.Len(t, tenantARows, 1)
	assert.Equal(t, tenantA.ID, tenantARows[0].TenantID)
}
