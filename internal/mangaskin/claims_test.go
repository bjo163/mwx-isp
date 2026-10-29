package mangaskin

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate the test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", ".."))
}

// writeFile drops a fixture under a temp root, with the mode gosec requires for
// test data.
func writeFile(t *testing.T, root, rel, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatalf("mkdir %s: %v", rel, err)
	}
	if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}

func TestClaimCatalogIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Claims() {
		if c.ID == "" || c.Statement == "" || c.File == "" || c.Symbol == "" {
			t.Errorf("claim %+v is missing a field", c)
		}
		if seen[c.ID] {
			t.Errorf("claim ID %s is declared twice", c.ID)
		}
		seen[c.ID] = true
	}
	if len(Claims()) < 10 {
		t.Errorf("the direction argument needs more than %d claims", len(Claims()))
	}
}

func TestAbsenceCatalogIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, a := range Absences() {
		if a.ID == "" || a.Statement == "" || a.Symbol == "" || a.Scope == "" || a.Gap == "" {
			t.Errorf("absence %+v is missing a field", a)
		}
		if seen[a.ID] {
			t.Errorf("absence ID %s is declared twice", a.ID)
		}
		seen[a.ID] = true
	}
}

func TestEveryClaimAboutTheShippedSkinConfirms(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed {
			t.Errorf("claim %s is %s: %s", f.Claim.ID, f.Status, f.Evidence)
		}
	}
}

// TestEveryCitationLandsOnACodeLine is the rule that stops a doc comment from
// proving a color: each recorded evidence line must be a line the file executes.
func TestEveryCitationLandsOnACodeLine(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed {
			continue
		}
		at := strings.LastIndex(f.Evidence, ":")
		if at < 0 || !strings.HasPrefix(f.Evidence, f.Claim.File) {
			t.Errorf("claim %s cites %q, not a file:line", f.Claim.ID, f.Evidence)
			continue
		}
		n, err := strconv.Atoi(f.Evidence[at+1:])
		if err != nil {
			t.Errorf("claim %s cites an unparsable line %q", f.Claim.ID, f.Evidence)
			continue
		}
		//nolint:gosec // G304: path from the claim catalog, resolved under the repository root.
		raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(f.Claim.File)))
		if err != nil {
			t.Errorf("claim %s: %v", f.Claim.ID, err)
			continue
		}
		lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
		if n < 1 || n > len(lines) {
			t.Errorf("claim %s cites line %d of a %d-line file", f.Claim.ID, n, len(lines))
			continue
		}
		line := strings.TrimSpace(lines[n-1])
		if strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*") || strings.HasPrefix(line, "*") {
			t.Errorf("claim %s is proved by a comment: %s:%d %q", f.Claim.ID, f.Claim.File, n, line)
		}
		if !containsBounded(line, f.Claim.Symbol) {
			t.Errorf("claim %s cites %s:%d which does not hold %q: %q",
				f.Claim.ID, f.Claim.File, n, f.Claim.Symbol, line)
		}
	}
}

// TestCommentMentionsAreNotEvidence builds a theme file that only names the
// symbols in prose and checks every claim over it comes back broken.
func TestCommentMentionsAreNotEvidence(t *testing.T) {
	root := t.TempDir()
	var b strings.Builder
	b.WriteString("// inkShadow, inkLine, halftone, accessibleAccent, accentColor, fontStacks.display\n")
	b.WriteString("// fontSize: 13, borderRadius: 3, MuiAppBar, prefers-reduced-motion, Restore defaults\n")
	b.WriteString("const lightPalette = {\n  primary: { main: '#5d7000' },\n};\n")
	writeFile(t, root, themeFile, b.String())
	for _, claim := range Claims() {
		if claim.File != themeFile {
			continue
		}
		//nolint:gosec // G304: fixture path built from the test's own temp root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claim.File)))
		if err != nil {
			t.Fatalf("read fixture: %v", err)
		}
		lines := strings.Split(blankComments(string(raw)), "\n")
		if locate(lines, claim.Symbol) > 0 {
			t.Errorf("a doc comment naming %q was accepted as evidence", claim.Symbol)
		}
	}
}

// TestMovedFileIsReportedBrokenNotDropped keeps both kinds of drift loud: a cited
// file that disappeared, and a symbol that moved out of the line it cited.
func TestMovedFileIsReportedBrokenNotDropped(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, themeFile, "const lightPalette = {\n  primary: { main: '#5d7000' },\n};\n")
	report, err := VerifyClaims(root)
	if err != nil {
		t.Fatalf("VerifyClaims on a partial tree must still report: %v", err)
	}
	kinds := map[Status]int{}
	for _, f := range report.Findings {
		kinds[f.Status]++
		if f.Status == StatusConfirmed {
			t.Errorf("claim %s confirmed against a stripped fixture at %s", f.Claim.ID, f.Evidence)
		}
	}
	if kinds[StatusBroken] != len(Claims()) {
		t.Errorf("every claim should come back broken on a stripped tree, got %v", kinds)
	}
}

func TestBoundedMatchCannotBeProvedByALongerName(t *testing.T) {
	lines := []string{"const ChipLabel = 1;"}
	if locate(lines, "Chip") > 0 {
		t.Error("ChipLabel was accepted as proof that Chip is used")
	}
	lines = []string{"<Chip size=\"small\" />"}
	if locate(lines, "Chip") == 0 {
		t.Error("the component itself was not matched")
	}
	if locate([]string{"  borderRadius: 3,"}, "borderRadius: 3") == 0 {
		t.Error("a spaced declaration was not matched")
	}
	if locate([]string{"// a note"}, "note") == 0 {
		t.Error("locate must work on whatever lines the caller blanked")
	}
}

func TestAbsenceScanIgnoresThePrototypeItself(t *testing.T) {
	found, err := scanFor(repoRoot(t), "web/src", "inkShadow")
	if err != nil {
		t.Fatalf("scanFor: %v", err)
	}
	if len(found) == 0 {
		t.Fatal("the scan should find the real inkShadow expression in theme.ts")
	}
	for _, f := range found {
		if strings.HasPrefix(f, "internal/mangaskin/") {
			t.Errorf("the scan matched the prototype's own source: %s", f)
		}
	}
}

func TestEveryGapStaysAGap(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, a := range report.Absences {
		if a.Status != StatusConfirmed {
			t.Errorf("absence %s is %s: %v - the memo needs rechecking", a.Absence.ID, a.Status, a.Hits)
		}
	}
}

func TestVacatedGapIsReportedWithItsHit(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "web/src/pages/Forced.tsx", "export const x = '@media (forced-colors: active)';\n")
	found, err := scanFor(root, "web/src", "forced-colors")
	if err != nil {
		t.Fatalf("scanFor: %v", err)
	}
	if len(found) != 1 || !strings.HasPrefix(found[0], "web/src/pages/Forced.tsx:") {
		t.Fatalf("a closed gap must report where it closed, got %v", found)
	}
}

func TestNoClaimPromisesAShippedSkin(t *testing.T) {
	banned := regexp.MustCompile(`(?i)\b(ship|deploy|rollout|release|interview|price|quote the client)\b`)
	for _, c := range Claims() {
		if banned.MatchString(c.Statement) || banned.MatchString(c.Note) {
			t.Errorf("claim %s speaks as if the prototype were product work: %q", c.ID, c.Statement)
		}
	}
	for _, a := range Absences() {
		if banned.MatchString(a.Statement) || banned.MatchString(a.Gap) {
			t.Errorf("absence %s speaks as if the prototype were product work: %q", a.ID, a.Statement)
		}
	}
}
