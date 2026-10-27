# 100-item portfolio triage checklist (2026-10-27)

Provenance: `MWX-ISP_100_ISSUES_MASTER_2026-10-27.md` (kept at the repository root as the
source inventory), baseline `main`/`dev` at `1b791f8` (v0.14.0). The inventory lists
100 draft IDs — 64 Engineering, 24 Frontier, 12 Economy — with Charity and Faith/ontology
set to zero because they do not apply to a RADIUS/ISP monolith.

**These are not GitHub issues and are not approved work.** #45–#54 (AUDIT-01..10) are closed
prior art and must not be re-opened or duplicated by these drafts. 96 of the 100 items are
`PROPOSED_VERIFICATION`: they order an inspection or a test first, so none of them is a
confirmed defect. Only the `VERIFIED_GAP` rows record something already demonstrable against
the current tree.

## Triage tags

| Tag | Meaning | Gate before code changes |
| --- | --- | --- |
| `LOKAL` | Executable inside this repository — read, assert, test, document. | Ordinary TR-F scope and the Go/web gates. |
| `ASET` | Needs a real asset: physical NAS, clean VPS host, LDAP/WA number, or an operator willing to be measured. | Blocked until the asset and its authorization exist. |
| `RISET` | Frontier/Economy experiment. Research notes and bounded prototypes only. | No product code until `docs/feature-checklist.md` gains an approved TR-F. |
| `RISET+ASET` | Research that additionally needs real data or hardware. | Both gates above. |

## Execution rules

- One item per round. Check the box only when its own acceptance criterion is met with
  citable evidence, and note the commit or PR next to it.
- Never widen `JwtSkipPrefix`, revive a payment endpoint, or add a customer-facing workflow
  to make a UI claim true — those are TR-N001/TR-N002 non-goals that need a scope decision.
- `main`/`dev` branch policy, tags, releases and existing issues stay untouched without a
  separate instruction.

## Progress

Done: 23 / 100.

- `ENG-056` (2026-10-27, `dev` @ `a171299`): honest affordances for staff-gated
  portal actions. `/home` bill lookup, voucher check and registration submit, plus the
  `/portal` lookup, now answer a 401 with one informational `StaffLoginNotice` carrying
  "Login staf" and "Hubungi WhatsApp" actions; the red alert is reserved for real lookup
  failures, and the raw `Unauthorized` body is never shown. The `/portal` offline payment
  fallback opens the branded WhatsApp number instead of an empty chat picker.
  `web/e2e/anonymous-actions.spec.ts` covers the four anonymous actions, the login
  navigation and the branded WhatsApp target (5/5 chromium, 18.4s). No API was widened:
  `TestLandingSurfaceEndpointsFailClosedAnonymously` still passes, and
  `docs/public-endpoint-hardening.md` records the presentation rule.

- `ENG-038` (2026-10-27, `dev` @ `7c2e06e`): voucher terminal-state concurrency. The
  radiusd tests drive `activateHotspotVoucher` from eight goroutines against expired and
  revoked rows: every caller gets an expiry in the past, the status stays terminal, no
  first-login timestamp appears, and the credential expiry does not move. A deterministic
  login → revoke → login interleaving keeps the revoked status, the original timestamps,
  and the disabled credential. The adminapi tests fire the real revoke handler from eight
  goroutines: exactly one response performs the revoke, all responses are 200, and retries
  replay `already_revoked`. A `used` voucher whose window already passed stays revoked
  through the lazy expiry marker. The credential fallback matches username and password,
  so a shared password on another username and a rotated credential are both left
  enabled. `docs/voucher-lifecycle.md` lists the new commands; TR-N006/A20 records the
  coverage. Full radiusd and adminapi suites pass (40.2s / 47.4s).

- `ENG-035` (2026-10-27, `dev` @ `c070535`): subscription lifecycle state machine and
  rollback evidence. `internal/adminapi/isp_subscription_state_machine_test.go` drives all 16
  pending/active/suspended/terminated × activate/suspend/reactivate/terminate pairs through the
  real handler: accepted transitions must leave subscription and linked account consistent
  (`active` ⇔ `enabled`) and append exactly one typed billing event; rejected pairs return
  `409 INVALID_SUBSCRIPTION_STATE` with no subscription, account, or event change, and an
  unknown action returns `404 UNKNOWN_ACTION`. SQLite triggers inject a failure at the account
  update and at the billing-event insert, proving the transaction rolls back both earlier
  writes on a 500. Relink coverage rejects an already-linked, unknown, or non-numeric RADIUS
  account without creating rows, and a trigger on the subscription insert proves the account
  created earlier in the same transaction is rolled back. The TR-F027 boundary links the
  tests; the full adminapi suite passes in 43.8s.

