# FUT-015 — What the console's own task surfaces already cost a keyboard operator

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `FUT-015` (`TR-F013`, P3, RISET, dependency `ENG-053`) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/contextualws` (read-only, imported by nothing).
Criterion under test: **Task usability dibanding baseline; keyboard fallback
setara.**

## The question

The card asks for an *adaptive contextual workspace built from the operator's
tasks*. Two words in it are measurable today and one is not:

1. **Contextual.** The console already ships surfaces that order destinations by
   task - the user guide's chapters, the getting-started checklist, the quick
   tour, the dashboard cards. Priced against the shipped menu in the unit the
   repository's own keyboard test already uses (forward focus stops inside the
   navigation region), how much of the walk does a task's own order remove?
2. **Keyboard fallback setara.** If those task groups were put in front of the
   menu, would anything a keyboard operator can reach today be lost? Every
   control the task surfaces use is enumerated and classified, so parity is a
   count, not an opinion.
3. **Adaptive.** This half is not closable from source, and the study says so
   rather than inventing a number: the product keeps no per-operator state
   outside browser storage, and usability against a baseline is a human
   measurement. `G01`, `G02`, `G03`, `G05` register exactly what each would need.

The deliverable is therefore a *measurement with a no-go register*, not a
mockup: what a contextual workspace would save, where the shipped task surfaces
already disagree with the menu about who may reach what, and which part of the
criterion this repository cannot answer without an asset.

## What counts as evidence

The rules the `ENG-006`, `ENG-007`, `ECO-006`, `FUT-001`, `FUT-002` and
`FUT-007` studies used, each enforced by a test in this package:

- **A citation is a code line.** Comments are blanked rune-for-rune before a
  symbol is matched, so line numbers survive and prose cannot confirm a claim
  (`TestTheCommentBlankerKeepsLineNumbers`; the blanker keeps `\n` and writes one
  space per rune, and `.yml` uses `#` rather than `//`).
- **A whole identifier.** `containsBounded` refuses an adjacent word character,
  so `filter` is not proven by `filteredMenuItems`' neighbour text alone.
- **Drift is reported, never dropped.** A missing file is `vacated`, a citation
  that no longer lands is `broken`, and `TestEveryStatementHolds` fails the build
  on either.
- **Every number in this memo is spliced from the run.** Nothing in the prose is
  typed by hand; the tables below are the verbatim output of
  `WS_DUMP=1 go test -run TestReportMarkdownDump -v ./internal/contextualws/`,
  and `render_test.go` refuses an empty table, a `%!` verb or a statement with no
  line in that output.
- **The absence scans skip the prototype.** `ownPackage = "contextualws"` is
  excluded, so the study cannot refute itself.

Three rules specific to this study:

- **The cost unit is borrowed, not invented.** A stop is a forward focus move
  inside the navigation region - the same thing `tabTo()` presses in
  `web/e2e/keyboard-navigation.spec.ts` (pinned as a mechanism line). Both
  columns of the cost table count that unit over the *same* deduplicated
  destination sequence; the only difference is which list numbers the stops.
- **The role arms come from the code.** `super|admin|operator` from the operator
  API validation, crossed with the single permission string the server can issue
  (`platform_admin`, from the auth handler) - 6 arms, no hypothetical role.
- **Reachability is separated from cost.** A destination the menu hides is
  reported twice, and distinctly: *no menu entry at all* versus *filtered away by
  that role's gate*, because the checklist's own filter and the menu's filter do
  not read the same thing.

## The prototype

| File | Role |
| --- | --- |
| `contextualws.go` | `Run`, the `Result` the memo splices, the ~42-counter totals table and the sha256 digest. |
| `evidence.go` | Corpus constants, comment blanking per file dialect, bounded matching, line walking, importer and import-set helpers. |
| `menu.go` | The shipped menu parsed into rows, the role arms, per-role forward stop numbers, the declared route set. |
| `tasks.go` | The task surfaces parsed from source (guide, checklist, tour, dashboard cards), step resolution, the edge table, runtime-built links. |
| `cost.go` | Baseline-vs-task stop counts, the surface/menu disagreement table, per-element affordance census, the mechanism pins. |
| `claims.go` | 24 statements with their checks, the 6-item gap register, the evidence gatherer. |
| `render.go` | Emits every table in this memo from one run. |

Nothing in `internal/` or `cmd/` imports it (`TestNothingImportsTheStudy`, and a
grep over the tree returns 0), it opens no database connection, and the only
files it touches are opened for reading (`TestRunningTheStudyWritesNothing`
snapshots `web/src`, `web/e2e`, `internal/adminapi` and `.github/workflows`).

## What the run says

**Task order is nearly free, and the shipped task surfaces already prove it.**
Over 102 task x role cells, walking the destinations through the menu costs
2300 forward focus stops; walking the same destination sequences in each task's
own order costs 745. That is 1555 stops removed, the largest single-task saving
is 47, and no cell costs more than the menu does - `TestGroupingNeverCostsMore`
and statement `C21` hold that invariant. The corpus is not hypothetical: 17
surfaces, 51 destination edges.

**The keyboard fallback is already equivalent, element for element.** All 51
edges ride a keyboard-native control (`Button` bound to the router,
`CardActionArea`, `MenuItemLink`); 0 need a pointer and 0 could not be
classified. The menu draws 7 section labels over 8 rows, and a label is a
`Typography` element - text, not a focus stop (`C06`) - so regrouping costs a
keyboard operator nothing. There is also nothing to displace: 0 type-to-filter
affordances exist in the navigation region, so ordering and grouping are the only
levers the console has today.

**Every task destination is a route the shell already declares.** 51 of 51 edges
resolve. A contextual workspace therefore adds no application, no new surface,
no new permission - it re-orders what exists. Two further controls
(`CustomerTimeline.tsx`, the record page) build their target at runtime and are
counted as an enumeration limit (`G06`), not as a pass.

**The real finding is a disagreement, not a saving.** 28 task x role cells offer
a destination that the same role's menu hides, and they fail in two different
ways:

- `guide.billing`, `guide.optional`, `guide.production` and `guide.start` offer
  `/system/config`, and `setup.checklist` offers it to admin and super, although
  the menu gates that row to a session holding the `platform_admin` permission.
  The checklist's own filter only knows levels (`ADMIN_LEVELS`), not issued
  permissions, so it hides the step from `operator` and shows it to `admin` who
  cannot see the menu row. `SystemConfigPage` runs no client-side role check of
  its own (`C15`: 0 role checks inside the page), so the menu filter is the only
  gate the console shows.
- `/operations` is offered to plain `operator` by the tour, `guide.daily` and
  `guide.sample-data`, although that role's menu filters the row away.
- `/account/settings` is offered by `guide.start` and by the checklist for every
  role yet appears in **no** role's menu at all - the one destination a task
  reaches that the shipped navigation never offers.

So the console is not consistent about who may reach what, and putting task
groups in front of the menu would widen that inconsistency rather than fix it.
Any contextual workspace proposal for `TR-F013` has to answer the gate question
first; that is a product decision (`G04`), not something a source read can make.

