# MWX-ISP Changelog

## v0.21.0 — 2026-10-31

Changes since v0.20.0:

### Features

- answer inner CHAP and MS-CHAP inside EAP-TTLS (ENG-005)
- surface stale monitoring data and retry failed loads (ENG-055/ENG-058)
- alert and summarize dead-lettered session actions
- brand invoice and receipt documents with issuer snapshots

### Fixes and Improvements

- scope dead-letter alerts and receipt balance to their tenant
- derive the inner MS-CHAP NT hash from UTF-16LE (ENG-005)
- satisfy gosec and errcheck on tenant document assets
- register the queue summary route
- keep tenant acceptance operator usernames within the platform limit

## v0.20.0 — 2026-10-31

Changes since v0.19.8:

### Features

- add tenant asset storage and verified domains
- add section editor and public published site
- add safe draft publish version and rollback APIs
- build tenant Package Studio and catalog presentation
- expand Brand Studio identity and design settings
- add tenant package presentation model and APIs

### Fixes and Improvements

- make sqlite and FUP period fixtures CI-safe
- keep custom-domain uniqueness per hostname
- populate release_sha on the republish path
- repair branding compile, hostname validation, and package visibility
- preserve explicit public visibility on insert
- correct scoped query and menu test counts
- repair contextual workspace menu count assertion
- close M19 CI and tenant isolation gaps
- import version parser for rollback endpoint
- include tenant brand and package metadata in tenant backfill
- remove unused platform context from public surface test
- repair tenant subdomain index and update tenant menu policy tests
- repair tenant branding route, authorization and frontend

## v0.19.8 — 2026-10-30

Changes since v0.19.7:

### Fixes and Improvements

- use request URL path instead of route pattern for JWT skip list

## v0.19.7 — 2026-10-30

Changes since v0.19.6:

### Fixes and Improvements

- move tenant branding route to auth.go for proper registration

## v0.19.6 — 2026-10-30

Changes since v0.19.5:

### Fixes and Improvements

- separate tenant branding routes into dedicated file

## v0.19.5 — 2026-10-30

Changes since v0.19.4:

### Fixes and Improvements

- move tenant branding public endpoint to avoid route conflicts

## v0.19.4 — 2026-10-30

Changes since v0.19.3:

### Fixes and Improvements

- change tenant branding route path to avoid conflicts

## v0.19.3 — 2026-10-30

Changes since v0.19.2:

### Fixes and Improvements

- resolve variable shadowing in tenant branding handlers

## v0.19.2 — 2026-10-30

Changes since v0.19.1:

### Fixes and Improvements

- restore platform branding routes and add tenant branding

## v0.19.1 — 2026-10-30

Changes since v0.19.0:

### Fixes and Improvements

- add tenant branding endpoint to JWT skip list

## v0.19.0 — 2026-10-30

Changes since v0.18.0:

### Features

- add tenant branding management UI

## v0.18.0 — 2026-10-30

Changes since v0.17.0:

### Features

- add per-tenant branding system

### Fixes and Improvements

- explicitly add subdomain endpoint to JWT skip list
- add wildcard subdomain routing for multi-tenant support

## v0.17.0 — 2026-10-30

Changes since v0.16.0:

### Features

- add subdomain-based tenant routing

## v0.16.0 — 2026-10-30

Changes since v0.15.0:

### Features

