// Command depratchet turns a govulncheck JSON stream into a pass/fail gate.
//
//	govulncheck -json ./... | go run ./tools/depratchet
//	go run ./tools/depratchet -findings scan.json
//
// The gate fails when the scan reports an advisory that is genuinely
// applicable — govulncheck emits a symbol-level finding, meaning the vulnerable
// function is reachable from the scanned packages — and the advisory is not
// covered by a valid exception. Findings that only show a required module or an
// imported package are printed as leads and never fail the gate, because
// "the dependency is present" is not the same claim as "the vulnerable code is
// callable". Advisory triage policy lives in docs/dependency-upgrade-policy.md.
//
// Exceptions live in a JSON register (default docs/security-exceptions.json).
// Every entry needs an id, a reason, a ticket and an expiry date; the register
// is machine-checked so an excuse cannot be added without a justification or
// outlive it:
//
//	{
//	  "exceptions": [
//	    {
//	      "id": "GO-2026-6617",
//	      "reason": "Fixed by the Go 1.27.2 toolchain bump; CI pins the shipping toolchain and rescans before release.",
//	      "ticket": "ENG-062",
//	      "expires": "2026-11-30"
//	    }
//	  ]
//	}
//
// The command exits 0 when every called advisory is fixed, absent or excused,
// 1 when the gate fails, and 2 when the input or the register cannot be read.
// The gate only ingests govulncheck output; the npm side is covered by the
// release-time process described in the policy document.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"
)

const (
	exitFail = 1
	exitErr  = 2

	defaultExceptions = "docs/security-exceptions.json"
	// minReasonChars keeps the register from filling up with "n/a" or "todo":
	// an exception has to state why the risk is acceptable, in a sentence.
	minReasonChars = 24
	// dateLayout is the format of an exception's expires field.
	dateLayout = "2006-01-02"
)

func main() {
	findingsPath := flag.String("findings", "-", "govulncheck JSON stream ('-' reads stdin)")
	exceptionsPath := flag.String("exceptions", defaultExceptions, "JSON exception register")
	now := flag.String("now", "", "reference date for expiry checks (YYYY-MM-DD); defaults to today")
	flag.Parse()

	nowTime := time.Now()
	if *now != "" {
		parsed, err := time.ParseInLocation(dateLayout, *now, time.Local)
		if err != nil {
			fmt.Fprintf(os.Stderr, "depratchet: -now: %v\n", err)
			os.Exit(exitErr)
		}
		nowTime = parsed
	}

	findings, closeFindings, err := openFindings(*findingsPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "depratchet:", err)
		os.Exit(exitErr)
	}
	defer closeFindings()

	ok, err := run(findings, *exceptionsPath, nowTime, os.Stdout)
	if err != nil {
		fmt.Fprintln(os.Stderr, "depratchet:", err)
		os.Exit(exitErr)
	}
	if !ok {
		os.Exit(exitFail)
	}
}

func openFindings(path string) (io.Reader, func(), error) {
	if path == "-" {
		return os.Stdin, func() {}, nil
	}
	data, err := os.ReadFile(path) //nolint:gosec // G304: the scan file is a path the operator passed on the command line
	if err != nil {
		return nil, func() {}, fmt.Errorf("read findings: %w", err)
	}
	return bytes.NewReader(data), func() {}, nil
}

// govulncheck JSON is a stream of messages, one JSON object per message, each
// carrying exactly one of config, SBOM, osv or finding. The structs below model
// only the fields the gate needs; see golang.org/x/vuln/internal/govulncheck.
type message struct {
	Config  *scanConfig  `json:"config"`
	OSV     *osvEntry    `json:"osv"`
	Finding *scanFinding `json:"finding"`
}

type scanConfig struct {
	ProtocolVersion string `json:"protocol_version"`
	ScannerName     string `json:"scanner_name"`
	ScannerVersion  string `json:"scanner_version"`
	DB              string `json:"db"`
	GoVersion       string `json:"go_version"`
	ScanLevel       string `json:"scan_level"`
	ScanMode        string `json:"scan_mode"`
}

