# RADIUS mutation regression corpus (FUT-011)

Research note for `FUT-011` (TR-F001): can the existing packet parser be
guarded by a generated, version-controlled mutation corpus, so that a parser
change which silently reclassifies a malformed frame fails the build?

Scope: bounded experiment inside `internal/radiusd`. No product code is
imported or changed, and the generator never runs against a live NAS.

## Question and constraints

ENG-011 delivered a fixed 22-frame corpus plus native fuzz targets. Fuzzing
finds inputs but keeps discovered failures only when it crashes, and Go fuzz
crashes are written to `testdata/fuzz` in a format meant for the fuzz target,
not for review. The open question was whether the existing builders can drive a
*deterministic mutation* pass whose outcome is pinned per mutant, so that an
accidental outcome change is visible in a pull request diff instead of waiting
for the next fuzz run.

Constraints used for the experiment:

- Mutants derive only from frames the current corpus already declares valid, so
  a "packet" outcome on a mutant means the parser recovered from damage rather
  than rejecting a frame it was never meant to accept.
- Mutation is positional, not random: every seed yields the same mutants in the
  same order on every machine.
- The corpus stays reviewable: seeds are capped at 512 bytes, and the 4096-byte
  boundary frame is excluded because ENG-011 already covers it directly.

## Generator

`internal/radiusd/mutation_corpus_test.go` walks `radiusFrameCorpus()`, keeps
the entries whose expectation is `expectPacket` and whose frame is at most
`mutationSeedLimit` (512 bytes), and applies seven mutation kinds:

| Kind | Mutation | Physical defect it models |
| --- | --- | --- |
| `flip_bit` | XOR `0x01` on the code, identifier, or last byte | Bit rot in transit |
| `truncate` | Cut to `len/2`, `len-1`, or `len-4`, length field untouched | Lost TCP segment on RadSec |
| `declared_length` | Rewrite bytes 2..3 to 0, 19, 20, 4097, `len-1`, `len+1` | Hostile or buggy length field |
| `attribute_length` | Rewrite the first attribute's Length to 0, 1, 255 | Broken TLV encoder |
| `duplicate_attributes` | Append the attribute list again, fix Length | Client retransmit folded into one frame |
| `drop_byte_keep_length` | Remove the last byte, keep Length | Off-by-one frame cut |
| `drop_byte_fix_length` | Remove the last byte, shrink Length to match | Short but self-consistent frame |

Each mutant is parsed with `parseTcpPacket` and classified into one of four
outcomes:

| Outcome | Meaning |
| --- | --- |
| `packet` | Parsed into a packet; code and attribute count are recorded |
| `malformed_frame` | Rejected by the framing rule (`errMalformedFrame`) |
| `short_read` | The reader ran out of bytes (`io.EOF` / `io.ErrUnexpectedEOF`) |
| `bad_attributes` | The frame was readable but an attribute failed |

## Corpus

`testdata/radius_mutation_corpus.json` holds 137 mutants generated from 8 valid
seeds:

| Outcome | Cases |
| --- | --- |
| `packet` | 41 |
| `short_read` | 37 |
| `bad_attributes` | 34 |
| `malformed_frame` | 25 |

| Kind | Cases |
| --- | --- |
| `declared_length` | 47 |
| `flip_bit` | 24 |
| `truncate` | 24 |
| `attribute_length` | 21 |
| `duplicate_attributes` | 7 |
| `drop_byte_keep_length` | 7 |
| `drop_byte_fix_length` | 7 |

Every outcome class and every kind is populated, so no assertion family is
vacuous. Two observations worth keeping:

- `packet` appears because several mutations are semantically harmless — a bit
  flip in an unused attribute value or a duplicated attribute list still yields
  a frameable packet. That is correct behavior, and pinning it stops a future
  parser from becoming arbitrarily strict.
- `duplicate_attributes` and the two drop-byte kinds produce only 7 cases each
  because they need a seed with a non-empty attribute list and at least 21
  bytes; eight seeds still give every kind real coverage.

## Golden workflow

```bash
go test ./internal/radiusd -run 'TestRadiusMutation' -count=1
```

A parser change that flips any outcome fails
`TestRadiusMutationCorpusMatchesGolden` with the generated and stored case
lists side by side. Intended changes regenerate the golden:

```bash
go test ./internal/radiusd -run 'TestRadiusMutation' -count=1 \
  -update-radius-mutation-corpus
```

The regeneration is a review artifact: the diff shows which bytes changed and
which outcome moved, which is exactly the reviewer context a fuzz crash file
does not provide.

## Decisions

1. **Generators mutate valid seeds only.** Mutating already-invalid frames
   would spend cases re-proving that the parser rejects garbage.
2. **Outcome, not error text, is the contract.** Error strings are internal; the
   four outcome classes match the invariants ENG-011 already asserts, so the
   two suites cannot drift apart.
3. **Golden file is regenerated explicitly, never on demand.** A stale golden
   fails; it does not self-heal, because self-healing would erase the signal.
4. **No timing or allocation assertions.** The harness pins classification
   only; performance regressions stay with benchmarks.
5. **No new parser entry point.** The generator reuses `parseTcpPacket` and
   `checkRadiusFrameContract`, so the mutation corpus cannot test a path that
   production does not use.

## Verification (2026-10-29)

- `go test ./internal/radiusd -run 'TestRadiusMutation' -count=1`: pass (3.6 s),
  137 subtests.
- `go test ./internal/radiusd -count=1`: pass (42.3 s).
- `go build ./...`: pass.
- `golangci-lint run ./internal/radiusd/...`: no findings in
  `mutation_corpus_test.go` (three pre-existing findings remain in
  `accounting_order_test.go` and `retransmission_test.go`, untouched here).

Tests added:

| Test | What it pins |
| --- | --- |
| `TestRadiusMutationCorpusMatchesGolden` | The stored mutants and their outcomes; replays each case and re-checks the frame contract |
| `TestRadiusMutationGeneratorIsDeterministic` | Two generations are byte-identical, so golden diffs contain no churn |
| `TestRadiusMutationGeneratorCoversEveryKind` | Every mutation kind stays populated and at least four seeds survive the filter |

## Limits

- The corpus covers frame parsing and attribute framing only. It does not enter
  the EAP state machine, the UDP listener, or authentication policy.
- The mutation set is positional and therefore finite. A fuzzer still covers
  combinations this generator never composes.
- The golden records classifications produced by the current parser. If an
  outcome is *wrong* today, the golden preserves the wrong answer; the file is a
  regression lock, not an oracle. ENG-011's RFC-derived assertions remain the
  source of truth for what an outcome *should* be.
- Seeds are capped at 512 bytes, so mutants never exercise the 4096-byte
  boundary; that path stays with ENG-011.