- run the TEAP Phase 2 exchange (M11.9)
- serve the TEAP Phase 1 tunnel on the RFC 9930 §4.1 envelope (M11.8)
- frame the RFC 9930 §4.1 TEAP envelope (M11.7)
- open a TLS 1.3 TEAP tunnel with opt-in session tickets (M11.2)
- chain TEAP Inner Methods under the RFC 9930 §3.6 bounds (M11.5)
- compute the TEAP Crypto-Binding MAC over the §6.3 BUFFER (M11.4)
- derive the TEAP key schedule with the RFC 9427 §2.2 labels (M11.6)
- add the RFC 9930 §4.2 TLV codec and re-anchor its rows
- re-anchor the TEAP catalog onto RFC 9930 (M11.1)
- measure which per-tenant statistics a response reconstructs, and price the governance that would stop it (FUT-020)
- measure a printable grunge incident card over proven fields only (FUT-018)
- measure what a navigation-only voice layer can be asked to do (FUT-016)
- price task-grouped navigation against the shipped menu (FUT-015)
- reproduce a profile-edit wire diff without touching the cache (FUT-007)
- measure triage over sanitized logs, abstention priced against guessing (FUT-002)
- measure what a read-only intent search could resolve (FUT-001)
- measure the shipped manga-ink skin against WCAG AA (FUT-017)
- measure the real onboarding migration surface (ECO-006)
- re-derive the EAP-PWD need for M12 from the tree (ENG-007)
- survey which TEAP mechanisms the EAP server already has (ENG-006)
- audit billing exceptions against the rules that create them (ECO-009)
- derive the RFC-to-feature-to-test navigation graph (FUT-023)
- derive a static NAS-ODP graph from operator relations (FUT-006)
- reconstruct billing and RADIUS state from stored evidence (FUT-022)
- add reviewed redacted incident bundle (ENG-060)
- let admins reverse a mis-entered payment (ENG-034)
- bound KDF cost compatibility and passphrase policy (ENG-026)
- gate called govulncheck advisories behind justified exceptions (ENG-062)
- share one billing/incident status vocabulary across EN and ID (ENG-054)
- add reproducible SBOM generator and dependency inventory (ENG-030)
- reload RadSec TLS material without restart (ENG-004)
- rotate SNMP credentials when the web secret changes (ENG-024)
- idempotent manual payment retries (ENG-033)
- redact secrets, customer PII and host IPs on log export (ENG-028)
- resolve EAP-TLS identity from certificate for anonymous outer NAI (ENG-001)

### Fixes and Improvements

- add HTTP router with redirect for complete Traefik config
- remove custom network to use Coolify-managed Traefik network
- add Traefik labels for e-isp.id routing
- add Coolify-specific compose without caddy
- carry RFC 5931 as the EAP-PWD text and unname the NAI file (M12.1)
- pin two M12 claims to the code that actually proves them
- make RADIUS user import all-or-nothing (ENG-059)
- bucket series by instant windows and localize nominals (ENG-057)
- measure list, count and retention plans before changing indexes (ENG-040)
- enforce the shared list response contract (ENG-063)
- keep a sample subscription adopted by a real customer (ENG-039)
- refuse a PNG that only decodes in its header (ENG-029)
- make EAP-OTP unreachable without a validator (ENG-008)
- bound probe targets against SSRF (ENG-027)
- pin RadSec peer verification and stop failed handshakes (ENG-009)
- make list rows keyboard reachable (ENG-053)
- answer staff-gated portal actions with an info notice (ENG-056)
- confine the reachability probe to IP literals and registered inventory (ENG-027)
- report a failed WhatsApp session revoke instead of swallowing it (ENG-025)
- stop trusting forwarded headers from private peers (ENG-023)
- keep finalized accounting history on session-id reuse (ENG-012)
- reconcile public landing/portal claims that need an operator JWT

## v0.15.0 — 2026-10-30

Changes since v0.14.1:

### Features

- run the TEAP Phase 2 exchange (M11.9)
- serve the TEAP Phase 1 tunnel on the RFC 9930 §4.1 envelope (M11.8)
- frame the RFC 9930 §4.1 TEAP envelope (M11.7)
- open a TLS 1.3 TEAP tunnel with opt-in session tickets (M11.2)
- chain TEAP Inner Methods under the RFC 9930 §3.6 bounds (M11.5)
- compute the TEAP Crypto-Binding MAC over the §6.3 BUFFER (M11.4)
- derive the TEAP key schedule with the RFC 9427 §2.2 labels (M11.6)
- add the RFC 9930 §4.2 TLV codec and re-anchor its rows
- re-anchor the TEAP catalog onto RFC 9930 (M11.1)
- measure which per-tenant statistics a response reconstructs, and price the governance that would stop it (FUT-020)
- measure a printable grunge incident card over proven fields only (FUT-018)
- measure what a navigation-only voice layer can be asked to do (FUT-016)
- price task-grouped navigation against the shipped menu (FUT-015)
- reproduce a profile-edit wire diff without touching the cache (FUT-007)
- measure triage over sanitized logs, abstention priced against guessing (FUT-002)
- measure what a read-only intent search could resolve (FUT-001)
- measure the shipped manga-ink skin against WCAG AA (FUT-017)
- measure the real onboarding migration surface (ECO-006)
- re-derive the EAP-PWD need for M12 from the tree (ENG-007)
- survey which TEAP mechanisms the EAP server already has (ENG-006)
- audit billing exceptions against the rules that create them (ECO-009)
- derive the RFC-to-feature-to-test navigation graph (FUT-023)
- derive a static NAS-ODP graph from operator relations (FUT-006)
- reconstruct billing and RADIUS state from stored evidence (FUT-022)
- add reviewed redacted incident bundle (ENG-060)
- let admins reverse a mis-entered payment (ENG-034)
- bound KDF cost compatibility and passphrase policy (ENG-026)
- gate called govulncheck advisories behind justified exceptions (ENG-062)
- share one billing/incident status vocabulary across EN and ID (ENG-054)
- add reproducible SBOM generator and dependency inventory (ENG-030)
- reload RadSec TLS material without restart (ENG-004)
- rotate SNMP credentials when the web secret changes (ENG-024)
- idempotent manual payment retries (ENG-033)
- redact secrets, customer PII and host IPs on log export (ENG-028)
- resolve EAP-TLS identity from certificate for anonymous outer NAI (ENG-001)

