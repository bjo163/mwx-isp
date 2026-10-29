# FUT-018 — What a printable grunge incident card is allowed to say

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-018` (`TR-F029`, P3, RISET, dependency `FUT-017`) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/incidentcard` (read-only, imported by nothing).
Criterion under test: **Print A4 hitam-putih dapat dibaca; stale/confidence
terlihat; tidak membocorkan nama pelanggan.**

## The question

The card asks for a *grunge-styled printable incident card* whose content is
"data yang terbukti saja" and which carries no PII. The criterion then states
three things that each have to be falsifiable, and the honest difficulty is that
all three are properties of a sheet that does not exist yet: there is no card
component in `web/src`, no exporter anywhere in the tree, and no printed proof.
So this round prices each clause against the product as it ships, which is the
only evidence a repository-only round can reach:

1. **"Tidak membocorkan nama pelanggan."** Not asserted - measured, through four
   independent readings of the same field set. A candidate slot is printed only
   when the shipped classifier calls its key harmless (`internal/redact`), *and*
   no select list in `internal/adminapi` proves the value is a column of an
   identity table, *and* no `Sprintf` composes such a column into its free text,
   *and* - for a key that literally says `name` - every assignment the handlers
   perform for it reads a persisted row that is not the subscriber's. The fourth
   check is the one that decides the criterion's own words, because the product
   prints three name-shaped fields and only the assignment proves which of them
   is a person.
2. **"Print A4 hitam-putih dapat dibaca."** The greys are derived, not chosen:
   every operational color token in `web/src/theme.ts` is converted through the
   WCAG relative-luminance formula to the grey it dries to on the white sheet the
   console's own `@media print` rule forces (`background-color: #fff
   !important`), then measured against that sheet at the chip's actual 10.9px,
   and against each other. The page budget is derived from the same file's
   metrics. What a printer actually lays down is not derivable from source and is
   recorded as `G01`.
3. **"Stale/confidence terlihat."** Per field, three separate questions: is it
   served by the API, is it read by any line of `web/src`, and is there a print
   path carrying the surface that reads it. The third is the one that hurts.

## What counts as evidence

A statement is pinned to a *code* line: comments are blanked per file dialect
before matching, the match is a whole identifier rather than a substring, and a
symbol that moves or disappears is reported as `broken` or `vacated` instead of
quietly dropping the row. Absence claims ("zero `@page` rules", "no PDF
dependency") are scans that exclude this study's own directory, so the prototype
cannot be its own evidence. Nothing is clock-dependent: the digest is a sha256
over tab-separated measured rows, and a run repeated after a ticking clock is
byte-identical (`TestTheDigestSurvivesATickingClock`).

One choice in the ledger is interpretive rather than parsed, and is named here
because it decides the name clause: `subscriberRows = {isp_customer,
radius_user}`. Those two models are the rows whose `name`-shaped columns describe
a person - `isp_customer.name` is the subscriber the complaint is about and
`radius_user.realname` is the same person on the account record. Every other
identity row in the census names a site, a piece of equipment, or the operator's
own company, so `net_nas.name` and `isp_odp.name` print. The two subscriber rows
are pinned by `A23`/`A24`, and both pin the finding that follows them: **the
shipped rules never name either column.** `radius_user.realname` is a "Contact
name" the classifier calls `Plain`, and `isp_customer.name` is a plain indexed
column on the customer row.

## The prototype

| File | Role |
| --- | --- |
| `incidentcard.go` | `Run`, the `Result` the memo splices, the 37-counter totals table and the sha256 digest. |
| `evidence.go` | File walking, comment blanking that keeps line numbers, bounded identifier matching, `structError`, and the import-set reader. |
| `rules.go` | The shipped classifier read out of `internal/redact/redact.go` - its fragments, its predicates and its class order - so the policy is parsed, not copied. |
| `slots.go` | The served response types, their fields and json keys, the identity-table census, the select-list projections and the prose assemblies. |
| `ledger.go` | The four checks applied to every candidate slot, and the print/refuse verdict each one earns. |
| `name.go` | The name trace: which assignment fills each name-shaped slot, what row it reads, and whether that row is the subscriber's. |
| `visuals.go` | Token parsing from `web/src/theme.ts`, the luminance conversion, the greys, the pairwise separations and the page metrics. |
| `print.go` | The paper census - which declarations `web/src` writes and which incident surfaces have a print path at all. |
| `card.go` | The ink channels, the status vocabulary, the non-color label channels and the one-page budget. |
| `claims.go` | 24 statements with their checks and the 7-item gap register. |
| `render.go` | Emits every table in this memo from one run. |
| `study_test.go` | 17 gates: the criterion's three clauses, the counters, and the no-write/no-importer guarantees. |
| `render_test.go` | 5 gates on the report itself: every section present, digest carried, tables unbroken, limits open, statements lined. The 23rd test function, `TestReportMarkdownDump`, is the harness that emits the tables below; it skips unless `CARD_DUMP=1`. |

Nothing in `internal/` or `cmd/` imports it (`TestNothingImportsTheStudy` plus a
grep over the tree that returns 0), it opens no database connection
(`TestTheStudyReadsNoDatabase`), and the only files it touches are opened for
reading (`TestTheStudyWritesNothing` snapshots `internal/adminapi`,
`internal/domain`, `internal/redact` and `web/src` before and after a run). Its
15 imports are stdlib only (`TestTheImportAllowlistIsNotVacuous` fails if the
parse finds nothing rather than passing on an empty set).

## What the run says

**The customer's name is not printed, and the reason is not the one the criterion
assumes.** 129 candidate slots come out of the 15 response types an operator sees
while investigating an incident; 108 print and 21 are refused. `customer_name`
appears twice in the ledger - the NOC impact roster and the flapping list - and
both refusals are the boring kind: the shipped classifier already names the key
subscriber PII (`A02`), and the roster's own SELECT proves the column arrives
from `isp_customer` (`A05`). Three name-shaped fields do print, and each is
traced rather than trusted: `nocImpactMonitoring.target_name` reads
`net_monitor_target.name`, `nocImpactScope.nas_name` reads `net_nas.name`, and
`nocImpactScope.name` reads three rows (`net_monitor_target`, `isp_odp`,
`net_nas`) across three assignments at `noc_impact.go:223`, `:273` and `:329`.
Those are a site, a port group and a NAS. The trace holds one thing off on its
own account: `SyslogEvent.severity_name` is filled by a call expression at
`telemetry.go:427`, so the study cannot state which row the label came from and
declines to print it - a conservative refusal, not a found leak.

**Most of the privacy defense is this ledger's, not the product's.** Splitting
the 21 refusals by which policy covers them: 8 sit on a column or key the shipped
rules already name; **13 sit past every shipped rule**, where this study is the
only thing standing in the way. The example that matters is the impacted roster:
`customer_id`, `customer_no`, `odp_code` and `odp_port` are all `Plain` to the
classifier and all provably filled from `isp_customer` by one select list, and
`username` likewise from `radius_user`. A card that printed them "because no rule
forbids it" would leak a subscriber's account identity while breaking nothing in
`internal/redact`. Two prose slots carry the same shape in the other direction:
`TroubleTicket.subject` is composed from `isp_customer.Name` at `isp.go:757` -
a name the rules never classify - and `description` from `isp_customer.Address`
at `isp.go:761`. The free-text sweep cannot see either, because its vocabulary is
credentials, URL credentials, e-mail and IPv4 (`A03`).

**Black and white reads well in the light skin and badly in the dark one.** All
18 light tokens clear the 4.5:1 floor against the forced-white sheet, the worst
at 8.32:1. In the dark palette - which is the default the product ships (`A16`) -
**8 of 18 fail**, and they are the ones a card would actually print:
`warning.main #fbbf24` dries at 3.03:1, `success.main #62d990` at 3.45:1,
`info.main #55cbd5` at 4.06:1, and the five `.light` tints land between 2.24:1
and 3.95:1. Since the print rule forces white paper in both modes, a card built
on the dark set prints unreadable chip text at the 10.9px the skin uses. The fix
is a product decision, not a finding: the printed surface has to ask for the
light token set.

