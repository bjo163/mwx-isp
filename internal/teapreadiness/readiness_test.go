package teapreadiness

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// repoRoot walks up from this file's directory to the module root, so the survey
// runs against the repository it documents rather than a fixture.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, here, _, ok := runtime.Caller(0)
	require.True(t, ok, "resolve the caller path")
	dir := filepath.Dir(here)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		require.NotEqual(t, dir, parent, "reached the filesystem root without go.mod")
		dir = parent
	}
}

// readRepoFile loads a repository file for citation checks.
func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	//nolint:gosec // G304: paths built from the repository root and catalog entries.
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(rel)))
	require.NoError(t, err, "read %s", rel)
	return string(raw)
}

// find returns the finding for one mechanism ID.
func find(t *testing.T, report Report, id string) Finding {
	t.Helper()
	for _, f := range report.Findings {
		if f.Mechanic.ID == id {
			return f
		}
	}
	t.Fatalf("no finding for %q", id)
	return Finding{}
}

// realReport scans the repository this package lives in.
func realReport(t *testing.T) Report {
	t.Helper()
	tree, err := Scan(repoRoot(t))
	require.NoError(t, err)
	return Analyze(tree)
}

func TestCatalogIDsAreUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range Mechanics() {
		require.False(t, seen[m.ID], "duplicate mechanism id %q", m.ID)
		seen[m.ID] = true
		assert.NotEmpty(t, m.Title, m.ID)
		assert.NotEmpty(t, m.Note, m.ID)
		assert.NotEmpty(t, m.Markers, m.ID)
	}
}

// TestEveryCitationIsQuotableHere is the anti-fabrication gate: a mechanism
// quotes a phrase out of a document this repository carries. M11.1 vendored the
// last two missing specifications (RFC 9930 and RFC 9427), so an entry with no
// document is now a defect rather than a disclosed exemption: a clause nobody
// has read cannot be cited.
func TestEveryCitationIsQuotableHere(t *testing.T) {
	for _, m := range Mechanics() {
		require.NotEmpty(t, m.Source.Document,
			"%s cites %s without naming a vendored document", m.ID, m.Source.RFC)
		assert.NotEmpty(t, m.Source.Quote, "%s names %s but quotes nothing", m.ID, m.Source.Document)
		doc := readRepoFile(t, "docs/rfcs/"+m.Source.Document)
		assert.Contains(t, collapseSpace(doc), collapseSpace(m.Source.Quote),
			"%s cites %s with a quote that is not in %s", m.ID, m.Source, m.Source.Document)
	}
}

// TestNoCitationPointsAtAnObsoletedSpecification pins the re-anchoring done by
// M11.1: RFC 9930 obsoletes RFC 7170, so the requirement catalog cites the
// current text. The obsolete document stays in docs/rfcs as history, but a
// mechanism read as a build requirement must not cite it.
func TestNoCitationPointsAtAnObsoletedSpecification(t *testing.T) {
	for _, m := range Mechanics() {
		assert.NotEqual(t, "rfc7170-teap.txt", m.Source.Document,
			"%s cites RFC 7170, which RFC 9930 obsoletes", m.ID)
		assert.NotEqual(t, "RFC 7170", m.Source.RFC, "%s cites an obsoleted specification", m.ID)
	}
}

// TestObsoletedSpecificationIsStillArchived keeps the re-anchoring honest in the
// other direction: the catalog moved off RFC 7170 because the newer text is in
// the archive, not because the older one was deleted.
func TestObsoletedSpecificationIsStillArchived(t *testing.T) {
	assert.Contains(t, readRepoFile(t, "docs/rfcs/rfc9930-teap-version-1.txt"),
		"Obsoletes: 7170", "the TEAP catalog cites RFC 9930 because it obsoletes RFC 7170")
	assert.FileExists(t, filepath.Join(repoRoot(t), "docs", "rfcs", "rfc7170-teap.txt"))
}

