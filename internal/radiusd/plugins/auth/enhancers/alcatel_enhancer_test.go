package enhancers

import (
	"context"
	"net"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/alcatel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestAlcatelAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-alcatel", NewAlcatelAcceptEnhancer().Name())
}

func TestAlcatelAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewAlcatelAcceptEnhancer()
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

func TestAlcatelAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name       string
		vendorCode string
		ipAddr     string
		addrPool   string
		wantIP     string
		wantPool   string
	}{
		{name: "alcatel user", vendorCode: vendors.CodeAlcatel, ipAddr: "192.0.2.10", addrPool: "pool-a", wantIP: "192.0.2.10", wantPool: "pool-a"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, ipAddr: "192.0.2.10", addrPool: "pool-a"},
		{name: "unset fields skipped", vendorCode: vendors.CodeAlcatel},
		{name: "ipv6 address skipped", vendorCode: vendors.CodeAlcatel, ipAddr: "2001:db8::1"},
		{name: "unparseable address skipped", vendorCode: vendors.CodeAlcatel, ipAddr: "not-an-ip"},
		{name: "N/A pool skipped", vendorCode: vendors.CodeAlcatel, addrPool: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", IpAddr: tt.ipAddr, AddrPool: tt.addrPool},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewAlcatelAcceptEnhancer().Enhance(context.Background(), authCtx))

			gotIP, err := alcatel.AATPPPAddress_Lookup(response)
			if tt.wantIP == "" {
				assert.ErrorIs(t, err, radius.ErrNoAttribute)
			} else {
				require.NoError(t, err)
				assert.True(t, net.ParseIP(tt.wantIP).Equal(gotIP))
			}
			assert.Equal(t, tt.wantPool, alcatel.AATIPPoolDefinition_GetString(response))
		})
	}
}
