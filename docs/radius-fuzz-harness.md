# RADIUS packet fuzz harness

Closure evidence for `ENG-011` (TR-F001): a corpus-based, fixed-budget
regression harness for defective and oversized RADIUS packets, plus native Go
fuzz targets for deeper exploration.

## Contract sources

- **RFC 2865 §3** (`docs/rfcs/rfc2865-radius-authentication.txt`): the `Length`
  field covers Code, Identifier, Length, Authenticator, and Attributes. The
  minimum length is 20, the maximum is 4096, a packet shorter than its `Length`
  field must be silently discarded, and bytes beyond `Length` are padding that
  must be ignored.
- **RFC 2865 §5**: an attribute is `Type, Length, Value` with `Length >= 2`; a
  length that runs past the packet is malformed.
- **RFC 2865 §5.26**: a Vendor-Specific attribute carries a four-octet vendor ID
  followed by vendor TLVs. Nested TLVs with impossible lengths must be ignored,
  not trusted.
- **RFC 3579 §3.2**: at most one Message-Authenticator attribute is allowed on a
  request; a value that is not sixteen octets, a duplicate, or a mismatch is
  invalid.

## Harness

`internal/radiusd/fuzz_packet_test.go` contains:

| Entry point | Scope |
| --- | --- |
| `TestRadiusPacketCorpusContract` | Deterministic table of 22 named frames: minimum/oversized/truncated lengths, attribute length 0/1/overrun, 4096-byte boundary, trailing padding, malformed and valid vendor TLVs, and Message-Authenticator variants. |
| `TestRadiusCorpusVendorAndMessageAuthenticatorDecoding` | Semantic outcome for the security-relevant corpus entries (Huawei VSA value, malformed nested TLV ignored, short/duplicate/zeroed Message-Authenticator rejected). |
| `FuzzParseTcpPacket` | The RadSec frame reader (`parseTcpPacket`): length handling, truncation, attribute parsing, and determinism across replays. |
| `FuzzParseAttributesAndVendorDecoding` | Attribute TLV parsing, Huawei/Mikrotik VSA getters, Message-Authenticator reader, packet re-encoding, and `verifyMessageAuthenticator` classification. |
| `FuzzMessageAuthenticator` | HMAC-MD5 computation over arbitrary wire bytes and secrets: fixed 16-byte output, input not mutated, deterministic. |

Invariants asserted for every input:

1. No panic.
2. A framing error (`errMalformedFrame`) is only produced when the declared
   `Length` is outside 20..4096.
3. A returned packet always comes from a complete frame whose declared `Length`
   is in range, and parsing consumes exactly that many bytes (padding untouched).
4. Attribute-level failures never masquerade as framing or stream errors.
5. Repeated parsing of identical bytes yields the same outcome, so the shared
   packet buffer pool cannot leak state between frames.

## Reproduction

Fixed budget used in CI and for closure evidence (seed corpus only, one pass):

```bash
GOTMPDIR="Z:/tmp-gobuild" go test ./internal/radiusd/ \
  -run 'TestRadiusPacketCorpusContract|TestRadiusCorpusVendorAndMessageAuthenticatorDecoding|FuzzParseTcpPacket|FuzzParseAttributesAndVendorDecoding|FuzzMessageAuthenticator' \
  -count=1
```

Extended exploration with a wall-clock budget:

```bash
GOTMPDIR="Z:/tmp-gobuild" go test ./internal/radiusd/ -run '^$' \
  -fuzz '^FuzzParseTcpPacket$' -fuzztime=30s
GOTMPDIR="Z:/tmp-gobuild" go test ./internal/radiusd/ -run '^$' \
  -fuzz '^FuzzParseAttributesAndVendorDecoding$' -fuzztime=30s
GOTMPDIR="Z:/tmp-gobuild" go test ./internal/radiusd/ -run '^$' \
  -fuzz '^FuzzMessageAuthenticator$' -fuzztime=30s
```

## Observed result (2026-10-27)

- `go test ./internal/radiusd/ -count=1`: pass (41.4 s).
- `FuzzParseTcpPacket` 25 s: 38,362 executions, 0 panics, 0 contract failures.
- `FuzzParseAttributesAndVendorDecoding` 20 s: 78,072 executions, 0 panics,
  0 contract failures.
- `FuzzMessageAuthenticator` 20 s: 65,807 executions, 0 panics, 0 contract
  failures.
- `go build ./...`: pass.
- `golangci-lint run ./internal/radiusd/...`: no findings in the harness files
  (three pre-existing findings remain in unrelated test files).

No failing input was written to `testdata/fuzz`; the working tree contained no
fuzz-generated artifacts after the runs.

## Limits

- The harness covers framing, attribute/VSA decoding, and Message-Authenticator
  computation. It does not fuzz the full EAP state machine, the UDP server entry
  point, authentication policy plugins, or HMAC timing behavior.
- Inputs are capped at 4100 bytes in the frame target and 4096 bytes in the
  attribute target; larger buffers cannot change a correctly framed parse.
- Fuzzing budgets are sampling, not proof of exhaustiveness. The deterministic
  corpus is the regression gate; extended fuzzing is a supplementary check.
