package migrationcheck

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// repoRoot returns the repository root as seen from this test file.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

func TestClaimCatalogIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Claims() {
		if c.ID == "" || c.Statement == "" || c.File == "" || c.Symbol == "" || c.ChecklistStep == "" {
			t.Errorf("claim %q is incomplete", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("duplicate claim id %q", c.ID)
		}
		seen[c.ID] = true
		if filepath.IsAbs(c.File) {
			t.Errorf("claim %q cites an absolute path", c.ID)
		}
	}
	for _, a := range Absences() {
		if a.ID == "" || a.Statement == "" || a.Symbol == "" || a.Scope == "" || a.Gap == "" {
			t.Errorf("absence %q is incomplete", a.ID)
		}
		if seen[a.ID] {
			t.Errorf("absence %q reuses a claim id", a.ID)
		}
		seen[a.ID] = true
	}
}

// TestChecklistCoversEveryClaim is the guard that keeps a new claim from being
// written and then left out of the operator checklist.
func TestChecklistCoversEveryClaim(t *testing.T) {
	order := checklistOrder()
	if len(order) != len(Claims()) {
		t.Fatalf("checklist has %d steps for %d claims", len(order), len(Claims()))
	}
	byID := map[string]bool{}
	for _, c := range Claims() {
		byID[c.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range order {
		if !byID[id] {
			t.Errorf("checklist orders unknown claim %q", id)
			continue
		}
		if seen[id] {
			t.Errorf("checklist orders %q twice", id)
		}
		seen[id] = true
	}
	for _, c := range Claims() {
		if !seen[c.ID] {
			t.Errorf("claim %q never reaches the checklist", c.ID)
		}
	}
}

func TestEveryCitedFileIsInTheRepository(t *testing.T) {
	root := repoRoot(t)
	for _, c := range Claims() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.File))); err != nil {
			t.Errorf("claim %q cites %s, which is not in the tree", c.ID, c.File)
		}
	}
	for _, a := range Absences() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(a.Scope))); err != nil {
			t.Errorf("absence %q scans %s, which is not in the tree", a.ID, a.Scope)
		}
	}
}

func TestRealTreeConfirmsEveryClaim(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed {
			t.Errorf("claim %s is %s: %s", f.Claim.ID, f.Status, f.Evidence)
		}
	}
	if report.Totals.Confirmed != report.Totals.Claims {
		t.Errorf("confirmed %d of %d claims", report.Totals.Confirmed, report.Totals.Claims)
	}
}

// TestRealTreeEveryGapHolds keeps the criterion honest: an absence that stops
// being true means the service already gained the capability and this study is
// the stale document.
func TestRealTreeEveryGapHolds(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, a := range report.Absences {
		if a.Status != StatusConfirmed {
			t.Errorf("gap %s is %s, found at %v", a.Absence.ID, a.Status, a.Hits)
		}
	}
}

// TestEveryEvidenceLineIsCode guards the citation rule: a verdict must be carried
// by an executable line, never by a comment that merely names the symbol.
func TestEveryEvidenceLineIsCode(t *testing.T) {
	root := repoRoot(t)
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed || !strings.HasSuffix(f.Claim.File, ".go") {
			continue
		}
		//nolint:gosec // G304: paths taken from the report this test just built.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Claim.File)))
		if err != nil {
			t.Fatalf("reading %s: %v", f.Claim.File, err)
		}
		hunks := strings.Split(string(raw), "\n")
		_, lineText, ok := strings.Cut(f.Evidence, f.Claim.File+":")
		if !ok {
			t.Fatalf("evidence %q does not name its file", f.Evidence)
		}
		n, convErr := strconv.Atoi(lineText)
		if convErr != nil || n < 1 || n > len(hunks) {
			t.Fatalf("%s evidence %q is not a line number", f.Claim.File, f.Evidence)
		}
		trimmed := strings.TrimSpace(hunks[n-1])
		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "*") {
			t.Errorf("claim %s rests on the prose line %d of %s", f.Claim.ID, n, f.Claim.File)
		}
	}
}

