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
// either quotes a document the repository carries, or says in its note that the
// specification is not vendored. A clause nobody has read cannot be cited.
func TestEveryCitationIsQuotableHere(t *testing.T) {
	for _, m := range Mechanics() {
		if m.Source.Document == "" {
			assert.Empty(t, m.Source.Quote, "%s quotes text but names no document", m.ID)
			assert.True(t, strings.Contains(m.Note, "docs/rfcs"),
				"%s cites %s with no vendored document, so its note must say so", m.ID, m.Source.RFC)
			continue
		}
		doc := readRepoFile(t, "docs/rfcs/"+m.Source.Document)
		assert.Contains(t, collapseSpace(doc), collapseSpace(m.Source.Quote),
			"%s cites %s with a quote that is not in %s", m.ID, m.Source, m.Source.Document)
	}
}

// TestCommentMentionsAreNotEvidence pins the rule that makes the Crypto-Binding
// verdict true: the word appears in the tree, but only inside comments, and the
// PEAP comment says the TLV is skipped.
func TestCommentMentionsAreNotEvidence(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "crypto-binding-tlv")
	assert.Equal(t, StatusMissing, finding.Status)
	assert.Empty(t, finding.Support)

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
		"result-tlv", "tunneled-eap-framing", "inner-avp-codec", "tls-fragmentation",
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

func TestRealTreeVerdictsForBuildMechanics(t *testing.T) {
	report := realReport(t)
	for _, id := range []string{
		"teap-eap-type", "teap-method-name", "teap-handler-wiring", "teap-config-choice",
		"teap-version-negotiation", "eap-payload-tlv", "intermediate-result-tlv", "pac-tlv",
		"inner-method-chaining", "identity-type-tlv", "authority-id-tlv", "basic-password-auth-tlv",
	} {
		finding := find(t, report, id)
		assert.Equal(t, StatusMissing, finding.Status, "%s must not be claimed as built", id)
		assert.Zero(t, finding.MarkerHits, id)
	}
}

func TestTLSTunnelVerdictReportsBothSides(t *testing.T) {
	report := realReport(t)
	finding := find(t, report, "tls13-outer-tunnel")
	assert.Equal(t, StatusBlocked, finding.Status)
	assert.NotEmpty(t, finding.Against, "the TLS 1.2 pin must be quoted")
	assert.NotEmpty(t, finding.Support, "the TLS 1.3 support that does exist must be quoted too")
	assert.Contains(t, finding.Against[0].Text, "MaxVersion")
}

func TestTotalsAccountForEveryMechanism(t *testing.T) {
	report := realReport(t)
	assert.Equal(t, len(Mechanics()), report.Totals.Mechanics)
	assert.Equal(t, report.Totals.Mechanics,
		report.Totals.Ready+report.Totals.Partial+report.Totals.Blocked+report.Totals.Missing)
	assert.Greater(t, report.Totals.Missing, report.Totals.Ready,
		"TEAP is a build project, not a configuration toggle")
}

func TestStatusRulesFromASyntheticTree(t *testing.T) {
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/teap_handler.go": {
			"const TypeTEAP = 39",
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
	for _, line := range []string{"const TypeTEAPv2Extra = 40", "const xTypeTEAP = 39"} {
		tree := Tree{Files: map[string][]string{
			"internal/radiusd/plugins/eap/handlers/other.go": {line},
		}}
		assert.Equal(t, StatusMissing, find(t, Analyze(tree), "teap-eap-type").Status, line)
	}
	tree := Tree{Files: map[string][]string{
		"internal/radiusd/plugins/eap/handlers/other.go": {"const TypeTEAP = 39"},
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
	assert.Contains(t, line, "RFC 7170 §4.2.4")
	assert.Contains(t, line, "evidence=internal/radiusd/plugins/eap")

	blocked := find(t, report, "tls13-outer-tunnel").String()
	assert.Contains(t, blocked, "contradicted-by=")

	missing := find(t, report, "pac-tlv").String()
	assert.Contains(t, missing, "evidence=none")
}

func TestRenderMarkdownSeparatesReuseBuildAndUnknown(t *testing.T) {
	report := realReport(t)
	rendered := RenderMarkdown(report)
	assert.Contains(t, rendered, "## Reusable now")
	assert.Contains(t, rendered, "## To build")
	assert.Contains(t, rendered, "## What this scan cannot answer")
	assert.Contains(t, rendered, "| result-tlv | RFC 7170 §4.2.4 | internal/radiusd/plugins/eap")
	assert.Contains(t, rendered, "`pac-tlv` — ")
	assert.Contains(t, rendered, "Which supplicants offer TEAP")
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