- `ENG-052` (2026-10-27, `dev` @ `447c0ad`): browser role matrix for the operator/admin/super
  button set. `web/e2e/role-matrix.spec.ts` seeds operator, admin, super and `platform_admin`
  sessions into `localStorage` and, against read-only API fixtures, asserts menu visibility
  (network/operations/operators vs. platform-only branding, config, certificates, tenants) plus
  customer and RADIUS-user create/import controls, voucher print (all roles) vs. batch generation
  (admin), and ticket creation (all roles) vs. WhatsApp dispatch (admin). All 16 cases pass on
  Chromium. The run exposed a real gating leak: React Admin's default empty state renders a
  Create button from the resource's create route, so operators saw an empty-state "Create" link
  on customers/packages/subscriptions/payments and the RADIUS-user empty hint told them to
  create. Lists now pass `empty={<RoleAwareEmpty />}`, and the RADIUS empty hint has a
  restricted variant; both follow `useIsAdmin`. The server half is unchanged and still enforced
  by `TestMutationRouteRoleMatrix`; `docs/access-control-matrix.md` records the UI/API split and
  the Playwright command.

- `ENG-011` (2026-10-27, `dev` @ `6fcd5ad`): fuzz regression for defective and oversized
  RADIUS packets. A deterministic corpus of 22 named frames covers the RFC 2865 §3 length
  window (19/0/3/4097/65535 rejected, exactly 4096 accepted), truncated headers and bodies,
  attribute TLVs with length 0/1/overrun, trailing padding, malformed and valid Huawei VSAs,
  and short/duplicate/zeroed Message-Authenticator values. Three native fuzz targets
  (`FuzzParseTcpPacket`, `FuzzParseAttributesAndVendorDecoding`, `FuzzMessageAuthenticator`)
  assert no panic, framing errors only outside 20..4096, exact frame consumption, and a
  deterministic replay. A 25s/20s/20s run executed 38,362 + 78,072 + 65,807 inputs with zero
  panics and zero contract violations; the commands, invariants and limits are in
  `docs/radius-fuzz-harness.md`, and the TR-F001 boundary now links the harness.

- `ENG-024` (2026-10-27, `dev` @ `db08c5f`): rotating `web.secret` and the sealed SNMP
  credentials. The monitor stores the SNMPv2c community and SNMPv3 auth/privacy passwords as
  AES-GCM ciphertext keyed from `web.secret`, so a secret change made every target report "SNMP
  credentials unavailable". `networkmonitor.RotateSNMPSecrets` now decrypts and re-seals all
  three columns for every tenant inside one transaction; any value that cannot be authenticated
  with the old secret aborts the rotation and leaves every row unchanged. The new
  `cmd/rotate-secrets` reads both secrets from the environment, dry-runs by default by running
  the updates and rolling them back, and prints counts only. Tests cover the dry-run rollback,
  all three columns, the atomic abort (first row rewritten, second fails, nothing persists), the
  empty/identical-secret guards, and CLI stdout/stderr containing neither plaintext nor either
  secret. `docs/secret-rotation-runbook.md` documents backup, cutover, reverse rotation and the
  snapshot trap: ciphertext in a pre-rotation backup stays bound to the old secret after restore.
  The backup/blueprint/checklist docs now link the procedure.

- `ENG-023` (2026-10-27, `dev` @ `818c331`): rate limiting behind a multi-hop reverse proxy.
  Echo's `X-Forwarded-For` extractor trusts loopback, link-local and private peers implicitly,
  so once `TOUGHRADIUS_TRUSTED_PROXIES` was set, a caller reaching the process from a LAN address
  could rotate the header and get a fresh bucket per attempt; the per-IP login limiter was
  bypassable. `trustedProxyIPExtractor` now disables link-local and private implicit trust and
  keeps only loopback, so a same-host proxy still works without configuration. New unit tests in
  `internal/webserver/server_test.go` pin the trust defaults, the walk through listed private
  proxies, an unlisted hop resolving to that hop (fail-closed, shared budget), and a malformed
  header resolving to the peer; `internal/adminapi/proxy_ratelimit_test.go` drives the real
  router and shows the spoof attempt reaching the handler and being throttled on the sixth
  attempt, while a listed proxy keeps per-client budgets. Both the unit and end-to-end tests fail
  against the previous extractor. Tenant/IP isolation was already pinned by
  `TestPublicRateLimitBucketsByTenantAndIP`. `docs/public-endpoint-hardening.md` now states that
  every hop, private ones included, must be listed.

- `ENG-022` (2026-10-27, `dev` @ `e093bd0`): admin browser-session threat model. The review
  mapped login/token, browser storage, logout, CORS and CSRF against the real code. The admin
  API is a 12h HS256 bearer token kept in `localStorage` and sent as an `Authorization` header;
  there are no cookies, no refresh endpoint and no server-side logout. That makes classic CSRF
  inapplicable rather than "exploitable", which the acceptance criterion asked for explicitly.
  `internal/adminapi/auth_session_test.go` drives the real router with forged, expired,
  `alg:none`, tampered, cookie-only and wrong-scheme credentials (all `401`), proves the tenant
  comes from the signed claim despite `X-Tenant-ID`/query overrides, pins the missing
  refresh/logout routes, proves login never sets a cookie, and ratchets the frontend so
  `document.cookie`/`withCredentials` cannot appear. Limits are documented in
  `docs/admin-session-threat-model.md`: a stolen token stays valid up to 12h unless the
  operator/membership/tenant is revoked, `localStorage` is XSS-readable, and no CSP header is
  sent. No production code changed.

