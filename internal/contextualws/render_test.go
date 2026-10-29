package contextualws

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoSections are the headings the memo splices. A renamed or dropped section
// fails the build instead of leaving the memo quoting an empty table.
var memoSections = []string{
	"## The measured numbers",
	"## The shipped menu, numbered the way a keyboard numbers it",
	"## The task surfaces the console already ships",
	"## Navigation cost: the shipped order against the task's own order",
	"## Where a task surface and the same role's menu disagree",
	"## Every destination a task offers, with the control that carries it",
	"## The navigation controls in the console, by element",
	"## The mechanism lines",
	"## The statements, pinned to code",
	"## What this study does not close",
	"## The import set this package builds on",
	"## The lines this study reads",
	"## What the digest locks",
}

func TestReportCarriesEverySection(t *testing.T) {
	out := Markdown(study(t))
	for _, s := range memoSections {
		if !strings.Contains(out, s) {
			t.Errorf("report has no %q section", s)
		}
	}
	assert.NotContains(t, out, "%!")
	assert.NotContains(t, out, "<no value>")
}

// TestReportPromisesNoHumanMeasurement keeps the memo honest about what a static
// study can say: focus stops are not usability, and a source-order number is not
// a rendered one.
func TestReportPromisesNoHumanMeasurement(t *testing.T) {
	out := strings.ToLower(Markdown(study(t)))
	for _, phrase := range []string{"faster in practice", "measured with users", "proven by ci",
		"will ship", "verified in a browser"} {
		assert.NotContains(t, out, phrase)
	}
}

// TestReportSplicesNoEmptyTable fails if a section the memo copies lost all its
// rows, which would otherwise leave a header with nothing under it in print.
func TestReportSplicesNoEmptyTable(t *testing.T) {
	out := Markdown(study(t))
	assert.NotContains(t, out, "(no rows)")
}

// TestReportEveryStatementHasALine catches a citation that stopped landing: a
// table cell reading "(file gone)" or "(line moved)" means the memo is quoting a
// drift, and the drift belongs in the memo body, not in a footnote nobody reads.
func TestReportEveryStatementHasALine(t *testing.T) {
	out := Markdown(study(t))
	assert.NotContains(t, out, "(file gone)")
	assert.NotContains(t, out, "(line moved)")
}

// TestReportHasNoPunctuationThatBreaksATable guards the splice: a pipe inside a
// measured cell would split the row.
func TestReportHasNoPunctuationThatBreaksATable(t *testing.T) {
	out := Markdown(study(t))
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		assert.LessOrEqual(t, strings.Count(line, " | "), 10, "over-wide table row: "+line)
	}
}

// WS_DUMP=1 go test -run TestReportMarkdownDump ./internal/contextualws/
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("WS_DUMP") != "1" {
		t.Skip("set WS_DUMP=1 to print the report the memo splices")
	}
	res, err := Run(repoRoot(t))
	require.NoError(t, err)
	t.Log("\n" + Markdown(res))
}
