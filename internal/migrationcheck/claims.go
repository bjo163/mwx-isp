// Package migrationcheck checks the claims behind ECO-006 (the onboarding and
// customer-migration service) against the code that would have to deliver it.
//
// It is an ENG-007-line research prototype. It only reads, no product code
// imports it, and it answers the three questions the ECO-006 criterion asks and
// a roadmap line cannot answer by itself: whether the import path really refuses
// a bad file without changing the database, whether any dry-run preview of a
// migration exists or has been deployed, and what a migration service would have
// to be built from. Every checklist step and every effort driver it reports is
// derived from a pinned identifier in the tree, so a statement that no longer
// matches a file is reported as broken rather than quietly dropped.
package migrationcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Claim is one statement the migration-service argument depends on, pinned to
// the file that proves it.
type Claim struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the study asserts about the current onboarding surface.
	Statement string
	// File is the repository-relative path that must prove it.
	File string
	// Symbol is the text that must appear in that file. It is matched as an
	// identifier, so a longer name cannot prove a shorter one.
	Symbol string
	// ChecklistStep is the migration-checklist line this claim decides, or why
	// the claim matters for the effort estimate.
	ChecklistStep string
}

// Absence is a negative claim: a symbol that must not appear within scope for
// the migration gap to be real rather than already filled.
type Absence struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the absence proves.
	Statement string
	// Symbol must not appear in any file under Scope.
	Symbol string
	// Scope is a repository-relative directory prefix, or a single file.
	Scope string
	// Gap says what a migration service would add if the absence holds.
	Gap string
}

// Status is one verification verdict.
type Status string

// The three verdicts a claim can reach. A broken claim is reported, never
// silently removed, because the memo it supports would otherwise keep a
// statement the code no longer makes.
const (
	StatusConfirmed Status = "confirmed"
	StatusBroken    Status = "broken"
	StatusVacated   Status = "vacated"
)

// Finding is one verified claim.
type Finding struct {
	// Claim is the statement checked.
	Claim Claim
	// Status is the verdict.
	Status Status
	// Evidence is the file:line that proves it, or where the search failed.
	Evidence string
	// Note carries Claim.ChecklistStep so a verdict reads as an action.
	Note string
}

// String renders one finding as a single line.
func (f Finding) String() string {
	return fmt.Sprintf("%s %s [%s] %s", f.Claim.ID, f.Status, f.Claim.File, f.Evidence)
}

// AbsenceFinding is one verified negative claim.
type AbsenceFinding struct {
	// Absence is the negative claim checked.
	Absence Absence
	// Status is confirmed when nothing matches, vacated when something does.
	Status Status
	// Hits lists the matching locations, capped for readability.
	Hits []string
}

// String renders one negative verdict.
func (a AbsenceFinding) String() string {
	if len(a.Hits) == 0 {
		return fmt.Sprintf("%s %s (no %q under %s)", a.Absence.ID, a.Status, a.Absence.Symbol, a.Absence.Scope)
	}
	return fmt.Sprintf("%s %s %s", a.Absence.ID, a.Status, strings.Join(a.Hits, ", "))
}

// maxHits caps how many locations one negative verdict reports.
const maxHits = 4

