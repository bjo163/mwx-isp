package migrationcheck

import (
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestWorkItemsAreWellFormed(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	known := map[string]bool{}
	for _, c := range Claims() {
		known[c.ID] = true
	}
	for _, a := range Absences() {
		known[a.ID] = true
	}
	items := WorkItems(report)
	if len(items) == 0 {
		t.Fatal("the study produced no work items")
	}
	for _, w := range items {
		if w.ID == "" || w.Title == "" || w.Driver == "" || len(w.Requires) == 0 {
			t.Errorf("work item %q is incomplete", w.ID)
		}
		if w.Small > w.Medium {
			t.Errorf("item %q has inverted thresholds", w.ID)
		}
		for _, id := range w.Requires {
			if !known[id] {
				t.Errorf("item %q depends on %q, which no claim or absence carries", w.ID, id)
			}
		}
	}
}

// TestRealTreeSizesAreAllKnown is the assertion that the effort items rest on
// evidence that still exists: an unknown size means a citation moved.
func TestRealTreeSizesAreAllKnown(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, w := range WorkItems(report) {
		if got := w.Size(report.Verdict); got == SizeUnknown {
			t.Errorf("item %q is unresizable on the real tree", w.ID)
		}
	}
}

func TestSizeIsUnknownWhenItsEvidenceMoved(t *testing.T) {
	w := WorkItem{ID: "x", Requires: []string{"one", "two"}, Count: 1, Small: 9}
	verdict := func(id string) Status {
		if id == "two" {
			return StatusBroken
		}
		return StatusConfirmed
	}
	if got := w.Size(verdict); got != SizeUnknown {
		t.Errorf("got %s, want %s", got, SizeUnknown)
	}
	if got := w.Size(func(string) Status { return StatusConfirmed }); got != SizeSmall {
		t.Errorf("got %s, want %s", got, SizeSmall)
	}
}

// TestSizeFollowsTheMeasuredCount keeps the estimate arithmetic honest: a widened
// validator has to show up as a bigger item, not as a rewritten paragraph.
func TestSizeFollowsTheMeasuredCount(t *testing.T) {
	confirmed := func(string) Status { return StatusConfirmed }
	small := WorkItem{Count: 18, Small: 18, Medium: 30}
	large := WorkItem{Count: 31, Small: 18, Medium: 30}
	if small.Size(confirmed) != SizeSmall {
		t.Errorf("18 with threshold 18 should be %s", SizeSmall)
	}
	if got := (WorkItem{Count: 19, Small: 18, Medium: 30}).Size(confirmed); got != SizeMedium {
		t.Errorf("got %s, want %s", got, SizeMedium)
	}
	if large.Size(confirmed) != SizeLarge {
		t.Errorf("got %s, want %s", large.Size(confirmed), SizeLarge)
	}
}

// TestEstimateCarriesNoInventedNumbers keeps the memo inside what a read-only
// study can say: t-shirt sizes yes, hours, days and prices no.
func TestEstimateCarriesNoInventedNumbers(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	out := RenderMarkdown(report)
	banned := []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(hour|hours|day|days|week|weeks|man-month)\b`),
		regexp.MustCompile(`(?i)\b(usd|eur|idr|rm)\b`),
		regexp.MustCompile(`[$€]`),
	}
	for _, re := range banned {
		if re.MatchString(out) {
			t.Errorf("the rendered estimate asserts pricing vocabulary it cannot support: %q", re.String())
		}
	}
	for _, w := range WorkItems(report) {
		row := "| " + w.Title + " | " + w.Size(report.Verdict) + " | " + w.Driver + " |"
		if !strings.Contains(out, row) {
			t.Errorf("the rendered estimate dropped the sized row:\n%s", row)
		}
		if !strings.HasPrefix(w.Driver, strconv.Itoa(w.Count)) {
			t.Errorf("item %s opens with an unmeasured clause: %q", w.ID, w.Driver)
		}
	}
}

func TestChecklistIsOrderedAndAnnotated(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	steps := BuildChecklist(report)
	if len(steps) != len(Claims()) {
		t.Fatalf("checklist produced %d steps for %d claims", len(steps), len(Claims()))
	}
	for i, s := range steps {
		if s.Sequence != i+1 {
			t.Errorf("step %d is out of sequence", s.Sequence)
		}
		if !s.Verified {
			t.Errorf("step %d (%s) is unverified on the real tree: %s", s.Sequence, s.ClaimID, s.Evidence)
		}
		if !strings.Contains(s.Evidence, ":") {
			t.Errorf("step %d carries no file:line citation: %q", s.Sequence, s.Evidence)
		}
	}
	// Dependency order: the plan catalog and node inventory are preconditions of
	// the customer file, so both must precede the run itself.
	positions := map[string]int{}
	for _, s := range steps {
		positions[s.ClaimID] = s.Sequence
	}
	for _, precondition := range []string{"packages-before-customers", "nodes-before-customers"} {
		if positions[precondition] > positions["one-transaction-per-file"] {
			t.Errorf("%s should be ordered before the write step", precondition)
		}
	}
}

func TestFindDryRunPrecedentsIgnoresTheImportAndThePrototype(t *testing.T) {
	found, err := FindDryRunPrecedents(repoRoot(t))
	if err != nil {
		t.Fatalf("FindDryRunPrecedents: %v", err)
	}
	for _, f := range found {
		if strings.HasSuffix(f, importFile) || strings.HasPrefix(f, "internal/migrationcheck/") {
			t.Errorf("the scan should exclude %s, found it anyway", f)
		}
	}
	if len(found) < 2 {
		t.Errorf("the tree has backup-restore and secret-rotation preflights, found %v", found)
	}
}

func TestFindImportTestsNamesTheContractTests(t *testing.T) {
	names, err := FindImportTests(repoRoot(t))
	if err != nil {
		t.Fatalf("FindImportTests: %v", err)
	}
	joined := strings.Join(names, " ")
	for _, want := range []string{"TestImportRadiusUsers_Errors", "TestImportRadiusUsers_RejectsUnusableValues", "TestUserImportCSV"} {
		if !strings.Contains(joined, want) {
			t.Errorf("the contract test %s was not found in %v", want, names)
		}
	}
}

// TestTheProductHasOneImportSurface is the measurement behind the estimate's
// central claim: everything a migration touches except the customer file is
// entered by hand today.
func TestTheProductHasOneImportSurface(t *testing.T) {
	sites, err := FindImportCallSites(repoRoot(t))
	if err != nil {
		t.Fatalf("FindImportCallSites: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("expected exactly one upload handler, got %v", sites)
	}
	if !strings.HasPrefix(sites[0], importFile) {
		t.Errorf("the only import surface is %s, not the customer handler", sites[0])
	}
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, w := range WorkItems(report) {
		if w.ID != "prebuild-dependencies" {
			continue
		}
		if !strings.Contains(w.Driver, "(1 upload handler exists in the whole admin API)") {
			t.Errorf("the manual-precondition item does not quote the measured surface: %q", w.Driver)
		}
	}
}
