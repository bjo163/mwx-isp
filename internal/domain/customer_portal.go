package domain

import "time"

// CustomerAuthChallenge stores only a keyed OTP digest. The OTP itself is
// delivered out-of-band and is never stored or returned by the API.
type CustomerAuthChallenge struct {
	ID          int64      `json:"id,string" gorm:"primaryKey"`
	TenantID    int64      `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_customer_auth_challenge_ref,priority:1"`
	CustomerID  int64      `json:"customer_id,string" gorm:"not null;index"`
	ChallengeID string     `json:"challenge_id" gorm:"size:64;not null;uniqueIndex:udx_customer_auth_challenge_ref,priority:2"`
	PhoneHash   string     `json:"-" gorm:"size:64;not null;index"`
	IPHash      string     `json:"-" gorm:"size:64;not null;index"`
	CodeHash    string     `json:"-" gorm:"size:64;not null"`
	Attempts    int        `json:"-" gorm:"not null;default:0"`
	ExpiresAt   time.Time  `json:"expires_at" gorm:"index;not null"`
	ConsumedAt  *time.Time `json:"-" gorm:"index"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TableName returns the short-lived customer OTP challenge table.
func (CustomerAuthChallenge) TableName() string { return "customer_auth_challenge" }

// CustomerPortalSession stores a keyed token digest, never the bearer token.
type CustomerPortalSession struct {
	ID         int64      `json:"id,string" gorm:"primaryKey"`
	TenantID   int64      `json:"-" gorm:"not null;default:1;index;uniqueIndex:udx_customer_portal_session_token,priority:1"`
	CustomerID int64      `json:"customer_id,string" gorm:"not null;index"`
	TokenHash  string     `json:"-" gorm:"size:64;not null;uniqueIndex:udx_customer_portal_session_token,priority:2"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"index;not null"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
	RevokedAt  *time.Time `json:"-" gorm:"index"`
	CreatedAt  time.Time  `json:"created_at"`
}

// TableName returns the customer self-care session table.
func (CustomerPortalSession) TableName() string { return "customer_portal_session" }
