// Package gorm provides GORM-backed implementations of repository interfaces.
package gorm

import (
	"context"
	"errors"
	"fmt"
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
			return fmt.Errorf("no records found with acct_session_id = %v", sessionId)
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
