# TEAP readiness for milestone M11 (ENG-006)

Date: 2026-10-29 · Tag: LOKAL (research) · Feature: TR-F004 · Milestone: M11
Prototype: `internal/teapreadiness` · Tests: `go test ./internal/teapreadiness/`

> **Citation basis superseded the same day.** This survey read its requirements
> out of `docs/rfcs/rfc7170-teap.txt`, which RFC 9930 obsoletes. M11.1 vendored
> RFC 9930 and RFC 9427 and re-anchored every citation in the catalog, which
> moved two verdicts: `inner-avp-codec` (ready) became `vendor-specific-tlv`
> (missing) and `pac-tlv` (missing) became `session-resumption` (blocked, pinned
> by `tlsengine/engine.go:140`). The report below is kept as the record of what
> was measured against RFC 7170; the current numbers are `Mechanics: 22 · ready 6
> · partial 7 · blocked 2 · missing 7`, and the reasons are in
> [teap-citation-anchoring-2026-10-29.md](teap-citation-anchoring-2026-10-29.md)
> and, for the seven partial rows, in M11.3's codec
> [`internal/radiusd/plugins/eap/teaptlv`](../internal/radiusd/plugins/eap/teaptlv/teaptlv.go).
> Partial means the §4.2 wire format can be encoded and decoded but nothing on the
> serving path calls it — no Compound MAC, no handler — so it is not a claim that
> TEAP works.

## Question

`docs/roadmap.md` lists M11 as "TEAP tunneled authentication" and
`docs/feature-checklist.md` marks it Planned (P3) with the caution that "the
client ecosystem is thinner than PEAP". Before anybody schedules that work, the
bounded question is what this repository already provides for TEAP and what it
does not — not whether TEAP is a good idea.

The inventory item is `ENG-006` and it is `LOKAL`, so the deliverable is a study
plus a read-only prototype. It is not a TEAP implementation, not a config option,
and not a claim about any vendor's supplicant.

## What counts as evidence

A mechanism counts as implemented only when a Go identifier in product code
proves it. Three rules make that meaningful:

1. **Comments are stripped before matching.** `peap_inner.go` contains the string
   "Cryptobinding", and the sentence around it says unknown TLVs *including*
   Cryptobinding "are skipped". Reading the RFC and writing about it is not the
   same as doing it, so `TestCommentMentionsAreNotEvidence` asserts the word is
   gone from the scanned tree, and that the only code supporting the Crypto-Binding
   row is the `teaptlv` codec — never that comment.
2. **Test files and `testdata` are excluded.** A test can define exactly the
   identifier a verdict looks for, in a file the server never imports.
   `TestScanReadsOnlyProductCode` proves a `_test.go` and a `testdata` fixture
   that both declare `TypeTEAP = 39` leave the verdict at `missing`.
3. **Word boundaries, not substrings.** The generated Microsoft vendor code
   spells `msaccteaptype_del`, which contains the letters "teap".
   `TestSubstringLookalikesDoNotProveTEAP` keeps that from becoming support for
   the method.

Requirements come from the specifications in `docs/rfcs`, and every citation
carries a phrase that must appear verbatim in that file. Two mechanisms are
sourced to RFC 9427 and RFC 9930, whose text this repository does not carry: they
are named with their RFC numbers, their notes say the clause cannot be quoted
locally, and `TestEveryCitationIsQuotableHere` enforces exactly that split.

## Prototype

| File | Role |
| --- | --- |
| `internal/teapreadiness/readiness.go` | The 22-mechanism catalog with its citations, the status rules, and the report types. |
| `internal/teapreadiness/scan.go` | Reads the three product directories, strips comments, drops tests, and fails if a directory is missing. |
| `internal/teapreadiness/render.go` | Renders the report as reusable / partly present / blocked / to-build tables with file:line evidence. |

`Scan` refuses to run against a root where `internal/radiusd/plugins/eap`,
`internal/radiusd` or `internal/app` is absent. An empty tree would make all 22
mechanisms look missing, and "nothing exists" is the most convincing thing this
package could get wrong.

Statuses are derived, never asserted: `ready` when every marker is found and
nothing contradicts it, `blocked` when contradicting code is found even though
the markers exist, `partial` when only some markers are found, `missing` when
none are.

## Verdict

`Mechanics: 22 · ready 7 · partial 0 · blocked 1 · missing 14`

Reusable now, with the line that proves it:

| mechanism | clause | evidence |
| --- | --- | --- |
| `tls-fragmentation` | RFC 3748 | `internal/radiusd/plugins/eap/handlers/tls_handler.go:308` |
| `tunneled-eap-framing` | RFC 7170 §4.2.10 | `internal/radiusd/plugins/eap/handlers/peap_inner.go:277` |
| `inner-avp-codec` | RFC 9930 | `internal/radiusd/plugins/eap/handlers/ttls_avp.go:62` |
| `result-tlv` | RFC 7170 §4.2.4 | `internal/radiusd/plugins/eap/handlers/peap_inner.go:389` |
| `exported-keying-material` | RFC 7170 §4.2.13 | `internal/radiusd/plugins/eap/tlsengine/appdata.go:125` |
| `protected-success` | RFC 9190 | `internal/radiusd/plugins/eap/handlers/tls_tunnel.go:170` |
| `anonymous-outer-identity` | RFC 4372 | `internal/radiusd/plugins/eap/coordinator.go:331` |

