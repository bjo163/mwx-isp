package tenancy

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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rawSQLMethods lists the GORM entry points that bypass the model-aware tenant
// callbacks (see docs/tenant-isolation.md). Every occurrence must either carry
// a tenant predicate in the same statement or declare why it is exempt.
var rawSQLMethods = map[string]bool{
	"Raw":   true,
	"Exec":  true,
	"Table": true,
	"Joins": true,
}

// unscopedAnnotation marks a deliberate cross-tenant statement. The reason is
// mandatory so an exemption cannot be added without an explanation.
const unscopedAnnotation = "tenancy:unscoped:"

type rawSQLViolation struct {
	File   string
	Line   int
	Method string
}

func (v rawSQLViolation) String() string {
	return fmt.Sprintf("%s:%d: %s without a tenant predicate or //%s reason",
		v.File, v.Line, v.Method, unscopedAnnotation)
}

// scanRawSQL reports every Raw/Exec/Table/Joins call in src whose enclosing
// statement neither mentions tenant_id nor carries the exemption annotation.
func scanRawSQL(file string, src []byte) ([]rawSQLViolation, error) {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var (
		violations []rawSQLViolation
		stack      []ast.Node
	)
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
		method, ok := calleeName(call)
		if !ok || !rawSQLMethods[method] {
			return true
		}

		scope := statementScope(stack)
		if scope != nil && subtreeHasTenantPredicate(scope) {
			return true
		}
		if annotatedUnscoped(fset, parsed, call) {
			return true
		}

		violations = append(violations, rawSQLViolation{
			File:   file,
			Line:   fset.Position(call.Pos()).Line,
			Method: method,
		})
		return true
	})

	return violations, nil
}

func calleeName(call *ast.CallExpr) (string, bool) {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	return sel.Sel.Name, true
}

// statementScope returns the innermost statement around the current node, so a
// tenant predicate counts only when it belongs to the same statement as the raw
// call. A file-level declaration is used for initializers outside a function.
func statementScope(stack []ast.Node) ast.Node {
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

func subtreeHasTenantPredicate(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if found {
			return false
		}
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if strings.Contains(strings.ToLower(lit.Value), "tenant_id") {
			found = true
			return false
		}
		return true
	})
	return found
}

// annotatedUnscoped accepts the exemption on the call's own line (trailing
// comment) or on the line directly above it.
func annotatedUnscoped(fset *token.FileSet, parsed *ast.File, call *ast.CallExpr) bool {
	target := fset.Position(call.Pos()).Line
	for _, group := range parsed.Comments {
		end := fset.Position(group.End()).Line
		if end != target && end != target-1 {
			continue
		}
		if hasUnscopedReason(group) {
			return true
		}
	}
	return false
}

func hasUnscopedReason(group *ast.CommentGroup) bool {
	for _, c := range group.List {
		text := strings.TrimSpace(strings.TrimPrefix(c.Text, "//"))
		text = strings.TrimSpace(strings.TrimPrefix(text, "/*"))
		text = strings.TrimSpace(strings.TrimSuffix(text, "*/"))
		if !strings.HasPrefix(text, unscopedAnnotation) {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(text, unscopedAnnotation)) != "" {
			return true
		}
	}
	return false
}

// TestRawSQLTenantScopeRatchet is the CI gate behind ENG-019: a new
// Table/Raw/Exec/Joins call fails the unit lane until it is tenant-scoped or
// explains why it is exempt. Keep docs/tenant-isolation.md in sync with any
// exemption it needs.
func TestRawSQLTenantScopeRatchet(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", ".."))
	roots := []string{"internal", "cmd", "pkg"}

	var violations []rawSQLViolation
	for _, dir := range roots {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if entry == nil {
					return nil // the directory does not exist in this tree
				}
				return err
			}
			if entry.IsDir() {
				name := entry.Name()
				if path != filepath.Join(root, dir) &&
					(strings.HasPrefix(name, ".") || name == "testdata" || name == "vendor" || name == "node_modules") {
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
			found, scanErr := scanRawSQL(filepath.ToSlash(rel), src)
			if scanErr != nil {
				return scanErr
			}
			violations = append(violations, found...)
			return nil
		})
		require.NoError(t, err)
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].File != violations[j].File {
			return violations[i].File < violations[j].File
		}
		return violations[i].Line < violations[j].Line
	})

	report := make([]string, 0, len(violations))
	for _, v := range violations {
		report = append(report, v.String())
	}
	assert.Empty(t, report, "raw SQL must stay tenant-scoped; exempt a statement with //%s <reason>",
		unscopedAnnotation)
}

func TestScanRawSQLFixtures(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "scoped table query passes",
			src: `package fixture
func f(db *gorm.DB) { db.Table("isp_customer").Where("tenant_id = ?", 1).Scan(nil) }`,
			want: 0,
		},
		{
			name: "join predicates count as the tenant predicate",
			src: `package fixture
func f(db *gorm.DB) {
	db.Table("radius_online AS o").
		Joins("LEFT JOIN radius_user AS u ON u.username = o.username AND u.tenant_id = o.tenant_id").
		Scan(nil)
}`,
			want: 0,
		},
		{
			name: "unscoped table query is reported",
			src: `package fixture
func f(db *gorm.DB) { db.Table("isp_customer").Scan(nil) }`,
			want: 1,
		},
		{
			name: "unscoped raw query is reported",
			src: `package fixture
func f(db *gorm.DB) { db.Raw("SELECT * FROM radius_user").Scan(nil) }`,
			want: 1,
		},
		{
			name: "another scoped statement does not cover this one",
			src: `package fixture
func f(db *gorm.DB) {
	db.Table("isp_customer").Where("tenant_id = ?", 1).Scan(nil)
	db.Table("isp_invoice").Scan(nil)
}`,
			want: 1,
		},
		{
			name: "annotation with a reason exempts the statement",
			src: `package fixture
func f(db *gorm.DB) {
	// tenancy:unscoped: platform migration, whole-table DDL by design
	db.Exec("CREATE INDEX i ON isp_customer (name)")
}`,
			want: 0,
		},
		{
			name: "annotation without a reason is still reported",
			src: `package fixture
func f(db *gorm.DB) {
	// tenancy:unscoped:
	db.Exec("CREATE INDEX i ON isp_customer (name)")
}`,
			want: 1,
		},
		{
			name: "a tenant_id literal in the same statement passes",
			src: `package fixture
func f(db *gorm.DB) { db.Exec("DELETE FROM radius_online WHERE tenant_id = 0") }`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			violations, err := scanRawSQL("fixture.go", []byte(tt.src))
			require.NoError(t, err)
			assert.Len(t, violations, tt.want, "%v", violations)
		})
	}
}
