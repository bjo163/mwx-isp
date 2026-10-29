package migrationcheck

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Work sizes are t-shirt labels. No hour or currency figure is claimed here: the
// master inventory starts every numerical baseline at UNKNOWN until measured, and
// a prototype that reads source cannot price a client engagement.
const (
	SizeSmall   = "S"
	SizeMedium  = "M"
	SizeLarge   = "L"
	SizeUnknown = "unknown"
)

// WorkItem is one piece of work the migration service needs, either in the code
// base or in the engagement process.
type WorkItem struct {
	// ID names the item in reports.
	ID string
	// Title is the work to be done.
	Title string
	// Requires lists the claim and absence IDs that decide the item.
	Requires []string
	// Count is the measured quantity the size is derived from.
	Count int
	// Driver explains what Count measures.
	Driver string
	// Small and Medium are the inclusive thresholds that turn Count into a size.
	Small  int
	Medium int
}

// Size returns the item's t-shirt size, or SizeUnknown when any requirement
// failed to verify. An item whose evidence moved is not silently downsized.
func (w WorkItem) Size(verdict func(id string) Status) string {
	for _, id := range w.Requires {
		if verdict(id) != StatusConfirmed {
			return SizeUnknown
		}
	}
	switch {
	case w.Count <= w.Small:
		return SizeSmall
	case w.Count <= w.Medium:
		return SizeMedium
	default:
		return SizeLarge
	}
}

// ChecklistStep is one operator-facing line of the migration checklist, tied to
// the claim that decides it.
type ChecklistStep struct {
	// Sequence is the 1-based position an operator would perform it in.
	Sequence int
	// ClaimID is the claim the step rests on.
	ClaimID string
	// Text is the step, taken from the claim's own ChecklistStep.
	Text string
	// Evidence is the file:line that proves the step, or the failure that voids
	// it.
	Evidence string
	// Verified tells whether the step still has support in the tree.
	Verified bool
}

// checklistOrder is the sequence a migration is actually performed in: agree the
// handover, build the dependencies, prepare the file, gate the run, verify after.
// A test pins it to the catalog so no claim can drift out of the checklist.
func checklistOrder() []string {
	return []string{
		"preview-is-designed-not-deployed",
		"import-needs-admin-level",
		"tenant-comes-from-the-session",
		"username-is-unique-per-tenant",
		"cleartext-password-is-mandatory",
		"password-is-stored-as-supplied",
		"packages-before-customers",
		"nodes-before-customers",
		"three-input-formats",
		"excel-sheet-name-is-fixed",
		"status-vocabulary-is-two-values",
		"expire-defaults-one-year",
		"profile-inherits-six-fields",
		"duplicate-rows-in-one-file-block",
		"existing-user-blocks-the-row",
		"validation-runs-before-any-write",
		"one-transaction-per-file",
		"rollback-is-told-to-the-operator",
		"per-row-errors-are-addressable",
		"atomicity-is-pinned-by-a-test",
	}
}

// BuildChecklist pairs each claim's checklist text with its verdict, in the order
// a migration is performed.
func BuildChecklist(report Report) []ChecklistStep {
	byID := map[string]Finding{}
	for _, f := range report.Findings {
		byID[f.Claim.ID] = f
	}
	var steps []ChecklistStep
	for i, id := range checklistOrder() {
		finding, ok := byID[id]
		if !ok {
			continue
		}
		steps = append(steps, ChecklistStep{
			Sequence: i + 1,
			ClaimID:  id,
			Text:     finding.Claim.ChecklistStep,
			Evidence: finding.Evidence,
			Verified: finding.Status == StatusConfirmed,
		})
	}
	return steps
}

