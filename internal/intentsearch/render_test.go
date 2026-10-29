package intentsearch

import (
	"strings"
	"testing"
)

var sections = []string{
	"## Verdict",
	"## What the console exposes today",
	"## Filter inputs the parser may compose",
	"## The name the menu shows and the path the API serves",
	"## The deny vocabulary",
	"## The 20 prompt fixtures",
	"### What each refusal said",
	"## Statements the proposal rests on",
	"## Gaps this study does not close",
}

func TestRenderMarkdownCarriesEverySection(t *testing.T) {
	out := RenderMarkdown(testReport(t))
	for _, s := range sections {
		if !strings.Contains(out, s) {
			t.Errorf("rendered report has no %q section", s)
		}
	}
	if strings.Contains(out, "%!") || strings.Contains(out, "<no value>") {
		t.Error("rendering produced a formatting artifact")
	}
}

func TestRenderListsEveryFixtureAndReason(t *testing.T) {
	r := testReport(t)
	out := RenderMarkdown(r)
	for _, res := range r.Results {
		if !strings.Contains(out, cell(res.Fixture.Prompt)) {
			t.Errorf("fixture %q is missing from the report", res.Fixture.Prompt)
		}
		if !strings.Contains(out, cell(res.Fixture.Why)) {
			t.Errorf("fixture %q lost its reason", res.Fixture.Prompt)
		}
	}
	for _, g := range r.Gaps {
		if !strings.Contains(out, g.ID) {
			t.Errorf("gap %s is missing", g.ID)
		}
	}
}

func TestTheVerdictLineStatesTheSafetyProperty(t *testing.T) {
	r := testReport(t)
	out := RenderMarkdown(r)
	if !strings.Contains(out, "Targets named that could reach a write | 0") {
		t.Errorf("the report does not carry the zero-write number:\n%s", section(out, "## Verdict"))
	}
	if r.Totals.Matched != r.Totals.Fixtures {
		t.Errorf("%d of %d fixtures matched", r.Totals.Matched, r.Totals.Fixtures)
	}
}

func TestRenderIsDeterministic(t *testing.T) {
	r := testReport(t)
	a, b := RenderMarkdown(r), RenderMarkdown(r)
	if a != b {
		t.Error("two renders of the same report differ")
	}
}

func section(out, header string) string {
	i := strings.Index(out, header)
	if i < 0 {
		return ""
	}
	rest := out[i:]
	if j := strings.Index(rest[len(header):], "\n## "); j >= 0 {
		return rest[:len(header)+j]
	}
	return rest
}
