package incidenttriage

import (
	"os"
	"strconv"
	"strings"
	"testing"
)

var memoSections = []string{
	"## Verdict",
	"## The labeled corpus",
	"## What redaction touched",
	"## What the export's own filters carry",
	"## The taxonomy and where its tokens come from",
	"## Three engines over the same corpus",
	"## Per-cause detail for the rule engine",
	"## What each engine said, incident by incident",
	"## No automatic remediate",
	"## Statements the study rests on",
	"## Gaps this study does not close",
}

func TestRenderMarkdownCarriesEverySection(t *testing.T) {
	out := RenderMarkdown(testResult(t))
	for _, s := range memoSections {
		if !strings.Contains(out, s) {
			t.Errorf("rendered report has no %q section", s)
		}
	}
	if strings.Contains(out, "%!") || strings.Contains(out, "<no value>") {
		t.Error("rendering produced a formatting artifact")
	}
}

// TestRenderReportsEveryLabel is the criterion's first half: the dataset labels
// are printed, case by case, not summarized away.
func TestRenderReportsEveryLabel(t *testing.T) {
	r := testResult(t)
	out := RenderMarkdown(r)
	for _, p := range r.Pairs {
		if !strings.Contains(out, p.Case.ID) {
			t.Errorf("case %s is missing from the report", p.Case.ID)
		}
		if !strings.Contains(out, cell(p.Case.Why)) {
			t.Errorf("case %s lost the reason for its label", p.Case.ID)
		}
	}
	for _, id := range CauseIDs() {
		if !strings.Contains(out, id) {
			t.Errorf("class %q never appears in the report", id)
		}
	}
}

// TestRenderPrintsTheHallucinationRate is the criterion's second half: every
// engine's rate is in the prose, including the engines that report zero.
func TestRenderPrintsTheHallucinationRate(t *testing.T) {
	r := testResult(t)
	out := RenderMarkdown(r)
	for _, s := range r.Scores {
		want := s.Engine + ": hallucination rate"
		if !strings.Contains(out, want) {
			t.Errorf("the verdict table has no %q row", want)
		}
		if !strings.Contains(out, s.Engine+": hallucinated") {
			t.Errorf("the verdict table has no %q row", s.Engine+": hallucinated")
		}
		rate := strconv.FormatFloat(s.HallucinationRate()*100, 'f', 1, 64) + "%"
		if !strings.Contains(out, rate) {
			t.Errorf("%s hallucination rate %s is not in the report", s.Engine, rate)
		}
	}
}

func TestRenderNamesEveryClaimAndGap(t *testing.T) {
	r := testResult(t)
	out := RenderMarkdown(r)
	for _, c := range r.Claims {
		if !strings.Contains(out, c.ID) {
			t.Errorf("claim %s is missing", c.ID)
		}
	}
	for _, g := range r.Gaps {
		if !strings.Contains(out, g.ID) {
			t.Errorf("gap %s is missing", g.ID)
		}
	}
	for _, token := range allSignals() {
		if !strings.Contains(out, `"`+token+`"`) {
			t.Errorf("taxonomy token %q is missing", token)
		}
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	r := testResult(t)
	a, b := RenderMarkdown(r), RenderMarkdown(r)
	if a != b {
		t.Error("two renders of the same run differ")
	}
}

// TestReportMarkdownDump prints the tables the memo splices in, so every number
// in the prose comes from the same run that produced the verdict.
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("TRIAGE_DUMP") != "1" {
		t.Skip("set TRIAGE_DUMP=1 to print the memo tables")
	}
	t.Log("\n" + RenderMarkdown(testResult(t)))
}
