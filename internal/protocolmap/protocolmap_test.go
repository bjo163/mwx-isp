package protocolmap

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refFixture(number int, slug string) Reference {
	return Reference{Number: number, Path: "docs/rfcs/rfc" + slug}
}

func sourceFixture(p string, isTest bool, numbers ...int) Source {
	return Source{Path: p, IsTest: isTest, RFCs: numbers}
}

func featureFixture(id string, paths []string, numbers ...int) Feature {
	return Feature{ID: id, Name: "feature " + id, Status: "Implemented", Paths: paths, RFCs: numbers}
}

func nodeIDs(nodes []Node) []string {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	return ids
}

func hasFinding(graph *Graph, kind, subject string) bool {
	for _, f := range graph.Findings {
		if f.Kind == kind && f.Subject == subject {
			return true
		}
	}
	return false
}

func TestBuildLinksReferenceToFeatureSourceAndTest(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt")},
		[]Feature{featureFixture("TR-F001", []string{"internal/radiusd/radius_auth.go"})},
		[]Source{
			sourceFixture("internal/radiusd/radius_auth.go", false, 2865),
			sourceFixture("internal/radiusd/radius_auth_test.go", true),
		},
	)

	assert.Equal(t, []string{"file:internal/radiusd/radius_auth.go"}, nodeIDs(graph.Targets("rfc:2865", NodeSource)))
	assert.Equal(t, []string{"feature:TR-F001"}, nodeIDs(graph.Targets("rfc:2865", NodeFeature)))
	assert.Empty(t, graph.Findings)
	assert.Equal(t, 1, graph.Stats.Linked)
	assert.Equal(t, 0, graph.Stats.Untested)
	assert.Equal(t, 1, graph.Stats.AnchoredFeatures)

	edges := map[string]string{}
	for _, e := range graph.Outgoing("file:internal/radiusd/radius_auth.go") {
		edges[e.To] = e.Via
	}
	assert.Equal(t, ViaEntryPoint, edges["feature:TR-F001"])
	assert.Equal(t, ViaColocated, edges["file:internal/radiusd/radius_auth_test.go"])
	assert.Equal(t, "docs/rfcs/rfc2865-radius-authentication.txt", graph.Nodes["rfc:2865"].Detail)
}

func TestBuildReportsUnlinkedReference(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt"), refFixture(5176, "5176-coa-disconnect.txt")},
		[]Feature{featureFixture("TR-F001", []string{"internal/radiusd/radius_auth.go"}, 2865)},
		[]Source{sourceFixture("internal/radiusd/radius_auth.go", false, 2865)},
	)

	assert.True(t, hasFinding(graph, FindingRFCUnlinked, "RFC 5176"))
	assert.False(t, hasFinding(graph, FindingRFCUnlinked, "RFC 2865"))
	assert.Equal(t, 1, graph.Stats.Unlinked)
	assert.Equal(t, 1, graph.Stats.Linked)
	for _, f := range graph.Findings {
		assert.Contains(t, f.Detail, "UNKNOWN")
	}
}

func TestBuildReportsCitationWithoutDocument(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt")},
		[]Feature{featureFixture("TR-F005", []string{"internal/radiusd/vendors"}, 5705)},
		[]Source{sourceFixture("internal/radiusd/vendors/huawei.go", false, 5705)},
	)

	assert.True(t, hasFinding(graph, FindingCitationWithoutFile, "internal/radiusd/vendors/huawei.go"))
	assert.True(t, hasFinding(graph, FindingCitationWithoutFile, "TR-F005"))
	assert.Equal(t, 1, graph.Stats.MissingFiles)
	assert.Equal(t, []string{"rfc:2865"}, nodeIDs(graph.NodesOf(NodeRFC)))
}

func TestBuildReportsSourceWithoutFeature(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2866, "2866-radius-accounting.txt")},
		[]Feature{featureFixture("TR-F002", []string{"internal/radiusd/radius_acct.go"})},
		[]Source{sourceFixture("internal/radiusd/undocumented.go", false, 2866)},
	)

	assert.True(t, hasFinding(graph, FindingSourceWithoutFeature, "internal/radiusd/undocumented.go"))
	assert.Equal(t, 0, graph.Stats.AnchoredFeatures)
}

func TestBuildReportsFeatureWithoutProtocolAnchor(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt")},
		[]Feature{
			featureFixture("TR-F001", []string{"internal/radiusd/plugins/status.go"}),
			featureFixture("TR-F006", []string{"internal/radiusd/plugins/auth"}),
		},
		[]Source{
			sourceFixture("internal/radiusd/plugins/status.go", false, 2865),
			sourceFixture("internal/radiusd/plugins/auth/checker.go", false),
		},
	)

	assert.True(t, hasFinding(graph, FindingFeatureWithoutProtocol, "TR-F006"))
	assert.False(t, hasFinding(graph, FindingFeatureWithoutProtocol, "TR-F001"))
}