### Fixes and Improvements

- carry RFC 5931 as the EAP-PWD text and unname the NAI file (M12.1)
- pin two M12 claims to the code that actually proves them
- make RADIUS user import all-or-nothing (ENG-059)
- bucket series by instant windows and localize nominals (ENG-057)
- measure list, count and retention plans before changing indexes (ENG-040)
- enforce the shared list response contract (ENG-063)
- keep a sample subscription adopted by a real customer (ENG-039)
- refuse a PNG that only decodes in its header (ENG-029)
- make EAP-OTP unreachable without a validator (ENG-008)
- bound probe targets against SSRF (ENG-027)
- pin RadSec peer verification and stop failed handshakes (ENG-009)
- make list rows keyboard reachable (ENG-053)
- answer staff-gated portal actions with an info notice (ENG-056)
- confine the reachability probe to IP literals and registered inventory (ENG-027)
- report a failed WhatsApp session revoke instead of swallowing it (ENG-025)
- stop trusting forwarded headers from private peers (ENG-023)
- keep finalized accounting history on session-id reuse (ENG-012)
- reconcile public landing/portal claims that need an operator JWT

## v0.14.1 — 2026-10-30

Changes since v0.14.0:

### Fixes and Improvements

- add Coolify-specific compose without caddy

## v0.14.0 — 2026-10-27

Changes since v0.13.0:

### Features

- queue manual suspension disconnects (H05)

### Fixes and Improvements

- verify installer recovery, update health and image metadata (H14)
- reject cross-tenant ticket and NAS foreign references (H01)

## v0.13.0 — 2026-10-27

Changes since v0.12.0:

### Features

- ship EN/ID locale and translate custom pages (H13)

## v0.12.0 — 2026-10-27

Changes since v0.11.0:

### Features

