package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/radback"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestRadbackAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-radback", NewRadbackAcceptEnhancer().Name())
}

func TestRadbackAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewRadbackAcceptEnhancer()
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

func TestRadbackAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name         string
		vendorCode   string
		addrPool     string
		domain       string
		wantPool     string
		wantProfile  string
	}{
		{name: "radback user", vendorCode: vendors.CodeRadback, addrPool: "pool-r", domain: "gold", wantPool: "pool-r", wantProfile: "gold"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, addrPool: "pool-r", domain: "gold"},
		{name: "unset fields skipped", vendorCode: vendors.CodeRadback},
		{name: "N/A values skipped", vendorCode: vendors.CodeRadback, addrPool: "N/A", domain: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", AddrPool: tt.addrPool, Domain: tt.domain},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewRadbackAcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantPool, radback.IPAddressPoolName_GetString(response))
			assert.Equal(t, tt.wantProfile, radback.SubscriberProfileName_GetString(response))
		})
	}
}