// Claims returns the statements the migration-service argument rests on. Each is
// pinned to the code that decides it, so the shape of the service is re-derived
// from the tree rather than recalled from an audit note.
func Claims() []Claim {
	return []Claim{
		{
			ID:            "validation-runs-before-any-write",
			Statement:     "Every row is validated and planned before the first database write, and a payload with any rejected row returns before the write phase.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "planUserImport",
			ChecklistStep: "Run the file through validation first; a rejected row means nothing was written, so the operator can fix and resubmit the same file.",
		},
		{
			ID:            "one-transaction-per-file",
			Statement:     "Accepted rows are inserted inside a single transaction.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "db.Transaction",
			ChecklistStep: "A database error mid-file rolls the whole batch back; there is no half-imported customer set to reconcile.",
		},
		{
			ID:            "rollback-is-told-to-the-operator",
			Statement:     "A rolled-back import answers with an explicit code that states no users were created.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "IMPORT_FAILED",
			ChecklistStep: "The migration report to the client can quote the exact outcome instead of an operator's memory.",
		},
		{
			ID:            "per-row-errors-are-addressable",
			Statement:     "Each rejected row is reported with its row number, username and reason.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "ImportUserError",
			ChecklistStep: "Hand the client the row-number list as the correction worksheet; row numbers are 1-based within the parsed payload.",
		},
		{
			ID:            "cleartext-password-is-mandatory",
			Statement:     "A row without a password value is rejected, so the migration export must carry a usable password per customer.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Password is required",
			ChecklistStep: "Agree the password handover with the client before extraction: no password, no row. Never store the exported file after signoff.",
		},
		{
			ID:            "existing-user-blocks-the-row",
			Statement:     "A username that already exists is rejected rather than updated, so the import cannot merge into an account the client already has.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Username already exists",
			ChecklistStep: "Decide the overlap policy with the client in advance: rename, delete, or skip the colliding accounts before the file is produced.",
		},
		{
			ID:            "duplicate-rows-in-one-file-block",
			Statement:     "The same username twice within one file is rejected, and the error names the first row that used it.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Username duplicates row",
			ChecklistStep: "Deduplicate the source export before upload; the file itself is the record of what was submitted.",
		},
		{
			ID:            "packages-before-customers",
			Statement:     "A row is rejected when its billing profile id does not exist in this deployment, so plans must be created first.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Associated billing profile not found",
			ChecklistStep: "Recreate packages and profiles in the target instance first, then map the client's plan names onto the new numeric ids.",
		},
		{
			ID:            "nodes-before-customers",
			Statement:     "A row naming a network node is rejected when that node does not exist yet.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Associated network node not found",
			ChecklistStep: "Bring the NAS and node inventory up before the customer file, or leave the node column empty.",
		},
		{
			ID:            "status-vocabulary-is-two-values",
			Statement:     "Status accepts only enabled and disabled; any other spelling is rejected instead of defaulting to enabled.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Status must be enabled or disabled",
			ChecklistStep: "Normalize the client's status column to those two literals; an unmapped value fails the row rather than silently activating service.",
		},
		{
			ID:            "expire-defaults-one-year",
			Statement:     "A row without an expiry date is given a date one year out.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "AddDate",
			ChecklistStep: "Confirm the renewal policy with the client before importing: a missing expiry date becomes a year of service, not an unlimited one.",
		},
		{
			ID:            "profile-inherits-six-fields",
			Statement:     "Address pool, domain, RADIUS class and the rate limits fall back to the billing profile when the row leaves them empty.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "profile.AddrPool",
			ChecklistStep: "Spot-check the inherited values after import; the file cannot show what the profile supplied.",
		},
		{
			ID:            "excel-sheet-name-is-fixed",
			Statement:     "The spreadsheet reader is handed one hard-coded sheet name.",
			File:          "internal/adminapi/users_import.go",
			Symbol:        "Sheet1",
			ChecklistStep: "Rename the client's worksheet to Sheet1, or export CSV or line-delimited JSON instead.",
		},
		{
			ID:            "three-input-formats",
			Statement:     "The shared file reader dispatches on the filename suffix across JSON, CSV and Excel.",
			File:          "internal/webserver/server.go",
			Symbol:        "ReadImportExcelData",
			ChecklistStep: "Pick the format the client's system exports cleanly; CSV is the least lossy for a hand-built mapping table.",
		},
		{
			ID:            "import-needs-admin-level",
			Statement:     "The import route is registered with a gate that admits only super and admin operators.",
			File:          "internal/adminapi/users.go",
			Symbol:        "importRadiusUsers",
			ChecklistStep: "The migration runs under an admin account, not the operator login the client's staff uses for routine work.",
		},
		{
			ID:            "tenant-comes-from-the-session",
			Statement:     "A create callback stamps the authenticated tenant onto every inserted record and refuses a mismatch.",
			File:          "internal/tenancy/scope.go",
			Symbol:        "scopeCreate",
			ChecklistStep: "Import while logged in to the destination tenant; the file cannot place rows in another tenant.",
		},
		{
			ID:            "username-is-unique-per-tenant",
			Statement:     "The database enforces one username per tenant, not one username globally.",
			File:          "internal/domain/radius.go",
			Symbol:        "udx_radius_user_tenant_username",
			ChecklistStep: "Two tenants may hold the same login name, so a shared-namespace client must decide the naming scheme first.",
		},
		{
			ID:            "password-is-stored-as-supplied",
			Statement:     "The domain record carries the password field the server later compares.",
			File:          "internal/domain/radius.go",
			Symbol:        "Password",
			ChecklistStep: "Treat the migration file as cleartext credential material: encrypted transport, named handover, deletion after acceptance.",
		},
		{
			ID:            "atomicity-is-pinned-by-a-test",
			Statement:     "A CI test asserts that a file with four bad rows reports zero successes and that no row from it reached the database.",
			File:          "internal/adminapi/users_import_test.go",
			Symbol:        "TestImportRadiusUsers_Errors",
			ChecklistStep: "The bad-file-rejects-without-change guarantee is regression-tested, not merely documented.",
		},
		{
			ID:            "preview-is-designed-not-deployed",
			Statement:     "The ENG-059 audit records the dry-run preview as awaiting approval and not implemented.",
			File:          "docs/radius-user-import-audit-2026-10-29.md",
			Symbol:        "not implemented",
			ChecklistStep: "Quote the audit's preview design to the client as the proposal, and do not demo it as a feature until the checklist approves it.",
		},
	}
}

