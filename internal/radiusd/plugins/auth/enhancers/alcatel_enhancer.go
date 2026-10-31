package enhancers

import (
	"context"
	"net"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/alcatel"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// AlcatelAcceptEnhancer adds the Alcatel (SMI Private Enterprise Code 3041)
// reply attributes a broadband access server needs to provision a subscriber.
type AlcatelAcceptEnhancer struct{}

// NewAlcatelAcceptEnhancer returns an AlcatelAcceptEnhancer ready for
// registration.
func NewAlcatelAcceptEnhancer() *AlcatelAcceptEnhancer {
	return &AlcatelAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the Alcatel enhancer.
func (e *AlcatelAcceptEnhancer) Name() string {
	return "accept-alcatel"
}

// Enhance adds Alcatel's provisioning attributes to the Access-Accept:
//
//   - AAT-PPP-Address (type 9, ipaddr): the subscriber's static framed IPv4
//     address, taken from the user's IpAddr. The value must parse as IPv4; an
//     IPv6 literal or garbage is skipped so a misconfiguration cannot emit a
//     malformed attribute.
//   - AAT-IP-Pool-Definition (type 63, string): the name of the NAS-local
//     address pool, resolved through GetAddrPool so profile inheritance is
//     honored. This is the Alcatel equivalent of the Cisco "ip:addr-pool"
//     AVPair emitted by CiscoAcceptEnhancer.
//
// Enhance is a no-op for a nil context/response/user, for non-Alcatel NAS
// devices, and when the source fields are unset, so it never mutates an
// Access-Accept it does not own.
func (e *AlcatelAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeAlcatel) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if common.IsNotEmptyAndNA(user.IpAddr) {
		if ip := net.ParseIP(user.IpAddr); ip != nil && ip.To4() != nil {
			_ = alcatel.AATPPPAddress_Set(resp, ip) //nolint:errcheck
		}
	}

	if addrPool := user.GetAddrPool(profileCache); common.IsNotEmptyAndNA(addrPool) {
		_ = alcatel.AATIPPoolDefinition_SetString(resp, addrPool) //nolint:errcheck
	}

	return nil
}