**Grey cannot tell the states apart, so the card must print words.** 22 status
values share 5 colors, and 28 of the color pairs sit below 3:1 against each other
once they dry - in the light palette every pair of `main` tones is within 1.23:1,
which is to say indistinguishable. What survives the sheet is what the console
already writes without color: an uppercase label, an outline-only chip, a
translated status string, and monospaced identifiers (all four counted in
`Enterprise.tsx`). A grunge card that encodes state as a fill color loses the
state; one that encodes it as the label keeps it.

**Stale and confidence are visible on screen and invisible on paper.** 10
stale/confidence slots reach the ledger and print. `stale` is read by 12 lines of
`web/src`, `confidence` by 7, `state` by 2, `generated_at` and `warnings` by 1
each - and 4 of the 10 (`confidence_reason`, `stale_after_seconds`,
`last_checked_at`, `interval_seconds`) are not read by the console at all. Then
the paper half: **6 incident surfaces exist and 0 of them have a print path**, so
every one of those 10 fields has zero print paths carrying it. "Stale terlihat"
is currently true only on screen; a card has to add the second half. The bounds
that make the two states meaningful do ship and are pinned: `confidence` is one
of three server-stated values (`A06`), `stale` a named state rather than an
inference (`A07`), staleness floored at five minutes so a fast monitor cannot
flap the printed verdict (`A09`), and the roster is bounded by a 24h lookback a
printed sheet cannot refresh (`A08`) - which is exactly why `generated_at` has to
be on the card.

**The sheet does not hold the ledger.** The skin's own metrics give a printable
height of 1026.5px (A4 at 96 css px/in, less an assumed 12.7mm margin, because
the tree writes no `@page` rule at all), a 32.8px title band, a 49.8px confidence
strip and 28.8px per slot row: **32 rows on one page against the 108 the ledger
allows**, a shortfall of 2173.9px. "Data yang terbukti saja" is therefore not a
constraint the card can satisfy by printing everything it is allowed to; it has
to select roughly a third of one surface's fields, and the selection is the
design work this round cannot make for the next one.

**The grunge survives the paper rules only partly, and nobody has decided the
rest.** Of the ten ink channels the skin is made of, 6 are named by an existing
`@media print` declaration - including `background-color`, which is precisely the
property the console uses to erase the panel fill the look is built on
(`A13`) - and 3 are named by nothing: the halftone dot grid, the uppercase
display titles and the monospaced identifiers. Three of those (the display face,
the halftone and any background at all) hinge on a browser setting and a driver
this round cannot read, and on a face the product never ships: `index.html`
fetches Inter and JetBrains Mono only, so the grunge titles fall back to whatever
Arial Narrow or Impact the printing machine has (`A19`, `G04`).

## The measured tables

Spliced verbatim from one run of `internal/incidentcard`.

## Field ledger

Every slot the incident response types serve, with the four checks applied: the shipped classifier, the query that fills it, the prose the server composes into it and - for a key that speaks of a named party - the assignment behind it.