// TestMovedFileIsReportedBrokenNotDropped proves a citation to a renamed file
// surfaces instead of disappearing, because that drift is how a stale service
// description survives a release.
func TestMovedFileIsReportedBrokenNotDropped(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"internal/adminapi", "internal/webserver", "internal/tenancy", "internal/domain", "internal/networkmonitor", "docs", "test/integration"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	// The files that absence scans must read exist, but carry none of the cited
	// symbols; the handler files are gone entirely.
	for _, rel := range []string{"internal/adminapi/users_import.go", "internal/adminapi/users.go", "internal/adminapi/users_import_test.go", "internal/webserver/server.go", "internal/tenancy/scope.go", "internal/domain/radius.go", "test/integration/adminapi_test.go"} {
		dir := filepath.Dir(filepath.Join(root, filepath.FromSlash(rel)))
		if err := os.MkdirAll(dir, 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), []byte("package placeholder\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	unreadable, movedSymbol := 0, 0
	for _, f := range report.Findings {
		switch {
		case f.Status != StatusBroken:
			t.Errorf("claim %s survived an emptied tree as %s", f.Claim.ID, f.Status)
		case strings.Contains(f.Evidence, "file unreadable"):
			unreadable++
		default:
			movedSymbol++
		}
	}
	if unreadable == 0 || movedSymbol == 0 {
		t.Errorf("both drift kinds should be reported, got %d unreadable and %d missing-symbol", unreadable, movedSymbol)
	}
}

func TestAbsenceScanIgnoresThePrototypeItself(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"internal/adminapi", "internal/migrationcheck"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(dir)), 0o750); err != nil {
			t.Fatal(err)
		}
	}
	// The prototype's own catalog quotes every symbol it hunts for.
	quote := "package migrationcheck\n\n// dry_run OnConflict importRadiusProfiles importNetNodes Hash\n"
	if err := os.WriteFile(filepath.Join(root, "internal", "migrationcheck", "claims.go"), []byte(quote), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "adminapi", "users_import.go"), []byte("package adminapi\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, a := range Absences() {
		found, err := scanFor(root, a)
		if err != nil {
			t.Fatalf("scanFor %s: %v", a.ID, err)
		}
		if len(found) != 0 {
			t.Errorf("absence %s matched the prototype's own words: %v", a.ID, found)
		}
	}
}

func TestCommentMentionsAreNotEvidence(t *testing.T) {
	text := "package adminapi\n\n// planUserImport is documented here but never defined.\nfunc other() {}\n"
	if got := locate(codeLines("internal/adminapi/users_import.go", text), "planUserImport"); got != 0 {
		t.Errorf("a comment proved the claim at line %d", got)
	}
	def := "package adminapi\n\nfunc planUserImport() {}\n"
	if got := locate(codeLines("internal/adminapi/users_import.go", def), "planUserImport"); got != 3 {
		t.Errorf("the definition was not found, got %d", got)
	}
}

func TestTrailingCommentIsNotEvidenceButAURLIsKept(t *testing.T) {
	line := `	value := c.QueryParam("second") // dry_run is spelled here once`
	if got := locate([]string{stripTrailingComment(line)}, "dry_run"); got != 0 {
		t.Errorf("a trailing comment proved the claim at line %d", got)
	}
	if got := locate([]string{stripTrailingComment(line)}, "QueryParam"); got != 1 {
		t.Errorf("the code part was cut too: %d", got)
	}
	kept := `	url := "https://example.com/a"`
	if got := stripTrailingComment(kept); !strings.Contains(got, "example.com") {
		t.Errorf("a URL inside a string was cut: %q", got)
	}
	if stripTrailingComment("// see https://example.com/a for the contract") != "" {
		t.Error("a comment-only line must blank out")
	}
}

func TestFindingsAreOrderedByClaimID(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for i := 1; i < len(report.Findings); i++ {
		if report.Findings[i-1].Claim.ID > report.Findings[i].Claim.ID {
			t.Errorf("findings out of order at %d", i)
		}
	}
}

func TestFindingStringCarriesVerdictAndEvidence(t *testing.T) {
	f := Finding{Claim: Claim{ID: "x", File: "a.go"}, Status: StatusConfirmed, Evidence: "a.go:3"}
	if got, want := f.String(), "x confirmed [a.go] a.go:3"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	a := AbsenceFinding{Absence: Absence{ID: "g", Symbol: "dry_run", Scope: "internal/"}, Status: StatusVacated, Hits: []string{"b.go:1"}}
	if got, want := a.String(), "g vacated b.go:1"; got != want {
		t.Errorf("got %q want %q", got, want)
	}
	bare := AbsenceFinding{Absence: Absence{ID: "g", Symbol: "dry_run", Scope: "internal/"}, Status: StatusConfirmed}
	if !strings.Contains(bare.String(), `no "dry_run"`) {
		t.Errorf("a held absence should name what it did not find: %q", bare.String())
	}
}

// TestNoClaimPromisesClientResearch keeps the study inside what the tree can
// prove. The ECO-006 criterion also asks for interviews with three prospective
// clients, which is a human-asset step this package cannot perform, so no claim
// may read as though it has been.
func TestNoClaimPromisesClientResearch(t *testing.T) {
	banned := []string{"interview", "prospective client", "calon klien", "signed off", "customer agreed", "engagement price"}
	for _, c := range Claims() {
		hay := strings.ToLower(c.Statement + " " + c.ChecklistStep)
		for _, word := range banned {
			if strings.Contains(hay, word) {
				t.Errorf("claim %s asserts %q, which source cannot prove", c.ID, word)
			}
		}
	}
	for _, a := range Absences() {
		hay := strings.ToLower(a.Statement + " " + a.Gap)
		for _, word := range banned {
			if strings.Contains(hay, word) {
				t.Errorf("absence %s asserts %q, which source cannot prove", a.ID, word)
			}
		}
	}
}

func TestMissingTreeIsAnError(t *testing.T) {
	if _, err := Verify(filepath.Join(t.TempDir(), "nope")); err == nil {
		t.Fatal("expected a missing tree to fail rather than report clean")
	}
}