**Adaptive is the part the tree cannot support.** The menu's own row count varies
by role (14 rows for `operator`, 23 for `admin+platform_admin`), so ordering cost
is already adaptive - only in the direction the operator cannot control. The
single piece of per-operator state the console keeps is browser storage
(`onboardingStorage.ts`), and the domain has 0 operator UI-preference models.
Learning from an operator's own work needs state that does not exist (`G05`).

**No gate in this repository can see a rendered focus order.** The browser suite
that would - 4 keyboard scenarios over 4 destinations, and a role matrix that
asserts a visibility expectation for all 9 gated destinations and no ungated one
- is never executed by CI: 0 Playwright invocations in `.github/workflows/ci.yml`.
The stop counts here are therefore derived from source order and are honest about
that limit (`G02`), while the role-matrix coverage claim is one the assertion
corpus does support (`C08`).

## The measured tables

Spliced verbatim from one run of `internal/contextualws`.

## The measured numbers

| measure | value |
| --- | --- |
| menu destinations parsed from the shipped menu | 23 |
| section labels the menu draws | 7 |
| destinations the menu gates by role | 9 |
| destinations the menu shows to every role | 14 |
| role arms the product can produce | 6 |
| resources declared in the app shell | 13 |
| in-layout custom routes | 12 |
| routes rendered outside the layout | 2 |
| task surfaces that already order destinations | 17 |
| steps across those surfaces | 51 |
| destination edges across those surfaces | 51 |
| edges that resolve to a declared route | 51 |
| edges whose destination is not a declared route | 0 |
| controls whose destination is built at runtime | 2 |
| user-guide chapters | 8 |
| getting-started steps | 7 |
| getting-started steps restricted to an admin level | 5 |
| getting-started steps a checklist cannot verify | 2 |
| quick-tour steps | 5 |
| dashboard quick links | 7 |
| edges whose control is keyboard-native as written | 51 |
| edges whose control needs a pointer | 0 |
| edges whose control the parser could not classify | 0 |
| task surfaces that filter by role | 1 |
| task surfaces that do not | 16 |
| edges shown to a role whose menu hides the destination | 34 |
| task x role cost rows measured | 102 |
| task rows whose every destination is in that role's menu | 74 |
| baseline focus stops over all cost rows | 2300 |
| task-grouped focus stops over the same rows | 745 |
| focus stops saved by grouping | 1555 |
| largest saving in one task | 47 |
| menu gates the browser role matrix asserts | 9 |
| menu gates no test asserts | 0 |
| type-to-filter affordances in the navigation region | 0 |
| keyboard scenarios in the shipped keyboard test | 4 |
| destinations those scenarios cover | 4 |
| CI jobs that run Playwright | 0 |
| operator UI-preference models in the domain | 0 |
| statements pinned to a code line | 24 |
| statements whose line moved | 0 |
| statements whose file is gone | 0 |
| limits this study leaves open | 6 |

## The shipped menu, numbered the way a keyboard numbers it