| response type | slot | type | class | source | disposition | why |
| --- | --- | --- | --- | --- | --- | --- |
| TroubleTicket | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| TroubleTicket | ticket_no | string | Plain | - | print | - |
| TroubleTicket | customer_id | int64 | Plain | isp_customer.id @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.id, a column no shipped rule names |
| TroubleTicket | subscription_id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| TroubleTicket | subject | string | Plain | prose @ internal/adminapi/isp.go:757 | refused | prose slot: the product formats isp_customer.Name into it, a column no shipped rule names |
| TroubleTicket | category | string | Plain | - | print | - |
| TroubleTicket | priority | string | Plain | - | print | - |
| TroubleTicket | status | string | Plain | - | print | - |
| TroubleTicket | assigned_technician | string | Plain | - | print | - |
| TroubleTicket | technician_phone | string | CustomerPII | - | refused | subscriber PII by key |
| TroubleTicket | description | string | Plain | prose @ internal/adminapi/isp.go:761 | refused | prose slot: the product formats isp_customer.Address into it, a column the shipped rules name |
| TroubleTicket | resolution_notes | string | Plain | - | print | - |
| TroubleTicket | scheduled_date | *time.Time | Plain | - | print | - |
| TroubleTicket | resolved_at | *time.Time | Plain | - | print | - |
| TroubleTicket | created_at | time.Time | Plain | - | print | - |
| TroubleTicket | updated_at | time.Time | Plain | - | print | - |
| NetMonitorIncident | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| NetMonitorIncident | target_id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| NetMonitorIncident | state | string | Plain | - | print | - |
| NetMonitorIncident | summary | string | Plain | - | print | - |
| NetMonitorIncident | started_at | time.Time | Plain | - | print | - |
| NetMonitorIncident | resolved_at | *time.Time | Plain | - | print | - |
| NetMonitorSample | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| NetMonitorSample | target_id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| NetMonitorSample | checked_at | time.Time | Plain | - | print | - |
| NetMonitorSample | reachable | bool | Plain | - | print | - |
| NetMonitorSample | latency_milliseconds | int64 | Plain | - | print | - |
| NetMonitorSample | packet_loss_percent | float64 | Plain | - | print | - |
| NetMonitorSample | interface_metrics_json | string | Plain | - | print | - |
| NetMonitorSample | error | string | Plain | - | print | - |
| NotificationOutbox | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| NotificationOutbox | dedupe_key | string | Plain | - | print | - |
| NotificationOutbox | event_type | string | Plain | - | print | - |
| NotificationOutbox | recipient | string | Plain | - | print | - |
| NotificationOutbox | body | string | Plain | - | print | - |
| NotificationOutbox | status | string | Plain | - | print | - |
| NotificationOutbox | attempts | int | Plain | - | print | - |
| NotificationOutbox | next_attempt_at | time.Time | Plain | - | print | - |
| NotificationOutbox | last_error | string | Plain | - | print | - |
| NotificationOutbox | created_at | time.Time | Plain | - | print | - |
| NotificationOutbox | sent_at | *time.Time | Plain | - | print | - |
| SyslogEvent | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| SyslogEvent | nas_ip | string | Address | - | refused | address or IP by key |
| SyslogEvent | facility | int | Plain | - | print | - |
| SyslogEvent | severity | int | Plain | - | print | - |
| SyslogEvent | severity_name | string | Plain | name trace @ internal/adminapi/telemetry.go:427 | refused | filled from an expression the study cannot resolve at internal/adminapi/telemetry.go:427, which is not a row this study can read |
| SyslogEvent | tag | string | Plain | - | print | - |
| SyslogEvent | message | string | Plain | - | print | - |
| SyslogEvent | synthetic | bool | Plain | - | print | - |
| SyslogEvent | created_at | time.Time | Plain | - | print | - |
| nocImpactResponse | scope | nocImpactScope | Plain | - | print | - |
| nocImpactResponse | state | string | Plain | - | print | - |
| nocImpactResponse | confidence | string | Plain | - | print | - |
| nocImpactResponse | confidence_reason | string | Plain | - | print | - |
| nocImpactResponse | monitoring | nocImpactMonitoring | Plain | - | print | - |
| nocImpactResponse | counts | nocImpactCounts | Plain | - | print | - |
| nocImpactResponse | impacted | []nocImpactSubscriber | Plain | - | print | - |
| nocImpactResponse | pagination | nocImpactPagination | Plain | - | print | - |
| nocImpactResponse | warnings | []string | Plain | - | print | - |
| nocImpactResponse | generated_at | time.Time | Plain | - | print | - |
| nocImpactIncident | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactIncident | summary | string | Plain | - | print | - |
| nocImpactIncident | started_at | time.Time | Plain | - | print | - |
| nocImpactMonitoring | monitored | bool | Plain | - | print | - |
| nocImpactMonitoring | enabled | bool | Plain | - | print | - |
| nocImpactMonitoring | status | string | Plain | - | print | - |
| nocImpactMonitoring | stale | bool | Plain | - | print | - |
| nocImpactMonitoring | stale_after_seconds | int | Plain | - | print | - |
| nocImpactMonitoring | last_checked_at | *time.Time | Plain | - | print | - |
| nocImpactMonitoring | interval_seconds | int | Plain | - | print | - |
| nocImpactMonitoring | target_id | string | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactMonitoring | target_name | string | Plain | - | print | name-shaped; every assignment reads net_monitor_target.name |
| nocImpactMonitoring | open_incident | *nocImpactIncident | Plain | - | print | - |
| nocImpactScope | type | string | Plain | - | print | - |
| nocImpactScope | id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactScope | name | string | Plain | - | print | name-shaped; every assignment reads isp_odp.name, net_monitor_target.name, net_nas.name |
| nocImpactScope | code | string | Plain | - | print | - |
| nocImpactScope | zone | string | Plain | - | print | - |
| nocImpactScope | address | string | CustomerPII | - | refused | subscriber PII by key |
| nocImpactScope | kind | string | Plain | - | print | - |
| nocImpactScope | target_id | string | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactScope | nas_id | string | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactScope | nas_name | string | Plain | - | print | name-shaped; every assignment reads net_nas.name |
| nocImpactCounts | customers | int64 | Plain | - | print | - |
| nocImpactCounts | subscriptions | int64 | Plain | - | print | - |
| nocImpactCounts | online_sessions | int64 | Plain | - | print | - |
| nocImpactCounts | open_tickets | int64 | Plain | - | print | - |
| nocImpactSubscriber | customer_id | string | Plain | isp_customer.id @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.id, a column no shipped rule names |
| nocImpactSubscriber | customer_no | string | Plain | isp_customer.customer_no @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.customer_no, a column no shipped rule names |
| nocImpactSubscriber | customer_name | string | CustomerPII | - | refused | subscriber PII by key |
| nocImpactSubscriber | odp_code | string | Plain | isp_customer.odp_code @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.odp_code, a column no shipped rule names |
| nocImpactSubscriber | odp_port | int | Plain | isp_customer.odp_port @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.odp_port, a column no shipped rule names |
| nocImpactSubscriber | subscription_id | string | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| nocImpactSubscriber | subscription_no | string | Plain | - | print | - |
| nocImpactSubscriber | subscription_status | string | Plain | - | print | - |
| nocImpactSubscriber | username | string | Plain | radius_user.username @ internal/adminapi/noc_impact.go:409 | refused | query projection: holds radius_user.username, a column no shipped rule names |
| nocImpactSubscriber | online | bool | Plain | - | print | - |
| nocImpactSubscriber | last_seen_at | *time.Time | Plain | - | print | - |
| nocImpactPagination | page | int | Plain | - | print | pagination handle, not incident content |
| nocImpactPagination | page_size | int | Plain | - | print | pagination handle, not incident content |
| nocImpactPagination | total | int64 | Plain | - | print | pagination handle, not incident content |
| flappingSubscriber | username | string | Plain | radius_user.username @ internal/adminapi/noc_impact.go:409 | refused | query projection: holds radius_user.username, a column no shipped rule names |
| flappingSubscriber | customer_no | string | Plain | isp_customer.customer_no @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.customer_no, a column no shipped rule names |
| flappingSubscriber | customer_name | string | CustomerPII | - | refused | subscriber PII by key |
| flappingSubscriber | phone | string | CustomerPII | - | refused | subscriber PII by key |
| flappingSubscriber | address | string | CustomerPII | - | refused | subscriber PII by key |
| flappingSubscriber | odp_code | string | Plain | isp_customer.odp_code @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.odp_code, a column no shipped rule names |
| flappingSubscriber | disconnect_count | int | Plain | - | print | - |
| flappingSubscriber | last_terminate_cause | string | Plain | prose @ internal/adminapi/isp_operations.go:1526 | refused | prose slot: the product formats radius_accounting.AcctSessionTime into it, a column no shipped rule names |
| flappingSubscriber | last_seen_time | time.Time | Plain | - | print | - |
| flappingSubscriber | severity | string | Plain | - | print | - |
| flappingSubscriber | suggested_resolution | string | Plain | - | print | - |
| customerTimelineEvent | event_id | string | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| customerTimelineEvent | source | string | Plain | - | print | - |
| customerTimelineEvent | type | string | Plain | - | print | - |
| customerTimelineEvent | occurred_at | time.Time | Plain | - | print | - |
| customerTimelineEvent | tenant_id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| customerTimelineEvent | customer_id | int64 | Plain | isp_customer.id @ internal/adminapi/noc_impact.go:408 | refused | query projection: holds isp_customer.id, a column no shipped rule names |
| customerTimelineEvent | subscription_id | int64 | Plain | - | print | opaque handle: needed to correlate, carries no subscriber value |
| customerTimelineEvent | summary | string | Plain | - | print | - |
| customerTimelineEvent | source_path | string | Plain | - | print | - |
| customerTimelineEvent | severity | string | Plain | - | print | - |
| systemLogEntry | timestamp | string | Plain | - | print | - |
| systemLogEntry | level | string | Plain | - | print | - |
| systemLogEntry | namespace | string | Plain | - | print | - |
| systemLogEntry | job | string | Plain | - | print | - |
| systemLogEntry | message | string | Plain | - | print | - |
| systemLogEntry | line | string | Plain | - | print | - |
| systemLogEntry | fields | map[string]unknown | Plain | - | print | free-form field map: the card prints the keys it recognizes only |

