# E2E and test-suite stability check (ENG-061, 2026-10-27)

Evidence for `ENG-061` / `TR-F022`: the browser suite and the Go suites must
pass repeatedly without reusing data from a previous run, and a failure report
must be shareable. Baseline: `dev` @ `4404b1b`.

## What was run

| Suite | Command | Result |
| --- | --- | --- |
| Chromium E2E, 3 repeats (75 tests) | `cd web && npx playwright test --project=chromium --repeat-each=3 --reporter=line` | 75/75 passed, 6 workers, 1.4 min |
| Go adminapi | `go test -count=3 ./internal/adminapi` | ok, 135.6 s |
| Go radiusd | `go test -count=3 ./internal/radiusd` | ok, 93.9 s |
| Go app + tenancy | `go test -count=3 ./internal/app ./internal/tenancy` | ok, 8.5 s / 1.3 s |

Before the fixes below, `go test -count=3 ./internal/radiusd` failed on the
second and third iteration.

## Findings and fixes

### 1. Cross-iteration database leakage (real defect)

Many tests opened SQLite with a fixed shared-cache name, for example
`file:radius-voucher-activation?mode=memory&cache=shared`. The shared cache
keeps the database alive for the whole process, so the second `-count`
iteration saw the first iteration's rows: accounting tests found two ledger
rows instead of one, voucher/cache tests found pre-existing users. This is
exactly the "no cross-fixture data" criterion, and it would also mask a test
that assumes an empty database.

Fixed by giving every in-memory DSN a per-call suffix
(`common.UUIDint64()` or `time.Now().UnixNano()`):

- `internal/radiusd/radius_test.go` (7 databases)
- `internal/radiusd/retransmission_test.go` (shared by the accounting order tests)
- `internal/tenancy/scope_test.go`
- `internal/app/initdb_test.go`
- `internal/app/tenant_migration_test.go` (3 databases)

`internal/adminapi` already used `t.Name()` plus `UUIDint64()` and stayed green
through the repeated runs, which is the pattern the rest of the tree now
follows.

### 2. Cold-start navigation timeout in the browser suite

With the default 30 s Playwright navigation timeout, a cold Vite server plus
several parallel workers timed out on the first `page.goto` (Vite compiles each
route on first hit). `web/playwright.config.ts` now sets
`navigationTimeout: 60_000` and `actionTimeout: 15_000`. With that, six workers
on a cold server complete in 1.4 min.

## Isolation model

- Playwright creates a fresh browser context per test; the token, user and
  permissions fixtures are installed per context through `addInitScript`, so
  login state cannot leak between tests.
- Every spec intercepts `**/api/v1/**`; no test reaches a real backend or a
  shared database. Mutation tests are deliberately absent — server authority is
  covered by the Go matrix (`TestMutationRouteRoleMatrix`).
- Go tests use in-memory SQLite scoped to the test, and tenancy callbacks are
  registered on each connection. Tests that share a database only do so inside
  one test (for example, the accounting order harness), never across tests.

## Sharing a failure report

`npx playwright test` writes `web/playwright-report/index.html` with traces,
videos and screenshots for failures (`web/test-results/`); both directories are
gitignored. Share the HTML report or `npx playwright show-report` rather than a
terminal transcript. The Go failure output contains the failing assertion and
the offending rows.

## CI policy and limits

- CI stays a single cheap build-and-test gate: Go tests plus the web build.
  There is no E2E lane; the three-repeat command above is the on-demand gate
  before a release or after touching the browser specs.
- Only Chromium is installed in this workspace. Firefox, WebKit and the mobile
  projects are configured but need `npx playwright install` before they can be
  used as evidence.
- This run does not replace a real-device or PostgreSQL repeat matrix; the
  integration tests that need PostgreSQL remain a separate suite.
