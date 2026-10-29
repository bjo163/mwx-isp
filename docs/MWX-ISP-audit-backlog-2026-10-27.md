# 100-item portfolio triage checklist (2026-10-27)

Provenance: `MWX-ISP_100_ISSUES_MASTER_2026-10-27.md` (kept at the repository root as the
source inventory), baseline `main`/`dev` at `1b791f8` (v0.14.0). The inventory lists
100 draft IDs — 64 Engineering, 24 Frontier, 12 Economy — with Charity and Faith/ontology
set to zero because they do not apply to a RADIUS/ISP monolith.

**These are not GitHub issues and are not approved work.** #45–#54 (AUDIT-01..10) are closed
prior art and must not be re-opened or duplicated by these drafts. 96 of the 100 items are
`PROPOSED_VERIFICATION`: they order an inspection or a test first, so none of them is a
confirmed defect. Only the `VERIFIED_GAP` rows record something already demonstrable against
the current tree.

## Triage tags

| Tag | Meaning | Gate before code changes |
| --- | --- | --- |
| `LOKAL` | Executable inside this repository — read, assert, test, document. | Ordinary TR-F scope and the Go/web gates. |
| `ASET` | Needs a real asset: physical NAS, clean VPS host, LDAP/WA number, or an operator willing to be measured. | Blocked until the asset and its authorization exist. |
| `RISET` | Frontier/Economy experiment. Research notes and bounded prototypes only. | No product code until `docs/feature-checklist.md` gains an approved TR-F. |
| `RISET+ASET` | Research that additionally needs real data or hardware. | Both gates above. |

## Execution rules

- One item per round. Check the box only when its own acceptance criterion is met with
  citable evidence, and note the commit or PR next to it.
- Never widen `JwtSkipPrefix`, revive a payment endpoint, or add a customer-facing workflow
  to make a UI claim true — those are TR-N001/TR-N002 non-goals that need a scope decision.
- `main`/`dev` branch policy, tags, releases and existing issues stay untouched without a
  separate instruction.

## Progress

Done: 61 / 100.

- `FUT-020` (2026-10-29, `dev` @ `a50f99f`): per-tenant private operational
  statistics, delivered as a census of what a served response already reconstructs
  plus a price for the governance that would stop it. The criterion has two halves
  and neither was checkable: no publication switch, cell floor or suppression rule
  exists in the tree, and nothing said which served number is already a person.
  `internal/opstats` reads the **11** statistics surfaces off the router, attributes
  all **157** served fields to the statement that produces them (same line,
  lookback, callee walk, chained GORM table), and walks a six-tenant fixture
  (**1** customer to **120**, **1189** stored rows over **12** tables, every row
  uniquely marked) through each of the **73** aggregate verbs. Scope is measured per
  surface, not assumed: **6** reach the database through the shipped tenant callback,
  **1** through a written comparison, **2** substitute the default tenant, **2** are
  platform surfaces that are meant to cross tenants, and **0** are unclassified.
  **Fixture tenant data unik tidak bisa direkonstruksi** is false inside a tenant and
  true across them: **0** aggregates over a subscriber table are served from a
  cross-tenant surface today, while **27** are a cell of one — **3** publish a stored
  value verbatim (`/dashboard/isp-stats outstanding` = `SUM(isp_invoice.balance)`,
  `today_input_gb`/`today_output_gb` = `SUM` over `radius_accounting`) and **24** say
  only that the tenant has exactly one subscriber, invoice or session. The extra
  finding that carries the weight is a rate in disguise: `ipv6_stats.adoption_rate`
  divides
  `online_with_ipv6` by `online_users` and the response serves both, the operand
  resolving through the call at `dashboard.go:150` and the alias at `:309` to the
  division at `:343`. **Publish hanya bila governance jelas** has no antecedent:
  **0 of 36** registry keys speak of statistics, benchmarks, aggregates,
  anonymisation or sharing, **0** suppression-vocabulary lines reach a statistics
  file, **0** statistics handlers call the shipped redactor, `dashboard_test.go`
  asserts tenant scope **0** times and the TR-F033 wording never says an aggregate
  may leave its tenant — while **8** production sites substitute the default tenant
  when no request context is present, which is exactly how a per-tenant number gets
  mis-attributed. The one governance shape that ships is the per-tenant notification
  opt-in (`domain.NotificationSettings`, **7** fields, `RiskAcknowledgedAt`), and
  that is what a publication gate would copy. A floor is priced, not chosen: `k=1`
  leaves **27** exposures, `k=2` leaves **0** but blanks **31 of the 73** fields for
  at least one tenant, and `k=3`/`k=5` add nothing — so the cost falls on `solo` and
  `pair`, the RT/RW Net end.
  [Study](per-tenant-operational-statistics-study-2026-10-29.md), `internal/opstats`
  (read-only, imported by nothing, stdlib-only, 29 gates, digest
  `7fc52cdec5af2d09bb4aa87aa6584d60d6fe5056f850d0aad2467bd2cbffe461`, **26 of 26**
  statements confirmed). **No-go, explicitly:** real tenant populations (`G01`), the
  decision to publish an aggregate outside its tenant, which is a TR-F scope change
  (`G02`), a live PostgreSQL to answer each parsed statement (`G03`), the threshold
  `k` itself (`G04`), a collision study over real per-tenant values (`G05`), the
  platform-admin trust boundary that serves another tenant's `tax_id`, billing
  address and contact phone (`G06`), and closing the default-tenant fallback, which
  is product code in `internal/adminapi` (`G07`). Research-only: nothing in this
  round deploys.

- `FUT-018` (2026-10-29, `dev` @ `de91f23`): printable grunge incident card,
  delivered as a measurement of what such a sheet may say rather than as a mockup.
  No card component, exporter or proof sheet exists in the tree, so the criterion's
  three claims were each made checkable against the product as it ships.
  **Tidak membocorkan nama pelanggan** is answered by four independent readings of
  one field set: **129** slots served by the **15** incident response types, of
  which **108** print and **21** are refused — by the shipped classifier parsed out
  of `internal/redact` (**7**), by a select list that proves the value is an
  identity column (**10**), by a `Sprintf` that composes such a column into prose
  (**3**), and by an assignment trace for keys that literally say `name` (**1**,
  `SyslogEvent.severity_name`, refused for being unresolvable). No customer name
  prints; the three name-shaped fields that do (`nocImpactScope.name`,
  `nocImpactScope.nas_name`, `nocImpactMonitoring.target_name`) print only because
  the trace proves their assignments read `isp_odp`/`net_nas`/`net_monitor_target`
  rows, and `subscriberRows = {isp_customer, radius_user}` is named as the one
  interpretive choice. The load-bearing finding is that **13 of the 21 refusals sit
  past every shipped rule** — `isp_customer.name`, `customer_no`,
  `radius_user.username` and `realname` are all `Plain` to `internal/redact`, and
  `TroubleTicket.subject` has a name formatted into it at `isp.go:757` — so a card
  built on "no rule forbids it" leaks subscriber identity while keeping CI green.
  **Print A4 hitam-putih dapat dibaca** is priced as greys, not chosen: all **18**
  light tokens clear the 4.5:1 floor on the white sheet the console's own print rule
  forces (worst **8.32:1**) while **8 of the 18** dark ones fail — the default skin
  (`A16`) prints `warning.main` at **3.03:1** and `success.main` at **3.45:1** — and
  **28 of 30** status pairs dry to the same grey, so the state has to ride the
  label, not the fill. **Stale/confidence terlihat** is split honestly in two:
  **10** such slots reach the ledger and print, **6** are read by `web/src`
  (`stale` **12** lines, `confidence` **7**) and **4** (`confidence_reason`,
  `stale_after_seconds`, `last_checked_at`, `interval_seconds`) are not read at all,
  while **0 of the 6** incident surfaces has a print path, so the paper half of the
  claim is currently empty. The sheet also cannot hold the answer: the skin's own
  metrics budget **32** slot rows against the **108** the ledger allows.
  [Study](printable-incident-card-study-2026-10-29.md), `internal/incidentcard`
  (read-only, imported by nothing, stdlib-only, 22 gates, digest
  `e667b8e47e590ac663f3b087de00fb1d7dcceb7db9724a22bfb9f764c26453ff`). **No-go,
  explicitly:** a real monochrome proof sheet (`G01`), a declared page geometry —
  zero `@page` rules exist (`G02`), a background-graphics decision — zero
  `print-color-adjust` declarations (`G03`), a shipped display face (`G04`), an
  exporter, since none exists in `web/package.json` or `go.mod` (`G05`), a redaction
  rule that reaches a name inside prose, which is a change to `internal/redact` and
  therefore product scope (`G06`), and incident data from a running deployment
  (`G07`).
- `FUT-016` (2026-10-29, `dev` @ `808bf61`): voice-assisted navigation for
  internal accessibility, delivered as a measurement of what a navigation-only
  vocabulary may be asked to do rather than as a microphone demo. The criterion's
  three claims were each made checkable. **Mutations are unreachable by
  construction**: the speakable vocabulary is exactly the **23** menu rows as the
  two shipped locales render them (**46** phrases, **0** destinations speakable in
  neither locale), and the answer carries a menu path with no method, id or
  payload, so **0** of the **78** write registrations is expressible -
  `TestTheAnswerTypeCarriesNoOperation` reflects over the type rather than trusting
  the prose. Behaviourally, an operation word derived from the server's own guard
  line and write-route tails (**17** action words → **32** refusal tokens, pruned
  to **28** because **4** are also screen words - `operators` names both
  `ApiPOST("/platform/tenants/:id/operators")` and the Operators screen) refuses
  before scoring: **7 of 7** operation transcripts refused. **Salah transkripsi**:
  over **58** authored transcripts in **14** error classes, **57** give the answer
  the study requires, **0** confident answers name a screen the transcript did not
  ask for and **0** reach a mutation; the single mismatch is `peladangan` at **70%**
  against a **78%** accept floor, reported rather than tuned away. The corpus broke
  the naive scorer once - `jaringan` was nearly answered as `/operations` - and the
  fragment-eligibility rule (a candidate may have no more words than the transcript
  has tokens) is what refuses it now, with **16-17** of **22-23** resembling labels
  dropped per one-word transcript. **Teks fallback**: measured per destination
  through `App.tsx`'s import graph - **23 of 23** trace to a page, **140** text or
  search inputs, **8** pages bind a filter - and the honest answer is a split one,
  **12** screens declare no typed input of their own, so the fallback there is
  navigation-level, not typed. **Voice optional** is a fact of absence, not a
  promise: **0** speech-API references in `web/src` and `web/e2e`, **0** microphone
  or recognizer entry points in `internal`. Two findings stay as refusals: the **9**
  gated destinations remain speakable exactly as their URLs stay reachable, and **4**
  Indonesian operation sentences are stopped by nothing but the answer type.
  **15/15** statements pin to code lines, **6** gaps stay open (an audio corpus, a
  human panel, a live browser session, per-role sessions, the `TR-F013` vocabulary
  scope call, a device and duty-cycle decision). Digest
  `443ac7d1d52ff577c7d47b2189cdc9abc03f6a8861dc7037e4f8e38e7b803b95`.
- `FUT-015` (2026-10-29, `dev` @ `e1b5261`): the adaptive contextual workspace,
  delivered as a measurement of the navigation the console already ships rather
  than a mockup of one it does not. The cost unit is borrowed from the repository
  - a forward focus stop inside the navigation region, what `tabTo()` presses in
  `web/e2e/keyboard-navigation.spec.ts` - and both columns count it over the *same*
  deduplicated destination sequence, so the only difference is which list numbers
  the stops. Measured over **17** task surfaces (8 guide chapters, the setup
  checklist, the quick tour, **7** dashboard cards) x **6** role arms derived from
  the levels the operator API validates crossed with the one permission string the
  auth handler can issue: **51** of **51** destination edges resolve to a declared
  route and **51/51** ride a keyboard-native control, **0** need a pointer - the
  criterion's "keyboard fallback setara" is already satisfied by construction.
  Ordering a task's own destinations in front of the menu removes **1555** of
  **2300** stops (**745** grouped, largest single task **47**) and never adds one.
  The load-bearing finding is a disagreement, not a saving: **28** task x role
  cells offer a destination that the same role's menu hides - `/system/config`,
  which the menu gates to the `platform_admin` permission while the checklist's own
  filter knows levels only, and `/operations` for plain `operator` - and
  `/account/settings` is reached by tasks yet appears in **no** role's menu, while
  `SystemConfigPage` runs **0** client-side role checks. The "adaptive" half is
  recorded as refusal: the only per-operator state is browser storage, **0**
  operator preference models exist in the domain, and CI runs **0** Playwright jobs
  so no gate observes a rendered focus order. **24/24** statements pin to code
  lines, **6** gaps stay open (human respondents, a live browser session, per-role
  sessions, a `TR-F013` scope decision, a workspace-state schema, runtime fixtures).
  Digest `24fbd60f48f066543bcfef48352f18e130cf22314c14a98586ced7d63b35efe2`.
- `FUT-007` (2026-10-29, `dev` @ `a02c698`): the bandwidth-profile what-if
  estimator, delivered as a reproducible wire diff over the product's own answer.
  The criterion's two halves were both made checkable. Reproducible: held-out keys
  are **derived**, not listed - shift only `user.ExpireTime` and diff, or parse each
  `getIntConfig` in `default_enhancer.go`, join it to the attribute it writes and
  compare with the source's own fallback - so **2** keys (`Session-Timeout`,
  `Acct-Interim-Interval`) leave the ledger, and the digest carries their keys
  without their values. That last clause was a bug, not a design choice: the first
  digest hashed the ticking value and changed every second on an idle tree, while
  the reproducibility test still passed because its two runs landed in one second;
  a test now sleeps across a boundary to keep it honest. Non-mutating: the getters'
  contract is a single `Get(int64)` (verified by reflection against
  `domain.ProfileCacheGetter`, against the **5** mutators `ProfileCache` itself
  exposes), and `TestTheEnhancersThemselvesWriteNothing` hands the enhancer chain
  pointers straight into the tracked tables instead of copies.
  Measured over **10** subscribers x **8** vendors with **8/8** registered enhancers
  running: **80** one-column probes, **36** silent; **3** of the **10** inherited
  columns (`ActiveNum`, `BindMac`, `BindVlan`) are read by a checker and by no
  enhancer, so they change admission, not the accept; one `UpRate` edit is one
  combined string on Mikrotik but average plus peak on Huawei and a VSA plus
  `Framed-Pool` on Cisco; user overrides and static snapshots both freeze the diff
  to empty, while an unresolvable profile falls back to the subscriber's stale
  values and emits `0k/0k` with `NA` pools. **23/23** statements pin to code lines,
  **7** gaps stay open - NAS confirmation, live-session impact, CoA
  re-authorization, and the preview route itself, which is a `TR-F008` product
  decision. Memo: `docs/profile-whatif-bandwidth-simulation-2026-10-29.md`.