| role arm | menu destination | label key | gate | forward tab stops | reachable |
| --- | --- | --- | --- | --- | --- |
| admin | / | menu.dashboard | none | 1 | yes |
| admin | /isp/customers | menu.customers | none | 2 | yes |
| admin | /isp/packages | menu.packages | none | 3 | yes |
| admin | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| admin | /isp/vouchers | menu.vouchers | none | 5 | yes |
| admin | /isp/tickets | menu.tickets | none | 6 | yes |
| admin | /isp/invoices | menu.invoices | none | 7 | yes |
| admin | /isp/payments | menu.payments | none | 8 | yes |
| admin | /radius/users | menu.radius_users | none | 9 | yes |
| admin | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| admin | /radius/online | menu.online_sessions | none | 11 | yes |
| admin | /radius/accounting | menu.accounting | none | 12 | yes |
| admin | /network/nodes | menu.network_nodes | none | 13 | yes |
| admin | /network/nas | menu.nas_devices | none | 14 | yes |
| admin | /network/ipam | menu.ipam | super|admin | 15 | yes |
| admin | /network/odp | menu.odp_management | super|admin | 16 | yes |
| admin | /operations | menu.operations | super|admin | 17 | yes |
| admin | /system/website | menu.website_editor | platform_admin | (not reachable) | no |
| admin | /system/branding | menu.branding | platform_admin | (not reachable) | no |
| admin | /system/config | menu.system_config | platform_admin | (not reachable) | no |
| admin | /system/operators | menu.operators | super|admin | 18 | yes |
| admin | /system/certificate | menu.certificates | platform_admin | (not reachable) | no |
| admin | /platform/tenants | menu.tenants | platform_admin | (not reachable) | no |
| admin+platform_admin | / | menu.dashboard | none | 1 | yes |
| admin+platform_admin | /isp/customers | menu.customers | none | 2 | yes |
| admin+platform_admin | /isp/packages | menu.packages | none | 3 | yes |
| admin+platform_admin | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| admin+platform_admin | /isp/vouchers | menu.vouchers | none | 5 | yes |
| admin+platform_admin | /isp/tickets | menu.tickets | none | 6 | yes |
| admin+platform_admin | /isp/invoices | menu.invoices | none | 7 | yes |
| admin+platform_admin | /isp/payments | menu.payments | none | 8 | yes |
| admin+platform_admin | /radius/users | menu.radius_users | none | 9 | yes |
| admin+platform_admin | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| admin+platform_admin | /radius/online | menu.online_sessions | none | 11 | yes |
| admin+platform_admin | /radius/accounting | menu.accounting | none | 12 | yes |
| admin+platform_admin | /network/nodes | menu.network_nodes | none | 13 | yes |
| admin+platform_admin | /network/nas | menu.nas_devices | none | 14 | yes |
| admin+platform_admin | /network/ipam | menu.ipam | super|admin | 15 | yes |
| admin+platform_admin | /network/odp | menu.odp_management | super|admin | 16 | yes |
| admin+platform_admin | /operations | menu.operations | super|admin | 17 | yes |
| admin+platform_admin | /system/website | menu.website_editor | platform_admin | 18 | yes |
| admin+platform_admin | /system/branding | menu.branding | platform_admin | 19 | yes |
| admin+platform_admin | /system/config | menu.system_config | platform_admin | 20 | yes |
| admin+platform_admin | /system/operators | menu.operators | super|admin | 21 | yes |
| admin+platform_admin | /system/certificate | menu.certificates | platform_admin | 22 | yes |
| admin+platform_admin | /platform/tenants | menu.tenants | platform_admin | 23 | yes |
| operator | / | menu.dashboard | none | 1 | yes |
| operator | /isp/customers | menu.customers | none | 2 | yes |
| operator | /isp/packages | menu.packages | none | 3 | yes |
| operator | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| operator | /isp/vouchers | menu.vouchers | none | 5 | yes |
| operator | /isp/tickets | menu.tickets | none | 6 | yes |
| operator | /isp/invoices | menu.invoices | none | 7 | yes |
| operator | /isp/payments | menu.payments | none | 8 | yes |
| operator | /radius/users | menu.radius_users | none | 9 | yes |
| operator | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| operator | /radius/online | menu.online_sessions | none | 11 | yes |
| operator | /radius/accounting | menu.accounting | none | 12 | yes |
| operator | /network/nodes | menu.network_nodes | none | 13 | yes |
| operator | /network/nas | menu.nas_devices | none | 14 | yes |
| operator | /network/ipam | menu.ipam | super|admin | (not reachable) | no |
| operator | /network/odp | menu.odp_management | super|admin | (not reachable) | no |
| operator | /operations | menu.operations | super|admin | (not reachable) | no |
| operator | /system/website | menu.website_editor | platform_admin | (not reachable) | no |
| operator | /system/branding | menu.branding | platform_admin | (not reachable) | no |
| operator | /system/config | menu.system_config | platform_admin | (not reachable) | no |
| operator | /system/operators | menu.operators | super|admin | (not reachable) | no |
| operator | /system/certificate | menu.certificates | platform_admin | (not reachable) | no |
| operator | /platform/tenants | menu.tenants | platform_admin | (not reachable) | no |
| operator+platform_admin | / | menu.dashboard | none | 1 | yes |
| operator+platform_admin | /isp/customers | menu.customers | none | 2 | yes |
| operator+platform_admin | /isp/packages | menu.packages | none | 3 | yes |
| operator+platform_admin | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| operator+platform_admin | /isp/vouchers | menu.vouchers | none | 5 | yes |
| operator+platform_admin | /isp/tickets | menu.tickets | none | 6 | yes |
| operator+platform_admin | /isp/invoices | menu.invoices | none | 7 | yes |
| operator+platform_admin | /isp/payments | menu.payments | none | 8 | yes |
| operator+platform_admin | /radius/users | menu.radius_users | none | 9 | yes |
| operator+platform_admin | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| operator+platform_admin | /radius/online | menu.online_sessions | none | 11 | yes |
| operator+platform_admin | /radius/accounting | menu.accounting | none | 12 | yes |
| operator+platform_admin | /network/nodes | menu.network_nodes | none | 13 | yes |
| operator+platform_admin | /network/nas | menu.nas_devices | none | 14 | yes |
| operator+platform_admin | /network/ipam | menu.ipam | super|admin | (not reachable) | no |
| operator+platform_admin | /network/odp | menu.odp_management | super|admin | (not reachable) | no |
| operator+platform_admin | /operations | menu.operations | super|admin | (not reachable) | no |
| operator+platform_admin | /system/website | menu.website_editor | platform_admin | 15 | yes |
| operator+platform_admin | /system/branding | menu.branding | platform_admin | 16 | yes |
| operator+platform_admin | /system/config | menu.system_config | platform_admin | 17 | yes |
| operator+platform_admin | /system/operators | menu.operators | super|admin | (not reachable) | no |
| operator+platform_admin | /system/certificate | menu.certificates | platform_admin | 18 | yes |
| operator+platform_admin | /platform/tenants | menu.tenants | platform_admin | 19 | yes |
| super | / | menu.dashboard | none | 1 | yes |
| super | /isp/customers | menu.customers | none | 2 | yes |
| super | /isp/packages | menu.packages | none | 3 | yes |
| super | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| super | /isp/vouchers | menu.vouchers | none | 5 | yes |
| super | /isp/tickets | menu.tickets | none | 6 | yes |
| super | /isp/invoices | menu.invoices | none | 7 | yes |
| super | /isp/payments | menu.payments | none | 8 | yes |
| super | /radius/users | menu.radius_users | none | 9 | yes |
| super | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| super | /radius/online | menu.online_sessions | none | 11 | yes |
| super | /radius/accounting | menu.accounting | none | 12 | yes |
| super | /network/nodes | menu.network_nodes | none | 13 | yes |
| super | /network/nas | menu.nas_devices | none | 14 | yes |
| super | /network/ipam | menu.ipam | super|admin | 15 | yes |
| super | /network/odp | menu.odp_management | super|admin | 16 | yes |
| super | /operations | menu.operations | super|admin | 17 | yes |
| super | /system/website | menu.website_editor | platform_admin | (not reachable) | no |
| super | /system/branding | menu.branding | platform_admin | (not reachable) | no |
| super | /system/config | menu.system_config | platform_admin | (not reachable) | no |
| super | /system/operators | menu.operators | super|admin | 18 | yes |
| super | /system/certificate | menu.certificates | platform_admin | (not reachable) | no |
| super | /platform/tenants | menu.tenants | platform_admin | (not reachable) | no |
| super+platform_admin | / | menu.dashboard | none | 1 | yes |
| super+platform_admin | /isp/customers | menu.customers | none | 2 | yes |
| super+platform_admin | /isp/packages | menu.packages | none | 3 | yes |
| super+platform_admin | /isp/subscriptions | menu.subscriptions | none | 4 | yes |
| super+platform_admin | /isp/vouchers | menu.vouchers | none | 5 | yes |
| super+platform_admin | /isp/tickets | menu.tickets | none | 6 | yes |
| super+platform_admin | /isp/invoices | menu.invoices | none | 7 | yes |
| super+platform_admin | /isp/payments | menu.payments | none | 8 | yes |
| super+platform_admin | /radius/users | menu.radius_users | none | 9 | yes |
| super+platform_admin | /radius/profiles | menu.radius_profiles | none | 10 | yes |
| super+platform_admin | /radius/online | menu.online_sessions | none | 11 | yes |
| super+platform_admin | /radius/accounting | menu.accounting | none | 12 | yes |
| super+platform_admin | /network/nodes | menu.network_nodes | none | 13 | yes |
| super+platform_admin | /network/nas | menu.nas_devices | none | 14 | yes |
| super+platform_admin | /network/ipam | menu.ipam | super|admin | 15 | yes |
| super+platform_admin | /network/odp | menu.odp_management | super|admin | 16 | yes |
| super+platform_admin | /operations | menu.operations | super|admin | 17 | yes |
| super+platform_admin | /system/website | menu.website_editor | platform_admin | 18 | yes |
| super+platform_admin | /system/branding | menu.branding | platform_admin | 19 | yes |
| super+platform_admin | /system/config | menu.system_config | platform_admin | 20 | yes |
| super+platform_admin | /system/operators | menu.operators | super|admin | 21 | yes |
| super+platform_admin | /system/certificate | menu.certificates | platform_admin | 22 | yes |
| super+platform_admin | /platform/tenants | menu.tenants | platform_admin | 23 | yes |

## The task surfaces the console already ships

