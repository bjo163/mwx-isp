# ECO-006 — Is there a migration service to sell, and what would it run on?

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`.
Backlog row: `ECO-006` (`TR-F027`, P2, RISET, dependency `ENG-059` delivered) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/migrationcheck` (read-only, imported by nothing).
Criterion under test: **mock migration checklist, dry-run test, effort estimate,
and interviews with three prospective clients.**

## The question

`ENG-059` audited the importer and stopped at "the bad file is rejected whole, and
the preview is designed but not shipped". `ECO-006` asks the commercial question
built on top of it: if MWX-ISP sold a controlled migration service - somebody
extracts a customer's CSV from their old system, maps it, loads it, and signs it
off - what does the product actually support today, what would the operator have
to do by hand, and what would the checklist say?

Three sub-questions, each answerable from the tree or not at all:

1. Does a rejected file really leave the database untouched, and is that guarantee
   pinned by a test rather than by prose?
2. Is there a dry-run of any kind in the migration path - and if not, is that a
   missing capability or an unapproved one?
3. What is the shape of the work: which columns does the importer accept, which
   rules reject a row, what must exist before a customer row can be validated, and
   what is left manual?

The fourth part of the criterion - interviews with three prospective clients -
cannot be answered from the tree at all. It is recorded as a no-go with its
blocking asset, not converted into an inference.

## What counts as evidence

Same three rules the `ENG-006` and `ENG-007` studies used, enforced by tests in
`internal/migrationcheck`:

- **A citation is a code line.** Comment-only lines are blanked before a symbol is
  matched, so a doc comment that names an endpoint proves the comment exists, not
  the behavior (`TestCommentMentionsAreNotEvidence`, `TestEveryEvidenceLineIsCode`).
- **A whole identifier.** Word-boundary matching, so a longer name cannot prove a
  shorter claim (`TestCommentMentionsAreNotEvidence`).
- **Drift is reported, never dropped.** A moved file or a vanished symbol is
  `broken`, and a gap that got filled is `vacated`
  (`TestMovedFileIsReportedBrokenNotDropped`).

Two rules specific to this study:

- **Nothing may be claimed about people.** No statement in the catalog, checklist
  or estimate may read as though a client was interviewed, agreed, or signed
  anything (`TestNoClaimPromisesClientResearch`).
- **No invented prices.** Sizes are t-shirt labels, each one computed from a
  measured count with published thresholds; the rendered estimate may not contain
  an hour, a day, a currency symbol or a currency code
  (`TestEstimateCarriesNoInventedNumbers`, `TestSizeFollowsTheMeasuredCount`).

## The prototype

| File | Role |
| --- | --- |
| `internal/migrationcheck/claims.go` | 20 claims pinned to an identifier, 5 negative claims, and the verification pass |
| `internal/migrationcheck/surface.go` | reads the import handler and the upload reader and extracts the vocabulary they accept |
| `internal/migrationcheck/estimate.go` | the ordered checklist, the sized work items, and the scans behind their drivers |
| `internal/migrationcheck/render.go` | the Markdown artifact this memo is transcribed from |

It writes nothing, calls no network, opens no database, and has no importers
outside its own tests.

## Verdict

**All 20 need claims confirm on the current tree; all 5 gaps still hold. The
criterion is met on three of its four parts, and the fourth is an asset gate.**

| Criterion part | Status | Measured support |
| --- | --- | --- |
| Mock migration checklist | **Met** | 20 ordered steps, each carrying a `file:line` citation (`TestChecklistCoversEveryClaim` fails if a claim is written and left off the list) |
| Dry-run test | **Met as a regression rehearsal, not as a product feature** | 8 named tests run a file through validation and assert the zero-write outcome; no `dry_run` flag exists in the importer, and the preview stays unapproved |
| Effort estimate | **Met, as relative sizes only** | 6 work items, each sized from a count this package measured; no clock or currency figure |
| Three client interviews | **No-go - asset gate** | requires named prospective clients and their consent; no source file can substitute |

## The claims the service rests on

