package enhancers

import (
	"context"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/radback"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// RadbackAcceptEnhancer adds Redback/radback (SMI Private Enterprise Code 2352)
// reply attributes for sessions terminating on a SmartEdge/Ericsson BNG.
type RadbackAcceptEnhancer struct{}

// NewRadbackAcceptEnhancer returns a RadbackAcceptEnhancer ready for
// registration.
func NewRadbackAcceptEnhancer() *RadbackAcceptEnhancer {
	return &RadbackAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the radback enhancer.
func (e *RadbackAcceptEnhancer) Name() string {
	return "accept-radback"
}

// Enhance adds Redback's provisioning attributes to the Access-Accept:
//
//   - IP-Address-Pool-Name (type 36, string): the subscriber's address pool,
//     resolved through GetAddrPool so profile inheritance is honored.
//   - Subscriber-Profile-Name (type 91, string): the per-subscriber policy
//     profile, taken from the generic Domain vendor-policy field via GetDomain.
//     This mirrors the Aruba/Huawei convention where Domain carries the NAS's
//     primary policy selector.
//
// Enhance is a no-op for a nil context/response/user, for non-radback NAS
// devices, and when the source fields are unset, so it never mutates an
// Access-Accept it does not own.
func (e *RadbackAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeRadback) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if addrPool := user.GetAddrPool(profileCache); common.IsNotEmptyAndNA(addrPool) {
		_ = radback.IPAddressPoolName_SetString(resp, addrPool) //nolint:errcheck
	}

	if profile := user.GetDomain(profileCache); common.IsNotEmptyAndNA(profile) {
		_ = radback.SubscriberProfileName_SetString(resp, profile) //nolint:errcheck
	}

	return nil
}
