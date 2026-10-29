# RADIUS protocol navigation graph (FUT-023)

Research note for `FUT-023` (TR-F021): can a new developer start from an RFC and
arrive at a real test and a real source file, and where does that path break?

Scope: bounded prototype in `internal/protocolmap`, read-only over repository
text. No product code imports it, nothing is written, and no protocol behavior
changes. Acceptance criterion from the backlog item: *a reference leads to a real
test and source; a missing link is reported as UNKNOWN.*

## Question and constraints

`docs/rfcs` holds 56 protocol documents. `docs/feature-checklist.md` names 32
feature contracts with an "Existing Entry Points / Modules" column. Go comments
and import paths cite RFC numbers. Nothing joins the three, so onboarding today
means grep and folklore. The open question was whether those three artifacts
already contain a navigable graph, and how much of the reference library is
decorative.

Constraints:

- Derivation only from stored text — no manual mapping table, because a hand-kept
  edge list is the thing that rots.
- A dead end is a first-class result. Absence is reported as `UNKNOWN`, never
  inferred as "not needed".
- Deterministic output, so the graph can be diffed between runs.

## How the graph is derived

`Scan(root)` loads three inputs and `Build` joins them:

| Input | Source | Extraction |
| --- | --- | --- |
| References | `docs/rfcs/rfcNNNN-*.txt` | the four-digit number from the file name |
| Features | `docs/feature-checklist.md` table rows | ID, status, backticked entry-point paths, RFC numbers named in the row |
| Go files | every `.go` file under the root, hidden and generated directories skipped | `(?i)\brfc[\s-]*(\d{4})\b` — this catches prose (`RFC 2865 §3`), import paths (`layeh.com/radius/rfc2866`), and document links |

Edges, each labelled with its derivation:

| Edge | Via |
| --- | --- |
| reference → file | the file names the number |
| reference → feature | the feature row names the number |
| reference → feature → file | the file is covered by that feature's entry points |
| file → test file | `_test.go` sits beside a citing source |

Entry-point matching treats a declared path as covering itself and everything
beneath it, and when several rows match a file only the **most specific**
declaration wins. Without that rule, `TR-F033` — which declares
`internal/domain`, `internal/adminapi`, `internal/radiusd`, … for tenancy —
absorbed nearly every protocol file and the Features column said the same thing
on every row.

## Survey of this repository (2026-10-29)

```
References: 56 (22 linked, 34 dead-ended) · Features: 32 (20 protocol-anchored) ·
Citing Go files: 110 source, 85 test · Cited numbers without a document: 13 ·
References without test evidence: 0
```

| Finding kind | Count | What it means |
| --- | --- | --- |
| `rfc_unlinked` | 34 | stored document named by no Go file and no feature row |
| `citation_without_file` | 50 | a cited number has no document in `docs/rfcs` (13 distinct numbers) |
| `source_without_feature` | 28 | a citing file is covered by no entry-point declaration |
| `feature_without_protocol` | 8 | a feature owns protocol-package code but anchors no reference |
| `rfc_without_test` | 0 | every reference that reaches code has test evidence |

**The criterion holds in the positive direction.** RFC 2865 reaches 60 source
files and 71 test files across 17 contracts; RFC 3576 (Dynamic Authorization)
reaches `internal/radiusd/coa_service.go` and `internal/radiusd/packet_format.go`
with five contracts and seven tests including `test/integration/coa_test.go`;
RFC 5176 (CoA/Disconnect) reaches both the Admin API and `coa_service.go`. A
developer can start from a numbered standard and land on a file and a test.

**The negative result is the useful one: 34 of 56 documents (61 %) are
decorative.** They are stored, never named. The list splits into three groups:

- *Reference library on purpose* — `RFC 2284` (obsolete EAP), `RFC 2882` (NAS
  requirements), `RFC 3580` (802.1X), `RFC 5080` (implementation issues),
  `RFC 6929` (extensions). Useful context, no code obligation.
- *Planned or partially built scope* — `RFC 4186`/`RFC 4187` (EAP-SIM/AKA),
  `RFC 4764` (EAP-PKS), `RFC 5448`, `RFC 6124`, `RFC 7170` (TEAP), `RFC 7585`.
  The EAP method registry exists; these methods do not.
- *Vendor and access-network documents nobody cites* — `RFC 4679` (DSL Forum
  VSA), `RFC 6431` (Huawei IPCP), `RFC 8157`, `RFC 8772`, `RFC 5904`/`RFC 7268`
  (IEEE 802.1X attributes), `RFC 5969`/`RFC 6911`-adjacent IPv6 documents,
  `RFC 5997` (Status-Server), `RFC 6613` (RADIUS over TCP). `RFC 6613` standing
  alone while `RFC 6614` (RadSec) is linked is the clearest example: the TCP
  framing that RadSec reuses is cited nowhere.

