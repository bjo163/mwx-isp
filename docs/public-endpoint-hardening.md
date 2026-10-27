# Public Endpoint Hardening (H03)

Abuse controls for the unauthenticated and customer-facing surfaces. All limits
are in-process token buckets; a restart clears them. For multi-replica
deployments, replace the store with a shared backend before raising limits.

## Client IP trust

`internal/webserver/server.go` always installs an `echo.IPExtractor`:

- Default: `echo.ExtractIPDirect()` — the socket peer address. `X-Forwarded-For`
  is ignored.
- With `TOUGHRADIUS_TRUSTED_PROXIES` set to a comma-separated CIDR list, the
  extractor honors `X-Forwarded-For` only from those proxies. Bare IPv4 entries
  become `/32`, bare IPv6 entries become `/128`; invalid entries are logged and
  skipped.

Without this policy Echo's legacy `RealIP()` fallback trusts a client-supplied
`X-Forwarded-For`, which lets a caller rotate spoofed addresses to bypass a
per-IP limiter. Tests cover both the trusted and untrusted paths.

### Implicit trust and multi-hop chains (ENG-023)

Echo's XFF extractor trusts loopback, link-local, and private peers implicitly
when no option says otherwise. That default is unsafe here: a caller reaching
the process from a LAN address could rotate `X-Forwarded-For` and pick its own
rate-limit bucket. The extractor therefore disables link-local and private
trust explicitly (`echo.TrustLinkLocal(false)`, `echo.TrustPrivateNet(false)`)
and leaves only loopback implicit, so a reverse proxy on the same host works
without being listed.

Listing a proxy does not extend trust to its neighbours. Every hop between the
client and the process must be listed, including private ones; the walk stops
at the first untrusted address, and that address becomes the bucket key:

- A public proxy in front of a private load balancer needs both entries.
- An unlisted private hop is treated as the client, so every client behind it
  shares one budget. That is fail-closed: shared throttling, never a bypass.
- An unparsable `X-Forwarded-For` entry resolves to the direct peer.

## Budgets

| Surface | Middleware | Sustained rate | Burst | Bucket key | `Retry-After` |
| --- | --- | --- | --- | --- | --- |
| `POST /auth/login` | `loginRateLimiter` | 1 request / 3 s | 5 | client IP | 3 s |
| `GET /portal/lookup` | `portalLookupRateLimiter` | 1 request / 2 s | 10 | route + tenant + IP | 2 s |
| `GET /portal/invoices/:id/payment-channel` | `portalPaymentChannelRateLimiter` | 1 request / 2 s | 10 | route + tenant + IP | 2 s |
| `GET /public/vouchers/check` | `voucherCheckRateLimiter` | 1 request / 2 s | 10 | route + tenant + IP | 2 s |
| `POST /public/register` | `publicRegistrationRateLimiter` | 1 request / 20 s | 3 | route + tenant + IP | 20 s |

Throttled responses are JSON `429 RATE_LIMITED` with a `Retry-After` header in
seconds. The tenant component keeps one noisy tenant or address from exhausting
another tenant's budget.

## Payload and disclosure controls

- `/public/register` additionally applies `middleware.BodyLimit("64K")`; the
  global cap is 16 MB for large restore uploads.
- `/portal/lookup` returns the same not-found response for an unknown customer
  number and an unknown phone number.
- `/public/vouchers/check` returns one generic failure for missing, revoked and
  expired codes.
- 5xx responses are redacted by the shared `fail` helper.

## Deployment note

Set `TOUGHRADIUS_TRUSTED_PROXIES` whenever the service runs behind a reverse
proxy or load balancer, listing every hop — a private network is not trusted
implicitly. If it is unset, all clients appear to come from the proxy address
and share a single budget.

## Verification

| Check | Command |
| --- | --- |
| Limiter burst, tenant/IP isolation, spoofed XFF, login 429 + `Retry-After` | `go test ./internal/adminapi/ -run 'Test.*RateLimit|TestRetryAfter' -count=1` |
| Spoof attempt from a private peer hits one budget through the real server | `go test ./internal/adminapi/ -run TestLoginRateLimit -count=1` |
| Trusted-proxy IP extraction, implicit private/link-local trust, multi-hop walk | `go test ./internal/webserver/ -count=1` |
| Anonymous 401 on the four landing-surface APIs | `go test ./internal/adminapi/ -run TestLandingSurfaceEndpointsFailClosedAnonymously -count=1` |
| Full unit suites | `go test ./internal/adminapi/ ./internal/webserver/ -count=1` |

Current limitation: customer-owned public self-service is not enabled, so
`/portal/*`, `/public/register`, `/public/vouchers/check` and `/public/packages`
remain behind the JWT middleware. The limits apply once such a caller is
authenticated and protect the tenant from abuse by an authenticated account.

Because those calls answer 401 for a visitor, `/home` and `/portal` must not
present them as anonymous self-service or as a checkout step. Both pages now
label the lookup as staff-operator gated, surface the 401 as "login staf
diperlukan", and route an anonymous visitor to WhatsApp or the staff login. The
fail-closed behaviour is asserted by `TestLandingSurfaceEndpointsFailClosedAnonymously`,
so widening `JwtSkipPrefix` to make the copy true breaks the test instead of the
non-goal.