func TestBuildReportsReferenceWithoutTestEvidence(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(3579, "3579-radius-eap-support.txt")},
		[]Feature{featureFixture("TR-F004", []string{"internal/radiusd/eap.go"})},
		[]Source{sourceFixture("internal/radiusd/eap.go", false, 3579)},
	)

	assert.True(t, hasFinding(graph, FindingProtocolWithoutTest, "RFC 3579"))
	assert.Equal(t, 1, graph.Stats.Untested)
}

func TestBuildCountsCitingTestAsEvidence(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(3579, "3579-radius-eap-support.txt")},
		[]Feature{featureFixture("TR-F004", []string{"internal/radiusd"})},
		[]Source{
			sourceFixture("internal/radiusd/eap.go", false, 3579),
			sourceFixture("internal/radiusd/eap_fragment_test.go", true, 3579),
		},
	)

	assert.Empty(t, graph.Findings)
	assert.Equal(t, 1, graph.Stats.Tests)
	assert.Contains(t, nodeIDs(graph.Targets("rfc:3579", NodeTest)), "file:internal/radiusd/eap_fragment_test.go")
}

func TestBuildIsIndependentOfInputOrder(t *testing.T) {
	refs := []Reference{refFixture(2865, "2865-radius-authentication.txt"), refFixture(2866, "2866-radius-accounting.txt")}
	features := []Feature{
		featureFixture("TR-F001", []string{"internal/radiusd/radius_auth.go"}),
		featureFixture("TR-F002", []string{"internal/radiusd/radius_acct.go"}, 2866),
	}
	sources := []Source{
		sourceFixture("internal/radiusd/radius_auth.go", false, 2865),
		sourceFixture("internal/radiusd/radius_acct.go", false, 2866),
	}
	forward := Build(refs, features, sources)
	reversed := Build(
		[]Reference{refs[1], refs[0]},
		[]Feature{features[1], features[0]},
		[]Source{sources[1], sources[0]},
	)

	assert.Equal(t, forward.Findings, reversed.Findings)
	assert.Equal(t, forward.Stats, reversed.Stats)
	var forwardEdges, reversedEdges []string
	for _, e := range forward.Edges() {
		forwardEdges = append(forwardEdges, e.String())
	}
	for _, e := range reversed.Edges() {
		reversedEdges = append(reversedEdges, e.String())
	}
	assert.Equal(t, forwardEdges, reversedEdges)
}

func TestOwnersOfMatchesExactAndNestedPaths(t *testing.T) {
	features := []Feature{
		featureFixture("TR-F005", []string{"internal/radiusd/vendors"}),
		featureFixture("TR-F007", []string{"internal/adminapi/users.go"}),
	}
	assert.Equal(t, []string{"TR-F005"}, ownersOf("internal/radiusd/vendors/huawei.go", features))
	assert.Equal(t, []string{"TR-F007"}, ownersOf("internal/adminapi/users.go", features))
	assert.Empty(t, ownersOf("internal/adminapi/profiles.go", features))
}

func TestRenderMarksDeadEndsUnknownAndIsDeterministic(t *testing.T) {
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt"), refFixture(5176, "5176-coa|disconnect.txt")},
		[]Feature{featureFixture("TR-F001", []string{"internal/radiusd/radius_auth.go"})},
		[]Source{sourceFixture("internal/radiusd/radius_auth.go", false, 2865)},
	)
	rendered := RenderMarkdown(graph)
	assert.Equal(t, rendered, RenderMarkdown(graph))

	assert.Contains(t, rendered, "| RFC 2865 | docs/rfcs/rfc2865-radius-authentication.txt | TR-F001 | internal/radiusd/radius_auth.go | UNKNOWN |")
	assert.Contains(t, rendered, "| RFC 5176 | docs/rfcs/rfc5176-coa\\|disconnect.txt | UNKNOWN | UNKNOWN | UNKNOWN |")
	assert.Contains(t, rendered, "| rfc_unlinked | RFC 5176 | target UNKNOWN")

	clean := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt")},
		[]Feature{featureFixture("TR-F001", []string{"internal/radiusd"})},
		[]Source{
			sourceFixture("internal/radiusd/radius_auth.go", false, 2865),
			sourceFixture("internal/radiusd/radius_auth_test.go", true, 2865),
		},
	)
	assert.Contains(t, RenderMarkdown(clean), "No dead-ended links.")
}

func TestRenderTruncatesLongTargetLists(t *testing.T) {
	var sources []Source
	for i := 0; i < maxRenderedTargets+3; i++ {
		sources = append(sources, sourceFixture("internal/radiusd/file"+strconv.Itoa(i)+".go", false, 2865))
	}
	graph := Build(
		[]Reference{refFixture(2865, "2865-radius-authentication.txt")},
		[]Feature{featureFixture("TR-F001", []string{"internal/radiusd"})},
		sources,
	)
	rendered := RenderMarkdown(graph)
	assert.Contains(t, rendered, "(+3 more)")
}

