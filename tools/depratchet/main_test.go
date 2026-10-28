package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const testNow = "2026-10-29"

func testClock(t *testing.T) time.Time {
	t.Helper()
	now, err := time.ParseInLocation(dateLayout, testNow, time.Local)
	if err != nil {
		t.Fatalf("parse test clock: %v", err)
	}
	return now.Add(12 * time.Hour)
}

func writeRegister(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "exceptions.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write register: %v", err)
	}
	return path
}

func emptyRegister(t *testing.T) string {
	t.Helper()
	return writeRegister(t, `{"exceptions": []}`)
}

func runGate(t *testing.T, stream, register string) (string, bool, error) {
	t.Helper()
	var out strings.Builder
	ok, err := run(strings.NewReader(stream), register, testClock(t), &out)
	return out.String(), ok, err
}

// findingStream builds a minimal but schema-accurate govulncheck stream: a
// config message, an OSV summary, then the given finding objects.
func findingStream(findings ...string) string {
	var b strings.Builder
	b.WriteString(`{"config":{"protocol_version":"v1.0.0","scanner_name":"govulncheck","scanner_version":"v1.9.0","db":"https://vuln.go.dev","go_version":"go1.27.0","scan_level":"symbol","scan_mode":"source"}}` + "\n")
	b.WriteString(`{"osv":{"id":"GO-2026-6617","summary":"HTTP/2 handling of a crafted request can cause a panic"}}` + "\n")
	for _, f := range findings {
		b.WriteString(f + "\n")
	}
	return b.String()
}

const calledFinding = `{"finding":{"osv":"GO-2026-6617","fixed_version":"go1.27.2","trace":[` +
	`{"module":"stdlib","version":"go1.27.0","package":"net/http","function":"Write","receiver":"(*chunkWriter)"},` +
	`{"module":"github.com/bjo163/mwx-isp","package":"github.com/bjo163/mwx-isp/internal/adminapi","function":"Start","position":{"filename":"internal/adminapi/server.go","line":42}}` +
	`]}}`

const secondCallPath = `{"finding":{"osv":"GO-2026-6617","fixed_version":"go1.27.2","trace":[` +
	`{"module":"stdlib","version":"go1.27.0","package":"net/http","function":"Write","receiver":"(*chunkWriter)"},` +
	`{"module":"github.com/bjo163/mwx-isp","package":"github.com/bjo163/mwx-isp/internal/networkmonitor","function":"Check","position":{"filename":"internal/networkmonitor/monitor.go","line":317}}` +
	`]}}`

const leadFinding = `{"finding":{"osv":"GO-2026-6866","trace":[{"module":"golang.org/x/net","version":"v0.60.0","package":"golang.org/x/net/html"}]}}`

const moduleOnlyFinding = `{"finding":{"osv":"GO-2026-5932","trace":[{"module":"golang.org/x/crypto","version":"v0.35.0"}]}}`