129 slots, 108 printable, 21 refused.

## Name-shaped slots and the rows behind them

The criterion forbids the customer's name, so every served key that says name is followed past its key to the assignment that fills it. A name the handlers do not account for, or account for with a subscriber row, stays off the sheet.

| slot | assignments | reads | disposition |
| --- | ---: | --- | --- |
| SyslogEvent.severity_name | 1 | an expression the study cannot resolve @ internal/adminapi/telemetry.go:427 | refused |
| nocImpactMonitoring.target_name | 1 | net_monitor_target.name @ internal/adminapi/noc_impact.go:548 | print |
| nocImpactScope.name | 3 | net_monitor_target.name @ internal/adminapi/noc_impact.go:223; isp_odp.name @ internal/adminapi/noc_impact.go:273; net_nas.name @ internal/adminapi/noc_impact.go:329 | print |
| nocImpactScope.nas_name | 1 | net_nas.name @ internal/adminapi/noc_impact.go:243 | print |
| nocImpactSubscriber.customer_name | 0 | none | refused |
| flappingSubscriber.customer_name | 0 | none | refused |

6 name-shaped slots served, 3 printed with every assignment traced to a row that is not the subscriber's, 3 held off.

## Identity tables the rules cover only partly

A persisted model counts as an identity table when the shipped rules already name one of its columns sensitive. The columns it serves that no rule names are the ones a card can reach without a guard.

| table | model | guarded columns | unguarded columns served |
| --- | --- | --- | --- |
| isp_customer | Customer | address, email, phone | city, created_at, customer_no, id, identity_no, latitude, longitude, name, notes, odp_code, odp_id, odp_port, province, status, updated_at |
| isp_hotspot_voucher | HotspotVoucher | password | batch_id, code, created_at, expires_at, first_login_at, id, package_id, price, quota_bytes, status, used_bytes, validity_seconds |
| isp_odp | ODP | address | code, created_at, id, latitude, longitude, name, notes, olt_name, optical_loss, pon_port, status, total_ports, updated_at, used_ports, zone |
| isp_trouble_ticket | TroubleTicket | technician_phone | assigned_technician, category, created_at, customer_id, description, id, priority, resolution_notes, resolved_at, scheduled_date, status, subject, subscription_id, ticket_no, updated_at |
| net_monitor_target | NetMonitorTarget | address | created_at, enabled, failure_threshold, id, interval_seconds, kind, last_checked_at, last_error, last_latency_milliseconds, last_packet_loss_percent, last_status, name, port, probe_type, snmp_auth_protocol, snmp_privacy_protocol, snmp_username, snmp_version, timeout_milliseconds, updated_at |
| net_nas | NetNas | ipaddr, secret | coa_port, created_at, hostname, id, identifier, model, name, node_id, remark, status, tags, updated_at, vendor_code |
| radius_accounting | RadiusAccounting | framed_ipaddr, framed_ipv6_address | acct_input_packets, acct_input_total, acct_output_packets, acct_output_total, acct_session_id, acct_session_time, acct_start_time, acct_stop_time, delegated_ipv6_prefix, framed_ipv6_prefix, framed_netmask, id, last_update, mac_addr, nas_addr, nas_class, nas_id, nas_paddr, nas_port, nas_port_id, nas_port_type, service_type, session_timeout, username |
| radius_online | RadiusOnline | framed_ipaddr, framed_ipv6_address | acct_input_packets, acct_input_total, acct_output_packets, acct_output_total, acct_session_id, acct_session_time, acct_start_time, delegated_ipv6_prefix, framed_ipv6_prefix, framed_netmask, id, last_update, mac_addr, nas_addr, nas_class, nas_id, nas_paddr, nas_port, nas_port_id, nas_port_type, service_type, session_timeout, username |
| radius_session_action_audit | RadiusSessionActionAudit | operator_ip | acct_session_id, action, attempts, created_at, error, error_cause, error_cause_text, id, nas_addr, operator_id, operator_name, response_code, rtt_ms, session_id, success, target, timed_out, triggered_at, username |
| radius_user | RadiusUser | address, email, ip_addr, password | active_num, addr_pool, bind_mac, bind_vlan, created_at, delegated_ipv6_prefix, delegated_ipv6_prefix_pool, domain, down_rate, expire_time, id, ipv6_addr, ipv6_prefix_pool, last_online, mac_addr, mobile, node_id, online_count, profile_id, profile_link_mode, radius_class, realname, remark, status, up_rate, updated_at, username, vlanid1, vlanid2 |
| sys_opr | SysOpr | email, password | created_at, id, last_login, level, mobile, realname, remark, status, updated_at, username |
| sys_opr_log | SysOprLog | opr_ip | id, opr_name, opt_action, opt_desc, opt_time |
| sys_product_branding | ProductBranding | contact_address, contact_email, contact_phone, contact_whatsapp | accent_color, coverage_areas, hero_headline, hero_subtitle, id, product_name, short_name, sla_latency, sla_uptime, tagline, ticker_text, updated_at |
| syslog_event | SyslogEvent | nas_ip | created_at, facility, id, message, severity, severity_name, synthetic, tag |
| tenant | Tenant | billing_address, contact_email, contact_phone | company_name, created_at, id, kind, name, slug, status, tax_id, updated_at |
| tenant_membership | TenantMembership | token_version | created_at, level, operator_id, status, tenant_id, updated_at |

