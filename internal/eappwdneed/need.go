// Package eappwdneed checks the claims behind milestone M12 (EAP-PWD) against
// the code that would have to host it.
//
// It is an ENG-007 research prototype. It only reads, no product code imports it,
// and it answers two questions that a roadmap line cannot answer by itself:
// whether every statement the need argument rests on is still true in the tree
// today, and whether the specifications this project plans to implement are
// actually vendored under the names it uses. A claim that no longer matches a
// file is reported as broken rather than quietly dropped.
package eappwdneed

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Claim is one statement the EAP-PWD need argument depends on, pinned to the
// file that proves it.
type Claim struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the memo asserts about the current authentication
	// surface.
	Statement string
	// File is the repository-relative path that must prove it.
	File string
	// Symbol is the text that must appear in that file. It is matched as an
	// identifier, so a longer name cannot prove a shorter one.
	Symbol string
	// Gap says what EAP-PWD would change if the claim holds.
	Gap string
}

// Absence is a negative claim: a symbol that must not appear anywhere in scope.
type Absence struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the absence proves.
	Statement string
	// Symbol must not appear in any file under Scope.
	Symbol string
	// Scope is the directory prefix the search covers.
	Scope string
}

// Status is one verification verdict.
type Status string

// The three verdicts a claim can reach. A broken claim is reported, never
// silently removed, because the memo it supports would otherwise keep a
// statement the code no longer makes.
const (
	StatusConfirmed Status = "confirmed"
	StatusBroken    Status = "broken"
	StatusVacated   Status = "vacated"
)

// Finding is one verified claim.
type Finding struct {
	// Claim is the statement checked.
	Claim Claim
	// Status is the verdict.
	Status Status
	// Evidence is the file:line that proves it, or where the search failed.
	Evidence string
	// Note carries Claim.Gap plus any qualification the verdict needs.
	Note string
}

// String renders one finding as a single line.
func (f Finding) String() string {
	return fmt.Sprintf("%s %s [%s] %s", f.Claim.ID, f.Status, f.Claim.File, f.Evidence)
}

// AbsenceFinding is one verified negative claim.
type AbsenceFinding struct {
	// Absence is the negative claim checked.
	Absence Absence
	// Status is confirmed when nothing matches, vacated when something does.
	Status Status
	// Hits lists the matching locations, capped for readability.
	Hits []string
}

// String renders one negative verdict.
func (a AbsenceFinding) String() string {
	if len(a.Hits) == 0 {
		return fmt.Sprintf("%s %s (no %q under %s)", a.Absence.ID, a.Status, a.Absence.Symbol, a.Absence.Scope)
	}
	return fmt.Sprintf("%s %s %s", a.Absence.ID, a.Status, strings.Join(a.Hits, ", "))
}

// maxHits caps how many locations one negative verdict reports.
const maxHits = 4

