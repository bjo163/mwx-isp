package policycheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Annotation is the exemption comment the production ratchet also understands.
// A statement may stay unscoped only when it carries this prefix plus a reason.
const Annotation = "tenancy:unscoped:"

var rawMethods = map[string]bool{"Raw": true, "Exec": true, "Table": true, "Joins": true}

// predicateMethods carry a SQL fragment that bounds rows but does not
// introduce a relation. They are read for their literal text only; a
// non-literal argument is left unchecked rather than making the whole
// statement opaque.
var predicateMethods = map[string]bool{"Where": true, "Having": true}

// Site is one raw-SQL statement found in the tree.
type Site struct {
	// File is the repository-relative path with forward slashes.
	File string
	// Line is the line of the first raw-SQL call in the statement.
	Line int
	// SQL is the joined text of every literal raw-SQL argument in the
	// statement, in source order.
	SQL string
	// Dynamic is true when the statement text cannot be checked statically:
	// an argument is built at runtime, the driving table comes from Model,
	// or no relation could be read from the text.
	Dynamic bool
	// SkipReason explains Dynamic for the survey report.
	SkipReason string
	// Annotated is true when the call carries a reasoned exemption comment.
	Annotated bool
	// Report is the scope classification; empty for dynamic sites.
	Report Report
}

// Scan walks root/internal, root/cmd and root/pkg for raw-SQL statements.
func Scan(root string) ([]Site, error) {
	var sites []Site
	for _, dir := range []string{"internal", "cmd", "pkg"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if entry == nil {
					return nil
				}
				return err
			}
			if entry.IsDir() {
				name := entry.Name()
				if path != base && (strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "node_modules") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			src, readErr := os.ReadFile(path) //nolint:gosec // path is walked from the repository root
			if readErr != nil {
				return readErr
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			found, scanErr := scanFile(filepath.ToSlash(rel), src)
			if scanErr != nil {
				return scanErr
			}
			sites = append(sites, found...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(sites, func(i, j int) bool {
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		return sites[i].Line < sites[j].Line
	})
	return sites, nil
}

// Findings returns the sites a table-level policy would reject: a statement
// whose text is known, references at least one unscoped relation, and carries
// no reasoned exemption.
func Findings(sites []Site) []Site {
	var findings []Site
	for _, site := range sites {
		if site.Dynamic || site.Annotated || site.Report.Scoped() {
			continue
		}
		findings = append(findings, site)
	}
	return findings
}

// scanFile returns one Site per Go statement that contains a raw-SQL call.
func scanFile(file string, src []byte) ([]Site, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", file, err)
	}

	seen := make(map[token.Pos]bool)
	var sites []Site
	var stack []ast.Node

	ast.Inspect(parsed, func(n ast.Node) bool {
		if n == nil {
			stack = stack[:len(stack)-1]
			return true
		}
		stack = append(stack, n)

		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || !rawMethods[sel.Sel.Name] {
			return true
		}
		scope := enclosingStatement(stack)
		if scope == nil || seen[scope.Pos()] {
			return true
		}
		seen[scope.Pos()] = true
		sites = append(sites, buildSite(fset, parsed, file, scope))
		return true
	})
	return sites, nil
}

// enclosingStatement returns the innermost statement (or declaration for
// package-level initialisers) that contains the call, which is the unit a GORM
// chain is written as.
func enclosingStatement(stack []ast.Node) ast.Node {
	for i := len(stack) - 1; i >= 0; i-- {
		if _, ok := stack[i].(ast.Stmt); ok {
			return stack[i]
		}
	}
	for i := len(stack) - 1; i >= 0; i-- {
		if _, ok := stack[i].(ast.Decl); ok {
			return stack[i]
		}
	}
	return nil
}

func buildSite(fset *token.FileSet, parsed *ast.File, file string, scope ast.Node) Site {
	site := Site{File: file, Line: fset.Position(scope.Pos()).Line}
	var parts []string
	var firstCall *ast.CallExpr
	hasModel := false
	hasRaw := false

	ast.Inspect(scope, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if sel.Sel.Name == "Model" {
			hasModel = true
			return true
		}
		if predicateMethods[sel.Sel.Name] {
			if len(call.Args) > 0 {
				if lit, ok := call.Args[0].(*ast.BasicLit); ok && lit.Kind == token.STRING {
					keyword := "WHERE"
					if sel.Sel.Name == "Having" {
						keyword = "HAVING"
					}
					parts = append(parts, keyword+" "+unquote(lit.Value))
				}
			}
			return true
		}
		if !rawMethods[sel.Sel.Name] {
			return true
		}
		hasRaw = true
		if firstCall == nil {
			firstCall = call
		}
		if len(call.Args) == 0 {
			site.Dynamic = true
			site.SkipReason = "runtime SQL"
			return true
		}
		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			site.Dynamic = true
			site.SkipReason = "runtime SQL"
			return true
		}
		text := unquote(lit.Value)
		// Table carries a relation without the clause keyword the analyzer
		// recognizes, so restore it.
		if sel.Sel.Name == "Table" {
			text = "FROM " + text
		}
		parts = append(parts, text)
		return true
	})

	site.SQL = strings.Join(parts, "\n")
	if firstCall != nil {
		site.Line = fset.Position(firstCall.Pos()).Line
		site.Annotated = annotatedUnscoped(fset, parsed, firstCall)
	}
	if hasRaw && hasModel {
		site.Dynamic = true
		site.SkipReason = "model-based driving table"
	}
	if !site.Dynamic {
		site.Report = Analyze(site.SQL)
		if len(site.Report.Tables) == 0 {
			site.Dynamic = true
			site.SkipReason = "no analysable relation"
		}
	}
	return site
}

func unquote(raw string) string {
	if len(raw) < 2 {
		return raw
	}
	switch raw[0] {
	case '`':
		return raw[1 : len(raw)-1]
	case '"':
		return strings.ReplaceAll(raw[1:len(raw)-1], `\"`, `"`)
	}
	return raw
}

// annotatedUnscoped accepts the exemption on the call's own line or the line
// directly above it, with a non-empty reason.
func annotatedUnscoped(fset *token.FileSet, parsed *ast.File, call *ast.CallExpr) bool {
	target := fset.Position(call.Pos()).Line
	for _, group := range parsed.Comments {
		end := fset.Position(group.End()).Line
		if end != target && end != target-1 {
			continue
		}
		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
			text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))
			if !strings.HasPrefix(text, Annotation) {
				continue
			}
			if strings.TrimSpace(strings.TrimPrefix(text, Annotation)) != "" {
				return true
			}
		}
	}
	return false
}
