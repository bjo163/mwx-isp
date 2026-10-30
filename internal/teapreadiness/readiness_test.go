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
// codec, the teapbind Compound MAC and the Phase 2 handler that now sends and
// verifies the TLV.
func TestCommentMentionsAreNotEvidence(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "crypto-binding-tlv")
	assert.Equal(t, StatusReady, finding.Status)
	require.NotEmpty(t, finding.Support, "the codec, the MAC and their caller are real code and must be quoted")
	var files []string
	for _, e := range finding.Support {
		files = append(files, e.File)
	}
	assert.Contains(t, files, "internal/radiusd/plugins/eap/handlers/teap_phase2.go",
		"the Phase 2 handler is what turns the codec into a handshake")
	assert.NotContains(t, files, "internal/radiusd/plugins/eap/handlers/peap_inner.go",
		"a comment saying the TLV is skipped proves nothing")
	assert.Equal(t, len(finding.Mechanic.Markers), finding.MarkerHits,
		"every marker the row names must exist in product code: the compound MAC in teapbind and the caller in the handler")
	assert.Contains(t, finding.Note, "§6.3", "the note must cite the BUFFER the MAC is computed over")

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

// TestSubstringLookalikesDoNotProveTEAP keeps marker matching on identifier
// boundaries rather than on substrings: the generated Microsoft vendor code spells
// the letters "teap" inside msaccteaptype_del, and the vendor's name must not read
// as support for the method. The real tree now declares the identifiers, so the
// lookalike is checked against a tree that carries only the near-miss names.
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

	// The real tree reaches these verdicts through the declarations, not the
	// substring: the same catalog run over a tree holding only lookalikes must not.
	report := Analyze(tree)
	for _, id := range []string{"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice"} {
		assert.Equal(t, StatusReady, find(t, report, id).Status, "%s must rest on the real identifier", id)
	}

	lookalike := Tree{Files: map[string][]string{
		"internal/vendor/ms_ad.go": {"msaccteaptype_del := 1", "eapTeapFlag := 2", "teapx := 3"},
	}}
	for _, id := range []string{"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice"} {
		assert.Equal(t, StatusMissing, find(t, Analyze(lookalike), id).Status,
			"%s was proved by a substring", id)
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

// TestRealTreeVerdictsForTheMethodItself records what M11.8 and M11.9 changed: the
// method now has its type, its name, a registered handler, a configuration entry and
// a Phase 2 that re-verifies the negotiated version through the Crypto-Binding TLV,
// so those five rows are ready on the strength of the serving path, not a codec.
func TestRealTreeVerdictsForTheMethodItself(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusReady, finding.Status, "%s is wired end to end", id)
		require.NotEmpty(t, finding.Support, id)
	}

	version := find(t, report, "teap-version-negotiation")
	assert.Equal(t, StatusReady, version.Status, "negotiation and the Phase 2 re-check both exist")
	assert.Equal(t, len(version.Mechanic.Markers), version.MarkerHits,
		"the Phase 2 verification marker must be real code, not an aspirational name")
	assert.Contains(t, version.Note, "ErrTEAPVersionUnsupported",
		"the note must name what the handler does with a version it cannot speak")
	assert.Contains(t, version.Note, "Received-Ver",
		"the note must name the field §3.1 makes the TLV receiver re-check")
}

// TestChainingRowServesATunnel records what teapinner and M11.9 did to the catalog:
// the sequence RFC 9930 §3.6.2 describes, the bounds of §3.6.5 and the Phase 2 loop
// that owns a Plan and a Session are all in the tree, so the row is ready.
func TestChainingRowServesATunnel(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "inner-method-chaining")
	assert.Equal(t, StatusReady, finding.Status, "the handler now runs a chain")
	require.NotEmpty(t, finding.Support)
	var files []string
	for _, e := range finding.Support {
		files = append(files, e.File)
	}
	assert.Contains(t, files, "internal/radiusd/plugins/eap/handlers/teap_phase2.go",
		"the handler is the caller that runs the plan")
	assert.Equal(t, len(finding.Mechanic.Markers), finding.MarkerHits,
		"the full marker set proves NewPlan and NewSession live in teapinner")
	assert.Contains(t, finding.Note, "teapinner", "the note must say what the package enforces")
	assert.Contains(t, finding.Note, "startPhase2", "the note must name the caller that runs the plan")
}

// TestTLVFormatRowsServePhase2 records what M11.9 did to the catalog: every §4.2
// TLV the Phase 2 exchange uses gained its caller, so those rows are ready rather
// than a codec with an aspirational marker.
func TestTLVFormatRowsServePhase2(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"crypto-binding-tlv", "eap-payload-tlv", "intermediate-result-tlv",
		"identity-type-tlv", "basic-password-auth-tlv",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusReady, finding.Status, "%s is served by Phase 2", id)
		assert.Equal(t, len(finding.Mechanic.Markers), finding.MarkerHits,
			"%s must not keep an aspirational marker", id)
		assert.Contains(t, finding.Note, "teaptlv", "%s must say what the codec does", id)
	}
}