func TestCalledAdvisoryWithoutExceptionFails(t *testing.T) {
	out, ok, err := runGate(t, findingStream(calledFinding), emptyRegister(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatal("gate passed; a called advisory without an exception must fail")
	}
	for _, want := range []string{
		"FAIL    GO-2026-6617",
		"stdlib@go1.27.0 (net/http)",
		"fixed in go1.27.2",
		"reachable symbol net/http.(*chunkWriter).Write",
		"call path: net/http.(*chunkWriter).Write → github.com/bjo163/mwx-isp/internal/adminapi.Start (internal/adminapi/server.go:42)",
		"1 problem(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestCalledAdvisoryWithValidExceptionPasses(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6617","reason":"The admin server binds loopback only; the HTTP/2 path is unreachable from untrusted networks.","ticket":"ENG-062","expires":"2026-11-30"}
	]}`)
	out, ok, err := runGate(t, findingStream(calledFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ok {
		t.Fatalf("gate failed with a valid exception:\n%s", out)
	}
	for _, want := range []string{
		"excused GO-2026-6617",
		"until 2026-11-30 (ENG-062)",
		"1 called",
		"pass — no called advisory lacks a fix or a valid exception",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestExpiredExceptionFails(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6617","reason":"The admin server binds loopback only; the HTTP/2 path is unreachable.","ticket":"ENG-062","expires":"2026-10-28"}
	]}`)
	out, ok, err := runGate(t, findingStream(calledFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed with an expired exception:\n%s", out)
	}
	for _, want := range []string{
		"expires 2026-10-28 has passed",
		"FAIL    GO-2026-6617",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestExceptionIsValidThroughItsExpiryDay(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6617","reason":"The admin server binds loopback only; the HTTP/2 path is unreachable.","ticket":"ENG-062","expires":"2026-10-29"}
	]}`)
	if _, ok, err := runGate(t, findingStream(calledFinding), register); err != nil || !ok {
		t.Fatalf("an exception must stay valid on its own expiry day: ok=%v err=%v", ok, err)
	}
}

func TestExceptionRequiresJustificationTicketAndExpiry(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6617","reason":"n/a","ticket":"","expires":"2027-01-01"}
	]}`)
	out, ok, err := runGate(t, findingStream(calledFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed with an unjustified exception:\n%s", out)
	}
	for _, want := range []string{
		"reason is missing or shorter than a sentence",
		"ticket is missing",
		"3 problem(s)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestStaleExceptionFails(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-9999","reason":"Kept around after the dependency was removed from the module graph.","ticket":"ENG-062","expires":"2027-01-01"}
	]}`)
	out, ok, err := runGate(t, findingStream(calledFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed with a stale exception:\n%s", out)
	}
	if !strings.Contains(out, "GO-2026-9999 exception is stale") {
		t.Fatalf("output does not explain the stale entry:\n%s", out)
	}
}

func TestPackageAndModuleLeadsDoNotFail(t *testing.T) {
	out, ok, err := runGate(t, findingStream(leadFinding, moduleOnlyFinding), emptyRegister(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ok {
		t.Fatalf("leads must not fail the gate:\n%s", out)
	}
	for _, want := range []string{
		"lead    GO-2026-6866 golang.org/x/net@v0.60.0 (golang.org/x/net/html)",
		"package golang.org/x/net/html is imported but no vulnerable symbol is called",
		"lead    GO-2026-5932 golang.org/x/crypto@v0.35.0",
		"module is required but its packages are not imported",
		"2 advisories: 0 called, 2 leads",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestDistinctCallPathsGroupIntoOneAdvisory(t *testing.T) {
	out, ok, err := runGate(t, findingStream(calledFinding, secondCallPath), emptyRegister(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed a called advisory:\n%s", out)
	}
	if got := strings.Count(out, "call path:"); got != 2 {
		t.Fatalf("want 2 call paths, got %d:\n%s", got, out)
	}
	if !strings.Contains(out, "1 advisory: 1 called") {
		t.Fatalf("findings for one record must count as one advisory:\n%s", out)
	}
}

func TestRepeatedCallPathIsReportedOnce(t *testing.T) {
	out, ok, err := runGate(t, findingStream(calledFinding, calledFinding), emptyRegister(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed a called advisory:\n%s", out)
	}
	if got := strings.Count(out, "call path:"); got != 1 {
		t.Fatalf("a repeated identical trace must be printed once, got %d:\n%s", got, out)
	}
}

func TestLeadForAnExcusedAdvisoryIsStale(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6866","reason":"The html package is not imported; the module is only in the graph.","ticket":"ENG-062","expires":"2027-01-01"}
	]}`)
	out, ok, err := runGate(t, findingStream(leadFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("an exception for a non-called advisory must not be accepted:\n%s", out)
	}
	if !strings.Contains(out, "GO-2026-6866 exception is stale") {
		t.Fatalf("output does not explain the stale entry:\n%s", out)
	}
}

func TestEmptyInputIsAnError(t *testing.T) {
	if _, _, err := runGate(t, "", emptyRegister(t)); err == nil {
		t.Fatal("empty input must be rejected, not treated as a clean scan")
	}
}

func TestMalformedStreamIsAnError(t *testing.T) {
	if _, _, err := runGate(t, `{"config":`, emptyRegister(t)); err == nil {
		t.Fatal("truncated JSON must be rejected")
	}
}

func TestUnknownExceptionFieldIsRejected(t *testing.T) {
	register := writeRegister(t, `{"exceptions":[
		{"id":"GO-2026-6617","reason":"A justified sentence lives here.","ticket":"ENG-062","expiry":"2026-11-30"}
	]}`)
	_, _, err := runGate(t, findingStream(calledFinding), register)
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("a misspelled field must fail loudly, got err=%v", err)
	}
}

func TestDuplicateExceptionIsRejected(t *testing.T) {
	entry := `{"id":"GO-2026-6617","reason":"A justified sentence lives here.","ticket":"ENG-062","expires":"2026-11-30"}`
	register := writeRegister(t, `{"exceptions":[`+entry+`,`+entry+`]}`)
	out, ok, err := runGate(t, findingStream(calledFinding), register)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatalf("gate passed with a duplicate exception:\n%s", out)
	}
	if !strings.Contains(out, "duplicate entry") {
		t.Fatalf("output does not explain the duplicate:\n%s", out)
	}
}

func TestMissingRegisterIsAnError(t *testing.T) {
	_, _, err := runGate(t, findingStream(calledFinding), filepath.Join(t.TempDir(), "absent.json"))
	if err == nil {
		t.Fatal("a missing register must be an error, not an implicit pass")
	}
}