// Claims returns the statements the M12 argument rests on. Each is pinned to the
// code that decides it, so the need for EAP-PWD is re-derived from the tree
// rather than recalled from a roadmap note.
func Claims() []Claim {
	return []Claim{
		{
			ID:        "password-storage-is-cleartext",
			Statement: "A RADIUS account stores the password the server compares, so challenge-response methods recompute a response from a secret the server holds.",
			File:      "internal/domain/radius.go",
			Symbol:    "Password",
			Gap:       "EAP-PWD is the only method that authenticates a password without the server ever holding it in a recomputable form.",
		},
		{
			ID:        "md5-needs-cleartext",
			Statement: "EAP-MD5 computes its expected response from the stored password.",
			File:      "internal/radiusd/plugins/eap/handlers/md5_handler.go",
			Symbol:    "GetPassword",
			Gap:       "Every non-tunneled password method in the tree inherits this exposure.",
		},
		{
			ID:        "mschapv2-needs-nt-response",
			Statement: "EAP-MSCHAPv2 builds the NT response the peer sent, which requires the password or its NT hash.",
			File:      "internal/radiusd/plugins/eap/handlers/mschapv2_handler.go",
			Symbol:    "GenerateNTResponse",
			Gap:       "The same exposure is what makes PEAP's inner method the weak half of PEAP.",
		},
		{
			ID:        "peap-inner-is-mschapv2-only",
			Statement: "PEAP's inner exchange accepts only EAP-MSCHAPv2; an inner response of any other EAP type is a protocol violation that rejects.",
			File:      "internal/radiusd/plugins/eap/handlers/peap_inner.go",
			Symbol:    "TypeMSCHAPv2",
			Gap:       "A password method that is not MS-CHAPv2 has no place inside the tunnel that exists today.",
		},
		{
			ID:        "ttls-inner-stops-at-pap-and-mschapv2",
			Statement: "EAP-TTLS answers inner PAP and MS-CHAP-V2, and returns a not-implemented error for anything else, so tunneled CHAP and MS-CHAP are declared but absent.",
			File:      "internal/radiusd/plugins/eap/handlers/ttls_inner.go",
			Symbol:    "ErrTTLSInnerNotImplemented",
			Gap:       "The tunnel that would carry an EAP-PWD exchange for a certificate-less client is only half built.",
		},
		{
			ID:        "eap-tls-authenticates-a-certificate-identity",
			Statement: "EAP-TLS grants only after mapping a verified peer-certificate identity, and rejects when the certificate carries no usable identity.",
			File:      "internal/radiusd/plugins/eap/handlers/tls_handler.go",
			Symbol:    "ErrTLSNoIdentity",
			Gap:       "This is the requirement EAP-PWD is proposed against: password authentication with no per-client certificate.",
		},
		{
			ID:        "ldap-bind-only-releases-cleartext",
			Statement: "The directory backend refuses to hand out a secret usable for challenge-response, so it serves only methods where the server sees the password.",
			File:      "internal/radiusd/ldap_backend.go",
			Symbol:    "GetPassword",
			Gap:       "A directory account can therefore never be proved by a method that needs the password hash.",
		},
		{
			ID:        "config-enum-has-no-password-only-method",
			Statement: "The selectable methods are eap-md5, eap-mschapv2, eap-tls, eap-peap and eap-ttls.",
			File:      "internal/app/config_manager.go",
			Symbol:    "eap-ttls",
			Gap:       "Nothing in the list is certificate-free and hash-free at the same time.",
		},
		{
			ID:        "unregistered-methods-are-dropped",
			Statement: "A configured method with no registered handler is removed from the allowlist rather than silently accepted.",
			File:      "internal/radiusd/auth_stages.go",
			Symbol:    "EapEnabledHandlers",
			Gap:       "Adding EAP-PWD means a handler, a registration and an enum entry, in that order; the fail-closed contract is already documented in docs/eap-otp-contract.md.",
		},
		{
			ID:        "otp-is-fail-closed-by-design",
			Statement: "The one-time-password method is declared but refuses to authenticate because no validator backend is configured.",
			File:      "internal/radiusd/plugins/eap/handlers/otp_handler.go",
			Symbol:    "ErrOTPNotConfigured",
			Gap:       "There is already a precedent for shipping a method type that denies rather than guesses.",
		},
		{
			ID:        "type-constants-stop-short-of-pwd",
			Statement: "The declared EAP type constants top out at EAP-TLV, so type 52 has no home yet.",
			File:      "internal/radiusd/plugins/eap/interfaces.go",
			Symbol:    "TypeTLV",
			Gap:       "interfaces.go is where a TypePWD constant would be added.",
		},
	}
}

// Absences returns the negative claims: what must not exist for the M12 need
// statement to be a gap rather than a duplicate.
func Absences() []Absence {
	return []Absence{
		{
			ID:        "no-pwd-method-anywhere",
			Statement: "No EAP-PWD method name exists in the product code.",
			Symbol:    "eap-pwd",
			Scope:     "internal/",
		},
		{
			ID:        "no-pwd-type-constant",
			Statement: "The EAP type assigned to EAP-PWD is not declared.",
			Symbol:    "TypePWD",
			Scope:     "internal/",
		},
		{
			ID:        "no-dragonfly-kex",
			Statement: "The DRP key exchange EAP-PWD is built on is not implemented.",
			Symbol:    "dragonfly",
			Scope:     "internal/",
		},
	}
}

