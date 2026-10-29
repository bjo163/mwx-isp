package protocolmap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// maxScannedFile bounds a single read. Generated assets and vendored payloads
// cannot hold protocol citations that matter to onboarding navigation.
const maxScannedFile = 1 << 20

var (
	// citation matches "RFC 2865" in prose, "rfc2865" in an import path, and
	// "rfc2866-..." in a document link.
	citation = regexp.MustCompile(`(?i)\brfc[\s-]*(\d{4})\b`)
	// docFileName extracts the RFC number from a docs/rfcs file name.
	docFileName = regexp.MustCompile(`^rfc(\d{4})-`)
	// featureIDPattern matches the first column of a checklist row.
	featureIDPattern = regexp.MustCompile(`^TR-F\d{3}$`)
	// backtickedPath extracts declared entry points from a checklist cell.
	backtickedPath = regexp.MustCompile("`([^`]+)`")
)

// skippedDirs are never walked: they hold dependencies, build output, or
// machine-generated corpora rather than hand-written protocol citations.
var skippedDirs = map[string]bool{
	"build": true, "coverage": true, "dist": true, "node_modules": true,
	"testdata": true, "tmp": true, "vendor": true,
}

// skipDir reports a directory the scan must not enter. Hidden directories are
// excluded because they hold tool state and nested repository copies whose
// duplicate citations would inflate every finding.
func skipDir(name string) bool {
	return strings.HasPrefix(name, ".") || skippedDirs[name]
}

// Scan derives the navigation graph for one repository root.
func Scan(root string) (*Graph, error) {
	refs, err := LoadReferences(root)
	if err != nil {
		return nil, err
	}
	features, err := LoadFeatures(root)
	if err != nil {
		return nil, err
	}
	sources, err := LoadSources(root)
	if err != nil {
		return nil, err
	}
	return Build(refs, features, sources), nil
}

// FindRoot walks upward from start until a directory holds both docs/rfcs and
// docs/feature-checklist.md, so the survey works from any package directory.
func FindRoot(start string) (string, bool) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", false
	}
	for {
		if fileExists(filepath.Join(dir, "docs", "feature-checklist.md")) &&
			dirExists(filepath.Join(dir, "docs", "rfcs")) {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// LoadReferences reads the protocol documents stored under docs/rfcs.
func LoadReferences(root string) ([]Reference, error) {
	dir := filepath.Join(root, "docs", "rfcs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read protocol references: %w", err)
	}
	var refs []Reference
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := docFileName.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		refs = append(refs, Reference{
			Number: number,
			Path:   filepath.ToSlash(filepath.Join("docs", "rfcs", entry.Name())),
		})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Number < refs[j].Number })
	return refs, nil
}

// LoadFeatures parses the feature checklist table into feature rows. A row is
// read as: ID, domain, name, scope, entry points, status, boundary.
func LoadFeatures(root string) ([]Feature, error) {
	raw, err := os.ReadFile(filepath.Join(root, "docs", "feature-checklist.md")) //nolint:gosec // G304: fixed repository path
	if err != nil {
		return nil, fmt.Errorf("read feature checklist: %w", err)
	}
	var features []Feature
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		cells := tableCells(line)
		if len(cells) < 7 || !featureIDPattern.MatchString(cells[0]) {
			continue
		}
		feature := Feature{
			ID:     cells[0],
			Name:   cells[2],
			Status: cells[5],
		}
		for _, match := range backtickedPath.FindAllStringSubmatch(cells[4], -1) {
			feature.Paths = append(feature.Paths, filepath.ToSlash(strings.TrimSpace(match[1])))
		}
		feature.RFCs = citedNumbers(strings.Join(cells, " "))
		features = append(features, feature)
	}
	return features, nil
}

// LoadSources walks the Go files of one repository and records the reference
// numbers each file cites. Test files are marked so the graph can tell evidence
// from implementation.
func LoadSources(root string) ([]Source, error) {
	var sources []Source
	err := filepath.WalkDir(root, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if p != root && skipDir(entry.Name()) {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), ".go") {
			return nil
		}
		info, err := entry.Info()
		if err != nil || info.Size() > maxScannedFile {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		raw, err := os.ReadFile(p) //nolint:gosec // G304: paths produced by WalkDir under the repository root
		if err != nil {
			return nil
		}
		cited := citedNumbers(string(raw))
		sources = append(sources, Source{
			Path:   filepath.ToSlash(rel),
			IsTest: strings.HasSuffix(entry.Name(), "_test.go"),
			RFCs:   cited,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan Go sources: %w", err)
	}
	return sources, nil
}

// citedNumbers returns the ascending unique reference numbers named in one text.
func citedNumbers(text string) []int {
	seen := map[int]bool{}
	var numbers []int
	for _, match := range citation.FindAllStringSubmatch(text, -1) {
		number, err := strconv.Atoi(match[1])
		if err != nil || seen[number] {
			continue
		}
		seen[number] = true
		numbers = append(numbers, number)
	}
	sort.Ints(numbers)
	return numbers
}

// tableCells splits one markdown table row into trimmed cells, or returns nil
// for a line that is not a data row.
func tableCells(line string) []string {
	if !strings.HasPrefix(line, "|") || !strings.HasSuffix(line, "|") {
		return nil
	}
	parts := strings.Split(strings.Trim(line, "|"), "|")
	cells := make([]string, 0, len(parts))
	for _, part := range parts {
		cells = append(cells, strings.TrimSpace(part))
	}
	return cells
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}