// TestRemainingPartialTLVRowsStayPartial records the two rows M11.9 did not close:
// the Authority-ID's provisioning step and the Vendor-Specific envelope are not
// part of authentication, so their markers stay incomplete on purpose.
func TestRemainingPartialTLVRowsStayPartial(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{"authority-id-tlv", "vendor-specific-tlv"} {
		finding := find(t, report, id)
		assert.Equal(t, StatusPartial, finding.Status, "%s still has no serving caller", id)
		assert.Positive(t, finding.MarkerHits, id)
		assert.Less(t, finding.MarkerHits, len(finding.Mechanic.Markers),
			"%s must stay partial: the format is not a provisioning exchange", id)
		assert.Contains(t, finding.Note, "teaptlv", "%s must say what the codec does", id)
	}
}

// TestKeyDerivationRowServesPhase2 records what teapkeys and M11.9 did to the
// catalog: RFC 9427 §2.2 is implemented and the Phase 2 handler now carries the
// chain into the MS-MPPE attributes, so the row is ready.
func TestKeyDerivationRowServesPhase2(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "rfc9427-key-derivation")
	assert.Equal(t, StatusReady, finding.Status, "the schedule now runs inside a handshake")
	require.NotEmpty(t, finding.Support)
	var files []string
	for _, e := range finding.Support {
		files = append(files, e.File)
	}
	assert.Contains(t, files, "internal/radiusd/plugins/eap/teapkeys/teapkeys.go",
		"the derivation must stay in the package that owns it")
	assert.Contains(t, files, "internal/radiusd/plugins/eap/handlers/teap_phase2.go",
		"the handler is the caller that advances the chain")
	assert.Equal(t, len(finding.Mechanic.Markers), finding.MarkerHits,
		"the caller marker must be real code")
	assert.Contains(t, finding.Note, "teapkeys", "the note must say what the package derives")
	assert.Contains(t, finding.Note, "startPhase2", "the note must name the caller that seeds the chain")
}

// TestTLSTunnelVerdictKeepsMethodsApart is the finding M11.2 closes and M11.8
// completes: the TEAP provider negotiates TLS 1.3 for its own tunnel, the handler
// serves it, and the TLS 1.2 cap EAP-TTLS keeps stays that method's own decision
// rather than a contradiction.
func TestTLSTunnelVerdictKeepsMethodsApart(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "tls13-outer-tunnel")
	assert.Equal(t, StatusReady, finding.Status, "the provider and the handler that serves it are both in the tree")
	assert.Empty(t, finding.Against, "the TTLS cap must not be attributed to TEAP")
	require.NotEmpty(t, finding.Support, "the TEAP provider that negotiates TLS 1.3 must be quoted")
	var files []string
	for _, e := range finding.Support {
		files = append(files, e.File)
	}
	assert.Contains(t, files, "internal/radiusd/plugins/eap/handlers/teap_config.go")
	// Evidence is capped at three lines and the provider fills them, so the caller is
	// proved where it lives rather than through the report's excerpt.
	assert.Contains(t, readRepoFile(t, "internal/radiusd/plugins/eap/handlers/teap_handler.go"),
		"ErrTEAPRequiresTLS13", "the caller that refuses a non-TLS-1.3 tunnel must exist")
	assert.Contains(t, finding.Note, "ErrTEAPRequiresTLS13", "the note must say what the handler enforces")
}

// TestResumptionVerdictReportsTheTicketBinding records the M11.9 completion of the
// row M11.2 opened: RFC 9930 replaces the deprecated PAC with TLS 1.3 session
// tickets, and the handler now associates the Phase 2 credentials with the ticket
// it issued (§3.11), refusing a ticket whose Phase 2 never completed.
func TestResumptionVerdictReportsTheTicketBinding(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "session-resumption")
	assert.Equal(t, StatusReady, finding.Status, "the ticket path and its §3.11 binding both exist")
	assert.Empty(t, finding.Against, "nothing pins tickets off in the TEAP tunnel")
	require.NotEmpty(t, finding.Support)

	// The deprecation itself is quotable from the current specification, so the
	// row cannot quietly turn back into "build the PAC".
	assert.Contains(t, collapseSpace(finding.Mechanic.Source.Quote), "the PAC has been deprecated")
	assert.Contains(t, finding.Note, "deprecated", "the note must say the PAC is not the mechanism to build")
	assert.Contains(t, finding.Note, "associate", "the note must name the step that records credentials")
	assert.Contains(t, finding.Note, "grantResumed", "the note must name the §3.5 path it enables")
}

