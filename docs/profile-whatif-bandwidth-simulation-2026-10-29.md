# FUT-007 - What a profile-inheritance what-if estimator can promise today

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-007` (`TR-F008`, P3, RISET) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/profilewhatif` (read-only, imported by nothing).
Criterion under test: **Output diff yang dapat direproduksi dan tidak memutasi
cache atau profile.**

## The question

The card asks for a what-if estimator over `internal/app/profile_cache.go`:
change one profile column, see the consequence for a fixture subscriber, read-only.
Two words in the criterion carry all the risk.

**"dapat direproduksi"** - a what-if answer that changes when nobody edited
anything is worse than no answer, because an operator reads it as a property of
the profile. So the estimator has to name, by measurement rather than by
convention, every part of an Access-Accept that is *not* a property of the
profile. This study found two, and both of them cost a test to find: one moves
with the wall clock, one moves with the tenant's own config file.

**"tidak memutasi cache atau profile"** - a preview that writes is a
configuration change dressed as a question. The prototype is built so that the
only object it can hand the product's enhancer chain is an interface with one
method, `Get(int64)`, and is then tested three ways: the injected shape is
checked by reflection against the product's own contract, the study is run over
tables the *test* owns and compared byte for byte afterwards, and the product's
enhancer chain is handed pointers straight into those tables so a write-through
has nowhere to hide.

The honest answer the study produced is narrower than the card implies, and worth
stating up front: **seven of the ten inherited columns reach the Access-Accept at
all**, the effect of a column depends on which NAS vendor the subscriber
terminates on, and the three remaining columns change whether a session is
accepted rather than what the accept says. A preview surface built on this can
diff a wire answer; it cannot yet promise "this edit changes nothing", because
admission is a different question from the accept.

## What counts as evidence

The same three rules the `ENG-006`, `ENG-007`, `ECO-006`, `FUT-001` and `FUT-002`
studies used, plus two this card forced:

- **A citation is a code line.** Comments are blanked before a symbol is matched,
  so a claim cannot be proven by prose describing it.
- **A whole identifier.** Bounded matching, so `GetUpRate` cannot be proven by
  `UpRate`, and a partial word in the other direction is refused too.
- **Drift is reported, never dropped.** A claim whose line no longer exists is
  printed as `broken`, a claim whose file disappeared as `vacated`.
- **Absence scans exclude this prototype's own directory.** Nothing in
  `internal/profilewhatif` may be evidence that the product does not do
  something.
- **No hand-typed attribute numbers.** Every wire identity comes from a source
  the study can re-measure: vendor VSAs are parsed out of the product's own
  generated `internal/radiusd/vendors/*/generated.go` files (`_<pkg>_VendorID`
  and each `<Pkg><Name>_Add/Set` pair), and standard attributes are named by
  *calling* the library setter on a probe packet and reading the number back off
  the wire. That is how `Acct-Interim-Interval` became `type-85` here rather than
  by someone quoting RFC 2865 from memory.
- **Runtime inputs are excluded by derivation.** The clock-dependent keys are
  found by moving only `user.ExpireTime` by an hour and diffing the resulting
  accept; the config-dependent keys are found by parsing every
  `getIntConfig(authCtx, app.ConfigX, N)` out of `default_enhancer.go`, joining
  the variable to the `_Set(` line that writes it, mapping that symbol to its
  wire key through the probe, and holding out the key when the emitted value
  equals the source's own fallback literal.

## How the estimator runs without a database, a config manager or a NAS

The prototype calls the product's real enhancers directly. `enhancerRoster` reads
the registration order out of `internal/radiusd/plugins/init.go` and constructs
that same list; each packet is one
`e.Enhance(ctx, &auth.AuthContext{Response: radius.New(radius.CodeAccessAccept, ...),
User: &fixture, Nas: vendors.VendorFor(code), Metadata: {"profile_cache": view}})`
over the roster, and the ledger is the attribute list read back off the signed
response. Nothing is stubbed on the answer side - `8/8` registered enhancers ran,
which is itself a checked number below.

Because the diff is taken between two packets built by the same code from the
same fixtures, "reproducible" is a testable property, not a claim: the digest
under *The measured numbers* is a sha256 over the scenario table, the effect
matrix, the baseline ledger, the held-out keys, the claims and the gaps, and the
suite asserts two independent runs produce the same digest.

