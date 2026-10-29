// Package protocolmap derives the navigation graph that connects a stored RADIUS
// protocol reference to the feature contract, the Go sources citing it, and the
// tests that exercise it.
//
// It is a FUT-023 research prototype: it only reads repository text, reports the
// links that dead-end as UNKNOWN, and is imported by no product code.
package protocolmap

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

// Node kinds in the derived graph.
const (
	// NodeRFC is a protocol document stored under docs/rfcs.
	NodeRFC = "rfc"
	// NodeFeature is one row of the feature checklist.
	NodeFeature = "feature"
	// NodeSource is a non-test Go file.
	NodeSource = "source"
	// NodeTest is a Go test file.
	NodeTest = "test"
)

// Derivation labels recorded on each edge.
const (
	// ViaCite means a Go file names the reference in its text.
	ViaCite = "cite"
	// ViaChecklist means a feature row names the reference.
	ViaChecklist = "checklist"
	// ViaSource means the reference reaches a feature through a citing file.
	ViaSource = "via-source"
	// ViaEntryPoint means a file is covered by a feature's declared paths.
	ViaEntryPoint = "entry-point"
	// ViaColocated means a test file sits beside a citing source file.
	ViaColocated = "colocated"
)

// Finding kinds reported when a navigation path dead-ends.
const (
	// FindingRFCUnlinked means a stored reference is cited by no source and by
	// no feature row, so nothing in the repository navigates to it.
	FindingRFCUnlinked = "rfc_unlinked"
	// FindingCitationWithoutFile means a source or feature row names an RFC
	// number that has no document under docs/rfcs.
	FindingCitationWithoutFile = "citation_without_file"
	// FindingSourceWithoutFeature means a citing Go file is not covered by any
	// feature's declared entry points, so its contract stays unnamed.
	FindingSourceWithoutFeature = "source_without_feature"
	// FindingFeatureWithoutProtocol means a feature owns code in a package that
	// cites references but anchors no reference itself.
	FindingFeatureWithoutProtocol = "feature_without_protocol"
	// FindingProtocolWithoutTest means a reference reaches source code that has
	// no test citing it and no test file beside it.
	FindingProtocolWithoutTest = "rfc_without_test"
)

// Reference is one protocol document stored under docs/rfcs.
type Reference struct {
	// Number is the four-digit RFC number taken from the file name.
	Number int
	// Path is the repository-relative document path.
	Path string
}

// Feature is one row of the feature checklist.
type Feature struct {
	// ID is the feature contract identifier, such as TR-F004.
	ID string
	// Name is the feature title column.
	Name string
	// Status is the status column.
	Status string
	// Paths are the declared entry points of the feature.
	Paths []string
	// RFCs are the reference numbers named in the feature row text.
	RFCs []int
}

// Source is one Go file together with the reference numbers its text cites.
type Source struct {
	// Path is the repository-relative Go file path.
	Path string
	// IsTest marks a test file, which counts as test evidence.
	IsTest bool
	// RFCs are the ascending reference numbers cited in the file, possibly empty.
	RFCs []int
}

// Node is one vertex of the graph.
type Node struct {
	// ID is the typed node identifier, such as rfc:2865.
	ID string
	// Kind is one of the Node* constants.
	Kind string
	// Label is the human-facing name shown in the rendered table.
	Label string
	// Detail is the repository path for a reference or file node, and the
	// checklist status for a feature node.
	Detail string
}

// Edge is one directed link between two nodes.
type Edge struct {
	// From is the origin node ID.
	From string
	// To is the destination node ID.
	To string
	// Via is one of the Via* constants.
	Via string
}

// String renders an edge as "from -> to (via)".
func (e Edge) String() string {
	return fmt.Sprintf("%s -> %s (%s)", e.From, e.To, e.Via)
}

// Finding records one dead-end in the graph.
type Finding struct {
	// Kind is one of the Finding* constants.
	Kind string
	// Subject is the node or path that dead-ends.
	Subject string
	// Detail states the missing link as UNKNOWN.
	Detail string
}

// String renders a finding as a sortable printable line.
func (f Finding) String() string {
	return fmt.Sprintf("%s %s: %s", f.Kind, f.Subject, f.Detail)
}

// Stats summarizes coverage so a reviewer can read the graph at a glance.
type Stats struct {
	// References is the number of documents under docs/rfcs.
	References int
	// Features is the number of feature rows.
	Features int
	// Files is the number of Go files inspected.
	Files   int // Sources is the number of non-test Go files citing a reference.
	Sources int
	// Tests is the number of test files citing a reference.
	Tests int
	// Linked is the number of references with at least one outgoing edge.
	Linked int
	// Unlinked is the number of references that dead-end.
	Unlinked int
	// MissingFiles is the number of cited RFC numbers with no document.
	MissingFiles int
	// Untested is the number of linked references without test evidence.
	Untested int
	// AnchoredFeatures is the number of features with a reference edge.
	AnchoredFeatures int
}