| task | surface | what it is called | source line | filters by role | carrier | keyboard |
| --- | --- | --- | --- | --- | --- | --- |
| dashboard.dashboard.active_subscriptions | dashboard card | dashboard.active_subscriptions | web/src/pages/Dashboard.tsx:456 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.customers | dashboard card | dashboard.customers | web/src/pages/Dashboard.tsx:455 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.invoices_this_month | dashboard card | dashboard.invoices_this_month | web/src/pages/Dashboard.tsx:458 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.outstanding | dashboard card | dashboard.outstanding | web/src/pages/Dashboard.tsx:460 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.overdue | dashboard card | dashboard.overdue | web/src/pages/Dashboard.tsx:461 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.payments_this_month | dashboard card | dashboard.payments_this_month | web/src/pages/Dashboard.tsx:459 | no | CardActionArea | native (line 465) |
| dashboard.dashboard.suspended_subscriptions | dashboard card | dashboard.suspended_subscriptions | web/src/pages/Dashboard.tsx:457 | no | CardActionArea | native (line 465) |
| guide.billing | user guide | Billing operations | web/src/pages/UserGuidePage.tsx:17 | no | Button | native (line 53) |
| guide.daily | user guide | Daily operations | web/src/pages/UserGuidePage.tsx:19 | no | Button | native (line 53) |
| guide.optional | user guide | Optional integrations | web/src/pages/UserGuidePage.tsx:20 | no | Button | native (line 53) |
| guide.production | user guide | Production readiness and troubleshooting | web/src/pages/UserGuidePage.tsx:21 | no | Button | native (line 53) |
| guide.radius | user guide | RADIUS profiles and NAS | web/src/pages/UserGuidePage.tsx:15 | no | Button | native (line 53) |
| guide.sample-data | user guide | Sample data included on first launch | web/src/pages/UserGuidePage.tsx:18 | no | Button | native (line 53) |
| guide.services | user guide | Customers and service lifecycle | web/src/pages/UserGuidePage.tsx:16 | no | Button | native (line 53) |
| guide.start | user guide | Start here | web/src/pages/UserGuidePage.tsx:14 | no | Button | native (line 53) |
| setup.checklist | getting started | Setup checklist | web/src/components/onboarding/GettingStartedCard.tsx:20 | yes | Button | native (line 56) |
| tour.quick | quick tour | Quick tour | web/src/components/onboarding/QuickTourDialog.tsx:10 | no | Button | native (line 40) |

## Navigation cost: the shipped order against the task's own order

| task | role arm | destinations | stops via the menu | stops via the task | stops saved | farthest destination | stops to it | not in that menu |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| dashboard.dashboard.active_subscriptions | admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.active_subscriptions | admin+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.active_subscriptions | operator | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.active_subscriptions | operator+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.active_subscriptions | super | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.active_subscriptions | super+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.customers | admin | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.customers | admin+platform_admin | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.customers | operator | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.customers | operator+platform_admin | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.customers | super | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.customers | super+platform_admin | 1 | 2 | 1 | 1 | /isp/customers | 2 | (none) |
| dashboard.dashboard.invoices_this_month | admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.invoices_this_month | admin+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.invoices_this_month | operator | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.invoices_this_month | operator+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.invoices_this_month | super | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.invoices_this_month | super+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | admin+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | operator | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | operator+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | super | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.outstanding | super+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | admin+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | operator | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | operator+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | super | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.overdue | super+platform_admin | 1 | 7 | 1 | 6 | /isp/invoices | 7 | (none) |
| dashboard.dashboard.payments_this_month | admin | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.payments_this_month | admin+platform_admin | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.payments_this_month | operator | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.payments_this_month | operator+platform_admin | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.payments_this_month | super | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.payments_this_month | super+platform_admin | 1 | 8 | 1 | 7 | /isp/payments | 8 | (none) |
| dashboard.dashboard.suspended_subscriptions | admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.suspended_subscriptions | admin+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.suspended_subscriptions | operator | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.suspended_subscriptions | operator+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.suspended_subscriptions | super | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| dashboard.dashboard.suspended_subscriptions | super+platform_admin | 1 | 4 | 1 | 3 | /isp/subscriptions | 4 | (none) |
| guide.billing | admin | 3 | 15 | 3 | 12 | /isp/payments | 8 | /system/config (filtered away for admin) |
| guide.billing | admin+platform_admin | 3 | 35 | 6 | 29 | /system/config | 20 | (none) |
| guide.billing | operator | 3 | 15 | 3 | 12 | /isp/payments | 8 | /system/config (filtered away for operator) |
| guide.billing | operator+platform_admin | 3 | 32 | 6 | 26 | /system/config | 17 | (none) |
| guide.billing | super | 3 | 15 | 3 | 12 | /isp/payments | 8 | /system/config (filtered away for super) |
| guide.billing | super+platform_admin | 3 | 35 | 6 | 29 | /system/config | 20 | (none) |
| guide.daily | admin | 4 | 41 | 10 | 31 | /operations | 17 | (none) |
| guide.daily | admin+platform_admin | 4 | 41 | 10 | 31 | /operations | 17 | (none) |
| guide.daily | operator | 4 | 24 | 6 | 18 | /radius/accounting | 12 | /operations (filtered away for operator) |
| guide.daily | operator+platform_admin | 4 | 24 | 6 | 18 | /radius/accounting | 12 | /operations (filtered away for operator+platform_admin) |
| guide.daily | super | 4 | 41 | 10 | 31 | /operations | 17 | (none) |
| guide.daily | super+platform_admin | 4 | 41 | 10 | 31 | /operations | 17 | (none) |
| guide.optional | admin | 2 | 17 | 1 | 16 | /operations | 17 | /system/config (filtered away for admin) |
| guide.optional | admin+platform_admin | 2 | 37 | 3 | 34 | /system/config | 20 | (none) |
| guide.optional | operator | 2 | 0 | 0 | 0 | (none) | 0 | /operations (filtered away for operator), /system/config (filtered away for operator) |
| guide.optional | operator+platform_admin | 2 | 17 | 2 | 15 | /system/config | 17 | /operations (filtered away for operator+platform_admin) |
| guide.optional | super | 2 | 17 | 1 | 16 | /operations | 17 | /system/config (filtered away for super) |
| guide.optional | super+platform_admin | 2 | 37 | 3 | 34 | /system/config | 20 | (none) |
| guide.production | admin | 4 | 37 | 6 | 31 | /network/nas | 14 | /system/config (filtered away for admin) |
| guide.production | admin+platform_admin | 4 | 57 | 10 | 47 | /system/config | 20 | (none) |
| guide.production | operator | 4 | 37 | 6 | 31 | /network/nas | 14 | /system/config (filtered away for operator) |
| guide.production | operator+platform_admin | 4 | 54 | 10 | 44 | /system/config | 17 | (none) |
| guide.production | super | 4 | 37 | 6 | 31 | /network/nas | 14 | /system/config (filtered away for super) |
| guide.production | super+platform_admin | 4 | 57 | 10 | 47 | /system/config | 20 | (none) |
| guide.radius | admin | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.radius | admin+platform_admin | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.radius | operator | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.radius | operator+platform_admin | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.radius | super | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.radius | super+platform_admin | 4 | 47 | 10 | 37 | /network/nas | 14 | (none) |
| guide.sample-data | admin | 9 | 78 | 45 | 33 | /operations | 17 | (none) |
| guide.sample-data | admin+platform_admin | 9 | 78 | 45 | 33 | /operations | 17 | (none) |
| guide.sample-data | operator | 9 | 61 | 36 | 25 | /network/nas | 14 | /operations (filtered away for operator) |
| guide.sample-data | operator+platform_admin | 9 | 61 | 36 | 25 | /network/nas | 14 | /operations (filtered away for operator+platform_admin) |
| guide.sample-data | super | 9 | 78 | 45 | 33 | /operations | 17 | (none) |
| guide.sample-data | super+platform_admin | 9 | 78 | 45 | 33 | /operations | 17 | (none) |
| guide.services | admin | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.services | admin+platform_admin | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.services | operator | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.services | operator+platform_admin | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.services | super | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.services | super+platform_admin | 4 | 18 | 10 | 8 | /radius/users | 9 | (none) |
| guide.start | admin | 2 | 0 | 0 | 0 | (none) | 0 | /account/settings (no menu entry at all), /system/config (filtered away for admin) |
| guide.start | admin+platform_admin | 2 | 20 | 2 | 18 | /system/config | 20 | /account/settings (no menu entry at all) |
| guide.start | operator | 2 | 0 | 0 | 0 | (none) | 0 | /account/settings (no menu entry at all), /system/config (filtered away for operator) |
| guide.start | operator+platform_admin | 2 | 17 | 2 | 15 | /system/config | 17 | /account/settings (no menu entry at all) |
| guide.start | super | 2 | 0 | 0 | 0 | (none) | 0 | /account/settings (no menu entry at all), /system/config (filtered away for super) |
| guide.start | super+platform_admin | 2 | 20 | 2 | 18 | /system/config | 20 | /account/settings (no menu entry at all) |
| setup.checklist | admin | 7 | 40 | 25 | 15 | /network/nas | 14 | /account/settings (no menu entry at all), /system/config (filtered away for admin) |
| setup.checklist | admin+platform_admin | 7 | 60 | 27 | 33 | /system/config | 20 | /account/settings (no menu entry at all) |
| setup.checklist | operator | 2 | 11 | 2 | 9 | /radius/online | 11 | /account/settings (no menu entry at all) |
| setup.checklist | operator+platform_admin | 2 | 11 | 2 | 9 | /radius/online | 11 | /account/settings (no menu entry at all) |
| setup.checklist | super | 7 | 40 | 25 | 15 | /network/nas | 14 | /account/settings (no menu entry at all), /system/config (filtered away for super) |
| setup.checklist | super+platform_admin | 7 | 60 | 27 | 33 | /system/config | 20 | /account/settings (no menu entry at all) |
| tour.quick | admin | 5 | 39 | 15 | 24 | /operations | 17 | (none) |
| tour.quick | admin+platform_admin | 5 | 39 | 15 | 24 | /operations | 17 | (none) |
| tour.quick | operator | 5 | 22 | 10 | 12 | /radius/online | 11 | /operations (filtered away for operator) |
| tour.quick | operator+platform_admin | 5 | 22 | 10 | 12 | /radius/online | 11 | /operations (filtered away for operator+platform_admin) |
| tour.quick | super | 5 | 39 | 15 | 24 | /operations | 17 | (none) |
| tour.quick | super+platform_admin | 5 | 39 | 15 | 24 | /operations | 17 | (none) |