// TestCommentMentionsAreNotEvidence pins the rule that sets the Crypto-Binding
// verdict: the word appears in peap_inner.go only inside a comment saying the TLV
// is skipped, and that mention is not evidence. What supports the row is the teaptlv
// codec and the teapbind Compound MAC, and the row stays "partial" because no
// handshake puts either of them on the wire.
func TestCommentMentionsAreNotEvidence(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "crypto-binding-tlv")
	assert.Equal(t, StatusPartial, finding.Status)
	require.NotEmpty(t, finding.Support, "the codec and the MAC are real code and must be quoted")
	for _, e := range finding.Support {
		assert.Contains(t,
			[]string{"internal/radiusd/plugins/eap/teaptlv/teaptlv.go",
				"internal/radiusd/plugins/eap/teapbind/teapbind.go"},
			e.File, "only the TLV codec and the MAC computation may support this row, never a comment")
	}
	assert.Less(t, finding.MarkerHits, len(finding.Mechanic.Markers),
		"a Crypto-Binding no handshake sends must keep the row out of ready")
	assert.Contains(t, finding.Note, "§6.3", "the note must name the clause that is unimplemented")

	raw := readRepoFile(t, "internal/radiusd/plugins/eap/handlers/peap_inner.go")
	assert.Contains(t, raw, "Cryptobinding", "the RFC-adjacent comment must still be in the tree for this test to mean anything")
	assert.Contains(t, raw, "are skipped", "the comment must say the TLV is skipped, not merely mention it")

	tree, err := Scan(repoRoot(t))
	require.NoError(t, err)
	var stripped strings.Builder
	for _, path := range tree.paths() {
		stripped.WriteString(strings.Join(tree.Files[path], "\n"))
	}
	assert.NotContains(t, stripped.String(), "ryptobinding", "comments must not survive into the scanned tree")
}

// TestSubstringLookalikesDoNotProveTEAP keeps the four naming verdicts honest on
// a real tree: the generated Microsoft vendor code spells the letters "teap"
// inside msaccteaptype_del, and that must not read as support for the method.
func TestSubstringLookalikesDoNotProveTEAP(t *testing.T) {
	tree, err := Scan(repoRoot(t))
	require.NoError(t, err)

	var hits int
	for _, path := range tree.paths() {
		for _, line := range tree.Files[path] {
			if strings.Contains(strings.ToLower(line), "teap") {
				hits++
			}
		}
	}
	assert.Positive(t, hits, "the tree must still contain the lookalike substring for this test to mean anything")

	report := Analyze(tree)
	for _, id := range []string{"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice"} {
		assert.Equal(t, StatusMissing, find(t, report, id).Status, "%s was proved by a substring", id)
	}
}

func TestRealTreeVerdictsForReusedMechanics(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"result-tlv", "tunneled-eap-framing", "tls-fragmentation",
		"protected-success", "anonymous-outer-identity", "exported-keying-material",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusReady, finding.Status, "%s should be reusable: %v", id, finding.Support)
		assert.NotEmpty(t, finding.Support, id)
		for _, e := range finding.Support {
			assert.FileExists(t, filepath.Join(repoRoot(t), filepath.FromSlash(e.File)))
			assert.Positive(t, e.Line, e.File)
		}
	}
}

// TestRealTreeVerdictsForBuildMechanics keeps the rows that describe the method
// itself at "missing": a codec or a key schedule sitting in the tree must not read
// as a TEAP method the server can negotiate.
func TestRealTreeVerdictsForBuildMechanics(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice",
		"teap-version-negotiation",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusMissing, finding.Status, "%s must not be claimed as built", id)
		assert.Zero(t, finding.MarkerHits, id)
	}
}

// TestChainingRowStaysPartial records what teapinner did to the catalog: the
// sequence RFC 9930 §3.6.2 describes and the bounds of §3.6.5 are now enforced in
// the tree, so the row is no longer "missing", but no TEAP handler runs a Plan
// inside a tunnel, so it must not read as a server that can chain either.
func TestChainingRowStaysPartial(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "inner-method-chaining")
	assert.Equal(t, StatusPartial, finding.Status, "a chain no tunnel runs is a library")
	require.NotEmpty(t, finding.Support)
	for _, e := range finding.Support {
		assert.Equal(t, "internal/radiusd/plugins/eap/teapinner/teapinner.go", e.File,
			"only the chaining package may support this row")
	}
	assert.Equal(t, 2, finding.MarkerHits, "the absent marker is the handler that would own a Plan")
	assert.Contains(t, finding.Note, "teapinner", "the note must say what the package enforces")
	assert.Contains(t, finding.Note, "NewTEAPHandler", "the note must name the caller that is missing")
}

