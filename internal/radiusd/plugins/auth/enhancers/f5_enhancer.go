package enhancers

import (
	"context"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/f5"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// F5AcceptEnhancer adds F5 BIG-IP (SMI Private Enterprise Code 3375) reply
// attributes for APM/Edge Client remote-access sessions.
type F5AcceptEnhancer struct{}

// NewF5AcceptEnhancer returns an F5AcceptEnhancer ready for registration.
func NewF5AcceptEnhancer() *F5AcceptEnhancer {
	return &F5AcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the F5 enhancer.
func (e *F5AcceptEnhancer) Name() string {
	return "accept-f5"
}

// Enhance adds F5's provisioning attribute to the Access-Accept:
//
//   - F5-LTM-User-Partition (type 3, string): the administrative partition the
//     authenticated user is placed in, taken from the generic Domain
//     vendor-policy field via GetDomain (so it honors profile inheritance). This
//     mirrors the Aruba/Huawei convention where Domain carries the NAS's primary
//     policy selector.
//
// The F5-LTM-User-Role/Shell/Console attributes are not written: they carry an
// administrative privilege level the subscriber record does not model, and
// inventing a value would grant a default privilege the operator never chose.
//
// Enhance is a no-op for a nil context/response/user, for non-F5 NAS devices,
// and when the source field is unset, so it never mutates an Access-Accept it
// does not own.
func (e *F5AcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeF5) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if partition := user.GetDomain(profileCache); common.IsNotEmptyAndNA(partition) {
		_ = f5.F5LTMUserPartition_SetString(resp, partition) //nolint:errcheck
	}

	return nil
}
