package migrationcheck

import (
	"regexp"
	"strings"
	"testing"
)

// TestRenderMarkdownRealTree pins the artifact the memo transcribes from, so a
// change in the tree shows up as a failing assertion instead of a stale sentence.
func TestRenderMarkdownRealTree(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	out := RenderMarkdown(report)
	for _, want := range []string{
		"Migration claims verified: 20/20; gaps still open: 5/5; import columns measured: 18; row rejection rules measured: 12.",
		"| no-import-dry-run | confirmed |",
		"| no-hash-import | confirmed |",
		"| no-conflict-update-import | confirmed |",
		"| no-package-import | confirmed |",
		"| no-node-import | confirmed |",
		"| username | username, Username, USERNAME |",
		"Spreadsheet sheet name required: `Sheet1`",
		"Associated billing profile not found",
		"internal/adminapi/system_backup.go",
		"internal/networkmonitor/secret_rotation.go",
		"test/integration/adminapi_test.go#TestUserImportCSV",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render dropped %q", want)
		}
	}
	if matched, _ := regexp.MatchString(`\|\s*Wire the proposed dry-run preview[^|]*\|\s*M\s*\|`, out); !matched {
		t.Errorf("the preview item is not sized from the measured surface:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "broken") || strings.Contains(line, "vacated") || strings.Contains(line, "UNVERIFIED") {
			t.Errorf("the real tree produced a failed verdict: %s", line)
		}
	}
	t.Log("\n" + out)
}

func TestRenderMarksUnverifiedSteps(t *testing.T) {
	r := Report{
		Findings: []Finding{
			{Claim: Claim{ID: "one-transaction-per-file", ChecklistStep: "do the thing", File: "y.go"}, Status: StatusBroken, Evidence: `symbol "x" not in y.go`},
			{Claim: Claim{ID: "packages-before-customers", ChecklistStep: "first thing", File: "y.go"}, Status: StatusConfirmed, Evidence: "y.go:9"},
		},
		Absences: []AbsenceFinding{{Absence: Absence{ID: "no-hash-import", Gap: "a gap"}, Status: StatusVacated, Hits: []string{"z.go:1"}}},
	}
	out := RenderMarkdown(r)
	for _, want := range []string{
		`first thing (y.go:9)`,
		`(UNVERIFIED: symbol "x" not in y.go)`,
		"| no-hash-import | vacated | z.go:1 | a gap |",
		"Migration claims verified: 1/2; gaps still open: 0/1; import columns measured: 0; row rejection rules measured: 0.",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("render dropped %q:\n%s", want, out)
		}
	}
}

func TestRenderEscapesPipesInEvidence(t *testing.T) {
	r := Report{
		Findings: []Finding{{Claim: Claim{ID: "a|b", Statement: "x|y"}, Status: StatusConfirmed, Evidence: "x.go:1"}},
		Absences: []AbsenceFinding{{Absence: Absence{ID: "c", Gap: "g"}, Status: StatusConfirmed}},
		Surface:  MigrationSurface{Sheet: "a|b"},
	}
	out := RenderMarkdown(r)
	if !strings.Contains(out, `a\|b`) {
		t.Errorf("a pipe in a claim id escaped into the table:\n%s", out)
	}
	if strings.Contains(out, "Migration sheet") {
		t.Error("unexpected heading")
	}
}

func TestRenderListsEmptyMeasurements(t *testing.T) {
	out := RenderMarkdown(Report{Surface: MigrationSurface{Formats: []string{"csv"}}})
	for _, want := range []string{"none", "Fields the profile supplies when the row leaves them empty: none", "- none found"} {
		if !strings.Contains(out, want) {
			t.Errorf("render dropped %q for an empty measurement:\n%s", want, out)
		}
	}
}
