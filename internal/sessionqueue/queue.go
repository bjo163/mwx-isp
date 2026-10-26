// Package sessionqueue persists required NAS session actions and retries them
// until they succeed or exhaust a bounded attempt budget.
//
// Billing suspension must converge the subscriber's actual access at the NAS
// with the database state even when the NAS is temporarily offline. A
// Disconnect-Request that fails is therefore recorded as a pending action
// instead of only being logged. The scheduler claims due actions, calls a
// caller-supplied sender (the RADIUS Dynamic Authorization client), and applies
// exponential backoff. Manual retries are admin-triggered and audited.
//
// Every query in this package is scoped to one tenant, and the queue never
// stores NAS secrets or subscriber passwords.
package sessionqueue

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// ActionDisconnect is the only action currently queued: it asks the NAS to
	// terminate the identified online session (RFC 5176 §2.1).
	ActionDisconnect = "disconnect"
	// ReasonBillingSuspension marks actions created by the billing scheduler.
	ReasonBillingSuspension = "billing_suspension"

	// StatusPending is a claimed-but-not-finished action awaiting its next
	// attempt; StatusRunning is held by one worker; StatusSucceeded,
	// StatusFailed and StatusCanceled are terminal.
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"

	// DefaultMaxAttempts bounds automatic retries so a permanently unreachable
	// NAS cannot cause an unbounded loop. Manual retries reset this budget.
	DefaultMaxAttempts = 8

	// staleRunningAfter is how long a claimed action may stay running before a
	// later scheduler pass assumes the worker died and makes it claimable again.
	staleRunningAfter = 10 * time.Minute
	// sendTimeout bounds one sender call so a hung transport cannot block the
	// scheduler loop indefinitely.
	sendTimeout = 20 * time.Second
	// maxBackoff caps the automatic retry interval.
	maxBackoff = 6 * time.Hour
)

// Errors returned by RetryNow and queue claims.
var (
	// ErrNotFound reports that the action does not exist in the tenant.
	ErrNotFound = errors.New("session action not found")
	// ErrNotRetryable reports that the action is already terminal.
	ErrNotRetryable = errors.New("session action is not retryable")
	// ErrRunning reports that another worker currently holds the action.
	ErrRunning    = errors.New("session action is already running")
	errNotClaimed = errors.New("session action was claimed by another worker")
	errNoSender   = errors.New("session action sender is not configured")
)

// Outcome is the caller-facing result of one Disconnect-Request attempt. It
// mirrors the structured fields of radiusd.CoAResult without importing the
// protocol package, which would create an import cycle.
type Outcome struct {
	Success        bool
	Target         string
	ResponseCode   string
	ErrorCause     int
	ErrorCauseText string
	Error          string
	TimedOut       bool
	Attempts       int
	RTTMillis      int64
}

// Sender performs one session action against the NAS. It must return promptly;
// the queue also enforces sendTimeout. Errors are reported through Outcome so a
// NAK and a transport failure are both recorded in the audit trail.
type Sender func(context.Context, domain.RadiusOnline) Outcome

// Actor identifies who triggered an attempt. The scheduler uses SystemActor;
// manual retries pass the authenticated operator.
type Actor struct {
	ID   int64
	Name string
	IP   string
}

// SystemActor is recorded for scheduler-driven attempts.
var SystemActor = Actor{Name: "system:scheduler"}

// Summary reports what one ProcessDue pass did. Attempted counts claimed
// actions, Succeeded/Failed/Canceled are terminal transitions, Retried counts
// actions rescheduled for another attempt, and AuditErrors counts audit rows
// that could not be written (the action state is still finalized).
type Summary struct {
	Attempted   int
	Succeeded   int
	Failed      int
	Canceled    int
	Retried     int
	AuditErrors int
}

// Backoff returns the delay before retrying after the given number of completed
// attempts: 1 minute, doubling to a 6-hour cap.
func Backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := time.Minute
	for i := 1; i < attempt && delay < maxBackoff; i++ {
		delay *= 2
	}
	if delay > maxBackoff {
		delay = maxBackoff
	}
	return delay
}

