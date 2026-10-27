# Hotspot voucher lifecycle

This document defines the server-enforced lifecycle for prepaid hotspot
vouchers (`isp_hotspot_voucher`) and the RADIUS credential linked to each
voucher. It is the reference for issue #49 (AUDIT-05) and must be updated when
authentication, accounting, or revocation behavior changes.

## State model

| Status | Meaning | Can authenticate? |
| --- | --- | --- |
| `active` | Generated and never used. `first_login_at` and `expires_at` are empty. | Yes, and the first successful login starts the validity window. |
| `used` | First login happened. `first_login_at` and `expires_at` are set. | Yes, until `expires_at`. |
| `expired` | The validity window ended. | No. |
| `revoked` | An operator revoked the voucher. The linked RADIUS user is `disabled`. | No. |

Allowed transitions:

```
active --first successful auth--> used --expires_at reached--> expired
active --operator revoke--> revoked
used   --operator revoke--> revoked
```

`expired` and `revoked` are terminal. Deleting a batch removes its vouchers and
their RADIUS credentials, but only when the stored username and password still
match, so a reused credential is never deleted by accident.

## Validity starts at first successful login

The generator sets the RADIUS user's `expire_time` far in the future so the
account exists, but it does not hand out a usable validity window. Validity is
owned by the voucher row:

1. The auth pipeline resolves the RADIUS user and validates credentials
   (PAP, CHAP, EAP and MAC paths all pass through the same stages).
2. After credentials pass, `activateHotspotVoucher` performs one conditional
   update: `WHERE id = ? AND status = 'active' AND first_login_at IS NULL`.
   The winner sets `first_login_at`, `expires_at = first_login_at +
   validity_seconds`, and `status = 'used'`, and writes the same expiry to
   `radius_user.expire_time`.
3. Concurrent first logins therefore share a single activation timestamp and a
   single expiry; late losers re-read the row and use the winner's expiry.
4. A failed password never consumes the validity window because activation runs
   only after credential validation succeeds.

Re-authentication never extends the window. Once `expires_at` has passed,
authentication is rejected and the voucher is marked `expired`.

## Concurrent sessions and expiry

Multiple simultaneous sessions for one voucher are allowed; a per-profile
`ActiveNum` limit, when configured, still applies through the existing online
count checker. Every Access-Accept carries `Session-Timeout` (RFC 2865 §5.27)
equal to the remaining validity in seconds, so all sessions for a voucher share
the same absolute end time even when they started at different moments.

When `expires_at` is reached:

- New authentications are rejected and the voucher becomes `expired`.
- Sessions already running are terminated by the NAS at `Session-Timeout`
  expiry; the NAS then re-authenticates and is rejected. The server does not
  schedule a proactive disconnect for expiry, because the standard
  timeout/re-authentication cycle already bounds the session lifetime.

## Revocation and live sessions

`POST /api/v1/isp/vouchers/:id/revoke` (admin only):

1. Atomically flips the voucher from `active`/`used` to `revoked`.
2. Disables the linked RADIUS user (`status = disabled`), so the credential
   cannot be reused even after the voucher row is removed.
3. Sends an RFC 5176 Disconnect-Request to every live session with that
   username and reports `disconnected_sessions` / `failed_disconnects`.
   The client retransmits once (default CoA timeout/retry policy).
4. Is idempotent: revoking an already-revoked voucher returns
   `already_revoked: true` and sends no further disconnect requests.

Expired vouchers cannot be revoked (`409 VOUCHER_EXPIRED`); they are already
unusable. A disconnect failure does not roll back the revoke: the credential is
already disabled, so no new session can start, and the NAS drops the stale
session at its next re-authentication or accounting stop. The failure is
returned to the operator and logged for follow-up.

## Data quota is intentionally disabled

Byte-quota enforcement is **not** implemented for vouchers. The system rejects
`quota_mb > 0` at generation time (`QUOTA_NOT_SUPPORTED`), stores
`quota_bytes = 0`, reports `quota_enforced: false` and
`usage_available: false` from the public check endpoint, and labels stored
non-zero quotas as "Not enforced" in the admin UI. Do not advertise voucher
data caps until Phase B (tenant-scoped byte accounting with interim/stop
reconciliation) ships.

## Enforcement points

| Control | Location |
| --- | --- |
| Reject disabled/expired user | `internal/radiusd/radius.go` (`getValidUser`) |
| First-login activation | `internal/radiusd/radius.go` (`activateHotspotVoucher`) |
| Auth hooks (PAP/CHAP/EAP) | `internal/radiusd/auth_stages.go` (`stagePluginAuth`, `stageEAPDispatch`) |
| Session-Timeout = remaining validity | `internal/radiusd/plugins/auth/enhancers/default_enhancer.go` |
| Revoke API | `internal/adminapi/isp_operations.go` (`revokeVoucher`) |
| Admin UI | `web/src/pages/HotspotVouchersPage.tsx` |

## Verification

- Unit: `go test ./internal/radiusd/ -run 'Voucher|Hotspot'`
- Unit: `go test ./internal/adminapi/ -run 'Voucher|Revoke'`
- Terminal-state concurrency: `go test ./internal/radiusd/ -run
  'TestHotspotVoucherTerminalStatesNeverReactivate|TestHotspotVoucherRevokedAfterFirstLoginStaysTerminal'`
  proves that expired/revoked rows never hand out a live window, and that
  login → revoke → login keeps the revoked status, the original timestamps,
  and the disabled credential.
- Revoke concurrency and credential linkage: `go test ./internal/adminapi/ -run
  'TestRevokeVoucherConcurrentRequestsConvergeOnOneTerminalState|TestRevokeVoucherPastValidityWindowStaysTerminal|TestRevokeVoucherFallbackDisablesOnlyMatchingCredential'`
  proves one fresh revoke per voucher under concurrent calls, a revoked status
  that survives the lazy expiry marker, and that the username+password fallback
  never disables an unrelated account.
- PostgreSQL + RADIUS integration: `test/integration/isp_voucher_test.go`
  (not-yet-activated, activation, expiry, concurrent first login, revocation).