Blocked by a decision already taken:

```
tls13-outer-tunnel  RFC 9190  contradicted by
  internal/radiusd/plugins/eap/handlers/tls_config.go:257  MaxVersion: tls.VersionTLS12
```

EAP-TLS can reach TLS 1.3 through the configured minimum, but the tunneled
providers pin the session to TLS 1.2. TEAP's outer tunnel is the TLS 1.3 form
(RFC 9190), so the work is a new config provider, not a config edit. This is the
one item where the tree both supports and contradicts the mechanism, which is why
`blocked` is separated from `missing`.

To build (14), grouped by what they cost:

- **Naming, no design needed**: `teap-eap-type` (type 39 is absent — the declared
  constants stop at `TypeTLV = 33`), `teap-method-name`, `teap-handler-wiring`
  (a constructor plus a case in the identity-phase name switch in
  `coordinator.go`), `teap-config-choice` (the `radius.EapMethod` enum).
- **The TLV layer TEAP is built from**: `eap-payload-tlv`,
  `intermediate-result-tlv`, `identity-type-tlv`, `authority-id-tlv`,
  `basic-password-auth-tlv`, `teap-version-negotiation`. PEAP and TTLS put a bare
  inner EAP packet in the tunnel; TEAP wraps every exchange in TLVs and negotiates
  a version octet first.
- **The two mechanisms that decide whether M11 is worth scheduling**:
  `crypto-binding-tlv` (RFC 7170 §4.2.13: "Phase 2 MUST always end with a
  Crypto-Binding TLV" — nothing computes or verifies one) and
  `inner-method-chaining` (the machine-then-user sequence the roadmap calls
  certificate plus password; the tunnel state machines run one inner method per
  session).
- **Optional but load-bearing for resumption**: `pac-tlv` — no PAC-Key,
  PAC-Opaque or PAC-Info exists, so every TEAP handshake would be a full one.
- **Crypto agility**: `rfc9427-key-derivation`, named but not quotable here.

## Decisions

1. **A mechanism is implemented when code proves it, not when the tree mentions
   it.** That is the whole difference between `crypto-binding-tlv` being `missing`
   and being `ready`, and the comments in `peap_inner.go` are the trap.
2. **`blocked` is not `missing`.** The TLS 1.2 pin is evidence of an existing
   decision that M11 has to revisit; reporting it as absent would hide the fact
   that someone chose 1.2 on purpose.
3. **Requirements are only quoted from documents this repository carries.** RFC
   9427 and RFC 9930 are named and flagged, not paraphrased from memory.
4. **No client-compatibility claim.** `TestNoVerdictClaimsClientCompatibility`
   forbids the words Windows, macOS, iOS, Android, wpa_supplicant and
   Interoperability in any title or note. The five `Unknown` entries say what a
   repository scan cannot answer and what each one needs instead — supplicant
   hardware (the `ENG-002` gate), a real TEAP peer, the pilot NAS (`ENG-013`).
5. **The scan reads three named directories.** A walk over the whole repository
   would find TEAP in `docs/` and report it as implemented.
6. **Stay a prototype.** Nothing imports it, nothing writes, and no config option,
   handler or endpoint is added. The roadmap keeps M11 Planned.

## What this means for M11

The reusable surface is real but it is the transport: framing, fragmentation, the
tunnel, the Result TLV, key export. Everything that makes TEAP a distinct method —
the TLV conversation, Crypto-Binding, chaining, PAC provisioning — is absent, and
the TLS 1.3 pin must be revisited first. `TestTotalsAccountForEveryMechanism`
asserts `missing > ready` for exactly this reason: M11 is a build project, not a
method switch, which matches the checklist's own advice to prioritize it only once
the client environment is fully controlled.

## Verification

- `go test ./internal/teapreadiness/` — 18 tests plus 5 subtests pass, including
  the negative cases (comment-only mention, test-file definition, substring
  lookalike, lookalike identifier, missing scan directory).
- `golangci-lint run ./internal/teapreadiness/...` — 0 issues.
- `go build ./...` unaffected; `grep -rl "internal/teapreadiness"` outside the
  package returns nothing, so the prototype is inert for the product.

## Residual limits

- The catalog lists mechanisms, not effort; nothing here sizes the work.
- Absence is claimed only against identifiers an implementer would naturally
  write. A TEAP PAC implementation named something else would not be found — and
  would also not exist today.
- `rfc9427-key-derivation` and the AVP-in-TEAP question cannot be cited locally
  until those texts are vendored under `docs/rfcs`.
- Client behavior remains unmeasured; that is `ENG-002`, blocked on hardware.