## Greys on the printed sheet

Each operational token converted to the grey its WCAG luminance dries to, then measured against the white sheet the product's own print rule forces. The floor is 4.5:1, the AA requirement for the 10.9px chip label the skin prints.

| mode | token | hex | grey 0-255 | grey vs paper | floor | verdict |
| --- | --- | --- | ---: | --- | --- | --- |
| light | error.dark | #8f1b12 | 17 | 18.88:1 | 4.50:1 | pass |
| light | error.light | #d9483d | 50 | 12.82:1 | 4.50:1 | pass |
| light | error.main | #b42318 | 28 | 17.04:1 | 4.50:1 | pass |
| light | info.dark | #075d68 | 23 | 17.93:1 | 4.50:1 | pass |
| light | info.light | #10a6b5 | 78 | 8.32:1 | 4.50:1 | pass |
| light | info.main | #087e8b | 43 | 14.16:1 | 4.50:1 | pass |
| light | primary.dark | #485600 | 20 | 18.42:1 | 4.50:1 | pass |
| light | primary.light | #7d9400 | 65 | 10.21:1 | 4.50:1 | pass |
| light | primary.main | #5d7000 | 35 | 15.72:1 | 4.50:1 | pass |
| light | secondary.dark | #8c0e43 | 16 | 19.03:1 | 4.50:1 | pass |
| light | secondary.light | #d43a78 | 47 | 13.39:1 | 4.50:1 | pass |
| light | secondary.main | #b01355 | 26 | 17.40:1 | 4.50:1 | pass |
| light | success.dark | #0d6538 | 25 | 17.58:1 | 4.50:1 | pass |
| light | success.light | #24a862 | 75 | 8.72:1 | 4.50:1 | pass |
| light | success.main | #13783f | 36 | 15.52:1 | 4.50:1 | pass |
| light | warning.dark | #744300 | 20 | 18.42:1 | 4.50:1 | pass |
| light | warning.light | #c27a12 | 65 | 10.21:1 | 4.50:1 | pass |
| light | warning.main | #925500 | 32 | 16.29:1 | 4.50:1 | pass |
| dark | error.dark | #ef4444 | 58 | 11.37:1 | 4.50:1 | pass |
| dark | error.light | #fca5a5 | 128 | 3.95:1 | 4.50:1 | FAIL |
| dark | error.main | #f87171 | 84 | 7.57:1 | 4.50:1 | pass |
| dark | info.dark | #2daab5 | 83 | 7.69:1 | 4.50:1 | pass |
| dark | info.light | #84e0e7 | 163 | 2.52:1 | 4.50:1 | FAIL |
| dark | info.main | #55cbd5 | 126 | 4.06:1 | 4.50:1 | FAIL |
| dark | primary.dark | #16a34a | 68 | 9.74:1 | 4.50:1 | pass |
| dark | primary.light | #4ade80 | 141 | 3.32:1 | 4.50:1 | FAIL |
| dark | primary.main | #22c55e | 105 | 5.49:1 | 4.50:1 | pass |
| dark | secondary.dark | #d51a69 | 41 | 14.55:1 | 4.50:1 | pass |
| dark | secondary.light | #ff70ad | 91 | 6.79:1 | 4.50:1 | pass |
| dark | secondary.main | #ff2e88 | 64 | 10.37:1 | 4.50:1 | pass |
| dark | success.dark | #32b76b | 91 | 6.79:1 | 4.50:1 | pass |
| dark | success.light | #8ae8ad | 169 | 2.35:1 | 4.50:1 | FAIL |
| dark | success.main | #62d990 | 138 | 3.45:1 | 4.50:1 | FAIL |
| dark | warning.dark | #f59e0b | 112 | 4.95:1 | 4.50:1 | pass |
| dark | warning.light | #fcd34d | 173 | 2.24:1 | 4.50:1 | FAIL |
| dark | warning.main | #fbbf24 | 148 | 3.03:1 | 4.50:1 | FAIL |

## Status marks told apart without hue

The same tokens measured against each other on the sheet. A pair below 3:1 is two statuses that print as the same grey, so the distinction has to live in the label.

