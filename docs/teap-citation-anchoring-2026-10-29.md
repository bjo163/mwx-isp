# Re-anchoring the TEAP catalog to the current specification (M11.1)

Date: 2026-10-29 · Tag: LOKAL (research + vendoring) · Feature: TR-F004 · Milestone: M11
Prototype: `internal/teapreadiness` · Tests: `go test ./internal/teapreadiness/`
Supersedes the citation basis of: [teap-m11-readiness-2026-10-29.md](teap-m11-readiness-2026-10-29.md)

## Question

The M11 survey scanned the product code against the 22 mechanisms of
`docs/rfcs/rfc7170-teap.txt`. Before any of its build lists could be scheduled,
one thing had to be checked first: whether that document is still the
specification. It is not — RFC 9930 (February 2026) obsoletes RFC 7170, and it
updates RFC 9427, which carries the TLS 1.3 key derivations TEAP needs. Neither
text was in the repository, and the survey had said so about exactly two of its
rows ("named but cannot be quoted").

The bounded question for this round is therefore not whether M11 is worth doing,
but what changes in the survey once the requirements are read from the document
that is currently normative. The deliverable is the two vendored texts plus the
re-anchored catalog; it is not a TEAP implementation, not a config option, and
not a claim about any client.

## What counts as evidence

The survey's three rules carry over unchanged: comments are stripped before a
marker is matched, test files and `testdata` are excluded, and matching uses word
boundaries so `msaccteaptype_del` does not prove TEAP. This round adds two:

1. **A citation names a document the repository carries.** The gate
   `TestEveryCitationIsQuotableHere` used to allow `Document: ""` when a
   specification was not vendored, provided the note said so. RFC 9930 and
   RFC 9427 are now vendored, so the exemption is gone and an empty document is a
   failure.
2. **A requirement is not cited from an obsoleted document.**
   `TestNoCitationPointsAtAnObsoletedSpecification` fails if any mechanism cites
   RFC 7170. The opposite failure mode is also pinned:
   `TestObsoletedSpecificationIsStillArchived` requires RFC 9930's own header to
   say `Obsoletes: 7170` and requires the old text to remain in `docs/rfcs`, so
   the re-anchoring cannot be quietly achieved by deleting the evidence that the
   survey was once run against RFC 7170.

## Vendoring

| File | Bytes | Lines | Source |
| --- | --- | --- | --- |
| `docs/rfcs/rfc9930-teap-version-1.txt` | 250,921 | 5,650 | `https://www.rfc-editor.org/rfc/rfc9930.txt` |
| `docs/rfcs/rfc9427-tls-based-eap-types-tls13.txt` | 50,352 | 1,011 | `https://www.rfc-editor.org/rfc/rfc9427.txt` |

Both were normalised to the archive's existing convention (CRLF, no BOM — 55 of
the 56 files already in `docs/rfcs` use it) rather than kept byte-for-byte as
served. `internal/eappwdneed`'s vendoring audit reads every
`rfcNNNN-slug.txt` file and fails on a filename/header number mismatch; it
passes, and its RFC 7170 control row is unchanged because that file still
carries the document its name promises.

## Quote reconciliation

Every one of the 16 catalog rows that cited RFC 7170 was re-checked against
RFC 9930 by whitespace-collapsed substring match — the same normalisation the
gate uses.

- 15 of the 16 quotes appear verbatim in RFC 9930. Section numbering for the TLV
  rows is unchanged (`4.2.2`, `4.2.3`, `4.2.4`, `4.2.10`, `4.2.11`, `4.2.12`,
  `4.2.13`, `4.2.14`, `4.1`), so those citations moved document-only.
- 4 rows needed a section correction because RFC 9930 relocated the prose: the
  first-Request sentence and the version-verification sentence are §3.1 (Version
  Negotiation), not §4.1/§4.2.13; "Phase 2 MUST always end with a Crypto-Binding
  TLV" is §3.6; and the chaining sentence is §A.26, which points at the normative
  §3.6.2.
- 1 quote does not appear in RFC 9930: "PAC TLV Format". The reason is the
  finding below, not a renumbering.

## What the current text changes

**The PAC is deprecated, so `pac-tlv` was the wrong build item.** RFC 9930 §2:
"[RFC7170] defined a Protected Access Credential (PAC) to mirror EAP-FAST
[RFC4851]. However, implementation experience and analysis determined that the
PAC was not necessary. Instead, TEAP performs session resumption using the
NewSessionTicket message as defined in Sections 2.1.2 and 2.1.3 of [RFC9190]. As
such, the PAC has been deprecated." RFC 9427 §2.2 says the same from the
transport side. The survey had put the PAC in "optional but load-bearing for
resumption" and counted PAC-Key/PAC-Opaque/PAC-Info as work to do. That work is
now scheduled against the wrong mechanism.

The replacement row is `session-resumption`, and it is the second blocked
mechanism, not a missing one: `internal/radiusd/plugins/eap/tlsengine/engine.go:140`
sets `SessionTicketsDisabled: true` on purpose, with the comment that EAP-TLS
does not implement resumption and RFC 9190 §2.1.3 makes it optional. That
decision is correct for EAP-TLS and is exactly what a TEAP tunnel has to
revisit. M11 therefore begins with two deliberate TLS 1.3 choices in the shared
engine, not with a handler.

