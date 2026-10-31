package adminapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// sessionActionStatuses is the set of statuses accepted by the list filter.
var sessionActionStatuses = map[string]bool{
	sessionqueue.StatusPending:   true,
	sessionqueue.StatusRunning:   true,
	sessionqueue.StatusSucceeded: true,
	sessionqueue.StatusFailed:    true,
	sessionqueue.StatusCanceled:  true,
}

// ListSessionActions returns durable session actions (currently billing-driven
// Disconnect-Requests) for the authenticated tenant, newest first. Operators use
// it to see which sessions are still pending, scheduled for retry, or failed
// after exhausting the automatic attempt budget.
//
// @Summary List queued and completed session actions
// @Tags OnlineSession
// @Param status query string false "Filter by status (pending, running, succeeded, failed, canceled)"
// @Param username query string false "Filter by subscriber username"
// @Param reason query string false "Filter by reason"
// @Success 200 {object} Response
// @Router /api/v1/session-actions [get]
func ListSessionActions(c echo.Context) error {
	page, pageSize := parsePagination(c)
	tenantID := domain.DefaultTenantID
	if scopedID, ok := tenancy.TenantID(c.Request().Context()); ok {
		tenantID = scopedID
	}
	query := GetDB(c).Model(&domain.RadiusSessionAction{}).Where("tenant_id = ?", tenantID)
	if raw := c.QueryParam("status"); raw != "" {
		statuses := strings.Split(raw, ",")
		for i := range statuses {
			statuses[i] = strings.TrimSpace(statuses[i])
			if !sessionActionStatuses[statuses[i]] {
				return fail(c, http.StatusBadRequest, "INVALID_STATUS", "Unknown session action status", nil)
			}
		}
		query = query.Where("status IN ?", statuses)
	}
	if username := c.QueryParam("username"); username != "" {
		query = query.Where("username = ?", username)
	}
	if reason := c.QueryParam("reason"); reason != "" {
		query = query.Where("reason = ?", reason)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to count session actions", err.Error())
	}
	var actions []domain.RadiusSessionAction
	if err := query.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&actions).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load session actions", err.Error())
	}
	return paged(c, actions, total, page, pageSize)
}

// sessionActionSummaryView is the per-tenant queue health surface used by the
// operations console and alerting. DeadLetter counts the actions that exhausted
// the automatic retry budget and now need an operator decision: retry them
// manually, cancel them, or fix the NAS and let a retry converge the session.
type sessionActionSummaryView struct {
	Counts          map[string]int64 `json:"counts"`
	DeadLetter      int64            `json:"dead_letter"`
	OldestPendingAt *time.Time       `json:"oldest_pending_at,omitempty"`
}

// SessionActionSummary reports how many durable session actions sit in each
// status for the authenticated tenant, how many are dead-lettered, and how old
// the oldest pending action is. It reads only the tenant's own queue.
//
// @Summary Summarize the tenant's session-action queue
// @Tags OnlineSession
// @Success 200 {object} Response
// @Router /api/v1/session-actions/summary [get]
func SessionActionSummary(c echo.Context) error {
	tenantID := domain.DefaultTenantID
	if scopedID, ok := tenancy.TenantID(c.Request().Context()); ok {
		tenantID = scopedID
	}
	var rows []struct {
		Status string
		Total  int64
	}
	if err := GetDB(c).Model(&domain.RadiusSessionAction{}).
		Select("status, COUNT(*) AS total").
		Where("tenant_id = ?", tenantID).
		Group("status").Scan(&rows).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to summarize session actions", err.Error())
	}
	counts := map[string]int64{
		sessionqueue.StatusPending: 0, sessionqueue.StatusRunning: 0, sessionqueue.StatusSucceeded: 0,
		sessionqueue.StatusFailed: 0, sessionqueue.StatusCanceled: 0,
	}
	for _, row := range rows {
		counts[row.Status] = row.Total
	}
	summary := sessionActionSummaryView{Counts: counts, DeadLetter: counts[sessionqueue.StatusFailed]}

	var oldest domain.RadiusSessionAction
	err := GetDB(c).Select("created_at").
		Where("tenant_id = ? AND status = ?", tenantID, sessionqueue.StatusPending).
		Order("created_at ASC").First(&oldest).Error
	switch {
	case err == nil:
		created := oldest.CreatedAt
		summary.OldestPendingAt = &created
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to inspect pending session actions", err.Error())
	}
	return ok(c, summary)
}

// sessionActionOutcome is the snake_case projection of sessionqueue.Outcome
// returned after a manual retry.
type sessionActionOutcome struct {
	Action         string `json:"action"`
	Target         string `json:"target"`
	Username       string `json:"username"`
	AcctSessionID  string `json:"acct_session_id"`
	Success        bool   `json:"success"`
	ResponseCode   string `json:"response_code,omitempty"`
	ErrorCause     int    `json:"error_cause,omitempty"`
	ErrorCauseText string `json:"error_cause_text,omitempty"`
	Attempts       int    `json:"attempts"`
	RTTMillis      int64  `json:"rtt_ms"`
	TimedOut       bool   `json:"timed_out"`
	Error          string `json:"error,omitempty"`
}

