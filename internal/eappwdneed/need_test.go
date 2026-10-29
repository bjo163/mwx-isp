// Tests for the EAP-PWD need verification. They run against the real repository
// tree, so a claim that drifts out of date fails a test instead of surviving in a
// memo.

package eappwdneed

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// repoRoot returns the repository root as seen from this file.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate this test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(here), "..", ".."))
}

// writeTree lays a minimal repository at root so a verdict can be forced.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatalf("mkdir %s: %v", rel, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
}

func TestClaimCatalogIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range Claims() {
		if seen[c.ID] {
			t.Errorf("duplicate claim ID %q", c.ID)
		}
		seen[c.ID] = true
		if c.Statement == "" || c.File == "" || c.Symbol == "" {
			t.Errorf("claim %q is not pinned to a statement, file and symbol", c.ID)
		}
		if c.Gap == "" {
			t.Errorf("claim %q states no consequence for EAP-PWD, so it proves nothing about the need", c.ID)
		}
	}
	for _, a := range Absences() {
		if a.ID == "" || a.Symbol == "" || a.Scope == "" {
			t.Errorf("negative claim %+v needs an ID, symbol and scope", a)
		}
	}
}

func TestEveryCitedFileIsInTheRepository(t *testing.T) {
	root := repoRoot(t)
	for _, c := range Claims() {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(c.File))); err != nil {
			t.Errorf("claim %q cites %s, which is not readable: %v", c.ID, c.File, err)
		}
	}
}

func TestRealTreeConfirmsEveryNeedClaim(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed {
			t.Errorf("claim %q is %s: %s\nthe M12 need argument still rests on it, so either the code moved or the memo has to change",
				f.Claim.ID, f.Status, f.Evidence)
		}
	}
	if report.Totals.Confirmed != report.Totals.Claims {
		t.Errorf("confirmed %d of %d claims", report.Totals.Confirmed, report.Totals.Claims)
	}
}

func TestRealTreeEveryGapStillOpen(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, a := range report.Absences {
		if a.Status != StatusConfirmed {
			t.Errorf("negative claim %q no longer holds: %v", a.Absence.ID, a.Hits)
		}
	}
	if report.Totals.AbsencesHeld != report.Totals.Absences {
		t.Errorf("held %d of %d gaps open", report.Totals.AbsencesHeld, report.Totals.Absences)
	}
}

func TestAbsenceScanIgnoresThePrototypeItself(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	// This package quotes "eap-pwd" in its own source. If the scan counted its
	// own text, the gap would look vacated and the survey would refute itself.
	for _, a := range report.Absences {
		for _, hit := range a.Hits {
			if strings.Contains(hit, "eappwdneed") {
				t.Errorf("negative claim %q was refuted by the prototype's own file %s", a.Absence.ID, hit)
			}
		}
	}
}

func TestMovedFileIsReportedBrokenNotDropped(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"internal/domain/doc.go":    "package domain\n",
		"docs/rfcs/rfc3748-eap.txt": "Request for Comments: 3748\n\nAbstract\n",
	})
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	broken := 0
	for _, f := range report.Findings {
		if f.Status == StatusBroken {
			broken++
			if f.Evidence == "" {
				t.Errorf("broken claim %q has no explanation", f.Claim.ID)
			}
		}
	}
	if broken != len(Claims()) {
		t.Errorf("an empty tree confirmed %d claims; every citation should report broken", len(Claims())-broken)
	}
}

func TestLongerNameDoesNotConfirmAShorterClaim(t *testing.T) {
	root := t.TempDir()
	// Only a compound name is present, so the bare "Password" claim must not be
	// satisfied by an accessor name that merely ends with the same word.
	writeTree(t, root, map[string]string{
		"internal/domain/doc.go":    "package domain\n",
		"internal/domain/radius.go": "package domain\n\nfunc GetPassword() string { return \"\" }\n",
		"docs/rfcs/rfc3748-eap.txt": "Request for Comments: 3748\n\nAbstract\n",
	})
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	f := findFinding(report, "password-storage-is-cleartext")
	if f == nil {
		t.Fatal("claim password-storage-is-cleartext missing from report")
	}
	if f.Status != StatusBroken {
		t.Errorf("compound names confirmed a bare field claim: %+v", f)
	}
}

func TestMissingAbsenceScopeIsAnError(t *testing.T) {
	root := t.TempDir()
	if _, err := Verify(root); err == nil {
		t.Fatal("Verify accepted a tree with no internal/ directory; a missing scope reads as an open gap")
	}
}

func TestFindingsAreOrderedByClaimID(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for i := 1; i < len(report.Findings); i++ {
		if report.Findings[i-1].Claim.ID > report.Findings[i].Claim.ID {
			t.Fatalf("findings not sorted by ID: %s before %s", report.Findings[i-1].Claim.ID, report.Findings[i].Claim.ID)
		}
	}
}

