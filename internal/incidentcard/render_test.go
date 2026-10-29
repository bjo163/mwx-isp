package incidentcard

import (
	"os"
	"strings"
	"testing"
)

// The memo splices these tables verbatim, so the rendering is itself under test: a
// section that stops printing, a cell that grows a pipe, or a citation that turns
// into a dash is a defect in the document, not in the report.

var reportSections = []string{
	"## Field ledger",
	"## Name-shaped slots and the rows behind them",
	"## Identity tables the rules cover only partly",
	"## Greys on the printed sheet",
	"## Status marks told apart without hue",
	"## Paper surface census",
	"## Grunge channels on paper",
	"## Page budget",
	"## State vocabulary and its non-color channels",
	"## Stale and confidence visibility",
	"## Pinned statements",
	"## What this round cannot answer",
	"## Totals",
}

func report(t *testing.T) string {
	t.Helper()
	return Markdown(mustRun(t, repoRoot(t)))
}

// TestReportPrintsEverySection keeps the splice surface whole.
func TestReportPrintsEverySection(t *testing.T) {
	out := report(t)
	for _, want := range reportSections {
		if !strings.Contains(out, want) {
			t.Errorf("section %q is missing from the report", want)
		}
	}
}

// TestReportCarriesTheDigest pins the reproducibility claim to the document that
// quotes it, so a memo copy cannot drift from the run it came from.
func TestReportCarriesTheDigest(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	out := report(t)
	if !strings.Contains(out, "Digest: `"+res.Digest+"`") {
		t.Errorf("the report does not carry the digest of the run it rendered")
	}
}

// TestReportHasNoPunctuationThatBreaksATable guards the splice: a pipe inside a
// measured cell would split the row in markdown.
func TestReportHasNoPunctuationThatBreaksATable(t *testing.T) {
	for _, line := range strings.Split(report(t), "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		if n := strings.Count(line, " | "); n > 10 {
			t.Errorf("over-wide table row (%d cells): %s", n+1, line)
		}
	}
}

// TestReportKeepsEveryLimitOpen states the no-go register as the round intends it:
// an asset-free round records its limits with a verdict instead of dropping them.
func TestReportKeepsEveryLimitOpen(t *testing.T) {
	out := report(t)
	res := mustRun(t, repoRoot(t))
	for _, g := range res.Gaps {
		if !strings.Contains(out, "| "+g.ID+" |") {
			t.Errorf("gap %s is not in the report", g.ID)
		}
		if g.Status != "no-go" {
			t.Errorf("gap %s carries status %q, want no-go", g.ID, g.Status)
		}
	}
	if strings.Contains(out, "| closed |") {
		t.Errorf("a gap was recorded as closed")
	}
}

// TestReportEveryStatementHasALine catches a citation that stopped landing: the
// line cell of a confirmed statement is a number, and a dash means the memo is
// quoting a drift.
func TestReportEveryStatementHasALine(t *testing.T) {
	out := report(t)
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "| A") || !strings.Contains(line, "confirmed") {
			continue
		}
		cells := strings.Split(strings.Trim(line, "| "), " | ")
		if len(cells) != 5 {
			t.Fatalf("statement row shape changed: %s", line)
		}
		if cells[3] == "-" || cells[3] == "" {
			t.Errorf("statement %s cites no line: %s", cells[0], line)
		}
	}
}

// CARD_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/incidentcard/
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("CARD_DUMP") != "1" {
		t.Skip("set CARD_DUMP=1 to print the report the memo splices")
	}
	t.Log("\n" + report(t))
}