| mode | first | second | grey contrast | distinct at 3:1 |
| --- | --- | --- | --- | --- |
| dark | success.main | warning.main | 1.14:1 | FAIL |
| dark | info.main | success.main | 1.18:1 | FAIL |
| dark | info.main | warning.main | 1.34:1 | FAIL |
| dark | info.main | primary.main | 1.35:1 | FAIL |
| dark | error.main | secondary.main | 1.37:1 | FAIL |
| dark | error.main | primary.main | 1.38:1 | FAIL |
| dark | primary.main | success.main | 1.59:1 | FAIL |
| dark | primary.main | warning.main | 1.81:1 | FAIL |
| dark | error.main | info.main | 1.87:1 | FAIL |
| dark | primary.main | secondary.main | 1.89:1 | FAIL |
| dark | error.main | success.main | 2.19:1 | FAIL |
| dark | error.main | warning.main | 2.50:1 | FAIL |
| dark | info.main | secondary.main | 2.55:1 | FAIL |
| dark | secondary.main | success.main | 3.00:1 | pass |
| dark | secondary.main | warning.main | 3.42:1 | pass |
| light | primary.main | success.main | 1.01:1 | FAIL |
| light | error.main | secondary.main | 1.02:1 | FAIL |
| light | primary.main | warning.main | 1.04:1 | FAIL |
| light | error.main | warning.main | 1.05:1 | FAIL |
| light | success.main | warning.main | 1.05:1 | FAIL |
| light | secondary.main | warning.main | 1.07:1 | FAIL |
| light | error.main | primary.main | 1.08:1 | FAIL |
| light | info.main | success.main | 1.10:1 | FAIL |
| light | error.main | success.main | 1.10:1 | FAIL |
| light | primary.main | secondary.main | 1.11:1 | FAIL |
| light | info.main | primary.main | 1.11:1 | FAIL |
| light | secondary.main | success.main | 1.12:1 | FAIL |
| light | info.main | warning.main | 1.15:1 | FAIL |
| light | error.main | info.main | 1.20:1 | FAIL |
| light | info.main | secondary.main | 1.23:1 | FAIL |

## Paper surface census

How many code lines in web/src write each paper declaration. A zero on a required token is the finding, not a missing row.

| declaration | lines in web/src |
| --- | ---: |
| A4 named on the sheet rule | 0 |
| backgrounds asked of the browser | 0 |
| backgrounds asked, webkit spelling | 0 |
| card kept whole across a break | 2 |
| card kept whole, css spelling | 0 |
| chrome removed | 9 |
| everything but the region erased | 7 |
| ink forced black | 2 |
| page geometry declared | 0 |
| panel fill erased in the sheet region | 0 |
| panel fill erased on paper | 1 |
| paper rules written | 10 |
| print action wired | 9 |
| sheet forced white | 1 |

### Files that write a paper rule

| file | declarations | incident surface |
| --- | --- | --- |
| web/src/pages/HotspotVouchersPage.tsx | paper rules written; print action wired; card kept whole across a break; ink forced black; chrome removed | no |
| web/src/resources/accounting.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/isp.tsx | paper rules written; print action wired; sheet forced white; ink forced black; panel fill erased on paper; chrome removed | no |
| web/src/resources/nas.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/nodes.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/onlineSessions.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/operators.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/radiusProfiles.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |
| web/src/resources/radiusUsers.tsx | paper rules written; print action wired; everything but the region erased; chrome removed | no |

### Incident surfaces

| file | print path |
| --- | --- |
| web/src/components/CustomMenu.tsx | no |
| web/src/components/CustomerTimeline.tsx | no |
| web/src/components/NocImpactPanel.tsx | no |
| web/src/pages/ODPPage.tsx | no |
| web/src/pages/OperationsPage.tsx | no |
| web/src/pages/TroubleTicketsPage.tsx | no |

## Grunge channels on paper

The manga-ink look is a set of paints. This counts each one in the token file and asks whether any @media print block in the tree names the property it draws on.

| channel | css property | lines in theme.ts | paper rules naming it | disposition |
| --- | --- | ---: | ---: | --- |
| ink panel outline | border | 8 | 1 | a paper rule names the property |
| ink rule under a header | border | 7 | 1 | a paper rule names the property |
| ink divider above a footer | border | 1 | 1 | a paper rule names the property |
| hard offset shadow | box-shadow | 6 | 2 | a paper rule names the property |
| halftone dot grid | background-image | 1 | 0 | no paper rule names the property |
| paper tint | background-color | 5 | 1 | a paper rule names the property |
| header band | background-color | 4 | 1 | a paper rule names the property |
| uppercase display titles | text-transform | 9 | 0 | no paper rule names the property |
| monospaced identifiers | font-family | 4 | 0 | no paper rule names the property |
| reserved status color | color | 0 | 2 | the skin never writes it |

## Page budget

One A4 sheet, converted at 96 css px per inch, less the margin the product never declares - the tree writes no @page rule, so the margin below is an assumption and the fit is reported with its spare room.

| measure | value |
| --- | --- |
| A4 sheet | 793.7 x 1122.5 px |
| assumed margin | 48.0 px (12.7 mm) |
| printable height | 1026.5 px |
| title band | 32.8 px |
| confidence strip | 49.8 px |
| one slot row | 28.8 px |
| rows the sheet holds | 32 |
| rows the ledger allows | 108 |
| spare | -2173.9 px |
| fits one page | FAIL |

## State vocabulary and its non-color channels

The chip colors come from the product's own statusTone map. Where several states share a color, only the label tells them apart once the sheet is grey.

| color | states sharing it | collisions |
| --- | --- | --- |
| default | closed, expired, low, normal, used | yes |
| error | critical, down, failed, open, urgent | yes |
| info | assigned, medium | yes |
| success | active, online, resolved, unused, up | yes |
| warning | high, in_progress, pending, reversed, scheduled | yes |

| channel | lines | first at |
| --- | ---: | --- |
| uppercase status label | 2 | web/src/components/Enterprise.tsx:79 |
| outline-only chip | 2 | web/src/components/Enterprise.tsx:81 |
| translated status text | 1 | web/src/components/Enterprise.tsx:80 |
| monospaced identifiers | 4 | web/src/components/Enterprise.tsx:31 |

## Stale and confidence visibility

The middle clause of the criterion. A field counts as visible on screen when web/src reads it, and on paper only when a print path carries the surface that renders it.

