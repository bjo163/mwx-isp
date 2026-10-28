package radiusd

import (
	"testing"

	eap "github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/registry"
	"github.com/stretchr/testify/require"
)

// fakeEAPHandler registers a method name without a working implementation; the
// allowlist only consults Name, so the rest of the interface is inert.
type fakeEAPHandler struct {
	name    string
	eapType uint8
}

func (h *fakeEAPHandler) Name() string                   { return h.name }
func (h *fakeEAPHandler) EAPType() uint8                 { return h.eapType }
func (h *fakeEAPHandler) CanHandle(*eap.EAPContext) bool { return false }
func (h *fakeEAPHandler) HandleIdentity(*eap.EAPContext) (bool, error) {
	return false, nil
}
func (h *fakeEAPHandler) HandleResponse(*eap.EAPContext) (bool, error) {
	return false, nil
}

// newAllowlistService builds the smallest AuthService that can evaluate the
// configured EAP allowlist.
func newAllowlistService(t *testing.T, enabled string) *AuthService {
	t.Helper()
	appCtx, _ := setupTestEnv(t)
	t.Cleanup(appCtx.Release)

	registry.ResetForTest()
	t.Cleanup(registry.ResetForTest)
	registry.RegisterEAPHandler(&fakeEAPHandler{name: "eap-md5", eapType: eap.TypeMD5Challenge})
	registry.RegisterEAPHandler(&fakeEAPHandler{name: "eap-mschapv2", eapType: eap.TypeMSCHAPv2})

	require.NoError(t, appCtx.ConfigMgr().Set("radius", "EapEnabledHandlers", enabled))
	return &AuthService{RadiusService: &RadiusService{appCtx: appCtx}}
}

// TestEAPAllowlistIgnoresMethodsWithoutHandler is the regression named by the
// item: EAP-OTP has a handler source file but no validator and is not
// registered, so listing it in the allowlist must not make it the effective
// method.
func TestEAPAllowlistIgnoresMethodsWithoutHandler(t *testing.T) {
	service := newAllowlistService(t, "eap-otp,eap-mschapv2")
	allowed := service.initAllowedEAPHandlers()

	require.Equal(t, []string{"eap-mschapv2"}, allowed)
	require.Equal(t, "eap-mschapv2", service.resolveEapMethod("eap-otp"),
		"an unavailable method must fall back to a registered one")
}

// TestEAPAllowlistWithoutUsableMethodDeniesEAP pins fail-closed behavior: an
// allowlist that names only unavailable methods must deny EAP, not silently
// enable every registered method.
func TestEAPAllowlistWithoutUsableMethodDeniesEAP(t *testing.T) {
	service := newAllowlistService(t, "eap-otp")
	allowed := service.initAllowedEAPHandlers()

	require.Equal(t, []string{eap.MethodUnavailable}, allowed)
	require.Equal(t, eap.MethodUnavailable, service.resolveEapMethod("eap-md5"))
}

// TestEAPAllowlistUnsetKeepsEveryMethod proves the guard does not change the
// default: with no allowlist every registered method stays selectable, and a
// preferred method without a handler is returned as-is, so the coordinator
// rejects it instead of substituting another method.
func TestEAPAllowlistUnsetKeepsEveryMethod(t *testing.T) {
	service := newAllowlistService(t, "")
	allowed := service.initAllowedEAPHandlers()

	require.Empty(t, allowed)
	require.Equal(t, "eap-md5", service.resolveEapMethod("eap-md5"))
	require.Equal(t, "eap-otp", service.resolveEapMethod("eap-otp"))
}
