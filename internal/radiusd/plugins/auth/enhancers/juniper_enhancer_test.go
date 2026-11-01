package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/juniper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestJuniperAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-juniper", NewJuniperAcceptEnhancer().Name())
}

func TestJuniperAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewJuniperAcceptEnhancer()
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

func TestJuniperAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name       string
		vendorCode string
		username   string
		addrPool   string
		vlanid2    int
		wantUser   string
		wantPool   string
		wantVlan   string
	}{
		{name: "juniper user", vendorCode: vendors.CodeJuniper, username: "bob", addrPool: "pool-b", vlanid2: 300, wantUser: "bob", wantPool: "pool-b", wantVlan: "300"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, username: "bob", addrPool: "pool-b", vlanid2: 300},
		{name: "reserved vlan skipped", vendorCode: vendors.CodeJuniper, username: "bob", vlanid2: 4095, wantUser: "bob"},
		{name: "zero vlan skipped", vendorCode: vendors.CodeJuniper, username: "bob", wantUser: "bob"},
		{name: "N/A pool skipped", vendorCode: vendors.CodeJuniper, username: "bob", addrPool: "N/A", wantUser: "bob"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: tt.username, AddrPool: tt.addrPool, Vlanid2: tt.vlanid2},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewJuniperAcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantUser, juniper.JuniperLocalUserName_GetString(response))
			assert.Equal(t, tt.wantPool, juniper.JuniperIPPoolName_GetString(response))
			assert.Equal(t, tt.wantVlan, juniper.JuniperVoIPVlan_GetString(response))
		})
	}
}
