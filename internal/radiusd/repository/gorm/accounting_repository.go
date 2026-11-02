// Package gorm provides GORM-backed implementations of repository interfaces.
package gorm

import (
	"context"
	"errors"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/repository"
	"github.com/bjo163/mwx-isp/pkg/common"
	"gorm.io/gorm"
)

// GormAccountingRepository is the GORM implementation of the accounting repository
type GormAccountingRepository struct {
	db *gorm.DB
}

// NewGormAccountingRepository creates an accounting repository instance
func NewGormAccountingRepository(db *gorm.DB) repository.AccountingRepository {
	return &GormAccountingRepository{db: db}
}

func (r *GormAccountingRepository) Create(ctx context.Context, accounting *domain.RadiusAccounting) error {
	if accounting.ID == 0 {
		accounting.ID = common.UUIDint64()
	}
	return r.db.WithContext(ctx).Create(accounting).Error
}

// UpdateStop finalizes the newest accounting row for acct_session_id.
//
// Acct-Session-Id is not unique in radius_accounting: a NAS may reuse an
// identifier for a later session once the previous one has stopped, so several
// finalized rows can share it. Updating every match would rewrite the history of
// the earlier sessions with the newest packet's counters. Selecting the row with
// the highest ID (snowflake IDs are time ordered) and updating it by primary key
// keeps earlier history intact, while duplicate stops of the same session still
// converge on the same row.
func (r *GormAccountingRepository) UpdateStop(ctx context.Context, sessionId string, accounting *domain.RadiusAccounting) error {
	var target domain.RadiusAccounting
	err := r.db.WithContext(ctx).
		Select("id").
		Where("acct_session_id = ?", sessionId).
		Order("id DESC").
		Take(&target).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return repository.ErrAccountingNotFound
		}
		return err
	}

	param := map[string]interface{}{
		"acct_stop_time":      time.Now(),
		"acct_input_total":    accounting.AcctInputTotal,
		"acct_output_total":   accounting.AcctOutputTotal,
		"acct_input_packets":  accounting.AcctInputPackets,
		"acct_output_packets": accounting.AcctOutputPackets,
		"acct_session_time":   accounting.AcctSessionTime,
	}

	return r.db.WithContext(ctx).
		Model(&domain.RadiusAccounting{}).
		Where("id = ?", target.ID).
		Updates(param).Error
}

// UpsertStop creates a terminal accounting row when none exists for sessionId
// and updates the newest one when it does.
//
// The accounting pipeline processes Accounting-Start asynchronously after the
// response is sent, so an Accounting-Stop can arrive before the Start task has
// persisted the accounting row. Rather than dropping the Stop with "no records
// found", UpsertStop materializes the final record from the Stop packet itself,
// so the session is still accounted for and no traffic is lost. If the Start
// task later runs, its Create is idempotent on acct_session_id and the existing
// row is left untouched; the Stop's counters remain the authoritative terminal
// values.
func (r *GormAccountingRepository) UpsertStop(ctx context.Context, sessionId string, accounting *domain.RadiusAccounting) error {
	// Check whether an accounting row already exists for this session.
	var existing domain.RadiusAccounting
	err := r.db.WithContext(ctx).
		Select("id").
		Where("acct_session_id = ?", sessionId).
		Order("id DESC").
		Take(&existing).Error
	if err == nil {
		// Existing row: update the newest one in place, same as UpdateStop.
		param := map[string]interface{}{
			"acct_stop_time":      time.Now(),
			"acct_input_total":    accounting.AcctInputTotal,
			"acct_output_total":   accounting.AcctOutputTotal,
			"acct_input_packets":  accounting.AcctInputPackets,
			"acct_output_packets": accounting.AcctOutputPackets,
			"acct_session_time":   accounting.AcctSessionTime,
		}
		return r.db.WithContext(ctx).
			Model(&domain.RadiusAccounting{}).
			Where("id = ?", existing.ID).
			Updates(param).Error
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// No existing row: insert a terminal record built from the Stop packet.
	// AcctStopTime is stamped now so the row is immediately queryable as a
	// completed session by reporting and reconciliation jobs. StartPending tells
	// a late Accounting-Start to complete this row rather than insert a duplicate.
	if accounting.ID == 0 {
		accounting.ID = common.UUIDint64()
	}
	accounting.AcctStopTime = time.Now()
	accounting.StartPending = true
	return r.db.WithContext(ctx).Create(accounting).Error
}

// FindBySessionId returns the newest accounting row for a session ID, if any.
// Returns ErrAccountingNotFound if no accounting row exists for the session.
func (r *GormAccountingRepository) FindBySessionId(ctx context.Context, sessionId string) (*domain.RadiusAccounting, error) {
	var row domain.RadiusAccounting
	err := r.db.WithContext(ctx).
		Where("acct_session_id = ?", sessionId).
		Order("id DESC").
		Take(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, repository.ErrAccountingNotFound
		}
		return nil, err
	}
	return &row, nil
}
