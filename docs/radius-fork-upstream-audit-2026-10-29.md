# RADIUS library fork / upstream audit — 2026-10-29

Feature: TR-F021 (protocol library maintenance) · backlog item ENG-016.

This is the periodic check of the RADIUS codec library: the build uses the org
fork `github.com/talkincode/radius`, while the import path stays
`layeh.com/radius`. The goal of the check is a recorded upstream/fork SHA pair,
a relevance matrix for every fork-only change, and an auditable
sync / defer / no-change decision — not a blind re-sync.

## Pinned state

| Item | Value |
| --- | --- |
| `require` line | `layeh.com/radius v0.0.0-20231213012653-1006025d24f8` |
| `replace` line | `layeh.com/radius => github.com/talkincode/radius v0.1.0` |
| Upstream commit behind the pseudo-version | `1006025d24f8fdd5b5ee8b3c15a714e6b24baaab` (2023-12-13, "fix IPv6Prefix bug") |
| Fork tag `v0.1.0` | annotated tag `1f5dfc3` → commit `8f214fbf5382d8f2ec9f8a3bb06c7d9fb0d35119` (2026-06-08) |
| Fork `master` HEAD | `8f214fbf5382` — identical to the tag (`ahead 0 / behind 0`) |
| Upstream `master` HEAD | `1006025d24f8` — identical to the commit in the `require` pseudo-version |
| Upstream maintenance | no commits since 2023-12-13; 57 open issues; issue #119 (IPv6Prefix padding bits) still open, never merged upstream |

Reproduce (no GitHub token required):

```bash
grep -E 'layeh.com/radius|talkincode/radius' go.mod
go list -m layeh.com/radius                      # shows the applied replacement
git ls-remote https://github.com/layeh/radius refs/heads/master
git ls-remote https://github.com/talkincode/radius refs/heads/master 'refs/tags/v0.1.0*'
```

Authenticated detail (commit bodies, per-file diff):

```bash
gh api repos/talkincode/radius/compare/1006025d24f8fdd5b5ee8b3c15a714e6b24baaab...v0.1.0 \
  --jq '.ahead_by, (.commits[] | .sha[0:12] + " " + (.commit.message|split("\n")[0])), .files[].filename'
```

## Upstream delta since the pinned commit

None. Upstream `master` still points at `1006025d24f8`, the exact commit the
`require` pseudo-version names, so there is nothing newer to take from upstream.
The upstream project has been dormant since 2023-12-13; the only activity
(`pushed_at` 2024-08-14) is non-`master` refs.

## Fork delta since upstream (2 commits, 9 files)

| Commit | Date | Summary | Files |
| --- | --- | --- | --- |
| `541733116cce` | 2026-06-08 | "Fix panic and wrong delete index in generated `_SetVendor`" | `dictionarygen/vendor.go`, seven regenerated dictionaries (aruba, microsoft, mikrotik, wispr, rfc4679), new `vendors/mikrotik/setvendor_test.go` |
| `8f214fbf5382` | 2026-06-08 | "Tolerate non-zero bits beyond the prefix in IPv6Prefix" | `attribute.go`, `attribute_test.go` |

### `_SetVendor` fix — classification: correctness / remote-input panic

The generated helper scanned sub-attributes at a fixed offset (`vsa[0]`,
`vsa[1]`) instead of the running cursor, and always advanced the cursor after
removing a match, so setting the same vendor attribute twice (or setting one
already present on an inbound packet) panicked with `slice bounds out of range`.
An emptied VSA was also removed with index `i+i`, which at `i ≥ 2` dropped
unrelated attributes.

Why it matters here: every vendor response enhancer and vendor attribute writer
(`internal/radiusd/plugins/...`, `internal/radiusd/vendors/*`) goes through
`<Vendor>_Set`. A malformed or repeated VSA on the wire must not crash the
server. The repository's own generated dictionaries already carry the corrected
template (cursor-based scan, `i+1` delete, rewritten VSA length), and
`internal/radiusd/vendors/mikrotik/setvendor_test.go` pins the regression
(`TestSetVendorReplaceSameAttribute` and the empty-VSA cases).