- `ENG-012` (2026-10-27, `dev` @ `9be97f3`): accounting Start/Interim/Stop out-of-order. The
  handlers already converged for duplicates and a delayed interim, but `UpdateStop` matched
  every accounting row with the `Acct-Session-Id`, so a NAS that reused an ID for a later
  session had the earlier session's finalized counters overwritten by the second stop. The
  repository now selects the newest row and updates it by primary key. The new
  `internal/radiusd/accounting_order_test.go` fixture drives the real handlers on GORM:
  `TestAccountingOutOfOrderStartInterimStopConverge` permutes duplicate Start/Interim/Stop plus
  a delayed interim and ends with no online row and one row holding the stop counters;
  `TestAccountingSessionIDReuseKeepsFinalizedHistory` keeps both reused sessions' rows intact
  (it fails against the previous update-every-match code); `TestAccountingSessionIDReuseIsTenantScoped`
  repeats the reuse on two tenants; `TestAccountingEventsWithoutStartDoNotInventHistory` proves
  Interim/Stop before Start create nothing. Documented in `docs/radius-retransmission.md`,
  including the limit that a Start processed after its own Stop is indistinguishable from a
  reused ID without a session-instance token.

- `ENG-010` (2026-10-27, `dev` @ `54aee93`): RADIUS retransmission and replay. The server has
  no completed-response cache, so the audit mapped what a duplicate packet actually does on
  each transport. Accounting is idempotent on `(tenant_id, acct_session_id)`:
  `TestRetransmittedAccountingPacketsHaveNoDoubleEffect` builds byte-identical Start, Interim
  and Stop packets, replays each one, and asserts one online row, one accounting row, final
  counters equal to the stop packet, and that a second tenant using the same session ID and
  username keeps its own online row. The auth duplicate window is in-process and keyed by
  tenant+username; `TestRetransmittedAccessRequestRateLimitIsTenantScoped` shows a replay in
  tenant 1 cannot throttle the same username in tenant 2. EAP state was already namespaced and
  serialized per tenant. The full matrix (per transport, cache key, side effects, RadSec
  in-flight map, the missing RFC 5080 response cache, and the one remaining duplicate effect —
  a zero-rate traffic sample from a replayed interim update) is in
  `docs/radius-retransmission.md`. No production code changed.

- `ENG-051` (2026-10-27, `dev` @ `5add7af`): website editor functional coverage. The
  "editor" is the global branding singleton: save is the publish path, reset is the only
  rollback, and the admin read is the read-only preview of the live record (there is no draft
  table). `TestBrandingRouteGuardAllowsOnlyPlatformAdmin` drives the registered routes through
  the real JWT/tenant middleware and asserts anonymous `401`, operator/admin/super-without-flag
  `403` on save, preview and reset, and success only for a super operator with `PlatformAdmin`;
  it also proves rejected saves persist nothing and reset removes the override.
  `TestBrandingPublishKeepsMarkupInert` publishes `<script>`/`onerror` payloads in the hero,
  subtitle and ticker fields and asserts the stored value is plain text, the public JSON body
  contains no raw `<script`, the decoded text round-trips unchanged, and reset returns the
  compiled defaults with no hostile text left behind.
  `TestBrandingEditorSourceRendersTextOnly` ratchets the frontend: no `.tsx` under `web/src`
  may use `dangerouslySetInnerHTML`, so branding values stay React text children. No production
  code changed; the TR-F029 coverage row in `docs/feature-checklist.md` was updated.

- `ENG-037` (2026-10-27, `dev` @ `e8da24f`): integer IDR boundaries. The invoice path
  already avoided floating point, so the remaining question was whether the edges hold.
  `TestInvoiceTotalsStayExactAtInt64Boundary` creates a `MaxInt64` package and asserts
  `Subtotal`/`Total`/`Balance`/item totals match exactly, then pays `1`, rejects a payment of
  `MaxInt64` (`ErrInvalidPayment`), and settles with `MaxInt64-1`, ending at
  `PaidAmount = MaxInt64`, `Balance = 0`, status paid without overflow.
  `TestRecordPaymentRejectsNonPositiveAmount` rejects `0`, `-1` and `MinInt64` without writing
  a payment row, and `TestSavePackagePriceBoundaries` pins the API contract: negative and
  `9223372036854775808` return `400`, `0` for a free package and `MaxInt64` both return `201`,
  and the max price round-trips without truncation. There is no manual invoice-item endpoint,
  so totals can only come from a stored package price. No production code changed; the
  invariant is documented under `docs/payment-concurrency.md`.

- `ENG-031` (2026-10-27, `dev` @ `74fa4c1`): invoice numbering under crash and concurrency.
  `NextDocumentSerial` already runs inside the document transaction, so the two required
  properties needed evidence rather than code: an aborted transaction rolls its increment back
  (retry reuses the number, no gap) and a committed invoice blocks a second number via the
  subscription/period unique index. `TestNextDocumentSerialRollsBackWithTransaction` and
  `TestGenerateMonthlyInvoicesRetryReusesRolledBackNumber` cover both on the unit path, and
  `TestPostgresInvoiceNumberingConcurrentAndRollback` races eight allocations, aborts a ninth,
  and asserts the retry receives it — this one runs in CI because SQLite has no real row locks
  (compiled locally with `-tags=integration`; skipped without `TEST_DATABASE_*`). The policy
  (per-tenant kind/period scope, no reuse after commit, what a visible gap means) is recorded in
  the new `docs/document-numbering.md`. No production code changed.

