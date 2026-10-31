package enhancers

import (
	"context"
	"strconv"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/juniper"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// juniperMaxVLANID is the largest assignable IEEE 802.1Q VLAN ID. VLAN 0 and
// 4095 are reserved, so a valid subscriber VLAN is 1..4094.
const juniperMaxVLANID = 4094

// JuniperAcceptEnhancer adds Juniper (SMI Private Enterprise Code 2636) reply
// attributes for sessions terminating on a Junos/MAG/BNG NAS.
type JuniperAcceptEnhancer struct{}

// NewJuniperAcceptEnhancer returns a JuniperAcceptEnhancer ready for
// registration.
func NewJuniperAcceptEnhancer() *JuniperAcceptEnhancer {
	return &JuniperAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the Juniper enhancer.
func (e *JuniperAcceptEnhancer) Name() string {
	return "accept-juniper"
}

// Enhance adds Juniper's provisioning attributes to the Access-Accept:
//
//   - Juniper-Local-User-Name (type 1, string): the local user whose session
//     the Access-Accept authorizes, set to the RADIUS username.
//   - Juniper-Ip-Pool-Name (type 36, string): the address pool the NAS allocates
//     from, resolved through GetAddrPool so profile inheritance is honored.
//   - Juniper-VoIP-Vlan (type 49, string): the voice VLAN carried as a decimal
//     string, taken from the user's Vlanid2. Only a valid 802.1Q ID (1..4094) is
//     emitted; Vlanid1 is the data VLAN and has no Juniper equivalent here.
//
// Enhance is a no-op for a nil context/response/user, for non-Juniper NAS
// devices, and when the source fields are unset, so it never mutates an
// Access-Accept it does not own.
func (e *JuniperAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeJuniper) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if common.IsNotEmptyAndNA(user.Username) {
		_ = juniper.JuniperLocalUserName_SetString(resp, user.Username) //nolint:errcheck
	}

	if addrPool := user.GetAddrPool(profileCache); common.IsNotEmptyAndNA(addrPool) {
		_ = juniper.JuniperIPPoolName_SetString(resp, addrPool) //nolint:errcheck
	}

	if vlan := user.Vlanid2; vlan >= 1 && vlan <= juniperMaxVLANID {
		_ = juniper.JuniperVoIPVlan_SetString(resp, strconv.Itoa(vlan)) //nolint:errcheck
	}

	return nil
}