| Claim | Evidence | Why it matters |
| --- | --- | --- |
| Every row is validated before the first write | `internal/adminapi/users_import.go:70` | a rejected row means nothing was written, so the same file can be corrected and resubmitted |
| Accepted rows go in inside one transaction | `internal/adminapi/users_import.go:76` | no half-imported customer set to reconcile |
| A rolled-back import states it | `internal/adminapi/users_import.go:84` | the client report quotes an outcome, not an operator's memory |
| Each rejected row is identified | `internal/adminapi/users_import.go:23` | the row-number list is the correction worksheet |
| A row without a password is refused | `internal/adminapi/users_import.go:121` | the export must carry a usable password per customer - no password, no row |
| An existing username is refused, never merged | `internal/adminapi/users_import.go:148` | the import cannot fold a migration into an account the client already has |
| The same username twice in one file is refused | `internal/adminapi/users_import.go:114` | deduplicate before upload |
| The billing profile must already exist | `internal/adminapi/users_import.go:138` | plans are rebuilt before customers, then mapped onto new numeric ids |
| The network node must already exist | `internal/adminapi/users_import.go:179` | site inventory is a precondition, or the column stays empty |
| Status accepts only `enabled` / `disabled` | `internal/adminapi/users_import.go:165` | an unmapped spelling fails the row instead of silently activating service |
| A missing expiry becomes one year | `internal/adminapi/users_import.go:100` | confirm the renewal policy before importing |
| Three columns inherit from the profile | `internal/adminapi/users_import.go:186` | spot-check inherited values after the run; the file cannot show them |
| The worksheet name is hard-coded to `Sheet1` | `internal/adminapi/users_import.go:64` | rename the client's sheet, or use CSV/JSON |
| Three input formats dispatch on the suffix | `internal/webserver/server.go:518` | pick the format the client's system exports cleanly |
| Import needs an admin-level login | `internal/adminapi/users.go:284` | the migration does not run on the client's routine operator account |
| The tenant comes from the session, not the file | `internal/tenancy/scope.go:39` | a payload cannot place rows in another tenant |
| Username is unique per tenant | `internal/domain/radius.go:52` | a shared-namespace client must settle the naming scheme first |
| The password is stored as supplied | `internal/domain/radius.go:54` | the migration file is cleartext credential material |
| Zero-write-on-rejection is a CI assertion | `internal/adminapi/users_import_test.go:131` | the guarantee is regression-tested, not just documented |
| The preview is designed and unapproved | `docs/radius-user-import-audit-2026-10-29.md:48` | quote it as a proposal; shipping it before a checklist decision would break the criterion |

## The gaps that are still gaps

| Gap | Verdict | What it costs a migration |
| --- | --- | --- |
| No `dry_run` flag in the importer | confirmed | the operator cannot see what a file will do before applying it |
| No path for a hashed password | confirmed | a client whose old system exports hashes cannot be migrated without a re-keying step |
| No upsert in the importer | confirmed | a re-run after a partial correction cannot refresh existing rows; delete and re-add |
| No import handler for profiles/packages | confirmed | the plan catalog every row depends on is rebuilt by hand first |
| No import handler for nodes/NAS | confirmed | site inventory is manual, so the estimate counts it separately |

Exactly **one** upload handler exists in the whole admin API
(`internal/adminapi/users_import.go:64`): customer rows. Everything else in a
migration is entered by hand today.

## What the importer actually accepts

Measured from the source, not from a template document:

- **18 columns, 41 accepted header spellings.** Aliases exist per column
  (`username` / `Username` / `USERNAME`; `profile_id` / `profileid` / `ProfileId`
  / `profile`), which is why a hand-built mapping table is usually a rename job
  rather than a code change.
- **12 distinct rejection reasons**, in source order: `Username is required`,
  `Username duplicates row %d`, `Password is required`, `Billing profile is
  required`, `Billing profile id must be a positive integer`, `Associated billing
  profile not found`, `Unable to check for an existing username`, `Username
  already exists`, `Invalid expire time format`, `Status must be enabled or
  disabled`, `Network node id must be a positive integer`, `Associated network
  node not found`.
- **11 values a row does not state.** Three fall back to the profile only when the
  cell is empty (`addr_pool`, `domain`, `radius_class`), six are always copied from
  the profile (`ActiveNum`, `UpRate`, `DownRate`, `IPv6PrefixPool`, `BindVlan`,
  `BindMac`), and two are defaulted (`status=enabled`, `expire_time=now+1y`). These
  are exactly the values a preview would have to surface.
- **Formats:** `json`, `csv`, and Excel as the fallback when the name is neither.

## Effort items, sized from those measurements

Sizes are t-shirt labels derived from the counts above with published thresholds;
no clock or currency figure is asserted, because the master inventory keeps those
baselines at UNKNOWN until an engagement measures them. An item reports `unknown`
instead of a size the moment one of its claims fails to verify.

| Item | Size | Measured driver |
| --- | --- | --- |
| Map the client's export onto the accepted columns | S | 18 columns read through 41 accepted header spellings |
| Rebuild the plan catalog and node inventory by hand | M | 2 dependent entity types with no import surface, 1 upload handler in the whole admin API |
| Agree the existing-account and re-run policy | M | 3 rejection reasons are collision rules a file cannot express |
| Run the cleartext password handover as a controlled step | M | 12 distinct row rejections decide what the handover file must contain |
| Wire the proposed dry-run preview behind an approval | M | 11 inherited or defaulted values a preview must surface; 2 preflight implementations already exist to copy |
| Keep the bad-file guarantee under regression | M | 8 named tests already pin the import's failure behavior |