## The measured numbers

| measure | value |
| --- | --- |
| fixture subscribers | 10 |
| fixture profiles | 4 |
| NAS vendors probed | 8 |
| profile columns | 10 |
| one-column probes run | 80 |
| probes that moved nothing | 36 |
| column/vendor pairs that moved something | 44 |
| scenarios | 20 |
| scenarios whose ledger moved | 15 |
| attribute lines across all scenario diffs | 26 |
| profile lookups total | 222 |
| profile lookups in one enhancement pass | 7 |
| stable attributes in the baseline accept | 5 |
| attribute keys held out | 2 |
| runtime config reads found in the enhancer | 1 |
| of those, emitted exactly the source default | 1 |
| vendor dictionary entries scanned | 561 |
| vendor dictionary sources scanned | 10 |
| standard attribute names probed | 10 |
| enhancers registered in the product | 8 |
| enhancers this study ran | 8 |
| columns read by a checker and by no enhancer | 3 |
| imports of this package | 31 |
| imports on the ban list | 0 |
| client calls into the wire library | 0 |
| package that owns CoA and Disconnect | radiusd (internal/radiusd/coa_service.go:240) |
| rate-asserting enhancer tests with no cache | 5 |
| statements checked | 23 |
| statements confirmed | 23 |
| gaps registered | 7 |
| gaps still open | 7 |

reproducibility digest: `b361a4d1b5da1e1f066fcb047581645a80eeaf739e36c43a2ce9a44f93d2b739`

The two numbers that needed a bug to appear are `stable attributes in the
baseline accept` (5) against the seven rows in the baseline table below, and
`attribute keys held out` (2). The first version of this study hashed the held-out
rows *with their values* into the digest, and the digest then changed every
second on an idle tree - Session-Timeout counts down in whole seconds. The
reproducibility test passed anyway, because the two runs inside it happened to
land in the same second. That is the failure mode this card is about, so it is now
pinned twice: the digest carries held-out keys without their values, and
`TestTheDigestSurvivesATickingClock` sleeps across a second boundary and compares
two full runs.

## The fixture profiles

| profile | column set as configured |
| --- | --- |
| bronze | UpRate=10240, DownRate=5120, ActiveNum=2, AddrPool="pool-bronze", Domain="bronze.example", RadiusClass="OU=bronze", IPv6PrefixPool="v6-bronze", DelegatedIpv6PrefixPool="pd-bronze", BindMac=0, BindVlan=0 |
| silver | UpRate=20480, DownRate=10240, ActiveNum=4, AddrPool="pool-silver", Domain="silver.example", RadiusClass="OU=silver", IPv6PrefixPool="v6-silver", DelegatedIpv6PrefixPool="pd-silver", BindMac=1, BindVlan=1 |
| gold | UpRate=51200, DownRate=204800, ActiveNum=8, AddrPool="pool-gold", Domain="gold.example", RadiusClass="OU=gold;guest", IPv6PrefixPool="v6-gold", DelegatedIpv6PrefixPool="pd-gold", BindMac=0, BindVlan=0 |
| unpolished | UpRate=0, DownRate=0, ActiveNum=0, AddrPool="NA", Domain="NA", RadiusClass="NA", IPv6PrefixPool="NA", DelegatedIpv6PrefixPool="NA", BindMac=0, BindVlan=0 |
## The fixture subscribers