- add tenant-scoped customer 360 timeline (#54)
- add tenant-scoped NOC impact view for targets, ODPs and NAS (#53)
- reconcile billing suspension with durable Disconnect retries (#52)
- add encrypted envelope, manifest, dry-run and PG drill (#50)
- complete hotspot voucher revoke and lifecycle guarantees

### Fixes and Improvements

- bound SNMP counter deltas for gosec G115
- harden existing surfaces and close audit gaps
- isolate FUP scheduler per tenant with idempotent state
- replay only release metadata when dev diverged
- prefer fast-forward release sync to avoid divergent dev commits (#51)
- scope FUP scheduler queries and disconnects by tenant (#46)
- upgrade golang.org/x/net to v0.60.0 (#45)
- eliminate ineffassign in TCP reachability diagnostic (#47); fix(test): scope concurrent ticket assertions to test fixtures (#48)

## v0.11.0 — 2026-10-05

Changes since v0.10.1:

### Features

- enforce ODP port capacity and occupancy validation, and handle empty catalog state (TR-F027, TR-F032) (#43)

## v0.10.1 — 2026-10-05

Changes since v0.10.0:

### Fixes and Improvements

- validate audit IP timestamp and support IPv6/custom ports in TCP probe (TR-F012, TR-F031) (#42)

## v0.10.0 — 2026-10-05

Changes since v0.9.0:

### Features

- expose website & home editor in sidebar and public website shortcut in appbar (TR-F013) (#41)

## v0.9.0 — 2026-10-05

Changes since v0.8.4:

### Features

- ODP empty value clearing and full voucher batch printing (TR-F012, TR-F013) (#40)

## v0.8.4 — 2026-10-05

Changes since v0.8.3:

### Fixes and Improvements

- restore dashboard route at root and fix post-login redirect (TR-F013)

## v0.8.3 — 2026-10-05

Changes since v0.8.2:

### Fixes and Improvements

- enable interactive prompts by default during vps-repair (TR-F020)

## v0.8.2 — 2026-10-05

Changes since v0.8.1:

### Fixes and Improvements

- verify Caddy local HTTP proxy against /admin/ endpoint (TR-F020)

## v0.8.1 — 2026-10-05

Changes since v0.8.0:

### Fixes and Improvements

- pass stdin to backup-db invocation with bash -s (TR-F020)
- remove duplicate database backup in vps-update (TR-F020)

## v0.8.0 — 2026-10-05

Changes since v0.7.5:

### Features

- add interactive VPS TLS modes

### Fixes and Improvements

- preserve local checkout changes before updates
- make registration and ticket numbering atomic
- record creator and verify cached expiry

## v0.7.5 — 2026-10-05

Changes since v0.7.4:

### Fixes and Improvements

- make batch generation atomic and expire credentials
- resume clean install from empty orphaned volumes

## v0.7.4 — 2026-10-05

Changes since v0.7.3:

### Fixes and Improvements

- restart stopped services before install backup

## v0.7.3 — 2026-10-05

Changes since v0.7.2:

### Fixes and Improvements

- recover clean VPS installs without env
- close unverified public billing flows

## v0.7.2 — 2026-10-05

Changes since v0.7.1:

### Fixes and Improvements

- close unverified public billing flows (#31)

## v0.7.1 — 2026-10-05

Changes since v0.7.0:

### Fixes and Improvements

- recover after uninstall removes current directory

## v0.7.0 — 2026-10-05

Changes since v0.6.0:

### Features

- add online VPS repair and uninstall

## v0.6.0 — 2026-10-05

Changes since v0.5.0:

### Features

- add SSH owner CLI for VPS operations

## v0.5.0 — 2026-10-05

Changes since v0.4.1:

### Features

- add private peer gossip service

## v0.4.1 — 2026-10-05

Changes since v0.4.0:

### Fixes and Improvements

- pull all service images before startup

## v0.4.0 — 2026-10-05

Changes since v0.3.2:

### Features

- simplify VPS installation to one command

## v0.3.2 — 2026-10-04

Changes since v0.3.1:

### Fixes and Improvements

- harden VPS installer and onboarding

## v0.3.1 — 2026-10-04

Changes since v0.3.0:

### Fixes and Improvements

- ignore placeholder comments in env checks

## v0.3.0 — 2026-10-04

Changes since v0.2.4:

### Features

- import local ISP enhancements

### Fixes and Improvements

- resolve CI findings in ISP integration
- install Docker prerequisites on VPS

## v0.2.4 — 2026-10-04

Changes since v0.2.3:

### Fixes and Improvements

- install Docker prerequisites on VPS (#17)

## v0.2.3 — 2026-10-04

Changes since v0.2.2:

### Fixes and Improvements

- harden VPS updates and recovery

## v0.2.2 — 2026-10-04

Changes since v0.2.1:

### Fixes and Improvements

- allow safe backfill of unpublished tags

## v0.2.1 — 2026-10-04

Changes since v0.2.0:

### Fixes and Improvements

- set tag when publishing reusable release

## v0.2.0 — 2026-10-04

Changes since v0.1.0:

### Features

- manage revocable tenant operator memberships
- show active organization
- edit tenant billing identity
- add platform tenant console
- automate SemVer changelog and publishing
- scope admin ORM requests by tenant
- bind admin login to tenant
- add tenant catalog and legacy migration

### Fixes and Improvements

- classify commits with empty bodies
- pass publisher permissions and secrets
- grant reusable publisher issue access
- scope destructive API operations
- sync metadata onto advancing dev
- scope audit log retention per tenant
- scope retention across organizations
- round-trip tenant data in system backups
- scope shared operations controls
- preserve platform access on restore
- restrict global controls to platform admins
- scope CoA NAS resolution
- scope demo seed cleanup
- scope network monitoring and alerts
- scope billing scheduler per tenant
- isolate RADIUS authentication state
- preserve scoped upsert SQL on PostgreSQL
- satisfy tenant migration lint
- advance PostgreSQL tenant sequence