The pattern the table shows is the actual finding: **the migration service is not
short of code so much as short of a decision surface.** Five of six items are
process work against a validator that already behaves correctly; the sixth, the
preview, is one small code change that is deliberately not made.

## The preview stays unshipped, on purpose

`ENG-059` designed `POST /api/v1/users/import?dry_run=true` and recorded it as
"awaiting approval - not implemented"
([radius-user-import-audit-2026-10-29.md](radius-user-import-audit-2026-10-29.md)).
`ECO-006`'s own criterion is the reason it must stay that way: shipping the
preview inside a research item would deploy a proposed surface without the
checklist approval the criterion demands. So the study tests the dry-run posture
without creating it:

- The validation phase is already side-effect-free by construction - `planUserImport`
  returns the planned rows and the errors before any write exists in the call path
  (`users_import.go:70`), which is what makes a later preview a wiring job.
- Two non-mutating preflight implementations already exist to copy, so the design
  is not novel: `restoreDryRun` in `internal/adminapi/system_backup.go` and
  `errRotationDryRun` in `internal/networkmonitor/secret_rotation.go`, both
  asserted by their own tests.
- The rehearsal that a dry-run would provide does exist as a CI path: 8 named tests
  put a file through validation and check the outcome, including
  `TestImportRadiusUsers_Errors` (four bad rows, zero successes, no row written)
  and the integration `TestUserImportCSV`.

That is the honest boundary: **a rehearsal against fixtures, not against a client's
file.** A real rehearsal needs a test tenant and a client export, and neither is in
the tree.

## No-go register

| Item | Why it is not executable here | Asset that unblocks it |
| --- | --- | --- |
| Interviews with three prospective clients | the tree cannot produce people, consent, or their requirements | named prospective clients and a signed interview scope |
| A rehearsal on a real client export | fixture CSV/JSON in tests is not a customer file | a sanitized client export plus a throwaway tenant |
| A price or duration for the service | every numerical baseline in the master inventory starts UNKNOWN | a scoped engagement to measure against |
| Shipping the dry-run preview | the criterion forbids deploying a proposed surface without approval | a `docs/feature-checklist.md` decision |

## Decisions

1. **The migration service is real but thin.** One upload handler, 18 columns, 12
   rejection rules, and a validator that refuses cleanly. Anything a sales
   conversation promises beyond "we will map and load your customer file" is
   currently manual entry.
2. **The blocker is a password-shape blocker, not a volume blocker.** A client
   whose old system stores hashes cannot be migrated at all without re-keying every
   account, because the importer has no hash path and the server compares a
   recomputable secret. That should be the first qualifying question in any
   engagement, and it is a question about the client's system, not this one.
3. **Dependencies are ordered by validation, not by preference.** Profiles and
   nodes must exist before a customer row can validate, and the plan catalog has no
   import surface, so the checklist puts the manual catalog work first.
4. **Do not build the preview inside this item.** It is a checklist decision, and
   the criterion treats an unapproved deployment as a failed acceptance. The
   measured driver (11 values to surface, 2 precedents to copy) is recorded so the
   decision can be made cheaply later.
5. **Sizes, not prices.** The study reports relative effort against measured
   counts and refuses to translate them into money or calendar time; no number in
   this memo is a baseline until an engagement measures it.
6. **`TR-N002` holds.** This is an operator-run migration of records into the
   existing `TR-F027` surface. It is not a CRM, not a customer self-service portal,
   and not a new import framework for every entity type.
7. **No client-support or client-intent claim is made.** Every statement here is
   about this repository, and `TestNoClaimPromisesClientResearch` fails if a claim
   starts reading as though a client had been consulted.

## What this means for the portfolio

`ECO-006` closes as delivered-with-scope: the checklist, the dry-run posture and the
effort estimate are measured and pinned; the interviews and any price stay on the
no-go register with the asset each needs. Nothing here schedules new product work.
If the preview is ever wanted, it is `ENG-059`'s design plus the approval line, not
a new study.

## Verification

- `go build -p 2 ./...` and `go vet ./internal/migrationcheck/` - clean.
- `go test -p 2 -count=1 ./internal/migrationcheck/` - 31 tests, all passing; the
  tables above are transcribed from `TestMeasureSurfaceRealTree`, which renders the
  report and logs it.
- `golangci-lint run ./internal/migrationcheck/...` - 0 issues.
- `internal/migrationcheck` has no importers outside its own tests.

## Residual limits

- The vocabulary is read from one handler. If a second import surface is added, the
  count assertion in `TestTheProductHasOneImportSurface` fails on purpose, and the
  checklist needs a new section rather than an edit to this memo.
- Extraction is textual. A validator rewritten into a helper outside
  `users_import.go` would shrink the measured counts without any behavior changing;
  the claim catalog would still confirm, and only the counts would move.
- Nothing here measures operator preference, client satisfaction, or willingness to
  pay. Those are exactly the four rows on the no-go register.