## Where a task surface and the same role's menu disagree

| task | role arm | how the surface decides | offered | hidden by the menu | which |
| --- | --- | --- | --- | --- | --- |
| guide.billing | admin | shows every step to every role | 3 | 1 | /system/config (hidden by that role's gate) |
| guide.billing | operator | shows every step to every role | 3 | 1 | /system/config (hidden by that role's gate) |
| guide.billing | super | shows every step to every role | 3 | 1 | /system/config (hidden by that role's gate) |
| guide.daily | operator | shows every step to every role | 4 | 1 | /operations (hidden by that role's gate) |
| guide.daily | operator+platform_admin | shows every step to every role | 4 | 1 | /operations (hidden by that role's gate) |
| guide.optional | admin | shows every step to every role | 2 | 1 | /system/config (hidden by that role's gate) |
| guide.optional | operator | shows every step to every role | 2 | 2 | /operations (hidden by that role's gate), /system/config (hidden by that role's gate) |
| guide.optional | operator+platform_admin | shows every step to every role | 2 | 1 | /operations (hidden by that role's gate) |
| guide.optional | super | shows every step to every role | 2 | 1 | /system/config (hidden by that role's gate) |
| guide.production | admin | shows every step to every role | 4 | 1 | /system/config (hidden by that role's gate) |
| guide.production | operator | shows every step to every role | 4 | 1 | /system/config (hidden by that role's gate) |
| guide.production | super | shows every step to every role | 4 | 1 | /system/config (hidden by that role's gate) |
| guide.sample-data | operator | shows every step to every role | 9 | 1 | /operations (hidden by that role's gate) |
| guide.sample-data | operator+platform_admin | shows every step to every role | 9 | 1 | /operations (hidden by that role's gate) |
| guide.start | admin | shows every step to every role | 2 | 2 | /account/settings (no menu entry at all), /system/config (hidden by that role's gate) |
| guide.start | admin+platform_admin | shows every step to every role | 2 | 1 | /account/settings (no menu entry at all) |
| guide.start | operator | shows every step to every role | 2 | 2 | /account/settings (no menu entry at all), /system/config (hidden by that role's gate) |
| guide.start | operator+platform_admin | shows every step to every role | 2 | 1 | /account/settings (no menu entry at all) |
| guide.start | super | shows every step to every role | 2 | 2 | /account/settings (no menu entry at all), /system/config (hidden by that role's gate) |
| guide.start | super+platform_admin | shows every step to every role | 2 | 1 | /account/settings (no menu entry at all) |
| setup.checklist | admin | filters its own steps | 7 | 2 | /account/settings (no menu entry at all), /system/config (hidden by that role's gate) |
| setup.checklist | admin+platform_admin | filters its own steps | 7 | 1 | /account/settings (no menu entry at all) |
| setup.checklist | operator | filters its own steps | 2 | 1 | /account/settings (no menu entry at all) |
| setup.checklist | operator+platform_admin | filters its own steps | 2 | 1 | /account/settings (no menu entry at all) |
| setup.checklist | super | filters its own steps | 7 | 2 | /account/settings (no menu entry at all), /system/config (hidden by that role's gate) |
| setup.checklist | super+platform_admin | filters its own steps | 7 | 1 | /account/settings (no menu entry at all) |
| tour.quick | operator | shows every step to every role | 5 | 1 | /operations (hidden by that role's gate) |
| tour.quick | operator+platform_admin | shows every step to every role | 5 | 1 | /operations (hidden by that role's gate) |

## Every destination a task offers, with the control that carries it

| task | surface | destination | what it is called | carrier | keyboard | declared route | declared by | source line |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| dashboard.dashboard.active_subscriptions | dashboard card | /isp/subscriptions | dashboard.active_subscriptions | CardActionArea | native | yes | resource isp/subscriptions | 456 |
| dashboard.dashboard.customers | dashboard card | /isp/customers | dashboard.customers | CardActionArea | native | yes | resource isp/customers | 455 |
| dashboard.dashboard.invoices_this_month | dashboard card | /isp/invoices | dashboard.invoices_this_month | CardActionArea | native | yes | resource isp/invoices | 458 |
| dashboard.dashboard.outstanding | dashboard card | /isp/invoices | dashboard.outstanding | CardActionArea | native | yes | resource isp/invoices | 460 |
| dashboard.dashboard.overdue | dashboard card | /isp/invoices | dashboard.overdue | CardActionArea | native | yes | resource isp/invoices | 461 |
| dashboard.dashboard.payments_this_month | dashboard card | /isp/payments | dashboard.payments_this_month | CardActionArea | native | yes | resource isp/payments | 459 |
| dashboard.dashboard.suspended_subscriptions | dashboard card | /isp/subscriptions | dashboard.suspended_subscriptions | CardActionArea | native | yes | resource isp/subscriptions | 457 |
| guide.billing | user guide | /isp/invoices | Invoices | Button | native | yes | resource isp/invoices | 17 |
| guide.billing | user guide | /isp/payments | Payments | Button | native | yes | resource isp/payments | 17 |
| guide.billing | user guide | /system/config | System configuration | Button | native | yes | custom route | 17 |
| guide.daily | user guide | / | Dashboard | Button | native | yes | dashboard | 19 |
| guide.daily | user guide | /radius/online | Online sessions | Button | native | yes | resource radius/online | 19 |
| guide.daily | user guide | /radius/accounting | Accounting | Button | native | yes | resource radius/accounting | 19 |
| guide.daily | user guide | /operations | Network & Alerts | Button | native | yes | custom route | 19 |
| guide.optional | user guide | /operations | Network & Alerts | Button | native | yes | custom route | 20 |
| guide.optional | user guide | /system/config | System configuration | Button | native | yes | custom route | 20 |
| guide.production | user guide | /network/nas | NAS devices | Button | native | yes | resource network/nas | 21 |
| guide.production | user guide | /radius/online | Online sessions | Button | native | yes | resource radius/online | 21 |
| guide.production | user guide | /radius/accounting | Accounting | Button | native | yes | resource radius/accounting | 21 |
| guide.production | user guide | /system/config | System configuration | Button | native | yes | custom route | 21 |
| guide.radius | user guide | /radius/profiles | RADIUS profiles | Button | native | yes | resource radius/profiles | 15 |
| guide.radius | user guide | /network/nas | NAS devices | Button | native | yes | resource network/nas | 15 |
| guide.radius | user guide | /radius/online | Online sessions | Button | native | yes | resource radius/online | 15 |
| guide.radius | user guide | /radius/accounting | Accounting | Button | native | yes | resource radius/accounting | 15 |
| guide.sample-data | user guide | /network/nodes | Nodes | Button | native | yes | resource network/nodes | 18 |
| guide.sample-data | user guide | /network/nas | NAS devices | Button | native | yes | resource network/nas | 18 |
| guide.sample-data | user guide | /radius/profiles | RADIUS profiles | Button | native | yes | resource radius/profiles | 18 |
| guide.sample-data | user guide | /isp/customers | Customers | Button | native | yes | resource isp/customers | 18 |
| guide.sample-data | user guide | /isp/packages | Packages | Button | native | yes | resource isp/packages | 18 |
| guide.sample-data | user guide | /isp/subscriptions | Subscriptions | Button | native | yes | resource isp/subscriptions | 18 |
| guide.sample-data | user guide | /isp/invoices | Invoices | Button | native | yes | resource isp/invoices | 18 |
| guide.sample-data | user guide | /isp/payments | Payments | Button | native | yes | resource isp/payments | 18 |
| guide.sample-data | user guide | /operations | Network & Alerts | Button | native | yes | custom route | 18 |
| guide.services | user guide | /isp/packages | Packages | Button | native | yes | resource isp/packages | 16 |
| guide.services | user guide | /isp/customers | Customers | Button | native | yes | resource isp/customers | 16 |
| guide.services | user guide | /isp/subscriptions | Subscriptions | Button | native | yes | resource isp/subscriptions | 16 |
| guide.services | user guide | /radius/users | RADIUS users | Button | native | yes | resource radius/users | 16 |
| guide.start | user guide | /account/settings | Account settings | Button | native | yes | custom route | 14 |
| guide.start | user guide | /system/config | System configuration | Button | native | yes | custom route | 14 |
| setup.checklist | getting started | /account/settings |  | Button | native | yes | custom route | 20 |
| setup.checklist | getting started | /system/config |  | Button | native | yes | custom route | 21 |
| setup.checklist | getting started | /radius/profiles |  | Button | native | yes | resource radius/profiles | 22 |
| setup.checklist | getting started | /network/nas |  | Button | native | yes | resource network/nas | 23 |
| setup.checklist | getting started | /isp/packages |  | Button | native | yes | resource isp/packages | 24 |
| setup.checklist | getting started | /isp/customers |  | Button | native | yes | resource isp/customers | 25 |
| setup.checklist | getting started | /radius/online |  | Button | native | yes | resource radius/online | 26 |
| tour.quick | quick tour | / |  | Button | native | yes | dashboard | 10 |
| tour.quick | quick tour | /isp/packages |  | Button | native | yes | resource isp/packages | 11 |
| tour.quick | quick tour | /radius/online |  | Button | native | yes | resource radius/online | 12 |
| tour.quick | quick tour | /isp/invoices |  | Button | native | yes | resource isp/invoices | 13 |
| tour.quick | quick tour | /operations |  | Button | native | yes | custom route | 14 |

## The navigation controls in the console, by element

| console file | element | kind | count |
| --- | --- | --- | --- |
| web/src/components/CustomAppBar.tsx | IconButton | native (handler) | 2 |
| web/src/components/CustomerTimeline.tsx | Button | native (handler) | 3 |
| web/src/components/CustomerTimeline.tsx | Button | native router | 1 |
| web/src/components/Enterprise.tsx | Button | native (handler) | 1 |
| web/src/components/MrtgTrafficGraph.tsx | Button | native (handler) | 1 |
| web/src/components/NocImpactPanel.tsx | Button | native (handler) | 2 |
| web/src/components/onboarding/GettingStartedCard.tsx | Button | native (handler) | 1 |
| web/src/components/onboarding/GettingStartedCard.tsx | Button | native router | 1 |
| web/src/components/onboarding/QuickTourDialog.tsx | Button | native (handler) | 3 |
| web/src/components/onboarding/QuickTourDialog.tsx | Button | native router | 1 |
| web/src/pages/BrandingPage.tsx | Button | native (handler) | 4 |
| web/src/pages/Dashboard.tsx | CardActionArea | native router | 1 |
| web/src/pages/DebugAccountSettings.tsx | Button | native (handler) | 1 |
| web/src/pages/HotspotVouchersPage.tsx | Button | native (handler) | 3 |
| web/src/pages/IPAMPage.tsx | Button | native (handler) | 1 |
| web/src/pages/ODPPage.tsx | Button | native (handler) | 1 |
| web/src/pages/OperationsPage.tsx | Button | native (handler) | 3 |
| web/src/pages/PlatformTenantsPage.tsx | Button | native (handler) | 5 |
| web/src/pages/SystemConfigPage.tsx | Button | native (handler) | 4 |
| web/src/pages/TroubleTicketsPage.tsx | Button | native (handler) | 3 |
| web/src/pages/UserGuidePage.tsx | Button | native router | 1 |

## The mechanism lines

| code line | mechanism | status |
| --- | --- | --- |
| web/src/components/CustomMenu.tsx:25 | the menu is a static source array | pinned |
| web/src/components/CustomMenu.tsx:68 | the menu filters by identity level or issued permission | pinned |
| web/src/components/CustomMenu.tsx:14 | a filtered row renders as a focusable menu link | pinned |
| web/src/components/CustomMenu.tsx:90 | a section label is not a focusable element | pinned |
| web/src/App.tsx:132 | a resource declaration makes a route | pinned |
| web/src/App.tsx:208 | a custom route registers a page | pinned |
| web/src/components/CustomAppBar.tsx:9 | the app bar carries the only chrome above the menu | pinned |
| web/src/components/onboarding/GettingStartedCard.tsx:67 | the checklist hides admin steps from a lower level | pinned |
| web/src/components/onboarding/GettingStartedCard.tsx:56 | a checklist step opens its route through the router link | pinned |
| web/src/pages/UserGuidePage.tsx:14 | a guide chapter carries an ordered link list | pinned |
| web/src/pages/UserGuidePage.tsx:53 | a guide link renders as a button bound to the router | pinned |
| web/src/components/onboarding/QuickTourDialog.tsx:40 | a tour step renders as a button bound to the router | pinned |
| web/src/pages/Dashboard.tsx:465 | a dashboard card is a clickable action area | pinned |
| web/src/components/onboarding/onboardingStorage.ts:17 | the only per-operator state is browser storage | pinned |
| web/e2e/keyboard-navigation.spec.ts:137 | the repo's own keyboard cost unit is a Tab press | pinned |
| web/src/components/CustomerTimeline.tsx:207 | a record page links to a destination built from the record | pinned |

## The statements, pinned to code

| id | statement | code line | status | measured | asserts an absence |
| --- | --- | --- | --- | --- | --- |
| C01 | the sidebar is a static array the source orders, so a tab-stop number is a property of the code | web/src/components/CustomMenu.tsx:25 | confirmed | 23 destinations in source order | no |
| C02 | the menu filters rows by identity level or by an issued permission | web/src/components/CustomMenu.tsx:70 | confirmed | 9 gated rows, 14 shown to every role | no |
| C03 | platform_admin is the only permission string the server can hand a session | internal/adminapi/auth.go:171 | confirmed | issued permissions: platform_admin | no |
| C04 | an operator level is one of the three the API validates | internal/adminapi/operators.go:272 | confirmed | 6 role arms = levels admin, operator, super x issued permissions | no |
| C05 | every destination the menu offers is a route the app shell declares | web/src/App.tsx:132 | confirmed | 23 menu destinations, 0 of them without a declared route | no |
| C06 | a section label is text, not a focus stop, so grouping never costs the keyboard anything today | web/src/components/CustomMenu.tsx:90 | confirmed | 8 menu rows carry a section label, 0 of them focusable | no |
| C07 | the navigation region has no type-to-filter affordance, so order and grouping are the only levers | web/src/components/CustomMenu.tsx:14 | confirmed | 0 filter or palette controls in the app bar, menu and layout | yes |
| C08 | the browser role matrix states a visibility expectation for every gated destination and no ungated one | web/e2e/role-matrix.spec.ts:32 | confirmed | 9 gated destinations asserted, 0 gated destinations no test asserts | yes |
| C09 | the user guide is already written as tasks: each chapter is an ordered list of destinations | web/src/pages/UserGuidePage.tsx:14 | confirmed | 8 chapters, 32 destination links | no |
| C10 | the getting-started checklist is the only surface that filters its own steps by role | web/src/components/onboarding/GettingStartedCard.tsx:67 | confirmed | 7 steps, 5 marked admin, 1 of 17 surfaces filter | yes |
| C11 | a checklist step opens its destination through a router-bound button | web/src/components/onboarding/GettingStartedCard.tsx:56 | confirmed | carrier: Button (native), line 56 | no |
| C12 | the quick tour walks five destinations, all of them declared routes | web/src/components/onboarding/QuickTourDialog.tsx:9 | confirmed | 5 steps, 0 unresolved | yes |
| C13 | the dashboard cards are click targets that route, and they are action areas rather than plain boxes | web/src/pages/Dashboard.tsx:10 | confirmed | 7 cards, carrier CardActionArea (native), line 465 | no |
| C14 | a task surface can offer a destination whose menu entry the same role cannot see | web/src/components/onboarding/GettingStartedCard.tsx:21 | confirmed | 28 task x role cells offer a hidden destination: guide.billing -> /system/config (hidden by that role's gate), guide.daily -> /operations (hidden by that role's gate), guide.start -> /account/settings (no menu entry at all) | yes |
| C15 | the config page performs no client-side role check, so the menu filter is the only gate the console shows | web/src/pages/SystemConfigPage.tsx:78 | confirmed | 0 role checks inside the page | yes |
| C16 | the only per-operator adaptive state the console keeps is browser storage, and it survives neither another browser nor another machine | web/src/components/onboarding/onboardingStorage.ts:17 | confirmed | 1 storage sections, 0 operator UI-preference models in the domain | yes |
| C17 | the keyboard cost this study counts is the unit the repository's own keyboard test already counts | web/e2e/keyboard-navigation.spec.ts:139 | confirmed | 4 scenarios, 4 distinct destinations driven to | no |
| C18 | CI never runs the browser suite, so no gate in this repository observes a real focus order | .github/workflows/ci.yml:97 | confirmed | 0 playwright invocations in the workflow, 1 frontend build steps | yes |
| C19 | the console mounts exactly one layout, and the menu inside it is the product's own component | web/src/components/CustomLayout.tsx:13 | confirmed | 1 layout wiring, 44 console files measured | no |
| C20 | every destination a task offers is a route the shell already declares, so a workspace adds no application | web/src/App.tsx:208 | confirmed | 51 of 51 task edges resolve to a declared route; 2 further controls build their target at runtime | yes |
| C21 | ordering a task's own destinations in front of the menu never costs a stop, and across the corpus it removes most of the walk | web/src/components/CustomMenu.tsx:68 | confirmed | 2300 baseline stops vs 745 grouped stops over 102 task x role cells | yes |
| C22 | the menu's own row count changes with the role, so ordering cost is already adaptive and only in the direction the role cannot control | web/src/components/CustomMenu.tsx:40 | confirmed | operator sees 14 rows, admin+platform_admin sees 23 rows | yes |
| C23 | the resources behind the destinations are read-mostly declared: a task that needs a write has to leave the list | web/src/App.tsx:132 | confirmed | 13 resources: 13 list, 10 create, 8 edit, 12 show | no |
| C24 | no product code imports this study, and the study imports nothing that can write | .github/workflows/ci.yml:175 | confirmed | 0 importers outside internal/contextualws | yes |

## What this study does not close

| id | what the study cannot close | what it would need | kind of asset |
| --- | --- | --- | --- |
| G01 | task usability compared against the baseline is a human measurement: what this study can price is focus stops, not the time or the errors an operator makes while using them | a panel of operators on a seeded console | human respondents |
| G02 | the stop counts here are derived from source order; the rendered focus order, including everything the component library injects above the menu, is only visible in a browser | a Playwright run against the dev server with a seeded session | live browser session |
| G03 | a destination the menu hides may still work when reached, or may fail at the API; the study reads the menu, the server answers the request | a signed session per role and a page-load probe | live per-role sessions |
| G04 | putting a task group in front of the menu is a product decision about TR-F013, not a finding this study can make | a scope decision on TR-F013 | product owner |
| G05 | adaptive in the sense of learning from an operator's own work needs state the product does not have; the study measured that the state is browser-local | a schema and API decision for per-operator workspace state | schema change |
| G06 | 2 navigation controls in the console build their target at runtime, so a workspace cannot enumerate them from source | seeded records that exercise those controls | runtime fixtures |

## The import set this package builds on

| package import |
| --- |
| crypto/sha256 |
| encoding/hex |
| fmt |
| github.com/stretchr/testify/assert |
| github.com/stretchr/testify/require |
| go/parser |
| go/token |
| io/fs |
| os |
| path/filepath |
| regexp |
| sort |
| strconv |
| strings |
| sync |
| testing |

## The lines this study reads

| code line | what it carries |
| --- | --- |
| web/src/components/CustomMenu.tsx:25 | the menu is a static source array |
| web/src/components/CustomMenu.tsx:68 | the menu filters by identity level or issued permission |
| web/src/components/CustomMenu.tsx:14 | a filtered row renders as a focusable menu link |
| web/src/components/CustomMenu.tsx:90 | a section label is not a focusable element |
| web/src/App.tsx:132 | a resource declaration makes a route |
| web/src/App.tsx:208 | a custom route registers a page |
| web/src/components/CustomAppBar.tsx:9 | the app bar carries the only chrome above the menu |
| web/src/components/onboarding/GettingStartedCard.tsx:67 | the checklist hides admin steps from a lower level |
| web/src/components/onboarding/GettingStartedCard.tsx:56 | a checklist step opens its route through the router link |
| web/src/pages/UserGuidePage.tsx:14 | a guide chapter carries an ordered link list |
| web/src/pages/UserGuidePage.tsx:53 | a guide link renders as a button bound to the router |
| web/src/components/onboarding/QuickTourDialog.tsx:40 | a tour step renders as a button bound to the router |
| web/src/pages/Dashboard.tsx:465 | a dashboard card is a clickable action area |
| web/src/components/onboarding/onboardingStorage.ts:17 | the only per-operator state is browser storage |
| web/e2e/keyboard-navigation.spec.ts:137 | the repo's own keyboard cost unit is a Tab press |
| web/src/components/CustomerTimeline.tsx:207 | a record page links to a destination built from the record |

## What the digest locks

The digest is the sha256 of the tab-separated rows below, so the memo's tables
are reproducible from the tree alone and move the moment the tree moves.

- 138 menu stop rows
- 102 cost rows
- 102 gate rows
- 51 edges
- 21 affordance rows
- 16 mechanism lines
- 24 statements
- 6 limits

- digest: `24fbd60f48f066543bcfef48352f18e130cf22314c14a98586ced7d63b35efe2`

## Verification

```
go build ./...                                     # clean
go vet ./...                                       # clean
gofmt -l internal/contextualws/                    # empty
golangci-lint run ./internal/contextualws/...      # 0 issues
go test -p 1 -count=1 ./internal/contextualws/     # ok (24 statements confirmed)
go test -p 1 -count=1 ./internal/...               # ok, whole internal tree
grep -rn contextualws --include=*.go internal/ cmd/ | grep -v ^internal/contextualws/   # 0 importers
```

The tables above come from
`WS_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/contextualws/`,
so every number in the prose is the same run. The digest
`24fbd60f48f066543bcfef48352f18e130cf22314c14a98586ced7d63b35efe2` locks 138 menu
stop rows, 102 cost rows, 102 gate rows, 51 edges, 21 affordance rows, 16
mechanism lines, 24 statements and 6 limits; it changes the moment the tree
changes.

## No-go register

Six limits stay open, and none of them is closable from the tree: four need an
asset this round cannot produce (people, a live browser session, per-role
sessions, seeded records), one needs a schema decision, and one - `G04` - needs a
product-owner call. Each is recorded as **no-go** rather than as done:

| id | needs | kind of asset | verdict |
| --- | --- | --- | --- |
| `G01` | operator panel on a seeded console for task usability vs baseline | human respondents | **no-go** - the study prices stops, not time or errors |
| `G02` | Playwright run against the dev server with a seeded session | live browser session | **no-go** - rendered focus order is not derivable from source |
| `G03` | a signed session per role and a page-load probe | live per-role sessions | **no-go** - a hidden destination may still load; only the server answers that |
| `G04` | scope decision on `TR-F013` | product owner | **no-go** - whether task groups go in front of the menu is a decision, not a finding |
| `G05` | schema and API decision for per-operator workspace state | schema change | **no-go** - the measured state is browser-local |
| `G06` | seeded records that exercise the 2 runtime-built controls | runtime fixtures | **no-go** - not enumerable from source |

## Residual limits

- A stop is a *forward* focus move inside the navigation region. The component
  library injects focusable chrome above and beside the menu, shift-movements and
  the browser's own order are out of scope, and the digest cannot see them.
- Task surfaces are parsed from four known components
  (`UserGuidePage.tsx`, `GettingStartedCard.tsx`, `QuickTourDialog.tsx`,
  `Dashboard.tsx`). A future task surface written some other way is invisible
  until the parser learns it; `TestNoTaskPointsOutsideTheApp` is what makes that
  failure loud rather than silent.
- The guide's chapter links are the task definition the product ships. The study
  prices that order; it does not claim it is the order an operator would choose
  (that is `G01`).
- Cost is symmetric by construction: a task whose steps already run in menu
  order saves 0 stops (e.g. `guide.start` for a role that can see everything it
  links to). The invariant asserted is "never costs more, saves something in
  aggregate", which is what the corpus actually supports.
- 23 menu destinations, 13 resources and 12 in-layout custom routes are today's
  totals; every table here moves with them, and the claims fail the build when
  they do.
