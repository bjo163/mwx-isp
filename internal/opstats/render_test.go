package opstats

import (
	"strings"
	"testing"
)

func report(t *testing.T) string {
	t.Helper()
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	return Markdown(res)
}

// TestReportPrintsEverySection keeps the memo's structure: a section that stops being
// measured stops being printable, and the memo would then quote a table that no longer
// exists.
func TestReportPrintsEverySection(t *testing.T) {
	out := report(t)
	for _, want := range []string{
		"## Statistics surfaces", "## The fixture", "## Aggregate census",
		"## Reconstructability", "## Suppression floors priced", "## Governance",
		"## Vocabulary searched", "## Default-tenant fallback sites",
		"## Pinned statements", "## What this round cannot answer", "## Totals",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("report lost %q", want)
		}
	}
}

// TestReportCarriesTheDigest ties the printed tables to the digest the memo quotes.
func TestReportCarriesTheDigest(t *testing.T) {
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if res.Digest == "" {
		t.Fatal("no digest")
	}
	out := Markdown(res)
	if !strings.Contains(out, "fixture stored rows") {
		t.Fatalf("the totals table lost the fixture population: %s", out[:200])
	}
}

// TestReportHasNoPunctuationThatBreaksATable keeps a measured value from splitting its own
// row, which would silently drop a field from the memo.
func TestReportHasNoPunctuationThatBreaksATable(t *testing.T) {
	out := report(t)
	for _, ln := range strings.Split(out, "\n") {
		if !strings.HasPrefix(ln, "| ") {
			continue
		}
		cells := strings.Split(strings.Trim(ln, "|"), " | ")
		if strings.Count(ln, "|") < 2 {
			t.Fatalf("row has no separators: %q", ln)
		}
		if len(cells) < 2 {
			t.Fatalf("row collapsed to one cell: %q", ln)
		}
	}
}

// TestReportKeepsEveryLimitOpen fails if the run hides a gap, so the no-go register cannot
// be closed by omission.
func TestReportKeepsEveryLimitOpen(t *testing.T) {
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	out := Markdown(res)
	for _, g := range res.Gaps {
		if !strings.Contains(out, g.ID+" | "+g.Need) {
			t.Fatalf("gap %s left the report", g.ID)
		}
	}
	if strings.Contains(out, "| closed |") {
		t.Fatal("a round that closes a limit in the report is not reporting this repository")
	}
}

// TestReportEveryStatementHasALine checks that each pinned statement reaches the page with a
// line number, so a claim cannot travel without its citation.
func TestReportEveryStatementHasALine(t *testing.T) {
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	out := Markdown(res)
	for _, c := range res.Claims {
		want := "| " + c.ID + " | " + c.Text + " | " + c.File + " | " + itoa(c.Line) + " | " + c.Status + " |"
		if !strings.Contains(out, want) {
			t.Fatalf("statement %s is not printed with its line: %q", c.ID, want)
		}
	}
}
