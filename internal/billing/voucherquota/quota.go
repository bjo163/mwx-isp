// Package voucherquota turns absolute RADIUS accounting counters into an
// idempotent aggregate byte quota for prepaid hotspot vouchers.
package voucherquota

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	StatusStart   = 1
	StatusStop    = 2
	StatusInterim = 3
)

// Result describes quota state after one accounting packet.
type Result struct {
	VoucherID      int64
	RadiusUserID   int64
	Username       string
	UsedBytes      int64
	QuotaBytes     int64
	Exhausted      bool
	NewlyExhausted bool
}

// ApplyAccounting applies one RADIUS packet using absolute cumulative counters.
// Duplicate packets contribute zero; older Acct-Session-Time packets are ignored.
// A Stop-before-Start row remains marked so a late Start cannot create a second
// usage row, and a later Start after an ordinary Stop creates a fresh row even
// when a NAS reuses Acct-Session-Id.
func ApplyAccounting(ctx context.Context, db *gorm.DB, username, sessionID string, status int, inputBytes, outputBytes int64, sessionTime int, now time.Time) (Result, error) {
	var result Result
	tenantID, ok := tenancy.TenantID(ctx)
	if !ok {
		return result, errors.New("voucher quota requires an authenticated tenant context")
	}
	if db == nil || username == "" || sessionID == "" {
		return result, nil
	}
	if inputBytes < 0 || outputBytes < 0 || sessionTime < 0 {
		return result, nil
	}
	if now.IsZero() {
		now = time.Now()
	}

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var user domain.RadiusUser
		if err := tx.Where("username = ?", username).First(&user).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		var voucher domain.HotspotVoucher
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("radius_user_id = ?", user.ID).First(&voucher).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return err
		}
		result.VoucherID, result.RadiusUserID, result.Username = voucher.ID, user.ID, username
		result.UsedBytes, result.QuotaBytes = voucher.UsedBytes, voucher.QuotaBytes
		if voucher.QuotaBytes <= 0 {
			return nil
		}
		if voucher.Status == "exhausted" {
			result.Exhausted = true
			return nil
		}
		if voucher.Status != "active" && voucher.Status != "used" {
			return nil
		}

		var usage domain.VoucherUsageSession
		usageErr := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("voucher_id = ? AND acct_session_id = ?", voucher.ID, sessionID).
			Order("id DESC").First(&usage).Error
		if usageErr != nil && !errors.Is(usageErr, gorm.ErrRecordNotFound) {
			return usageErr
		}
		found := usageErr == nil
		var delta int64

		switch status {
		case StatusStart:
			if found && (usage.EndedAt == nil || usage.StartPending) {
				// Active Start retransmission, or a Start that arrives after a
				// Stop already materialized the terminal row.
				return nil
			}
			usage = domain.VoucherUsageSession{
				TenantID: tenantID, VoucherID: voucher.ID, Username: username,
				AcctSessionID: sessionID, InputBytes: inputBytes, OutputBytes: outputBytes,
				SessionTime: sessionTime, CreatedAt: now, UpdatedAt: now,
			}
			delta = saturatingAdd(inputBytes, outputBytes)
			if err := tx.Create(&usage).Error; err != nil {
				return err
			}

		case StatusInterim, StatusStop:
			if found && usage.EndedAt != nil {
				// A duplicate Stop or a late Interim after a terminal packet.
				return nil
			}
			if !found {
				usage = domain.VoucherUsageSession{
					TenantID: tenantID, VoucherID: voucher.ID, Username: username,
					AcctSessionID: sessionID, InputBytes: 0, OutputBytes: 0,
					CreatedAt: now, UpdatedAt: now,
				}
			}
			// Never let an out-of-order interim overwrite newer counters. Stop
			// still closes the tracker but contributes no negative/duplicate usage.
			if sessionTime >= usage.SessionTime {
				deltaInput := counterDelta(usage.InputBytes, inputBytes, sessionTime, usage.SessionTime)
				deltaOutput := counterDelta(usage.OutputBytes, outputBytes, sessionTime, usage.SessionTime)
				delta = saturatingAdd(deltaInput, deltaOutput)
				usage.InputBytes = maxCounter(usage.InputBytes, inputBytes, sessionTime, usage.SessionTime)
				usage.OutputBytes = maxCounter(usage.OutputBytes, outputBytes, sessionTime, usage.SessionTime)
				usage.SessionTime = sessionTime
			}
			usage.UpdatedAt = now
			if status == StatusStop {
				usage.EndedAt = &now
				if !found {
					// The asynchronous Start handler may still arrive later.
					usage.StartPending = true
				}
			}
			if found {
				if err := tx.Save(&usage).Error; err != nil {
					return err
				}
			} else if err := tx.Create(&usage).Error; err != nil {
				return err
			}
		default:
			return nil
		}

		if delta > 0 {
			if voucher.UsedBytes > math.MaxInt64-delta {
				voucher.UsedBytes = math.MaxInt64
			} else {
				voucher.UsedBytes += delta
			}
			updates := map[string]any{"used_bytes": voucher.UsedBytes}
			if voucher.UsedBytes >= voucher.QuotaBytes {
				voucher.Status = "exhausted"
				updates["status"] = "exhausted"
				if err := tx.Model(&domain.RadiusUser{}).Where("id = ?", user.ID).Update("status", "disabled").Error; err != nil {
					return err
				}
				result.Exhausted = true
				result.NewlyExhausted = true
			}
			if err := tx.Model(&voucher).Updates(updates).Error; err != nil {
				return err
			}
		}
		result.UsedBytes = voucher.UsedBytes
		result.QuotaBytes = voucher.QuotaBytes
		if voucher.Status == "exhausted" {
			result.Exhausted = true
		}
		return nil
	})
	return result, err
}

func counterDelta(previous, current int64, currentSessionTime, previousSessionTime int) int64 {
	if current >= previous {
		return current - previous
	}
	if currentSessionTime > previousSessionTime {
		// NAS counter reset during this session; count the post-reset total.
		return current
	}
	return 0
}

func maxCounter(previous, current int64, currentSessionTime, previousSessionTime int) int64 {
	if current >= previous || currentSessionTime > previousSessionTime {
		return current
	}
	return previous
}

func saturatingAdd(a, b int64) int64 {
	if a > math.MaxInt64-b {
		return math.MaxInt64
	}
	return a + b
}
