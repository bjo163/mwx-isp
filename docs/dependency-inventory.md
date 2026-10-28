# Dependency inventory: SBOM, licences, and advisory triage (TR-F022 / ENG-030)

The repository publishes a reproducible software bill of materials for everything
it ships, plus a register that turns scanner output into investigated findings.
A scanner hit is recorded as a lead and gets a status; it is never reprinted as
"vulnerability" on its own.

## What is inventoried

| Artefact | Source of truth | Generated file |
| --- | --- | --- |
| Backend | `go.mod` / `go.sum` module graph | `sbom/go.cdx.json` |
| Frontend | `web/package-lock.json`, production dependencies | `sbom/web.cdx.json` |

The built frontend is embedded into the Go binary (`web/static.go`), so the npm
side covers only production dependencies. Dev tooling (Playwright, ESLint,
TypeScript) is pinned in the lockfile and reviewable there, but it is not
distributed and therefore not in the shipped BOM.

Both files are CycloneDX 1.5 JSON with package URLs, versions, licence
identifiers and, for Go, `go:indirect` / `go:replaces` properties. A replaced
module is listed under the path and version that is actually compiled, with the
original requirement kept as `go:replaces`.

## Generating and checking

```sh
go run ./tools/sbom          # write sbom/go.cdx.json and sbom/web.cdx.json
go run ./tools/sbom -check   # fail if the committed copies are stale
go test ./tools/sbom/        # licence mapping, replace handling, normalisation
```

The generator needs a populated module cache (it downloads modules that are in
the graph but not yet extracted) and `npm` for the frontend document.

Regenerate whenever `go.mod`, `go.sum` or `web/package-lock.json` changes. There
is no CI lane for it: the delivery policy is one cheap build/test gate, so the
BOM is refreshed as a release-time step and `-check` is the guard before
tagging. A stale file fails `-check`; nothing else reads it at runtime.

## Why the output is reproducible

Same inputs, same bytes:

- no `metadata.timestamp`, no random `serialNumber`, no tool version;
- no machine-local module cache paths;
- every component list sorted (`go` by name+version, npm by `bom-ref`);
- versions come from `go.sum` and `package-lock.json`, not from "latest";
- a licence that cannot be classified is written as `NOASSERTION`, never guessed.

A diff in the committed files therefore means the dependency set (or a licence
text) really changed. Verified on 2026-10-28: two consecutive runs plus
`-check` produced byte-identical documents.

## Licence summary (2026-10-28)

Backend, 134 modules:

| Licence | Modules |
| --- | --- |
| MIT | 65 |
| BSD-3-Clause | 44 |
| Apache-2.0 | 12 |
| BSD-2-Clause | 5 |
| MPL-2.0 | 5 |
| ISC | 2 |
| GPL-3.0-only | 1 |

Frontend, 150 production components:

| Licence | Packages |
| --- | --- |
| MIT | 135 |
| Apache-2.0 | 4 |
| BSD-3-Clause | 4 |
| ISC | 2 |
| 0BSD | 2 |
| BSD-2-Clause | 1 |
| `(MPL-2.0 OR Apache-2.0)` | 1 (`dompurify`, dual-licensed) |
| no licence field | 1 (`css-mediaquery` 0.1.2) |

Components that carry obligations beyond attribution:

| Component | Licence | Note |
| --- | --- | --- |
| `github.com/talkincode/radius` | MPL-2.0 | The RADIUS protocol library actually linked (`go:replaces layeh.com/radius`). MPL is file-level copyleft: modifications to its files must stay under MPL. |
| `go.mau.fi/whatsmeow`, `go.mau.fi/util` | MPL-2.0 | WhatsApp integration; same file-level obligation. |
| `github.com/hashicorp/go-uuid`, `github.com/hashicorp/golang-lru/v2` | MPL-2.0 | Reached through the WhatsApp dependency graph. |
| `go.mau.fi/libsignal` | GPL-3.0-only | Signal protocol library required by `whatsmeow`. Strong copyleft: a distributed binary that includes it carries source-availability obligations for the combined work. This is a licensing review item for whoever ships the WhatsApp-enabled build, not a conclusion reached here. |
| `css-mediaquery` | unknown | No licence field in the package manifest. Needs a manual read of the package before it is relied on further. |

No AGPL component is present. `css-mediaquery` is the only `NOASSERTION` on the
npm side and there is none on the Go side.

## Advisory register (govulncheck, 2026-10-28)

Scan: `govulncheck -show verbose ./...` from `golang.org/x/vuln` v1.9.0 with the
local `go1.27.0 windows/amd64` toolchain; 67 root packages, 74 modules plus the
standard library.