- `ENG-032` (2026-10-27, `dev` @ `5c1946a`): manual suspension ownership. The automatic
  paths already respected the invariant — `SuspendOverdueSubscriptions` only selects
  subscriptions whose status is still `active`, so it never rewrites a `manual`
  `suspension_reason` and never emits a duplicate `subscription_suspended` event, and
  `RecordPayment` reactivates only `billing_overdue` subscriptions after the last overdue
  invoice is settled. What was missing was evidence across the two transitions together.
  `TestManualSuspensionSurvivesSchedulerAndPayment` runs the scheduler then a full payment
  against a manually suspended subscription with an overdue invoice and asserts the status,
  reason, disabled RADIUS user, and the absence of both events;
  `TestManualSuspendSurvivesFullPayment` drives the operator `suspend` endpoint followed by
  the real payment handler and reaches the same state. The ownership invariant is documented
  under `docs/session-action-queue.md`. No production code changed.

- `ENG-033` (2026-10-27, `dev` @ `c89624d`): idempotent manual payment retries. `RecordPayment`
  had no retry guard, so two identical requests each added to `PaidAmount` as long as balance
  remained. Manual payments now accept an optional `Idempotency-Key` header (1–64 printable
  ASCII, same rule as public registration) stored in a new tenant-unique
  `isp_payment.idempotency_key` index. The lookup runs under the existing invoice row lock and
  before the closed/balance guards: an identical retry returns the original payment with `200` +
  `Idempotent-Replay: true` without touching money state, even after the invoice is fully paid;
  reusing the key with a different invoice/amount/method/reference returns
  `409 IDEMPOTENCY_CONFLICT`; a concurrent duplicate that loses the unique-index race is mapped to
  the same replay path. Requests without a key remain independent so two equal cash instalments
  stay two payments. The admin payment dialog sends one key per dialog session. Evidence:
  `TestRecordPaymentIdempotentRetryDoesNotDoubleBook`,
  `TestRecordPaymentRetryAfterInvoicePaidReportsDuplicate`,
  `TestRecordPaymentWithoutKeyRecordsSeparateIdenticalPayments`, and the endpoint-level
  `TestCreatePaymentIdempotencyKeyReplaysWithoutDoubleBooking`. Contract recorded in
  `docs/payment-concurrency.md`.

- `ENG-028` (2026-10-27, `dev` @ `59a429d`): PII and secret redaction on log export.
  `/admin/loki/query` returned each log line verbatim, so a password, bearer token, customer
  name, phone, e-mail or subscriber IP written by any service reached the operator response
  unchanged. The export path now classifies fields by data class (`system_logs.go`):
  secret-like keys become `[REDACTED]`, customer identifiers become deterministic
  `sha256:<16 hex>` pseudonyms, addresses keep only their IPv4 `/24` or IPv6 `/48` prefix, and
  free text is swept for `password=`/`token:` assignments, e-mails and IPv4 literals. Filtering
  still runs on the raw line, so operators keep their existing search behaviour, and the
  deterministic pseudonyms keep two entries correlatable. Rules and limits (free-text phone and
  IPv6 are not masked) are recorded in the new canonical `docs/log-redaction.md`. Evidence:
  `TestRedactSystemLogEntryRemovesSensitiveDataClasses`,
  `TestRedactSystemLogEntryPseudonymsAreStable`, `TestMaskSystemLogAddressKeepsOnlyTheNetworkPrefix`
  and the endpoint-level `TestQuerySystemLogsExportsOnlyRedactedEntries`.

- `ENG-019` (2026-10-27, `dev` @ `fc7390d`): raw-SQL tenancy ratchet. The existing inventory in
  `docs/tenant-isolation.md` already covered all nine production `Table`/`Raw`/`Exec`/`Joins`
  sites, so the gap was enforcement: nothing failed when a new unscoped query appeared.
  `TestRawSQLTenantScopeRatchet` (`internal/tenancy/rawsql_ratchet_test.go`) now parses every
  production file under `internal/`, `cmd/` and `pkg/` with `go/ast` and fails when a raw call
  neither mentions `tenant_id` in its own statement nor carries `// tenancy:unscoped: <reason>`.
  The scan is statement-scoped (a scoped query elsewhere in the function cannot mask it), the
  exemption requires a non-empty reason, and fixture subtests prove an unscoped query is
  reported while a scoped one passes. The three platform-level sites in `internal/app/app.go`
  (tenant sequence bootstrap, unique-index DDL, startup dedup) carry the annotation and stay
  documented in the inventory. No production query changed behaviour.