// Verify reads the files the claims cite and checks each one. A missing file is
// reported as broken with its path, because a citation to a moved file is exactly
// the kind of drift that lets a stale argument survive.
func Verify(root string) (Report, error) {
	var report Report
	for _, claim := range Claims() {
		finding := Finding{Claim: claim, Note: claim.Gap}
		//nolint:gosec // G304: paths from the catalog, resolved under the repository root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claim.File)))
		if err != nil {
			finding.Status = StatusBroken
			finding.Evidence = "file unreadable"
			report.Findings = append(report.Findings, finding)
			continue
		}
		lines := codeLines(string(raw))
		if at := locate(lines, claim.Symbol); at > 0 {
			finding.Status = StatusConfirmed
			finding.Evidence = fmt.Sprintf("%s:%d", claim.File, at)
		} else {
			finding.Status = StatusBroken
			finding.Evidence = fmt.Sprintf("symbol %q not in %s", claim.Symbol, claim.File)
		}
		report.Findings = append(report.Findings, finding)
	}

	for _, absence := range Absences() {
		found, err := scanFor(root, absence)
		if err != nil {
			return Report{}, err
		}
		status := StatusConfirmed
		if len(found) > 0 {
			status = StatusVacated
		}
		report.Absences = append(report.Absences, AbsenceFinding{Absence: absence, Status: status, Hits: found})
	}

	vendoring, err := AuditVendoring(root)
	if err != nil {
		return Report{}, err
	}
	report.Totals = Totals{
		Claims:      len(report.Findings),
		Absences:    len(report.Absences),
		FilesRead:   len(report.Findings),
		RFCFindings: vendoring.Findings(),
		DocsCarried: len(vendoring.Docs),
	}
	for _, f := range report.Findings {
		if f.Status == StatusConfirmed {
			report.Totals.Confirmed++
		}
	}
	for _, a := range report.Absences {
		if a.Status == StatusConfirmed {
			report.Totals.AbsencesHeld++
		}
	}
	report.Vendoring = vendoring
	sort.SliceStable(report.Findings, func(i, j int) bool { return report.Findings[i].Claim.ID < report.Findings[j].Claim.ID })
	return report, nil
}

// Totals summarizes a verification run.
type Totals struct {
	// Claims counts the positive statements checked.
	Claims int
	// Confirmed counts claims whose symbol was found in the cited file.
	Confirmed int
	// Absences counts the negative claims checked.
	Absences int
	// AbsencesHeld counts negative claims still true.
	AbsencesHeld int
	// FilesRead counts cited files.
	FilesRead int
	// RFCFindings counts vendoring problems under docs/rfcs.
	RFCFindings int
	// DocsCarried counts specification files found under docs/rfcs.
	DocsCarried int
}

// Report is one verification run.
type Report struct {
	// Findings is ordered by claim ID.
	Findings []Finding
	// Absences is ordered as Absences returns.
	Absences []AbsenceFinding
	// Vendoring is the docs/rfcs audit for the cited specifications.
	Vendoring Vendoring
	// Totals summarizes both halves.
	Totals Totals
}

// locate returns the first line number (one-based) whose identifier tokens
// include name, or 0. Matching whole tokens keeps a longer name from proving a
// shorter claim.
func locate(lines []string, name string) int {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
	for i, line := range lines {
		if re.MatchString(line) {
			return i + 1
		}
	}
	return 0
}

// codeLines splits source into lines with comments removed, keeping line numbers
// intact. A claim is proved by an identifier the compiler sees, not by a sentence
// that mentions it, so a doc comment naming a symbol is not evidence that the
// symbol exists.
func codeLines(text string) []string {
	lines := strings.Split(text, "\n")
	out := make([]string, len(lines))
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		out[i] = stripTrailingComment(line)
	}
	return out
}

// stripTrailingComment removes a trailing // comment. A line whose quotes are
// unbalanced before the slashes is left alone, because the slashes are inside a
// string literal rather than starting a comment.
func stripTrailingComment(line string) string {
	at := strings.Index(line, "//")
	if at < 0 {
		return line
	}
	if strings.Count(line[:at], `"`)%2 != 0 {
		return line
	}
	return line[:at]
}

// scanFor searches the scope for an absent symbol. The prototype's own directory
// is excluded: this package quotes the statements it tests, so it would otherwise
// refute every negative claim it makes.
func scanFor(root string, absence Absence) ([]string, error) {
	base := filepath.Join(root, filepath.FromSlash(absence.Scope))
	if _, err := os.Stat(base); err != nil {
		return nil, fmt.Errorf("eappwdneed: absence scope %s: %w", absence.Scope, err)
	}
	self := filepath.ToSlash(filepath.Join(absence.Scope, "eappwdneed")) + "/"
	var hits []string
	re := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(absence.Symbol))
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		slash := filepath.ToSlash(rel)
		if strings.HasPrefix(slash, self) || !strings.HasSuffix(slash, ".go") {
			return nil
		}
		//nolint:gosec // G304: paths walked under the repository's own internal tree.
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if re.Match(raw) {
			hits = append(hits, slash)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("eappwdneed: scanning %s: %w", absence.Scope, err)
	}
	sort.Strings(hits)
	if len(hits) > maxHits {
		hits = hits[:maxHits]
	}
	return hits, nil
}