// EnqueueDisconnect records a required Disconnect-Request for the session. The
// idempotency key is derived from the tenant, reason and session row, so
// repeated scheduler runs do not enqueue duplicates. It reports whether a new
// row was inserted.
func EnqueueDisconnect(db *gorm.DB, session domain.RadiusOnline, reason string, now time.Time) (bool, error) {
	if session.ID <= 0 || session.TenantID <= 0 {
		return false, fmt.Errorf("session action requires a tenant and session ID")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return false, fmt.Errorf("session action requires a reason")
	}
	if now.IsZero() {
		now = time.Now()
	}
	key := fmt.Sprintf("%s:%s:%d", ActionDisconnect, reason, session.ID)
	action := domain.RadiusSessionAction{
		TenantID:       session.TenantID,
		IdempotencyKey: key,
		SessionID:      session.ID,
		AcctSessionID:  session.AcctSessionId,
		Username:       session.Username,
		NasAddr:        session.NasAddr,
		Action:         ActionDisconnect,
		Reason:         reason,
		Status:         StatusPending,
		MaxAttempts:    DefaultMaxAttempts,
		NextRetryAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	result := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "idempotency_key"}},
		DoNothing: true,
	}).Create(&action)
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected > 0, nil
}

// ProcessDue claims and executes pending actions for one tenant whose retry
// time has arrived. Running actions abandoned by a crashed worker are made
// claimable again after staleRunningAfter. Per-action database errors are
// reported through the returned error after the pass continues; a nil sender
// makes the call a no-op.
func ProcessDue(ctx context.Context, db *gorm.DB, tenantID int64, now time.Time, send Sender, limit int) (Summary, error) {
	var summary Summary
	if tenantID <= 0 {
		return summary, fmt.Errorf("tenant ID is required")
	}
	if send == nil {
		return summary, nil
	}
	if now.IsZero() {
		now = time.Now()
	}
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	scoped := db.WithContext(tenancy.WithTenantID(ctx, tenantID))

	if err := recoverStale(scoped, tenantID, now); err != nil {
		return summary, err
	}

	var due []domain.RadiusSessionAction
	if err := scoped.
		Where("tenant_id = ? AND status = ? AND next_retry_at <= ?", tenantID, StatusPending, now).
		Order("next_retry_at ASC, id ASC").
		Limit(limit).
		Find(&due).Error; err != nil {
		return summary, err
	}

	var firstErr error
	for i := range due {
		action, err := claim(scoped, tenantID, due[i].ID, []string{StatusPending}, now, false)
		if err != nil {
			if errors.Is(err, errNotClaimed) {
				continue
			}
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		_, terminal, auditErr := execute(ctx, scoped, action, now, send, SystemActor)
		summary.Attempted++
		if auditErr != nil {
			summary.AuditErrors++
		}
		switch terminal {
		case StatusSucceeded:
			summary.Succeeded++
		case StatusFailed:
			summary.Failed++
		case StatusCanceled:
			summary.Canceled++
		case StatusPending:
			summary.Retried++
		}
	}
	return summary, firstErr
}

// RetryNow force-claims a pending or failed action and executes it immediately
// with the supplied operator actor. A manual retry resets the automatic attempt
// budget, increments ManualRetries, and writes an audit row with the operator's
// identity. The returned action reflects the post-attempt state.
func RetryNow(ctx context.Context, db *gorm.DB, tenantID, id int64, now time.Time, send Sender, actor Actor) (*domain.RadiusSessionAction, Outcome, error) {
	if tenantID <= 0 || id <= 0 {
		return nil, Outcome{}, ErrNotFound
	}
	if send == nil {
		return nil, Outcome{}, errNoSender
	}
	if now.IsZero() {
		now = time.Now()
	}
	if strings.TrimSpace(actor.Name) == "" {
		actor.Name = "unknown"
	}
	scoped := db.WithContext(tenancy.WithTenantID(ctx, tenantID))

	var current domain.RadiusSessionAction
	if err := scoped.Where("id = ? AND tenant_id = ?", id, tenantID).First(&current).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, Outcome{}, ErrNotFound
		}
		return nil, Outcome{}, err
	}
	switch current.Status {
	case StatusRunning:
		return &current, Outcome{}, ErrRunning
	case StatusSucceeded, StatusCanceled:
		return &current, Outcome{}, ErrNotRetryable
	}

	action, err := claim(scoped, tenantID, id, []string{StatusPending, StatusFailed}, now, true)
	if err != nil {
		if errors.Is(err, errNotClaimed) {
			var latest domain.RadiusSessionAction
			if loadErr := scoped.Where("id = ? AND tenant_id = ?", id, tenantID).First(&latest).Error; loadErr == nil {
				if latest.Status == StatusRunning {
					return &latest, Outcome{}, ErrRunning
				}
				return &latest, Outcome{}, ErrNotRetryable
			}
		}
		return nil, Outcome{}, err
	}

	outcome, _, auditErr := execute(ctx, scoped, action, now, send, actor)
	var updated domain.RadiusSessionAction
	if err := scoped.Where("id = ? AND tenant_id = ?", id, tenantID).First(&updated).Error; err != nil {
		return nil, outcome, err
	}
	if auditErr != nil {
		return &updated, outcome, fmt.Errorf("persist session action audit: %w", auditErr)
	}
	return &updated, outcome, nil
}

