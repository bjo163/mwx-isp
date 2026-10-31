package enhancers

import (
	"context"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// MicrosoftAcceptEnhancer adds Microsoft (SMI Private Enterprise Code 311)
// reply attributes for sessions terminating on Windows RRAS / NPS.
type MicrosoftAcceptEnhancer struct{}

// NewMicrosoftAcceptEnhancer returns a MicrosoftAcceptEnhancer ready for
// registration.
func NewMicrosoftAcceptEnhancer() *MicrosoftAcceptEnhancer {
	return &MicrosoftAcceptEnhancer{}
}

// Name returns the response-enhancer registry key for the Microsoft enhancer.
func (e *MicrosoftAcceptEnhancer) Name() string {
	return "accept-microsoft"
}

// Enhance adds Microsoft's policy attribute to the Access-Accept:
//
//   - MS-Service-Class (type 42, string): the service class the RAS server
//     applies to the connection, taken from the generic Domain vendor-policy
//     field via GetDomain (so it honors profile inheritance). This mirrors the
//     Aruba/Huawei convention where Domain carries the NAS's primary policy
//     selector.
//
// The MS-MPPE encryption attributes (type 12/13) are not written here: they are
// key material derived from the user's password during EAP/CHAP authentication,
// not a policy value this enhancer may invent.
//
// Enhance is a no-op for a nil context/response/user, for non-Microsoft NAS
// devices, and when the source field is unset, so it never mutates an
// Access-Accept it does not own.
func (e *MicrosoftAcceptEnhancer) Enhance(ctx context.Context, authCtx *auth.AuthContext) error {
	if authCtx == nil || authCtx.Response == nil || authCtx.User == nil {
		return nil
	}
	if !matchVendor(authCtx, vendors.CodeMicrosoft) {
		return nil
	}

	user := authCtx.User
	resp := authCtx.Response

	// Get profile cache from metadata
	var profileCache interface{}
	if authCtx.Metadata != nil {
		profileCache = authCtx.Metadata["profile_cache"]
	}

	if serviceClass := user.GetDomain(profileCache); common.IsNotEmptyAndNA(serviceClass) {
		_ = microsoft.MSServiceClass_SetString(resp, serviceClass) //nolint:errcheck
	}

	return nil
}