// RetrySessionAction immediately retries a pending or failed session action. A
// manual retry resets the automatic attempt budget, records the authenticated
// operator on the audit row, and reports the NAS's structured response.
//
// Unknown actions return 404, and actions that are already running or in a
// succeeded/canceled state return 409. A NAK or timeout is reported with HTTP
// 200 and success=false: the attempt genuinely ran and its outcome is payload.
//
// @Summary Retry a session action immediately
// @Tags OnlineSession
// @Param id path int true "Session action ID"
// @Success 200 {object} Response
// @Router /api/v1/session-actions/{id}/retry [post]
func RetrySessionAction(c echo.Context) error {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid session action ID", nil)
	}

	tenantID := domain.DefaultTenantID
	if scopedID, ok := tenancy.TenantID(c.Request().Context()); ok {
		tenantID = scopedID
	}
	actor := sessionqueue.Actor{IP: c.RealIP()}
	if op, err := resolveOperatorFromContext(c); err == nil && op != nil {
		actor.ID = op.ID
		actor.Name = op.Username
	}

	action, outcome, err := sessionqueue.RetryNow(c.Request().Context(), GetDB(c), tenantID, id,
		time.Now(), sessionActionSenderForRequest(c), actor)
	switch {
	case errors.Is(err, sessionqueue.ErrNotFound):
		return fail(c, http.StatusNotFound, "NOT_FOUND", "Session action not found", nil)
	case errors.Is(err, sessionqueue.ErrRunning):
		return fail(c, http.StatusConflict, "ACTION_RUNNING", "Session action is already being executed", nil)
	case errors.Is(err, sessionqueue.ErrNotRetryable):
		return fail(c, http.StatusConflict, "NOT_RETRYABLE", "Only pending or failed actions can be retried", nil)
	case err != nil:
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to retry session action", err.Error())
	}

	logSessionActionAttempt(actor, action, outcome)
	return ok(c, struct {
		Action  domain.RadiusSessionAction `json:"action"`
		Outcome sessionActionOutcome       `json:"outcome"`
	}{
		Action: *action,
		Outcome: sessionActionOutcome{
			Action:         action.Action,
			Target:         outcome.Target,
			Username:       action.Username,
			AcctSessionID:  action.AcctSessionID,
			Success:        outcome.Success,
			ResponseCode:   outcome.ResponseCode,
			ErrorCause:     outcome.ErrorCause,
			ErrorCauseText: outcome.ErrorCauseText,
			Attempts:       outcome.Attempts,
			RTTMillis:      outcome.RTTMillis,
			TimedOut:       outcome.TimedOut,
			Error:          outcome.Error,
		},
	})
}

// sessionActionSenderForRequest builds a sessionqueue.Sender that resolves the
// NAS through the request database handle (so tenant scoping and test doubles
// both apply) and runs a synchronous RFC 5176 Disconnect-Request.
func sessionActionSenderForRequest(c echo.Context) sessionqueue.Sender {
	return func(ctx context.Context, session domain.RadiusOnline) sessionqueue.Outcome {
		var nas domain.NetNas
		if err := GetDB(c).WithContext(ctx).Where("ipaddr = ?", session.NasAddr).First(&nas).Error; err != nil {
			return sessionqueue.Outcome{Target: session.NasAddr,
				Error: fmt.Sprintf("resolve NAS for session %d: %v", session.ID, err)}
		}
		target := radiusd.CoATargetFromNas(&nas)
		result, err := sessionCoAService().Disconnect(ctx, target, radiusd.SessionIdentityFromOnline(&session))
		if err != nil {
			return sessionqueue.Outcome{Target: target.Addr, Error: err.Error()}
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

// logSessionActionAttempt logs the operator-triggered retry. The durable audit
// row is written by the queue with the operator identity and structured result.
func logSessionActionAttempt(actor sessionqueue.Actor, action *domain.RadiusSessionAction, outcome sessionqueue.Outcome) {
	fields := []zap.Field{
		zap.String("namespace", "adminapi"),
		zap.String("operator", actor.Name),
		zap.String("action", action.Action),
		zap.String("username", action.Username),
		zap.String("acct_session_id", action.AcctSessionID),
		zap.String("status", action.Status),
		zap.Bool("success", outcome.Success),
		zap.String("response_code", outcome.ResponseCode),
		zap.Int("attempts", outcome.Attempts),
	}
	if outcome.Success {
		zap.L().Info("operator retried a session action", fields...)
	} else {
		zap.L().Warn("operator retried a session action without success", fields...)
	}
}
