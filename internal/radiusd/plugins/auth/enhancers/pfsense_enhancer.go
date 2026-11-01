package enhancers

import (
	"context"
	"math"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/pfsense"
)

// PfSenseAcceptEnhancer adds pfSense (SMI Private Enterprise Code 13644) reply
// attributes for captive-portal and VPN subscriber sessions.
type PfSenseAcceptEnhancer struct{}

// NewPfSenseAcceptEnhancer returns a PfSenseAcceptEnhancer ready for
// registration.
func NewPfSenseAcceptEnhancer() *PfSenseAcceptEnhancer {
	return &PfSenseAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the pfSense enhancer.
func (e *PfSenseAcceptEnhancer) Name() string {
	return "accept-pfsense"
}

// Enhance adds pfSense's rate-limit attributes to the Access-Accept:
//
//   - pfSense-Bandwidth-Max-Up (type 1, integer): the subscriber's upload
//     ceiling in kbit/s (the unit dummynet expects), taken from GetUpRate.
//   - pfSense-Bandwidth-Max-Down (type 2, integer): the subscriber's download
//     ceiling in kbit/s, taken from GetDownRate.
//
// Both rates are stored in kbit/s in the user/profile model, so they are sent
// unchanged; a zero or negative rate means "no limit configured" and is skipped
// rather than advertised as a zero-bandwidth pipe.
//
// Enhance is a no-op for a nil context/response/user, for non-pfSense NAS
// devices, and when the source fields are unset, so it never mutates an
// Access-Accept it does not own.
func (e *PfSenseAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodePfSense) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if up := user.GetUpRate(profileCache); up > 0 && up <= math.MaxUint32 {
		_ = pfsense.PfSenseBandwidthMaxUp_Set(resp, pfsense.PfSenseBandwidthMaxUp(up)) //nolint:errcheck,gosec // G115: value guarded to 1..MaxUint32
	}
	if down := user.GetDownRate(profileCache); down > 0 && down <= math.MaxUint32 {
		_ = pfsense.PfSenseBandwidthMaxDown_Set(resp, pfsense.PfSenseBandwidthMaxDown(down)) //nolint:errcheck,gosec // G115: value guarded to 1..MaxUint32
	}

	return nil
}