| field | served by | lines in web/src | print paths carrying it | note |
| --- | --- | ---: | ---: | --- |
| confidence | nocImpactResponse | 7 | 0 | three values; inferred means the roster is a guess bounded by the 24h lookback |
| confidence_reason | nocImpactResponse | 0 | 0 | the sentence that says why the level is what it is |
| generated_at | nocImpactResponse | 1 | 0 | the as-of stamp a printed sheet cannot refresh |
| interval_seconds | nocImpactMonitoring | 0 | 0 | how often a reading is due |
| last_checked_at | nocImpactMonitoring | 0 | 0 | when the last reading was taken |
| stale | nocImpactMonitoring | 12 | 0 | the monitor stopped answering, not that the link came back |
| stale_after_seconds | nocImpactMonitoring | 0 | 0 | the bound that decides it, floored at five minutes |
| state | NetMonitorIncident | 2 | 0 | six values, from operational to not_monitored |
| state | nocImpactResponse | 2 | 0 | six values, from operational to not_monitored |
| warnings | nocImpactResponse | 1 | 0 | what the server refused to conclude |

## Pinned statements

Each statement is checked against a code line of a named file after comments are blanked. Drift is reported, never dropped.

| id | statement | file | line | status |
| --- | --- | --- | ---: | --- |
| A01 | the card inherits its privacy policy from the shipped classifier, not from this study | internal/redact/redact.go | 61 | confirmed |
| A02 | customer_name is one of the keys that classifier already treats as subscriber PII | internal/redact/redact.go | 46 | confirmed |
| A03 | the free-text sweep only knows credentials, URL credentials, e-mail and IPv4 - a personal name is not in its vocabulary | internal/redact/redact.go | 56 | confirmed |
| A04 | the NOC impact response is a served type, so its slots are candidates rather than inventions | internal/adminapi/noc_impact.go | 98 | confirmed |
| A05 | the impacted roster selects the customer's name column straight out of the identity table | internal/adminapi/noc_impact.go | 408 | confirmed |
| A06 | confidence is one of three values the server states about its own conclusion | internal/adminapi/noc_impact.go | 28 | confirmed |
| A07 | the stale state is a named value, not an inference the card has to make | internal/adminapi/noc_impact.go | 33 | confirmed |
| A08 | a printed roster is bounded by a 24h lookback the sheet cannot refresh | internal/adminapi/noc_impact.go | 20 | confirmed |
| A09 | staleness has a floor of five minutes so a fast monitor cannot flap the printed state | internal/adminapi/noc_impact.go | 24 | confirmed |
| A10 | the auto-ticket composes the customer's name into the ticket description before the card ever sees it | internal/adminapi/isp_operations.go | 1115 | confirmed |
| A11 | the ticket the card would print is a persisted row with prose columns | internal/domain/isp_operations.go | 80 | confirmed |
| A12 | the console forces the sheet white and the ink black before anything is printed | web/src/resources/isp.tsx | 1192 | confirmed |
| A13 | the same print rule erases the panel fill the grunge skin is made of | web/src/resources/isp.tsx | 1210 | confirmed |
| A14 | a status chip draws an outlined label, so the state survives without a fill | web/src/components/Enterprise.tsx | 81 | confirmed |
| A15 | the state vocabulary that decides chip color is a map in the product, not a fixture | web/src/components/Enterprise.tsx | 67 | confirmed |
| A16 | the skin ships dark as the default, so the dark token set is the one most cards would print | web/src/App.tsx | 126 | confirmed |
| A17 | the ink outline that draws a panel border is one alpha the theme picks per mode | web/src/theme.ts | 72 | confirmed |
| A18 | the halftone texture is a background image, not a border | web/src/theme.ts | 77 | confirmed |
| A19 | the display face that carries the grunge titles is a local font stack, never fetched | web/src/theme.ts | 38 | confirmed |
| A20 | the print action exists on export surfaces the incident screens do not share | web/src/resources/isp.tsx | 947 | confirmed |
| A21 | the scope name the card would print is read off the monitor target row | internal/adminapi/noc_impact.go | 223 | confirmed |
| A22 | the NAS name is assigned to the scope field, not carried in a select list | internal/adminapi/noc_impact.go | 243 | confirmed |
| A23 | the account record stores a person's real name under a key the shipped rules never name | internal/domain/radius.go | 48 | confirmed |
| A24 | the subscriber's own name is a plain indexed column on the customer row, also unnamed by the rules | internal/domain/isp.go | 61 | confirmed |

24 of 24 confirmed, 0 broken, 0 vacated.

## What this round cannot answer

Each row needs something the repository does not have. The status is the deliverable: an explicit no-go, with the reason.

| id | needs | status | why |
| --- | --- | --- | --- |
| G01 | a printed proof sheet from a real monochrome laser | no-go | the greys here are the WCAG luminance conversion a printer approximates; only paper settles it |
| G02 | a page geometry the product declares | no-go | zero @page rules exist in web/src, so the printable box is the browser's default and the fit is computed against an assumed 12.7mm margin |
| G03 | a decision on background graphics | no-go | zero print-color-adjust declarations, so whether the halftone and offset shadow reach paper is a driver setting this round cannot read |
| G04 | a display face that ships with the product | no-go | web/index.html fetches Inter and JetBrains Mono only; the grunge titles fall back to a local Arial Narrow or Impact |
| G05 | an exporter that produces the card | no-go | no PDF dependency exists in web/package.json or go.mod, so the only export path is the browser print dialog |
| G06 | a redaction rule that covers a name in prose | no-go | the card's only defense is refusing the prose slots and tracing each name-shaped slot to its assignment; closing the gap means editing internal/redact, a product change outside a RISET round |
| G07 | incident data from a running deployment | no-go | the ledger measures served shapes, not a real outage's values |

## Totals

The run's headline numbers, all of them counts of the tables above.

