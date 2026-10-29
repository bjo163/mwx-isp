# FUT-020 — Which per-tenant operational statistics a response can already reconstruct

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-020` (`TR-F033`, P3, RISET) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Anchor: `internal/adminapi/dashboard.go`. Prototype: `internal/opstats`
(read-only, imported by nothing).
Criterion under test: **Fixture tenant data unik tidak bisa direkonstruksi dari
response; publish hanya bila governance jelas.**

## The question

The criterion has two halves and they are not measurable the same way.

1. **"Fixture tenant data unik tidak bisa direkonstruksi dari response."** The
   backlog row names a fixture, and the repository has no tenant populations to
   measure, so this round builds one: six tenants spanning the whole commercial
   shape of the product - a one-customer RT/RW Net up to a 120-customer town -
   with every stored row carrying a marker no other row shares. A published
   number is then *reconstructable* when the cell it covers holds one row: a
   `SUM` over one invoice is that invoice's own balance, a `COUNT` of one over
   `radius_user` is a statement that the tenant has exactly one subscriber, and
   a percentage served beside the two counts it divides is neither of those
   things - it is a restatement. 11 registered surfaces, 157 served fields, 73
   of them aggregates this study scores.
2. **"Publish hanya bila governance jelas."** This half is an absence scan. A
   publication gate would have to exist somewhere a product could turn it on,
   say what it withholds, and be written down; so the run counts the config keys
   that speak of statistics, the vocabulary a suppression rule would be named
   with, whether the shipped redactor reaches a statistics handler at all,
   whether the tenancy contract says a word about an aggregate leaving its
   tenant, and what the only per-tenant opt-in the product already ships looks
   like.

The honest difficulty is that the first half cannot be answered with a yes or no
about the code as it ships, because nothing publishes across tenants today. What
the run can say - and does, in numbers - is which of these statistics would
survive a publication, and that none of the machinery that would decide
*whether* is in the tree.

## What counts as evidence

A statement is pinned to a *code* line: comments are blanked per file before
matching, the match is a whole identifier rather than a substring, and a symbol
that moves or disappears is reported as `broken` or `vacated` instead of quietly
dropping the row. Absence scans walk `internal`, `pkg` and `cmd` for Go files and
exclude this study's own directory, so the prototype cannot be its own evidence -
which matters here more than in most rounds, because a suppression-vocabulary
scan would otherwise be satisfied by the very identifiers this study searches for.

Three choices are interpretive rather than parsed, and all three are named here
because they decide verdicts:

- **`personalTables`** - the eight rows whose counts describe a person
  (`isp_customer`, `isp_subscription`, `isp_invoice`, `isp_payment`,
  `isp_trouble_ticket`, `radius_user`, `radius_online`, `radius_accounting`)
  versus the four that describe a thing the operator owns (`radius_profile`,
  `isp_ipam_pool`, `net_monitor_target`, `tenant`). A cell of one over the first
  set is a subscriber; over the second it is a device.
- **`moneyColumns` / `bytesColumns`** - which aggregated column makes a `SUM`
  publish an amount rather than a count. A `SUM` over one row is only the worst
  kind of exposure if the column is money or bytes, so this list is what turns
  `outstanding` into "publishes one stored value verbatim".
- **the cell rule** - `username`, `customer_no`, `profile_name`, `date`, `hour`,
  `table`, `row`, else `tenant`, first one present in the served shape winning. It
  says how many stored rows one published value covers, which is the whole basis
  of the reconstructability tables.

The fixture is the study's model of a deployment, not the repository's data: every
number in the reconstructability and floor tables is arithmetic about that fixture
plus the aggregate verb the handler writes, never a claim about a real customer's
balance. Nothing is clock-dependent - the digest is a sha256 over the measured rows
and a run repeated after a ticking clock is byte-identical
(`TestTheDigestSurvivesATickingClock`).

## The prototype

| File | Role |
| --- | --- |
| `opstats.go` | `Run`, the `Result` this memo splices, the 30-counter totals table, the bucket rule and the sha256 digest. |
| `evidence.go` | File walking, comment blanking that keeps line numbers, bounded identifier matching, struct-field reading and the import-set reader. |
| `surfaces.go` | The eleven registered statistics surfaces, their served types expanded to every published key, the statement each number is attributed to, and the division a rate field is composed from. |
| `scoping.go` | How each surface reaches the database - the shipped tenant callback, a written comparison, a default-tenant fallback, or a deliberately cross-tenant platform view. |
| `fixture.go` | The synthetic installation: six tenants, 1189 stored rows, twelve tables, every marker unique, and the cell rule that groups them. |
| `model.go` | The fixture run through each aggregate verb: smallest cell, tenants with a cell of one, tenants blanked at each floor, and the verdict that answers the criterion per field. |
| `governance.go` | The permission half - the config registry, the suppression vocabulary, the redactor's reach, the fallback sites, the contract's own words and the shipped opt-in precedent. |
| `claims.go` | 26 statements with their checks and the 7-item gap register. |
| `render.go` | Emits every table in this memo from one run. |
| `opstats_test.go` | 23 gates: the fixture, the attribution, the buckets, the floors, the governance counts and the no-write/no-importer guarantees. |
| `render_test.go` | 5 gates on the report itself: every section present, digest carried, tables unbroken, limits open, statements lined. The 29th test function, `TestReportMarkdownDump`, is the harness that emits the tables below; it skips unless `OPSTATS_DUMP=1`. |

Nothing in `internal/` or `cmd/` imports it (`TestNothingImportsTheStudy` plus a
grep over the tree that returns 0), it opens no database connection
(`TestTheStudyReadsNoDatabase`), and the only files it touches are opened for
reading (`TestTheStudyWritesNothing` snapshots `internal/adminapi`,
`internal/domain`, `internal/tenancy` and `internal/app` before and after a run).
Its 15 imports are stdlib only (`TestTheImportAllowlistIsNotVacuous`).

## What the run says

**No aggregate leaves its tenant today, and the reason is the tenancy layer, not a
statistics policy.** Of the 11 surfaces, 6 are scoped by the GORM tenant callback
registered once in `internal/tenancy/scope.go` (`A05`/`A06`), 1 by a comparison
written in the handler (`/dashboard/stats`, `dashboard.go:284`), 2 by a
default-tenant fallback, and 2 deliberately cross-tenant (the platform tenant list
and the backup manifest). Counting the aggregates that sit over a personal table
*and* on a surface meant to cross tenants gives **0**. What the two platform
surfaces actually publish is identity, not statistics: `/platform/tenants` serves
the persisted `tenant` row including `tax_id` and `billing_address` (`A16`, `A17`),
and `/system/backup` serves `record_counts` - per-table totals for the whole
installation (`A20`) - which is a count over every tenant's rows at once and is the
one cross-tenant number the product publishes today. It is not scored as an
aggregate here because the census never attributes it to a verb the fixture can
run.

**On the fixture, more than a third of the published numbers are a single stored
row.** 73 aggregates scored: **3 publish a stored value verbatim**, **24 say the
tenant has exactly one**, **1 is a count in disguise**, 5 cover a thing the
operator owns, 0 stay above a cell of one, and 40 have no source table the study
can attribute. The three verbatim cases are the ones the criterion is about:
`/dashboard/isp-stats outstanding` is `SUM(isp_invoice.balance)` and a
one-invoice tenant publishes that invoice's exact amount in rupiah;
`/dashboard/stats today_input_gb` and `today_output_gb` are `SUM` over
`radius_accounting`, and a tenant whose only session yesterday was one subscriber
publishes that subscriber's own traffic total. The 24 membership cases are the same
shape one level down: `customers = 1` names the tenant's size, which is itself a
unique fact about an RT/RW Net account.

**A percentage is not softer than its numerator when the numerator ships beside
it.** `ipv6_stats.adoption_rate` is composed at `dashboard.go:343` as
`OnlineWithIPv6 / onlineTotal`, and the `onlineTotal` it divides is passed in as
`stats.OnlineUsers` at `dashboard.go:150` - so the ratio divides
`ipv6_stats.online_with_ipv6` by `online_users`, both served in the same response.
Publishing the three together adds no information; publishing the rate and dropping
the two counts would still let a reader recover one from the other. This is the
only rate in the census whose operands both resolve: the monitor's
`uptime_percent` divides two locals the response never serves
(`reachableCount / samples`), and `packet_loss_percent` is composed outside the
file the surface is registered in, so both stay unattributed rather than being
credited with a defense the study has not read.

**A floor of two removes every exposure and takes the small end of the customer
base with it.** At floor 1, 27 fields still reconstruct a stored value. At floor
2, **0 reconstruct and 31 are suppressed somewhere** - and 31 of the 73 scored
fields publish nothing at all for at least one tenant. Raising the floor to 3 or 5
suppresses one more field and costs no additional coverage, which is the useful
part of the table: the price is paid almost entirely at `k = 2`, and it is paid by
the tenants the fixture models as `solo` and `pair`. A benchmark whose cell floor
is 2 is a benchmark that does not describe an RT/RW Net installation, because the
tenant that made the cell a cell of one is the tenant that gets withheld.

**The permission half of the criterion has no mechanism in the tree.** 36 config
keys in the registry `internal/app/config_schemas.json` renders on the system
config page, and **0** of them speak of statistics, benchmarks, aggregates,
anonymisation or sharing (`A22`). The eight identifiers a suppression rule would be
named with - `kAnonymity`, `minCell`, `CellFloor`, `SuppressCell`, `anonymiz`,
`suppress`, `Pseudonymize`, `RiskAcknowledgedAt` - produce **0 matches inside the
files that serve statistics**; `Pseudonymize`'s 12 lines are the log and
support-bundle tool in `internal/redact` (`A21`), and the shipped redactor is
called by **0** statistics files. `docs/feature-checklist.md`'s own TR-F033 row
names no statistic, aggregate, benchmark or publication in the wording before the
research note this round appended to the same row (`P06` = 0;
`TestTheContractRowMarksItsResearchNote` re-reads that row and fails if the
contract itself starts naming them, or if the note stops being this study's
citation). So "publish hanya bila governance jelas" currently has no antecedent:
there is nothing to turn on, nothing that says what would be withheld, and no
contract text that would permit or forbid it.

**One governance shape does ship, and it is the one a publication gate should
copy.** `domain.NotificationSettings` (`internal/domain/network_monitor.go:73`) is
a per-tenant opt-in row with `whatsapp_enabled`, a stored
`RiskAcknowledgedAt` and a `TenantID` scope - 7 fields, `A23`, `P05`. The product
already treats "an action that carries a subscriber's data to someone outside the
tenant" as an explicit, acknowledged, per-tenant opt-in. That is the mechanism the
criterion asks the statistics path to have, and it is not used there.

**The live hazard is mis-attribution, not publication.** **69** sites read a tenant
id and substitute the default tenant when no context is present; **8 of them are
production code** (`context.go:28`, `noc_impact.go:181`, `customer_timeline.go:92`,
`isp.go:1297`, `session_action_queue.go:43` and `:112`, `system_backup.go:232`,
`platform_tenants.go:354`) and 61 are tests. A background aggregation job has no
request context, so the first thing a published per-tenant statistic would meet is
not a privacy gate but `A09`: a job without a tenant reports the default tenant's
numbers rather than refusing. `/network/impact` and
`/isp/customers/:id/timeline` are already classed `fallback` by this run, and
`dashboard_test.go` asserts tenant scope **0** times (`P08`) - the largest
aggregate surface in the product has no test that a tenant sees only its own
numbers.

**The census is mostly attributed, and where it is not, that is the finding.** 33
of 73 aggregates reach a table and a verb; 40 do not, and the 40 are not a gap in
the search so much as a description of the product: 28 are the two
`TrafficSeriesDTO` surfaces whose per-bucket numbers are accumulated in Go from
 SNMP and accounting samples rather than produced by a SQL verb, 4 are dashboard
series (`auth_trend.count`, `profile_distribution.value`, both `traffic_24h`
values) composed the same way, 3 are the monitor's composed rates and latency, 3
are timeline pagination, and 2 are backup constants. A statistic the database
never computed cannot be priced against a stored row, so the run says so instead of
borrowing a neighbour's statement.

## The measured tables

Spliced verbatim from one run of `internal/opstats`.

## Statistics surfaces

Every aggregate surface the console registers, how its query reaches the database, and how many statements go through a domain model (where the tenant callback can reach) versus a raw table (where it cannot).

| surface | audience | handler | served type | scope | evidence | model queries | raw queries |
| --- | --- | --- | --- | --- | --- | ---: | ---: |
| /dashboard/stats | tenant-operator | GetDashboardStats | DashboardStats | explicit | internal/adminapi/dashboard.go:284 | 12 | 1 |
| /dashboard/isp-stats | tenant-operator | getISPDashboardStats | ispDashboardStats | callback | internal/adminapi/isp.go:1584 | 29 | 0 |
| /network/impact | tenant-operator | getNetworkImpact | nocImpactCounts | fallback | internal/adminapi/noc_impact.go:181 | 0 | 5 |
| /network/monitor-targets | tenant-operator | listMonitorTargets | monitorTargetDTO | callback | internal/adminapi/network_monitor.go:110 | 4 | 0 |
| /network/monitor-targets/:id/traffic | tenant-operator | getMonitorTargetTraffic | TrafficSeriesDTO | callback | internal/adminapi/telemetry.go:91 | 1 | 0 |
| /radius/traffic/user/:username | tenant-operator | getUserTraffic | TrafficSeriesDTO | callback | internal/adminapi/telemetry.go:91 | 1 | 0 |
| /network/ipam/pools | tenant-operator | listIPAMPools | IPAMPool | callback | internal/adminapi/isp_operations.go:633 | 13 | 0 |
| /network/diagnostics/flapping | tenant-operator | detectFlappingSubscribers | flappingSubscriber | callback | internal/adminapi/isp_operations.go:1470 | 13 | 0 |
| /isp/customers/:id/timeline | tenant-operator | getCustomerTimeline | customerTimelineResponse | fallback | internal/adminapi/customer_timeline.go:92 | 6 | 0 |
| /platform/tenants | platform-admin | listPlatformTenants | Tenant | platform | internal/adminapi/platform_tenants.go:341 | 6 | 0 |
| /system/backup | platform-admin | backupSystem | backupManifest | platform | internal/adminapi/system_backup.go:292 | 0 | 0 |

11 surfaces, 5 scope classes: 6 callback, 1 explicit, 2 fallback, 2 platform, 0 unattributed.

## The fixture

The synthetic installation the criterion names: six tenants, every stored row carrying a value no other row shares. These are the populations a statistic is measured over, not real customers.

| tenant | isp_customer | isp_invoice | isp_ipam_pool | isp_payment | isp_subscription | isp_trouble_ticket | net_monitor_target | radius_accounting | radius_online | radius_profile | radius_user | tenant |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| solo | 1 | 1 | 1 | 1 | 1 | 1 | 3 | 1 | 1 | 2 | 1 | 1 |
| pair | 2 | 2 | 1 | 2 | 2 | 0 | 3 | 2 | 2 | 2 | 2 | 1 |
| hamlet | 5 | 5 | 2 | 5 | 5 | 2 | 3 | 5 | 5 | 3 | 5 | 1 |
| village | 12 | 24 | 2 | 12 | 12 | 3 | 3 | 9 | 9 | 4 | 12 | 1 |
| district | 40 | 40 | 3 | 40 | 40 | 5 | 3 | 22 | 22 | 5 | 40 | 1 |
| town | 120 | 120 | 4 | 120 | 120 | 9 | 3 | 60 | 60 | 6 | 120 | 1 |

6 tenants, 1189 stored rows across 12 tables, markers unique: yes.

## Aggregate census

Every field a statistics response serves, with the statement that produces it: the verb the handler writes, the table the verb reads, and the cell rule that says how many stored rows one published value covers.

| surface | field | type | shape | measure | verb | table | cell | attribution | line |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: |
| /dashboard/isp-stats | active_subscriptions | int64 | scalar | count | COUNT | isp_subscription | tenant | same line | 1589 |
| /dashboard/isp-stats | customers | int64 | scalar | count | COUNT | isp_customer | tenant | same line | 1588 |
| /dashboard/isp-stats | invoices_this_month | int64 | scalar | count | COUNT | isp_invoice | tenant | same line | 1592 |
| /dashboard/isp-stats | online_users | int64 | scalar | count | COUNT | radius_online | tenant | same line | 1591 |
| /dashboard/isp-stats | outstanding | int64 | scalar | money | SUM | isp_invoice | tenant | same line | 1594 |
| /dashboard/isp-stats | overdue | int64 | scalar | count | COUNT | isp_invoice | tenant | same line | 1595 |
| /dashboard/isp-stats | payments_this_month | int64 | scalar | count | COUNT | isp_payment | tenant | same line | 1593 |
| /dashboard/isp-stats | suspended_subscriptions | int64 | scalar | count | COUNT | isp_subscription | tenant | same line | 1590 |
| /dashboard/stats | auth_trend.count | int64 | series | count | composed | - | date | assignment only | 251 |
| /dashboard/stats | auth_trend.date | string | series | time | - | - | date | - | 53 |
| /dashboard/stats | disabled_users | int64 | scalar | count | COUNT | radius_user | tenant | same line | 124 |
| /dashboard/stats | expired_users | int64 | scalar | count | COUNT | radius_user | tenant | same line | 127 |
| /dashboard/stats | ipv6_stats.adoption_rate | float64 | scalar | rate | - | - | tenant | ratio read from its own assignment | 343 |
| /dashboard/stats | ipv6_stats.online_with_delegated_prefix | int64 | scalar | count | COUNT | radius_online | tenant | same line, model from the query built above | 322 |
| /dashboard/stats | ipv6_stats.online_with_framed_prefix | int64 | scalar | count | COUNT | radius_online | tenant | same line, model from the query built above | 319 |
| /dashboard/stats | ipv6_stats.online_with_ipv6 | int64 | scalar | count | COUNT | radius_online | tenant | same line, model from the query built above | 316 |
| /dashboard/stats | ipv6_stats.online_with_ipv6_address | int64 | scalar | count | COUNT | radius_online | tenant | same line, model from the query built above | 316 |
| /dashboard/stats | ipv6_stats.users_with_delegated_prefix | int64 | scalar | count | COUNT | radius_user | tenant | same line, model from the query built above | 338 |
| /dashboard/stats | ipv6_stats.users_with_static_address | int64 | scalar | count | COUNT | radius_user | tenant | same line, model from the query built above | 335 |
| /dashboard/stats | online_users | int64 | scalar | count | COUNT | radius_online | tenant | same line | 118 |
| /dashboard/stats | profile_distribution.profile_id | int64 | series | id | - | - | profile_name | - | 70 |
| /dashboard/stats | profile_distribution.profile_name | string | series | label | - | - | profile_name | - | 71 |
| /dashboard/stats | profile_distribution.value | int64 | series | count | composed | - | profile_name | assignment only | 298 |
| /dashboard/stats | today_acct_count | int64 | scalar | count | COUNT | radius_accounting | tenant | same line | 133 |
| /dashboard/stats | today_auth_count | int64 | scalar | count | COUNT | radius_online | tenant | same line | 130 |
| /dashboard/stats | today_input_gb | float64 | scalar | bytes | SUM | radius_accounting | tenant | above line, model from the query built above | 141 |
| /dashboard/stats | today_output_gb | float64 | scalar | bytes | SUM | radius_accounting | tenant | above line, model from the query built above | 141 |
| /dashboard/stats | total_profiles | int64 | scalar | count | COUNT | radius_profile | tenant | same line | 121 |
| /dashboard/stats | total_users | int64 | scalar | count | COUNT | radius_user | tenant | same line | 115 |
| /dashboard/stats | traffic_24h.download_gb | float64 | series | bytes | composed | - | hour | assignment only | 265 |
| /dashboard/stats | traffic_24h.hour | string | series | time | - | - | hour | - | 61 |
| /dashboard/stats | traffic_24h.upload_gb | float64 | series | bytes | composed | - | hour | assignment only | 264 |
| /isp/customers/:id/timeline | customer_id | int64 | scalar | id | - | - | tenant | - | 45 |
| /isp/customers/:id/timeline | events.customer_id | int64 | series | id | - | - | tenant | - | 31 |
| /isp/customers/:id/timeline | events.event_id | string | series | label | - | - | tenant | - | 26 |
| /isp/customers/:id/timeline | events.occurred_at | time.Time | series | other | - | - | tenant | - | 29 |
| /isp/customers/:id/timeline | events.severity | string | series | label | - | - | tenant | - | 35 |
| /isp/customers/:id/timeline | events.source | string | series | label | - | - | tenant | - | 27 |
| /isp/customers/:id/timeline | events.source_path | string | series | label | - | - | tenant | - | 34 |
| /isp/customers/:id/timeline | events.subscription_id | int64 | series | id | - | - | tenant | - | 32 |
| /isp/customers/:id/timeline | events.summary | string | series | label | - | - | tenant | - | 33 |
| /isp/customers/:id/timeline | events.tenant_id | int64 | series | id | - | - | tenant | - | 30 |
| /isp/customers/:id/timeline | events.type | string | series | label | - | - | tenant | - | 28 |
| /isp/customers/:id/timeline | from | time.Time | scalar | other | - | - | tenant | - | 46 |
| /isp/customers/:id/timeline | pagination.page | int | scalar | count | composed | - | tenant | assignment only | 180 |
| /isp/customers/:id/timeline | pagination.page_size | int | scalar | count | composed | - | tenant | assignment only | 180 |
| /isp/customers/:id/timeline | pagination.total | int64 | scalar | count | composed | - | tenant | assignment only | 180 |
| /isp/customers/:id/timeline | sources | []string | series | other | - | - | tenant | - | 48 |
| /isp/customers/:id/timeline | to | time.Time | scalar | other | - | - | tenant | - | 47 |
| /network/diagnostics/flapping | address | string | scalar | label | - | - | username | - | 1460 |
| /network/diagnostics/flapping | customer_name | string | scalar | label | - | - | username | - | 1458 |
| /network/diagnostics/flapping | customer_no | string | scalar | label | - | - | username | - | 1457 |
| /network/diagnostics/flapping | disconnect_count | int | scalar | count | composed | radius_accounting | username | assignment only | 1525 |
| /network/diagnostics/flapping | last_seen_time | time.Time | scalar | other | - | - | username | - | 1464 |
| /network/diagnostics/flapping | last_terminate_cause | string | scalar | label | - | - | username | - | 1463 |
| /network/diagnostics/flapping | odp_code | string | scalar | label | - | - | username | - | 1461 |
| /network/diagnostics/flapping | phone | string | scalar | label | - | - | username | - | 1459 |
| /network/diagnostics/flapping | severity | string | scalar | label | - | - | username | - | 1465 |
| /network/diagnostics/flapping | suggested_resolution | string | scalar | label | - | - | username | - | 1466 |
| /network/diagnostics/flapping | username | string | scalar | label | - | - | username | - | 1456 |
| /network/impact | customers | int64 | scalar | count | COUNT | isp_customer | tenant | above line, chain built in nocImpactSubscriptionQuery | 400 |
| /network/impact | online_sessions | int64 | scalar | count | COUNT | radius_online | tenant | above line | 499 |
| /network/impact | open_tickets | int64 | scalar | count | COUNT | isp_trouble_ticket | tenant | above line | 509 |
| /network/impact | subscriptions | int64 | scalar | count | COUNT | isp_subscription | tenant | above line, chain built in nocImpactSubscriptionQuery | 393 |
| /network/ipam/pools | capacity_label | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 69 |
| /network/ipam/pools | cidr | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 58 |
| /network/ipam/pools | created_at | time.Time | scalar | other | SELECT | isp_ipam_pool | row | persisted column | 70 |
| /network/ipam/pools | description | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 66 |
| /network/ipam/pools | dns_primary | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 62 |
| /network/ipam/pools | dns_secondary | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 63 |
| /network/ipam/pools | gateway | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 61 |
| /network/ipam/pools | id | int64 | scalar | id | SELECT | isp_ipam_pool | row | persisted column | 55 |
| /network/ipam/pools | ip_version | int | scalar | count | - | isp_ipam_pool | row | not found | 0 |
| /network/ipam/pools | name | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 57 |
| /network/ipam/pools | pool_type | string | scalar | label | SELECT | isp_ipam_pool | row | persisted column | 60 |
| /network/ipam/pools | tenantid | int64 | scalar | count | - | isp_ipam_pool | row | not found | 0 |
| /network/ipam/pools | total_ips | int64 | scalar | count | - | isp_ipam_pool | row | not found | 0 |
| /network/ipam/pools | updated_at | time.Time | scalar | other | SELECT | isp_ipam_pool | row | persisted column | 71 |
| /network/ipam/pools | used_ips | int64 | scalar | count | - | isp_ipam_pool | row | not found | 0 |
| /network/monitor-targets | heartbeats.checked_at | time.Time | series | other | - | - | tenant | - | 40 |
| /network/monitor-targets | heartbeats.id | int64 | series | id | - | - | tenant | - | 39 |
| /network/monitor-targets | heartbeats.latency_milliseconds | int64 | series | count | composed | - | tenant | assignment only | 80 |
| /network/monitor-targets | heartbeats.packet_loss_percent | float64 | series | rate | - | - | tenant | composition | 43 |
| /network/monitor-targets | heartbeats.reachable | bool | series | flag | - | - | tenant | - | 41 |
| /network/monitor-targets | snmp_secret_configured | bool | scalar | flag | - | - | tenant | - | 48 |
| /network/monitor-targets | uptime_percent | float64 | scalar | rate | - | - | tenant | ratio read from its own assignment | 85 |
| /network/monitor-targets/:id/traffic | available_interfaces | []string | series | other | - | - | tenant | - | 30 |
| /network/monitor-targets/:id/traffic | avg_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 230 |
| /network/monitor-targets/:id/traffic | avg_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 231 |
| /network/monitor-targets/:id/traffic | counter_resets | int | scalar | count | composed | - | tenant | assignment only | 240 |
| /network/monitor-targets/:id/traffic | current_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 226 |
| /network/monitor-targets/:id/traffic | current_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 227 |
| /network/monitor-targets/:id/traffic | data_available | bool | scalar | flag | - | - | tenant | - | 47 |
| /network/monitor-targets/:id/traffic | entity_id | string | scalar | label | - | - | tenant | - | 27 |
| /network/monitor-targets/:id/traffic | in_label | string | scalar | label | - | - | tenant | - | 41 |
| /network/monitor-targets/:id/traffic | interface_name | string | scalar | label | - | - | tenant | - | 29 |
| /network/monitor-targets/:id/traffic | last_updated | *time.Time | scalar | other | - | - | tenant | - | 43 |
| /network/monitor-targets/:id/traffic | missing_intervals | int | scalar | count | composed | - | tenant | assignment only | 241 |
| /network/monitor-targets/:id/traffic | out_label | string | scalar | label | - | - | tenant | - | 42 |
| /network/monitor-targets/:id/traffic | peak_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 228 |
| /network/monitor-targets/:id/traffic | peak_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 229 |
| /network/monitor-targets/:id/traffic | points.in_bps | int64 | series | count | composed | - | tenant | assignment only | 190 |
| /network/monitor-targets/:id/traffic | points.out_bps | int64 | series | count | composed | - | tenant | assignment only | 191 |
| /network/monitor-targets/:id/traffic | points.timestamp | time.Time | series | other | - | - | tenant | - | 18 |
| /network/monitor-targets/:id/traffic | sample_count | int | scalar | count | composed | - | tenant | assignment only | 238 |
| /network/monitor-targets/:id/traffic | scope | string | scalar | label | - | - | tenant | - | 40 |
| /network/monitor-targets/:id/traffic | stale | bool | scalar | flag | - | - | tenant | - | 44 |
| /network/monitor-targets/:id/traffic | stale_after_seconds | int | scalar | count | composed | - | tenant | assignment only | 247 |
| /network/monitor-targets/:id/traffic | title | string | scalar | label | - | - | tenant | - | 28 |
| /network/monitor-targets/:id/traffic | total_in_bytes | int64 | scalar | bytes | composed | - | tenant | assignment only | 232 |
| /network/monitor-targets/:id/traffic | total_out_bytes | int64 | scalar | bytes | composed | - | tenant | assignment only | 233 |
| /platform/tenants | billing_address | string | scalar | label | SELECT | tenant | row | persisted column | 18 |
| /platform/tenants | company_name | string | scalar | label | SELECT | tenant | row | persisted column | 16 |
| /platform/tenants | contact_email | string | scalar | label | SELECT | tenant | row | persisted column | 19 |
| /platform/tenants | contact_phone | string | scalar | label | SELECT | tenant | row | persisted column | 20 |
| /platform/tenants | created_at | time.Time | scalar | other | SELECT | tenant | row | persisted column | 21 |
| /platform/tenants | id | int64 | scalar | id | SELECT | tenant | row | persisted column | 11 |
| /platform/tenants | kind | string | scalar | label | SELECT | tenant | row | persisted column | 14 |
| /platform/tenants | name | string | scalar | label | SELECT | tenant | row | persisted column | 12 |
| /platform/tenants | slug | string | scalar | label | SELECT | tenant | row | persisted column | 13 |
| /platform/tenants | status | string | scalar | label | SELECT | tenant | row | persisted column | 15 |
| /platform/tenants | tax_id | string | scalar | label | SELECT | tenant | row | persisted column | 17 |
| /platform/tenants | updated_at | time.Time | scalar | other | SELECT | tenant | row | persisted column | 22 |
| /radius/traffic/user/:username | available_interfaces | []string | series | other | - | - | tenant | - | 30 |
| /radius/traffic/user/:username | avg_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 230 |
| /radius/traffic/user/:username | avg_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 231 |
| /radius/traffic/user/:username | counter_resets | int | scalar | count | composed | - | tenant | assignment only | 240 |
| /radius/traffic/user/:username | current_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 226 |
| /radius/traffic/user/:username | current_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 227 |
| /radius/traffic/user/:username | data_available | bool | scalar | flag | - | - | tenant | - | 47 |
| /radius/traffic/user/:username | entity_id | string | scalar | label | - | - | tenant | - | 27 |
| /radius/traffic/user/:username | in_label | string | scalar | label | - | - | tenant | - | 41 |
| /radius/traffic/user/:username | interface_name | string | scalar | label | - | - | tenant | - | 29 |
| /radius/traffic/user/:username | last_updated | *time.Time | scalar | other | - | - | tenant | - | 43 |
| /radius/traffic/user/:username | missing_intervals | int | scalar | count | composed | - | tenant | assignment only | 241 |
| /radius/traffic/user/:username | out_label | string | scalar | label | - | - | tenant | - | 42 |
| /radius/traffic/user/:username | peak_in_bps | int64 | scalar | count | composed | - | tenant | assignment only | 228 |
| /radius/traffic/user/:username | peak_out_bps | int64 | scalar | count | composed | - | tenant | assignment only | 229 |
| /radius/traffic/user/:username | points.in_bps | int64 | series | count | composed | - | tenant | assignment only | 190 |
| /radius/traffic/user/:username | points.out_bps | int64 | series | count | composed | - | tenant | assignment only | 191 |
| /radius/traffic/user/:username | points.timestamp | time.Time | series | other | - | - | tenant | - | 18 |
| /radius/traffic/user/:username | sample_count | int | scalar | count | composed | - | tenant | assignment only | 238 |
| /radius/traffic/user/:username | scope | string | scalar | label | - | - | tenant | - | 40 |
| /radius/traffic/user/:username | stale | bool | scalar | flag | - | - | tenant | - | 44 |
| /radius/traffic/user/:username | stale_after_seconds | int | scalar | count | composed | - | tenant | assignment only | 247 |
| /radius/traffic/user/:username | title | string | scalar | label | - | - | tenant | - | 28 |
| /radius/traffic/user/:username | total_in_bytes | int64 | scalar | bytes | composed | - | tenant | assignment only | 232 |
| /radius/traffic/user/:username | total_out_bytes | int64 | scalar | bytes | composed | - | tenant | assignment only | 233 |
| /system/backup | created_at | time.Time | scalar | other | - | - | tenant | - | 75 |
| /system/backup | encrypted | bool | scalar | flag | - | - | tenant | - | 77 |
| /system/backup | format_version | int | scalar | count | composed | - | tenant | assignment only | 147 |
| /system/backup | kdf | string | scalar | label | - | - | tenant | - | 83 |
| /system/backup | kdf_iterations | int | scalar | count | composed | - | tenant | assignment only | 162 |
| /system/backup | payload_sha256 | string | scalar | label | - | - | tenant | - | 81 |
| /system/backup | record_counts | map[string]int | map | other | - | - | map-key | - | 79 |
| /system/backup | salt | string | scalar | label | - | - | tenant | - | 87 |
| /system/backup | schema_version | string | scalar | label | - | - | tenant | - | 73 |

157 served fields, 73 of them aggregates this study scores.

## Reconstructability

The fixture run through each aggregate verb. A cell of one stored row is the case that matters: the published number is then that row's own value, or a statement that the tenant has exactly one.

| surface | field | class | verb | table | cell | smallest cell | tenants with a cell of one | tenants blanked at floor 2 | verdict |
| --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | --- |
| /dashboard/isp-stats | active_subscriptions | membership | COUNT | isp_subscription | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | customers | membership | COUNT | isp_customer | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | invoices_this_month | membership | COUNT | isp_invoice | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | online_users | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | outstanding | value-exposed | SUM | isp_invoice | tenant | 1 | 1 | 1 | publishes one stored value verbatim |
| /dashboard/isp-stats | overdue | membership | COUNT | isp_invoice | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | payments_this_month | membership | COUNT | isp_payment | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/isp-stats | suspended_subscriptions | membership | COUNT | isp_subscription | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | auth_trend.count | unmodeled | composed | - | date | 0 | 0 | 0 | no source table attributed |
| /dashboard/stats | disabled_users | membership | COUNT | radius_user | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | expired_users | membership | COUNT | radius_user | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.adoption_rate | rate-disguise | - | - | tenant | 0 | 0 | 0 | a count in disguise - it divides ipv6_stats.online_with_ipv6 / online_users, both served in the same response |
| /dashboard/stats | ipv6_stats.online_with_delegated_prefix | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.online_with_framed_prefix | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.online_with_ipv6 | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.online_with_ipv6_address | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.users_with_delegated_prefix | membership | COUNT | radius_user | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | ipv6_stats.users_with_static_address | membership | COUNT | radius_user | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | online_users | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | profile_distribution.value | unmodeled | composed | - | profile_name | 0 | 0 | 0 | no source table attributed |
| /dashboard/stats | today_acct_count | membership | COUNT | radius_accounting | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | today_auth_count | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | today_input_gb | value-exposed | SUM | radius_accounting | tenant | 1 | 1 | 1 | publishes one stored value verbatim |
| /dashboard/stats | today_output_gb | value-exposed | SUM | radius_accounting | tenant | 1 | 1 | 1 | publishes one stored value verbatim |
| /dashboard/stats | total_profiles | impersonal | COUNT | radius_profile | tenant | 2 | 0 | 0 | aggregate over a thing the operator owns |
| /dashboard/stats | total_users | membership | COUNT | radius_user | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /dashboard/stats | traffic_24h.download_gb | unmodeled | composed | - | hour | 0 | 0 | 0 | no source table attributed |
| /dashboard/stats | traffic_24h.upload_gb | unmodeled | composed | - | hour | 0 | 0 | 0 | no source table attributed |
| /isp/customers/:id/timeline | pagination.page | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /isp/customers/:id/timeline | pagination.page_size | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /isp/customers/:id/timeline | pagination.total | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/diagnostics/flapping | disconnect_count | membership | composed | radius_accounting | username | 1 | 6 | 6 | publishes that the tenant has exactly one |
| /network/impact | customers | membership | COUNT | isp_customer | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /network/impact | online_sessions | membership | COUNT | radius_online | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /network/impact | open_tickets | membership | COUNT | isp_trouble_ticket | tenant | 1 | 1 | 2 | publishes that the tenant has exactly one |
| /network/impact | subscriptions | membership | COUNT | isp_subscription | tenant | 1 | 1 | 1 | publishes that the tenant has exactly one |
| /network/ipam/pools | ip_version | impersonal | - | isp_ipam_pool | row | 1 | 6 | 6 | one device or catalog row, not a subscriber |
| /network/ipam/pools | tenantid | impersonal | - | isp_ipam_pool | row | 1 | 6 | 6 | one device or catalog row, not a subscriber |
| /network/ipam/pools | total_ips | impersonal | - | isp_ipam_pool | row | 1 | 6 | 6 | one device or catalog row, not a subscriber |
| /network/ipam/pools | used_ips | impersonal | - | isp_ipam_pool | row | 1 | 6 | 6 | one device or catalog row, not a subscriber |
| /network/monitor-targets | heartbeats.latency_milliseconds | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets | heartbeats.packet_loss_percent | unmodeled | - | - | tenant | 0 | 0 | 0 | a ratio no served number is composed from |
| /network/monitor-targets | uptime_percent | unmodeled | - | - | tenant | 0 | 0 | 0 | a ratio with an operand no served key answers to: unserved:reachableCount / unserved:samples |
| /network/monitor-targets/:id/traffic | avg_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | avg_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | counter_resets | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | current_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | current_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | missing_intervals | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | peak_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | peak_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | points.in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | points.out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | sample_count | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | stale_after_seconds | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | total_in_bytes | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /network/monitor-targets/:id/traffic | total_out_bytes | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | avg_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | avg_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | counter_resets | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | current_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | current_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | missing_intervals | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | peak_in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | peak_out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | points.in_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | points.out_bps | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | sample_count | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | stale_after_seconds | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | total_in_bytes | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /radius/traffic/user/:username | total_out_bytes | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /system/backup | format_version | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |
| /system/backup | kdf_iterations | unmodeled | composed | - | tenant | 0 | 0 | 0 | no source table attributed |

73 aggregates scored: 3 publish a stored value verbatim, 24 say the tenant has exactly one, 1 are a count in disguise, 5 cover a thing the operator owns, 0 stay above a cell of one, 40 have no source table attributed.

## Suppression floors priced

How high the cell floor has to be before no fixture value survives, and what that costs the tenants a benchmark is meant to describe.

| floor | fields still reconstructing | fields suppressed somewhere | fields blank for at least one tenant | fields scored |
| --- | ---: | ---: | ---: | ---: |
| 1 | 27 | 0 | 0 | 73 |
| 2 | 0 | 31 | 31 | 73 |
| 3 | 0 | 32 | 31 | 73 |
| 5 | 0 | 32 | 31 | 73 |

A floor that removes an exposure also withholds the tenant that caused it, which is why the smallest tenants in the fixture - the RT/RW Net end - are the ones a benchmark would lose.

## Governance

The permission half of the criterion. Each row is a mechanism a publication would need, counted in the tree after comments are blanked.

| id | mechanism | lines | files | where | reading |
| --- | --- | ---: | ---: | --- | --- |
| P01 | a config switch that turns aggregate publication on | 0 | 1 | internal/app/config_schemas.json | the registry the system config page renders is the only operator-editable switch; a mechanism absent from it cannot be enabled |
| P02 | a cell-size floor or suppression rule in the Go tree | 0 | 0 | - | nothing withholds a statistic that covers one row |
| P03 | an anonymiser applied to a served number | 0 | 0 |  | the shipped privacy code speaks of keys and text, not of numbers |
| P04 | the shipped redactor reaching a statistics handler | 0 | 9 | internal/adminapi statistics files | a statistics handler never calls the redactor, so no served number is filtered by the product's own policy |
| P05 | a per-tenant consent row with a stored risk acknowledgement | 7 | 1 | internal/domain/network_monitor.go:73 | the product already owns the governance shape publication needs: WhatsAppEnabled, RiskAcknowledgedAt and a TenantID scope |
| P06 | TR-F033 naming a statistic, an aggregate, a benchmark or publication | 0 | 1 | docs/feature-checklist.md | the contract wording that governs this data - read up to the research note appended to the same row - never mentions an aggregate leaving a tenant |
| P07 | a fallback to the default tenant when no tenant context is present | 69 | 1 | internal/adminapi/access_control_test.go:114, internal/adminapi/access_control_test.go:119, internal/adminapi/access_control_test.go:130, internal/adminapi/access_control_test.go:133, internal/adminapi/access_control_test.go:355, internal/adminapi/access_control_test.go:53, internal/adminapi/auth_test.go:33, internal/adminapi/context.go:28, internal/adminapi/customer_timeline.go:92, internal/adminapi/diagnostic_target_test.go:76, internal/adminapi/diagnostic_target_test.go:83, internal/adminapi/diagnostic_target_test.go:89, internal/adminapi/diagnostic_target_test.go:94, internal/adminapi/isp.go:1297, internal/adminapi/isp_invoice_math_test.go:25, internal/adminapi/isp_invoice_math_test.go:34, internal/adminapi/isp_operations_test.go:157, internal/adminapi/isp_operations_test.go:190, internal/adminapi/isp_operations_test.go:556, internal/adminapi/isp_payment_idempotency_test.go:22, internal/adminapi/isp_payment_idempotency_test.go:36, internal/adminapi/isp_subscription_state_machine_test.go:242, internal/adminapi/isp_subscription_state_machine_test.go:247, internal/adminapi/isp_subscription_state_machine_test.go:251, internal/adminapi/isp_subscription_state_machine_test.go:264, internal/adminapi/isp_subscription_state_machine_test.go:297, internal/adminapi/isp_subscription_state_machine_test.go:302, internal/adminapi/isp_subscription_state_machine_test.go:364, internal/adminapi/isp_subscription_state_machine_test.go:52, internal/adminapi/isp_subscription_state_machine_test.go:57, internal/adminapi/isp_subscription_state_machine_test.go:62, internal/adminapi/isp_subscription_state_machine_test.go:74, internal/adminapi/isp_subscription_suspension_test.go:25, internal/adminapi/isp_subscription_suspension_test.go:27, internal/adminapi/isp_subscription_suspension_test.go:30, internal/adminapi/isp_subscription_suspension_test.go:35, internal/adminapi/isp_subscription_suspension_test.go:45, internal/adminapi/noc_impact.go:181, internal/adminapi/platform_tenants.go:354, internal/adminapi/platform_tenants_test.go:105, internal/adminapi/platform_tenants_test.go:124, internal/adminapi/platform_tenants_test.go:27, internal/adminapi/platform_tenants_test.go:59, internal/adminapi/public_surface_test.go:75, internal/adminapi/public_surface_test.go:78, internal/adminapi/session_action_queue.go:112, internal/adminapi/session_action_queue.go:43, internal/adminapi/system_backup.go:232, internal/adminapi/system_backup_test.go:116, internal/adminapi/system_backup_test.go:152, internal/adminapi/system_backup_test.go:43, internal/adminapi/tenant_reference_test.go:125, internal/adminapi/tenant_reference_test.go:131, internal/adminapi/tenant_reference_test.go:136, internal/adminapi/tenant_reference_test.go:153, internal/adminapi/tenant_reference_test.go:167, internal/adminapi/tenant_reference_test.go:174, internal/adminapi/tenant_reference_test.go:207, internal/adminapi/tenant_reference_test.go:214, internal/adminapi/tenant_reference_test.go:220, internal/adminapi/tenant_reference_test.go:230, internal/adminapi/tenant_reference_test.go:236, internal/adminapi/tenant_reference_test.go:246, internal/adminapi/tenant_reference_test.go:258, internal/adminapi/tenant_reference_test.go:265, internal/adminapi/tenant_reference_test.go:271, internal/adminapi/tenant_reference_test.go:279, internal/adminapi/tenant_reference_test.go:290, internal/adminapi/test_helpers_test.go:67 | a job with no request context reports the default tenant's numbers rather than refusing, which is how a per-tenant statistic gets mis-attributed |
| P08 | a tenant-scope assertion in the dashboard's own test file | 0 | 1 | internal/adminapi/dashboard_test.go | the largest aggregate surface has no test that a tenant sees only its own numbers |

36 config keys in the registry, 0 of them about statistics, benchmarks, aggregates, anonymisation or sharing.

## Vocabulary searched

The identifiers a suppression or anonymisation rule would be named with, and where the tree actually uses them. The last column counts the matches inside the files that serve statistics - a word used by the RADIUS guard is not a rule about a cell.

| term | lines | files | first | in statistics files | searched for |
| --- | ---: | ---: | --- | ---: | --- |
| kAnonymity | 0 | 0 | - | 0 | a k-anonymity constant would set the smallest publishable cell |
| minCell | 0 | 0 | - | 0 | a minimum cell size would withhold a cell that covers too few rows |
| CellFloor | 0 | 0 | - | 0 | a cell floor is the name a suppression rule for a statistic would carry |
| SuppressCell | 0 | 0 | - | 0 | a named suppression of one cell is the mechanism the criterion needs |
| anonymiz | 0 | 0 | - | 0 | an anonymiser would decide what may leave the tenant |
| suppress | 0 | 0 | - | 0 | the generic word, which the tree uses for a RADIUS guard action and for seeding, not for a statistic |
| Pseudonymize | 12 | 4 | internal/redact/redact.go:82 | 0 | the shipped pseudonymiser, which is a log and support-bundle tool |
| RiskAcknowledgedAt | 16 | 6 | internal/adminapi/isp_operations_test.go:159 | 0 | a stored acknowledgement, the governance shape the notification settings already use |

## Default-tenant fallback sites

Code that reads a tenant id and substitutes the default tenant when no context is present. A background aggregation job has no request context, so this is where one tenant's number can be filed under another.

| file | line |
| --- | ---: |
| internal/adminapi/access_control_test.go | 114 |
| internal/adminapi/access_control_test.go | 119 |
| internal/adminapi/access_control_test.go | 130 |
| internal/adminapi/access_control_test.go | 133 |
| internal/adminapi/access_control_test.go | 355 |
| internal/adminapi/access_control_test.go | 53 |
| internal/adminapi/auth_test.go | 33 |
| internal/adminapi/context.go | 28 |
| internal/adminapi/customer_timeline.go | 92 |
| internal/adminapi/diagnostic_target_test.go | 76 |
| internal/adminapi/diagnostic_target_test.go | 83 |
| internal/adminapi/diagnostic_target_test.go | 89 |
| internal/adminapi/diagnostic_target_test.go | 94 |
| internal/adminapi/isp.go | 1297 |
| internal/adminapi/isp_invoice_math_test.go | 25 |
| internal/adminapi/isp_invoice_math_test.go | 34 |
| internal/adminapi/isp_operations_test.go | 157 |
| internal/adminapi/isp_operations_test.go | 190 |
| internal/adminapi/isp_operations_test.go | 556 |
| internal/adminapi/isp_payment_idempotency_test.go | 22 |
| internal/adminapi/isp_payment_idempotency_test.go | 36 |
| internal/adminapi/isp_subscription_state_machine_test.go | 242 |
| internal/adminapi/isp_subscription_state_machine_test.go | 247 |
| internal/adminapi/isp_subscription_state_machine_test.go | 251 |
| internal/adminapi/isp_subscription_state_machine_test.go | 264 |
| internal/adminapi/isp_subscription_state_machine_test.go | 297 |
| internal/adminapi/isp_subscription_state_machine_test.go | 302 |
| internal/adminapi/isp_subscription_state_machine_test.go | 364 |
| internal/adminapi/isp_subscription_state_machine_test.go | 52 |
| internal/adminapi/isp_subscription_state_machine_test.go | 57 |
| internal/adminapi/isp_subscription_state_machine_test.go | 62 |
| internal/adminapi/isp_subscription_state_machine_test.go | 74 |
| internal/adminapi/isp_subscription_suspension_test.go | 25 |
| internal/adminapi/isp_subscription_suspension_test.go | 27 |
| internal/adminapi/isp_subscription_suspension_test.go | 30 |
| internal/adminapi/isp_subscription_suspension_test.go | 35 |
| internal/adminapi/isp_subscription_suspension_test.go | 45 |
| internal/adminapi/noc_impact.go | 181 |
| internal/adminapi/platform_tenants.go | 354 |
| internal/adminapi/platform_tenants_test.go | 105 |
| internal/adminapi/platform_tenants_test.go | 124 |
| internal/adminapi/platform_tenants_test.go | 27 |
| internal/adminapi/platform_tenants_test.go | 59 |
| internal/adminapi/public_surface_test.go | 75 |
| internal/adminapi/public_surface_test.go | 78 |
| internal/adminapi/session_action_queue.go | 112 |
| internal/adminapi/session_action_queue.go | 43 |
| internal/adminapi/system_backup.go | 232 |
| internal/adminapi/system_backup_test.go | 116 |
| internal/adminapi/system_backup_test.go | 152 |
| internal/adminapi/system_backup_test.go | 43 |
| internal/adminapi/tenant_reference_test.go | 125 |
| internal/adminapi/tenant_reference_test.go | 131 |
| internal/adminapi/tenant_reference_test.go | 136 |
| internal/adminapi/tenant_reference_test.go | 153 |
| internal/adminapi/tenant_reference_test.go | 167 |
| internal/adminapi/tenant_reference_test.go | 174 |
| internal/adminapi/tenant_reference_test.go | 207 |
| internal/adminapi/tenant_reference_test.go | 214 |
| internal/adminapi/tenant_reference_test.go | 220 |
| internal/adminapi/tenant_reference_test.go | 230 |
| internal/adminapi/tenant_reference_test.go | 236 |
| internal/adminapi/tenant_reference_test.go | 246 |
| internal/adminapi/tenant_reference_test.go | 258 |
| internal/adminapi/tenant_reference_test.go | 265 |
| internal/adminapi/tenant_reference_test.go | 271 |
| internal/adminapi/tenant_reference_test.go | 279 |
| internal/adminapi/tenant_reference_test.go | 290 |
| internal/adminapi/test_helpers_test.go | 67 |

## Pinned statements

Each statement is checked against a code line of a named file after comments are blanked. Drift is reported, never dropped.

| id | statement | file | line | status |
| --- | --- | --- | ---: | --- |
| A01 | the dashboard snapshot is a served type, so its numbers are candidates rather than inventions | internal/adminapi/dashboard.go | 19 | confirmed |
| A02 | the tenant's user total is a model query, the one path the shipped tenant callback can reach | internal/adminapi/dashboard.go | 115 | confirmed |
| A03 | the ISP dashboard publishes an outstanding money sum | internal/adminapi/isp.go | 208 | confirmed |
| A04 | the column that sum reads is a stored invoice balance | internal/domain/isp.go | 141 | confirmed |
| A05 | the tenant predicate is injected by a GORM callback registered once | internal/tenancy/scope.go | 32 | confirmed |
| A06 | the injected read scope is one function over the ORM query path | internal/tenancy/scope.go | 64 | confirmed |
| A07 | a tenant id arrives from the signed claim, not from a request parameter | internal/adminapi/auth.go | 253 | confirmed |
| A08 | the request tenant is set in one place, from the authenticated operator | internal/adminapi/auth.go | 226 | confirmed |
| A09 | a handler without a tenant context stands in the default tenant instead of refusing | internal/adminapi/context.go | 28 | confirmed |
| A10 | the NOC impact counts are their own served sub-struct | internal/adminapi/noc_impact.go | 71 | confirmed |
| A11 | the impacted roster counts over an explicitly written tenant comparison | internal/adminapi/noc_impact.go | 385 | confirmed |
| A12 | the traffic series carries absolute byte sums, not a bucketed band | internal/adminapi/telemetry.go | 37 | confirmed |
| A13 | the per-subscriber traffic route takes a username from the path | internal/adminapi/telemetry.go | 56 | confirmed |
| A14 | a flapping diagnosis groups accounting by username | internal/adminapi/isp_operations.go | 1482 | confirmed |
| A15 | the IPAM pool numbers are capacity over a CIDR | internal/domain/isp_operations.go | 64 | confirmed |
| A16 | the platform tenant list serves the persisted tenant row, not a statistics view | internal/adminapi/platform_tenants.go | 91 | confirmed |
| A17 | the tenant row itself carries a tax identifier and a billing address | internal/domain/tenant.go | 17 | confirmed |
| A18 | the platform operator view exposes another tenant's operator real name | internal/adminapi/platform_tenants.go | 60 | confirmed |
| A19 | platform access is a level check, not a permission string | internal/adminapi/platform_tenants.go | 65 | confirmed |
| A20 | the backup manifest holds per-table record counts for the whole installation | internal/adminapi/backup_envelope.go | 79 | confirmed |
| A21 | the shipped pseudonymiser exists and is a log and bundle tool | internal/redact/redact.go | 104 | confirmed |
| A22 | the config registry the system config page renders is a flat list of keyed items | internal/app/config_schemas.json | 4 | confirmed |
| A23 | the product already ships a per-tenant consent row with a stored risk acknowledgement | internal/domain/network_monitor.go | 77 | confirmed |
| A24 | a ratchet test already enforces tenant scope on raw SQL by column name | internal/tenancy/rawsql_ratchet_test.go | 171 | confirmed |
| A25 | the dashboard test file asserts the response, and this run counts its tenant references separately | internal/adminapi/dashboard_test.go | 15 | confirmed |
| A26 | the tenancy policy checker classifies a relation but is not wired as a gate | internal/tenancy/policycheck/policy.go | 36 | confirmed |

26 of 26 confirmed, 0 broken, 0 vacated.

## What this round cannot answer

Each row needs something the repository does not have. The status is the deliverable: an explicit no-go, with the reason.

| id | needs | status | why |
| --- | --- | --- | --- |
| G01 | tenant populations from a running deployment | no-go | the cell sizes here are the fixture's; a one-customer RT/RW Net and a thousand-line ISP behave differently and only stored rows say which is which |
| G02 | a governance decision about publishing an aggregate outside its tenant | no-go | TR-F033 as written says nothing about an aggregate leaving a tenant, so the permission is a scope change in docs/feature-checklist.md, not something a research round can grant |
| G03 | the database's own answer to each parsed statement | no-go | this model applies the verb a handler writes to a synthetic table; a real query plan, a collation or a NULL edge only shows up against PostgreSQL |
| G04 | the suppression threshold itself | no-go | k is a policy number. The run prices k at 1, 2, 3 and 5 and reports what each costs a tenant in coverage; choosing one is a product decision |
| G05 | a collision study of real per-tenant values | no-go | every fixture tenant has a distinct aggregate because every marker is distinct, so the fingerprint result is a property of the fixture, not of real IDR sums |
| G06 | the platform-admin trust boundary | no-go | the platform surface serves a tenant's tax id, billing address and contact phone to a super operator. Whether that is acceptable is a governance question the tree records no answer to |
| G07 | closing the default-tenant fallback | no-go | the sites are counted and cited, but making a context-less job refuse instead of reporting the default tenant edits internal/adminapi, which is product code and outside a RISET round |

## Totals

The run's headline numbers, all of them counts of the tables above.

| measure | value |
| --- | --- |
| surfaces surveyed | 11 |
| surface specs that did not resolve | 0 |
| served fields | 157 |
| aggregate fields scored | 73 |
| publishing a stored value verbatim | 3 |
| saying the tenant has exactly one | 24 |
| rates beside their own denominator | 1 |
| aggregates over a thing the operator owns | 5 |
| aggregates above a cell of one | 0 |
| aggregates with no source table attributed | 40 |
| aggregates over a personal table | 27 |
| fields whose per-tenant fixture value is distinct | 27 |
| fields with a cell of one in the fixture | 31 |
| cross-tenant aggregates served today | 0 |
| surfaces scoped by the tenant callback | 6 |
| surfaces scoped by a written comparison | 1 |
| surfaces with a default-tenant fallback | 2 |
| surfaces meant to cross tenants | 2 |
| config keys in the registry | 36 |
| config keys about statistics or sharing | 0 |
| statistics files calling the shipped redactor | 0 |
| default-tenant fallback sites | 69 |
| tenant assertions in the dashboard test | 0 |
| fields in the shipped opt-in precedent | 7 |
| fixture tenants | 6 |
| fixture stored rows | 1189 |
| exposures left at floor 2 | 0 |
| statements pinned | 26 |
| gaps open | 7 |
| packages this study imports | 15 |


Digest: `7fc52cdec5af2d09bb4aa87aa6584d60d6fe5056f850d0aad2467bd2cbffe461`

## Verification

```
go build -p 1 ./...                                # clean
go vet -p 1 ./internal/opstats/                    # clean
gofmt -l internal/opstats/                         # empty
golangci-lint run ./internal/opstats/              # 0 issues
go test -p 1 -count=1 ./internal/opstats/          # ok (29 gates run + the dump harness skipped, 26 of 26 statements confirmed)
go test -p 1 -count=1 ./internal/...                # ok, whole internal tree
grep -rn opstats --include=*.go internal/ cmd/ | grep -v ^internal/opstats/   # 0 importers
```

The tables above come from
`OPSTATS_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/opstats/`,
so every number in the prose is the same run. The digest
`7fc52cdec5af2d09bb4aa87aa6584d60d6fe5056f850d0aad2467bd2cbffe461` locks 11
surface rows, 157 census fields with the statement and attribution each one is
given, 73 exposure verdicts with their operands and wording, 4 priced floors, 11
scope rows with the line each class cites, the 72 fixture population cells, 8
governance mechanisms, 8 vocabulary terms with their statistics-file counts, 69
fallback sites, 26 pinned statements and 7 open limits. It therefore changes the
moment a handler starts publishing a number this study has not scored, and
`TestTheDigestSurvivesATickingClock` proves it does not change when only the clock
does.

## No-go register

Seven limits stay open and none is closable from the tree. Three need data or a
decision the repository does not hold, two need a running PostgreSQL installation,
one is a product change in `internal/adminapi`, and one is a governance question
nobody has answered. Each is recorded as **no-go**, with the reason, rather than as
done:

| id | needs | kind of asset | verdict |
| --- | --- | --- | --- |
| `G01` | tenant populations from a running deployment | live data | **no-go** - the cell sizes here are the fixture's; a one-customer RT/RW Net and a thousand-line ISP behave differently and only stored rows say which is which |
| `G02` | a governance decision about publishing an aggregate outside its tenant | a scope change in `docs/feature-checklist.md` | **no-go** - TR-F033 as written says nothing about an aggregate leaving a tenant, so the permission is a product-and-legal decision a research round cannot grant |
| `G03` | the database's own answer to each parsed statement | a running PostgreSQL | **no-go** - this model applies the verb a handler writes to a synthetic table; a real query plan, a collation or a NULL edge only shows up against a live server |
| `G04` | the suppression threshold itself | a policy number | **no-go** - `k` is a product decision. The run prices `k` at 1, 2, 3 and 5 and reports what each costs a tenant in coverage; choosing one is not derivable from source |
| `G05` | a collision study of real per-tenant values | live data | **no-go** - every fixture tenant has a distinct aggregate because every marker is distinct, so the fingerprint result is a property of the fixture, not of real rupiah sums |
| `G06` | the platform-admin trust boundary | a governance answer | **no-go** - `/platform/tenants` serves another tenant's `tax_id`, billing address and contact phone to a super-operator (`A16`/`A17`/`A18`); whether that is acceptable is a question the tree records no answer to |
| `G07` | closing the default-tenant fallback | a change to `internal/adminapi` | **no-go** - the 8 production sites are counted and cited, but making a context-less job refuse instead of reporting the default tenant edits product code, which is outside a RISET round |

## Residual limits

- **The fixture is a model, not a measurement.** Every reconstructability and
  floor number is arithmetic about six synthetic tenants and the verb a handler
  writes. A real tenant with three invoices in a month makes `outstanding` a sum
  over three rows rather than one, and a real tenant with one subscriber makes it
  exactly what this table says. The study's claim is about the *shape* - which
  fields can ever be a cell of one - not about how often one occurs. `G01`, `G05`.
- **40 of 73 aggregates are scored as unattributed, and that is a real blind
  spot.** The two `TrafficSeriesDTO` surfaces and the composed dashboard series
  build their numbers in Go from samples, so no SQL verb exists to run the fixture
  through. They are reported as unmodeled rather than assumed safe: a Go-composed
  average over one subscriber's samples is exactly as reconstructable as a `SUM`,
  and this run cannot price it.
- **Attribution reads the handler's file, one hop.** A statement produced by a
  helper in another package, or by a query the study's chained-`Table` reader does
  not parse, lands in the unattributed set. The ratchet that keeps this honest is
  `TestEveryAggregateIsAttributed`, which fails if a field with a verb in the same
  file leaves the census without one.
- **Scope class is the weakest evidence found, not the strongest.** A surface is
  `fallback` if any reachable line substitutes the default tenant, even when a
  callback also covers it, because that is the property that matters for a
  publication: one unprotected path is enough. `TestEveryScopeRowCitesWhatItsClassClaims`
  re-reads the cited line and refuses a class the line does not support.
- **Absence is measured, not argued.** A zero in the governance table means a scan
  ran over `internal`, `pkg`, `cmd` and `docs` and found nothing - with this
  study's own package excluded (`ownPackage = "opstats"`, so a term defined here
  cannot confirm its own absence). The vulnerability of such a claim is the search
  list, which is why all eight identifiers are printed with their line, file and
  statistics-file counts rather than a summary verdict.
- **The rate finding covers the rates this census can resolve.**
  `ipv6_stats.adoption_rate` is traced through its operands and a parameter back to
  the served field; `uptime_percent` divides two locals the response never serves,
  and `packet_loss_percent` is composed outside the file its surface is registered
  in. Neither is credited with being safe.
- **This round wrote into the row it measures.** The TR-F033 contract row is where
  a round's findings get appended, and `P06` counts that row for the words
  "statistic, aggregate, benchmark, publish" - so the note and the measurement now
  share a line. The scanner reads only the wording before the `Research note (`
  marker, which is defensible because a finding reports a permission rather than
  granting one, but it is a convention this study introduced and a reader should
  know it exists. `TestTheContractRowMarksItsResearchNote` keeps it narrow: exactly
  one marker, the marker must be followed by this study's citation, and any of the
  four words appearing in the contract wording before it fails the build - which is
  also the tripwire for the day TR-F033 is amended to actually allow publication.
- **Counts, not a design.** 27 reconstructable aggregates and a floor that blanks
  31 fields for at least one tenant say what a publication gate would have to
  withhold and what it would cost. Whether to publish at all, to whom, and under
  what acknowledgement is `G02`, and no number in this memo answers it.