| GO ID | Component | Reachability reported | Status | Action |
| --- | --- | --- | --- | --- |
| GO-2026-6809 | `crypto/x509` | RadSec peer verification (`Serve` → `x509.Certificate.Verify`) | investigated — toolchain | build with go1.27.3+ |
| GO-2026-6617 | `net/http` (HTTP/2) | admin server and network monitor | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6613 | `net/http` (HTTP/1 CONNECT) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6612 | `net/http` (HTTP/2) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6611 | `net/http` (HTTP/2) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6610 | `net/http` (HTTP/2) | admin server, monitor client | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6609 | `net/http` (Range) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6608 | `mime/multipart`, `net/textproto` | imported by the admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6607 | `crypto/tls` (ECH) | TLS handshakes | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6605 | `net/http` (HTTP/1 CONNECT) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6603 | `net/http` (HTTP/2 Trailer) | admin server | investigated — toolchain | build with go1.27.2+ |
| GO-2026-6604 | `os` (Windows junctions) | imported, no call path | not affected — no reachable call | fixed by the same toolchain bump |
| GO-2026-6600 | `html/template` | imported, no call path | not affected — no reachable call | fixed by the same toolchain bump |
| GO-2026-6599 | `html/template` | imported, no call path | not affected — no reachable call | fixed by the same toolchain bump |
| GO-2026-6866 | `golang.org/x/net/html` | module required, package not imported | not affected — package unused | bump to `x/net@v0.62.0` at the next tidy |
| GO-2026-6860 | `golang.org/x/net/html` | module required, package not imported | not affected — package unused | bump to `x/net@v0.62.0` at the next tidy |
| GO-2026-5932 | `golang.org/x/crypto/openpgp` | module required, package not imported | not affected — package unused | none; the package is unmaintained by design |

How to read that table:

- The eleven standard-library rows are one root cause — an outdated build
  toolchain — not eleven independent defects in MWX-ISP code. They are fixed by
  patching Go, which is why the status is "toolchain" rather than a code change.
- `govulncheck -show verbose` prints *conservative* symbol paths. Its traces
  include interface-dispatch edges that cannot really be taken, for example
  `csv.Writer.Error` "eventually calling" `http2.chunkWriter.Write` in the
  benchmark reporter. A trace is therefore a lead to read, not proof that the
  vulnerable code is exercised with attacker-controlled input.
- The scan above used the local toolchain. CI pins `GO_VERSION: 1.26`, so its
  standard-library set is different and has to be rescanned with the toolchain it
  actually builds with; the two must be aligned before release claims are made
  from either.
- No entry here claims a CVE or an exploitability verdict. `GO-` identifiers
  point at the vuln DB record; the status column is the team's decision.

### Triage protocol for future hits

1. Record the `GO-` / `GHSA-` ID, the component and version, and the scanner
   that produced it.
2. Find the call path in the trace and read it. Classify as `not affected — no
   reachable call`, `not affected — package unused`, `fix-scheduled` (with the
   version that fixes it), or `accepted risk` (with the reason and reviewer).
3. Only `fix-scheduled` and `accepted risk` need an owner and a date; the other
   two close immediately with the evidence line.
4. Re-run the scan after any toolchain, module or lockfile change and update this
   table instead of opening a second one.

### From register to gate

The table above is the human record; [`tools/depratchet`](../tools/depratchet/main.go)
is the machine check over the same scan:

```sh
govulncheck -json ./... | go run ./tools/depratchet
```

It fails when `govulncheck` reports a called vulnerable symbol and
`docs/security-exceptions.json` has no valid entry for that advisory. Package-
and module-level findings are printed as leads and never fail. Each exception
needs a sentence of justification, an owner ticket and an expiry, and an
exception that outlives its date, points at nothing, or leaves a fixed
dependency unrescanned fails the gate. The upgrade cadence, PR risk tiers and
exception lifetimes are in
[dependency-upgrade-policy.md](dependency-upgrade-policy.md).

## Known limits

- Licence detection is a small fingerprint table over `LICENSE` / `LICENCE` /
  `COPYING` files. `NOASSERTION` means "not classified — read it", not "no
  licence". Dual licences are reported as the expression the package declares.
- The Go document covers the module graph, a superset of the packages linked into
  the binary. The scanner's `./...` picture is the narrower, compiled view.
- The standard library and the toolchain are not components in `go.cdx.json`;
  their advisories are tracked in the register above instead.
- The npm document is what `npm sbom --omit=dev` reports. A package that ships
  without a licence field (`css-mediaquery`) stays that way in the BOM until the
  package metadata or the dependency changes.
