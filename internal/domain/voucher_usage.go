package domain

import "time"

// VoucherUsageSession stores the last absolute RADIUS accounting counters for
// one prepaid voucher session. Absolute counters make retransmitted interim
// packets idempotent; finalized rows also preserve session-ID reuse history.
type VoucherUsageSession struct {
	ID            int64      `json:"id,string" gorm:"primaryKey"`
	TenantID      int64      `json:"-" gorm:"not null;default:1;index:idx_voucher_usage_lookup,priority:1"`
	VoucherID     int64      `json:"voucher_id,string" gorm:"not null;index:idx_voucher_usage_lookup,priority:2"`
	Username      string     `json:"username" gorm:"size:128;not null;index"`
	AcctSessionID string     `json:"acct_session_id" gorm:"size:128;not null;index:idx_voucher_usage_lookup,priority:3"`
	NASAddress    string     `json:"nas_address" gorm:"size:64"`
	InputBytes    int64      `json:"input_bytes,string"`
	OutputBytes   int64      `json:"output_bytes,string"`
	SessionTime   int        `json:"session_time"`
	StartPending  bool       `json:"-" gorm:"not null;default:false"`
	EndedAt       *time.Time `json:"ended_at,omitempty" gorm:"index"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// TableName returns the persistent per-session voucher accounting table.
func (VoucherUsageSession) TableName() string { return "isp_voucher_usage_session" }