| subscriber | link mode | profile | columns set on the user row |
| --- | --- | --- | --- |
| dyn-full | dynamic | bronze | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA |
| dyn-rate-override | dynamic | bronze | UpRate=1024, DownRate=2048, AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA |
| dyn-na-sentinel | dynamic | silver | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA |
| static-snapshot | static | bronze | UpRate=10240, DownRate=5120, ActiveNum=2, AddrPool=pool-bronze, Domain=bronze.example, RadiusClass=OU=bronze, IPv6PrefixPool=v6-bronze, DelegatedIpv6PrefixPool=pd-bronze |
| dyn-zero-profile | dynamic | unpolished | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA |
| dyn-bind-mac | dynamic | silver | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA, MacAddr=00:11:22:33:44:55 |
| dyn-bind-vlan | dynamic | silver | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA, Vlanid1=100,Vlanid2=200 |
| dyn-v6-static | dynamic | gold | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA, IpAddr=10.10.10.10, IpV6Addr=2001:db8::10, DelegatedIpv6Prefix=2001:db8:10::/48 |
| dyn-aruba | dynamic | gold | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA, Vlanid1=120,Vlanid2=0 |
| dyn-cisco | dynamic | bronze | AddrPool=NA, Domain=NA, RadiusClass=NA, IPv6PrefixPool=NA, DelegatedIpv6PrefixPool=NA |
`bronze`, `silver` and `gold` are ordinary rows. `unpolished` exists to put
the `NA` sentinel on every string column at once, and `dyn-zero-profile` exists to
link a subscriber to it - that pairing is what produces scenario `S15` below.

## What one Access-Accept says before any edit

| wire key | dictionary name | value | written by |
| --- | --- | --- | --- |
| type-27 | SessionTimeout | 5503597 | default-accept, held out of every diff |
| type-85 | AcctInterimInterval | 120 | default-accept, held out of every diff |
| type-88 | FramedPool | pool-bronze | default-accept |
| type-25 | Class | OU=bronze | default-accept |
| type-100 | FramedIPv6Pool | v6-bronze | default-accept |
| type-171 | DelegatedIPv6PrefixPool | pd-bronze | default-accept |
| 14988-8 | MikrotikRateLimit | 10240k/5120k | accept-mikrotik |
## What the ledger refuses to diff

| wire key | dictionary name | value | why held out |
| --- | --- | --- | --- |
| type-27 | SessionTimeout | 5503597 | clock or config |
| type-85 | AcctInterimInterval | 120 | clock or config |

The config reads found in the enhancer source:

| config key | source default | wire attribute | emitted | equals default |
| --- | --- | --- | --- | --- |
| ConfigRadiusAcctInterimInterval | 120 | type-85 | 120 | yes |
Both held-out rows arrived there by derivation, not by a list.
`type-27 SessionTimeout` moves when only the expiry column moves
(`internal/radiusd/plugins/auth/enhancers/default_enhancer.go:55` -
`time.Until(user.ExpireTime).Seconds()`). `type-85 AcctInterimInterval` moved
because the enhancer asked the tenant's config for a number, got the source's own
fallback `120`, and emitted exactly that; a tenant that sets
`ConfigRadiusAcctInterimInterval` to something else would emit something else, so
a preview may not diff it. Only one config read exists in the accept path, which
is itself a measured fact worth reading before someone assumes the estimator is
config-blind.

## Which wire attributes move when exactly one profile column changes

| profile column | NAS vendor code | attributes that moved |
| --- | --- | --- |
| UpRate | 14988 | 14988-8 |
| DownRate | 14988 | 14988-8 |
| AddrPool | 14988 | type-88 |
| RadiusClass | 14988 | type-25 |
| IPv6PrefixPool | 14988 | type-100 |
| DelegatedIpv6PrefixPool | 14988 | type-171 |
| UpRate | 2011 | 2011-2, 2011-3 |
| DownRate | 2011 | 2011-5, 2011-6 |
| AddrPool | 2011 | type-88 |
| Domain | 2011 | 2011-138 |
| RadiusClass | 2011 | type-25 |
| IPv6PrefixPool | 2011 | type-100 |
| DelegatedIpv6PrefixPool | 2011 | type-171 |
| UpRate | 25506 | 25506-1, 25506-2 |
| DownRate | 25506 | 25506-4, 25506-5 |
| AddrPool | 25506 | type-88 |
| RadiusClass | 25506 | type-25 |
| IPv6PrefixPool | 25506 | type-100 |
| DelegatedIpv6PrefixPool | 25506 | type-171 |
| UpRate | 3902 | 3902-89 |
| DownRate | 3902 | 3902-83 |
| AddrPool | 3902 | type-88 |
| RadiusClass | 3902 | type-25 |
| IPv6PrefixPool | 3902 | type-100 |
| DelegatedIpv6PrefixPool | 3902 | type-171 |
| UpRate | 10055 | 10055-1 |
| DownRate | 10055 | 10055-2 |
| AddrPool | 10055 | type-88 |
| RadiusClass | 10055 | type-25 |
| IPv6PrefixPool | 10055 | type-100 |
| DelegatedIpv6PrefixPool | 10055 | type-171 |
| AddrPool | 14823 | type-88 |
| Domain | 14823 | 14823-1 |
| RadiusClass | 14823 | type-25 |
| IPv6PrefixPool | 14823 | type-100 |
| DelegatedIpv6PrefixPool | 14823 | type-171 |
| AddrPool | 9 | 9-1, type-88 |
| RadiusClass | 9 | type-25 |
| IPv6PrefixPool | 9 | type-100 |
| DelegatedIpv6PrefixPool | 9 | type-171 |
| AddrPool | 0 | type-88 |
| RadiusClass | 0 | type-25 |
| IPv6PrefixPool | 0 | type-100 |
| DelegatedIpv6PrefixPool | 0 | type-171 |
This is the matrix the card is really asking for, and the three findings in it
are what a preview product has to accommodate:

