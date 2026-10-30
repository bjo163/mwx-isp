package contextualws

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// repoRoot walks up from the package directory to the module root, so the study
// always reads the tree it lives in.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.ToSlash(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", wd)
		}
		dir = parent
	}
}

var (
	runOnce   *Result
	runErr    error
	studyOnce sync.Once
)

func study(t *testing.T) *Result {
	t.Helper()
	studyOnce.Do(func() { runOnce, runErr = Run(repoRoot(t)) })
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	return runOnce
}

// TestEveryStatementHolds is the gate the criterion asks for: a memo sentence that
// no longer lands on a code line fails the build instead of staying in print.
func TestEveryStatementHolds(t *testing.T) {
	res := study(t)
	if len(res.Claims) == 0 {
		t.Fatal("no statements to check")
	}
	for _, c := range res.Claims {
		if c.Status != confirmed {
			t.Errorf("%s not confirmed (%s at %s:%d): %s", c.ID, c.Status, c.File, c.Line, c.Text)
		}
	}
}

// TestTheDigestSurvivesATickingClock proves the study is a measurement of the
// tree, not of the moment it ran.
func TestTheDigestSurvivesATickingClock(t *testing.T) {
	root := repoRoot(t)
	first, err := Run(root)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := Run(root)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if first.Digest != second.Digest {
		t.Errorf("digest moved between runs: %s then %s", first.Digest, second.Digest)
	}
	if first.Totals.ClaimsConfirmed != second.Totals.ClaimsConfirmed {
		t.Errorf("the statement table changed size between runs")
	}
}

// TestRunningTheStudyWritesNothing is the read-only promise as a number: every
// file the study reads is checked for size and mtime before and after the run.
func TestRunningTheStudyWritesNothing(t *testing.T) {
	root := repoRoot(t)
	before := snapshot(t, root)
	if _, err := Run(root); err != nil {
		t.Fatalf("Run: %v", err)
	}
	after := snapshot(t, root)
	if before != after {
		t.Errorf("the corpus moved while the study ran")
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for _, rel := range corpusFiles(root) {
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			b.WriteString(rel + " gone\n")
			continue
		}
		b.WriteString(rel + "=" + itoa(int(st.Size())) + "@" + st.ModTime().String() + "\n")
	}
	return b.String()
}

// corpusFiles is everything the study reads, so the proof covers the whole input
// set rather than the handful of files the author happened to think of.
func corpusFiles(root string) []string {
	var out []string
	for _, dir := range []string{"web/src", "web/e2e", "internal/adminapi", ".github/workflows"} {
		out = append(out, walkFiles(root, dir, ".go", ".ts", ".tsx", ".yml")...)
	}
	return out
}

// TestTheStudyImportsNothingThatCanSendOrWrite closes the protocol half of the
// promise: the package may read files and parse them, and nothing else.
func TestTheStudyImportsNothingThatCanSendOrWrite(t *testing.T) {
	allowed := map[string]bool{
		"crypto/sha256": true, "encoding/hex": true, "fmt": true, "go/parser": true,
		"go/token": true, "io/fs": true, "os": true, "path/filepath": true,
		"regexp": true, "sort": true, "strconv": true, "strings": true,
		"testing": true, "sync": true,
		// The two non-stdlib paths are the test framework, and a test file is the
		// only place they appear: packageImports reads the whole package, tests
		// included, so the allowlist covers them rather than excluding them.
		"github.com/stretchr/testify/assert":  true,
		"github.com/stretchr/testify/require": true,
	}
	for _, imp := range packageImports(repoRoot(t), ownPackageDir(repoRoot(t))) {
		if !allowed[imp] {
			t.Errorf("import %q is outside the read-only allowlist", imp)
		}
	}
}

// TestNothingImportsTheStudy keeps the research package a study, not a feature.
func TestNothingImportsTheStudy(t *testing.T) {
	if got := importersOutside(repoRoot(t)); len(got) > 0 {
		t.Errorf("product code imports the study: %s", strings.Join(got, ", "))
	}
}

