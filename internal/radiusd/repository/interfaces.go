// Package repository defines storage interfaces used by the RADIUS pipeline.
package repository

import (
	"context"
	"errors"

	"github.com/bjo163/mwx-isp/internal/domain"
)

// ErrAmbiguousNASIP means more than one tenant configured the same RADIUS
// client source address, so the server cannot safely select a shared secret.
var ErrAmbiguousNASIP = errors.New("ambiguous NAS source IP")

// ErrAccountingNotFound means no finalized accounting row exists yet for the
// given Acct-Session-Id. This is the expected result when an Accounting-Stop
// arrives before the matching Accounting-Start has been persisted (the Start
// task runs asynchronously after the Access-Accept/Accounting-Response is sent,
// so a retransmitted or reordered Stop can legitimately win the race).
var ErrAccountingNotFound = errors.New("no accounting record found for acct_session_id")

// UserRepository defines user data access operations
type UserRepository interface {
	// GetByUsername finds a user by username
	GetByUsername(ctx context.Context, username string) (*domain.RadiusUser, error)

	// GetByMacAddr finds a user by MAC address
	GetByMacAddr(ctx context.Context, macAddr string) (*domain.RadiusUser, error)

	// UpdateMacAddr updates the user's MAC address
	UpdateMacAddr(ctx context.Context, username, macAddr string) error

	// UpdateVlanId updates the user's VLAN ID
	UpdateVlanId(ctx context.Context, username string, vlanId1, vlanId2 int) error

	// UpdateLastOnline updates the last online time
	UpdateLastOnline(ctx context.Context, username string) error

	// UpdateField updates a specified user field
	UpdateField(ctx context.Context, username string, field string, value interface{}) error
}

// SessionRepository manages online sessions
type SessionRepository interface {
	// Create inserts an online session idempotently keyed on Acct-Session-Id.
	// It returns created=true when a new row was inserted, and created=false
	// when a row with the same Acct-Session-Id already exists (for example a
	// retransmitted Accounting-Start). Duplicate inserts are a no-op rather
	// than an error so accounting stays idempotent under UDP retransmission.
	Create(ctx context.Context, session *domain.RadiusOnline) (created bool, err error)

	// Update updates session data
	Update(ctx context.Context, session *domain.RadiusOnline) error

	// Delete deletes a session
	Delete(ctx context.Context, sessionId string) error

	// GetBySessionId finds a session by its ID
	GetBySessionId(ctx context.Context, sessionId string) (*domain.RadiusOnline, error)

	// CountByUsername counts online sessions per user
	CountByUsername(ctx context.Context, username string) (int, error)

	// Exists checks whether the session exists
	Exists(ctx context.Context, sessionId string) (bool, error)

	// BatchDelete deletes sessions in bulk
	BatchDelete(ctx context.Context, ids []string) error

	// BatchDeleteByNas deletes sessions by NAS
	BatchDeleteByNas(ctx context.Context, nasAddr, nasId string) error
}

// AccountingRepository defines accounting record operations
type AccountingRepository interface {
	// Create Create accounting record
	Create(ctx context.Context, accounting *domain.RadiusAccounting) error

	// UpdateStop updates stop time and traffic counters for the newest
	// accounting row of sessionId. Returns ErrAccountingNotFound when no
	// accounting row exists yet for the session (the Start task has not been
	// persisted), so callers can fall back to UpsertStop.
	UpdateStop(ctx context.Context, sessionId string, accounting *domain.RadiusAccounting) error

	// UpsertStop creates a terminal accounting row when none exists yet and
	// updates the newest one when it does. It is the stop-side counterpart of
	// the asynchronous Start handler: if a Stop wins the race and arrives
	// before the Start task has persisted the accounting row, UpsertStop
	// materializes the final record from the Stop packet alone so the session
	// is still accounted for and no traffic is lost.
	// FindBySessionId returns the newest accounting row for a session ID, if any,
	// including its StartPending marker.
	// Returns ErrAccountingNotFound if no accounting row exists for the session.
	FindBySessionId(ctx context.Context, sessionId string) (*domain.RadiusAccounting, error)

	UpsertStop(ctx context.Context, sessionId string, accounting *domain.RadiusAccounting) error
}

// NasRepository manages NAS devices
type NasRepository interface {
	// GetByIP finds a NAS by IP
	GetByIP(ctx context.Context, ip string) (*domain.NetNas, error)

	// GetByIdentifier finds a NAS by identifier
	GetByIdentifier(ctx context.Context, identifier string) (*domain.NetNas, error)

	// GetByIPOrIdentifier finds a NAS by IP or identifier
	GetByIPOrIdentifier(ctx context.Context, ip, identifier string) (*domain.NetNas, error)
}