// Graph is the derived navigation graph.
type Graph struct {
	// Nodes maps a node ID to its vertex.
	Nodes map[string]Node
	// Findings lists every dead-end, sorted by kind then subject.
	Findings []Finding
	// Stats summarizes coverage.
	Stats Stats

	edges map[string]map[string]string
}

func rfcID(number int) string    { return "rfc:" + strconv.Itoa(number) }
func featureID(id string) string { return "feature:" + id }
func fileID(p string) string     { return "file:" + p }

// New returns an empty graph.
func New() *Graph {
	return &Graph{
		Nodes: map[string]Node{},
		edges: map[string]map[string]string{},
	}
}

func (g *Graph) addNode(id, kind, label, detail string) {
	if _, ok := g.Nodes[id]; !ok {
		g.Nodes[id] = Node{ID: id, Kind: kind, Label: label, Detail: detail}
	}
}

// addEdge records a directed link, keeping the first derivation that produced it
// so repeated citations do not duplicate a rendered row.
func (g *Graph) addEdge(from, to, via string) {
	if g.edges[from] == nil {
		g.edges[from] = map[string]string{}
	}
	if _, ok := g.edges[from][to]; !ok {
		g.edges[from][to] = via
	}
}

// Outgoing returns the edges leaving one node, sorted by destination.
func (g *Graph) Outgoing(id string) []Edge {
	var out []Edge
	for to, via := range g.edges[id] {
		out = append(out, Edge{From: id, To: to, Via: via})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].To == out[j].To {
			return out[i].Via < out[j].Via
		}
		return out[i].To < out[j].To
	})
	return out
}

// NodesOf returns every node of one kind, sorted by ID.
func (g *Graph) NodesOf(kind string) []Node {
	var nodes []Node
	for _, n := range g.Nodes {
		if n.Kind == kind {
			nodes = append(nodes, n)
		}
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].ID < nodes[j].ID })
	return nodes
}

// Edges returns every edge, sorted by origin then destination.
func (g *Graph) Edges() []Edge {
	var out []Edge
	for id := range g.Nodes {
		out = append(out, g.Outgoing(id)...)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].From == out[j].From {
			return out[i].To < out[j].To
		}
		return out[i].From < out[j].From
	})
	return out
}

// Targets returns the nodes one node links to, filtered by kind, in link order.
func (g *Graph) Targets(id, kind string) []Node {
	var out []Node
	seen := map[string]bool{}
	for _, e := range g.Outgoing(id) {
		n, ok := g.Nodes[e.To]
		if !ok || n.Kind != kind || seen[e.To] {
			continue
		}
		seen[e.To] = true
		out = append(out, n)
	}
	return out
}

// finder accumulates unique findings.
type finder map[string]Finding

func (f finder) add(kind, subject, detail string) {
	key := strings.Join([]string{kind, subject, detail}, "\x00")
	if _, ok := f[key]; !ok {
		f[key] = Finding{Kind: kind, Subject: subject, Detail: detail}
	}
}

func (f finder) sorted() []Finding {
	out := make([]Finding, 0, len(f))
	for _, v := range f {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		if out[i].Subject != out[j].Subject {
			return out[i].Subject < out[j].Subject
		}
		return out[i].Detail < out[j].Detail
	})
	return out
}

