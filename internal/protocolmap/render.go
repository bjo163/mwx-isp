package protocolmap

import (
	"fmt"
	"sort"
	"strings"
)

// maxRenderedTargets bounds one table cell so the navigation document stays
// readable when a reference reaches many files.
const maxRenderedTargets = 5

// RenderMarkdown returns the onboarding view of the graph: one row per stored
// reference with its feature contracts, sources, and tests. A link that cannot
// be derived renders as UNKNOWN instead of blank, so a reader cannot mistake a
// gap for an omission.
func RenderMarkdown(g *Graph) string {
	var b strings.Builder
	b.WriteString("# RADIUS protocol navigation graph\n\n")
	fmt.Fprintf(&b, "References: %d (%d linked, %d dead-ended) · Features: %d (%d protocol-anchored) · "+
		"Citing Go files: %d source, %d test · Cited numbers without a document: %d · References without test evidence: %d\n\n",
		g.Stats.References, g.Stats.Linked, g.Stats.Unlinked,
		g.Stats.Features, g.Stats.AnchoredFeatures,
		g.Stats.Sources, g.Stats.Tests, g.Stats.MissingFiles, g.Stats.Untested)

	b.WriteString("## Reference to feature, source, and test\n\n")
	b.WriteString("| RFC | Document | Features | Sources | Tests |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, rfc := range g.NodesOf(NodeRFC) {
		number := strings.TrimPrefix(rfc.ID, "rfc:")
		document := rfc.Detail
		if document == "" {
			document = unknown
		}
		fmt.Fprintf(&b, "| RFC %s | %s | %s | %s | %s |\n",
			number, cell(document),
			cell(list(g.Targets(rfc.ID, NodeFeature), featureName)),
			cell(list(g.Targets(rfc.ID, NodeSource), nodeName)),
			cell(list(testEvidence(g, rfc.ID), nodeName)))
	}

	b.WriteString("\n## Dead-ended links\n\n")
	if len(g.Findings) == 0 {
		b.WriteString("No dead-ended links.\n")
		return b.String()
	}
	b.WriteString("| Kind | Subject | Missing link |\n")
	b.WriteString("| --- | --- | --- |\n")
	for _, f := range g.Findings {
		fmt.Fprintf(&b, "| %s | %s | %s |\n", f.Kind, cell(f.Subject), cell(f.Detail))
	}
	fmt.Fprintf(&b, "\nTotal findings: %d.\n", len(g.Findings))
	return b.String()
}

// unknown marks a link the graph could not derive.
const unknown = "UNKNOWN"

// testEvidence lists the test nodes reachable from a reference: the tests that
// cite it directly, then the tests sitting beside a citing source.
func testEvidence(g *Graph, id string) []Node {
	seen := map[string]bool{}
	out := g.Targets(id, NodeTest)
	for _, test := range out {
		seen[test.ID] = true
	}
	var beside []Node
	for _, source := range g.Targets(id, NodeSource) {
		for _, e := range g.Outgoing(source.ID) {
			if node, ok := g.Nodes[e.To]; ok && node.Kind == NodeTest && !seen[node.ID] {
				seen[node.ID] = true
				beside = append(beside, node)
			}
		}
	}
	sort.Slice(beside, func(i, j int) bool { return beside[i].ID < beside[j].ID })
	return append(out, beside...)
}

func nodeName(n Node) string { return n.Label }

// featureName shortens a feature node to its contract ID.
func featureName(n Node) string {
	if id, _, found := strings.Cut(n.Label, " "); found {
		return id
	}
	return n.Label
}

func list(nodes []Node, name func(Node) string) string {
	if len(nodes) == 0 {
		return unknown
	}
	parts := make([]string, 0, maxRenderedTargets+1)
	for i, n := range nodes {
		if i == maxRenderedTargets {
			parts = append(parts, fmt.Sprintf("(+%d more)", len(nodes)-maxRenderedTargets))
			break
		}
		parts = append(parts, name(n))
	}
	return strings.Join(parts, ", ")
}

// cell renders one markdown table cell, escaping pipes.
func cell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
