package radiusd

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting"
	accthandlers "github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting/handlers"
	vendorparserspkg "github.com/bjo163/mwx-isp/internal/radiusd/plugins/vendorparsers"
	repogorm "github.com/bjo163/mwx-isp/internal/radiusd/repository/gorm"
	"github.com/bjo163/mwx-isp/internal/tenancy"
)

func retransmissionDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:retransmission-"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(
		&domain.RadiusOnline{},
		&domain.RadiusAccounting{},
		&domain.RadiusTrafficSample{},
	))
	return db
}

// accountingPacket builds a real Accounting-Request with a fixed Identifier and
// Request Authenticator. Two calls return byte-identical packets, which is what
// a NAS retransmission looks like on the wire.
func accountingPacket(t *testing.T, status rfc2866.AcctStatusType, sessionID, username string,
	inputOctets, outputOctets, sessionTime uint32) *radius.Packet {
	t.Helper()
	packet := radius.New(radius.CodeAccountingRequest, []byte("retransmission-fixture-secret"))
	packet.Identifier = 42
	packet.Authenticator = [16]byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88,
		0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xF0, 0x0F}
	require.NoError(t, rfc2866.AcctStatusType_Set(packet, status))
	require.NoError(t, rfc2866.AcctSessionID_SetString(packet, sessionID))
	require.NoError(t, rfc2865.UserName_SetString(packet, username))
	require.NoError(t, rfc2866.AcctInputOctets_Set(packet, rfc2866.AcctInputOctets(inputOctets)))
	require.NoError(t, rfc2866.AcctOutputOctets_Set(packet, rfc2866.AcctOutputOctets(outputOctets)))
	require.NoError(t, rfc2866.AcctSessionTime_Set(packet, rfc2866.AcctSessionTime(sessionTime)))
	return packet
}

func accountingFixture(t *testing.T, tenantID int64, nas *domain.NetNas,
	packet *radius.Packet, username string, status rfc2866.AcctStatusType) *accounting.AccountingContext {
	t.Helper()
	return &accounting.AccountingContext{
		Context:    tenancy.WithTenantID(context.Background(), tenantID),
		Request:    &radius.Request{Packet: packet},
		VendorReq:  &vendorparserspkg.VendorRequest{},
		Username:   username,
		NAS:        nas,
		NASIP:      nas.Ipaddr,
		StatusType: int(status),
	}
}