// WorkItems returns the work the service needs, each sized from a quantity this
// package measured rather than from an opinion.
func WorkItems(report Report) []WorkItem {
	surface := report.Surface
	manual := 0
	for _, a := range report.Absences {
		switch a.Absence.ID {
		case "no-package-import", "no-node-import":
			if a.Status == StatusConfirmed {
				manual++
			}
		}
	}
	overlap := 0
	for _, r := range surface.Rejections {
		if strings.Contains(strings.ToLower(r), "exist") || strings.Contains(strings.ToLower(r), "duplicate") {
			overlap++
		}
	}
	return []WorkItem{
		{
			ID:       "map-client-export",
			Title:    "Map the client's customer export onto the accepted columns",
			Requires: []string{"three-input-formats", "excel-sheet-name-is-fixed", "status-vocabulary-is-two-values"},
			Count:    len(surface.Columns),
			Driver:   fmt.Sprintf("%d columns read through %d accepted header spellings", len(surface.Columns), surface.HeaderCount()),
			Small:    18,
			Medium:   30,
		},
		{
			ID:       "prebuild-dependencies",
			Title:    "Rebuild the plan catalog and node inventory by hand before the customer file",
			Requires: []string{"packages-before-customers", "nodes-before-customers"},
			Count:    manual,
			Driver: fmt.Sprintf("%d dependent entity types have no import surface today (%d upload handler exists in the whole admin API)",
				manual, len(report.ImportSurfaces)),
			Small:  1,
			Medium: 2,
		},
		{
			ID:       "decide-overlap-policy",
			Title:    "Agree the existing-account and re-run policy with the client",
			Requires: []string{"existing-user-blocks-the-row", "no-conflict-update-import"},
			Count:    overlap,
			Driver:   fmt.Sprintf("%d rejection reasons are collision rules the file cannot express", overlap),
			Small:    1,
			Medium:   3,
		},
		{
			ID:       "hand-over-credentials",
			Title:    "Run the cleartext password handover as a controlled step",
			Requires: []string{"cleartext-password-is-mandatory", "password-is-stored-as-supplied", "no-hash-import"},
			Count:    len(surface.Rejections),
			Driver:   fmt.Sprintf("%d distinct row rejections decide what the handover file must contain", len(surface.Rejections)),
			Small:    8,
			Medium:   14,
		},
		{
			ID:       "add-dry-run-preview",
			Title:    "Wire the proposed dry-run preview behind an approval",
			Requires: []string{"no-import-dry-run", "preview-is-designed-not-deployed"},
			Count:    len(surface.ConditionalFields) + len(surface.ProfileFields) + len(surface.Defaults),
			Driver: fmt.Sprintf("%d row values are inherited or defaulted today, and a preview must surface each one (%d non-mutating preflight implementations already exist to copy)",
				len(surface.ConditionalFields)+len(surface.ProfileFields)+len(surface.Defaults), len(report.Precedents)),
			Small:  8,
			Medium: 14,
		},
		{
			ID:       "prove-no-change-on-bad-file",
			Title:    "Keep the bad-file-rejects-without-change guarantee under regression",
			Requires: []string{"validation-runs-before-any-write", "atomicity-is-pinned-by-a-test"},
			Count:    len(report.ImportTests),
			Driver:   fmt.Sprintf("%d named tests already pin the import's failure behavior", len(report.ImportTests)),
			Small:    4,
			Medium:   8,
		},
	}
}

// dryRunRE matches a non-mutating preflight in either spelling the tree uses.
var dryRunRE = regexp.MustCompile(`(?i)dry.?run`)

// FindDryRunPrecedents lists the files outside the import handler that already
// implement a dry run, so the preview's size is quoted against reusable code
// rather than against a guess.
func FindDryRunPrecedents(root string) ([]string, error) {
	base := filepath.Join(root, "internal")
	if _, err := os.Stat(base); err != nil {
		return nil, fmt.Errorf("migrationcheck: precedents: %w", err)
	}
	var found []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if !strings.HasSuffix(slash, ".go") || strings.HasSuffix(slash, "_test.go") ||
			strings.HasPrefix(slash, "internal/migrationcheck/") || slash == importFile {
			return nil
		}
		//nolint:gosec // G304: paths walked inside the repository under review.
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, line := range codeLines(slash, string(raw)) {
			if dryRunRE.MatchString(line) {
				found = append(found, slash)
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("migrationcheck: scanning precedents: %w", err)
	}
	sort.Strings(found)
	return found, nil
}

// importCallRE matches a handler that reads an uploaded file through the shared
// reader, i.e. an entity type the product can import.
var importCallRE = regexp.MustCompile(`webserver\.ImportData\(`)

// FindImportCallSites lists every place the product takes an import file. One
// entry means one importable entity type, and the rest of a migration is manual
// work the effort estimate has to count separately.
func FindImportCallSites(root string) ([]string, error) {
	base := filepath.Join(root, "internal")
	if _, err := os.Stat(base); err != nil {
		return nil, fmt.Errorf("migrationcheck: import sites: %w", err)
	}
	var found []string
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if !strings.HasSuffix(slash, ".go") || strings.HasSuffix(slash, "_test.go") ||
			strings.HasPrefix(slash, "internal/migrationcheck/") {
			return nil
		}
		//nolint:gosec // G304: paths walked inside the repository under review.
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range codeLines(slash, string(raw)) {
			if importCallRE.MatchString(line) {
				found = append(found, fmt.Sprintf("%s:%d", slash, i+1))
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("migrationcheck: scanning import sites: %w", err)
	}
	sort.Strings(found)
	return found, nil
}

// importTestRE matches the name given to a test function.
var importTestRE = regexp.MustCompile(`func (Test[A-Za-z0-9_]*Import[A-Za-z0-9_]*)\(`)

// importTestFiles are the CI files that pin the import contract.
var importTestFiles = []string{
	"internal/adminapi/users_import_test.go",
	"test/integration/adminapi_test.go",
}

// FindImportTests lists the import-related test function names CI runs, which is
// the evidence that the bad-file guarantee is regression-protected rather than
// merely described.
func FindImportTests(root string) ([]string, error) {
	var names []string
	for _, rel := range importTestFiles {
		//nolint:gosec // G304: paths from the catalog, resolved under the repository root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("migrationcheck: %s: %w", rel, err)
		}
		for _, m := range importTestRE.FindAllStringSubmatch(string(raw), -1) {
			names = append(names, rel+"#"+m[1])
		}
	}
	sort.Strings(names)
	return names, nil
}