// Absences returns the negative claims: what must not exist for ECO-006 to be a
// gap to close rather than a capability to document.
func Absences() []Absence {
	return []Absence{
		{
			ID:        "no-import-dry-run",
			Statement: "The user import has no dry-run or preview flag.",
			Symbol:    "dry_run",
			Scope:     "internal/adminapi/users_import.go",
			Gap:       "The operator cannot see what a file will do before applying it, which is the first thing a paid migration service needs.",
		},
		{
			ID:        "no-hash-import",
			Statement: "The user import has no path for a hashed password.",
			Symbol:    "Hash",
			Scope:     "internal/adminapi/users_import.go",
			Gap:       "A client whose old system exports hashes rather than passwords cannot be migrated at all without a re-keying step.",
		},
		{
			ID:        "no-conflict-update-import",
			Statement: "The user import never uses an upsert, so it can only add accounts.",
			Symbol:    "OnConflict",
			Scope:     "internal/adminapi/users_import.go",
			Gap:       "Re-running a migration after a partial correction cannot refresh existing rows; the operator must delete and re-add.",
		},
		{
			ID:        "no-package-import",
			Statement: "No import handler exists for billing profiles or packages.",
			Symbol:    "importRadiusProfiles",
			Scope:     "internal/adminapi/",
			Gap:       "The plan catalog, which every customer row depends on, is rebuilt by hand before the customer file can even be validated.",
		},
		{
			ID:        "no-node-import",
			Statement: "No import handler exists for network nodes or NAS devices.",
			Symbol:    "importNetNodes",
			Scope:     "internal/adminapi/",
			Gap:       "Site inventory is a manual precondition of the migration, so the effort estimate must count it separately.",
		},
	}
}

// Verify reads the files the claims cite and checks each one. A missing file is
// reported as broken with its path, because a citation to a moved file is exactly
// the kind of drift that lets a stale service description survive a release.
func Verify(root string) (Report, error) {
	var report Report
	for _, claim := range Claims() {
		finding := Finding{Claim: claim, Note: claim.ChecklistStep}
		//nolint:gosec // G304: paths from the catalog, resolved under the repository root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claim.File)))
		if err != nil {
			finding.Status = StatusBroken
			finding.Evidence = "file unreadable"
			report.Findings = append(report.Findings, finding)
			continue
		}
		lines := codeLines(claim.File, string(raw))
		if at := locate(lines, claim.Symbol); at > 0 {
			finding.Status = StatusConfirmed
			finding.Evidence = fmt.Sprintf("%s:%d", claim.File, at)
		} else {
			finding.Status = StatusBroken
			finding.Evidence = fmt.Sprintf("symbol %q not in %s", claim.Symbol, claim.File)
		}
		report.Findings = append(report.Findings, finding)
	}

	for _, absence := range Absences() {
		found, err := scanFor(root, absence)
		if err != nil {
			return Report{}, err
		}
		status := StatusConfirmed
		if len(found) > 0 {
			status = StatusVacated
		}
		report.Absences = append(report.Absences, AbsenceFinding{Absence: absence, Status: status, Hits: found})
	}

	var err error
	report.Surface, err = MeasureSurface(root)
	if err != nil {
		return Report{}, err
	}
	report.Precedents, err = FindDryRunPrecedents(root)
	if err != nil {
		return Report{}, err
	}
	report.ImportTests, err = FindImportTests(root)
	if err != nil {
		return Report{}, err
	}
	report.ImportSurfaces, err = FindImportCallSites(root)
	if err != nil {
		return Report{}, err
	}

	report.Totals = Totals{
		Claims:       len(report.Findings),
		Absences:     len(report.Absences),
		FilesRead:    len(report.Findings),
		ColumnFields: len(report.Surface.Columns),
		RejectRules:  len(report.Surface.Rejections),
		Precedents:   len(report.Precedents),
		ImportTests:  len(report.ImportTests),
	}
	for _, f := range report.Findings {
		if f.Status == StatusConfirmed {
			report.Totals.Confirmed++
		}
	}
	for _, a := range report.Absences {
		if a.Status == StatusConfirmed {
			report.Totals.AbsencesHeld++
		}
	}
	sort.SliceStable(report.Findings, func(i, j int) bool { return report.Findings[i].Claim.ID < report.Findings[j].Claim.ID })
	return report, nil
}