1. **Not every column reaches the wire.** 36 of the 80 one-column probes moved
   nothing. `ActiveNum`, `BindMac` and `BindVlan` are missing from the table
   entirely - no column/vendor pair moved an attribute for them, and the scan
   says why: all three are read by a checker and by no enhancer, so they change
   admission, not the accept. `UpRate`/`DownRate` are absent on Aruba (`14823`)
   because that enhancer emits no rate attribute at all.
2. **The same column means different things per vendor.** One `UpRate` edit is
   two attributes on Huawei (`2011-2` average and `2011-3` peak), two on H3C
   (`25506-1`, `25506-2`), one on iKuai (`10055-1`), one *combined string* on
   Mikrotik
   (`14988-8`, `"<up>k/<down>k"` - which is why S01 and S02 differ only inside one
   attribute's text), and one on ZTE (`3902-89`). A preview that shows
   "before/after rate" per column will be wrong on at least Mikrotik and Huawei
   unless it shows attributes, not columns.
3. **Some columns are emitted twice, by different owners.** `AddrPool` on Cisco
   (`9`) moves both `9-1` and the standard `type-88 FramedPool`; on every other
   vendor it moves only `type-88`. A vendor VSA and a standard attribute are not
   interchangeable in a diff, and the report keeps them apart by identity
   (`14988-8` vs `type-88`) rather than by a name someone typed.

## The scenarios, one diff each

| scenario | NAS | cache reads | ledger diff |
| --- | --- | --- | --- |
| S01 | 14988 | 12 | 14988-8 changed 10240k/5120k -> 33333k/5120k |
| S02 | 14988 | 12 | 14988-8 changed 10240k/5120k -> 10240k/44444k |
| S03 | 2011 | 14 | 2011-2 changed 10485760 -> 34132992; 2011-3 changed 41943040 -> 136531968 |
| S04 | 25506 | 12 | 25506-4 changed 20971520 -> 182042624; 25506-5 changed 5242880 -> 45510656 |
| S05 | 3902 | 12 | 3902-89 changed 10485760 -> 34132992 |
| S06 | 10055 | 12 | 10055-1 changed 83886080 -> 273063936 |
| S07 | 14988 | 8 | (empty: the accept says the same thing) |
| S08 | 2011 | 14 | 2011-138 changed silver.example -> whatif.example |
| S09 | 9 | 10 | 9-1 changed ip:addr-pool=pool-bronze -> ip:addr-pool=whatif-pool; type-88 changed pool-bronze -> whatif-pool |
| S10 | 14988 | 12 | type-25 changed OU=bronze -> OU=whatif |
| S11 | 14988 | 12 | type-100 changed v6-bronze -> whatif-v6 |
| S12 | 14988 | 12 | type-171 changed pd-bronze -> whatif-pd |
| S13 | 14988 | 12 | 14988-8 changed 10240k/5120k -> 51200k/204800k; type-100 changed v6-bronze -> v6-gold; type-171 changed pd-bronze -> pd-gold; type-25 changed OU=bronze -> OU=gold;guest; type-88 changed pool-bronze -> pool-gold |
| S14 | 14988 | 6 | (empty: the accept says the same thing) |
| S15 | 14988 | 6 | 14988-8 changed 10240k/5120k -> 0k/0k; type-100 changed v6-bronze -> NA; type-171 changed pd-bronze -> NA; type-25 changed OU=bronze -> NA; type-88 changed pool-bronze -> NA |
| S16 | 14988 | 12 | (empty: the accept says the same thing) |
| S17 | 14988 | 12 | (empty: the accept says the same thing) |
| S18 | 14823 | 10 | (empty: the accept says the same thing) |
| S19 | 14823 | 10 | 14823-1 changed gold.example -> whatif.example |
| S20 | 14988 | 12 | 14988-8 changed 0k/0k -> 33333k/0k |
The four scenarios that carry the meaning are the ones an operator would
guess wrong:

- **S07 - empty.** The subscriber row already sets `UpRate=1024, DownRate=2048`,
  so editing the profile's rate changes nothing on the wire. This is the
  precedence rule `GetUpRate` implements (`u.UpRate > 0` wins), and a preview that
  silently reported "10240k -> 33333k" here would be lying about the customer's
  session.
- **S14 - empty.** Detaching a subscriber from dynamic linking to a static
  snapshot, *then* editing the profile, also changes nothing: the snapshot froze
  the values onto the user row and the getter never asks the cache again. Only 6
  cache lookups happen on this path versus 12 on the dynamic one, which is the
  same fact measured from the other side.
- **S15 - not empty, and not in the direction anyone wants.** The subscriber is
  linked to `unpolished`, and the cache is replaced by one that always errors.
  The accept does not stay put and it does not fail closed: the getters log,
  return the *user's own stale values*, and the wire answer becomes
  `0k/0k` with `NA` pools. A profile edit that lands on a row the cache cannot
  resolve is therefore a wire-visible event, and a preview must model the failure
  path rather than promise "no change".
- **S20 - a half-filled edit.** Setting the first rate on a profile whose rates
  are both zero moves `14988-8` from `0k/0k` to `33333k/0k`: only the upstream
  half of the string changed. Any diff UI that treats a rate as one object will
  hide exactly the half the operator is asking about.

S13 (move a subscriber from `bronze` to `gold`) is the most expensive diff in the
set: five attributes move at once - the combined rate string, both IPv6 pools,
`FramedPool`, and `Class`, whose value is the profile's `RadiusClass` string
`OU=gold;guest`. Nothing in the accept announces that a plan change happened
apart from those five, which is precisely the summary a preview exists to give.

## The statements, pinned to code

| id | statement | code line | status | measured |
| --- | --- | --- | --- | --- |
| C01 | the domain exposes one inheritance getter per column the study can change | internal/domain/radius_user_getters.go:22 | confirmed | 10 |
| C02 | a getter only consults the profile when the subscriber is linked dynamically | internal/domain/radius_user_getters.go:29 | confirmed |  |
| C03 | a nonzero user value beats the profile | internal/domain/radius_user_getters.go:24 | confirmed |  |
| C04 | an empty string and NA both mean unset, so the subscriber inherits | internal/domain/radius_user_getters.go:100 | confirmed | 8 getters test the sentinel |
| C05 | when the profile cannot be read the getter returns the stale user value and logs | internal/domain/radius_user_getters.go:33 | confirmed | 10 fallback log sites |
| C06 | static link mode is 0 and dynamic is 1 | internal/domain/radius_user_getters.go:9 | confirmed |  |
| C07 | Mikrotik renders both rates as one string attribute | internal/radiusd/plugins/auth/enhancers/mikrotik_enhancer.go:43 | confirmed |  |
| C08 | Huawei emits a peak as four times the average | internal/radiusd/plugins/auth/enhancers/huawei_enhancer.go:46 | confirmed |  |
| C09 | iKuai scales the same column by 1024*8 | internal/radiusd/plugins/auth/enhancers/ikuai_enhancer.go:43 | confirmed |  |
| C10 | Aruba reads the subscriber's own VLAN column, never a profile getter | internal/radiusd/plugins/auth/enhancers/aruba_enhancer.go:67 | confirmed |  |
| C11 | the default enhancer derives Session-Timeout from the expiry column | internal/radiusd/plugins/auth/enhancers/default_enhancer.go:55 | confirmed |  |
| C12 | the default enhancer writes Acct-Interim-Interval from a runtime config read | internal/radiusd/plugins/auth/enhancers/default_enhancer.go:56 | confirmed |  |
| C13 | the profile cache has a five minute default TTL | internal/app/profile_cache.go:30 | confirmed |  |
| C14 | the cache is keyed by profile id alone | internal/app/profile_cache.go:15 | confirmed |  |
| C15 | the profile update endpoint invalidates the cache after writing | internal/adminapi/profiles.go:450 | confirmed | 4 invalidate sites in adminapi |
| C16 | no production caller warms or writes the cache; only invalidation is used | internal/adminapi/profiles.go:450 | confirmed | 0 write-through call sites |
| C17 | the profile cache is injected on the authentication side and never on the accounting side | internal/radiusd/auth_plugin_runner.go:57 | confirmed | 0 cache reads under plugins/accounting |
| C18 | the CoA and Disconnect clients live in a package this prototype does not import | internal/radiusd/coa_service.go:258 | confirmed | radiusd (internal/radiusd/coa_service.go:240) |
| C19 | the roster the study runs is the order the product registers | internal/radiusd/plugins/init.go:37 | confirmed | 8 registrations |
| C20 | the runner hands the real profile cache to the same metadata key the study injects | internal/radiusd/auth_plugin_runner.go:57 | confirmed |  |
| C21 | every standard attribute name in the report is a setter the product calls | internal/radiusd/plugins/auth/enhancers/default_enhancer.go:55 | confirmed | 10/10 probe symbols found in product code |
| C22 | the cache type the product injects has mutators; the interface the getters accept does not | internal/domain/radius_user_getters.go:14 | confirmed | 5 mutators on ProfileCache: Invalidate, InvalidateAll, Set, Stats, Stop |
| C23 | an enhancer reaches every inherited column through a getter, never as a field | internal/radiusd/plugins/auth/enhancers/mikrotik_enhancer.go:40 | confirmed | 0 inherited columns read as fields anywhere in the enhancer tree |
`23/23` statements land on a code line today, with two that carry the
read-only argument:

- **C22** counts the exported mutators on the product's `ProfileCache` - `Set`,
  `Invalidate`, `InvalidateAll`, `Stats`, `Stop` - all of which the prototype
  cannot reach, because the getters ask for an interface whose only method is
  `Get`. `TestTheOnlyCacheShapeTheGettersAccept` checks that by reflection on the
  product's own contract type, so the read-only claim is a property of the type
  the product chose, not of this prototype's discipline.
- **C23** counts inherited columns that an enhancer reads as a struct field
  instead of through a getter: `0`. The enhancer tree does read user fields
  directly (`IpAddr`, `IpV6Addr`, `DelegatedIpv6Prefix`, Aruba's `Vlanid1` per
  C10), and none of those is one of the ten the study can change. That is what
  makes a per-column diff well-defined at all.

## What this study does not close

| id | gap | evidence | open |
| --- | --- | --- | --- |
| G01 | the management API has no what-if or preview route for a profile edit | 0 preview endpoints among the 5 profile handlers: CreateProfile, DeleteProfile, GetProfile, ListProfiles, UpdateProfile | yes |
| G02 | no product test runs a rate-bearing vendor enhancer with a profile cache | 5 rate-asserting enhancer test files without a cache | yes |
| G03 | three inherited columns change whether a session is accepted, not what the accept says | 3 columns read by a checker and by no enhancer | yes |
| G04 | the study ships no standard-attribute dictionary of its own | 0 code sites naming a standard attribute by its display name | yes |
| G05 | no NAS is reachable from CI, so the ledger is never confirmed on real hardware | asset-blocked: needs a lab NAS per vendor | yes |
| G06 | re-authorization after a profile edit is out of reach | radiusd (internal/radiusd/coa_service.go:240) owns CoA and is not in the prototype import set | yes |
| G07 | how many live subscribers a profile edit actually touches is unknown | asset-blocked: needs production session counts | yes |
## The import set this package builds on

| import | banned |
| --- | --- |
| context | no |
| crypto/sha256 | no |
| encoding/hex | no |
| errors | no |
| fmt | no |
| github.com/bjo163/mwx-isp/internal/domain | no |
| github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth | no |
| github.com/bjo163/mwx-isp/internal/radiusd/plugins/auth/enhancers | no |
| github.com/bjo163/mwx-isp/internal/radiusd/vendors | no |
| github.com/stretchr/testify/assert | no |
| github.com/stretchr/testify/require | no |
| go/parser | no |
| go/token | no |
| io/fs | no |
| layeh.com/radius | no |
| layeh.com/radius/rfc2865 | no |
| layeh.com/radius/rfc2869 | no |
| layeh.com/radius/rfc3162 | no |
| layeh.com/radius/rfc4818 | no |
| layeh.com/radius/rfc6911 | no |
| net | no |
| os | no |
| path/filepath | no |
| reflect | no |
| sort | no |
| strconv | no |
| strings | no |
| sync | no |
| testing | no |
| time | no |
| unicode | no |
## Verification

Reproduce the numbers, then the gates:

```
export GOCACHE=/z/tmp/mwx-gocache GOTMPDIR=/z/tmp/mwx-gotmp GOMODCACHE=/z/tmp/mwx-gomodcache
go test -p 1 -count=1 ./internal/profilewhatif/                     # ok, 23 statements
golangci-lint run ./internal/profilewhatif/...                       # 0 issues
go vet -p 1 ./internal/profilewhatif/
grep -rl internal/profilewhatif --include=*.go . \
  | grep -v '^\./internal/profilewhatif/'                            # 0 importers
WHATIF_DUMP=1 go test -p 1 -count=1 -run TestReportMarkdownDump -v \
  ./internal/profilewhatif/                                          # every table above
go test -p 1 -count=1 ./internal/radiusd/... ./internal/app/... \
  ./internal/domain/... ./internal/adminapi/...                       # all ok
```

The digest printed by the dump run is
`b361a4d1b5da1e1f066fcb047581645a80eeaf739e36c43a2ce9a44f93d2b739`, identical from
two separate processes.

Every table above is the dump. No row in this memo was written by hand.

**Mutation harness.** Each patch was applied to product code, the suite run, and
the file restored:

| mutation | product file | caught by |
| --- | --- | --- |
| an enhancer reads the checker-only column (`_ = user.GetActiveNum(profileCache)`) | `internal/radiusd/plugins/auth/enhancers/default_enhancer.go` | `TestCheckerOnlyColumnsMoveNothing` |
| the Mikrotik rate format changed (`%dk/%dk` -> `%dk\|%dk`) | `internal/radiusd/plugins/auth/enhancers/mikrotik_enhancer.go` | `TestTheLedgerShowsWhatASuccessfulEditDoes` |
| an enhancer writes through the subscriber pointer (`user.UpRate = 999`) | `internal/radiusd/plugins/auth/enhancers/default_enhancer.go` | `TestTheEnhancersThemselvesWriteNothing` |
| the digest re-admits a held-out value | `internal/profilewhatif/whatif.go` (prototype-side) | `TestTheDigestSurvivesATickingClock` |

The third row is the one that makes the criterion's second half real: it is the
only mutation that a copy would have absorbed. The test that refuses the copies
is `TestTheEnhancersThemselvesWriteNothing`, which hands the enhancer chain
pointers straight into the tracked tables.

## No-go register

Explicitly not decidable from this repository, so not attempted here:

- **Real NAS behaviour (G05).** The ledger is the accept the *product* builds, not
  the session a Cisco or Huawei box accepts. Confirming it needs a lab NAS per
  vendor, which is an asset, not a work item.
- **How many subscribers an edit touches (G07).** The estimator can say what an
  accept does; it cannot say how many live sessions reauthorise. Needs production
  session counts.
- **Re-authorization after the edit (G06).** CoA and Disconnect live in
  `internal/radiusd/coa_service.go`, the package this prototype bans from its
  import set, so "and then the session is refreshed" is out of reach by
  construction.
- **A shipped preview surface (G01).** `internal/adminapi/profiles.go` exposes
  `CreateProfile`, `DeleteProfile`, `GetProfile`, `ListProfiles`, `UpdateProfile`
  and no preview route. Adding one is a `TR-F008` product decision - this study
  only establishes what such a route could honestly output, and what it would have
  to refuse to promise.
