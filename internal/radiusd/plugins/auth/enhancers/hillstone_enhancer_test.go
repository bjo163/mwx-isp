package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/hillstone"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestHillstoneAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-hillstone", NewHillstoneAcceptEnhancer().Name())
}

func TestHillstoneAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewHillstoneAcceptEnhancer()
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

func TestHillstoneAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name       string
		vendorCode string
		domain     string
		addrPool   string
		wantRole   string
		wantPool   string
	}{
		{name: "hillstone user", vendorCode: vendors.CodeHillstone, domain: "employee", addrPool: "pool-h", wantRole: "employee", wantPool: "pool-h"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, domain: "employee", addrPool: "pool-h"},
		{name: "unset fields skipped", vendorCode: vendors.CodeHillstone},
		{name: "N/A values skipped", vendorCode: vendors.CodeHillstone, domain: "N/A", addrPool: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", Domain: tt.domain, AddrPool: tt.addrPool},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewHillstoneAcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantRole, hillstone.HillstoneUserRoleBame_GetString(response))
			assert.Equal(t, tt.wantPool, hillstone.HillstoneVPNDHCPPool_GetString(response))
		})
	}
}