### `IPv6Prefix` fix — classification: attribute codec tolerance

`IPv6Prefix` rejected any Framed-IPv6-Prefix / Delegated-IPv6-Prefix whose bits
past the prefix length were non-zero. RFC 3162 §2.3 does require those bits to
be zero, but equipment frequently sends a full interface identifier behind a
`/64`, or a trailing bit on a non-octet prefix. Upstream issue #118 introduced
the strict check, and issue #119 (open upstream) reports the resulting
rejections; the fork masks the padding bits instead of erroring while keeping
the tolerance for omitted trailing zero octets.

Why it matters here: the huawei dictionary decodes these attributes, and the
auth enhancers echo the framed/delegated prefix into Access-Accept. A legitimate
subscriber session with host bits set would otherwise be decoded as an error.
`internal/radiusd/radiuslib_contract_test.go` pins the behavior for both a
`/64` with host bits and a `/61` with a trailing bit, and asserts that an
out-of-range prefix length and a length-less attribute are still rejected.

## Relevance matrix

| Change | Category | Reachable from network input | Used by MWX-ISP | Decision |
| --- | --- | --- | --- | --- |
| `_SetVendor` panic / wrong delete index | correctness, availability | yes (vendor VSA in Access-Request) | yes — all `internal/radiusd/vendors` setters | keep (already carried by `v0.1.0`; regression test in repo) |
| `IPv6Prefix` padding tolerance | attribute encoding / interop | yes (Framed-IPv6-Prefix, Delegated-IPv6-Prefix) | yes — huawei dictionary, auth enhancers | keep (already carried by `v0.1.0`; new contract test) |
| Regenerated vendor dictionaries | generated output of the fix above | — | yes | keep (same tag) |
| Upstream commits after `1006025d24f8` | — | — | — | none exist |

## What qualifies for a sync (decision criteria)

Apply these when a future check finds upstream commits or a suspected library
defect. "Sync" means cherry-pick into `talkincode/radius`, tag, bump the
`replace` line, run `go mod tidy`, and add a regression test here.

1. **Sync immediately** — remote crash/panic reachable from a RADIUS packet;
   authentication or secret handling (CHAP, Tunnel-Password, Message-Authenticator,
   EAP fragmentation/state); a codec that accepts malformed attributes or
   rejects well-formed RFC attributes; any change to bytes emitted on the wire
   (interop with real NAS).
2. **Sync after review** — new RFC dictionary content, API additions, dependency
   bumps that do not change protocol behavior.
3. **Defer / no-op** — documentation, CI, cosmetic, Go-version churn, or a
   behavior change that contradicts the RFC and has no field rationale.
4. **Never sync silently** — a fix without an upstream issue/PR reference, or one
   that changes packet bytes without a lab trace, gets a written no-go here first.

Every sync must state the commit range, the risk assessment, whether protocol
behavior changes, and the test that proves it.

## Decision for this run

**No change.** Upstream has no commits after the pinned `1006025d24f8`, and the
fork's `master` equals its `v0.1.0` tag, so the pinned fork already contains
both fork-only fixes. `go.mod` / `go.sum` are consistent (`go mod tidy -diff`
empty, `go mod verify` reports all modules verified).

Verification run:

```bash
go test ./internal/radiusd/ -run TestRadiusLibraryToleratesIPv6PrefixPaddingBits -count=1
go test ./internal/radiusd/vendors/... -count=1
go mod tidy -diff && go mod verify
```

All passed (2026-10-29, `dev`). Next check: re-run the `git ls-remote` pair weekly
or when a protocol defect is suspected; if upstream `master` moves past
`1006025d24f8`, triage each new commit against the criteria above.

## Limits

- The check compares code and commit metadata; it does not re-validate against
  real hardware, and none of the fork's behavior is claimed to be verified on a
  live NAS (that is ENG-013/ENG-050, blocked on lab access).
- Upstream is effectively unmaintained; a security fix may never appear there,
  so the fork remains the place where RADIUS codec fixes must be developed.