func TestTotalsAccountForEveryMechanism(t *testing.T) {
	report := realReport(t)
	assert.Equal(t, len(Mechanics()), report.Totals.Mechanics)
	assert.Equal(t, report.Totals.Mechanics,
		report.Totals.Ready+report.Totals.Partial+report.Totals.Blocked+report.Totals.Missing)
	// Every row either serves a handshake or names the provisioning step this server
	// does not offer, so the partial table holds exactly those two.
	assert.Positive(t, report.Totals.Partial, "the provisioning and vendor rows must stay visible")
	// No mechanism contradicts the tree any more: M11.2 removed both TLS 1.3 tunnel
	// pins, so the remaining work is writing code, not revisiting decisions.
	assert.Equal(t, 0, report.Totals.Blocked)
	assert.Equal(t, 0, report.Totals.Missing, "every cataloged mechanism now has code on the serving path or a codec")
	// Measured against this repository as it stands: M11.9 gave the Phase 2 rows
	// their caller, so twenty mechanisms serve a handshake and two remain partial -
	// the §3.11 Authority-ID provisioning step and the Vendor-Specific envelope.
	assert.Equal(t, 20, report.Totals.Ready)
	assert.Equal(t, 2, report.Totals.Partial)
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
	// This tree has the handler the tunnel row names but no provider and no TLS 1.3
	// constant, so it must read as one third of a tunnel, not as a verdict borrowed
	// from the wiring row above it.
	assert.Equal(t, StatusPartial, find(t, report, "tls13-outer-tunnel").Status)
}

func TestBlockedBeatsReadyMarkers(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/teap_config.go": {
			"import \"crypto/tls\"",
			"cfg := &tls.Config{MinVersion: tls.VersionTLS13, MaxVersion: tls.VersionTLS12}",
		},
	}}
	finding := find(t, Analyze(tree), "tls13-outer-tunnel")
	assert.Equal(t, StatusBlocked, finding.Status)
}

// TestContradictionOutsideItsScopeDoesNotBlock pins AgainstIn: the TLS 1.2 cap
// EAP-TTLS sets is that method's own decision, so it must not travel to a row that
// asks what TEAP does - while the same literal inside the TEAP provider still
// blocks, as the test above proves.
func TestContradictionOutsideItsScopeDoesNotBlock(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/tls_config.go": {
			"cfg := &tls.Config{MaxVersion: tls.VersionTLS12}",
		},
	}}
	finding := find(t, Analyze(tree), "tls13-outer-tunnel")
	assert.Equal(t, StatusMissing, finding.Status)
	assert.Empty(t, finding.Against)
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

	blocked := find(t, Analyze(Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/teap_config.go": {"cfg := &tls.Config{MaxVersion: tls.VersionTLS12}"},
	}}), "tls13-outer-tunnel").String()
	assert.Contains(t, blocked, "contradicted-by=")
	assert.Contains(t, blocked, "teap_config.go")

	missing := find(t, Analyze(Tree{Files: map[string][]string{}}), "teap-handler-wiring").String()
	assert.Contains(t, missing, "evidence=none")

	partial := find(t, report, "authority-id-tlv").String()
	assert.Contains(t, partial, "partial")
	assert.Contains(t, partial, "markers=2/3", "the codec and the Start are in the tree, the provisioning step is not")
	assert.Contains(t, partial, "teap_handler.go")
}

func TestRenderMarkdownSeparatesReuseBuildAndUnknown(t *testing.T) {
	report := realReport(t)
	rendered := RenderMarkdown(report)
	assert.Contains(t, rendered, "## Reusable now")
	assert.Contains(t, rendered, "## Partly present, not yet wired")
	assert.Contains(t, rendered, "## To build")
	assert.Contains(t, rendered, "## What this scan cannot answer")
	assert.Contains(t, rendered, "Which supplicants offer TEAP")

	// Nothing is entirely absent any more: every cataloged mechanism has code, so
	// the build list is empty and its heading is all that remains.
	toBuild := section(t, rendered, "## To build", "## Notes per mechanism")
	assert.NotContains(t, toBuild, "- `", "no mechanism may still be listed as missing")

	// A row with incomplete markers is evidence, not machinery the method runs, so it
	// must not sit in the "Reusable now" table.
	reusable := section(t, rendered, "## Reusable now", "## Partly present")
	assert.Contains(t, reusable, "| result-tlv | RFC 9930 §4.2.4 | internal/radiusd/plugins/eap")
	assert.NotContains(t, reusable, "authority-id-tlv", "the provisioning row belongs to the partial table")
	assert.Equal(t, 20, dataRows(reusable), "only ready rows may be claimed as reusable now")

	partial := section(t, rendered, "## Partly present", "## Blocked by existing choices")
	assert.Contains(t, partial, "| authority-id-tlv | RFC 9930 §4.2.2 |")
	assert.Contains(t, partial, "| vendor-specific-tlv | RFC 9930 §4.2.8 |")
	assert.Contains(t, partial, "internal/radiusd/plugins/eap/teaptlv/teaptlv.go")
	assert.Contains(t, partial, "2 of 3 markers")
	assert.Contains(t, partial, "1 of 4 markers")
	assert.Equal(t, 2, dataRows(partial), "only rows with an incomplete marker set belong in the partial table")
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
		"internal/radiusd/plugins/eap/handlers/teap_config.go": {"cfg := &tls.Config{MaxVersion: tls.VersionTLS12 | mask}"},
	}}
	rendered := RenderMarkdown(Analyze(tree))
	assert.Contains(t, rendered, "## Blocked by existing choices")
	assert.Contains(t, rendered, `tls.VersionTLS12 \| mask`, "a pipe inside a source line must not break the table row")
	assert.NotContains(t, rendered, "| tls.VersionTLS12 | mask |")
}
