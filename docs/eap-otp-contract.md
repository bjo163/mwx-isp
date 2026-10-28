# EAP-OTP: fail-closed status and backend contract (TR-F004 / ENG-008)

EAP-OTP (EAP Type 5, RFC 3748 §5.4) sends a text challenge and treats the
peer's next EAP-Response as a one-time password. MWX-ISP ships the handler source
`internal/radiusd/plugins/eap/handlers/otp_handler.go` but has **no OTP
validation backend**, so EAP-OTP is not an available method. This document
records how that is enforced, which tests pin it, and what a real backend has to
provide before the method may be enabled.

## Why it is not available

The original implementation compared the response with a hardcoded `123456`,
which is an authentication bypass: any peer that guessed the literal was
accepted. The handler now always returns `eap.ErrOTPNotConfigured` from
`HandleResponse`, and `HandleIdentity` is unreachable in production because the
handler is not registered.

## Four independent guards

Each layer would be enough on its own; together a configuration mistake in one
of them cannot enable the method.

| Layer | Behavior | Test |
| --- | --- | --- |
| Handler response | `HandleResponse` returns `ErrOTPNotConfigured` for every input, including the former fixed password | `TestOTPHandler_HandleResponse_NeverAuthenticates` |
| Registration | `plugins.InitPlugins` does not register `TypeOTP` (5), so the coordinator cannot dispatch an OTP exchange | `TestInitPlugins_EAPHandlers` |
| Method setting | `radius.EapMethod` is enum-validated to `eap-md5/eap-mschapv2/eap-tls/eap-peap/eap-ttls`; `eap-otp` cannot be saved | `TestSaveSettings_UnknownAndInvalidValuesRejected` |
| Handler allowlist | `radius.EapEnabledHandlers` drops names without a registered handler; if nothing usable remains, every method is denied (`eap.MethodUnavailable`) instead of falling back | `TestEAPAllowlistIgnoresMethodsWithoutHandler`, `TestEAPAllowlistWithoutUsableMethodDeniesEAP`, `TestEAPAllowlistWithoutUsableMethodDeniesEAPEndToEnd` |

The allowlist is snapshotted when the auth service starts
(`NewAuthService` → `initAllowedEAPHandlers`), and the effective method is
resolved per request from `radius.EapMethod`, so a runtime change to
`EapMethod` takes effect on the next request while the allowlist needs a restart.

## Verification

```bash
go test ./internal/radiusd/plugins/eap/handlers/ -run 'TestOTPHandler' -count=1
go test ./internal/radiusd/plugins/ -run 'TestInitPlugins_EAPHandlers' -count=1
go test ./internal/radiusd/plugins/eap/ -run 'TestHandleEAPRequest_IdentityResponse_UnavailableMethod' -count=1
go test ./internal/radiusd/ -run 'TestEAPAllowlist' -count=1
go test ./internal/app/ -run 'TestSaveSettings_UnknownAndInvalidValuesRejected' -count=1
```

## Backend contract before enabling EAP-OTP

Enabling the method means wiring a validator and then reversing all four guards
deliberately. A backend must satisfy:

1. **Issuance.** The challenge is generated per handshake and bound to the RADIUS
   `State` value and the authenticated identity, not to a shared constant. The
   `EAPState` record already carries `Username`, `StateID`, and `Method`; the
   code (or its hash) belongs there or in a dedicated store with the same
   lifetime as the state.
2. **Single use.** A code is consumed on the first verification attempt,
   successful or not, so a captured response cannot be replayed within the state
   TTL.
3. **Expiry.** Codes expire on a short, configured lifetime, independent of the
   EAP state TTL, and an expired code rejects with a distinct reason.
4. **Constant-time comparison.** Compare with `crypto/subtle` over a hash, never
   `==` on the plaintext code.
5. **Attempt limit.** Bound the number of wrong codes per state and per user, and
   reject the state after the limit so the challenge cannot be brute-forced.
6. **Delivery is out of band.** The backend only verifies a code that was
   delivered through a separate channel (SMS, e-mail, authenticator app). MWX-ISP
   must not generate and display the code to the same client.
7. **No silent fallback.** A validator error (backend unreachable, code expired,
   unknown user) rejects with `ErrOTPNotConfigured`-style failure semantics; it
   must never authenticate and never fall back to the local password.

To turn the method on, a change must, in one PR:

- implement `HandleResponse` against such a validator and delete the
  always-reject branch;
- register the handler in `plugins.InitPlugins`;
- add `eap-otp` to the enum in `internal/app/config_schemas.json` **and** in the
  hardcoded fallback schema in `internal/app/config_manager.go`;
- add positive, wrong-code, replay, expired-code, attempt-limit, and
  backend-down tests, plus an end-to-end handshake in
  `internal/radiusd/eap_integration_test.go`;
- update this document, the TR-F004 row, and the operator docs that list the
  selectable methods.

## Limits

- The guards above cover this repository's configuration paths. A build that
  patches out `InitPlugins` and the settings enum could obviously enable
  anything; no library-level protection is claimed.
- `HandleIdentity` still builds and sends an OTP challenge if the handler is
  registered by hand. That is intentional: the refusal lives in `HandleResponse`
  and in the registration/configuration guards, so a manually registered handler
  still cannot authenticate.
- No real OTP backend has been evaluated. The contract above is the minimum, not
  a substitute for a security review of the chosen delivery channel.
