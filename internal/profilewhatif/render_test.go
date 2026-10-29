package profilewhatif

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memoSections are the headings the memo splices. A renamed or dropped section fails
// the build instead of silently leaving the memo quoting an empty table.
var memoSections = []string{
	"## The measured numbers",
	"## The fixture profiles",
	"## The fixture subscribers",
	"## What one Access-Accept says before any edit",
	"## What the ledger refuses to diff",
	"## Which wire attributes move when exactly one profile column changes",
	"## The scenarios, one diff each",
	"## The statements, pinned to code",
	"## What this study does not close",
	"## The import set this package builds on",
}

func TestReportCarriesEverySection(t *testing.T) {
	out := Markdown(studyResult(t))
	for _, s := range memoSections {
		if !strings.Contains(out, s) {
			t.Errorf("report has no %q section", s)
		}
	}
	assert.NotContains(t, out, "%!")
	assert.NotContains(t, out, "<no value>")
}

// TestReportNamesNoForbiddenVerb keeps the preview framed as a preview: the report may
// name the package that owns CoA, but never promises an action.
func TestReportNamesNoForbiddenVerb(t *testing.T) {
	out := Markdown(studyResult(t))
	for _, phrase := range []string{"will disconnect", "applied", "session restarted", "pushed to the NAS"} {
		assert.NotContains(t, strings.ToLower(out), strings.ToLower(phrase))
	}
}

// TestReportMarkdownDump prints the report for the memo. Run with
// WHATIF_DUMP=1 go test -run TestReportMarkdownDump ./internal/profilewhatif/
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("WHATIF_DUMP") != "1" {
		t.Skip("set WHATIF_DUMP=1 to print the report the memo splices")
	}
	res, err := Run(repoRoot(t))
	require.NoError(t, err)
	t.Log("\n" + Markdown(res))
}