- `ENG-001` (2026-10-27, `dev` @ `4026637`): anonymous outer NAI for EAP-TLS. The gap was real and
  in two places: `validateIdentity` compared the certificate identity against the outer
  `state.Username`, and `stageLoadUser` resolved the account by the outer User-Name *before* EAP
  ran, so `anonymous@realm` failed either way. Now `eap.IsAnonymousNAI` recognises the RFC 7542
  §2.4 forms (`@realm`, `anonymous@realm`, bare `anonymous`); the handler adopts the certified
  identity and still rejects a mismatch for named clients (RFC 5216 §5.2), including before the
  TLS 1.3 protected success indication; `stageLoadUser` defers the lookup for anonymous-NAI EAP
  requests and `stageEAPDispatch` loads the account from the tenant-scoped EAP state via the new
  `Coordinator.AuthenticatedUsername` (refuses unsuccessful, still-anonymous, cross-tenant and
  missing state) before the plugins run. Open items: `NewTLSHandler` untrusted-CA and mismatch
  behaviour are unchanged; PEAP/TTLS keep their own inner-identity handling and are out of scope.
  Evidence: `TestIsAnonymousNAI`, `TestCoordinator_AuthenticatedUsername`,
  `TestTLSHandler_FullHandshake_AnonymousOuterNAI`,
  `TestTLSHandler_TLS13_AnonymousOuterNAI`, `TestStageLoadUser_DefersAnonymousOuterNAIForEAP`,
  plus the CI-backed `anonymous outer NAI resolves the account from the certificate` subtest in
  `test/integration/eap_tls_test.go` (runs in the integration lane). Roadmap M10.5 closed.

- `ENG-021` (2026-10-27, `dev` @ `404a3fe`): `/home` and `/portal` no longer advertise anonymous
  self-service or a checkout step; a 401 is reported as "staff login required" and an
  anonymous visitor is offered WhatsApp or the staff login. The fail-closed behaviour of
  `/public/packages`, `/public/register`, `/public/vouchers/check` and `/portal/lookup` is
  now asserted by `TestLandingSurfaceEndpointsFailClosedAnonymously`. Scope stayed inside the
  existing `Harden` rows of `docs/feature-checklist.md`.

- `ENG-017` (2026-10-27, `dev` @ `a3803b1`): the membership-revocation mechanism already
  existed (`membership_version` claim vs live `TenantMembership.TokenVersion` in
  `resolveOperatorFromContext`, bumped by `SysOpr.AfterUpdate`), so this was an audit + regression
  task rather than a missing mechanism. The gap the inventory named was that no test exercised
  the real mutation endpoints -- the existing `TestTenantContextMiddlewareRejectsRevokedMembership`
  only hits the middleware with a direct DB update. `TestPrivilegeDowngradeInvalidatesLiveToken`
  in `internal/adminapi/access_control_test.go` now proves a downgrade to `operator`, a disable,
  and a delete all reject the operator's previously-issued token on a real mutation endpoint
  (401), that the membership token version actually bumps, that the operator row and its grant
  are removed together, and that a cross-tenant token cannot read, update, or write into
  another tenant's rows (404, no persisted side effect). No production code changed; the finding
  is recorded in `docs/access-control-matrix.md` under a new "Immediate revocation (TR-F033)"
  section.

