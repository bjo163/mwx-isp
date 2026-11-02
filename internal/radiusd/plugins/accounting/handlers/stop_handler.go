package handlers

import (
	"errors"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting"
	vendorparserspkg "github.com/bjo163/mwx-isp/internal/radiusd/plugins/vendorparsers"
	"github.com/bjo163/mwx-isp/internal/radiusd/repository"
	"go.uber.org/zap"
	"layeh.com/radius/rfc2866"
)

// StopHandler handles Accounting-Stop packets and finalizes session accounting.
//
// The handler updates terminal traffic/session counters in the accounting
// ledger and then removes the online-session row so the session no longer
// appears active.
type StopHandler struct {
	sessionRepo    repository.SessionRepository
	accountingRepo repository.AccountingRepository
}

// NewStopHandler constructs a StopHandler with repositories used to update the
// final accounting counters and clear online-session state.
func NewStopHandler(
	sessionRepo repository.SessionRepository,
	accountingRepo repository.AccountingRepository,
) *StopHandler {
	return &StopHandler{
		sessionRepo:    sessionRepo,
		accountingRepo: accountingRepo,
	}
}

// Name returns the stable plugin name used by the accounting dispatcher.
func (h *StopHandler) Name() string {
	return "StopHandler"
}

// CanHandle reports whether the context represents an Accounting-Stop packet.
func (h *StopHandler) CanHandle(ctx *accounting.AccountingContext) bool {
	return ctx.StatusType == int(rfc2866.AcctStatusType_Value_Stop)
}

// Handle updates the session's stop-time counters and removes the matching
// online-session row.
//
// When updating the accounting row fails, Handle logs the error and still tries
// to remove the online-session record so stale sessions do not remain online
// forever. Handle returns an error only when deleting the online-session row
// fails.
//
// When the Start task has not yet persisted the accounting row (it runs
// asynchronously after the response is sent), Handle falls back to UpsertStop
// which materializes the terminal record from the Stop packet itself so no
// traffic is lost. In that case there is no online-session row to delete, so
// Handle treats a missing online session as a no-op rather than an error.
func (h *StopHandler) Handle(acctCtx *accounting.AccountingContext) error {
	zap.L().Debug("stop handler called",
		zap.String("namespace", "radius"),
		zap.String("username", acctCtx.Username),
		zap.String("session_id", rfc2866.AcctSessionID_GetString(acctCtx.Request.Packet)),
		zap.Int("status_type", acctCtx.StatusType),
	)
	vendorReq := acctCtx.VendorReq
	if vendorReq == nil {
		vendorReq = &vendorparserspkg.VendorRequest{}
	}

	// Build online session data
	online := buildOnlineFromRequest(acctCtx, vendorReq)
	sessionId := rfc2866.AcctSessionID_GetString(acctCtx.Request.Packet)
	zap.L().Debug("stop handler online session id",
		zap.String("namespace", "radius"),
		zap.String("session_id", sessionId),
		zap.String("online_session_id", online.AcctSessionId),
	)

	// Update accounting record stop time
	acctRecord := domain.RadiusAccounting{
		TenantID:          online.TenantID,
		Username:          online.Username,
		AcctSessionId:     online.AcctSessionId,
		AcctInputTotal:    online.AcctInputTotal,
		AcctOutputTotal:   online.AcctOutputTotal,
		AcctInputPackets:  online.AcctInputPackets,
		AcctOutputPackets: online.AcctOutputPackets,
		AcctSessionTime:   online.AcctSessionTime,
		// If Start hasn't run yet, we don't know the exact start time. Use the
		// session time from the Stop packet to approximate it.
		AcctStartTime:     time.Now().Add(-time.Duration(online.AcctSessionTime) * time.Second),
	}

	err := h.accountingRepo.UpdateStop(acctCtx.Context, sessionId, &acctRecord)
	if err != nil {
		// The Start task runs asynchronously after the response is sent, so a
		// Stop can legitimately arrive before the accounting row exists. Rather
		// than dropping the Stop, fall back to UpsertStop which materializes the
		// terminal record from the Stop packet itself so the session is still
		// accounted for and no traffic is lost.
		if errors.Is(err, repository.ErrAccountingNotFound) {
			zap.L().Warn("accounting stop arrived before start, upserting terminal record",
				zap.String("namespace", "radius"),
				zap.String("username", acctCtx.Username),
				zap.String("session_id", sessionId),
			)
			if upsertErr := h.accountingRepo.UpsertStop(acctCtx.Context, sessionId, &acctRecord); upsertErr != nil {
				zap.L().Error("upsert radius accounting stop error",
					zap.String("namespace", "radius"),
					zap.String("username", acctCtx.Username),
					zap.String("session_id", sessionId),
					zap.Error(upsertErr),
				)
			}
		} else {
			zap.L().Error("update radius accounting stop error",
				zap.String("namespace", "radius"),
				zap.String("username", acctCtx.Username),
				zap.String("session_id", sessionId),
				zap.Error(err),
			)
		}
	} else {
		zap.L().Debug("update radius accounting stop success",
			zap.String("namespace", "radius"),
			zap.String("username", acctCtx.Username),
			zap.String("session_id", sessionId),
		)
	}

	// Delete the online session. If the Stop arrived before the Start task,
	// there is no online row yet; treat the delete as a no-op rather than an
	// error, because the accounting side has already been handled above.
	if delErr := h.sessionRepo.Delete(acctCtx.Context, sessionId); delErr != nil {
		zap.L().Error("delete radius online error",
			zap.String("namespace", "radius"),
			zap.String("username", acctCtx.Username),
			zap.String("session_id", sessionId),
			zap.Error(delErr),
		)
		return delErr
	}

	return nil
}
