package mangaskin

import (
	"strconv"
	"strings"
	"testing"
)

func renderReport(t *testing.T) string {
	t.Helper()
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return RenderMarkdown(report)
}

func TestRenderMarkdownCarriesEverySectionTheCriterionAsksFor(t *testing.T) {
	text := renderReport(t)
	for _, section := range []string{
		"## Verdict",
		"## Claims the direction rests on",
		"## Gaps the criterion depends on",
		"## Contrast of the states the console is made of",
		"### AA violations",
		"### Measured, exempt from AA",
		"## The prototype skin, as token re-aimings",
		"### What changes when they are applied",
		"## Color vocabulary measured under web/",
		"## Rollback posture",
	} {
		if !strings.Contains(text, section) {
			t.Errorf("report is missing the %q section", section)
		}
	}
	for _, bad := range []string{"%!(", "<no value>", "!!(MISSING)"} {
		if strings.Contains(text, bad) {
			t.Errorf("report contains a formatting artifact %q", bad)
		}
	}
}

// TestRenderMarkdownNamesEveryClaimAndState keeps the tables honest: a claim that
// dropped out of the report would still be quoted in the memo, because the memo is
// written from this text.
func TestRenderMarkdownNamesEveryClaimAndState(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	text := RenderMarkdown(report)
	for _, f := range report.Findings {
		if !strings.Contains(text, f.Claim.ID) {
			t.Errorf("claim %s is missing from the report", f.Claim.ID)
		}
	}
	for _, a := range report.Absences {
		if !strings.Contains(text, a.Absence.ID) {
			t.Errorf("gap %s is missing from the report", a.Absence.ID)
		}
	}
	for _, st := range States() {
		if !strings.Contains(text, st.ID) {
			t.Errorf("state %s is missing from the report", st.ID)
		}
	}
}

// TestRenderMarkdownNumbersComeFromTheRun is the anti-transcription rule: the
// verdict sentence has to be the same counts the slices hold.
func TestRenderMarkdownNumbersComeFromTheRun(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	text := RenderMarkdown(report)
	head := ""
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "Color pairs measured:") {
			head = line
			break
		}
	}
	if head == "" {
		t.Fatal("no verdict line found")
	}
	for _, want := range []string{
		strconv.Itoa(report.Totals.States),
		strconv.Itoa(report.Totals.Held),
		strconv.Itoa(report.Totals.Exempt),
		strconv.Itoa(report.Totals.FailHeld),
		strconv.Itoa(report.Totals.VariantFailHeld),
		strconv.Itoa(report.Totals.NewColors),
		strconv.Itoa(report.Totals.Stylesheets),
	} {
		if !strings.Contains(head, want) {
			t.Errorf("verdict line %q does not carry the measured %s", head, want)
		}
	}
	if !strings.Contains(text, "Stylesheets under web/: 0") {
		t.Error("the rollback table lost the measured stylesheet count")
	}
}

// pairKey collapses a report line's padding so a needle can name a state and a
// mode without depending on the column width the table happened to pick.
func pairKey(state, mode string) string {
	return strings.Join(strings.Fields(state+" "+mode), " ")
}

func collapse(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func TestRenderMarkdownSplitsViolationsFromExemptPairs(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	text := RenderMarkdown(report)
	viol := collapse(section(text, "### AA violations"))
	exempt := collapse(section(text, "### Measured, exempt from AA"))
	for _, m := range report.Summary.FailHeld {
		if !strings.Contains(viol, pairKey(m.State.ID, m.Mode)) {
			t.Errorf("held violation %s/%s is not in the violations section", m.State.ID, m.Mode)
		}
		if strings.Contains(exempt, pairKey(m.State.ID, m.Mode)) {
			t.Errorf("held violation %s/%s leaked into the exempt section", m.State.ID, m.Mode)
		}
	}
	for _, m := range report.Summary.Fail {
		if m.State.Kind != KindExempt {
			continue
		}
		if !strings.Contains(exempt, pairKey(m.State.ID, m.Mode)) {
			t.Errorf("exempt pair %s/%s is not in the exempt section", m.State.ID, m.Mode)
		}
		if strings.Contains(viol, pairKey(m.State.ID, m.Mode)) {
			t.Errorf("exempt pair %s/%s was reported as an AA violation", m.State.ID, m.Mode)
		}
	}
}

// TestRenderIsDeterministic is the map-iteration guard: two runs over the same tree
// have to produce byte-identical text, or the memo is quoting one of two reports.
func TestRenderIsDeterministic(t *testing.T) {
	a, b := renderReport(t), renderReport(t)
	if a != b {
		t.Error("two runs over the same tree rendered different reports")
	}
}

func TestReportMarkdownDump(t *testing.T) {
	t.Logf("\n%s", renderReport(t))
}

// section returns the body of one heading, up to the next heading of the same or
// higher level.
func section(text, heading string) string {
	at := strings.Index(text, heading)
	if at < 0 {
		return ""
	}
	rest := text[at+len(heading):]
	for _, stop := range []string{"\n### ", "\n## "} {
		if cut := strings.Index(rest, stop); cut >= 0 {
			rest = rest[:cut]
		}
	}
	return rest
}