// TestRetransmittedAccountingPacketsHaveNoDoubleEffect replays byte-identical
// Accounting-Start, Interim-Update and Stop packets per tenant. The tenant is
// the isolation unit: two tenants may use the same Acct-Session-Id and username
// and must still get their own rows, while a replay inside one tenant must not
// duplicate the online row, the accounting row or the final counters.
func TestRetransmittedAccountingPacketsHaveNoDoubleEffect(t *testing.T) {
	db := retransmissionDB(t)
	sessionRepo := repogorm.NewGormSessionRepository(db)
	accountingRepo := repogorm.NewGormAccountingRepository(db)
	startHandler := accthandlers.NewStartHandler(sessionRepo, accountingRepo)
	updateHandler := accthandlers.NewUpdateHandler(sessionRepo)
	stopHandler := accthandlers.NewStopHandler(sessionRepo, accountingRepo)

	nasA := &domain.NetNas{ID: 701, TenantID: 1, Name: "nas-a", Ipaddr: "198.51.100.1", Secret: "a"}
	nasB := &domain.NetNas{ID: 702, TenantID: 2, Name: "nas-b", Ipaddr: "198.51.100.2", Secret: "b"}
	const sessionID = "retransmitted-session"
	const username = "shared-username"

	start := func(nas *domain.NetNas) *accounting.AccountingContext {
		return accountingFixture(t, nas.TenantID, nas,
			accountingPacket(t, rfc2866.AcctStatusType_Value_Start, sessionID, username, 0, 0, 0),
			username, rfc2866.AcctStatusType_Value_Start)
	}

	first, second := start(nasA), start(nasA)
	firstBytes, err := first.Request.Packet.Encode()
	require.NoError(t, err)
	secondBytes, err := second.Request.Packet.Encode()
	require.NoError(t, err)
	require.Equal(t, firstBytes, secondBytes, "the fixture must replay identical packet bytes")

	require.NoError(t, startHandler.Handle(first))
	require.NoError(t, startHandler.Handle(second))
	require.NoError(t, startHandler.Handle(start(nasB)))

	for _, tenantID := range []int64{1, 2} {
		var online, accountingRows int64
		require.NoError(t, db.Model(&domain.RadiusOnline{}).Where("tenant_id = ?", tenantID).Count(&online).Error)
		require.NoError(t, db.Model(&domain.RadiusAccounting{}).Where("tenant_id = ?", tenantID).Count(&accountingRows).Error)
		assert.Equalf(t, int64(1), online, "tenant %d must keep exactly one online row", tenantID)
		assert.Equalf(t, int64(1), accountingRows, "tenant %d must keep exactly one accounting row", tenantID)
	}

	// Interim-Update replays refresh the same row instead of appending history.
	interim := func(nas *domain.NetNas) *accounting.AccountingContext {
		return accountingFixture(t, nas.TenantID, nas,
			accountingPacket(t, rfc2866.AcctStatusType_Value_InterimUpdate, sessionID, username, 2048, 4096, 60),
			username, rfc2866.AcctStatusType_Value_InterimUpdate)
	}
	require.NoError(t, updateHandler.Handle(interim(nasA)))
	require.NoError(t, updateHandler.Handle(interim(nasA)))
	var accountingRows int64
	require.NoError(t, db.Model(&domain.RadiusAccounting{}).Where("tenant_id = ?", 1).Count(&accountingRows).Error)
	assert.Equal(t, int64(1), accountingRows, "interim replay must not append a billing record")
	var online domain.RadiusOnline
	require.NoError(t, db.Where("tenant_id = ? AND acct_session_id = ?", 1, sessionID).First(&online).Error)
	assert.Equal(t, int64(2048), online.AcctInputTotal)
	assert.Equal(t, int64(4096), online.AcctOutputTotal)
	assert.Equal(t, 60, online.AcctSessionTime)

	// Stop replays finalize the same rows; the session leaves the online table
	// once and the accounting row keeps one set of final counters.
	stop := func(nas *domain.NetNas) *accounting.AccountingContext {
		return accountingFixture(t, nas.TenantID, nas,
			accountingPacket(t, rfc2866.AcctStatusType_Value_Stop, sessionID, username, 8192, 16384, 120),
			username, rfc2866.AcctStatusType_Value_Stop)
	}
	require.NoError(t, stopHandler.Handle(stop(nasA)))
	require.NoError(t, stopHandler.Handle(stop(nasA)))

	var onlineA int64
	require.NoError(t, db.Model(&domain.RadiusOnline{}).Where("tenant_id = ?", 1).Count(&onlineA).Error)
	assert.Zero(t, onlineA, "a stop replay must not resurrect or duplicate the online row")
	var final domain.RadiusAccounting
	require.NoError(t, db.Where("tenant_id = ? AND acct_session_id = ?", 1, sessionID).First(&final).Error)
	assert.Equal(t, int64(8192), final.AcctInputTotal)
	assert.Equal(t, int64(16384), final.AcctOutputTotal)
	assert.Equal(t, 120, final.AcctSessionTime)
	require.NoError(t, db.Model(&domain.RadiusAccounting{}).Where("tenant_id = ?", 1).Count(&accountingRows).Error)
	assert.Equal(t, int64(1), accountingRows)

	// Tenant B's session with the same session ID is untouched by tenant A's
	// stop and still online.
	var tenantB domain.RadiusOnline
	require.NoError(t, db.Where("tenant_id = ? AND acct_session_id = ?", 2, sessionID).First(&tenantB).Error)
	assert.Equal(t, int64(2), tenantB.TenantID)
}

// TestRetransmittedAccessRequestRateLimitIsTenantScoped proves the in-memory
// duplicate window is keyed per tenant: a retransmission for tenant A must not
// throttle the same username on tenant B, and the duplicate inside tenant A is
// rejected without touching tenant B's state.
func TestRetransmittedAccessRequestRateLimitIsTenantScoped(t *testing.T) {
	service := &RadiusService{authRate: newAuthRateLimiter(4)}
	require.NoError(t, service.CheckAuthRateLimitForTenant(1, "same-username"))
	err := service.CheckAuthRateLimitForTenant(1, "same-username")
	require.Error(t, err, "a retransmitted Access-Request inside the window is throttled")
	require.NoError(t, service.CheckAuthRateLimitForTenant(2, "same-username"),
		"another tenant with an identical username must keep its own window")

	service.ReleaseAuthRateLimitForTenant(1, "same-username")
	require.NoError(t, service.CheckAuthRateLimitForTenant(1, "same-username"),
		"releasing the window must allow an immediate retry")
}