| measure | value |
| --- | --- |
| slots in the ledger | 129 |
| slots a card may print | 108 |
| refused by the shipped rules | 7 |
| refused by a query provenance | 10 |
| refused by composed prose | 3 |
| refused by the name trace | 1 |
| refusals a shipped rule already covers | 8 |
| refusals only this ledger covers | 13 |
| name-shaped slots served | 6 |
| name-shaped slots printed from a traced non-subscriber row | 3 |
| name-shaped slots held off | 3 |
| identity tables | 16 |
| their guarded columns | 30 |
| their unguarded columns served | 235 |
| light tokens readable on paper | 18 of 18 |
| dark tokens readable on paper | 10 of 18 |
| status pairs indistinguishable in grey | 28 |
| files writing a paper rule | 9 |
| incident surfaces | 6 |
| incident surfaces with a print path | 0 |
| @page rules in web/src | 0 |
| print-color-adjust declarations | 0 |
| grunge channels no paper rule names | 3 |
| grunge channels a paper rule erases | 6 |
| states in the vocabulary | 22 |
| colors shared by more than one state | 5 |
| stale/confidence fields visible on screen | 6 |
| stale/confidence fields with no print path carrying them | 10 |
| rows the ledger wants against the sheet | 108 of 32 |
| statements pinned | 24 of 24 |
| gaps open | 7 |
| packages this study imports | 15 |

Digest: `e667b8e47e590ac663f3b087de00fb1d7dcceb7db9724a22bfb9f764c26453ff`

## Verification

```
go build -p 1 ./...                                # clean
go vet -p 1 ./internal/incidentcard/               # clean
gofmt -l internal/incidentcard/                   # empty
golangci-lint run ./internal/incidentcard/         # 0 issues
go test -p 1 -count=1 ./internal/incidentcard/     # ok (22 gates run + the dump harness skipped, 24 of 24 statements confirmed)
go test -p 1 -count=1 ./internal/...               # ok, whole internal tree
grep -rn incidentcard --include=*.go internal/ cmd/ | grep -v ^internal/incidentcard/   # 0 importers
```

The tables above come from
`CARD_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/incidentcard/`,
so every number in the prose is the same run. The digest
`e667b8e47e590ac663f3b087de00fb1d7dcceb7db9724a22bfb9f764c26453ff` locks 129
ledger rows, 6 name-trace rows, 16 identity tables, 36 greys, 30 status pairs,
14 paper declarations, 10 ink channels, 22 states, 10 visibility rows, 24
pinned statements and 7 open limits - and it covers the guard flag and the
assignment each name-shaped slot reads, so a card that quietly started printing
`isp_customer.name` would change it. It changes the moment the tree changes, and
`TestTheDigestSurvivesATickingClock` proves it does not change when only the
clock does.

## No-go register

Seven limits stay open and none is closable from the tree. Four need an object
this round cannot produce (a proof sheet, a font, an exporter, a live outage),
two are browser-and-driver settings the repository does not declare, and one is a
product change in `internal/redact`. Each is recorded as **no-go**, with the
reason, rather than as done:

| id | needs | kind of asset | verdict |
| --- | --- | --- | --- |
| `G01` | a printed proof sheet from a real monochrome laser | hardware | **no-go** - the greys here are the WCAG luminance conversion a printer approximates; only paper settles whether `#fbbf24` at 10.9px is readable |
| `G02` | a page geometry the product declares | a `@page` rule in `web/src` | **no-go** - zero declarations exist, so the printable box is the browser's default and the 32-row budget is computed against an assumed 12.7mm margin |
| `G03` | a decision on background graphics | a `print-color-adjust` declaration | **no-go** - zero declarations exist, so whether the halftone grid and the hard offset shadow reach paper is a driver setting this round cannot read |
| `G04` | a display face that ships with the product | a licensed font asset | **no-go** - `web/index.html` fetches Inter and JetBrains Mono only; the grunge titles fall back to whatever Arial Narrow or Impact the printing machine has |
| `G05` | an exporter that produces the card | a PDF dependency and a template | **no-go** - none exists in `web/package.json` or `go.mod`, so the only export path is the browser print dialog, with its own headers and footers |
| `G06` | a redaction rule that covers a name in prose | a change to `internal/redact` | **no-go** - the card's only defense is refusing the 3 composed slots and tracing each name-shaped key; closing the gap means editing the shipped classifier, which is product scope, not a RISET round |
| `G07` | incident data from a running deployment | a live outage | **no-go** - the ledger measures served shapes; what a real ticket's `summary`, `body` or `message` contains on a Tuesday afternoon is not derivable from source |

## Residual limits

- **Free text is the hole the ledger cannot plug.** Every allowed slot whose type
  is `string` and whose content a technician typed - `TroubleTicket.subject` is
  refused, but `resolution_notes`, `NetMonitorIncident.summary`,
  `NotificationOutbox.body`, `SyslogEvent.message` and `systemLogEntry.message`
  all print - could contain a customer's name written by hand. The shipped
  free-text sweep only recognizes credentials, URL credentials, e-mail and IPv4
  (`A03`), so a name typed into a note passes every check in this study. `G06`
  and `G07` are the same hole seen from two sides.
- **The name trace follows assignments, not data flow.** It reads composite
  literals and field writes in `internal/adminapi`; a name assembled two hops
  away - through a service call, a helper that returns a string, or a struct
  copied field by field - is recorded as an unresolvable expression and refused,
  which is the conservative direction but is not the same as proof. That is
  exactly why `SyslogEvent.severity_name` is off the sheet.
- **`subscriberRows = {isp_customer, radius_user}` is a judgement.** It is the
  round's one interpretive choice, pinned by `A23`/`A24` and stated in the open
  above. A reader who disagrees - who counts `sys_opr.realname` as subscriber
  data too - gets a different set of printable names, and the ledger would have
  to be re-run with that row added.
- **The greys are a model of paper, not paper.** WCAG relative luminance is the
  best available proxy from source, and `TestTheCardFitsTheSheetItClaims` prices
  the readable floor at the chip size the skin actually renders (10.9px, AA's
  4.5:1), but toner density, dot gain and a driver's dithering are outside the
  repository. `G01`.
- **The page is assumed, and the budget is per row.** 32 rows come from an
  undeclared margin and one slot per line; a real card with a two-column roster,
  a per-foram header band and a halftone footer would fit fewer, and the
  halftone and shadow may or may not print at all. `G02`, `G03`.
- **Counts, not review.** 108 printable slots is what the four checks allow, not
  what an operator needs on a sheet during an outage; selecting the third of them
  that belongs on a card, and the layout that carries them, is the design work
  this round hands on.
