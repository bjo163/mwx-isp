package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/pfsense"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestPfSenseAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-pfsense", NewPfSenseAcceptEnhancer().Name())
}

func TestPfSenseAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewPfSenseAcceptEnhancer()
	ctx := context.Background()

	tests := []struct {
		name    string
		authCtx *auth.AuthContext
	}{
		{name: "nil context"},
		{name: "nil response", authCtx: &auth.AuthContext{User: &domain.RadiusUser{}}},
		{name: "nil user", authCtx: &auth.AuthContext{Response: radius.New(radius.CodeAccessAccept, []byte("secret"))}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, enhancer.Enhance(ctx, tt.authCtx))
		})
	}
}

func TestPfSenseAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name       string
		vendorCode string
		upRate     int
		downRate   int
		wantUp     uint32
		wantDown   uint32
	}{
		{name: "rates in kbit/s", vendorCode: vendors.CodePfSense, upRate: 10240, downRate: 51200, wantUp: 10240, wantDown: 51200},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, upRate: 10240, downRate: 51200},
		{name: "zero rates skipped", vendorCode: vendors.CodePfSense},
		{name: "negative rates skipped", vendorCode: vendors.CodePfSense, upRate: -1, downRate: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", UpRate: tt.upRate, DownRate: tt.downRate},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewPfSenseAcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantUp, uint32(pfsense.PfSenseBandwidthMaxUp_Get(response)))
			assert.Equal(t, tt.wantDown, uint32(pfsense.PfSenseBandwidthMaxDown_Get(response)))
		})
	}
}