// TestTLVFormatRowsStayPartial records what the teaptlv codec did and did not
// change: every §4.2 TLV row gained its encoder, so none is "missing" any more,
// but each still lacks the second marker — the handler or the MAC that would make
// the format serve a handshake — so none reaches "ready" either.
func TestTLVFormatRowsStayPartial(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"crypto-binding-tlv", "eap-payload-tlv", "intermediate-result-tlv",
		"identity-type-tlv", "authority-id-tlv", "basic-password-auth-tlv",
		"vendor-specific-tlv",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusPartial, finding.Status, "%s should be a codec with no caller", id)
		assert.Positive(t, finding.MarkerHits, id)
		assert.Less(t, finding.MarkerHits, len(finding.Mechanic.Markers),
			"%s must stay partial: the format is not the handshake", id)
		assert.Contains(t, finding.Note, "teaptlv", "%s must say what the codec does", id)
	}
}

// TestKeyDerivationRowStaysPartial records what the teapkeys schedule did to the
// catalog: RFC 9427 §2.2 is now implemented in the tree, so the row is no longer
// "missing", but nothing runs it inside a handshake, so it must not read as a session
// whose keys were derived either.
func TestKeyDerivationRowStaysPartial(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "rfc9427-key-derivation")
	assert.Equal(t, StatusPartial, finding.Status, "a key schedule with no caller is a library")
	require.NotEmpty(t, finding.Support)
	for _, e := range finding.Support {
		assert.Equal(t, "internal/radiusd/plugins/eap/teapkeys/teapkeys.go", e.File,
			"only the derivation package may support this row")
	}
	assert.Equal(t, 1, finding.MarkerHits, "the absent marker is the caller")
	assert.Contains(t, finding.Note, "teapkeys", "the note must say what the package derives")
	assert.Contains(t, finding.Note, "teapChain", "the note must name the caller that is missing")
}

func TestTLSTunnelVerdictReportsBothSides(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "tls13-outer-tunnel")
	assert.Equal(t, StatusBlocked, finding.Status)
	assert.NotEmpty(t, finding.Against, "the TLS 1.2 pin must be quoted")
	assert.NotEmpty(t, finding.Support, "the TLS 1.3 support that does exist must be quoted too")
	assert.Contains(t, finding.Against[0].Text, "MaxVersion")
}

// TestResumptionVerdictReportsTheTicketPin is the finding that made M11.1 worth
// running: RFC 9930 replaces the deprecated PAC with TLS 1.3 session tickets, and
// the shared tunnel engine deliberately turns tickets off.
func TestResumptionVerdictReportsTheTicketPin(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "session-resumption")
	assert.Equal(t, StatusBlocked, finding.Status)
	require.NotEmpty(t, finding.Against, "the ticket pin must be quoted as the contradiction")
	assert.Contains(t, finding.Against[0].Text, "SessionTicketsDisabled: true")

	// The deprecation itself is quotable from the current specification, so the
	// row cannot quietly turn back into "build the PAC".
	assert.Contains(t, collapseSpace(finding.Mechanic.Source.Quote), "the PAC has been deprecated")
	assert.Contains(t, finding.Note, "deprecated", "the note must say the PAC is not the mechanism to build")
}