**The "reusable inner AVP codec" claim was wrong.** The survey listed
`inner-avp-codec` as ready on the strength of `parseTTLSAVPs` /
`encodeTTLSAVP`. RFC 9930 does not carry RADIUS AVPs inside TEAP: §4.2.8 defines
a Vendor-Specific TLV that nests vendor-defined TLVs, adds M/R handling rules,
and requires that a vendor TLV carrying an authentication protocol define MSK and
EMSK. The TTLS reader is the nearest precedent, not the encoding TEAP needs, so
the row is `vendor-specific-tlv` and it is missing. Reusing the survey's own
rule — a mechanism counts as implemented only when a Go identifier proves that
mechanism — keeping it as ready would have been the one verdict in the table
that a comment, rather than behaviour, supported.

**Chaining is bounded by the specification, not only by our state machine.** The
survey's note said our tunnel state machines run one inner method per session.
True, and RFC 9930 §3.6.5 narrows what may be chained even once that is fixed:
inner methods MUST crypto-bind their own keys to the tunnel, a tunneled method
(PEAP, EAP-TTLS, EAP-FAST) MUST NOT be nested inside TEAP, inner-method
resumption MUST NOT be permitted, and the pair the document names as widely
supported is EAP-TLS with EAP-MSCHAPv2. §3.6.5 also states that this is what
updates RFC 9427.

**A password carried as a TLV yields no key material.** RFC 9930 §4.2.8 names
Basic-Password-Auth-Req as the case where MSK and EMSK "cannot be derived from
cryptographic primitives" and MUST be set to zero; RFC 9427 §2.2 turns that into
32 zero octets for the inner session key. The roadmap's "certificate plus
password" is therefore only a key-producing combination when the password leg is
an inner EAP method; as a TLV it changes what the Crypto-Binding is computed
over. RFC 9930 §5.1 lists which combinations it actually documents as working,
and the row now points there instead of implying the rest was settled.

**The key-derivation requirement is quotable, and it is strict about length.**
`rfc9427-key-derivation` moves from "named, cannot be quoted" to a §2.2 citation
(`IMSK[j] = TLS-Exporter("TEAPbindkey@ietf.org", secret, 32)`). RFC 9427 §2.1
also warns that unlike TLS 1.2, the exporter's output depends on the length
passed to it, so requesting more and truncating produces a different key rather
than a shorter one. Note for the implementer: the existing pure-EAP-TLS path
requests 128 octets and slices, which is what RFC 9190 §2.3 prescribes for that
method; TEAP's own derivations in RFC 9427 §2.2 ask for exact lengths (32, 40,
60, 64), so the pattern must not be copied.

## Catalog before and after

| | before (RFC 7170 basis) | after (RFC 9930 basis) |
| --- | --- | --- |
| Mechanics | 22 | 22 |
| ready | 7 | 6 |
| blocked | 1 | 2 |
| missing | 14 | 14 |
| rows citing a document the repository does not carry | 2 | 0 |

Two rows changed identity rather than moving between buckets: `pac-tlv` (missing)
became `session-resumption` (blocked), and `inner-avp-codec` (ready) became
`vendor-specific-tlv` (missing). The survey's conclusion survives the move — TEAP
is a build project, not a method switch — with `missing` (14) still ahead of
`ready` (6).

## Verification

- `go test ./internal/teapreadiness/` — 24 tests plus 5 subtests pass, including
  the three added this round: the obsoleted-citation gate, the archive-retention
  gate, and the ticket-pin verdict for `session-resumption`. (The survey's "18
  tests" counted `readiness_test.go` only; `scan_test.go` carries the rest.)
- `go test ./internal/eappwdneed/` — the vendoring audit passes over 58
  specification files: no filename/header mismatch, and its RFC 7170 control row
  unchanged.
- The quote reconciliation above was produced by matching every catalog quote
  against the collapsed text of both documents, not by reading them side by side
  and trusting the eye.
- `go build -p 1 ./...` and `go test -p 1 -count=1 ./internal/...` pass; nothing
  imports `internal/teapreadiness`, so no product behaviour changed.

## Residual limits

- The re-anchoring moves citations and two verdicts. It does not size the work or
  change the 14 absent mechanisms other than the two rows named above.
- RFC 9930 §5.1's claims about which combinations "work" are the document's own;
  this round adds no measurement of a peer, and the `Unknown` list still sends
  client behaviour to the `ENG-002` hardware gate.
- `docs/rfcs/rfc7542-eap-pwd.txt` was still mislabeled when this round ran (it is
  the Network Access Identifier, not EAP-PWD; the EAP-PWD specification is
  RFC 5931). It was deliberately not touched here and was recorded as `M12.1`,
  which corrected it later the same day: RFC 5931 is now carried and the NAI file
  is named `rfc7542-network-access-identifier.txt`.
- RFC 9930 is 5,650 lines of which this round quoted about a dozen phrases. The
  catalog is a readiness inventory, not a conformance checklist; a TEAP
  implementation still has to read the whole document.