// Totals summarizes one verification run.
type Totals struct {
	// Claims counts the positive statements checked.
	Claims int
	// Confirmed counts claims whose symbol was found in the cited file.
	Confirmed int
	// Absences counts the negative claims checked.
	Absences int
	// AbsencesHeld counts negative claims still true.
	AbsencesHeld int
	// FilesRead counts cited files.
	FilesRead int
	// ColumnFields counts the import columns the validator reads.
	ColumnFields int
	// RejectRules counts the distinct reasons a row can be refused.
	RejectRules int
	// Precedents counts non-mutating preflight implementations elsewhere in the
	// product code.
	Precedents int
	// ImportTests counts the named tests that pin the import contract.
	ImportTests int
}

// Report is one verification run.
type Report struct {
	// Findings is ordered by claim ID.
	Findings []Finding
	// Absences is ordered as Absences returns.
	Absences []AbsenceFinding
	// Surface is the measured import vocabulary.
	Surface MigrationSurface
	// Precedents lists files that already implement a dry run.
	Precedents []string
	// ImportTests lists the import-related test names CI runs.
	ImportTests []string
	// ImportSurfaces lists the handlers that take an uploaded file.
	ImportSurfaces []string
	// Totals summarizes both halves.
	Totals Totals
}

// Verdict returns the status carried by a claim or absence ID, so a work item can
// be sized only while the evidence behind it holds.
func (r Report) Verdict(id string) Status {
	for _, f := range r.Findings {
		if f.Claim.ID == id {
			return f.Status
		}
	}
	for _, a := range r.Absences {
		if a.Absence.ID == id {
			return a.Status
		}
	}
	return StatusBroken
}

// locate returns the first line number (one-based) whose identifier tokens
// include name, or 0. Matching whole tokens keeps a longer name from proving a
// shorter claim.
func locate(lines []string, name string) int {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	for i, line := range lines {
		if re.MatchString(line) {
			return i + 1
		}
	}
	return 0
}

// codeLines splits text into lines and blanks the ones that are prose. A doc
// comment naming an endpoint proves the comment exists, not that the code does,
// so a claim must be carried by an executable line. Go comment markers are
// stripped; other file types are kept as written.
func codeLines(name, text string) []string {
	lines := strings.Split(text, "\n")
	if !strings.HasSuffix(name, ".go") {
		return lines
	}
	inBlock := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		switch {
		case inBlock:
			lines[i] = ""
			if strings.HasPrefix(trimmed, "*/") {
				inBlock = false
			}
		case strings.HasPrefix(trimmed, "/*"):
			lines[i] = ""
			if !strings.Contains(trimmed, "*/") {
				inBlock = true
			}
		case strings.HasPrefix(trimmed, "//"), strings.HasPrefix(trimmed, "*"):
			lines[i] = ""
		default:
			lines[i] = stripTrailingComment(line)
		}
	}
	return lines
}

// stripTrailingComment cuts a trailing Go comment at the first // that is not
// inside a string literal. A URL in a string must survive the cut.
func stripTrailingComment(line string) string {
	for i := 0; i < len(line); i++ {
		if line[i] != '/' || i+1 >= len(line) || line[i+1] != '/' {
			continue
		}
		if strings.Count(line[:i], `"`)%2 == 1 {
			continue
		}
		return line[:i]
	}
	return line
}

// scanFor searches the scope, which may be one file or a directory prefix, for a
// symbol that must be absent. The prototype's own package is skipped because its
// catalog quotes the symbols it hunts for.
func scanFor(root string, absence Absence) ([]string, error) {
	base := filepath.Join(root, filepath.FromSlash(absence.Scope))
	if _, err := os.Stat(base); err != nil {
		return nil, fmt.Errorf("migrationcheck: absence scope %s: %w", absence.Scope, err)
	}
	var hits []string
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(absence.Symbol))
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if !strings.HasSuffix(slash, ".go") || strings.HasPrefix(slash, "internal/migrationcheck/") {
			return nil
		}
		//nolint:gosec // G304: paths walked inside the repository under review.
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range codeLines(slash, string(raw)) {
			if re.MatchString(line) {
				hits = append(hits, fmt.Sprintf("%s:%d", slash, i+1))
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("migrationcheck: scanning %s: %w", absence.Scope, err)
	}
	sort.Strings(hits)
	if len(hits) > maxHits {
		hits = hits[:maxHits]
	}
	return hits, nil
}
