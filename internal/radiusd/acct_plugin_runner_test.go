package radiusd

import (
	"context"
	"net"
	"sync/atomic"
	"testing"

	"github.com/bjo163/mwx-isp/internal/app"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/accounting"
	vendorparsers "github.com/bjo163/mwx-isp/internal/radiusd/plugins/vendorparsers"
	"github.com/bjo163/mwx-isp/internal/radiusd/registry"
	"github.com/bjo163/mwx-isp/pkg/metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2866"
)

type mockAccountingHandler struct {
	name       string
	calls      *int32
	canHandle  bool
	expectType int
	err        error
}

func (h *mockAccountingHandler) Name() string { return h.name }
func (h *mockAccountingHandler) CanHandle(ctx *accounting.AccountingContext) bool {
	if h.expectType != 0 {
		return ctx.StatusType == h.expectType
	}
	return h.canHandle
}
func (h *mockAccountingHandler) Handle(ctx *accounting.AccountingContext) error {
	atomic.AddInt32(h.calls, 1)
	return h.err
}

func TestHandleAccountingWithPluginsDispatchesToHandler(t *testing.T) {
	registry.ResetForTest()
	t.Cleanup(registry.ResetForTest)

	var calls int32
	registry.RegisterAccountingHandler(&mockAccountingHandler{
		name:      "handler",
		calls:     &calls,
		canHandle: true,
	})

	acctSvc := &AcctService{RadiusService: &RadiusService{}}
	packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	_ = rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Start)
	req := &radius.Request{
		Packet:     packet,
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1813},
	}

	vendorReq := &vendorparsers.VendorRequest{}
	nas := &domain.NetNas{Identifier: "NAS-1"}

	err := acctSvc.HandleAccountingWithPlugins(
		context.Background(),
		req,
		vendorReq,
		"alice",
		nas,
		"10.0.0.1",
	)
	if err != nil {
		t.Fatalf("HandleAccountingWithPlugins returned error: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected handler to be called once, got %d", got)
	}
}

func TestHandleAccountingWithPluginsNoHandler(t *testing.T) {
	registry.ResetForTest()
	t.Cleanup(registry.ResetForTest)

	acctSvc := &AcctService{RadiusService: &RadiusService{}}
	packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	_ = rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_Stop)
	req := &radius.Request{
		Packet:     packet,
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1813},
	}

	err := acctSvc.HandleAccountingWithPlugins(
		context.Background(),
		req,
		&vendorparsers.VendorRequest{},
		"alice",
		&domain.NetNas{},
		"10.0.0.1",
	)
	if err == nil {
		t.Fatalf("expected error when no handlers registered")
	}
}

func TestHandleAccountingWithPluginsParsesInterimUpdateStatusType(t *testing.T) {
	registry.ResetForTest()
	t.Cleanup(registry.ResetForTest)

	var calls int32
	registry.RegisterAccountingHandler(&mockAccountingHandler{
		name:       "interim-update-handler",
		calls:      &calls,
		expectType: int(rfc2866.AcctStatusType_Value_InterimUpdate),
	})

	acctSvc := &AcctService{RadiusService: &RadiusService{}}
	packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
	_ = rfc2866.AcctStatusType_Set(packet, rfc2866.AcctStatusType_Value_InterimUpdate)
	req := &radius.Request{
		Packet:     packet,
		RemoteAddr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1813},
	}

	err := acctSvc.HandleAccountingWithPlugins(
		context.Background(),
		req,
		&vendorparsers.VendorRequest{},
		"alice",
		&domain.NetNas{},
		"10.0.0.1",
	)
	if err != nil {
		t.Fatalf("expected Interim-Update handler to be called, got error: %v", err)
	}

	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("expected handler to be called once, got %d", got)
	}
}

// TestHandleAccountingStartAndStopCounters pins that the radus_online and
// radus_offline counters track successfully handled Start and Stop accounting,
// so an operator scraping /metrics sees session-event volumes instead of two
// permanently zero series.
func TestHandleAccountingStartAndStopCounters(t *testing.T) {
	registry.ResetForTest()
	t.Cleanup(registry.ResetForTest)
	require.NoError(t, metrics.InitMetrics(""))

	var calls int32
	registry.RegisterAccountingHandler(&mockAccountingHandler{
		name:      "handler",
		calls:     &calls,
		canHandle: true,
	})

	acctSvc := &AcctService{RadiusService: &RadiusService{}}
	account := func(statusType rfc2866.AcctStatusType) {
		t.Helper()
		packet := radius.New(radius.CodeAccountingRequest, []byte("secret"))
		_ = rfc2866.AcctStatusType_Set(packet, statusType)
		req := &radius.Request{
			Packet:     packet,
			RemoteAddr: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 1813},
		}
		require.NoError(t, acctSvc.HandleAccountingWithPlugins(
			context.Background(), req, &vendorparsers.VendorRequest{}, "alice", &domain.NetNas{}, "10.0.0.1",
		))
	}

	onlineBefore := app.GetRadiusMetrics(app.MetricsRadiusOline)
	offlineBefore := app.GetRadiusMetrics(app.MetricsRadiusOffline)

	account(rfc2866.AcctStatusType_Value_Start)
	assert.Equal(t, onlineBefore+1, app.GetRadiusMetrics(app.MetricsRadiusOline))
	assert.Equal(t, offlineBefore, app.GetRadiusMetrics(app.MetricsRadiusOffline))

	account(rfc2866.AcctStatusType_Value_Stop)
	assert.Equal(t, offlineBefore+1, app.GetRadiusMetrics(app.MetricsRadiusOffline))

	// An Interim-Update changes neither session-event counter.
	account(rfc2866.AcctStatusType_Value_InterimUpdate)
	assert.Equal(t, onlineBefore+1, app.GetRadiusMetrics(app.MetricsRadiusOline))
	assert.Equal(t, offlineBefore+1, app.GetRadiusMetrics(app.MetricsRadiusOffline))
}