type osvEntry struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type scanFinding struct {
	OSV          string      `json:"osv"`
	FixedVersion string      `json:"fixed_version"`
	Trace        []scanFrame `json:"trace"`
}

type scanFrame struct {
	Module   string        `json:"module"`
	Version  string        `json:"version"`
	Package  string        `json:"package"`
	Function string        `json:"function"`
	Receiver string        `json:"receiver"`
	Position *scanPosition `json:"position"`
}

type scanPosition struct {
	Filename string `json:"filename"`
	Line     int    `json:"line"`
}

// scan is the decoded stream.
type scan struct {
	config   *scanConfig
	summary  map[string]string
	findings []scanFinding
}

// parseScan decodes the stream and rejects input that is not a complete
// govulncheck report, so an empty file or a text-mode capture cannot pass for a
// clean scan.
func parseScan(r io.Reader) (*scan, error) {
	dec := json.NewDecoder(r)
	result := &scan{summary: make(map[string]string)}
	for {
		var msg message
		if err := dec.Decode(&msg); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("parse govulncheck stream: %w", err)
		}
		switch {
		case msg.Config != nil:
			result.config = msg.Config
		case msg.OSV != nil:
			if msg.OSV.ID != "" && msg.OSV.Summary != "" {
				result.summary[msg.OSV.ID] = msg.OSV.Summary
			}
		case msg.Finding != nil:
			if msg.Finding.OSV == "" {
				return nil, errors.New("parse govulncheck stream: finding without an osv id")
			}
			result.findings = append(result.findings, *msg.Finding)
		}
	}
	if result.config == nil || result.config.ProtocolVersion == "" {
		return nil, errors.New("input is not a govulncheck JSON stream: no config message")
	}
	return result, nil
}

// advisory aggregates every finding govulncheck emitted for one OSV record.
type advisory struct {
	id      string
	summary string
	called  bool
	module  string
	version string
	pkg     string
	symbol  string
	fixed   string
	traces  [][]scanFrame
}