- `FUT-002` (2026-10-29, `dev` @ `62c80bb`): incident triage over the sanitized
  operator log export, delivered as a measurement rather than a model. The card
  asked for a labeled dataset, a reported hallucination rate and no automatic
  remediate, so all three had to be made checkable: `SanitizedLine` is only
  producible through the export's own `internal/redact` path, a hallucination is
  defined in code as a named cause whose evidence does not hold (not as "wrong"),
  and the no-remediate half is read off the type by reflection plus the **9**
  mutating verbs measured from the admin API's own route registrations.
  `internal/incidenttriage` (read-only, imported by nothing) runs three engines
  over **20** synthetic incidents built from message templates the product
  writes, **17** with one cause label and **3** labeled `unresolved` on purpose:
  the token rules score **100%** accuracy at **0%** hallucination by abstaining on
  exactly those three, a subsystem-name baseline fitted on the same labels answers
  **19 of 20** and every answer is unsupported (**42.1%** accuracy, **100%**
  hallucination), and a never-abstains counterfactual buys **3** extra decisions
  for a **15%** hallucination rate - which is the number anyone shipping a
  "summarize this incident" button would be paying. **20 of 20** taxonomy signals
  resolve to a product code line, the corpus planted **14** secret values and
  **0** survived the redaction sweep, and the incident-by-incident table prints
  the log line and the `file:line` behind every verdict. Two findings came out of
  the scan contradicting the draft memo, and the draft moved instead of the
  numbers: `namespace` - one of the two filters the export offers - is written with
  only **3** literals across **55** sites and `job` is written nowhere, and
  `internal/domain`, `internal/notify` and `internal/ldapauth` attach no namespace
  at all, so four of the ten cause classes reach the operator in an empty subsystem
  bucket. **16** statements confirmed, **6** gaps open, **4** mutations of the
  safety properties all caught by the suite. **5-row no-go register** in
  `docs/incident-triage-sanitized-logs-2026-10-29.md`: a shipped triage surface
  (`TR-F031` scope decision, `TR-N003` boundary), precision on real tenant logs
  (needs a real labeled incident corpus - six of ten classes here have one
  incident), any AI/LLM comparison (needs a sanctioned model and the data-handling
  decision; `no-model-client` is an open gap), per-tenant log access (needs a live
  seeded admin session per role), and remediation suggestions of any kind (the
  card's criterion is explicitly *no* automatic remediate). Acceptance: `go test
  -p 1 -count=1 ./internal/incidenttriage/` passes 28 cases with the dump skipped,
  `golangci-lint run ./internal/incidenttriage/...` reports 0 issues, and the
  package has no importers in `internal/`, `cmd/` or `pkg/`.

- `FUT-001` (2026-10-29, `dev` @ `8f9b85c`): the read-only intent search proof of
  concept, delivered as a resolver whose safety half is structural rather than
  promised. The card asked for prompts that navigate menus and compose filters
  without running a NAS or billing mutation, so the order of the resolver is the
  result: deny first, then score, then absorb-or-refuse, then compose, and a
  resolution can only ever name a `GET` registration. `internal/intentsearch`
  (read-only, imported by nothing) scans the console instead of carrying a
  hand-written map: **68 read** and **78 mutating** registrations behind **13 menu
  entries**, **12 declared filter inputs** with their value vocabularies, and the
  data provider's own menu-name-to-path table (**10** entries, **4** rename a menu
  name, **4** menu entries are not in it at all). The deny vocabulary is measured
  too - **9** words from a non-GET route segment plus **5** from the action-value
  guard at `internal/adminapi/isp.go:1188`, union **12** - because `suspend`, the
  mutation operators ask for most, never appears in a path, so any list built from
  paths alone is incomplete by construction. All **20** criterion prompts resolve
  as expected (**10** navigate, **6** with a composed filter, **4** refuse), the
  corpus reaches **13 of 13** cataloged screens, and it names **0** targets that
  are not a `GET` route; only **2 of 6** filter prompts get a value, because a
  value may come from the page's own vocabulary and nothing else - `paket yang
  aktif` stops at the screen rather than inventing `active`. **18** statements are
  pinned to a code line with comment lines blanked first, and **5** gaps are
  re-checked on every run. Two findings constrain any future palette regardless of
  the parser: the customer list accepts no date parameter, and the dashboard totals
  live at `GET /dashboard/stats` with no `<Resource>` entry to point at. **5-row
  no-go register** in `docs/operator-intent-search-2026-10-29.md`: a shipped
  palette (needs a `TR-F013` scope decision), measured resolution quality (needs
  named respondents - the 20 prompts are constructed, not observed), a
  state-word translation layer (needs an owner decision), free text beyond the
  catalog (`TR-N001`/`TR-N002`), and per-role intent visibility (needs a live
  seeded admin session per role). Acceptance: `go test -p 1
  ./internal/intentsearch/` passes 21 cases with the dump skipped,
  `golangci-lint run ./internal/intentsearch/...` reports 0 issues, and the package
  has no importers in `internal/` or `cmd/`.

- `FUT-017` (2026-10-29, `dev` @ `7a8b944`): the manga skin study, resolved as a
  measurement of the direction that already shipped rather than a second one. The
  card asked for an "experimental neo-brutalist manga skin via existing tokens",
  but `TR-F029` sanctions exactly one manga-ink direction and names a theme/plugin
  marketplace as a non-goal - so building a competing skin would be a scope
  decision, not research. `internal/mangaskin` (read-only, imported by nothing)
  parses `web/src/theme.ts` and resolves **22 operator-visible states** in both
  modes through the token path: **44 color pairs**, **40 judged by WCAG AA**, **4
  measured but exempt** (decoration and inactive text, reported so the pass rate
  cannot hide them). Every pair is computed from the sRGB linearisation in the
  package, never eyeballed, and **14 positive claims** plus **5 negative claims**
  are pinned to a code line with comment lines blanked first. Result: **one AA
  violation** in the shipped skin - the dark panel outline at **2.34:1** against
  the 3:1 non-text floor, which is `alpha(ink, isDark ? 0.28 : 0.7)` at
  `web/src/theme.ts:72`, and closing it uses the `0.7` the same line already
  writes for light mode. `Ink Max` proves the closure costs **0 new colors**
  (`TestInkMaxClosesTheHeldViolationsWithoutIntroducingAColor`) and lifts the two
  exempt empty-value pairs to 8.35/8.87:1. Rollback is met structurally, not by
  promise: **0 stylesheets under `web/`**, **1 `createAppTheme` definition**
  feeding **3 call sites**, both modes from it, plus the branding screen's
  restore-to-defaults action - so a skin decision never needs a CSS fork. The
  honest limit is scope: **170 literal colors** live outside `theme.ts` across 84
  files (**119** of them in `LandingPage.tsx`) with **61 hand-written**
  `mode === 'dark' ?` branches, and **four status groups declare no
  `contrastText`**, leaving the label over a filled chip to the component library;
  a token-level decision therefore re-skins the console, not the site. **5-row
  no-go register** in `docs/manga-skin-contrast-audit-2026-10-29.md`: the
  browser-rendered AA pass (needs a live seeded admin session), applying the alpha
  change (needs a `TR-F029` scope decision), operator preference between
  directions (needs human respondents), a bundled display webfont (needs a
  licensing decision), and a forced-colors/Windows-high-contrast pass (needs a
  direction decision). Acceptance: `go test -p 1 ./internal/mangaskin/` passes 33
  cases, `golangci-lint run ./internal/mangaskin/...` reports 0 issues, and the
  package has no importers in `internal/` or `cmd/`.

- `ECO-006` (2026-10-29, `dev` @ `4b2ea0a`): onboarding and migration service
  validation, measured off the importer instead of argued. `internal/migrationcheck`
  (read-only, imported by nothing) pins **20 statements** the service rests on to a
  Go identifier in product code and holds **5 negative claims**; all 20 confirm and
  all 5 gaps stay open - the importer has no `dry_run` flag, no hashed-password path
  and no upsert, and there is no profile/package or node/NAS import handler at all,
  so the plan catalog every customer row depends on is still rebuilt by hand. The
  vocabulary is measured rather than assumed: **one upload handler** in the whole
  admin API (`users_import.go:64`) accepts **18 columns under 41 header spellings**,
  refuses a row for **12 distinct reasons**, and **11 values** a submitted row never
  states because they are inherited from the billing profile or defaulted. The
  effort half is deliberately 6 t-shirt-sized items with no clock or currency
  figure (`TestEstimateCarriesNoInventedNumbers`), because nothing in this tree
  derives either. Evidence rules carry over from the ENG-006/ENG-007 studies:
  comment lines blanked first, every citation must land on a code line, drift
  reported `broken`/`vacated` instead of dropped, and the scan excludes its own
  package. The dry-run half of the criterion resolved against implementation: the
  validation phase is already side-effect-free by construction and 8 named tests
  assert the zero-write outcome, whereas shipping the proposed preview would deploy
  a surface before its checklist was approved. **4-row no-go register** in
  `docs/onboarding-migration-service-2026-10-29.md`: client interviews and their
  willingness-to-pay (needs named respondents and consent), a rehearsal against a
  real operator export (needs a client file), any price or duration figure (needs
  the two above), and shipping the dry-run preview (needs a checklist decision).
  31 tests pass, lint clean.

- `ENG-007` (2026-10-29, `dev` @ `eff93d1`): EAP-PWD need research for M12,
  re-derived from the tree rather than recalled. `internal/eappwdneed` (read-only,
  imported by nothing) pins the 11 statements the need argument rests on to a Go
  identifier in product code and tests 3 negative claims. All 11 confirm and all 3
  gaps stay open: every password path still needs a recomputable secret
  (`md5_handler.go:97`, `mschapv2_handler.go:264`, `ldap_backend.go:120`), neither
  tunnel can carry a third inner method (`peap_inner.go:148`, `ttls_inner.go:89`),
  the type constants stop at EAP-TLV 33 (`interfaces.go:32`), and no `eap-pwd`,
  `TypePWD` or `dragonfly` code exists. Evidence rules follow the ENG-006 study:
  comment-only lines are blanked so a doc comment naming a symbol proves nothing
  (`TestCommentMentionsAreNotEvidence`), every citation must land on a code line
  (`TestEveryRealCitationIsACodeLine`), a moved citation is reported `broken`
  instead of dropped (`TestMovedFileIsReportedBrokenNotDropped`), and the scan
  excludes its own package so it cannot refute itself
  (`TestAbsenceScanIgnoresThePrototypeItself`). The vendoring half carries the
  finding that matters: the audit reads the number each carried RFC prints about
  itself and compares it with the numbers this project's own documents cite
  (`TestEverySpecNumberIsCitedByAProjectDocument`), which leaves **2 findings
  across 56 files** - RFC 5931 is carried nowhere while
  `docs/rfcs/rfc7542-eap-pwd.txt` is the Network Access Identifier spec, and
  `docs/rfcs/rfc3851-eap-md5.txt` is S/MIME although EAP-MD5 is RFC 3748 §5.4.
  `rfc7170-teap.txt` is kept as a positive control so "missing" and "misnamed"
  cannot be confused. `docs-site/src/en/rfc-index.md` already records both cases;
  `docs/rfcs/FILE-NAMING.md` still asserts the wrong pairings, and this round
  renamed no file and fetched no document - that needs its own decision.
  Decision: M12 stays Planned, gated on a document rather than on an asset, and no
  method, enum entry or endpoint was added. Acceptance:
  `go test ./internal/eappwdneed/` passes 25 cases and
  `golangci-lint run ./internal/eappwdneed/...` reports 0 issues. Evidence:
  [eap-pwd-m12-need-2026-10-29.md](eap-pwd-m12-need-2026-10-29.md).
- `ENG-006` (2026-10-29, `dev` @ `e1ae01e`): TEAP readiness for M11, measured
  instead of remembered. `internal/teapreadiness` (read-only, imported by
  nothing) holds a catalog of the 22 mechanisms RFC 7170 needs and scans the
  product code for the Go identifiers that would prove each one: 7 reusable now
  (tunneled framing, fragmentation, keying-material export, Result TLV,
  protected success, inner AVP codec, anonymous NAI), 1 **blocked** rather than
  missing — `tls_config.go:257` pins the tunneled handshake to TLS 1.2 while
  TEAP's outer tunnel is the TLS 1.3 form — and 14 absent, including the two that
  decide whether M11 is worth scheduling: Crypto-Binding TLV and inner-method
  chaining. Evidence rules were the whole problem: the tree already contains
  "Cryptobinding", inside a comment that says the TLV is skipped, so comments are
  stripped before matching (`TestCommentMentionsAreNotEvidence`), `_test.go` and
  `testdata` are excluded (`TestScanReadsOnlyProductCode`), and word boundaries
  stop the generated `msaccteaptype_del` from reading as the method
  (`TestSubstringLookalikesDoNotProveTEAP`). Every requirement quotes a document
  under `docs/rfcs`; RFC 9427 and RFC 9930 are not vendored, so those two name
  their RFC number and say they cannot be cited locally
  (`TestEveryCitationIsQuotableHere`). Scope stayed bounded: the word Windows,
  macOS, iOS, Android or wpa_supplicant may not appear in any verdict
  (`TestNoVerdictClaimsClientCompatibility`), and five `Unknown` entries carry the
  client questions to the assets `ENG-002` and `ENG-013` need. Decision: M11
  remains Planned — no handler, config enum or endpoint was added. Acceptance:
  `go test ./internal/teapreadiness/` passes 18 cases plus 5 subtests and
  `golangci-lint run ./internal/teapreadiness/...` reports 0 issues. Evidence:
  [teap-m11-readiness-2026-10-29.md](teap-m11-readiness-2026-10-29.md).

- `ECO-009` (2026-10-29, `dev` @ `dcf4a7d`): revenue-leakage audit over the ledger
  that already exists. The rule was that no rupiah may be claimed without a row
  behind it, so `internal/revenue` (read-only, imported by nothing) states its
  findings as exceptions to eleven guards that `internal/billing/service.go` and
  `internal/radiusd/radius.go` already enforce — `generateMonthlyInvoices`,
  `ProcessOverdueInvoices`, `SuspendOverdueSubscriptions`, `RecordPayment` — and
  `TestEveryRuleIsTraceableToRealSource` opens those files and asserts the named
  function exists. Eleven kinds; only four carry an IDR figure (unbilled periods
  at the package price, a past-grace balance, a duplicate charge, money booked on
  a void invoice); seven refuse one and `Totals.Unquantified` counts them, because
  a still-enabled account or an undisconnected session has no price in any stored
  row. `Audit` partitions rows by `tenant_id` before applying any rule, so a
  cross-tenant finding is structurally impossible, and each slice is sorted before
  indexing, so row order cannot change the verdict. Showcase: 8 findings,
  IDR 400000 at stake, 5 of them deliberately unquantified. Decision: stay a
  prototype — TR-N001 forbids growing this into charging/ERP reporting, and no
  endpoint, dashboard or schedule is added. Acceptance: `go test
  ./internal/revenue/` passes 28 cases (including the negative ones: clean ledger,
  grace still running, month before its billing day, reversed payment, manual
  suspension) and `golangci-lint run ./internal/revenue/...` reports 0 issues.
  Unproven half of the criterion: the sanitized sample export — `Load(db)` is
  ready for it, but no sample database exists locally, so every number here is
  synthetic. Evidence:
  [revenue-leakage-audit-2026-10-29.md](revenue-leakage-audit-2026-10-29.md).

- `FUT-023` (2026-10-29, `dev` @ `f4de6b2`): RFC→feature→test navigation graph.
  Nothing joined `docs/rfcs`, the checklist entry points, and the Go files that
  cite a number, so "which test implements RFC 3576?" was a grep. The prototype
  `internal/protocolmap` (read-only, imported by nothing) derives that graph and
  renders every unprovable link as `UNKNOWN`. Matching rule: the citation regex
  also counts import paths (`layeh.com/radius/rfc2866`) and document links;
  entry-point ownership resolves to the **most specific** declaration, otherwise
  `TR-F033`, which declares `internal/radiusd` among others, absorbs every
  protocol row. Survey: 56 references, 22 linked, **34 never named anywhere**
  (incl. RFC 6613 RADIUS-over-TCP while RFC 6614 RadSec is linked, RFC 5997
  Status-Server, EAP-SIM/AKA/TEAP set); 13 cited numbers have no document, split
  between real gaps (5705, 8446, 7627, 9427, 4515) and general-internet RFCs used
  as vocabulary (3339, 5424, 1918…); 8 features own protocol-package code with no
  anchor; 28 citing files sit outside every entry-point declaration
  (`test/integration`, `cmd`, `scripts`, the research prototypes); 0 references
  reach code without test evidence. Decision: keep it unenforced — unlinked
  references stay in the library on purpose, and no CI gate depends on the
  counts. Acceptance: `go test ./internal/protocolmap/` passes 14 cases including
  the deterministic repository survey and `golangci-lint run ./internal/protocolmap/...`
  reports 0 issues. Evidence:
  [radius-protocol-navigation-graph-2026-10-29.md](radius-protocol-navigation-graph-2026-10-29.md).

- `FUT-011` (2026-10-29, `dev` @ `319699a`): generated RADIUS mutation
  regression corpus. ENG-011 pins 22 named frames and fuzzes the parser, but a
  fuzz crash file is not a reviewable artifact and a fixed corpus only covers
  the damage someone thought of. `internal/radiusd/mutation_corpus_test.go`
  now derives 137 positional mutants from the eight valid seed frames under 512
  bytes using seven kinds (bit flip, truncate, declared length, attribute
  length, duplicate attribute list, drop-byte keeping length, drop-byte fixing
  length) and pins each mutant's parse outcome class in
  `testdata/radius_mutation_corpus.json`. Distribution: 41 `packet`, 37
  `short_read`, 34 `bad_attributes`, 25 `malformed_frame`, every kind populated.
  A parser change that reclassifies any mutant fails
  `TestRadiusMutationCorpusMatchesGolden`; regeneration is an explicit
  `-update-radius-mutation-corpus` run, so the golden diff is the review
  signal. Decision: keep outcome classes (not error text) as the contract, and
  keep the file generation-time only — no product code imports it. Acceptance:
  `go test ./internal/radiusd -run 'TestRadiusMutation'` passes 137 subtests,
  the full package passes (42.3 s), and `golangci-lint run ./internal/radiusd/...`
  reports no findings in the new files. Evidence:
  [radius-mutation-regression-2026-10-29.md](radius-mutation-regression-2026-10-29.md).

- `FUT-006` (2026-10-29, `dev` @ `f026ff4`): static NAS-ODP graph. The ODP
  registry has `olt_name` and `pon_port` as free text and no NAS foreign key, so
  a topology map can only be derived by name. The prototype
  `internal/topology` (analysis only, imported by nothing) matches the trimmed
  case-folded OLT name against NAS name, then hostname, then identifier inside
  one tenant, keeps every candidate when a name is ambiguous, refuses to cross
  the tenant boundary, counts distinct non-terminated customer ports, and
  renders a deterministic script-free SVG with dashed candidate edges and
  unlinked boxes. Findings: one typo unlinks a whole splitter (`OLT-ZTE-2` vs
  `OLT-ZTE-02`), names are not unique, cross-tenant collisions are real
  (`ENG-018`), and `pon_port` is opaque text. Decision: no product UI until an
  explicit validated ODP-to-NAS relation exists; the renderer is not the
  blocker, the data model is. Acceptance: `go test ./internal/topology/`
  passes eight cases and `golangci-lint run ./internal/topology/...` reports 0
  issues. Evidence: [nas-odp-graph-2026-10-29.md](nas-odp-graph-2026-10-29.md).

- `FUT-022` (2026-10-29, `dev` @ `31551a4`): event replay simulator. Billing and
  RADIUS rows record outcomes but not the path to them. The prototype
  `internal/replay` (read-only, imported by nothing) folds the ordered
  `isp_billing_event` trail with the payment rows into paid/balance/status, and
  replays `radius_accounting` plus the online, action-queue, and audit rows into
  a session lifetime, then reports every disagreement with the stored rows.
  Findings: the billing trail is sufficient to re-derive money state because
  event descriptions name payment numbers and the payment rows carry the
  amounts; interim updates are not stored, so session replay reproduces
  contradictions, not traffic; a failed billing disconnect that the audit
  records as successful and the online row contradicts is now one report. A
  divergence states both values and never writes. Decision: keep it an
  unenforced prototype until there is a production corpus to measure false
  positives, a tenant/time bound, and an audited operator surface. Acceptance:
  `go test ./internal/replay/` passes sixteen cases against pure fixtures and
  SQLite, coverage of every divergence kind, and
  `golangci-lint run ./internal/replay/...` reports 0 issues. Evidence:
  [event-replay-simulator-2026-10-29.md](event-replay-simulator-2026-10-29.md).

- `FUT-009` (2026-10-29, `dev` @ `c372ee3`): billing-suspend explanation
  trace. `suspension_reason=billing_overdue` says that billing suspended a
  subscription but not which invoice or threshold did it. The prototype
  `internal/billing/suspendtrace` (analysis only, imported by nothing) reads
  the subscription, the earliest unpaid overdue invoice, the linked RADIUS user,
  and the `invoice_overdue`/`subscription_suspended` events, and returns ordered
  steps with evidence plus explicit gaps. Findings: the common case is fully
  reconstructible; grace days are read live (changing them rewrites the
  explanation) and the job clock is not stored; a later payment leaves the
  suspension without a matching invoice, so the trace reports `not_matched`
  instead of implying the paid invoice caused it; manual suspensions record
  neither operator nor reason beyond `suspension_reason`; a re-enabled user is
  flagged as divergent. Decision: keep the explanation derived, never stored,
  and record any new rule input with the suspension event if the rule changes.
  Acceptance: `go test ./internal/billing/suspendtrace/` passes six cases and
  `golangci-lint run ./internal/billing/suspendtrace/...` reports 0 issues.
  Evidence:
  [billing-suspend-explanation-trace-2026-10-29.md](billing-suspend-explanation-trace-2026-10-29.md).

- `FUT-010` (2026-10-29, `dev` @ `e521c02`): table-level tenant-scope study.
  `ENG-019`'s ratchet asks only whether the string `tenant_id` appears in a raw
  statement, so a comment, an `ORDER BY`, or a join equality between two
  unscoped tables satisfies it without bounding a row. The prototype
  `internal/tenancy/policycheck` (analysis only, imported by nothing) extracts
  each Go statement's literal SQL, restores the clause keyword GORM strips,
  lexes it, and scopes every relation either through a constant tenant
  predicate or through a fixpoint over equality edges. Over the corpus: 11 raw
  statements, 5 analyzable and all 5 fully scoped (including the three-table
  `noc_impact` chain), 6 opaque (3 runtime SQL, 2 `Model(&X{})` driving tables,
  1 nested `FROM` already exempted). Decision: keep the string ratchet as the
  enforced gate; revisit enforcement when the analyzable share grows. Limits:
  no subquery/CTE descent, no model-to-table mapping, non-literal `Where`
  fragments are skipped, bare `tenant_id` is ambiguous in joins. Acceptance:
  `go test ./internal/tenancy/policycheck/` passes its fixture matrix and the
  corpus survey, `golangci-lint run ./internal/tenancy/policycheck/...` reports
  0 issues. Evidence:
  [tenant-scope-policy-checker-2026-10-29.md](tenant-scope-policy-checker-2026-10-29.md).

- `ENG-060` (2026-10-29, `dev` @ `e3e5e57`): redacted support bundle. The log
  export already redacted its output, but an operator with an incident could
  still copy raw `mwx-isp.yml` and the raw log by hand, bypassing that pass. The
  rules now live in `internal/redact` and both `/admin/loki/query` and the new
  `cmd/support-bundle` call them, so the two cannot drift. Staging and exporting
  are separate commands: `Stage` writes `config.redacted.yaml`, the log tail
  (2 MiB, line-aligned), a redaction report with per-class counts and the dotted
  keys that were redacted, `meta.json`, and a manifest with a sha256 per file
  plus a bundle checksum, all under `pending_review`. `Approve` is the only path
  to a `.tar.gz`: it requires a reviewer, re-hashes every staged file, rejects a
  bundle edited after staging (`ErrChecksumMismatch`), refuses a second approval,
  and writes a `sha256sum` sidecar. Secrets become `[REDACTED]`, customer
  identifiers a stable `sha256` pseudonym, and addresses keep only their /24 or
  /48 prefix; free text is swept for credential assignments, URL userinfo,
  e-mails, and IPv4 literals. The tool opens no socket, reads no database, and
  writes only under `-out`. Acceptance: `go test ./internal/redact/
  ./internal/supportbundle/ ./cmd/support-bundle/` and the existing
  `adminapi` redaction tests pass, `go build ./...` passes, `golangci-lint run
  ./internal/redact/... ./internal/supportbundle/... ./cmd/support-bundle/...`
  reports 0 issues, and a staged→approved demo run produces a six-entry archive.
  Limits: phone numbers, postal addresses, and IPv6 only by key name, config
  comments/formatting are lost on re-encode, the sidecar is not a signature, and
  the tool is not wired into the admin UI. Evidence:
  [support-bundle-2026-10-29.md](support-bundle-2026-10-29.md).

- `ENG-034` (2026-10-29, `dev` @ `9bc7368`): operator payment correction. No
  correction path existed: `billing.RecordPayment` is append-only and the
  portal payment endpoints are fail-closed (503/410/503), so a mis-entered
  payment could only be fixed by editing money rows by hand. `ReversePayment`
  now keeps the payment number, amount and timestamp, marks the row `reversed`,
  restores exactly that amount on the invoice, and derives the invoice status
  from the restored balance (paid→issued, paid→overdue when the due date
  passed, paid→partial when other payments remain, partial→issued). Void
  invoices (`409 INVOICE_VOID`), non-received payments
  (`409 PAYMENT_NOT_REVERSIBLE`), and payments that disagree with the invoice
  ledger are rejected before any write. The payment and invoice are locked
  `FOR UPDATE` in one transaction with a `payment_reversed` billing event and a
  `sys_opr_logs` row carrying the operator, client IP and a trimmed 8–200
  character reason (`400 INVALID_CORRECTION_REASON`), so a repeat call cannot
  duplicate the trail. The route is admin-only
  (`POST /isp/payments/:id/reverse`, operator gets 403) and a cross-tenant
  payment id returns 404 through the tenant-scoped `GetDB(c)`. The customer
  timeline now summarizes a reversed payment as `(reversed)` with warning
  severity instead of reporting it as settled, and the console shows the
  action only to admins on received payments. Acceptance is the
  paid/partial/void matrix with a role check and a correction trail: `go test
  ./internal/billing/` and `go test ./internal/adminapi/ -run
  'TestCustomerTimeline|TestPaymentReversal'` pass, `golangci-lint run
  ./internal/billing/... ./internal/adminapi/...` reports only the four
  pre-existing `auth_session_test.go` gosec findings, and `npm run type-check`
  plus `npm run build` pass. Limits: no Playwright coverage of the new dialog,
  and no concurrent payment-vs-reversal race test beyond the shared row locks.
  Evidence: [payment-correction-policy-2026-10-29.md](payment-correction-policy-2026-10-29.md).

- `ENG-026` (2026-10-29, `dev` @ `33c0f5e`): backup KDF and envelope
  evaluation. Measured PBKDF2-SHA256 at the counts that bound the policy on
  go1.27.0 (Ryzen 5 7500F): 310k = 41.7 ms, 600k = 79.5 ms, 1.2M = 161.6 ms,
  2.4M = 355.3 ms, and a full 600k archive open = 87.1 ms, so the 600k written
  cost is kept. The inspected decrypt path compared the recorded iteration
  count to the compiled constant for exact equality; raising the cost would
  have rejected every archive written by an earlier release with no format
  change to signal it. Restore now accepts a bounded recorded range
  (600,000–2,400,000) and derives with the manifest value, the minimum blocks a
  downgrade and the maximum bounds CPU before authentication, and fixture tests
  restore archives at 600k, 600,001 and 2.4M while rejecting 0, 599,999,
  2,400,001 and 2^30 without writing rows. New encrypted exports require a
  12-character passphrase (`BACKUP_PASSPHRASE_TOO_SHORT`); restoring stays
  length-agnostic, so an archive written earlier with a short passphrase still
  restores. Container format and version are unchanged. `go test
  ./internal/adminapi/ ./config/` and `golangci-lint run ./internal/adminapi/
  ./config/` pass (only the pre-existing auth_session_test.go gosec findings
  remain). Limit: the benchmark ran on a desktop, not on a claimed VPS host, so
  a cost bump needs a rerun on deployment hardware. Evidence:
  [backup-kdf-evaluation-2026-10-29.md](backup-kdf-evaluation-2026-10-29.md).

- `ENG-036` (2026-10-29, `dev` @ `39caa5f`): per-tenant FUP scheduler trail.
  #46's per-tenant loop already continues after a tenant error, but nothing
  failed if later scheduler work made one tenant's failure abort the run, and
  the scheduler exported no telemetry. The fixture
  `TestSchedFUPCheckIsolatesTenantFailuresWithBoundedTelemetry` seeds four
  tenants, injects a GORM query failure for the first two through a test-only
  callback, and asserts the two queued after them are still throttled in the
  same run while the failing tenants are untouched. The scheduler now
  increments two counters (`fup_scheduler_tenant_attempts_total`,
  `fup_scheduler_tenant_failures_total`) whose label set is fixed — no tenant
  or subscriber dimension — and the test pins that set, so metric cardinality
  does not grow with the tenant count. Failure alerts keep one dedupe key per
  job per minute, so two failing tenants produce one outbox row, not one per
  tenant. `go test ./internal/app/` and `golangci-lint run ./internal/app/`
  pass. Limit: the fixture injects the failure at the query layer; a panicking
  tenant is handled by the job-level recover instead.

- `ENG-062` (2026-10-29, `dev` @ `5173519`): dependency upgrade ratchet. The
  acceptance asked for a script that fails on a genuinely applicable advisory
  and still offers a reasoned exception path. `tools/depratchet` reads the
  `govulncheck -json` stream and separates symbol-level findings — the
  vulnerable function is called — from package/module findings, where the
  dependency is present but no vulnerable symbol is reachable; only the former
  fail. Against the 2026-10-29 scan (v1.9.0, DB 2026-10-27, go1.27.0) it exits 1
  with 11 called advisories and 6 leads when the register is empty, and exits 0
  once each called advisory has an entry. The register
  (`docs/security-exceptions.json`) is machine-checked: reason of at least a
  sentence, ticket and expiry are required; duplicates, unknown fields,
  expired entries and entries for non-called advisories all fail. The eleven
  entries mirror the inventory's "toolchain" status — fixed by Go 1.27.2 or
  1.27.3 — and expire 2026-11-30 so the toolchain bump cannot slip silently.
  Unit tests cover the pass, fail and malformed-register paths.
  `docs/dependency-upgrade-policy.md` adds the Go/npm cadence, PR risk tiers
  T0–T3 and exception lifetimes. No CI lane is added: the gate runs at release
  and on dependency PRs. Limits: `npm audit` is not ingested (package presence
  without call paths), govulncheck traces remain conservative, and the scan has
  to run under the shipping toolchain before release claims are made from it.

- `ENG-003` (2026-10-29, `dev` @ `d463ec4`): EAP-TLS certificate revocation
  policy. The premise the item asked to test is confirmed: the engine verifies
  the client chain with `RequireAndVerifyClientCert` and a `ClientCAs` pool, and
  Go's TLS stack never evaluates CRLs or OCSP, so a **revoked client
  certificate is accepted until it expires** — the same holds for RadSec
  (`VerifyClientCertIfGiven`). `docs/eap-tls-revocation-policy.md` records the
  threat model (leaked credential versus compromised CA), coverage per identity
  (EAP-TLS chain yes, PEAP/TTLS n/a, RadSec out of scope, server cert out of
  scope), the CRL-over-OCSP decision (no network round trip on the RADIUS hot
  path), the freshness rule (`ThisUpdate`..`NextUpdate`, no grace), and the
  fail-mode table: stale / unsigned / not-yet-valid CRL rejects the handshake,
  an issuer without a CRL is accepted and that is the load-bearing coverage
  limit. The proof of concept adds one optional `Config.VerifyConnection` hook
  to the engine (nil keeps current behaviour) and exercises four real
  handshakes in `revocation_poc_test.go`: the current engine accepts a revoked
  certificate, the proposed verifier rejects it, a stale CRL fails closed, and
  an issuer without a CRL is accepted. `go build ./...`,
  `go test ./internal/radiusd/plugins/eap/...` and
  `golangci-lint run ./internal/radiusd/plugins/eap/tlsengine/` pass
  (`docs/feature-checklist.md` TR-F004 updated). **Not wired into production**
  and not claimed as deployed: enabling it needs CRL material in the managed
  certificate store, a config schema entry, a refresh policy and UI. Limits:
  no OCSP, no RadSec wiring, no cross-signed/delegated-CRL coverage, and no
  per-handshake benchmark.

- `ENG-059` (2026-10-29, `dev` @ `06e7ae4`): RADIUS user import row errors and
  rollback. The importer wrote row by row with no transaction, so a file with
  four bad rows imported the good one and left it committed — the item's own
  acceptance test asserted `success = 1`, i.e. the defect was encoded as
  expected behaviour. It now validates every row first (`planUserImport`, no
  writes) and inserts a fully valid payload in one transaction, so a bad file
  is rejected with per-row errors and the user table is untouched. Three more
  holes closed while validating: `status` was free text and the auth checker
  only rejects `disabled`, so `nonaktif` or `active ` imported a
  **login-capable** user (now an `enabled`/`disabled` allowlist); in-file
  duplicate usernames were caught only because the first insert had already
  committed (now an explicit `seen` map); and a non-numeric `profile_id` was
  reported as "profile required" while a non-numeric `node_id` was silently
  coerced to 0 (both now rejected as invalid). The DB error text no longer
  reaches the client on the commit path. The UI shows the rejection with the
  first row error instead of "Import complete: 0 succeeded". A dry-run preview
  is designed in `docs/radius-user-import-audit-2026-10-29.md` but **not
  deployed**, because the acceptance criterion forbids shipping it without a
  checklist decision. Tests: the error case now asserts zero writes and that
  the valid row was not created; new cases cover in-file duplicates, unknown
  status, non-numeric profile/node ids, and a disabled-status happy path.
  `go test ./internal/adminapi/ -run TestImportRadiusUsers` (4.0 s), `go vet`,
  `npm run type-check`, `npm run build` pass. Limits: no forced mid-transaction
  failure in SQLite (atomicity rests on GORM rollback plus the
  `(tenant_id, username)` unique index); Excel path not re-exercised with a
  binary fixture.

- `ENG-057` (2026-10-29, `dev` @ `e308244`): dashboard chart timezone and
  nominal audit. The 7-day authentication trend and 24-hour traffic series were
  grouped by SQL `DATE()` / `TO_CHAR(date_trunc('hour', …))` on Postgres and
  `strftime(…, 'localtime')` on SQLite — i.e. in the **database session
  timezone** — while the response labels and the "today" boundary were computed
  from Go `time.Local`. The comment on the dashboard test admitted the gap
  ("due to timezone differences …"), which is the defect. With the DSN
  hardcoding `TimeZone=Asia/Jakarta` and `system.location` configured
  separately, a session starting at 00:30 local could be filed under the
  previous day by the SQL, miss the label-keyed lookup and silently become a
  zero. Both series now build half-open instant windows in the configured
  location (`dayWindows`, `hourWindows`) and fold all windows into one
  `COUNT/SUM(CASE WHEN acct_start_time >= ? AND acct_start_time < ? …)`
  statement, so assignment depends only on the stored instant and the query
  count does not grow; `dateBucketExpression` is gone. On the client, every
  formatter was built without a locale (`Intl.DateTimeFormat(undefined, …)`,
  `Intl.NumberFormat()`), the outstanding amount was hardcoded to `en-US`, the
  day/hour labels used `hour12: false` (which can render midnight as `24:00`),
  and the chart tooltips printed raw floats; all now follow React-Admin
  `useLocale()` with `hourCycle: 'h23'`, plus `valueFormatter` on both charts.
  Tests pin the fix deterministically in a fixed `UTC+7` zone
  (`TestDayWindowsUseLocalCalendarDays`,
  `TestCountSessionsPerWindowBucketsByInstant`) and the stats test now requires
  the record in the last day bucket with exactly one non-zero traffic bucket.
  `go test ./internal/adminapi/` (45.5 s), `npm run type-check` and
  `npm run build` pass. Limits recorded in
  `docs/dashboard-timezone-nominal-audit-2026-10-29.md`: the Postgres
  instant-comparison path is argued, not matrix-tested; a distroless deploy
  without zoneinfo still logs a `timezone config error` and runs in UTC (the
  shipped image installs `tzdata`); other admin pages may still use
  browser-locale formatting.

- `ENG-054` (2026-10-29, `dev` @ `ef3c286`): ID/EN reconciliation of billing and
  incident status tags. `StatusChip` rendered the raw enum uppercased, so every
  customer, package, subscription, invoice and payment status stayed English
  under `id-ID`, and the billing dropdowns hardcoded English `name` strings;
  ticket chips carried their own per-page copy of the same labels. The values
  now live in one vocabulary, `web/src/i18n/status.ts` (billing lifecycle,
  ticket lifecycle, priorities, connectivity), rendered through a `status.*`
  namespace in `en-US.ts` and `indonesian-overrides.ts`. Both dictionaries are
  typed as a complete `StatusMessages` map, so `tsc` — which `npm run build`
  runs in CI — fails when a value lacks an Indonesian label instead of silently
  falling back to English. `StatusChip` translates through `useTranslate` with
  the uppercase enum as its default, `useStatusChoices` feeds the
  customer/package/subscription/invoice `SelectInput` filters and forms, and
  the ticket table chip now uses the shared catalog (same wording as the
  previous page map: Terbuka, Terjadwal, Dalam Proses, Selesai, Tertutup).
  `npm run type-check` and `npm run build` pass. Limit: this verifies the
  catalogs compile, not how the operator sees them; the local ESLint run is
  broken by the repo's TypeScript 7 / `@typescript-eslint` 7 mismatch
  (pre-existing, CI gates the web build instead).

- `ENG-040` (2026-10-29, `dev` @ `925aad0`): accounting retention and query-plan
  study. `tools/acctbench` seeds synthetic session volume into a scratch
  PostgreSQL database, runs the statements behind `ListAccounting` and the daily
  retention delete, and records `EXPLAIN (ANALYZE, BUFFERS)` plus p50/p95, so an
  index or partitioning decision rests on plans. At 1M and 5M rows the default
  list page stays at 0.05 ms (eight buffers), page 201 at 1.5 ms, and the
  retention cutoff at 0.1 ms on `acct_stop_time`; the heavy statement is the
  7-day count, 6 ms at 1M and 42 ms at 5M rows where one tenant holds 625k rows.
  Two candidates were measured and rejected: `(tenant_id, acct_start_time DESC)`
  is never chosen (1.00–1.08x for +8.3% size) and `pg_trgm` on `username` is
  3.7x slower at 1M rows and equal at 5M for +250 MB, because common trigrams
  scan as many index pages as the tenant's heap. Decision: no index, no
  partitioning, and no archive pipeline — the window stays configurable through
  `radius.AccountingHistoryDays` (0 disables) and backups, not a second table,
  cover deleted history; revisit triggers are recorded in
  `docs/accounting-retention-query-plan.md`. One measurement pitfall is
  documented too: pgx caches one prepared plan per SQL text per connection, so a
  reused pool hides the effect of a new index.

- `ENG-016` (2026-10-29, `dev` @ `985908e`): periodic fork/upstream check of the
  RADIUS codec library. The build resolves `layeh.com/radius` through the
  `talkincode` fork, so a protocol-level fix can exist in one chain and not the
  other with nothing local to show it. Upstream `master` is still
  `1006025d24f8` (2023-12-13) — exactly the commit the `require` pseudo-version
  names, so there is no upstream delta to take — while fork `v0.1.0`
  (`8f214fbf5382`, 2026-06-08) equals fork `master` and is two commits ahead:
  the generated `_SetVendor` panic/wrong-delete-index fix (reachable from a
  repeated vendor VSA on the wire) and the `IPv6Prefix` padding-bit tolerance
  (RFC 3162 requires zero padding, real equipment ignores it). Both are already
  used here — the repo's own generated dictionaries carry the corrected
  template and `setvendor_test.go` pins it, but the prefix tolerance had no
  local test, so `radiuslib_contract_test.go` now pins a `/64` with host bits
  and a `/61` with a trailing bit while still rejecting out-of-range lengths
  and length-less attributes; reverting the `replace` to upstream fails the
  test. `docs/radius-fork-upstream-audit-2026-10-29.md` records the SHAs, the
  reproduce commands, a relevance matrix, the sync/defer/no-change criteria and
  this run's **no change** decision (upstream dormant since 2023, 57 open
  issues, issue #119 never merged). `go mod tidy -diff` empty, `go mod verify`
  clean.

- `ENG-063` (2026-10-29, `dev` @ `9ea95b8`): API response contract, pagination
  and error testing. The collection endpoints the React Admin dataProvider maps
  were exercised against the documented grammar and three real breaks surfaced.
  Sessions, accounting and profiles ignored `pageSize` entirely (only `perPage`
  was parsed) and reset an oversized `perPage` to 10 instead of clamping it to
  the cap; they now share `parsePaginationWith` with a 10/100 compact profile
  while the rest of the API keeps 20/1000. `paged()` passed a nil slice straight
  to the encoder, so a page past the end (or any empty result) returned
  `"data": null` — a payload React Admin calls array methods on; the envelope now
  normalizes nil slices to `[]`. The users list counted through
  `Session{NewDB:true}`, which dropped the filters, so `meta.total` reported the
  unfiltered row count for every filtered user search; the count now runs on an
  independently built filtered query. `api_contract_test.go` seeds five rows per
  model and pins the success/error envelopes, pageSize-over-perPage precedence,
  per-endpoint defaults and caps, empty pages as `[]`, filter totals, sort
  injection falling back to the documented order without leaking passwords, and
  the 401/403/400/404 and validation mappings; a contract lint parses
  `resourcePathMap` and fails when a mapped UI resource has no case. The
  contract is documented in `docs/api-contract.md`.

- `ENG-039` (2026-10-29, `dev` @ `ca7afdc`): demo-marker protection during
  clean-up. Cleanup was exercised against real records parked next to sample
  data and one data-loss path surfaced: a sample subscription reparented to a
  real customer was still found through its marker event under the old sample
  customer, its sample invoices were deleted first, and the subscription then
  disappeared with them — and the next seed run would have reused that marker
  for a sample customer. The loop now loads the subscription, treats a
  non-sample owner as adoption (keep the subscription, drop only the seed link)
  and deletes the marker event even when the subscription survives.
  `cleanup_protection_test.go` adds three fixtures: an operator record set that
  points at sample inventory (a real user keeps its sample profile and node, a
  real subscription keeps its sample package, a sample customer with a real
  invoice keeps both) and reads back recovered credentials unchanged; the
  adopted-subscription case including regeneration on the next seed run; and a
  two-tenant inventory where cleaning tenant A removes its unreferenced node,
  NAS, profile, user and accounting rows while tenant B and both tenants' real
  credentials stay untouched.

- `ENG-029` (2026-10-29, `dev` @ `4af4425`): PNG branding upload validation.
  The upload path was exercised against the four failure shapes the issue named —
  content-type mismatch, an oversized file, a pixel bomb, and unusual PNG
  encodings — and one real gap fell out: `readBrandLogo` used
  `png.DecodeConfig`, which stops after the IHDR chunk, so a 33-byte file
  claiming a legal 8×8 canvas was accepted, stored, and served from
  `/api/v1/public/branding/logo` as an image no browser can render. The reader
  now decodes the whole stream after the dimension check; the 1 MiB file cap and
  the 2048×2048 cap bound that work, so a crafted header cannot turn it into a
  decompression bomb. `branding_upload_test.go` pins the rest: JPEG/SVG/HTML/gzip
  bytes and empty files are refused whatever filename or part content type they
  declare; a 1 MiB+1 file and a 2 MiB body are refused; IHDR-only bombs at
  2049×8, 8×2049, 4096×4096, 100000×100000 and the uint32 maximum are refused by
  the dimension policy; grayscale, paletted, RGBA and 16-bit NRGBA PNGs are
  accepted and served byte-identical as `image/png` with `nosniff`. Ownership is
  pinned too: a tenant-scoped request still writes the single global branding
  row, every public read sees it, and a tenant administrator is refused by
  `requirePlatformAdmin`. A valid PNG declared as `image/svg+xml` is stored
  deliberately — the bytes decide, and the response type never follows the
  client's label.

- `ENG-008` (2026-10-29, `dev` @ `5b934ae`): EAP-OTP cannot be enabled without
  a validator. The handler already rejected every response after the hardcoded
  `123456` accept was removed, but the surrounding configuration could still
  route traffic at it. Four independent guards are now pinned by tests: the
  handler answers `ErrOTPNotConfigured` and is never registered, `eap-otp` is
  absent from the `radius.EapMethod` enum so the settings API refuses it, and
  `buildAllowedEAPHandlers` filters `radius.EapEnabledHandlers` against
  `registry.EAPHandlerNames()` — an unknown name is logged and dropped, and if
  nothing usable remains the allowlist becomes the reserved
  `eap.MethodUnavailable`, which the coordinator rejects. That last change closes
  the silent-widening hole where a misspelled allowlist fell back to
  `eap-md5`. `docs/eap-otp-contract.md` states the seven properties a real OTP
  backend must satisfy (issuance bound to State and identity, single use,
  expiry, constant-time compare, attempt limit, out-of-band delivery, no silent
  fallback) and the checklist to follow before the method can be registered.

- `ENG-027` (2026-10-28, `dev` @ `82bc7be`): network diagnostic SSRF hardening.
  The diagnostics ping endpoint already refused hostnames and required private
  targets to be registered, but the scheduled monitor targets did not: they
  accepted loopback, so an admin-authored target could point the poller at the
  daemon's own listeners, and the HTTP probe followed up to five redirects,
  whose `Location` header is an address no operator validated — a monitored
  device could pivot the probe anywhere the host can reach. `CheckTargetAddress`
  now holds one policy (explicit IP literal only, so no DNS step exists to
  rebind; no unspecified, "this network", broadcast, loopback, link-local, or
  multicast destination; private IPv4 and IPv6 unique-local still allowed
  because that is where an ISP's devices live). The API applies it on create and
  update, and `Monitor.checkTarget` re-applies it at egress so a row written
  before the rule is refused — logged, no fabricated down sample, no incident.
  The HTTP probe stops at the first response with `http.ErrUseLastResponse`,
  keeping the existing "3xx counts as reachable" result. The policy doc gained
  the monitor-target section, a "not a scanner" section (one address, one probe
  type, one port, 500-target cap, eight concurrent polls), and the tests:
  address table, egress refusal for a stored loopback row, redirect pin, and a
  handler test for loopback/metadata/hostname/private. The previous deliberate
  "loopback stays available for a local service" allowance is reversed and
  documented; a local service remains reachable through the host's LAN address.

- `ENG-030` (2026-10-28, `dev` @ `bf4b18e`): SBOM and license inventory.
  `tools/sbom` writes CycloneDX 1.5 for the Go module graph (`go list -m all`,
  plus `go mod download -json` for the modules `go list` leaves unextracted) and
  for the frontend's production dependencies (`npm sbom --omit=dev`), then
  removes every field npm and Go regenerate per run: no timestamp, no random
  serial number, no tool version, no module-cache path, all lists sorted. The
  result is 134 Go and 150 npm components in `sbom/`, and `-check` fails only
  when the dependency set really changed. Licenses are fingerprinted from the
  module's own text, with MPL and the GPL family matched against the header only
  because MPL 2.0 §3.3 names the AGPL in its body; one npm package
  (`css-mediaquery`) ships no license field and stays `NOASSERTION` as a review
  gap. `docs/dependency-inventory.md` records the license summary, the copyleft
  review items — `go.mau.fi/libsignal` is GPL-3.0-only and reachable through
  whatsmeow, and the `talkincode/radius` fork is MPL-2.0, not MIT — and a
  17-row advisory register that marks each govulncheck hit as
  investigated/not-affected with its reason. The 11 reachable findings are all
  stdlib issues fixed in go1.27.2, which also exposed a CI mismatch: the workflow
  pins `GO_VERSION: 1.26` while the local toolchain is go1.27.0, so CI compiles
  against a different stdlib than the one checked here.

- `ENG-009` (2026-10-28, `dev` @ `3e05648`): RadSec peer-verification regression.
  The negative matrix found a real defect: `Serve` reported a rejected TLS
  handshake as a recoverable packet-parse error and re-entered the read loop, so
  a peer with an untrusted certificate left the goroutine spinning on a dead
  connection (CPU burn, connection never released, `Shutdown` blocked). `Serve`
  now completes the handshake explicitly before any per-peer lookup, returns on
  any transport error, and skips a frame only when its attributes were fully
  consumed (`errMalformedPacketAttribute`). `radsec_handshake_matrix_test.go`
  runs the real server behind `initTLSConfig` and pins the policy: trusted client
  accepted, untrusted/expired/wrong-EKU/TLS 1.0-1.1 peers rejected with zero
  packets handled, failed handshake releases the connection, and the snapshot
  keeps `VerifyClientCertIfGiven`, the TLS 1.2 floor and the CA pool. The policy
  is documented and its known limit stated: a certificate-less peer is still
  accepted, and a missing CA bundle now logs a warning at load time.

- `ENG-004` (2026-10-28, `dev` @ `3cf307c`): RadSec/EAP certificate rotation.
  RadSec parsed its key pair and client CA bundle once, so any rotation forced a
  restart that dropped live TLS connections. `internal/radiusd/radsec_tls.go` now
  stats the three configured files on each handshake and swaps in a validated
  snapshot when one changed; startup stays fail-closed (expired or unparsable
  material refuses to listen) and a failed reload keeps the last known-good
  snapshot and logs the reason. `radsec_rotation_test.go` drives real TLS
  handshakes through a running listener: serial 1001 → 1002 after replacement,
  1001 again after rollback, old+new client CA accepted during an overlap bundle,
  invalid/expired replacements kept out, and concurrent handshakes during a
  bundle rewrite. EAP already resolved material per handshake; the new
  `TestSettingsTLSConfigProvider_RotatesBetweenHandshakes` proves the 101 → 202 →
  101 switch. `docs/radsec-certificate-rotation.md` holds the procedure, rollback
  and the log line to observe.

- `ENG-061` (2026-10-27, `dev` @ `521963e`): repeatable test runs. Repeated
  `go test -count=N` runs failed on the second iteration because five files opened
  fixed-name in-memory SQLite DSNs and the shared cache kept the previous rows alive.
  Every such DSN now carries a per-open unique suffix (14 in total), and
  `go test -count=3 ./internal/radiusd ./internal/app ./internal/tenancy ./internal/adminapi`
  passes (93.9s / 8.5s / 1.3s / 135.6s). The browser suite failed its first `goto`
  against a cold Vite server, so `navigationTimeout` is 60s and an action timeout of
  15s is explicit; `npx playwright test --project=chromium --repeat-each=3` is 75/75
  in 1.4 min. `docs/e2e-stability-2026-10-27.md` records the baseline, the isolation
  model, how failure artifacts are read, and the limits (only Chromium is installed;
  no E2E CI lane by owner preference).

- `ENG-053` (2026-10-27, `dev` @ `4404b1b`): keyboard path through the critical
  grids and the payment dialog. The audit found a real gap: customer, invoice and NAS rows
  were pointer-only because React Admin's `rowClick` listens for clicks and the data cells
  hold no focusable element. Each list now has a per-row `ShowButton` with a
  record-specific accessible name ("Open CUST-0001"), and `web/e2e/keyboard-navigation.spec.ts`
  tabs to it, asserts `:focus-visible`, and opens the record with Enter. The invoice
  payment dialog already met the criterion: focus moves inside, Tab/Shift+Tab stay trapped,
  Escape closes, and focus returns to "Record Payment". All 25 chromium e2e tests pass
  (33.3s); TR-F013 and the three surface rows record the new evidence.

- `ENG-056` (2026-10-27, `dev` @ `a171299`): honest affordances for staff-gated
  portal actions. `/home` bill lookup, voucher check and registration submit, plus the
  `/portal` lookup, now answer a 401 with one informational `StaffLoginNotice` carrying
  "Login staf" and "Hubungi WhatsApp" actions; the red alert is reserved for real lookup
  failures, and the raw `Unauthorized` body is never shown. The `/portal` offline payment
  fallback opens the branded WhatsApp number instead of an empty chat picker.
  `web/e2e/anonymous-actions.spec.ts` covers the four anonymous actions, the login
  navigation and the branded WhatsApp target (5/5 chromium, 18.4s). No API was widened:
  `TestLandingSurfaceEndpointsFailClosedAnonymously` still passes, and
  `docs/public-endpoint-hardening.md` records the presentation rule.

- `ENG-038` (2026-10-27, `dev` @ `7c2e06e`): voucher terminal-state concurrency. The
  radiusd tests drive `activateHotspotVoucher` from eight goroutines against expired and
  revoked rows: every caller gets an expiry in the past, the status stays terminal, no
  first-login timestamp appears, and the credential expiry does not move. A deterministic
  login → revoke → login interleaving keeps the revoked status, the original timestamps,
  and the disabled credential. The adminapi tests fire the real revoke handler from eight
  goroutines: exactly one response performs the revoke, all responses are 200, and retries
  replay `already_revoked`. A `used` voucher whose window already passed stays revoked
  through the lazy expiry marker. The credential fallback matches username and password,
  so a shared password on another username and a rotated credential are both left
  enabled. `docs/voucher-lifecycle.md` lists the new commands; TR-N006/A20 records the
  coverage. Full radiusd and adminapi suites pass (40.2s / 47.4s).

- `ENG-035` (2026-10-27, `dev` @ `c070535`): subscription lifecycle state machine and
  rollback evidence. `internal/adminapi/isp_subscription_state_machine_test.go` drives all 16
  pending/active/suspended/terminated × activate/suspend/reactivate/terminate pairs through the
  real handler: accepted transitions must leave subscription and linked account consistent
  (`active` ⇔ `enabled`) and append exactly one typed billing event; rejected pairs return
  `409 INVALID_SUBSCRIPTION_STATE` with no subscription, account, or event change, and an
  unknown action returns `404 UNKNOWN_ACTION`. SQLite triggers inject a failure at the account
  update and at the billing-event insert, proving the transaction rolls back both earlier
  writes on a 500. Relink coverage rejects an already-linked, unknown, or non-numeric RADIUS
  account without creating rows, and a trigger on the subscription insert proves the account
  created earlier in the same transaction is rolled back. The TR-F027 boundary links the
  tests; the full adminapi suite passes in 43.8s.

- `ENG-052` (2026-10-27, `dev` @ `447c0ad`): browser role matrix for the operator/admin/super
  button set. `web/e2e/role-matrix.spec.ts` seeds operator, admin, super and `platform_admin`
  sessions into `localStorage` and, against read-only API fixtures, asserts menu visibility
  (network/operations/operators vs. platform-only branding, config, certificates, tenants) plus
  customer and RADIUS-user create/import controls, voucher print (all roles) vs. batch generation
  (admin), and ticket creation (all roles) vs. WhatsApp dispatch (admin). All 16 cases pass on
  Chromium. The run exposed a real gating leak: React Admin's default empty state renders a
  Create button from the resource's create route, so operators saw an empty-state "Create" link
  on customers/packages/subscriptions/payments and the RADIUS-user empty hint told them to
  create. Lists now pass `empty={<RoleAwareEmpty />}`, and the RADIUS empty hint has a
  restricted variant; both follow `useIsAdmin`. The server half is unchanged and still enforced
  by `TestMutationRouteRoleMatrix`; `docs/access-control-matrix.md` records the UI/API split and
  the Playwright command.

- `ENG-011` (2026-10-27, `dev` @ `6fcd5ad`): fuzz regression for defective and oversized
  RADIUS packets. A deterministic corpus of 22 named frames covers the RFC 2865 §3 length
  window (19/0/3/4097/65535 rejected, exactly 4096 accepted), truncated headers and bodies,
  attribute TLVs with length 0/1/overrun, trailing padding, malformed and valid Huawei VSAs,
  and short/duplicate/zeroed Message-Authenticator values. Three native fuzz targets
  (`FuzzParseTcpPacket`, `FuzzParseAttributesAndVendorDecoding`, `FuzzMessageAuthenticator`)
  assert no panic, framing errors only outside 20..4096, exact frame consumption, and a
  deterministic replay. A 25s/20s/20s run executed 38,362 + 78,072 + 65,807 inputs with zero
  panics and zero contract violations; the commands, invariants and limits are in
  `docs/radius-fuzz-harness.md`, and the TR-F001 boundary now links the harness.

- `ENG-024` (2026-10-27, `dev` @ `db08c5f`): rotating `web.secret` and the sealed SNMP
  credentials. The monitor stores the SNMPv2c community and SNMPv3 auth/privacy passwords as
  AES-GCM ciphertext keyed from `web.secret`, so a secret change made every target report "SNMP
  credentials unavailable". `networkmonitor.RotateSNMPSecrets` now decrypts and re-seals all
  three columns for every tenant inside one transaction; any value that cannot be authenticated
  with the old secret aborts the rotation and leaves every row unchanged. The new
  `cmd/rotate-secrets` reads both secrets from the environment, dry-runs by default by running
  the updates and rolling them back, and prints counts only. Tests cover the dry-run rollback,
  all three columns, the atomic abort (first row rewritten, second fails, nothing persists), the
  empty/identical-secret guards, and CLI stdout/stderr containing neither plaintext nor either
  secret. `docs/secret-rotation-runbook.md` documents backup, cutover, reverse rotation and the
  snapshot trap: ciphertext in a pre-rotation backup stays bound to the old secret after restore.
  The backup/blueprint/checklist docs now link the procedure.

- `ENG-023` (2026-10-27, `dev` @ `818c331`): rate limiting behind a multi-hop reverse proxy.
  Echo's `X-Forwarded-For` extractor trusts loopback, link-local and private peers implicitly,
  so once `TOUGHRADIUS_TRUSTED_PROXIES` was set, a caller reaching the process from a LAN address
  could rotate the header and get a fresh bucket per attempt; the per-IP login limiter was
  bypassable. `trustedProxyIPExtractor` now disables link-local and private implicit trust and
  keeps only loopback, so a same-host proxy still works without configuration. New unit tests in
  `internal/webserver/server_test.go` pin the trust defaults, the walk through listed private
  proxies, an unlisted hop resolving to that hop (fail-closed, shared budget), and a malformed
  header resolving to the peer; `internal/adminapi/proxy_ratelimit_test.go` drives the real
  router and shows the spoof attempt reaching the handler and being throttled on the sixth
  attempt, while a listed proxy keeps per-client budgets. Both the unit and end-to-end tests fail
  against the previous extractor. Tenant/IP isolation was already pinned by
  `TestPublicRateLimitBucketsByTenantAndIP`. `docs/public-endpoint-hardening.md` now states that
  every hop, private ones included, must be listed.

- `ENG-022` (2026-10-27, `dev` @ `e093bd0`): admin browser-session threat model. The review
  mapped login/token, browser storage, logout, CORS and CSRF against the real code. The admin
  API is a 12h HS256 bearer token kept in `localStorage` and sent as an `Authorization` header;
  there are no cookies, no refresh endpoint and no server-side logout. That makes classic CSRF
  inapplicable rather than "exploitable", which the acceptance criterion asked for explicitly.
  `internal/adminapi/auth_session_test.go` drives the real router with forged, expired,
  `alg:none`, tampered, cookie-only and wrong-scheme credentials (all `401`), proves the tenant
  comes from the signed claim despite `X-Tenant-ID`/query overrides, pins the missing
  refresh/logout routes, proves login never sets a cookie, and ratchets the frontend so
  `document.cookie`/`withCredentials` cannot appear. Limits are documented in
  `docs/admin-session-threat-model.md`: a stolen token stays valid up to 12h unless the
  operator/membership/tenant is revoked, `localStorage` is XSS-readable, and no CSP header is
  sent. No production code changed.

- `ENG-012` (2026-10-27, `dev` @ `9be97f3`): accounting Start/Interim/Stop out-of-order. The
  handlers already converged for duplicates and a delayed interim, but `UpdateStop` matched
  every accounting row with the `Acct-Session-Id`, so a NAS that reused an ID for a later
  session had the earlier session's finalized counters overwritten by the second stop. The
  repository now selects the newest row and updates it by primary key. The new
  `internal/radiusd/accounting_order_test.go` fixture drives the real handlers on GORM:
  `TestAccountingOutOfOrderStartInterimStopConverge` permutes duplicate Start/Interim/Stop plus
  a delayed interim and ends with no online row and one row holding the stop counters;
  `TestAccountingSessionIDReuseKeepsFinalizedHistory` keeps both reused sessions' rows intact
  (it fails against the previous update-every-match code); `TestAccountingSessionIDReuseIsTenantScoped`
  repeats the reuse on two tenants; `TestAccountingEventsWithoutStartDoNotInventHistory` proves
  Interim/Stop before Start create nothing. Documented in `docs/radius-retransmission.md`,
  including the limit that a Start processed after its own Stop is indistinguishable from a
  reused ID without a session-instance token.

- `ENG-010` (2026-10-27, `dev` @ `54aee93`): RADIUS retransmission and replay. The server has
  no completed-response cache, so the audit mapped what a duplicate packet actually does on
  each transport. Accounting is idempotent on `(tenant_id, acct_session_id)`:
  `TestRetransmittedAccountingPacketsHaveNoDoubleEffect` builds byte-identical Start, Interim
  and Stop packets, replays each one, and asserts one online row, one accounting row, final
  counters equal to the stop packet, and that a second tenant using the same session ID and
  username keeps its own online row. The auth duplicate window is in-process and keyed by
  tenant+username; `TestRetransmittedAccessRequestRateLimitIsTenantScoped` shows a replay in
  tenant 1 cannot throttle the same username in tenant 2. EAP state was already namespaced and
  serialized per tenant. The full matrix (per transport, cache key, side effects, RadSec
  in-flight map, the missing RFC 5080 response cache, and the one remaining duplicate effect —
  a zero-rate traffic sample from a replayed interim update) is in
  `docs/radius-retransmission.md`. No production code changed.

- `ENG-051` (2026-10-27, `dev` @ `5add7af`): website editor functional coverage. The
  "editor" is the global branding singleton: save is the publish path, reset is the only
  rollback, and the admin read is the read-only preview of the live record (there is no draft
  table). `TestBrandingRouteGuardAllowsOnlyPlatformAdmin` drives the registered routes through
  the real JWT/tenant middleware and asserts anonymous `401`, operator/admin/super-without-flag
  `403` on save, preview and reset, and success only for a super operator with `PlatformAdmin`;
  it also proves rejected saves persist nothing and reset removes the override.
  `TestBrandingPublishKeepsMarkupInert` publishes `<script>`/`onerror` payloads in the hero,
  subtitle and ticker fields and asserts the stored value is plain text, the public JSON body
  contains no raw `<script`, the decoded text round-trips unchanged, and reset returns the
  compiled defaults with no hostile text left behind.
  `TestBrandingEditorSourceRendersTextOnly` ratchets the frontend: no `.tsx` under `web/src`
  may use `dangerouslySetInnerHTML`, so branding values stay React text children. No production
  code changed; the TR-F029 coverage row in `docs/feature-checklist.md` was updated.

- `ENG-037` (2026-10-27, `dev` @ `e8da24f`): integer IDR boundaries. The invoice path
  already avoided floating point, so the remaining question was whether the edges hold.
  `TestInvoiceTotalsStayExactAtInt64Boundary` creates a `MaxInt64` package and asserts
  `Subtotal`/`Total`/`Balance`/item totals match exactly, then pays `1`, rejects a payment of
  `MaxInt64` (`ErrInvalidPayment`), and settles with `MaxInt64-1`, ending at
  `PaidAmount = MaxInt64`, `Balance = 0`, status paid without overflow.
  `TestRecordPaymentRejectsNonPositiveAmount` rejects `0`, `-1` and `MinInt64` without writing
  a payment row, and `TestSavePackagePriceBoundaries` pins the API contract: negative and
  `9223372036854775808` return `400`, `0` for a free package and `MaxInt64` both return `201`,
  and the max price round-trips without truncation. There is no manual invoice-item endpoint,
  so totals can only come from a stored package price. No production code changed; the
  invariant is documented under `docs/payment-concurrency.md`.

- `ENG-031` (2026-10-27, `dev` @ `74fa4c1`): invoice numbering under crash and concurrency.
  `NextDocumentSerial` already runs inside the document transaction, so the two required
  properties needed evidence rather than code: an aborted transaction rolls its increment back
  (retry reuses the number, no gap) and a committed invoice blocks a second number via the
  subscription/period unique index. `TestNextDocumentSerialRollsBackWithTransaction` and
  `TestGenerateMonthlyInvoicesRetryReusesRolledBackNumber` cover both on the unit path, and
  `TestPostgresInvoiceNumberingConcurrentAndRollback` races eight allocations, aborts a ninth,
  and asserts the retry receives it — this one runs in CI because SQLite has no real row locks
  (compiled locally with `-tags=integration`; skipped without `TEST_DATABASE_*`). The policy
  (per-tenant kind/period scope, no reuse after commit, what a visible gap means) is recorded in
  the new `docs/document-numbering.md`. No production code changed.

- `ENG-032` (2026-10-27, `dev` @ `5c1946a`): manual suspension ownership. The automatic
  paths already respected the invariant — `SuspendOverdueSubscriptions` only selects
  subscriptions whose status is still `active`, so it never rewrites a `manual`
  `suspension_reason` and never emits a duplicate `subscription_suspended` event, and
  `RecordPayment` reactivates only `billing_overdue` subscriptions after the last overdue
  invoice is settled. What was missing was evidence across the two transitions together.
  `TestManualSuspensionSurvivesSchedulerAndPayment` runs the scheduler then a full payment
  against a manually suspended subscription with an overdue invoice and asserts the status,
  reason, disabled RADIUS user, and the absence of both events;
  `TestManualSuspendSurvivesFullPayment` drives the operator `suspend` endpoint followed by
  the real payment handler and reaches the same state. The ownership invariant is documented
  under `docs/session-action-queue.md`. No production code changed.

- `ENG-033` (2026-10-27, `dev` @ `c89624d`): idempotent manual payment retries. `RecordPayment`
  had no retry guard, so two identical requests each added to `PaidAmount` as long as balance
  remained. Manual payments now accept an optional `Idempotency-Key` header (1–64 printable
  ASCII, same rule as public registration) stored in a new tenant-unique
  `isp_payment.idempotency_key` index. The lookup runs under the existing invoice row lock and
  before the closed/balance guards: an identical retry returns the original payment with `200` +
  `Idempotent-Replay: true` without touching money state, even after the invoice is fully paid;
  reusing the key with a different invoice/amount/method/reference returns
  `409 IDEMPOTENCY_CONFLICT`; a concurrent duplicate that loses the unique-index race is mapped to
  the same replay path. Requests without a key remain independent so two equal cash instalments
  stay two payments. The admin payment dialog sends one key per dialog session. Evidence:
  `TestRecordPaymentIdempotentRetryDoesNotDoubleBook`,
  `TestRecordPaymentRetryAfterInvoicePaidReportsDuplicate`,
  `TestRecordPaymentWithoutKeyRecordsSeparateIdenticalPayments`, and the endpoint-level
  `TestCreatePaymentIdempotencyKeyReplaysWithoutDoubleBooking`. Contract recorded in
  `docs/payment-concurrency.md`.

- `ENG-028` (2026-10-27, `dev` @ `59a429d`): PII and secret redaction on log export.
  `/admin/loki/query` returned each log line verbatim, so a password, bearer token, customer
  name, phone, e-mail or subscriber IP written by any service reached the operator response
  unchanged. The export path now classifies fields by data class (`system_logs.go`):
  secret-like keys become `[REDACTED]`, customer identifiers become deterministic
  `sha256:<16 hex>` pseudonyms, addresses keep only their IPv4 `/24` or IPv6 `/48` prefix, and
  free text is swept for `password=`/`token:` assignments, e-mails and IPv4 literals. Filtering
  still runs on the raw line, so operators keep their existing search behaviour, and the
  deterministic pseudonyms keep two entries correlatable. Rules and limits (free-text phone and
  IPv6 are not masked) are recorded in the new canonical `docs/log-redaction.md`. Evidence:
  `TestRedactSystemLogEntryRemovesSensitiveDataClasses`,
  `TestRedactSystemLogEntryPseudonymsAreStable`, `TestMaskSystemLogAddressKeepsOnlyTheNetworkPrefix`
  and the endpoint-level `TestQuerySystemLogsExportsOnlyRedactedEntries`.

- `ENG-019` (2026-10-27, `dev` @ `fc7390d`): raw-SQL tenancy ratchet. The existing inventory in
  `docs/tenant-isolation.md` already covered all nine production `Table`/`Raw`/`Exec`/`Joins`
  sites, so the gap was enforcement: nothing failed when a new unscoped query appeared.
  `TestRawSQLTenantScopeRatchet` (`internal/tenancy/rawsql_ratchet_test.go`) now parses every
  production file under `internal/`, `cmd/` and `pkg/` with `go/ast` and fails when a raw call
  neither mentions `tenant_id` in its own statement nor carries `// tenancy:unscoped: <reason>`.
  The scan is statement-scoped (a scoped query elsewhere in the function cannot mask it), the
  exemption requires a non-empty reason, and fixture subtests prove an unscoped query is
  reported while a scoped one passes. The three platform-level sites in `internal/app/app.go`
  (tenant sequence bootstrap, unique-index DDL, startup dedup) carry the annotation and stay
  documented in the inventory. No production query changed behaviour.

- `ENG-001` (2026-10-27, `dev` @ `4026637`): anonymous outer NAI for EAP-TLS. The gap was real and
  in two places: `validateIdentity` compared the certificate identity against the outer
  `state.Username`, and `stageLoadUser` resolved the account by the outer User-Name *before* EAP
  ran, so `anonymous@realm` failed either way. Now `eap.IsAnonymousNAI` recognises the RFC 7542
  §2.4 forms (`@realm`, `anonymous@realm`, bare `anonymous`); the handler adopts the certified
  identity and still rejects a mismatch for named clients (RFC 5216 §5.2), including before the
  TLS 1.3 protected success indication; `stageLoadUser` defers the lookup for anonymous-NAI EAP
  requests and `stageEAPDispatch` loads the account from the tenant-scoped EAP state via the new
  `Coordinator.AuthenticatedUsername` (refuses unsuccessful, still-anonymous, cross-tenant and
  missing state) before the plugins run. Open items: `NewTLSHandler` untrusted-CA and mismatch
  behaviour are unchanged; PEAP/TTLS keep their own inner-identity handling and are out of scope.
  Evidence: `TestIsAnonymousNAI`, `TestCoordinator_AuthenticatedUsername`,
  `TestTLSHandler_FullHandshake_AnonymousOuterNAI`,
  `TestTLSHandler_TLS13_AnonymousOuterNAI`, `TestStageLoadUser_DefersAnonymousOuterNAIForEAP`,
  plus the CI-backed `anonymous outer NAI resolves the account from the certificate` subtest in
  `test/integration/eap_tls_test.go` (runs in the integration lane). Roadmap M10.5 closed.

- `ENG-021` (2026-10-27, `dev` @ `404a3fe`): `/home` and `/portal` no longer advertise anonymous
  self-service or a checkout step; a 401 is reported as "staff login required" and an
  anonymous visitor is offered WhatsApp or the staff login. The fail-closed behaviour of
  `/public/packages`, `/public/register`, `/public/vouchers/check` and `/portal/lookup` is
  now asserted by `TestLandingSurfaceEndpointsFailClosedAnonymously`. Scope stayed inside the
  existing `Harden` rows of `docs/feature-checklist.md`.

- `ENG-017` (2026-10-27, `dev` @ `a3803b1`): the membership-revocation mechanism already
  existed (`membership_version` claim vs live `TenantMembership.TokenVersion` in
  `resolveOperatorFromContext`, bumped by `SysOpr.AfterUpdate`), so this was an audit + regression
  task rather than a missing mechanism. The gap the inventory named was that no test exercised
  the real mutation endpoints -- the existing `TestTenantContextMiddlewareRejectsRevokedMembership`
  only hits the middleware with a direct DB update. `TestPrivilegeDowngradeInvalidatesLiveToken`
  in `internal/adminapi/access_control_test.go` now proves a downgrade to `operator`, a disable,
  and a delete all reject the operator's previously-issued token on a real mutation endpoint
  (401), that the membership token version actually bumps, that the operator row and its grant
  are removed together, and that a cross-tenant token cannot read, update, or write into
  another tenant's rows (404, no persisted side effect). No production code changed; the finding
  is recorded in `docs/access-control-matrix.md` under a new "Immediate revocation (TR-F033)"
  section.

- `ENG-018` (2026-10-27, `dev` @ `38df193`): audit + regression for ambiguous-NAS and
  identical-username tenant consistency on the RADIUS side. `TestRadiusTenantIsolationForNASAndUsername`
  in `internal/radiusd/radius_test.go` proves that a duplicate NAS source IP with no tenant context
  fails closed (`ErrAmbiguousNASIP`), that tenant-scoped `GetNasForTenant`/`GetValidUserForTenant`
  resolve only the caller's tenant even when two tenants share the same NAS IP and the same
  username, and that accounting Start/Stop writes persist only inside the caller's tenant
  (a cross-tenant stop reports a miss and does not touch the other tenant's row). No production
  code changed.

Critical path from the inventory: `ENG-001 → ENG-002`, `ENG-017 → ENG-018`,
`ENG-041 → ENG-043 → ENG-045 → ECO-005`. Every `ASET` row that starts a chain
(`ENG-041`, `ENG-013`, `ENG-049`, `ENG-002`) is blocked on hardware or a host that has not
been claimed yet, so the unblocked queue is `LOKAL`-only.

## Waves

## Wave 0 — 5 item (LOKAL 4 · ASET 1)

- [x] `ENG-017` TR-F033 P1 — Audit penurunan hak anggota tenant saat token lama aktif · LOKAL · PROPOSED_VERIFICATION · commit `a3803b1`
- [x] `ENG-001` TR-F004 P1 — Audit & rancang outer anonymous NAI pada EAP-TLS 1.3 · LOKAL · VERIFIED_GAP · commit `4026637`
- [x] `ENG-018` TR-F033 P1 — Konsistensi tenant untuk NAS ambigu dan username sama · LOKAL · dep ENG-017 · commit `38df193`
- [x] `ENG-021` TR-F018 P1 — Rekonsiliasi landing/portal publik yang faktanya JWT-only · LOKAL · VERIFIED_GAP
- [ ] `ENG-041` TR-F020 P1 — Uji instalasi fresh pada VPS Ubuntu/Debian bersih · ASET · VERIFIED_GAP · **blocked: no clean VPS host claimed**

## Wave 1 — 37 item (LOKAL 30 · ASET 7)

- [x] `ENG-032` TR-F028 P1 — Invariant suspend manual tidak tertimpa bayar otomatis · LOKAL · PROPOSED_VERIFICATION · commit `5c1946a`
- [x] `ENG-019` TR-F033 P1 — Ratcheting audit query SQL raw dan joins tenancy · LOKAL · PROPOSED_VERIFICATION · commit `fc7390d`
- [x] `ENG-028` TR-F022 P1 — Penyamaran PII di logs, traces dan support evidence · LOKAL · PROPOSED_VERIFICATION · commit `59a429d`
- [ ] `ENG-055` TR-F031 P1 — State loading, stale-data dan retry pada OperationsPage · LOKAL · dep ENG-049 · PROPOSED_VERIFICATION
- [x] `ENG-011` TR-F001 P1 — Harness fuzz paket RADIUS cacat dan oversized · LOKAL · PROPOSED_VERIFICATION · commit `6fcd5ad`
- [x] `ENG-031` TR-F028 P1 — Invoice numbering tahan retry setelah crash PostgreSQL · LOKAL · PROPOSED_VERIFICATION · commit `74fa4c1`
- [x] `ENG-033` TR-F028 P1 — Idempotency pembayaran manual dengan request ulang · LOKAL · PROPOSED_VERIFICATION · commit `c89624d`
- [x] `ENG-052` TR-F016 P1 — E2E matriks tombol per role operator/admin/super · LOKAL · PROPOSED_VERIFICATION · commit `447c0ad`
- [x] `ENG-009` TR-F003 P1 — Regression RadSec mTLS sertifikat tidak tepercaya · LOKAL · VERIFIED_GAP · commit `3e05648`
- [x] `ENG-010` TR-F001 P1 — Audit duplikasi RADIUS retransmission dan replay · LOKAL · PROPOSED_VERIFICATION · commit `54aee93`
- [x] `ENG-012` TR-F002 P1 — Test accounting Start/Interim/Stop out-of-order · LOKAL · PROPOSED_VERIFICATION · commit `9be97f3`
- [x] `ENG-022` TR-F016 P1 — Threat-model autentikasi browser dan sesi admin · LOKAL · PROPOSED_VERIFICATION · commit `e093bd0`
- [x] `ENG-023` TR-F018 P1 — Validasi rate limiting saat reverse proxy multi-hop · LOKAL · PROPOSED_VERIFICATION · commit `818c331`
- [x] `ENG-024` TR-F031 P1 — Runbook rotasi secret SNMP dan encrypted community · LOKAL · PROPOSED_VERIFICATION · commit `db08c5f`
- [x] `ENG-025` TR-F030 P1 — Audit perlindungan kredensial koneksi WhatsApp · LOKAL · PROPOSED_VERIFICATION · commit `aef2150`
- [x] `ENG-027` TR-F031 P1 — Hardening diagnostik jaringan terhadap SSRF · LOKAL · PROPOSED_VERIFICATION · commit `82bc7be`
- [x] `ENG-035` TR-F027 P1 — Konsistensi status Subscription dan RadiusUser saat transisi · LOKAL · PROPOSED_VERIFICATION · commit `c070535`
- [x] `ENG-037` TR-F028 P1 — Validasi integer IDR dan batas invoice items · LOKAL · PROPOSED_VERIFICATION · commit `e8da24f`
- [x] `ENG-038` TR-F007 P1 — Voucher terminal-state concurrency regressions · LOKAL · PROPOSED_VERIFICATION · commit `7c2e06e`
- [x] `ENG-056` TR-F029 P1 — Informasi jujur untuk action portal/payment yang disabled · LOKAL · dep ENG-021 · PROPOSED_VERIFICATION · commit `a171299`
- [x] `ENG-061` TR-F022 P1 — Stabilitas test E2E CI branch main dan dev · LOKAL · PROPOSED_VERIFICATION · commit `521963e`
- [ ] `ENG-064` TR-F010 P1 — Pemeriksaan session-action queue dead-letter dan alert · LOKAL · dep ENG-013 · PROPOSED_VERIFICATION
- [x] `ENG-053` TR-F013 P2 — Navigasi keyboard pada data-grid dan dialog kritis · LOKAL · PROPOSED_VERIFICATION · commit `4404b1b`
- [x] `ENG-004` TR-F003 P2 — Kesiapan rotasi sertifikat RadSec dan EAP · LOKAL · PROPOSED_VERIFICATION · commit `3cf307c`
- [x] `ENG-030` TR-F022 P2 — Inventory SBOM backend/frontend serta lisensi · LOKAL · PROPOSED_VERIFICATION · commit `bf4b18e`
- [x] `ENG-008` TR-F004 P2 — Pastikan EAP-OTP tak dapat diaktifkan tanpa validator · LOKAL · PROPOSED_VERIFICATION · commit `5b934ae`
- [x] `ENG-029` TR-F029 P2 — PNG branding parser dan metadata validation · LOKAL · PROPOSED_VERIFICATION · commit `4af4425`
- [x] `ENG-039` TR-F032 P2 — Perlindungan marker demo data saat clean-up · LOKAL · PROPOSED_VERIFICATION · commit `ca7afdc`
- [x] `ENG-051` TR-F029 P2 — Test fungsional website editor yang masih kurang coverage · LOKAL · VERIFIED_GAP · commit `5add7af`
- [x] `ENG-063` TR-F012 P2 — Contract testing respons API, paginasi dan error · LOKAL · PROPOSED_VERIFICATION · commit `9ea95b8`
- [ ] `ENG-013` TR-F010 P1 — Pilot RFC 5176 CoA/Disconnect pada NAS nyata · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-002` TR-F004 P1 — Matriks interoperabilitas EAP-TLS 1.2/1.3 dengan supplicant nyata · ASET · dep ENG-001 · PROPOSED_VERIFICATION
- [ ] `ENG-043` TR-F020 P1 — Drill pemulihan seluruh volume VPS dari backup · ASET · dep ENG-041 · VERIFIED_GAP
- [ ] `ENG-020` TR-F033 P1 — Latihan restore lintas tenant setelah perubahan skema · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-042` TR-F020 P1 — Simulasi kegagalan registry ketika upgrade aplikasi · ASET · dep ENG-041 · PROPOSED_VERIFICATION
- [ ] `ENG-015` TR-F025 P2 — Profil biaya LDAP bind di bawah concurrent load · ASET · PROPOSED_VERIFICATION
- [ ] `ENG-046` TR-F020 P2 — Uji akses localhost SSH tunnel pada instalasi tanpa DNS · ASET · dep ENG-041 · PROPOSED_VERIFICATION

## Wave 2 — 25 item (LOKAL 13 · RISET 2 · ASET 7 · RISET+ASET 3)

- [ ] `ENG-058` TR-F031 P2 — Label confidence dan mapping pada NOC impact UI · LOKAL · dep ENG-055 · PROPOSED_VERIFICATION
- [x] `ENG-036` TR-F033 P2 — Jejak scheduler FUP terpisah per tenant sesudah perbaikan · LOKAL · dep ENG-019 · PROPOSED_VERIFICATION · commit `39caa5f`
- [x] `ENG-054` TR-F023 P2 — Rekonsiliasi ID/EN seluruh status tagihan dan insiden · LOKAL · PROPOSED_VERIFICATION · commit `ef3c286`
- [x] `ENG-057` TR-F015 P2 — Audit grafik dashboard konsistensi timezone dan nominal · LOKAL · PROPOSED_VERIFICATION · commit `e308244`
- [x] `ENG-059` TR-F007 P2 — Audit import RADIUS user dengan preview dry-run saja · LOKAL · PROPOSED_VERIFICATION · commit `06e7ae4`
- [x] `ENG-062` TR-F022 P2 — Sistem upgrade dependency dengan ratchet keamanan · LOKAL · dep ENG-030 · PROPOSED_VERIFICATION · commit `5173519`
- [x] `ENG-003` TR-F004 P2 — Pemetaan kebijakan pencabutan sertifikat EAP-TLS · LOKAL · PROPOSED_VERIFICATION · commit `d463ec4`
- [ ] `ENG-005` TR-F004 P2 — Evaluasi dukungan inner EAP-TTLS CHAP dan MS-CHAP · LOKAL · dep ENG-002 · PROPOSED_VERIFICATION
- [x] `ENG-026` TR-F020 P2 — Evaluasi parameter KDF dan envelope backup · LOKAL · PROPOSED_VERIFICATION · commit `33c0f5e`
- [x] `ENG-034` TR-F028 P2 — Pembetulan transaksi payment operator yang salah input · LOKAL · PROPOSED_VERIFICATION · commit `9bc7368`
- [x] `ENG-060` TR-F022 P2 — Studi support bundle tersensor untuk insiden operator · LOKAL · dep ENG-028 · PROPOSED_VERIFICATION · commit `e3e5e57`
- [ ] `ECO-003` TR-F020 P2 — Model total cost of ownership deployment ISP · RISET · dep ENG-045 · PROPOSED_VERIFICATION
- [x] `ECO-009` TR-F028 P2 — Audit peluang revenue leakage dari invoice yang sudah ada · RISET · dep ENG-031 · PROPOSED_VERIFICATION · commit `dcf4a7d`
- [ ] `ENG-049` TR-F031 P2 — UAT SNMPv3 authPriv dengan perangkat sungguhan · ASET · VERIFIED_GAP
- [ ] `ENG-045` TR-F020 P2 — Tetapkan RTO/RPO operator berdasarkan rehearsal · ASET · dep ENG-043 · PROPOSED_VERIFICATION
- [ ] `ENG-050` TR-F010 P2 — Matrix NAS/firmware teruji untuk RADIUS dan CoA · ASET · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `ENG-048` TR-F030 P2 — UAT WhatsApp pairing ke nomor uji opt-in · ASET · VERIFIED_GAP
- [ ] `ENG-014` TR-F005 P2 — Prioritas vendor VSA berbasis model NAS riil · ASET · dep ENG-013 · PROPOSED_VERIFICATION
- [ ] `ENG-044` TR-F017 P2 — Studi PostgreSQL WAL PITR untuk RPO terukur · ASET · dep ENG-043 · PROPOSED_VERIFICATION
- [ ] `ENG-047` TR-F020 P2 — Audit health dan peringatan expiry sertifikat deployment · ASET · dep ENG-041 · PROPOSED_VERIFICATION
- [ ] `ECO-001` TR-F027 P2 — Wawancara willingness-to-pay pada operator ISP kecil · RISET+ASET · PROPOSED_VERIFICATION
- [ ] `ECO-002` TR-F029 P2 — Ukur waktu tugas operator sebelum/sesudah MWX-ISP · RISET+ASET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `ECO-007` TR-F030 P2 — Pengendalian biaya telemetri dan alert whatsmeow · RISET+ASET · dep ENG-048 · PROPOSED_VERIFICATION
- [x] `ENG-016` TR-F021 P2 — Pemeriksaan berkala perbedaan fork layeh dan upstream · LOKAL · PROPOSED_VERIFICATION · commit `985908e`
- [x] `ENG-040` TR-F011 P2 — Studi archival accounting retention dan query plan · LOKAL · PROPOSED_VERIFICATION · commit `925aad0`

## Wave 3 — 23 item (LOKAL 2 · RISET 21)

- [x] `ENG-006` TR-F004 P3 — Riset kompatibilitas TEAP terbatas untuk milestone M11 · LOKAL · PROPOSED_VERIFICATION · commit `e1ae01e`
- [x] `ENG-007` TR-F004 P3 — Riset kebutuhan EAP-PWD milestone M12 · LOKAL · PROPOSED_VERIFICATION · commit `eff93d1`
- [ ] `ECO-004` TR-F020 P2 — Evaluasi lisensi managed support vs self-hosted · RISET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `ECO-005` TR-F020 P2 — Desain paket dukungan & SLA yang realistis · RISET · dep ENG-045 · PROPOSED_VERIFICATION
- [x] `ECO-006` TR-F027 P2 — Validasi jasa onboarding dan migrasi pelanggan · RISET · dep ENG-059 · PROPOSED_VERIFICATION · commit `4b2ea0a`
- [ ] `ECO-008` TR-F015 P2 — Cost allocation unit per NAS dan subscriber aktif · RISET · dep ECO-003 · PROPOSED_VERIFICATION
- [ ] `ECO-011` TR-F023 P2 — Kajian strategi distribusi community/open-source · RISET · dep ECO-001 · PROPOSED_VERIFICATION
- [ ] `FUT-013` TR-F031 P3 — Offline read-only NOC snapshot dengan TTL dan enkripsi · RISET · dep ENG-055 · PROPOSED_VERIFICATION
- [x] `FUT-017` TR-F029 P3 — Experimental neo-brutalist manga skin via existing tokens · RISET · dep ENG-053 · PROPOSED_VERIFICATION · commit `7a8b944`
- [x] `FUT-001` TR-F013 P3 — Proof-of-concept intent search untuk perintah operator read-only · RISET · dep ENG-052 · PROPOSED_VERIFICATION · commit `8f9b85c`
- [x] `FUT-002` TR-F031 P3 — AI incident triage berbasis log tersensor · RISET · dep ENG-028 · PROPOSED_VERIFICATION · commit `62c80bb`
- [ ] `FUT-003` TR-F031 P3 — Ringkasan insiden berbukti tanpa mengklaim pelanggan offline · RISET · dep ENG-058 · PROPOSED_VERIFICATION
- [ ] `FUT-004` TR-F031 P3 — Deteksi pola anomali latency dari metrik historis bounded · RISET · dep ENG-049 · PROPOSED_VERIFICATION
- [ ] `FUT-005` TR-F031 P3 — Graph hipotesis penyebab insiden dengan uncertainty · RISET · dep ENG-058 · PROPOSED_VERIFICATION
- [x] `FUT-006` TR-F031 P3 — Visual graph statis NAS-ODP dari relasi yang telah diisi operator · RISET · dep ENG-018 · PROPOSED_VERIFICATION · commit `f026ff4`
- [ ] `FUT-008` TR-F010 P3 — Preview dampak CoA terhadap sesi yang aktif sebelum aksi · RISET · dep ENG-013 · PROPOSED_VERIFICATION
- [x] `FUT-009` TR-F028 P3 — Explanation trace billing-suspend dari rule engine existing · RISET · dep ENG-032 · PROPOSED_VERIFICATION · commit `c372ee3`
- [x] `FUT-010` TR-F033 P3 — Static tenant-scope policy checker dari query AST sederhana · RISET · dep ENG-019 · PROPOSED_VERIFICATION · commit `e521c02`
- [x] `FUT-011` TR-F001 P3 — Pembangkitan regression tests untuk paket RADIUS via mutator · RISET · dep ENG-011 · PROPOSED_VERIFICATION · commit `319699a`
- [ ] `FUT-012` TR-F004 P3 — Eksperimen adversarial EAP TLS fragment scheduler · RISET · dep ENG-002 · PROPOSED_VERIFICATION
- [ ] `FUT-021` TR-F005 P3 — Knowledge base kompatibilitas firmware berbasis bukti uji · RISET · dep ENG-050 · PROPOSED_VERIFICATION
- [x] `FUT-022` TR-F028 P3 — Event replay simulator untuk reproduksi billing dan NAS · RISET · dep ENG-032 · PROPOSED_VERIFICATION · commit `31551a4`
- [x] `FUT-023` TR-F021 P3 — Graph semantik relasi protokol RADIUS dan feature ID · RISET · dep ENG-016 · PROPOSED_VERIFICATION · commit `f4de6b2`

## Wave 4 — 10 item (RISET 8 · RISET+ASET 2)

- [ ] `ECO-012` TR-F005 P2 — Partner interoperability kit berbasis fixture NAS · RISET · dep ENG-050 · PROPOSED_VERIFICATION
- [ ] `ECO-010` TR-F031 P3 — Riset risiko churn berbasis sinyal operasional agregat · RISET · dep ECO-002 · PROPOSED_VERIFICATION
- [x] `FUT-007` TR-F008 P3 — Simulasi perubahan profil bandwidth tanpa menulis NAS · RISET · dep ENG-016 · PROPOSED_VERIFICATION · commit `a02c698`
- [ ] `FUT-014` TR-F031 P3 — Field diagnostics mini-mode untuk teknisi yang offline · RISET · dep FUT-013 · PROPOSED_VERIFICATION
- [x] `FUT-015` TR-F013 P3 — Adaptive contextual workspace dari tugas operator · RISET · dep ENG-053 · PROPOSED_VERIFICATION · commit `e1b5261`
- [x] `FUT-016` TR-F013 P3 — Voice-assisted navigation untuk aksesibilitas internal · RISET · dep ENG-053 · PROPOSED_VERIFICATION · commit `808bf61`
- [x] `FUT-018` TR-F029 P3 — Riset visual incident card grunge yang bisa dicetak · RISET · dep FUT-017 · PROPOSED_VERIFICATION · commit `de91f23`
- [x] `FUT-020` TR-F033 P3 — Per-tenant statistik operasional privat berbasis agregat · RISET · dep ENG-017 · PROPOSED_VERIFICATION · commit `a50f99f`
- [ ] `FUT-019` TR-F031 P3 — Edge collector polling read-only pada jaringan terbatas · RISET+ASET · dep ENG-049 · PROPOSED_VERIFICATION
- [ ] `FUT-024` TR-F031 P3 — Prediksi kapasitas network yang dibatasi data SNMP nyata · RISET+ASET · dep ENG-049 · PROPOSED_VERIFICATION

## Suggested next batch

Empty for Waves 0-2. `ECO-009` (2026-10-29, `dcf4a7d`) was the last Wave 0-2 row
executable without a new asset, so that ordered queue is exhausted. Counts are
taken from the checkboxes in the wave lists, not from memory:

| Wave | Items | Done | Open | Open that needs an asset or a dependency on one |
| --- | ---: | ---: | ---: | --- |
| 0 | 5 | 4 | 1 | the 1 (`ENG-041` clean VPS) |
| 1 | 37 | 28 | 9 | all 9 — 7 `ASET` rows plus `ENG-055`/`ENG-064`, whose only unsatisfied dependency is an `ASET` row |
| 2 | 25 | 12 | 13 | all 13 — 7 `ASET`, 3 `RISET+ASET`, plus `ENG-005`, `ENG-058` and `ECO-003` behind one |
| **Waves 0-2** | **67** | **44** | **23** | **23 of 23** |

The inventory-wide total adds the rows delivered out of wave order because they
were locally executable — **12** Wave 3 rows (`ENG-006`, `ENG-007`, `ECO-006`,
`FUT-001`, `FUT-002`, `FUT-006`, `FUT-009`, `FUT-010`, `FUT-011`, `FUT-017`,
`FUT-022`, `FUT-023`) and **5** Wave 4 rows (`FUT-007`, `FUT-015`, `FUT-016`,
`FUT-018`, `FUT-020`).

Every remaining open row is listed with the reason it cannot be closed by
repository work in the blocked register below, so the Wave 0-2 close-out is a
complete statement: 44 delivered, 23 no-go, none undecided.

Work continues in Waves 3-4, where the rows that need no asset remain. Ordered so
each step stays executable without hardware; `ENG-006` (TEAP readiness for M11),
`ENG-007` (EAP-PWD need for M12), `ECO-006` (onboarding and migration
validation), `FUT-017` (the manga-ink contrast audit, closed as a measurement
of the shipped direction), `FUT-001` (the intent search proof of concept, closed
as a resolver that can only name a read route), `FUT-002` (incident triage
over sanitized logs, closed as three engines with a measured hallucination rate
and no path from a verdict to an action) and `FUT-007` (the bandwidth-profile
what-if estimator, closed as a reproducible wire diff that holds the product's
answer instead of predicting one, with the cache-miss fallback and the
checker-only columns recorded as the parts a preview must refuse to promise) were
the first seven and all closed 2026-10-29. `FUT-015` (the contextual workspace
study, closed as a pricing of the task surfaces the console already ships, with
the role-gate disagreement it exposes and the human/live-session half it cannot
close) is the eighth and `FUT-016` (the voice-navigation study, closed as a
measurement of what a navigation-only vocabulary can be asked to do, with the
audio-corpus, panel and live-session half it cannot close) the ninth, also
2026-10-29. `FUT-018` (the printable incident-card study, closed as a measurement
of what a grunge A4 sheet may say — four independent checks over the 129 slots the
incident response types serve, greys derived from the shipped tokens, and the
paper/proof-sheet half recorded no-go) is the tenth and `FUT-020` (the
per-tenant operational-statistics study, closed as a census of the 157 fields the
11 statistics surfaces serve, with the 27 aggregates that are a cell of one, the
one rate that turns out to be a count, the price of a cell floor and the empty
governance half recorded as no-go) is the eleventh and last — all eleven closed
2026-10-29.

Nothing asset-free is left in Waves 3-4.

The other 11 open Wave 3 rows and 5 open Wave 4 rows inherit the same no-go as
the register below: they are `ASET`/`RISET+ASET`, or they depend on a row that is.

## Blocked register (Waves 0-2)

Explicit status for every open Wave 0-2 row. No-go means the item cannot be
closed by repository work at all; it needs an asset the project does not have.

**No-go — needs a clean VPS host (`ENG-041` chain).** Nothing in this chain can
start until a host is claimed, and no local substitute proves the claim:
`ENG-041` (fresh install), `ENG-043` (volume-loss restore drill), `ENG-042`
(registry outage during upgrade), `ENG-046` (localhost SSH tunnel without DNS),
`ENG-047` (deployment certificate health and expiry), `ENG-044` (PostgreSQL WAL
PITR), `ENG-045` (RTO/RPO from a rehearsal), `ECO-003` (TCO model, depends on
`ENG-045`).

**No-go — needs a real NAS in a lab (`ENG-013` chain).** `ENG-013` (RFC 5176
CoA/Disconnect pilot), `ENG-050` (tested NAS/firmware matrix), `ENG-014` (VSA
priority from a real device model), `ENG-064` (queue dead-letter and alert check,
depends on `ENG-013`).

**No-go — needs supplicant hardware.** `ENG-002` (EAP-TLS 1.2/1.3
interoperability matrix with real supplicants), `ENG-005` (TTLS inner CHAP /
MS-CHAP evaluation, depends on `ENG-002`).

**No-go — needs a running SNMPv3 device.** `ENG-049` (SNMPv3 authPriv UAT),
`ENG-055` (OperationsPage loading, stale and retry states, depends on `ENG-049`),
`ENG-058` (NOC impact confidence labels, depends on `ENG-055`).

**No-go — needs a live WhatsApp pairing number.** `ENG-048` (pairing UAT to an
opt-in test number), `ECO-007` (telemetry and whatsmeow cost control, depends on
`ENG-048`).

**No-go — needs a second tenant database and a restore target.** `ENG-020`
(cross-tenant restore after a schema change).

**No-go — needs concurrent LDAP load against a real directory.** `ENG-015`
(LDAP bind cost profile).

**No-go — needs human respondents.** `ECO-001` (willingness-to-pay interviews),
`ECO-002` (operator task timing before/after, depends on `ECO-001`).

Blocked items do not gate the Wave 0-2 close-out: each one is either an ASET row
whose asset does not exist, or a LOKAL/RISET row whose only unsatisfied
dependency is such an ASET row.