**Thirteen cited numbers have no local document**, and they are not one problem:

- Protocol numbers the implementation genuinely depends on but the library
  lacks: `5705` (EAP key derivation, cited by `tls_handler.go`, `ttls_mschapv2.go`,
  `tlsengine/appdata.go`), `8446` (TLS 1.3), `7627` (TLS session hash),
  `9427` (EAP-FAST), `4515` (LDAP filter escape, next to the LDAP RFCs already
  stored).
- General internet RFCs used as vocabulary, correctly outside a RADIUS library:
  `1123`, `1918`, `3021`, `3079`, `3164`, `3339`, `4849`, `5424`.

**Eight features own protocol-adjacent code without naming a standard**:
`TR-F006`, `TR-F009`, `TR-F011`, `TR-F016`, `TR-F018`, `TR-F022`, `TR-F030`,
`TR-F031`. Some are billing or topology scope where no RFC applies; others,
notably `TR-F006` (authentication policy: MAC/VLAN binding) and `TR-F011`
(session control), implement behavior that RFC 2865/2866/3576 do describe.

**28 citing files have no owning feature row**, concentrated in three places that
the checklist deliberately does not claim: `test/integration/*` (14 files),
`cmd/` and `scripts/` tooling, and the research prototypes
(`internal/replay`, `internal/protocolmap`, `internal/sessionqueue`).

## Decisions

1. **Text citation counts as the link, not a curated edge list.** A comment or
   an import path is the mechanism developers already use; the graph follows it
   instead of adding paperwork.
2. **Most specific entry-point wins.** Ownership is navigation help, not
   permission. `TR-F033` remains the tenancy contract; it just stops pretending
   to be the contract for every file.
3. **Unlinked references are reported, not pruned.** `docs/rfcs` is allowed to be
   a library. The graph's job is to say which shelves nothing walks to.
4. **No product surface yet.** Rendering is a markdown string, not a page; the
   checklist stays the human-readable source of truth.
5. **The prototype stays unenforced.** Nothing in CI fails if a reference loses
   its links; the numbers are a review signal, not a gate.

## Verification

- `go test ./internal/protocolmap -count=1`: pass (0.98 s), 14 tests.
- `go build ./...`: pass.
- `golangci-lint run ./internal/protocolmap/...`: 0 issues.
- The repository survey runs in the suite (`TestScanRepositoryGraph`) and is
  deterministic: two `Scan` passes render byte-identical markdown.

Tests:

| Test | What it pins |
| --- | --- |
| `TestBuildLinksReferenceToFeatureSourceAndTest` | The full RFC → source → feature → colocated-test path, with edge labels |
| `TestBuildReportsUnlinkedReference` | A stored document nothing names becomes `rfc_unlinked`, and every detail says UNKNOWN |
| `TestBuildReportsCitationWithoutDocument` | A cited number with no file is reported for both the file and the feature row |
| `TestBuildReportsSourceWithoutFeature` | A citing file no row claims is reported |
| `TestBuildReportsFeatureWithoutProtocolAnchor` | A feature owning code in a protocol package with no anchor is reported |
| `TestBuildReportsReferenceWithoutTestEvidence` | A reference reaching untested code is reported |
| `TestBuildCountsCitingTestAsEvidence` | A citing test satisfies the evidence link |
| `TestBuildIsIndependentOfInputOrder` | Findings, stats, and edges are order-independent |
| `TestOwnersOfMatchesExactAndNestedPaths` | Directory and exact-file entry points, and specificity |
| `TestRenderMarksDeadEndsUnknownAndIsDeterministic` | UNKNOWN cells, pipe escaping, repeated render |
| `TestRenderTruncatesLongTargetLists` | A cell caps at five targets with an overflow marker |
| `TestScanReadsFixtureRepository` | Loader over a synthetic tree, including an import-path citation |
| `TestFindRootWalksUpToRepository` | Root discovery from the package directory |
| `TestScanRepositoryGraph` | Survey invariants over the real repository |

## Limits

- **Text matching over-claims.** A Go file that merely stores the string
  `rfc2865-radius-authentication.txt` counts as citing RFC 2865; the prototype's
  own fixtures appear in the RFC 2865 evidence list. The graph proves *mention*,
  not *conformance*.
- **Ownership is a path prefix.** A file under a declared directory is claimed by
  that row even when the row is only incidentally related, which is why the
  most-specific rule is load-bearing rather than cosmetic.
- **No clause-level links.** The graph connects an RFC number to a file; it does
  not know which section a function implements, so `reference-rfc` review still
  has to read.
- **Feature rows carry the vocabulary.** A row with no entry points contributes
  nothing; a renamed directory silently becomes `source_without_feature`.
- **Non-RADIUS citations inflate `citation_without_file`.** RFC 3339 or RFC 5424
  in a Go comment is not a missing RADIUS reference, so the 13-number list needs
  the human split recorded above.
