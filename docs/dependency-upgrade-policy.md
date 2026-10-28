# Dependency upgrade policy: cadence, PR risk tiers, and the security ratchet (TR-F022 / ENG-062)

Dependencies are inventoried by [`tools/sbom`](../tools/sbom/main.go) and triaged through
the register in [dependency-inventory.md](dependency-inventory.md). Those
artefacts answer *what is shipped* and *what the scanner said*. This document
adds the other half: when dependencies are upgraded, how risky each upgrade is
allowed to be per review tier, and a gate that fails when a vulnerability the
scanner can actually reach has neither a fix nor a recorded justification.

This does not re-patch dependencies. The AUDIT-01..10 work (#45–#54) already
applied the known patches; this policy governs the process that keeps them
applied.

## 1. The ratchet: `tools/depratchet`

```sh
# On demand or at release time, from the repository root:
govulncheck -json ./... | go run ./tools/depratchet

# Or from a saved scan:
govulncheck -json ./... > /z/tmp/govulncheck.json
go run ./tools/depratchet -findings /z/tmp/govulncheck.json
go run ./tools/depratchet -now 2026-12-01 -findings /z/tmp/govulncheck.json   # check expiries
go test ./tools/depratchet/
```

The register defaults to [`docs/security-exceptions.json`](security-exceptions.json);
`-exceptions` points elsewhere.

### What decides pass and fail

`govulncheck` emits findings at three levels, and the gate treats them
differently:

| Finding level | How it appears in the stream | Gate behaviour |
| --- | --- | --- |
| Symbol (called) | First frame has a `function` | **Fails** unless the advisory has a valid exception |
| Package (imported, not called) | Single frame with `package`, no `function` | Printed as a `lead`, never fails |
| Module (required, not imported) | Single frame with only `module` | Printed as a `lead`, never fails |

This is the difference between "the dependency is present" and "the vulnerable
code is callable". A module in the graph is normal; a called vulnerable symbol
is a decision someone has to make. The gate fails only on the latter.

Exit codes: `0` every called advisory is fixed, absent or excused; `1` at least
one is neither; `2` the scan or the register could not be read (including an
empty or truncated stream, so a crashed scanner cannot look like a clean scan).

### Register rules enforced by the tool

Each entry has exactly four fields, and the tool rejects unknown fields so a
typo (`expiry` instead of `expires`) cannot silently disable a check:

```json
{
  "exceptions": [
    {
      "id": "GO-2026-6608",
      "reason": "Why this call path does not carry the risk today, in a sentence.",
      "ticket": "ENG-062",
      "expires": "2026-11-30"
    }
  ]
}
```

The gate fails when an exception:

- has no id, or duplicates another entry;
- has a `reason` shorter than a sentence (24 characters — `n/a` and `todo` are
  rejected), or no `ticket`;
- has an `expires` that is not `YYYY-MM-DD` or has passed (the expiry day itself
  is still valid);
- names an advisory that this scan does not report as **called** — that is
  register rot: either the dependency was fixed and the entry must go, or the
  advisory was never applicable in the first place.

A missing register file is an error, not an empty register: the excuse list is
part of the reviewable tree, so its disappearance is a finding of its own.

### What the gate does not do

- It does not decide exploitability. `govulncheck` prints conservative traces
  that include interface-dispatch edges which cannot be taken (the register in
  [dependency-inventory.md](dependency-inventory.md) records examples). A called
  finding is a lead that needs a sentence of human reasoning — that sentence is
  the exception.
- It does not ingest `npm audit`. The frontend is a single embedded bundle of
  production packages; npm reports package presence without call paths, so
  there is no reachability signal to gate on. npm is covered by the cadence
  below and by the release-time `npm audit --omit=dev` review.
- It does not open tickets, patch versions or run on a schedule. It is a
  command a person runs and a reviewer can re-run.
- It adds no CI lane. CI still has the single build/test gate (plus the
  existing non-blocking `security` job); the ratchet runs at release and on any
  dependency PR, where its output is attached to the PR.

## 2. Update cadence

Cadence is triggered by evidence, not by a calendar alone.

| Surface | Routine cadence | Security trigger | Verification |
| --- | --- | --- | --- |
| Go toolchain (`go.mod` `go` directive, CI `GO_VERSION`, release build) | quarterly minor bump | any called standard-library advisory: patch release within 7 days, exception max 30 days | `go build ./...`, `go test ./...`, `go test -tags=integration ./test/integration/...`, ratchet rerun with the shipping toolchain |
| Go modules | monthly batch of patch/minor updates | called advisory with a fixed version: update within 7 days, no exception | full + integration tests, `tools/sbom -check` regenerated, ratchet rerun |
| Frontend production dependencies | tied to releases, at least monthly | `npm audit --omit=dev` finding with a fix | `npm ci && npm run build`, browser smoke on the changed screen, `web.cdx.json` regenerated |
| Frontend dev tooling (ESLint, TypeScript, Playwright, Vite) | monthly | not shippable, so no advisory gate | `npm ci && npm run build`; the lockfile diff is the review surface |
| RADIUS protocol fork (`talkincode/radius` via `replace`) | per the upstream sync skill | upstream security/codec fix: sync before the next release | `go build ./...`, unit + integration, [radius-fork-upstream-audit-2026-10-29.md](radius-fork-upstream-audit-2026-10-29.md) |

Rules that apply to every row:

1. A dependency change and its register update land in the same PR. Changing a
   dependency without re-running the ratchet is an incomplete change.
2. Regenerate the SBOM in the same PR. `go run ./tools/sbom -check` is the
   release guard before tagging.
3. The scanner toolchain and the shipping toolchain must match before any
   release claim is made from a scan; the standard-library rows depend on it.
4. If a called advisory has a published fix, fix it. An exception is for
   "cannot fix yet / not actually reachable", never for "the upgrade is
   inconvenient".

## 3. PR risk tiers

The tier sets the minimum evidence; reviewers may ask for more, never less.

| Tier | Examples | Required before merge | Extra |
| --- | --- | --- | --- |
| **T0 — toolchain patch** | Go patch release, `GO_VERSION` bump | build + unit tests | ratchet rerun under the new toolchain; register trimmed of advisories the bump fixes |
| **T1 — dependency patch/minor** | `golang.org/x/*`, Echo patch, React Admin patch | build + unit tests + `tools/sbom -check` | integration tests when the module is on a protocol path |
| **T2 — protocol / crypto / auth libraries** | `crypto/tls`-adjacent modules, tlsengine inputs, RADIUS fork sync, JWT, password hashing | T1 plus integration suite and the relevant protocol acceptance test | explicit note in the PR on which RFC or threat model was re-read; rollback plan |
| **T3 — major or schema-affecting** | GORM major, Echo major, database driver, lockfile-wide frontend major bumps | T2 plus an isolated staging run and a documented rollback | separate PR; no bundling with feature work |

Tenancy, authentication and payment code must not be edited to accommodate a
dependency upgrade inside the same PR; if the upgrade forces it, the PR is T3.

## 4. Exception lifecycle

- **Who**: the reviewer who accepts the risk names themselves in the PR; the
  ticket field points at the work that removes the exception.
- **How long**: a called advisory with a fix available gets no exception beyond
  the fix window above. A called advisory with a fix only available through a
  toolchain bump gets at most 30 days. A called advisory with no published fix
  gets at most 90 days, and the reason must say why the call path is not
  exploitable today.
- **Expiry is a deadline, not a reminder**: the gate fails the day after
  `expires`. Extending means editing the register in a reviewed PR with a new
  reason — not bumping the date silently.
- **Removal**: when the dependency is fixed, upgraded or dropped, delete the
  entry in the same PR. A leftover entry fails the gate as stale, so the
  register cannot accumulate folklore.

## 5. Verification (2026-10-29)

Environment: `govulncheck@v1.9.0`, DB pulled 2026-10-27, local `go1.27.0`,
67 root packages, 74 modules plus the standard library.

1. **The gate fails for genuinely applicable advisories.** With an empty
   register:

   ```sh
   $ go run ./tools/depratchet -findings /z/tmp/govulncheck.json
   FAIL    GO-2026-6603 stdlib@v1.27.0 (net/http/internal/http2) fixed in v1.27.2 — reachable symbol net/http/internal/http2.*Transport.NewClientConn: …
           call path: net/http/internal/http2.*Transport.AddConn → net/http.*Transport.http2AddConn → … → github.com/bjo163/mwx-isp/internal/networkmonitor.systemProbe.Check (internal/networkmonitor/monitor.go:317)
   …
   depratchet: govulncheck v1.9.0, go1.27.0, scan symbol/source; 17 advisories: 11 called, 6 leads, 0 excused
   depratchet: 11 problem(s); fix the dependency or add an exception to docs/security-exceptions.json
   $ echo $?
   1
   ```

   The same scan without exceptions lists the six leads (`GO-2026-5932`,
   `GO-2026-6599`, `GO-2026-6600`, `GO-2026-6604`, `GO-2026-6860`,
   `GO-2026-6866`) and does not fail on them.

2. **The exception path is reasoned and bounded.** The eleven called advisories
   carry entries in [security-exceptions.json](security-exceptions.json), each
   with the call path that makes it applicable, why that path does not carry
   the risk today, the ticket that removes it and the expiry. With the register
   in place the same scan passes:

   ```sh
   $ go run ./tools/depratchet -findings /z/tmp/govulncheck.json
   …
   depratchet: govulncheck v1.9.0, go1.27.0, scan symbol/source; 17 advisories: 11 called, 6 leads, 11 excused
   depratchet: pass — no called advisory lacks a fix or a valid exception
   $ echo $?
   0
   ```

3. **Negative cases are automated.** `go test ./tools/depratchet/` covers: a
   called advisory without an exception fails; a valid exception passes; an
   expired exception fails; an exception stays valid on its own expiry day; a
   short reason, a missing ticket and a missing expiry each fail; a stale
   exception fails; a duplicate entry fails; a misspelled field is rejected; an
   empty stream is an error rather than a pass; package and module leads do not
   fail; distinct call paths group under one advisory while an identical
   repeated path is printed once.

## 6. Known limits

- The scan reflects the toolchain it ran under. CI pins Go 1.26 while this scan
  used 1.27.0, so the standard-library rows differ; the register's expiry forces
  the toolchain alignment before the excuses lapse.
- `govulncheck` reachability is conservative in both directions: it can report a
  path through an interface dispatch that the code cannot take, and symbol-level
  analysis does not model runtime input. The human sentence in the exception is
  the only thing that narrows this.
- Exceptions are per advisory id, not per call path. Two call paths for one
  record share one decision.
- The register is not signed or access-controlled beyond normal review; it is
  only as trustworthy as the PR that changes it.