// aggregate groups findings by OSV id. govulncheck emits one finding per call
// path, so a record can arrive several times; up to three distinct traces are
// kept as examples.
func aggregate(s *scan) []advisory {
	byID := make(map[string]*advisory)
	seenTraces := make(map[string]bool)
	for _, f := range s.findings {
		adv := byID[f.OSV]
		if adv == nil {
			adv = &advisory{id: f.OSV, summary: s.summary[f.OSV]}
			byID[f.OSV] = adv
		}
		if f.FixedVersion != "" && adv.fixed == "" {
			adv.fixed = f.FixedVersion
		}
		if len(f.Trace) == 0 {
			continue
		}
		first := f.Trace[0]
		if adv.module == "" {
			adv.module, adv.version, adv.pkg = first.Module, first.Version, first.Package
		}
		if first.Function != "" {
			adv.called = true
			adv.module, adv.version, adv.pkg = first.Module, first.Version, first.Package
			adv.symbol = symbolName(first)
			key := f.OSV + "\x00" + traceKey(f.Trace)
			if len(adv.traces) < 3 && !seenTraces[key] {
				adv.traces = append(adv.traces, f.Trace)
				seenTraces[key] = true
			}
		}
	}
	out := make([]advisory, 0, len(byID))
	for _, adv := range byID {
		out = append(out, *adv)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

func symbolName(f scanFrame) string {
	name := f.Function
	if f.Receiver != "" {
		name = f.Receiver + "." + name
	}
	if f.Package != "" {
		name = f.Package + "." + name
	}
	return name
}

// traceKey identifies a call path so the same path printed twice by
// govulncheck is only reported once.
func traceKey(trace []scanFrame) string {
	var b strings.Builder
	for _, f := range trace {
		b.WriteString(f.Module + "|" + f.Version + "|" + f.Package + "|" + f.Receiver + "|" + f.Function)
		if f.Position != nil {
			fmt.Fprintf(&b, "|%s:%d", f.Position.Filename, f.Position.Line)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// exception register.

type exceptionFile struct {
	Exceptions []exception `json:"exceptions"`
}

type exception struct {
	ID      string `json:"id"`
	Reason  string `json:"reason"`
	Ticket  string `json:"ticket"`
	Expires string `json:"expires"`
}

func loadExceptions(path string, now time.Time) (map[string]exception, []string, error) {
	data, err := os.ReadFile(path) //nolint:gosec // G304: the register is a committed path inside the checkout
	if err != nil {
		return nil, nil, fmt.Errorf("read exception register: %w (create %s, or pass -exceptions with an existing file)", err, path)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var file exceptionFile
	if err := dec.Decode(&file); err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var trailing json.RawMessage
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, nil, fmt.Errorf("parse %s: trailing data after the exception register", path)
	}

	valid := make(map[string]exception, len(file.Exceptions))
	seen := make(map[string]bool, len(file.Exceptions))
	var problems []string
	for _, ex := range file.Exceptions {
		label := "exception " + ex.ID
		if strings.TrimSpace(ex.ID) == "" {
			problems = append(problems, "exception without an id")
			continue
		}
		if seen[ex.ID] {
			problems = append(problems, fmt.Sprintf("%s: duplicate entry", label))
			continue
		}
		seen[ex.ID] = true
		var entryProblems []string
		if len(strings.TrimSpace(ex.Reason)) < minReasonChars {
			entryProblems = append(entryProblems, "reason is missing or shorter than a sentence")
		}
		if strings.TrimSpace(ex.Ticket) == "" {
			entryProblems = append(entryProblems, "ticket is missing")
		}
		expires, err := time.ParseInLocation(dateLayout, strings.TrimSpace(ex.Expires), time.Local)
		if err != nil {
			entryProblems = append(entryProblems, "expires must be YYYY-MM-DD")
		} else if expires.Before(startOfDay(now)) {
			entryProblems = append(entryProblems, fmt.Sprintf("expires %s has passed", ex.Expires))
		}
		for _, p := range entryProblems {
			problems = append(problems, label+": "+p)
		}
		if len(entryProblems) == 0 {
			valid[ex.ID] = ex
		}
	}
	return valid, problems, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// run evaluates the scan and writes the report. It returns false when the gate
// fails; a non-nil error means the inputs could not be evaluated at all.
func run(findings io.Reader, exceptionsPath string, now time.Time, out io.Writer) (bool, error) {
	// A failed write means the report could not be delivered, which does not
	// change the gate result, so those errors are dropped.
	printf := func(format string, args ...any) { _, _ = fmt.Fprintf(out, format, args...) }

	s, err := parseScan(findings)
	if err != nil {
		return false, err
	}
	exceptions, problems, err := loadExceptions(exceptionsPath, now)
	if err != nil {
		return false, err
	}

	advisories := aggregate(s)
	var excused, unexcused, leads []advisory
	for _, adv := range advisories {
		if ex, ok := exceptions[adv.id]; adv.called && ok {
			excused = append(excused, adv)
			printf("excused %s %s fixed in %s — until %s (%s): %s\n",
				adv.id, component(adv), fixedIn(adv), ex.Expires, ex.Ticket, strings.TrimSpace(ex.Reason))
			continue
		}
		switch {
		case adv.called:
			unexcused = append(unexcused, adv)
			printf("FAIL    %s %s fixed in %s — %s\n", adv.id, component(adv), fixedIn(adv), describe(adv))
			for _, trace := range adv.traces {
				printf("        call path: %s\n", formatTrace(trace))
			}
		default:
			leads = append(leads, adv)
			printf("lead    %s %s — %s\n", adv.id, component(adv), describe(adv))
		}
	}

	// An exception only excuses a called advisory. Anything else is register
	// rot: the entry either points at an advisory that is gone or was never
	// applicable, and in both cases the excuse no longer describes reality.
	var stale []exception
	for id, ex := range exceptions {
		found := false
		for _, adv := range advisories {
			if adv.id == id && adv.called {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, ex)
		}
	}
	sort.Slice(stale, func(i, j int) bool { return stale[i].ID < stale[j].ID })
	for _, ex := range stale {
		printf("FAIL    %s exception is stale: no called advisory with that id in this scan; delete the entry or rescan\n", ex.ID)
	}
	sort.Strings(problems)
	for _, p := range problems {
		printf("FAIL    %s\n", p)
	}

	cfg := s.config
	printf("depratchet: %s %s, %s, scan %s/%s; %d %s: %d called, %d leads, %d excused\n",
		scannerName(cfg), cfg.ScannerVersion, goVersion(cfg), cfg.ScanLevel, cfg.ScanMode,
		len(advisories), plural(len(advisories), "advisory", "advisories"),
		len(unexcused)+len(excused), len(leads), len(excused))

	failed := len(unexcused) + len(stale) + len(problems)
	if failed == 0 {
		printf("depratchet: pass — no called advisory lacks a fix or a valid exception\n")
		return true, nil
	}
	printf("depratchet: %d problem(s); fix the dependency or add an exception to %s\n", failed, exceptionsPath)
	printf("depratchet: an exception needs a sentence of justification, an owner ticket and an expiry, for example\n")
	printf("depratchet:   {\"id\": \"%s\", \"reason\": \"why this call path is not reachable with attacker input today\", \"ticket\": \"ENG-000\", \"expires\": \"%s\"}\n",
		exampleID(advisories), now.AddDate(0, 1, 0).Format(dateLayout))
	return false, nil
}

func scannerName(cfg *scanConfig) string {
	if cfg.ScannerName != "" {
		return cfg.ScannerName
	}
	return "govulncheck"
}

func goVersion(cfg *scanConfig) string {
	if cfg.GoVersion != "" {
		return cfg.GoVersion
	}
	return "unknown toolchain"
}

func component(adv advisory) string {
	name := adv.module
	if adv.version != "" {
		name += "@" + adv.version
	}
	if adv.pkg != "" && adv.pkg != adv.module {
		name += " (" + adv.pkg + ")"
	}
	return name
}

func fixedIn(adv advisory) string {
	if adv.fixed == "" {
		return "no fix published"
	}
	return adv.fixed
}

func describe(adv advisory) string {
	var kind string
	switch {
	case adv.called:
		kind = "reachable symbol " + adv.symbol
	case adv.module == "stdlib":
		kind = "standard library advisory with no vulnerable symbol reachable from the scanned packages"
	case adv.pkg != "":
		kind = "package " + adv.pkg + " is imported but no vulnerable symbol is called"
	default:
		kind = "module is required but its packages are not imported"
	}
	if summary := strings.TrimSpace(adv.summary); summary != "" {
		kind += ": " + truncate(summary, 90)
	}
	return kind
}

func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	return strings.TrimSpace(s[:limit]) + "…"
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// formatTrace renders the vulnerable symbol first, then the path to the entry
// point, which is the order govulncheck emits frames in.
func formatTrace(trace []scanFrame) string {
	parts := make([]string, 0, len(trace))
	for _, f := range trace {
		switch {
		case f.Function != "":
			parts = append(parts, symbolName(f))
		case f.Package != "":
			parts = append(parts, f.Package)
		default:
			parts = append(parts, f.Module)
		}
	}
	entry := ""
	if last := trace[len(trace)-1]; last.Position != nil && last.Position.Line > 0 {
		entry = fmt.Sprintf(" (%s:%d)", last.Position.Filename, last.Position.Line)
	}
	if len(parts) > 4 {
		parts = append(append(parts[:2], "…"), parts[len(parts)-1])
	}
	return strings.Join(parts, " → ") + entry
}

func exampleID(advisories []advisory) string {
	for _, adv := range advisories {
		if adv.called {
			return adv.id
		}
	}
	if len(advisories) > 0 {
		return advisories[0].id
	}
	return "GO-0000-0000"
}
