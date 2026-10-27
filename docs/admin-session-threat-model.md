# Admin browser session threat model (ENG-022)

Scope: operator login for the React Admin SPA and the `/api/v1` management API
under TR-F016. RADIUS authentication and the customer portal are separate
surfaces.

## Credential and trust boundary

- Login returns an HS256 JWT signed with `web.secret`
  (`internal/adminapi/auth.go:182-197`). Claims: `sub`, `tenant_id`,
  `membership_version`, `username`, `role`, `exp`, `iat`, `nbf`, `iss`.
- The token has a 12h absolute lifetime (`tokenTTL`, `auth.go:23`). There is no
  refresh token and no sliding renewal.
- The SPA keeps the token in `localStorage` and sends it as
  `Authorization: Bearer <token>` (`web/src/utils/apiClient.ts:26-29`). The API
  is served same-origin at `/admin`, so the browser exists inside one origin.
- Every protected request re-resolves the operator from the signed claims and
  re-checks tenant status, operator status, and membership `token_version`
  (`auth.go:223-289`).

## Threat table

| # | Threat | Current behaviour | Verdict |
| --- | --- | --- | --- |
| 1 | Forged token (wrong key, `alg: none`, tampered payload) | Echo JWT pins `HS256`, verifies the signature, and rejects the request with `401`. | Mitigated; `TestAdminTokenAbuseIsRejected`. |
| 2 | Expired token | `exp` is validated by the JWT library; `401`. | Mitigated; same test. |
| 3 | Cookie or ambient credentials | The API reads only the `Authorization: Bearer` header. Cookies, `Token <jwt>`, and query parameters are ignored (`401`). | Mitigated; same test. |
| 4 | Classic CSRF | No ambient credential exists to replay, so cross-site form posts and `fetch` cannot authenticate. There is nothing to exploit, which is why this review does not label CSRF as a vulnerability. | Not applicable today; the ratchet in `TestAdminBrowserStorageContractKeepsBearerHeaderAuth` fails if cookie credentials or `withCredentials` appear. |
| 5 | CORS-driven cross-origin reads | CORS is disabled by default; when enabled, only the `TOUGHRADIUS_WEB_CORS_ORIGINS` allow-list is reflected. | Mitigated; `internal/webserver/security_test.go`. |
| 6 | XSS stealing the token | The token is readable by any script in the origin. React escapes text children, the branding editor publishes inert text, and `TestBrandingEditorSourceRendersTextOnly` forbids `dangerouslySetInnerHTML`. No `Content-Security-Policy` header is sent (`internal/webserver/server.go:101-107`). | Accepted risk: an XSS in the admin SPA yields token theft for up to 12h. Cookie storage would trade this for CSRF and is deliberately not used. |
| 7 | Logout does not revoke server-side | "Logout" clears `localStorage`; `/api/v1/auth/logout` and `/api/v1/auth/refresh` are not registered routes. A copied token stays valid until `exp`. | Documented limit; `TestAdminSessionHasNoServerSideRefreshOrLogout`. |
| 8 | Revocation after theft | Disabling the operator, revoking the membership, or disabling the tenant fails the next request immediately because the token version and account state are re-read. | Mitigated; `TestResolveOperatorFromContext_DisabledAccount`, `TestResolveOperatorFromContext_IgnoresInjectedOperatorInProduction`, `TestTenantContextMiddlewareRejectsRevokedMembership`, and the ENG-017 regression. |
| 9 | Tenant confusion | The tenant is taken from the signed `tenant_id` claim. A client header or query parameter claiming another tenant is ignored. | Mitigated; `TestAdminTenantBoundFromSignedToken`. |
| 10 | Login brute force | Per-client-IP token bucket (`3s` refill, burst 5), with `RealIP` derived from trusted proxies only. | Mitigated; `auth_ratelimit_test.go`. |
| 11 | Deploying with the placeholder signing key | Empty or `DefaultWebSecret` is fatal in production runtime and a warning in debug. | Mitigated; `internal/app/app.go:348-380`. |
| 12 | Unauthenticated route enumeration | Unmatched `/api/v1/*` paths still pass the JWT middleware, so anonymous callers get `401` and authenticated callers `404`. | Mitigated; `TestAdminSessionHasNoServerSideRefreshOrLogout`. |

## Consequences for operators

- A stolen token is usable for up to 12h unless the account, membership, or
  tenant is revoked. Rotate `web.secret` to invalidate every outstanding token;
  that also ends every session, so treat it as an incident action.
- If a deployment enables cross-origin admin access, keep the CORS allow-list
  exact and never add cookie-based auth without a CSRF design.

## Verification

```bash
go test ./internal/adminapi/ -run 'TestAdminTokenAbuseIsRejected|TestAdminSessionHasNoServerSideRefreshOrLogout|TestAdminTenantBoundFromSignedToken|TestAdminBrowserStorageContractKeepsBearerHeaderAuth' -count=1
go test ./internal/webserver/ -run 'TestCors|TestSecurityHeaders' -count=1
go test ./internal/adminapi/ -count=1
```

## Deliberately not claimed

- The review does not assert that CSRF is exploitable; the bearer-header design
  removes the ambient credential that CSRF needs.
- `localStorage` is not claimed to be XSS-safe. It is a deliberate trade-off that
  requires the CSP work to be evaluated separately.
