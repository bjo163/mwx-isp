# RADIUS user import audit — row errors, rollback, and the preview decision

Feature: TR-F007 (RADIUS user management) · backlog item ENG-059.

Acceptance for this item: **a bad file is rejected without changing anything**,
and a proposed dry-run preview is **not** deployed without checklist approval.

## Method

Read the whole path before editing:

- `internal/adminapi/users_import.go` (handler + row mapping) and
  `users_import_test.go` at `dev` @ `5ec717f`.
- `webserver.ImportData` (`internal/webserver/server.go:496`) for parsing of
  `.xlsx` / `.csv` / `.json` and for the synthetic `id` it injects.
- `status_checker.go` for what a `status` value actually does at authentication
  time, plus the `(tenant_id, username)` unique index in `internal/domain/radius.go:52`.
- The operator surface, `ImportUsersButton` in `web/src/resources/radiusUsers.tsx`.

The old test carried the defect as an expectation: a five-row file with four
invalid rows asserted `success = 1`, i.e. the four valid writes were expected to
land even though the file was bad.

## Findings

| # | Severity | Finding | Evidence (before) | Decision |
| --- | --- | --- | --- | --- |
| F1 | High | The importer wrote row by row with no transaction, so a payload that failed halfway left every earlier row committed. "File buruk ditolak tanpa perubahan" was false: the acceptance test itself asserted a partial import. | per-row `db.Create` in the loop; `TestImportRadiusUsers_Errors` expected `Success == 1` | Replaced by two phases: `planUserImport` validates **all** rows without writing, and only a fully valid payload is inserted inside one `db.Transaction`. Any row error returns the per-row report with zero writes. |
| F2 | High | `status` was free text. `StatusChecker` only rejects `disabled` (`common.DISABLED`), so a typo such as `nonaktif`, `suspend`, or `active ` imported a **login-capable** user. | `status := strings.ToLower(...)` with no allowlist | Reject anything outside `enabled` / `disabled` (case-insensitive); empty still defaults to `enabled`. |
| F3 | Medium | Duplicate usernames **inside one file** were only caught because the first row had already been inserted and the second one's `SELECT COUNT` saw it. A validate-first design must not rely on that side effect. | `exists` count query per row | `planUserImport` tracks usernames as it goes and rejects the later row with `Username duplicates row N`. |
| F4 | Medium | `profile_id: "abc"` produced "Billing profile is required" (because `cast.ToInt64` yielded 0), and a non-numeric `node_id` was silently dropped to 0 instead of rejected — a row could import without the intended node binding. | `cast.ToInt64(importMapString(...))` | Split "missing" from "invalid": the profile id must parse as a positive integer; a present `node_id` must parse and resolve, otherwise the row is rejected. |
| F5 | Medium | There is no dry-run preview: the file is applied on upload, and the operator learns what happened only from the counts afterwards. | `ImportUsersButton` posts straight to `/users/import` | Preview **designed, not deployed** (see below). The criterion forbids shipping it without a checklist decision; the validation phase now exists as a side-effect-free function, so wiring it later is small. |
| F6 | Low | Silent defaults: `expire_time` becomes now + 1 year, `status` becomes `enabled`, and `addr_pool` / `domain` / `radius_class` are inherited from the profile. None of it is visible before the import. | defaults in the row builder | Kept as documented behaviour; the preview design surfaces them per row as "planned value". |
| F7 | Low | A failed `db.Create` returned `err.Error()` in the per-row message, exposing table/column names to the client. With one transaction the failure is batch-level, so the response now carries a generic message and the driver error goes to the operator log only (`fail()` redacts 5xx details). | `Message: err.Error()` | Fixed by the new error path. |
| F8 | Low | Passwords are trimmed (`strings.TrimSpace`) and stored as plain text. Plaintext is a pre-existing TR-F007 requirement (CHAP/PAP verify against it); trimming means a password with meaningful leading/trailing spaces cannot be imported. | `importMapString` | No change; recorded as a documented limit. |

## Decisions

1. **Atomic import.** Validation is separated from writing and the writes share
   one transaction. The response shape is unchanged (`total`, `success`,
   `failed`, `errors`), and `failed > 0` now always implies `success == 0`.
   The endpoint still answers `200` when a file is rejected so the operator
   keeps the per-row report; a database failure during the commit returns `500`
   with a generic message after the rollback.
2. **No preview shipped.** The criterion says the proposal must not be deployed
   without approval, so only the enabling refactor (`planUserImport`) went in.

### Proposed preview (awaiting approval — not implemented)

- `POST /api/v1/users/import?dry_run=true` (or a `dry_run` form field) parses
  the file, calls `planUserImport`, and returns the same `ImportUserResult`
  without opening a transaction. No new validator code is required.
- The response should add a `plan` array for row-level truth: `row`,
  `username`, `profile_id`, `status`, `expire_time`, `node_id`, and which values
  were inherited from the profile (F6).
- UI: upload → show the table of planned rows and errors → a second explicit
  "Import N users" button sends the same file without `dry_run`.
- Approval needed: a new request parameter plus one UI step on the TR-F007
  surface, which the item's checklist gate requires before delivery.

## Verification

```bash
go test ./internal/adminapi/ -run TestImportRadiusUsers -count=1
go vet ./internal/adminapi/
cd web && npm run type-check && npm run build
```

Results on 2026-10-29 (`dev`): import tests `ok … 4.0s`; `go vet` clean;
`tsc --noEmit` clean; `vite build` succeeded.

Tests changed or added in `internal/adminapi/users_import_test.go`:

- `TestImportRadiusUsers_Errors` — the five-row file now asserts
  `success = 0`, `failed = 4` and that the table still holds only the
  pre-existing user, including an explicit check that the valid row `ok_user`
  was **not** written.
- `TestImportRadiusUsers_RejectsDuplicateRowsWithinFile` — rows 1 and 2 share a
  username; row 2 is reported as duplicating row 1 and nothing is created.
- `TestImportRadiusUsers_RejectsUnusableValues` — table-driven: non-numeric
  profile id, unknown status (`nonaktif`), non-numeric node id. Each is a
  single-row file that must be rejected with an empty table afterwards.
- `TestImportRadiusUsers_AcceptsDisabledStatus` — `"Disabled"` is normalised and
  imported, and the stored status is `common.DISABLED`.

The existing JSON, CSV and profile-inheritance (`RadiusClass`) tests still pass
with `success == total`, which pins the happy path under the transaction.

## Limits

- A mid-transaction database failure could not be forced in the SQLite test
  setup; atomicity rests on GORM's `Transaction` rollback plus the `(tenant_id,
  username)` unique index, which turns two concurrent imports of the same
  username into a rollback rather than a duplicate row.
- The new tests use JSON payloads; the Excel path is covered only by the shared
  parser (`webserver.ImportData`) and was not re-exercised with a binary fixture.
- Per-row validation still issues one profile lookup and one existence query per
  row (as before). This is fine for operator-sized files; there is no explicit
  row-count or file-size cap beyond the HTTP server limits.
- No preview is available, so the "what would this file do" question still has
  no answer in the product until the proposal above is approved.