func TestTotalsAccountForEveryMechanism(t *testing.T) {
	report := realReport(t)
	assert.Equal(t, len(Mechanics()), report.Totals.Mechanics)
	assert.Equal(t, report.Totals.Mechanics,
		report.Totals.Ready+report.Totals.Partial+report.Totals.Blocked+report.Totals.Missing)
	// The work left is what a TEAP implementation still has to write: the absent
	// mechanisms plus the half of each partial one. That set must stay larger than the
	// mechanisms already reusable, or the report would read as a method that only
	// needs switching on.
	assert.Greater(t, report.Totals.Missing+report.Totals.Partial, report.Totals.Ready,
		"TEAP is a build project, not a configuration toggle")
	// Both blockers are TLS 1.3 tunnel settings the tree pins on purpose, so an
	// M11 plan starts by revisiting two existing decisions rather than by adding
	// a handler.
	assert.Equal(t, 2, report.Totals.Blocked)
	// Measured against this repository as it stands: six serving-path mechanisms
	// are reusable, nine have a piece in the tree and no caller (the seven §4.2 TLV
	// formats, the RFC 9427 key schedule and the §3.6 chaining rules), and the
	// remaining five - the method itself, its wiring, its name and its version
	// negotiation - are absent.
	assert.Equal(t, 6, report.Totals.Ready)
	assert.Equal(t, 9, report.Totals.Partial)
	assert.Equal(t, 5, report.Totals.Missing)
}

func TestStatusRulesFromASyntheticTree(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/teap_handler.go": {
			"const TypeTEAP = 55",
			"func NewTEAPHandler() *TEAPHandler { return nil }",
			"func NewTEAPHandlerWithConfig(p TLSConfigProvider) *TEAPHandler { return nil }",
		},
		"internal/app/config_manager.go": {`Enum: []string{"eap-teap"}`},
	}}
	report := Analyze(tree)
	assert.Equal(t, StatusReady, find(t, report, "teap-eap-type").Status)
	assert.Equal(t, StatusReady, find(t, report, "teap-handler-wiring").Status)
	// A configuration literal is code, not prose, so selecting the method in a
	// schema counts as support.
	assert.Equal(t, StatusReady, find(t, report, "teap-config-choice").Status)
	// Nothing is named and nothing contradicts it, so it stays missing rather
	// than borrowing evidence from the mechanism next to it.
	assert.Equal(t, StatusMissing, find(t, report, "teap-method-name").Status)
	assert.Equal(t, StatusMissing, find(t, report, "tls13-outer-tunnel").Status)
}

func TestBlockedBeatsReadyMarkers(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/tunnel.go": {
			"import \"crypto/tls\"",
			"cfg := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS12}",
		},
	}}
	finding := find(t, Analyze(tree), "tls13-outer-tunnel")
	assert.Equal(t, StatusBlocked, finding.Status)
}

func TestLookalikeNamesDoNotProveMarkers(t *testing.T) {
	// Word boundaries keep a longer or prefixed constant from proving the
	// mechanism the catalog asked for.
	for _, line := range []string{"const TypeTEAPv2Extra = 55", "const xTypeTEAP = 55"} {
		tree := Tree{Files: map[string][]string{
			"internal/radiusd/plugins/eap/handlers/other.go": {line},
		}}
		assert.Equal(t, StatusMissing, find(t, Analyze(tree), "teap-eap-type").Status, line)
	}
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/other.go": {"const TypeTEAP = 55"},
	}}
	assert.Equal(t, StatusReady, find(t, Analyze(tree), "teap-eap-type").Status)
}

func TestFindingsAreOrderedByMechanismID(t *testing.T) {
	report := realReport(t)
	var previous string
	for _, f := range report.Findings {
		assert.Greater(t, f.Mechanic.ID, previous, "findings must be ordered by id")
		previous = f.Mechanic.ID
	}
}

func TestReportListsWhatTheScanCannotAnswer(t *testing.T) {
	report := realReport(t)
	require.NotEmpty(t, report.Unknown)
	seen := map[string]bool{}
	for _, u := range report.Unknown {
		assert.NotEmpty(t, u.Topic)
		assert.False(t, seen[u.Topic], "duplicate unknown topic %q", u.Topic)
		seen[u.Topic] = true
		assert.Regexp(t, regexp.MustCompile(`ENG-\d{3}|supplicant|no file|not vendored|no text|needs a peer`),
			u.Reason, "%s must name why the repository cannot answer it", u.Topic)
	}
}

