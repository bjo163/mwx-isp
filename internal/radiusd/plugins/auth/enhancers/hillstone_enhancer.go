package enhancers

import (
	"context"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/hillstone"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// HillstoneAcceptEnhancer adds Hillstone (SMI Private Enterprise Code 28557)
// reply attributes for SSL VPN / NAC sessions.
type HillstoneAcceptEnhancer struct{}

// NewHillstoneAcceptEnhancer returns a HillstoneAcceptEnhancer ready for
// registration.
func NewHillstoneAcceptEnhancer() *HillstoneAcceptEnhancer {
	return &HillstoneAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the Hillstone enhancer.
func (e *HillstoneAcceptEnhancer) Name() string {
	return "accept-hillstone"
}

// Enhance adds Hillstone's provisioning attributes to the Access-Accept:
//
//   - Hillstone-User-Role-Bame (type 9, string): the subscriber's role, taken
//     from the generic Domain vendor-policy field via GetDomain (so it honors
//     profile inheritance). The attribute name keeps the "Bame" spelling of the
//     shipped FreeRADIUS dictionary; the wire type is what the NAS matches.
//   - Hillstone-VPN-DHCP-Pool (type 102, string): the DHCP pool the VPN gateway
//     allocates the client address from, resolved through GetAddrPool so
//     profile inheritance is honored.
//
// Enhance is a no-op for a nil context/response/user, for non-Hillstone NAS
// devices, and when the source fields are unset, so it never mutates an
// Access-Accept it does not own.
func (e *HillstoneAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeHillstone) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if role := user.GetDomain(profileCache); common.IsNotEmptyAndNA(role) {
		_ = hillstone.HillstoneUserRoleBame_SetString(resp, role) //nolint:errcheck
	}

	if addrPool := user.GetAddrPool(profileCache); common.IsNotEmptyAndNA(addrPool) {
		_ = hillstone.HillstoneVPNDHCPPool_SetString(resp, addrPool) //nolint:errcheck
	}

	return nil
}