// Build derives the graph from stored references, feature rows, and Go files.
// Inputs may arrive in any order; the result is deterministic.
func Build(refs []Reference, features []Feature, sources []Source) *Graph {
	g := New()
	docOf := map[int]string{}
	for _, ref := range refs {
		g.addNode(rfcID(ref.Number), NodeRFC, "RFC "+strconv.Itoa(ref.Number), ref.Path)
		docOf[ref.Number] = ref.Path
	}
	for _, f := range features {
		g.addNode(featureID(f.ID), NodeFeature, f.ID+" "+f.Name, f.Status)
	}

	nonTest := map[string]bool{}
	testFiles := map[string]bool{}
	for _, s := range sources {
		kind := NodeSource
		if s.IsTest {
			kind = NodeTest
			testFiles[s.Path] = true
		} else {
			nonTest[s.Path] = true
		}
		g.addNode(fileID(s.Path), kind, s.Path, s.Path)
		g.Stats.Files++
	}

	// A package that cites any reference is protocol-relevant, and so is every
	// package beneath it: that is the only basis used to decide whether a
	// feature is expected to anchor a reference.
	protocolDirs := map[string]bool{}
	for _, s := range sources {
		if len(s.RFCs) > 0 {
			protocolDirs[path.Dir(s.Path)] = true
		}
	}

	findings := finder{}
	missingDocs := map[int]bool{}
	citedSources := map[int]bool{}
	tested := map[int]bool{}
	anchored := map[string]bool{}
	ownsProtocolFile := map[string][]string{}

	for _, s := range sources {
		owners := ownersOf(s.Path, features)
		for _, owner := range owners {
			if protocolPackage(path.Dir(s.Path), protocolDirs) {
				ownsProtocolFile[owner] = append(ownsProtocolFile[owner], s.Path)
			}
		}
		if len(s.RFCs) == 0 {
			continue
		}
		if s.IsTest {
			g.Stats.Tests++
		} else {
			g.Stats.Sources++
		}
		if len(owners) == 0 {
			findings.add(FindingSourceWithoutFeature, s.Path,
				"feature UNKNOWN - no checklist row declares an entry point covering this file")
		}
		// A citing test is direct evidence; a test file beside a citing source is
		// indirect evidence, because it exercises that file without naming the
		// reference.
		beside := testFiles[strings.TrimSuffix(s.Path, ".go")+"_test.go"]
		for _, number := range s.RFCs {
			if _, ok := docOf[number]; !ok {
				missingDocs[number] = true
				findings.add(FindingCitationWithoutFile, s.Path,
					fmt.Sprintf("document UNKNOWN - RFC %d is cited but docs/rfcs holds no rfc%04d file", number, number))
				continue
			}
			g.addEdge(rfcID(number), fileID(s.Path), ViaCite)
			if !s.IsTest {
				citedSources[number] = true
			}
			if s.IsTest || beside {
				tested[number] = true
			}
			for _, owner := range owners {
				g.addEdge(rfcID(number), featureID(owner), ViaSource)
				g.addEdge(fileID(s.Path), featureID(owner), ViaEntryPoint)
				anchored[owner] = true
			}
		}
	}

	for _, s := range sources {
		if !s.IsTest {
			continue
		}
		base := strings.TrimSuffix(s.Path, "_test.go") + ".go"
		if nonTest[base] {
			g.addEdge(fileID(base), fileID(s.Path), ViaColocated)
		}
	}

	for _, f := range features {
		for _, number := range f.RFCs {
			if _, ok := docOf[number]; !ok {
				missingDocs[number] = true
				findings.add(FindingCitationWithoutFile, f.ID,
					fmt.Sprintf("document UNKNOWN - RFC %d is named in the row but docs/rfcs holds no rfc%04d file", number, number))
				continue
			}
			g.addEdge(rfcID(number), featureID(f.ID), ViaChecklist)
			anchored[f.ID] = true
		}
	}

	for number, doc := range docOf {
		id := rfcID(number)
		if len(g.Outgoing(id)) == 0 {
			findings.add(FindingRFCUnlinked, "RFC "+strconv.Itoa(number),
				fmt.Sprintf("target UNKNOWN - %s is stored but no Go file or feature row names RFC %d", doc, number))
			continue
		}
		if citedSources[number] && !tested[number] {
			findings.add(FindingProtocolWithoutTest, "RFC "+strconv.Itoa(number),
				"test UNKNOWN - RFC "+strconv.Itoa(number)+" reaches source code with no citing test and no colocated test file")
		}
	}

	for _, f := range features {
		if anchored[f.ID] {
			g.Stats.AnchoredFeatures++
			continue
		}
		if owned := ownsProtocolFile[f.ID]; len(owned) > 0 {
			sort.Strings(owned)
			findings.add(FindingFeatureWithoutProtocol, f.ID,
				fmt.Sprintf("protocol anchor UNKNOWN - %s owns %d protocol-package file(s) but names no RFC, first: %s", f.ID, len(owned), owned[0]))
		}
	}

	g.Findings = findings.sorted()
	for _, n := range g.Nodes {
		switch n.Kind {
		case NodeRFC:
			g.Stats.References++
		case NodeFeature:
			g.Stats.Features++
		}
	}
	for number := range docOf {
		if len(g.Outgoing(rfcID(number))) > 0 {
			g.Stats.Linked++
		} else {
			g.Stats.Unlinked++
		}
		if citedSources[number] && !tested[number] {
			g.Stats.Untested++
		}
	}
	g.Stats.MissingFiles = len(missingDocs)
	return g
}

// protocolPackage reports whether a directory, or one of its ancestors, holds a
// file that cites a reference.
func protocolPackage(dir string, relevant map[string]bool) bool {
	for current := dir; current != "" && current != "." && current != "/"; {
		if relevant[current] {
			return true
		}
		parent := path.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	return false
}

// ownersOf returns the most specific feature IDs whose declared entry points
// cover a path. A declared path matches itself and everything beneath it, which
// is how the checklist uses directory entries such as
// internal/radiusd/plugins/auth. When several rows match, only the longest
// declaration survives: a platform row that claims internal/radiusd must not
// outvote the row that names the exact file, or navigation would point every
// protocol file at the same broad contract.
func ownersOf(filePath string, features []Feature) []string {
	best := -1
	var owners []string
	for _, f := range features {
		for _, declared := range f.Paths {
			declared = strings.Trim(strings.TrimSpace(declared), "/")
			if declared == "" {
				continue
			}
			if filePath != declared && !strings.HasPrefix(filePath, declared+"/") {
				continue
			}
			switch {
			case len(declared) > best:
				best = len(declared)
				owners = append(owners[:0], f.ID)
			case len(declared) == best:
				owners = append(owners, f.ID)
			}
		}
	}
	sort.Strings(owners)
	return owners
}
