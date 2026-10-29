package voicenav

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoSections are the headings the memo splices. A renamed or dropped section fails
// the build instead of leaving the memo quoting an empty table.
var memoSections = []string{
	"## The measured numbers",
	"## The vocabulary: the menu rows and the labels the two locales render",
	"## What the console performs as an operation, and what the refusal list keeps",
	"## The derived refusal set, and exactly what the pruning rules removed",
	"## The transcript corpus, scored",
	"## Per error class",
	"## The typed fallback, destination by destination",
	"## The absence scan: anything already listening",
	"## The mechanism lines",
	"## The statements, pinned to code",
	"## What this study does not close",
	"## The import set this package builds on",
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
// study can say: a scored transcript is not a spoken one, and a resolver is not a
// recognizer.
func TestReportPromisesNoHumanMeasurement(t *testing.T) {
	out := strings.ToLower(Markdown(study(t)))
	for _, phrase := range []string{"measured with users", "proven by ci", "asr accuracy",
		"will ship", "verified in a browser", "tested on a phone"} {
		assert.NotContains(t, out, phrase)
	}
}

// TestReportSplicesNoEmptyTable fails if a section the memo copies lost all its rows,
// which would otherwise leave a header with nothing under it in print.
func TestReportSplicesNoEmptyTable(t *testing.T) {
	assert.NotContains(t, Markdown(study(t)), "(no rows)")
}

// TestReportEveryStatementHasALine catches a citation that stopped landing: a table
// cell reading "(file gone)" or "(line moved)" means the memo is quoting a drift, and
// the drift belongs in the memo body, not in a footnote nobody reads.
func TestReportEveryStatementHasALine(t *testing.T) {
	out := Markdown(study(t))
	assert.NotContains(t, out, "(file gone)")
	assert.NotContains(t, out, "(line moved)")
}

// TestReportHasNoPunctuationThatBreaksATable guards the splice: a pipe inside a
// measured cell would split the row.
func TestReportHasNoPunctuationThatBreaksATable(t *testing.T) {
	for _, line := range strings.Split(Markdown(study(t)), "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		assert.LessOrEqual(t, strings.Count(line, " | "), 10, "over-wide table row: "+line)
	}
}

// TestReportKeepsEveryLimitOpen states the no-go register as the round intends it: an
// asset-free round records its limits with a verdict instead of dropping them.
func TestReportKeepsEveryLimitOpen(t *testing.T) {
	out := Markdown(study(t))
	assert.NotContains(t, out, "closed")
	for _, g := range study(t).Gaps {
		assert.Contains(t, out, "| "+g.ID+" |")
	}
}

// VOICE_DUMP=1 go test -p 1 -count=1 -v -run TestReportMarkdownDump ./internal/voicenav/
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("VOICE_DUMP") != "1" {
		t.Skip("set VOICE_DUMP=1 to print the report the memo splices")
	}
	res, err := Run(repoRoot(t))
	require.NoError(t, err)
	t.Log("\n" + Markdown(res))
}
