package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestMicrosoftAcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-microsoft", NewMicrosoftAcceptEnhancer().Name())
}

func TestMicrosoftAcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewMicrosoftAcceptEnhancer()
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

func TestMicrosoftAcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name        string
		vendorCode  string
		domain      string
		wantService string
	}{
		{name: "service class from domain", vendorCode: vendors.CodeMicrosoft, domain: "vpn-users", wantService: "vpn-users"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, domain: "vpn-users"},
		{name: "empty domain skipped", vendorCode: vendors.CodeMicrosoft},
		{name: "N/A domain skipped", vendorCode: vendors.CodeMicrosoft, domain: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", Domain: tt.domain},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewMicrosoftAcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantService, microsoft.MSServiceClass_GetString(response))
		})
	}
}