// TestNoVerdictClaimsClientCompatibility enforces the bounded scope of ENG-006:
// this scan says what the server code can do, never what a supplicant does.
func TestNoVerdictClaimsClientCompatibility(t *testing.T) {
	report := realReport(t)
	forbidden := []string{"Windows", "macOS", "iOS", "Android", "wpa_supplicant", "Interoperability"}
	for _, f := range report.Findings {
		for _, word := range forbidden {
			assert.NotContains(t, f.Note, word, "%s makes a client claim this scan cannot support", f.Mechanic.ID)
			assert.NotContains(t, f.Mechanic.Title, word, "%s makes a client claim", f.Mechanic.ID)
		}
	}
}

func TestFindingStringNamesVerdictClauseAndEvidence(t *testing.T) {
	report := realReport(t)
	line := find(t, report, "result-tlv").String()
	assert.Contains(t, line, "result-tlv ready")
	assert.Contains(t, line, "RFC 9930 §4.2.4")
	assert.Contains(t, line, "evidence=internal/radiusd/plugins/eap")

	blocked := find(t, report, "tls13-outer-tunnel").String()
	assert.Contains(t, blocked, "contradicted-by=")

	missing := find(t, report, "teap-handler-wiring").String()
	assert.Contains(t, missing, "evidence=none")

	partial := find(t, report, "crypto-binding-tlv").String()
	assert.Contains(t, partial, "partial")
	assert.Contains(t, partial, "markers=2/3", "the MAC is in the tree, the handshake that sends it is not")
	assert.Contains(t, partial, "teapbind/teapbind.go")
}

func TestRenderMarkdownSeparatesReuseBuildAndUnknown(t *testing.T) {
	report := realReport(t)
	rendered := RenderMarkdown(report)
	assert.Contains(t, rendered, "## Reusable now")
	assert.Contains(t, rendered, "## Partly present, not yet wired")
	assert.Contains(t, rendered, "## To build")
	assert.Contains(t, rendered, "## What this scan cannot answer")
	assert.Contains(t, rendered, "`teap-handler-wiring` — ")
	assert.Contains(t, rendered, "Which supplicants offer TEAP")

	// A codec nothing calls is evidence, not machinery an implementer can reuse, so
	// it must not sit in the "Reusable now" table.
	reusable := section(t, rendered, "## Reusable now", "## Partly present")
	assert.Contains(t, reusable, "| result-tlv | RFC 9930 §4.2.4 | internal/radiusd/plugins/eap")
	assert.NotContains(t, reusable, "teaptlv", "the TLV codec belongs to the partial table")
	assert.Equal(t, 6, dataRows(reusable), "only ready rows may be claimed as reusable now")

	partial := section(t, rendered, "## Partly present", "## Blocked by existing choices")
	assert.Contains(t, partial, "| crypto-binding-tlv | RFC 9930 §3.6 |")
	assert.Contains(t, partial, "internal/radiusd/plugins/eap/teaptlv/teaptlv.go")
	assert.Contains(t, partial, "internal/radiusd/plugins/eap/teapkeys/teapkeys.go")
	assert.Contains(t, partial, "internal/radiusd/plugins/eap/teapinner/teapinner.go")
	assert.Contains(t, partial, "1 of 2 markers")
	assert.Equal(t, 9, dataRows(partial), "every piece with no caller belongs in the partial table")
}

// section returns the rendered document between two headings.
func section(t *testing.T, rendered, from, until string) string {
	t.Helper()
	start := strings.Index(rendered, from)
	end := strings.Index(rendered, until)
	require.Positive(t, start, "missing heading %q", from)
	require.Positive(t, end, "missing heading %q", until)
	require.Less(t, start, end, "%q must come before %q", from, until)
	return rendered[start:end]
}

// dataRows counts the table body rows in a rendered section, leaving out its
// column header.
func dataRows(section string) int {
	return strings.Count(section, "\n| ") - 1
}

func TestRenderMarkdownEscapesPipesInCodeEvidence(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/x.go": {"cfg := &tls.Config{MaxVersion: tls.VersionTLS12 | mask}"},
	}}
	rendered := RenderMarkdown(Analyze(tree))
	assert.Contains(t, rendered, "## Blocked by existing choices")
	assert.Contains(t, rendered, `tls.VersionTLS12 \| mask`, "a pipe inside a source line must not break the table row")
	assert.NotContains(t, rendered, "| tls.VersionTLS12 | mask |")
}