func TestFindingStringCarriesVerdictAndEvidence(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	f := findFinding(report, "type-constants-stop-short-of-pwd")
	if f == nil {
		t.Fatal("claim type-constants-stop-short-of-pwd missing from report")
	}
	line := f.String()
	for _, want := range []string{"confirmed", f.Claim.File, f.Evidence} {
		if !strings.Contains(line, want) {
			t.Errorf("Finding.String() = %q, missing %q", line, want)
		}
	}
}

func TestNoClaimAssertsClientSupport(t *testing.T) {
	// The survey reads this repository. Nothing here can show that a supplicant
	// offers EAP-PWD, so no claim may imply one does.
	forbidden := []string{"Windows", "macOS", "iOS", "Android", "wpa_supplicant", "Interoperability", "interoperability", "all clients", "every client"}
	for _, c := range Claims() {
		for _, word := range forbidden {
			if strings.Contains(c.Statement, word) || strings.Contains(c.Gap, word) {
				t.Errorf("claim %q asserts client-side support (%s), which this scan cannot see", c.ID, word)
			}
		}
	}
}

// findFinding returns one finding by claim ID.
func findFinding(r Report, id string) *Finding {
	for i := range r.Findings {
		if r.Findings[i].Claim.ID == id {
			return &r.Findings[i]
		}
	}
	return nil
}

func TestCommentMentionsAreNotEvidence(t *testing.T) {
	// A doc comment that names a symbol does not prove the symbol exists in the
	// code the compiler sees, and a URL inside a string is not a comment.
	source := "package handlers\n\n" +
		"// handleInnerEAP runs EAP-PWD inside the tunnel.\n" +
		"func Nothing() {\n" +
		"\turl := \"https://example.invalid/eap-pwd\"\n" +
		"\t_ = url // See the Reference\n" +
		"}\n"
	lines := codeLines(source)
	if got := locate(lines, "handleInnerEAP"); got != 0 {
		t.Errorf("a doc comment on line %d confirmed a symbol the code never declares", got)
	}
	if got := locate(lines, "Nothing"); got != 4 {
		t.Errorf("code line moved: located Nothing at %d, want 4", got)
	}
	if got := locate(lines, "example.invalid"); got != 5 {
		t.Errorf("a URL was blanked as if it were a comment: got %d, want 5", got)
	}
	if got := locate(lines, "See the Reference"); got != 0 {
		t.Errorf("trailing comment survived at line %d", got)
	}
}

func TestEveryRealCitationIsACodeLine(t *testing.T) {
	root := repoRoot(t)
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, f := range report.Findings {
		if f.Status != StatusConfirmed {
			continue
		}
		at := strings.LastIndex(f.Evidence, ":")
		if at < 0 {
			t.Fatalf("evidence %q is not a file:line citation", f.Evidence)
		}
		//nolint:gosec // G304: a path this package just read from the repository.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(f.Evidence[:at])))
		if err != nil {
			t.Fatalf("reading %s: %v", f.Evidence, err)
		}
		var num int
		if _, err := fmt.Sscanf(f.Evidence[at+1:], "%d", &num); err != nil {
			t.Fatalf("bad citation %q: %v", f.Evidence, err)
		}
		line := strings.TrimSpace(codeLines(string(raw))[num-1])
		if line == "" {
			t.Errorf("%s cites a comment-only line; the claim needs a code line behind it", f.Evidence)
		}
	}
}

// TestRenderMarkdownRealTree is the probe the memo was transcribed from.
func TestRenderMarkdownRealTree(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	out := RenderMarkdown(report)
	if !strings.Contains(out, "need claims confirmed") {
		t.Errorf("render dropped its verdict line:\n%s", out)
	}
	for _, want := range []string{"eap-pwd-spec (RFC 5931 EAP-PWD) | mislabeled", "rfc7542-eap-pwd.txt", "Network Access Identifier"} {
		if !strings.Contains(out, want) {
			t.Errorf("render does not show that the file named for EAP-PWD is %q:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "teap-spec-control (RFC 7170 TEAP) | carried") {
		t.Errorf("render lost the positively carried control:\n%s", out)
	}
	t.Log("\n" + out)
}

func TestRenderMarkdownEscapesPipesInEvidence(t *testing.T) {
	r := Report{
		Findings:  []Finding{{Claim: Claim{ID: "a|b", File: "x.go"}, Status: StatusConfirmed, Evidence: "x.go:1"}},
		Absences:  []AbsenceFinding{{Absence: Absence{ID: "c"}, Status: StatusConfirmed}},
		Vendoring: Vendoring{Specs: []SpecStatus{{Spec: Spec{Number: "5931", Topic: "E|F"}, State: SpecAbsent}}},
	}
	out := RenderMarkdown(r)
	if strings.Contains(out, "a|b |") {
		t.Errorf("an unescaped pipe broke a table cell:\n%s", out)
	}
	if !strings.Contains(out, `E\|F`) {
		t.Errorf("topic pipe not escaped:\n%s", out)
	}
}