// TestTheMenuParserReadsEveryRow pins the corpus itself: a menu row the parser
// loses would silently renumber every cost in the memo.
func TestTheMenuParserReadsEveryRow(t *testing.T) {
	res := study(t)
	if len(res.Menu) != 26 {
		t.Errorf("menu parsed to %d rows, the shipped array has 26", len(res.Menu)
	}
	if res.Menu[0].To != "/" {
		t.Errorf("the menu's first entry is %q, not the dashboard", res.Menu[0].To)
	}
	if res.Totals.Destinations != len(res.Menu) {
		t.Error("the totals and the parsed menu disagree about the row count")
	}
}

// TestNoTaskPointsOutsideTheApp is the negative half of C20: every destination a
// task surface offers is a route the shell declares.
func TestNoTaskPointsOutsideTheApp(t *testing.T) {
	res := study(t)
	for _, e := range res.Edges {
		if !e.Resolves {
			t.Errorf("task %s offers %q, which the app shell does not declare", e.Task, e.To)
		}
	}
	if res.Totals.UnresolvedEdges != 0 {
		t.Errorf("totals disagree with the edge table")
	}
}

// TestGroupingNeverCostsMore states the comparison the criterion asks for as an
// invariant: putting a task's own destinations in front of the menu can only
// remove forward stops, never add one.
func TestGroupingNeverCostsMore(t *testing.T) {
	res := study(t)
	for _, c := range res.Costs {
		if c.SavedStops < 0 {
			t.Errorf("%s for %s costs %d more stops grouped than shipped", c.Task, c.Role, -c.SavedStops)
		}
	}
	if res.Totals.StopsSaved <= 0 {
		t.Error("grouping saves nothing anywhere, so the comparison has no result")
	}
	if res.Totals.StopsSaved <= 0 {
		t.Error("no stops saved anywhere, so the comparison has no result")
	}
}

// TestTheChecklistIsTheOnlyFilteringSurface pins C10 from the other side: the
// study's claim that one surface filters and three do not must come from the
// parse, not from the sentence.
func TestTheChecklistIsTheOnlyFilteringSurface(t *testing.T) {
	res := study(t)
	surfaces := map[string]bool{}
	for _, task := range res.Tasks {
		surfaces[task.Surface] = task.Gates
	}
	if !surfaces["getting started"] {
		t.Error("the checklist parsed as not filtering its steps")
	}
	for _, name := range []string{"user guide", "quick tour", "dashboard card"} {
		if v, ok := surfaces[name]; !ok {
			t.Errorf("no %s surface parsed", name)
		} else if v {
			t.Errorf("the %s surface parsed as filtering by role", name)
		}
	}
}

// TestTheDisagreementIsReal pins the headline finding: a destination the menu
// hides for a level is still offered to that level by a task surface.
func TestTheDisagreementIsReal(t *testing.T) {
	res := study(t)
	var offered []string
	for _, g := range res.Gates {
		if g.Role != "operator" || g.HiddenEdges == 0 {
			continue
		}
		if g.Surfaces == "shows every step to every role" {
			offered = append(offered, g.Task)
			continue
		}
		for _, h := range g.Hidden {
			if !strings.Contains(h, "no menu entry at all") {
				t.Errorf("%s filters its own steps yet offers an operator %q, which its menu hides",
					g.Task, h)
			}
		}
	}
	if len(offered) == 0 {
		t.Error("no unfiltered surface offers an operator a destination the menu hides")
	}
	// The checklist is the honest case: it filters, and the only destination it
	// still shows an operator is one the menu never lists at all. That is a step
	// off the navigation graph, not a gate the two surfaces disagree about.
	for _, g := range res.Gates {
		if g.Task != "setup.checklist" || g.Role != "operator" {
			continue
		}
		for _, h := range g.Hidden {
			if !strings.Contains(h, "no menu entry at all") {
				t.Errorf("expected only an off-menu step, got %q", h)
			}
		}
	}
}

// TestAnOffMenuDestinationExists names the asymmetry the workspace question
// turns on: a task can point at a route the navigation region never lists.
func TestAnOffMenuDestinationExists(t *testing.T) {
	res := study(t)
	declared := declaredSet(res.Resources, res.Routes)
	inMenu := map[string]bool{}
	for _, m := range res.Menu {
		inMenu[m.To] = true
	}
	var offMenu []string
	for to := range declared {
		if !inMenu[to] {
			offMenu = append(offMenu, to)
		}
	}
	if len(offMenu) == 0 {
		t.Error("every declared route is in the menu, so the menu is the whole application")
	}
	var steps []string
	for _, e := range res.Edges {
		if inMenu[e.To] {
			continue
		}
		steps = append(steps, e.Task+" -> "+e.To)
	}
	if len(steps) == 0 {
		t.Error("no task surface reaches an off-menu route, so the finding is empty")
	}
}

// TestRoleArmsComeFromTheCode checks the role vocabulary is derived, not typed:
// the arms are the cross product the API can actually produce.
func TestRoleArmsComeFromTheCode(t *testing.T) {
	res := study(t)
	if len(res.Roles) != len(operatorLevels(repoRoot(t)))*2 {
		t.Errorf("role arms = %d, expected levels x (no permission, issued permission)", len(res.Roles))
	}
	seen := map[string]bool{}
	for _, r := range res.Roles {
		if seen[r.Name] {
			t.Errorf("duplicate role arm %q", r.Name)
		}
		seen[r.Name] = true
		if r.Level == "" {
			t.Errorf("role arm %q has no level", r.Name)
		}
	}
}

// TestNoFilterAffordanceInTheNavigationRegion is the reason ordering is the only
// lever a workspace can pull: nothing in the chrome lets an operator skip ahead.
func TestNoFilterAffordanceInTheNavigationRegion(t *testing.T) {
	res := study(t)
	if res.Totals.NavFilterAffordances != 0 {
		t.Errorf("%d type-to-filter affordances found, so the menu is no longer order-bound",
			res.Totals.NavFilterAffordances)
	}
}

// TestCiNeverRunsTheBrowserSuite records the limit the study cannot cross: the
// numbers are derived from source order because no gate observes a focus order.
func TestCiNeverRunsTheBrowserSuite(t *testing.T) {
	res := study(t)
	if res.Totals.CIPlaywrightRuns != 0 {
		t.Errorf("CI now runs Playwright (%d invocations): the study must be re-run against the browser",
			res.Totals.CIPlaywrightRuns)
	}
	if res.Totals.KeyboardScenarios == 0 {
		t.Error("the shipped keyboard test vanished; the cost unit has no owner")
	}
}

// TestTheCommentBlankerKeepsLineNumbers guards the evidence rule itself: prose
// must disappear without moving the code, or every citation below a block comment
// would point at the wrong line.
func TestTheCommentBlankerKeepsLineNumbers(t *testing.T) {
	nl := "\n"
	files := []string{appFile, menuFile, dashboardFile, guideFile, setupFile, tourFile}
	for _, rel := range append(files, ciFile) {
		src, ok := readRaw(repoRoot(t), rel)
		if !ok {
			t.Fatalf("%s unreadable", rel)
		}
		blanks := strings.Count(src, "/*") + strings.Count(src, "\n//")
		raw := strings.Split(src, nl)
		kept := strings.Split(blankComments(src), nl)
		if len(raw) != len(kept) {
			t.Errorf("%s: blanking %d comments changed the line count from %d to %d",
				rel, blanks, len(raw), len(kept))
			continue
		}
		for i := range raw {
			if len([]rune(raw[i])) != len([]rune(kept[i])) {
				t.Errorf("%s:%d blanking changed the width of the line", rel, i+1)
				break
			}
			if strings.HasPrefix(strings.TrimSpace(raw[i]), "//") && strings.TrimSpace(kept[i]) != "" {
				t.Errorf("%s:%d a comment line survived blanking", rel, i+1)
			}
		}
	}
}