func TestScanReadsFixtureRepository(t *testing.T) {
	root := t.TempDir()
	writeFixture(t, root, "docs/rfcs/rfc2865-radius-authentication.txt", "RFC 2865\n")
	writeFixture(t, root, "docs/feature-checklist.md", strings.Join([]string{
		"| ID | Domain | Feature | Standard Scope | Existing Entry Points / Modules | Status | Development Boundary |",
		"| --- | --- | --- | --- | --- | --- | --- |",
		"| TR-F001 | RADIUS protocol | Authentication | Support Access-Request per RFC 2865. | `internal/radiusd/radius_auth.go` | Implemented | Keep the pipeline. |",
		"| TR-F002 | RADIUS protocol | Accounting | Support Accounting-Request. | `internal/radiusd/radius_inet.go` | Implemented | Keep records. |",
	}, "\n"))
	writeFixture(t, root, "internal/radiusd/radius_auth.go", "package radiusd // implements RFC 2865 §3\n")
	writeFixture(t, root, "internal/radiusd/radius_auth_test.go", "package radiusd // RFC 2865 length rule\n")
	writeFixture(t, root, "internal/radiusd/radius_acct.go", "import rfc2866 \"layeh.com/radius/rfc2866\"\n")

	graph, err := Scan(root)
	require.NoError(t, err)

	assert.Equal(t, 1, graph.Stats.References)
	assert.Equal(t, 2, graph.Stats.Features)
	assert.Equal(t, 2, graph.Stats.Sources)
	assert.Equal(t, 1, graph.Stats.Tests)
	assert.True(t, hasFinding(graph, FindingCitationWithoutFile, "internal/radiusd/radius_acct.go"),
		"an import path naming rfc2866 must count as a citation")
	assert.True(t, hasFinding(graph, FindingSourceWithoutFeature, "internal/radiusd/radius_acct.go"))

	// Skipped directories must not contribute citations.
	writeFixture(t, root, "node_modules/rfc3579.js", "RFC 3579\n")
	writeFixture(t, root, "internal/radiusd/testdata/rfc3162.txt", "RFC 3162\n")
	skipped, err := Scan(root)
	require.NoError(t, err)
	for _, f := range skipped.Findings {
		assert.NotContains(t, f.Subject, "node_modules")
		assert.NotContains(t, f.Subject, "testdata")
	}
}

func TestFindRootWalksUpToRepository(t *testing.T) {
	root, found := FindRoot(".")
	require.True(t, found, "expected to locate the repository root from the package directory")
	checklist := filepath.Join(root, "docs", "feature-checklist.md")
	require.FileExists(t, checklist)
}

// TestScanRepositoryGraph is the FUT-023 survey over this repository. It asserts
// the invariants of the derived graph, not its current row counts, so the study
// can be rerun after any protocol change.
func TestScanRepositoryGraph(t *testing.T) {
	root, found := FindRoot(".")
	if !found {
		t.Skip("repository root not found")
	}
	graph, err := Scan(root)
	require.NoError(t, err)

	require.Greater(t, graph.Stats.References, 50, "docs/rfcs should hold the full reference set")
	assert.Greater(t, graph.Stats.Linked, 20, "many references should reach code or a feature row")
	assert.Greater(t, graph.Stats.Unlinked, 0, "the survey should find dead-ended references")
	assert.Greater(t, graph.Stats.AnchoredFeatures, 5)
	assert.Equal(t, graph.Stats.References, graph.Stats.Linked+graph.Stats.Unlinked)

	byKind := map[string]int{}
	for _, f := range graph.Findings {
		byKind[f.Kind]++
		assert.Contains(t, f.Detail, "UNKNOWN", "every dead-end must be explicit: %s", f.String())
	}
	assertSortedFindings(t, graph.Findings)
	assert.Equal(t, RenderMarkdown(graph), RenderMarkdown(rescan(t, root)))

	var kinds []string
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	t.Logf("references=%d linked=%d unlinked=%d features=%d anchored=%d citing files=%d/%d tests=%d findings=%v",
		graph.Stats.References, graph.Stats.Linked, graph.Stats.Unlinked,
		graph.Stats.Features, graph.Stats.AnchoredFeatures, graph.Stats.Sources, graph.Stats.Files,
		graph.Stats.Tests, byKind)
}

// rescan derives the graph again so the render determinism check compares two
// independent runs.
func rescan(t *testing.T, root string) *Graph {
	t.Helper()
	graph, err := Scan(root)
	require.NoError(t, err)
	return graph
}

// assertSortedFindings pins the ordering the rendered report depends on.
func assertSortedFindings(t *testing.T, findings []Finding) {
	t.Helper()
	for i := 1; i < len(findings); i++ {
		assert.Less(t, findings[i-1].String(), findings[i].String(), "findings must stay sorted")
	}
}

func writeFixture(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600)) //nolint:gosec // G306: test fixture inside t.TempDir
}