- `ENG-018` (2026-10-27, `dev` @ `38df193`): audit + regression for ambiguous-NAS and
  identical-username tenant consistency on the RADIUS side. `TestRadiusTenantIsolationForNASAndUsername`
  in `internal/radiusd/radius_test.go` proves that a duplicate NAS source IP with no tenant context
  fails closed (`ErrAmbiguousNASIP`), that tenant-scoped `GetNasForTenant`/`GetValidUserForTenant`
  resolve only the caller's tenant even when two tenants share the same NAS IP and the same
  username, and that accounting Start/Stop writes persist only inside the caller's tenant
  (a cross-tenant stop reports a miss and does not touch the other tenant's row). No production
  code changed.

Critical path from the inventory: `ENG-001 → ENG-002`, `ENG-017 → ENG-018`,
`ENG-041 → ENG-043 → ENG-045 → ECO-005`. Every `ASET` row that starts a chain
(`ENG-041`, `ENG-013`, `ENG-049`, `ENG-002`) is blocked on hardware or a host that has not
been claimed yet, so the unblocked queue is `LOKAL`-only.

## Waves

## Wave 0 — 5 item (LOKAL 4 · ASET 1)

- [x] `ENG-017` TR-F033 P1 — Audit penurunan hak anggota tenant saat token lama aktif · LOKAL · PROPOSED_VERIFICATION · commit `a3803b1`
- [x] `ENG-001` TR-F004 P1 — Audit & rancang outer anonymous NAI pada EAP-TLS 1.3 · LOKAL · VERIFIED_GAP · commit `4026637`
- [x] `ENG-018` TR-F033 P1 — Konsistensi tenant untuk NAS ambigu dan username sama · LOKAL · dep ENG-017 · commit `38df193`
- [x] `ENG-021` TR-F018 P1 — Rekonsiliasi landing/portal publik yang faktanya JWT-only · LOKAL · VERIFIED_GAP
- [ ] `ENG-041` TR-F020 P1 — Uji instalasi fresh pada VPS Ubuntu/Debian bersih · ASET · VERIFIED_GAP · **blocked: no clean VPS host claimed**

## Wave 1 — 37 item (LOKAL 30 · ASET 7)

- [x] `ENG-032` TR-F028 P1 — Invariant suspend manual tidak tertimpa bayar otomatis · LOKAL · PROPOSED_VERIFICATION · commit `5c1946a`
- [x] `ENG-019` TR-F033 P1 — Ratcheting audit query SQL raw dan joins tenancy · LOKAL · PROPOSED_VERIFICATION · commit `fc7390d`
- [x] `ENG-028` TR-F022 P1 — Penyamaran PII di logs, traces dan support evidence · LOKAL · PROPOSED_VERIFICATION · commit `59a429d`
- [ ] `ENG-055` TR-F031 P1 — State loading, stale-data dan retry pada OperationsPage · LOKAL · dep ENG-049 · PROPOSED_VERIFICATION
- [x] `ENG-011` TR-F001 P1 — Harness fuzz paket RADIUS cacat dan oversized · LOKAL · PROPOSED_VERIFICATION · commit `6fcd5ad`
- [x] `ENG-031` TR-F028 P1 — Invoice numbering tahan retry setelah crash PostgreSQL · LOKAL · PROPOSED_VERIFICATION · commit `74fa4c1`
- [x] `ENG-033` TR-F028 P1 — Idempotency pembayaran manual dengan request ulang · LOKAL · PROPOSED_VERIFICATION · commit `c89624d`
- [x] `ENG-052` TR-F016 P1 — E2E matriks tombol per role operator/admin/super · LOKAL · PROPOSED_VERIFICATION · commit `447c0ad`
- [ ] `ENG-009` TR-F003 P1 — Regression RadSec mTLS sertifikat tidak tepercaya · LOKAL · dep ENG-004 · PROPOSED_VERIFICATION
- [x] `ENG-010` TR-F001 P1 — Audit duplikasi RADIUS retransmission dan replay · LOKAL · PROPOSED_VERIFICATION · commit `54aee93`
- [x] `ENG-012` TR-F002 P1 — Test accounting Start/Interim/Stop out-of-order · LOKAL · PROPOSED_VERIFICATION · commit `9be97f3`
- [x] `ENG-022` TR-F016 P1 — Threat-model autentikasi browser dan sesi admin · LOKAL · PROPOSED_VERIFICATION · commit `e093bd0`
- [x] `ENG-023` TR-F018 P1 — Validasi rate limiting saat reverse proxy multi-hop · LOKAL · PROPOSED_VERIFICATION · commit `818c331`
- [x] `ENG-024` TR-F031 P1 — Runbook rotasi secret SNMP dan encrypted community · LOKAL · PROPOSED_VERIFICATION · commit `db08c5f`
- [x] `ENG-025` TR-F030 P1 — Audit perlindungan kredensial koneksi WhatsApp · LOKAL · PROPOSED_VERIFICATION · commit `aef2150`
- [ ] `ENG-027` TR-F031 P1 — Hardening diagnostik jaringan terhadap SSRF · LOKAL · PROPOSED_VERIFICATION
- [x] `ENG-035` TR-F027 P1 — Konsistensi status Subscription dan RadiusUser saat transisi · LOKAL · PROPOSED_VERIFICATION · commit `c070535`
- [x] `ENG-037` TR-F028 P1 — Validasi integer IDR dan batas invoice items · LOKAL · PROPOSED_VERIFICATION · commit `e8da24f`
- [x] `ENG-038` TR-F007 P1 — Voucher terminal-state concurrency regressions · LOKAL · PROPOSED_VERIFICATION · commit `7c2e06e`
- [x] `ENG-056` TR-F029 P1 — Informasi jujur untuk action portal/payment yang disabled · LOKAL · dep ENG-021 · PROPOSED_VERIFICATION · commit `a171299`
- [ ] `ENG-061` TR-F022 P1 — Stabilitas test E2E CI branch main dan dev · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-064` TR-F010 P1 — Pemeriksaan session-action queue dead-letter dan alert · LOKAL · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `ENG-053` TR-F013 P2 — Navigasi keyboard pada data-grid dan dialog kritis · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-004` TR-F003 P2 — Kesiapan rotasi sertifikat RadSec dan EAP · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-030` TR-F022 P2 — Inventory SBOM backend/frontend serta lisensi · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-008` TR-F004 P2 — Pastikan EAP-OTP tak dapat diaktifkan tanpa validator · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-029` TR-F029 P2 — PNG branding parser dan metadata validation · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-039` TR-F032 P2 — Perlindungan marker demo data saat clean-up · LOKAL · PROPOSED_VERIFICATION
- [x] `ENG-051` TR-F029 P2 — Test fungsional website editor yang masih kurang coverage · LOKAL · VERIFIED_GAP · commit `5add7af`
- [ ] `ENG-063` TR-F012 P2 — Contract testing respons API, paginasi dan error · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-013` TR-F010 P1 — Pilot RFC 5176 CoA/Disconnect pada NAS nyata · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-002` TR-F004 P1 — Matriks interoperabilitas EAP-TLS 1.2/1.3 dengan supplicant nyata · ASET · dep ENG-001 · PROPOSED_VERIFICATION
- [ ] `ENG-043` TR-F020 P1 — Drill pemulihan seluruh volume VPS dari backup · ASET · dep ENG-041 · VERIFIED_GAP
- [ ] `ENG-020` TR-F033 P1 — Latihan restore lintas tenant setelah perubahan skema · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-042` TR-F020 P1 — Simulasi kegagalan registry ketika upgrade aplikasi · ASET · dep ENG-041 · PROPOSED_VERIFICATION
- [ ] `ENG-015` TR-F025 P2 — Profil biaya LDAP bind di bawah concurrent load · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-046` TR-F020 P2 — Uji akses localhost SSH tunnel pada instalasi tanpa DNS · ASET · dep ENG-041 · PROPOSED_VERIFICATION

## Wave 2 — 25 item (LOKAL 13 · RISET 2 · ASET 7 · RISET+ASET 3)

- [ ] `ENG-058` TR-F031 P2 — Label confidence dan mapping pada NOC impact UI · LOKAL · dep ENG-055 · PROPOSED_VERIFICATION
- [ ] `ENG-016` TR-F021 P2 — Pemeriksaan berkala perbedaan fork layeh dan upstream · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-036` TR-F033 P2 — Jejak scheduler FUP terpisah per tenant sesudah perbaikan · LOKAL · dep ENG-019 · PROPOSED_VERIFICATION
- [ ] `ENG-040` TR-F011 P2 — Studi archival accounting retention dan query plan · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-054` TR-F023 P2 — Rekonsiliasi ID/EN seluruh status tagihan dan insiden · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-057` TR-F015 P2 — Audit grafik dashboard konsistensi timezone dan nominal · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-059` TR-F007 P2 — Audit import RADIUS user dengan preview dry-run saja · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-062` TR-F022 P2 — Sistem upgrade dependency dengan ratchet keamanan · LOKAL · dep ENG-030 · PROPOSED_VERIFICATION
- [ ] `ENG-003` TR-F004 P2 — Pemetaan kebijakan pencabutan sertifikat EAP-TLS · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-005` TR-F004 P2 — Evaluasi dukungan inner EAP-TTLS CHAP dan MS-CHAP · LOKAL · dep ENG-002 · PROPOSED_VERIFICATION
- [ ] `ENG-026` TR-F020 P2 — Evaluasi parameter KDF dan envelope backup · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-034` TR-F028 P2 — Pembetulan transaksi payment operator yang salah input · LOKAL · dep ENG-033 · PROPOSED_VERIFICATION
- [ ] `ENG-060` TR-F022 P2 — Studi support bundle tersensor untuk insiden operator · LOKAL · dep ENG-028 · PROPOSED_VERIFICATION
- [ ] `ECO-003` TR-F020 P2 — Model total cost of ownership deployment ISP · RISET · dep ENG-045 · PROPOSED_VERIFICATION
- [ ] `ECO-009` TR-F028 P2 — Audit peluang revenue leakage dari invoice yang sudah ada · RISET · dep ENG-031 · PROPOSED_VERIFICATION
- [ ] `ENG-049` TR-F031 P2 — UAT SNMPv3 authPriv dengan perangkat sungguhan · ASET · VERIFIED_GAP
- [ ] `ENG-045` TR-F020 P2 — Tetapkan RTO/RPO operator berdasarkan rehearsal · ASET · dep ENG-043 · PROPOSED_VERIFICATION
- [ ] `ENG-050` TR-F010 P2 — Matrix NAS/firmware teruji untuk RADIUS dan CoA · ASET · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `ENG-048` TR-F030 P2 — UAT WhatsApp pairing ke nomor uji opt-in · ASET · VERIFIED_GAP
- [ ] `ENG-014` TR-F005 P2 — Prioritas vendor VSA berbasis model NAS riil · ASET · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `ENG-044` TR-F017 P2 — Studi PostgreSQL WAL PITR untuk RPO terukur · ASET · dep ENG-043 · PROPOSED_VERIFICATION
- [ ] `ENG-047` TR-F020 P2 — Audit health dan peringatan expiry sertifikat deployment · ASET · dep ENG-041 · PROPOSED_VERIFICATION
- [ ] `ECO-001` TR-F027 P2 — Wawancara willingness-to-pay pada operator ISP kecil · RISET+ASET · PROPOSED_VERIFICATION
- [ ] `ECO-002` TR-F029 P2 — Ukur waktu tugas operator sebelum/sesudah MWX-ISP · RISET+ASET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `ECO-007` TR-F030 P2 — Pengendalian biaya telemetri dan alert whatsmeow · RISET+ASET · dep ENG-048 · PROPOSED_VERIFICATION

## Wave 3 — 23 item (LOKAL 2 · RISET 21)

- [ ] `ENG-006` TR-F004 P3 — Riset kompatibilitas TEAP terbatas untuk milestone M11 · LOKAL · PROPOSED_VERIFICATION
- [ ] `ENG-007` TR-F004 P3 — Riset kebutuhan EAP-PWD milestone M12 · LOKAL · PROPOSED_VERIFICATION
- [ ] `ECO-004` TR-F020 P2 — Evaluasi lisensi managed support vs self-hosted · RISET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `ECO-005` TR-F020 P2 — Desain paket dukungan & SLA yang realistis · RISET · dep ENG-045 · PROPOSED_VERIFICATION
- [ ] `ECO-006` TR-F027 P2 — Validasi jasa onboarding dan migrasi pelanggan · RISET · dep ENG-059 · PROPOSED_VERIFICATION
- [ ] `ECO-008` TR-F015 P2 — Cost allocation unit per NAS dan subscriber aktif · RISET · dep ECO-003 · PROPOSED_VERIFICATION
- [ ] `ECO-011` TR-F023 P2 — Kajian strategi distribusi community/open-source · RISET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `FUT-013` TR-F031 P3 — Offline read-only NOC snapshot dengan TTL dan enkripsi · RISET · dep ENG-055 · PROPOSED_VERIFICATION
- [ ] `FUT-017` TR-F029 P3 — Experimental neo-brutalist manga skin via existing tokens · RISET · dep ENG-053 · PROPOSED_VERIFICATION
- [ ] `FUT-001` TR-F013 P3 — Proof-of-concept intent search untuk perintah operator read-only · RISET · dep ENG-052 · PROPOSED_VERIFICATION
- [ ] `FUT-002` TR-F031 P3 — AI incident triage berbasis log tersensor · RISET · dep ENG-028 · PROPOSED_VERIFICATION
- [ ] `FUT-003` TR-F031 P3 — Ringkasan insiden berbukti tanpa mengklaim pelanggan offline · RISET · dep ENG-058 · PROPOSED_VERIFICATION
- [ ] `FUT-004` TR-F031 P3 — Deteksi pola anomali latency dari metrik historis bounded · RISET · dep ENG-049 · PROPOSED_VERIFICATION
- [ ] `FUT-005` TR-F031 P3 — Graph hipotesis penyebab insiden dengan uncertainty · RISET · dep ENG-058 · PROPOSED_VERIFICATION
- [ ] `FUT-006` TR-F031 P3 — Visual graph statis NAS-ODP dari relasi yang telah diisi operator · RISET · dep ENG-018 · PROPOSED_VERIFICATION
- [ ] `FUT-008` TR-F010 P3 — Preview dampak CoA terhadap sesi yang aktif sebelum aksi · RISET · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `FUT-009` TR-F028 P3 — Explanation trace billing-suspend dari rule engine existing · RISET · dep ENG-032 · PROPOSED_VERIFICATION
- [ ] `FUT-010` TR-F033 P3 — Static tenant-scope policy checker dari query AST sederhana · RISET · dep ENG-019 · PROPOSED_VERIFICATION
- [ ] `FUT-011` TR-F001 P3 — Pembangkitan regression tests untuk paket RADIUS via mutator · RISET · dep ENG-011 · PROPOSED_VERIFICATION
- [ ] `FUT-012` TR-F004 P3 — Eksperimen adversarial EAP TLS fragment scheduler · RISET · dep ENG-002 · PROPOSED_VERIFICATION
- [ ] `FUT-021` TR-F005 P3 — Knowledge base kompatibilitas firmware berbasis bukti uji · RISET · dep ENG-050 · PROPOSED_VERIFICATION
- [ ] `FUT-022` TR-F028 P3 — Event replay simulator untuk reproduksi billing dan NAS · RISET · dep ENG-032 · PROPOSED_VERIFICATION
- [ ] `FUT-023` TR-F021 P3 — Graph semantik relasi protokol RADIUS dan feature ID · RISET · dep ENG-016 · PROPOSED_VERIFICATION

## Wave 4 — 10 item (RISET 8 · RISET+ASET 2)

- [ ] `ECO-012` TR-F005 P2 — Partner interoperability kit berbasis fixture NAS · RISET · dep ENG-050 · PROPOSED_VERIFICATION
- [ ] `ECO-010` TR-F031 P3 — Riset risiko churn berbasis sinyal operasional agregat · RISET · dep ECO-002 · PROPOSED_VERIFICATION
- [ ] `FUT-007` TR-F008 P3 — Simulasi perubahan profil bandwidth tanpa menulis NAS · RISET · dep ENG-016 · PROPOSED_VERIFICATION
- [ ] `FUT-014` TR-F031 P3 — Field diagnostics mini-mode untuk teknisi yang offline · RISET · dep FUT-013 · PROPOSED_VERIFICATION
- [ ] `FUT-015` TR-F013 P3 — Adaptive contextual workspace dari tugas operator · RISET · dep ENG-053 · PROPOSED_VERIFICATION
- [ ] `FUT-016` TR-F013 P3 — Voice-assisted navigation untuk aksesibilitas internal · RISET · dep ENG-053 · PROPOSED_VERIFICATION
- [ ] `FUT-018` TR-F029 P3 — Riset visual incident card grunge yang bisa dicetak · RISET · dep FUT-017 · PROPOSED_VERIFICATION
- [ ] `FUT-020` TR-F033 P3 — Per-tenant statistik operasional privat berbasis agregat · RISET · dep ENG-017 · PROPOSED_VERIFICATION
- [ ] `FUT-019` TR-F031 P3 — Edge collector polling read-only pada jaringan terbatas · RISET+ASET · dep ENG-049 · PROPOSED_VERIFICATION
- [ ] `FUT-024` TR-F031 P3 — Prediksi kapasitas network yang dibatasi data SNMP nyata · RISET+ASET · dep ENG-049 · PROPOSED_VERIFICATION

## Suggested next batch

Ordered so each step stays executable without new hardware:

1. `ENG-053` — keyboard navigation on data grids and critical dialogs.
2. `ENG-061` — CI E2E stability on `main` and `dev`.
3. `ENG-009` — RadSec mTLS untrusted-certificate regression (follows `ENG-004`).
4. `ENG-004` — RadSec/EAP certificate rotation readiness.