// claim atomically takes ownership of an action and increments its attempt
// counter. The fencing token in ClaimToken prevents a worker whose sender
// overran from finalizing an action that a later retry has since claimed.
func claim(db *gorm.DB, tenantID, id int64, from []string, now time.Time, reset bool) (*domain.RadiusSessionAction, error) {
	token, err := newClaimToken()
	if err != nil {
		return nil, err
	}
	updates := map[string]any{
		"status":          StatusRunning,
		"claim_token":     token,
		"last_attempt_at": now,
		"updated_at":      now,
	}
	if reset {
		updates["attempts"] = 1
		updates["manual_retries"] = gorm.Expr("manual_retries + 1")
	} else {
		updates["attempts"] = gorm.Expr("attempts + 1")
	}
	result := db.Model(&domain.RadiusSessionAction{}).
		Where("id = ? AND tenant_id = ? AND status IN ?", id, tenantID, from).
		Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, errNotClaimed
	}
	var action domain.RadiusSessionAction
	if err := db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&action).Error; err != nil {
		return nil, err
	}
	if action.ClaimToken != token {
		return nil, errNotClaimed
	}
	return &action, nil
}

// execute loads the target session, cancels the action when the desired access
// state no longer requires a change, otherwise invokes the sender and finalizes
// the action. The returned terminal status is one of succeeded, failed,
// canceled or pending (scheduled for retry). Audit failure is returned
// separately because it must not roll back the action state.
func execute(ctx context.Context, db *gorm.DB, action *domain.RadiusSessionAction, now time.Time, send Sender, actor Actor) (Outcome, string, error) {
	tenantCtx := tenancy.WithTenantID(ctx, action.TenantID)

	var session domain.RadiusOnline
	if err := db.WithContext(tenantCtx).
		Where("id = ? AND tenant_id = ?", action.SessionID, action.TenantID).
		First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Outcome{}, StatusCanceled, finalize(db, action, now, StatusCanceled, Outcome{}, "session is no longer online")
		}
		return Outcome{}, StatusRunning, err
	}

	// If the subscriber was re-enabled before the queued Disconnect ran, the
	// required access state has already been restored; do not terminate a new
	// or still-valid session.
	var user domain.RadiusUser
	if err := db.WithContext(tenantCtx).Where("username = ?", action.Username).First(&user).Error; err == nil {
		if !strings.EqualFold(strings.TrimSpace(user.Status), "disabled") {
			return Outcome{}, StatusCanceled, finalize(db, action, now, StatusCanceled, Outcome{}, "subscriber access was restored before the action ran")
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Outcome{}, StatusRunning, err
	}

	attemptCtx, cancel := context.WithTimeout(tenantCtx, sendTimeout)
	outcome := send(attemptCtx, session)
	cancel()

	status := StatusPending
	if outcome.Success {
		status = StatusSucceeded
	} else if action.Attempts >= action.MaxAttempts {
		status = StatusFailed
	}

	if err := finalize(db, action, now, status, outcome, ""); err != nil {
		return outcome, status, err
	}
	auditErr := writeAudit(db, action, now, outcome, actor)
	return outcome, status, auditErr
}

