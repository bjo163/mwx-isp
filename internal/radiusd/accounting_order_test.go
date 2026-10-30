package radiusd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"layeh.com/radius/rfc2866"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting"
	accthandlers "github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting/handlers"
	repogorm "github.com/bjo163/mwx-isp/internal/radiusd/repository/gorm"
)

// accountingOrderHarness wires the real accounting handlers to GORM
// repositories so ordering tests exercise the same insert/update/delete paths
// the UDP accounting service uses.
type accountingOrderHarness struct {
	db      *gorm.DB
	start   *accthandlers.StartHandler
	interim *accthandlers.UpdateHandler
	stop    *accthandlers.StopHandler
}

func newAccountingOrderHarness(t *testing.T) *accountingOrderHarness {
	t.Helper()
	db := retransmissionDB(t)
	sessionRepo := repogorm.NewGormSessionRepository(db)
	accountingRepo := repogorm.NewGormAccountingRepository(db)
	return &accountingOrderHarness{
		db:      db,
		start:   accthandlers.NewStartHandler(sessionRepo, accountingRepo),
		interim: accthandlers.NewUpdateHandler(sessionRepo),
		stop:    accthandlers.NewStopHandler(sessionRepo, accountingRepo),
	}
}

func (h *accountingOrderHarness) send(t *testing.T, nas *domain.NetNas, status rfc2866.AcctStatusType,
	sessionID, username string, in, out, secs uint32) error {
	t.Helper()
	packet := accountingPacket(t, status, sessionID, username, in, out, secs)
	ctx := accountingFixture(t, nas.TenantID, nas, packet, username, status)
	return h.dispatch(t, ctx)
}

func (h *accountingOrderHarness) dispatch(t *testing.T, ctx *accounting.AccountingContext) error {
	t.Helper()
	switch rfc2866.AcctStatusType(ctx.StatusType) { //nolint:gosec // G115: the fixture only ever sets the three start/interim/stop values.
	case rfc2866.AcctStatusType_Value_Start:
		return h.start.Handle(ctx)
	case rfc2866.AcctStatusType_Value_InterimUpdate:
		return h.interim.Handle(ctx)
	case rfc2866.AcctStatusType_Value_Stop:
		return h.stop.Handle(ctx)
	}
	t.Fatalf("unhandled accounting status type %d", ctx.StatusType)
	return nil
}

// history returns the accounting ledger rows for one tenant/session ordered by
// insertion (Acct-Session-Id is intentionally not unique, see ENG-012).
func (h *accountingOrderHarness) history(t *testing.T, tenantID int64, sessionID string) []domain.RadiusAccounting {
	t.Helper()
	var rows []domain.RadiusAccounting
	require.NoError(t, h.db.
		Where("tenant_id = ? AND acct_session_id = ?", tenantID, sessionID).
		Order("id ASC").Find(&rows).Error)
	return rows
}

func (h *accountingOrderHarness) onlineCount(t *testing.T, tenantID int64, sessionID string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, h.db.Model(&domain.RadiusOnline{}).
		Where("tenant_id = ? AND acct_session_id = ?", tenantID, sessionID).
		Count(&count).Error)
	return count
}

func requireFinalCounters(t *testing.T, row domain.RadiusAccounting, in, out int64, secs int) {
	t.Helper()
	assert.Equal(t, in, row.AcctInputTotal, "acct_input_total")
	assert.Equal(t, out, row.AcctOutputTotal, "acct_output_total")
	assert.Equal(t, secs, row.AcctSessionTime, "acct_session_time")
	assert.False(t, row.AcctStopTime.IsZero(), "a finalized row must carry acct_stop_time")
}

// TestAccountingOutOfOrderStartInterimStopConverge replays the packets of one
// session with duplicates and a delayed interim update. Whatever order the
// copies are processed in, the session must end with no online row and exactly
// one accounting row carrying the stop counters; a stale interim must not
// regress finalized history.
func TestAccountingOutOfOrderStartInterimStopConverge(t *testing.T) {
	h := newAccountingOrderHarness(t)
	nas := &domain.NetNas{ID: 801, TenantID: 1, Name: "nas-order-a", Ipaddr: "203.0.113.1", Secret: "a"}
	const sessionID = "ordered-session"
	const username = "ordered-user"

	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	// Retransmitted start while the session is live.
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 1024, 2048, 30))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 2048, 4096, 60))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 8192, 16384, 120))
	// Retransmitted stop after the session was finalized.
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 8192, 16384, 120))

	// Delayed interim update that reached the handler after the stop.
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 4096, 8192, 90))

	assert.Zero(t, h.onlineCount(t, nas.TenantID, sessionID), "a stopped session must not stay online")
	rows := h.history(t, nas.TenantID, sessionID)
	require.Len(t, rows, 1, "permuted copies must not append extra billing rows")
	requireFinalCounters(t, rows[0], 8192, 16384, 120)
}

