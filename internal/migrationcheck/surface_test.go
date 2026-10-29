package migrationcheck

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMeasureSurfaceRealTree is the probe the memo is transcribed from.
func TestMeasureSurfaceRealTree(t *testing.T) {
	surface, err := MeasureSurface(repoRoot(t))
	if err != nil {
		t.Fatalf("MeasureSurface: %v", err)
	}
	t.Logf("columns=%d headers=%d rejections=%d sheet=%q formats=%v",
		len(surface.Columns), surface.HeaderCount(), len(surface.Rejections), surface.Sheet, surface.Formats)
	for _, f := range surface.Columns {
		t.Logf("column %-14s aliases=%s", f.Column, strings.Join(f.Aliases, "/"))
	}
	for _, r := range surface.Rejections {
		t.Logf("reject: %s", r)
	}
	t.Logf("conditional=%v profile=%v", surface.ConditionalFields, surface.ProfileFields)

	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	for _, w := range WorkItems(report) {
		t.Logf("work %-28s size=%-7s count=%d driver=%s", w.ID, w.Size(report.Verdict), w.Count, w.Driver)
	}
	t.Logf("precedents=%v", report.Precedents)
	t.Logf("importTests=%v", report.ImportTests)
	t.Log("\n" + RenderMarkdown(report))
}

func TestMeasureSurfaceRequiresTheImportHandler(t *testing.T) {
	if _, err := MeasureSurface(t.TempDir()); err == nil {
		t.Fatal("expected a tree without the import handler to fail")
	}
}

func TestSurfaceReadsTheRealVocabulary(t *testing.T) {
	surface, err := MeasureSurface(repoRoot(t))
	if err != nil {
		t.Fatalf("MeasureSurface: %v", err)
	}
	if surface.Sheet != "Sheet1" {
		t.Errorf("the reader is handed %q, not the sheet the code hard-codes", surface.Sheet)
	}
	want := map[string]bool{"username": false, "password": false, "profile_id": false, "expire_time": false, "status": false}
	for _, f := range surface.Columns {
		if _, ok := want[f.Column]; ok {
			want[f.Column] = true
		}
	}
	for column, found := range want {
		if !found {
			t.Errorf("column %q was not extracted", column)
		}
	}
	if got := strings.Join(surface.Formats, " "); !strings.Contains(got, "csv") || !strings.Contains(got, "json") {
		t.Errorf("the format dispatch was not measured: %q", got)
	}
	joined := strings.Join(surface.Rejections, " | ")
	for _, reason := range []string{"Username is required", "Password is required", "Username already exists", "Associated billing profile not found"} {
		if !strings.Contains(joined, reason) {
			t.Errorf("rejection %q was not extracted from the validator", reason)
		}
	}
}

func TestSurfaceExtractionIsCaseSensitiveOnHeaders(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "adminapi"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "internal", "webserver"), 0o750); err != nil {
		t.Fatal(err)
	}
	src := "package adminapi\n\nfunc f(item map[string]interface{}) string {\n" +
		"\treturn importMapString(item, \"realname\", \"Realname\")\n}\n"
	if err := os.WriteFile(filepath.Join(root, "internal", "adminapi", "users_import.go"), []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "webserver", "server.go"), []byte("package webserver\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	surface, err := MeasureSurface(root)
	if err != nil {
		t.Fatalf("MeasureSurface: %v", err)
	}
	if len(surface.Columns) != 1 || surface.Columns[0].Column != "realname" {
		t.Fatalf("got %+v", surface.Columns)
	}
	if strings.Join(surface.Columns[0].Aliases, ",") != "realname,Realname" {
		t.Errorf("spellings lost: %+v", surface.Columns[0].Aliases)
	}
	if surface.Sheet != "" {
		t.Errorf("a fixture without an ImportData call reported sheet %q", surface.Sheet)
	}
}
