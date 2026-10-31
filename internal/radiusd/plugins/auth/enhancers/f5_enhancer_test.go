package enhancers

import (
	"context"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/f5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

func TestF5AcceptEnhancer_Name(t *testing.T) {
	assert.Equal(t, "accept-f5", NewF5AcceptEnhancer().Name())
}

func TestF5AcceptEnhancer_Enhance_NilSafety(t *testing.T) {
	enhancer := NewF5AcceptEnhancer()
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

func TestF5AcceptEnhancer_Enhance(t *testing.T) {
	tests := []struct {
		name          string
		vendorCode    string
		domain        string
		wantPartition string
	}{
		{name: "partition from domain", vendorCode: vendors.CodeF5, domain: "tenant-a", wantPartition: "tenant-a"},
		{name: "other vendor untouched", vendorCode: vendors.CodeHuawei, domain: "tenant-a"},
		{name: "empty domain skipped", vendorCode: vendors.CodeF5},
		{name: "N/A domain skipped", vendorCode: vendors.CodeF5, domain: "N/A"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := radius.New(radius.CodeAccessAccept, []byte("secret"))
			authCtx := &auth.AuthContext{
				Response: response,
				User:     &domain.RadiusUser{Username: "testuser", Domain: tt.domain},
				Nas:      &domain.NetNas{VendorCode: tt.vendorCode},
			}

			require.NoError(t, NewF5AcceptEnhancer().Enhance(context.Background(), authCtx))

			assert.Equal(t, tt.wantPartition, f5.F5LTMUserPartition_GetString(response))
		})
	}
}