// TestAccountingSessionIDReuseKeepsFinalizedHistory covers a NAS that reuses an
// Acct-Session-Id for a later session. The second stop must finalize only the
// newest row so the first session's counters survive as billing history.
func TestAccountingSessionIDReuseKeepsFinalizedHistory(t *testing.T) {
	h := newAccountingOrderHarness(t)
	nas := &domain.NetNas{ID: 802, TenantID: 1, Name: "nas-order-b", Ipaddr: "203.0.113.2", Secret: "b"}
	const sessionID = "reused-session"
	const username = "reused-user"

	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 1000, 2000, 30))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 4000, 8000, 60))

	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 500, 1500, 20))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 2000, 6000, 45))
	// Retransmitted stop of the reused session must stay on the newest row.
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 2000, 6000, 45))

	assert.Zero(t, h.onlineCount(t, nas.TenantID, sessionID))
	rows := h.history(t, nas.TenantID, sessionID)
	require.Len(t, rows, 2, "each reused session keeps its own history row")
	requireFinalCounters(t, rows[0], 4000, 8000, 60)
	requireFinalCounters(t, rows[1], 2000, 6000, 45)
}

// TestAccountingSessionIDReuseIsTenantScoped repeats the session-id reuse across
// two tenants: finalizing tenant 2's reused session must not touch tenant 1's
// rows even though both use the same Acct-Session-Id and username.
func TestAccountingSessionIDReuseIsTenantScoped(t *testing.T) {
	h := newAccountingOrderHarness(t)
	nasA := &domain.NetNas{ID: 803, TenantID: 1, Name: "nas-order-c", Ipaddr: "203.0.113.3", Secret: "c"}
	nasB := &domain.NetNas{ID: 804, TenantID: 2, Name: "nas-order-d", Ipaddr: "203.0.113.4", Secret: "d"}
	const sessionID = "shared-reused-session"
	const username = "shared-reused-user"

	require.NoError(t, h.send(t, nasA, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nasA, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 4000, 8000, 60))
	require.NoError(t, h.send(t, nasA, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nasA, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 2000, 6000, 45))

	require.NoError(t, h.send(t, nasB, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0))
	require.NoError(t, h.send(t, nasB, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 6000, 7000, 80))
	require.NoError(t, h.send(t, nasB, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 7000, 9000, 90))

	rowsA := h.history(t, 1, sessionID)
	require.Len(t, rowsA, 2)
	requireFinalCounters(t, rowsA[0], 4000, 8000, 60)
	requireFinalCounters(t, rowsA[1], 2000, 6000, 45)

	rowsB := h.history(t, 2, sessionID)
	require.Len(t, rowsB, 1)
	requireFinalCounters(t, rowsB[0], 7000, 9000, 90)
}

// TestAccountingEventsWithoutStartDoNotInventHistory is the negative case: a
// stop or interim that arrives before any start (delayed start still in flight)
// must not fabricate an accounting row or an online row. The ledger stays empty
// until a start actually creates the session.
func TestAccountingEventsWithoutStartDoNotInventHistory(t *testing.T) {
	h := newAccountingOrderHarness(t)
	nas := &domain.NetNas{ID: 805, TenantID: 1, Name: "nas-order-e", Ipaddr: "203.0.113.5", Secret: "e"}
	const sessionID = "unstarted-session"
	const username = "unstarted-user"

	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 1024, 2048, 30))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 2048, 4096, 60))
	require.NoError(t, h.send(t, nas, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 2048, 4096, 60))

	assert.Empty(t, h.history(t, nas.TenantID, sessionID), "accounting history must not be invented without a start")
	assert.Zero(t, h.onlineCount(t, nas.TenantID, sessionID))
}