// finalize writes the post-attempt state only if the caller still owns the
// action. Cancel reasons are stored in LastError; attempt results are stored
// from outcome.
func finalize(db *gorm.DB, action *domain.RadiusSessionAction, now time.Time, status string, outcome Outcome, cancelReason string) error {
	updates := map[string]any{
		"status":      status,
		"claim_token": "",
		"updated_at":  now,
	}
	switch status {
	case StatusSucceeded:
		updates["succeeded_at"] = now
		updates["next_retry_at"] = time.Time{}
	case StatusFailed:
		updates["next_retry_at"] = time.Time{}
	case StatusCanceled:
		updates["next_retry_at"] = time.Time{}
		updates["last_error"] = truncate(cancelReason, 512)
	default:
		updates["next_retry_at"] = now.Add(Backoff(action.Attempts))
	}
	if status != StatusCanceled {
		updates["last_response_code"] = truncate(outcome.ResponseCode, 32)
		updates["last_error_cause"] = outcome.ErrorCause
		updates["last_error_cause_text"] = truncate(outcome.ErrorCauseText, 128)
		updates["last_error"] = truncate(outcome.Error, 512)
		updates["last_timed_out"] = outcome.TimedOut
		updates["last_wire_attempts"] = outcome.Attempts
		updates["last_rtt_millis"] = outcome.RTTMillis
	}
	result := db.Model(&domain.RadiusSessionAction{}).
		Where("id = ? AND tenant_id = ? AND status = ? AND claim_token = ?",
			action.ID, action.TenantID, StatusRunning, action.ClaimToken).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errNotClaimed
	}
	return nil
}

// writeAudit appends an operator-readable record for every completed attempt.
// Automatic attempts use the system actor; manual retries carry the operator's
// identity. Secrets are never part of the queued fields or the outcome.
func writeAudit(db *gorm.DB, action *domain.RadiusSessionAction, now time.Time, outcome Outcome, actor Actor) error {
	record := domain.RadiusSessionActionAudit{
		TenantID:       action.TenantID,
		SessionID:      action.SessionID,
		AcctSessionID:  action.AcctSessionID,
		Action:         action.Action,
		Username:       action.Username,
		NasAddr:        action.NasAddr,
		Target:         outcome.Target,
		OperatorID:     actor.ID,
		OperatorName:   actor.Name,
		OperatorIP:     actor.IP,
		Success:        outcome.Success,
		ResponseCode:   truncate(outcome.ResponseCode, 64),
		ErrorCause:     outcome.ErrorCause,
		ErrorCauseText: truncate(outcome.ErrorCauseText, 128),
		Attempts:       outcome.Attempts,
		RTTMillis:      outcome.RTTMillis,
		TimedOut:       outcome.TimedOut,
		Error:          truncate(outcome.Error, 512),
		TriggeredAt:    now,
	}
	return db.WithContext(tenancy.WithTenantID(context.Background(), action.TenantID)).Create(&record).Error
}

// recoverStale requeues actions whose worker disappeared. The attempt counter
// was already incremented at claim time, so crash recovery cannot create an
// unbounded retry loop.
func recoverStale(db *gorm.DB, tenantID int64, now time.Time) error {
	return db.Model(&domain.RadiusSessionAction{}).
		Where("tenant_id = ? AND status = ? AND updated_at < ?", tenantID, StatusRunning, now.Add(-staleRunningAfter)).
		Updates(map[string]any{
			"status":        StatusPending,
			"claim_token":   "",
			"next_retry_at": now,
			"updated_at":    now,
		}).Error
}

func newClaimToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate session action claim token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return value[:max]
}
