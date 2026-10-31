# TR-F005 / M5.1 Vendor VSA Gap Baseline

This document is the M5.1 inventory for "vendor backlog and dictionary gaps"
under TR-F005. It is intentionally snapshot-style so the next M5 batch can
execute without re-discovering the same scope.

> Refreshed after the M5.5 batch (2026-10-31), which added the seven remaining
> Access-Accept enhancers — `alcatel`, `juniper`, `radback`, `microsoft`, `f5`,
> `hillstone` and `pfSense` — and restored the three dictionaries (f5,
> hillstone, pfSense) those enhancers need. Every vendor dictionary in the tree
> now has a parser and/or enhancer consumer, so this inventory no longer names a
> missing enhancer.
>
> Earlier rounds: M5.2 request parsers (#449–#454), M5.3 Aruba response
> enhancer (#456), and M5.4 Cisco `cisco-avpair` response enhancer (#543). A
> dictionary-pruning pass had removed the then-zero-reference generated packages
> (`f5`, `hillstone`, `pfSense`, `unix`) and replaced the `radback` generated
> dictionary with a hand-maintained minimal one; M5.5 restores f5/hillstone/
> pfSense together with their consumers, and `unix` stays removed. CI enforces
> that every package under `internal/radiusd/vendors/` keeps at least one
> external reference (`scripts/check-vendor-dict-usage.sh`).

## Scope and Method

Inspection sources:

- `internal/radiusd/vendors/*`
- `internal/radiusd/vendors/codes.go`
- `internal/radiusd/plugins/vendorparsers/parsers/init.go` (parser registration —
  the source of truth for what is actually active, not just the `*_parser.go`
  files present)
- `internal/radiusd/plugins/init.go` (`RegisterResponseEnhancer` registration)
- `share/dictionary` and `share/dictionary.*`

Snapshot commands (reproducible):

```bash
git rev-parse --short HEAD                                  # 9882f79e
ls internal/radiusd/vendors
grep -E 'Code[A-Z]' internal/radiusd/vendors/codes.go
grep -E 'Code(Huawei|H3C|ZTE|Radback|Alcatel|Aruba|Juniper)' \
  internal/radiusd/plugins/vendorparsers/parsers/init.go    # registered parsers
grep RegisterResponseEnhancer internal/radiusd/plugins/init.go
grep -c '^\$INCLUDE' share/dictionary                       # 213
```

## Coverage Snapshot (current)

- Vendor dictionary packages: `14` — 13 generated with radius-dict-gen plus the
  hand-maintained minimal `radback` package
- Registered vendor parsers (`parsers/init.go`): `default + huawei + h3c + zte +
  radback + alcatel + aruba + juniper` (7 vendors + default)
- Registered response enhancers (`plugins/init.go`): `default` plus `huawei`,
  `h3c`, `zte`, `mikrotik`, `ikuai`, `aruba`, `cisco`, `alcatel`, `juniper`,
  `radback`, `microsoft`, `f5`, `hillstone` and `pfsense` (14 vendors + default)
- `share/dictionary` includes `213` `$INCLUDE dictionary.*` entries; the repo
  ships in-tree packages for `13` of those vendors (ikuai has no
  `share/dictionary.ikuai`; radback has no `share/dictionary.radback`).
  Removed dictionaries can be regenerated from `share/dictionary.*` with
  radius-dict-gen when a parser or enhancer actually needs them.
- CI gate: `scripts/check-vendor-dict-usage.sh` fails the Lint job if any
  vendor dictionary package has zero external references.

## Why a parser is not always worth adding

`vendorparsers.VendorRequest` carries only `MacAddr`, `Vlanid1`, `Vlanid2`. A
vendor **parser** therefore adds value only when the vendor encodes MAC/VLAN with
a vendor-specific **request** VSA; otherwise it merely duplicates `DefaultParser`,
which already reads the standard `Calling-Station-Id`. The response **enhancer**
(rate / VLAN / `avpair` reply attributes) is a separate, demand-driven track and
does not depend on a parser.

## M5.1 Gap Matrix (current)

| Vendor Package | Vendor ID | `Code*` Constant | Parser | Enhancer | Dictionary Source |
| --- | ---: | --- | --- | --- | --- |
| `alcatel` | 3041 | present | present | present | `share/dictionary.alcatel` |
| `aruba` | 14823 | present | present | present | `share/dictionary.aruba` |
| `cisco` | 9 | present | missing | present | `share/dictionary.cisco` |
| `f5` | 3375 | present | missing | present | `share/dictionary.f5` |
| `h3c` | 25506 | present | present | present | `share/dictionary.h3c` |
| `hillstone` | 28557 | present | missing | present | `share/dictionary.hillstone` |
| `huawei` | 2011 | present | present | present | `share/dictionary.huawei` |
| `ikuai` | 10055 | present | missing | present | no `share/dictionary.ikuai` in tree |
| `juniper` | 2636 | present | present | present | `share/dictionary.juniper` |
| `microsoft` | 311 | present | missing | present | `share/dictionary.microsoft` (also used by EAP MS-CHAPv2) |
| `mikrotik` | 14988 | present | missing | present | `share/dictionary.mikrotik` |
| `pfsense` | 13644 | present | missing | present | `share/dictionary.pfsense` |
| `radback` | 2352 | present | present | present | hand-maintained minimal package (Mac-Addr 145, Bind-Dot1q-Vlan-Tag-Id 54, IP-Address-Pool-Name 36, Subscriber-Profile-Name 91); no `share/dictionary.radback`/`.redback` in tree |
| `zte` | 3902 | present | present | present | `share/dictionary.zte` |

Still removed from the pruning pass: `unix` (4). Its `Code*` constant does not
exist and it has no consumer; recreate constant, package and consumer together
if one is ever actually wired.

## Delta since the first baseline (#433)

The M5.2 / M5.3 batches closed the high-value request-side parser gaps and the
first response enhancer that the first baseline listed as pending:

- `radback` request parser — MAC + VLAN (#449)
- `alcatel` request parser — MAC, plus the `vendors.CodeAlcatel` constant (#450)
- `aruba` request parser — VLAN, plus the `vendors.CodeAruba` constant (#451)
- `juniper` request parser — VoIP VLAN (#453)
- `aruba` Access-Accept response enhancer (#456, M5.3)
- `cisco` Access-Accept response enhancer — emits `Cisco-AVPair="ip:addr-pool=<pool>"`
  from `GetAddrPool` (#543, M5.4)

The first baseline also reported `alcatel` / `aruba` as missing their `Code*`
constant; both now exist in `codes.go`.

## Delta: dictionary pruning

A repository audit found that 4 of the 15 generated dictionary packages had
zero external references, and that `radback` (15k generated lines, 1,708
accessors) was consumed through exactly two accessors. The pruning pass:

- removed `internal/radiusd/vendors/{f5,hillstone,pfSense,unix}` (zero
  references outside their own package at that time)
- replaced `radback/generated.go` with a hand-maintained minimal package
  keeping the exact accessor names and wire behavior used by
  `radback_parser.go` (`MacAddr_GetString`, `BindDot1qVlanTagID_Get`) and its
  tests; round-trip tests pin the RFC 2865 §5.26 wire contract
- added the `scripts/check-vendor-dict-usage.sh` CI gate so a new dictionary
  cannot merge without a parser/enhancer (or other consumer) wiring it in

Regenerating a removed vendor: run radius-dict-gen against the corresponding
`share/dictionary.*` source and add the package back together with the consumer
that needs it.

## Delta: M5.5 response-enhancer batch

Seven Access-Accept enhancers close the response-side gaps the first baseline
listed as demand-driven. Each follows the M5.3/M5.4 pattern: vendor gate via
`matchVendor`, nil-safety, profile-inherited getters, register in
`internal/radiusd/plugins/init.go`, and sample-based tests covering the vendor
match, the emitted values, `N/A`/empty skipping and nil safety.

| Enhancer | Emits | Source field |
| --- | --- | --- |
| `accept-alcatel` | `AAT-PPP-Address` (9, ipaddr), `AAT-IP-Pool-Definition` (63, string) | `IpAddr` (IPv4 only), `GetAddrPool` |
| `accept-juniper` | `Juniper-Local-User-Name` (1), `Juniper-Ip-Pool-Name` (36), `Juniper-VoIP-Vlan` (49) | `Username`, `GetAddrPool`, `Vlanid2` (1..4094) |
| `accept-radback` | `IP-Address-Pool-Name` (36), `Subscriber-Profile-Name` (91) | `GetAddrPool`, `GetDomain` |
| `accept-microsoft` | `MS-Service-Class` (42) | `GetDomain` |
| `accept-f5` | `F5-LTM-User-Partition` (3) | `GetDomain` |
| `accept-hillstone` | `Hillstone-User-Role-Bame` (9), `Hillstone-VPN-DHCP-Pool` (102) | `GetDomain`, `GetAddrPool` |
| `accept-pfsense` | `pfSense-Bandwidth-Max-Up` (1), `pfSense-Bandwidth-Max-Down` (2) | `GetUpRate`, `GetDownRate` (kbit/s, positive only) |

`Domain` is the generic vendor-policy field already used by the Huawei and
Aruba enhancers; it is mapped to the one primary policy selector each of these
NAS platforms reads. Attributes a subscriber record does not model (F5 role /
shell / console, MS-MPPE key material) are deliberately not invented.

The three restored dictionaries (`f5`, `hillstone`, `pfsense`) are fresh
radius-dict-gen output from the in-tree `share/dictionary.*` sources, and the
`radback` minimal package gained the two reply-side string accessors it now
serves.

## Remaining gaps / backlog for the next M5 batch

The high-value request-side parsers and every response enhancer shipped in tree
are now delivered (see Deltas above). What remains is deliberately
demand-driven:

1. **New vendors only on demand.** Adding an enhancer (or a dictionary) for a
   vendor outside the current matrix follows the same pattern: dictionary +
   consumer + registration + sample-based tests. No vendor in `share/dictionary`
   is a current target on its own.
2. **Parsers — high-value request-side MAC/VLAN vendors are done.** Add a new
   parser only when a vendor dictionary proves a vendor-specific MAC/VLAN
   **request** encoding (otherwise it duplicates `DefaultParser`).
3. **Symmetry-only parsers stay deferred.** `mikrotik`, `ikuai` already ship
   response enhancers; their VLAN VSAs (e.g. `Mikrotik-Wireless-VLANID`) are
   Access-Accept reply attributes and their request side uses the standard
   `Calling-Station-Id`, so a dedicated parser would behave identically to
   `DefaultParser`.
4. **Hygiene — resolved.** The `unix` package (VendorID 4) was removed in the
   pruning pass; it never had a `Code*` constant or any consumer. Recreate the
   constant, package, and consumer together if a `unix` parser or enhancer is
   ever actually wired.
5. **Per-vendor acceptance set (unchanged).**
   - parser + registration in `parsers/init.go`, and/or enhancer + registration
     in `plugins/init.go`
   - sample-based parser / enhancer tests with real attribute samples
   - `go test ./internal/radiusd/...` and `golangci-lint` (v2.12.2) green

## Non-Goals of M5.1

- No runtime behavior changes
- No new parser/enhancer implementation in this round
- No expansion beyond TR-F005 scope
