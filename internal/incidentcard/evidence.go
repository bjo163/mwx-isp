package incidentcard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// This file repeats the evidence rules the earlier studies use: a citation must
// land on a code line, an identifier must match as a whole, and drift is reported
// instead of dropped. It is a package-local copy on purpose - research packages
// that import each other stop being independent evidence.
const ownPackage = "incidentcard"

// lines returns a repo-relative file with comments blanked out, or nil when the
// file is gone. A nil return is what makes a citation read "vacated" rather than
// silently false.
func lines(root, rel string) []string {
	src, ok := readFile(filepath.Join(root, filepath.FromSlash(rel)))
	if !ok {
		return nil
	}
	return strings.Split(blankFor(rel, src), "\n")
}

// blankFor picks the comment dialect by file name.
func blankFor(rel, src string) string {
	switch filepath.Ext(filepath.ToSlash(rel)) {
	case ".yml", ".yaml":
		return blankHashComments(src)
	default:
		return blankComments(src)
	}
}

// blankHashComments turns every line-leading "#" comment into spaces of the same
// length. A "#" inside a scalar that is not at a word start stays code.
func blankHashComments(src string) string {
	runes := []rune(src)
	for i := 0; i < len(runes); i++ {
		if runes[i] != '#' {
			continue
		}
		if i > 0 && runes[i-1] != ' ' && runes[i-1] != '\t' && runes[i-1] != '\n' && runes[i-1] != '\r' {
			continue
		}
		for i < len(runes) && runes[i] != '\n' {
			runes[i] = ' '
			i++
		}
		i--
	}
	return string(runes)
}

// blankComments turns every comment into spaces of the same length, so a symbol
// mentioned only in prose cannot confirm a claim. It is quote-aware, newlines are
// preserved, and rune width is kept, because a citation is a line number.
func blankComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	runes := []rune(src)
	for i := 0; i < len(runes); {
		switch {
		case runes[i] == '"' || runes[i] == '\'' || runes[i] == '`':
			quote := runes[i]
			b.WriteRune(runes[i])
			i++
			for i < len(runes) && runes[i] != quote {
				if runes[i] == '\\' && i+1 < len(runes) {
					b.WriteRune(runes[i])
					b.WriteRune(runes[i+1])
					i += 2
					continue
				}
				b.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				b.WriteRune(runes[i])
				i++
			}
		case runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '/':
			for i < len(runes) && runes[i] != '\n' {
				b.WriteRune(' ')
				i++
			}
		case runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '*':
			b.WriteRune(' ')
			i++
			b.WriteRune(' ')
			i++
			for i < len(runes) {
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					break
				}
				if runes[i] == '\n' {
					b.WriteRune('\n')
				} else {
					b.WriteRune(' ')
				}
				i++
			}
			if i < len(runes) {
				b.WriteRune(' ')
				i++
				if i < len(runes) {
					b.WriteRune(' ')
					i++
				}
			}
		default:
			b.WriteRune(runes[i])
			i++
		}
	}
	return b.String()
}

func isIdentRune(r rune) bool {
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}

func isIdent(token string) bool {
	if token == "" {
		return false
	}
	for _, r := range token {
		if !isIdentRune(r) {
			return false
		}
	}
	return true
}

// containsBounded matches token as a whole word when it is identifier-shaped, and
// as a phrase otherwise, refusing an adjacent word character either way.
func containsBounded(text, token string) bool {
	if text == "" || token == "" {
		return false
	}
	if !isIdent(token) {
		return strings.Contains(text, token)
	}
	for i := 0; i+len(token) <= len(text); i++ {
		if text[i:i+len(token)] != token {
			continue
		}
		if i > 0 && isIdentRune(rune(text[i-1])) {
			continue
		}
		if j := i + len(token); j < len(text) && isIdentRune(rune(text[j])) {
			continue
		}
		return true
	}
	return false
}

// countMatches counts the code lines in one file that carry the token.
func countMatches(root, rel, tok string) int {
	n := 0
	for _, ln := range lines(root, rel) {
		if containsBounded(ln, tok) {
			n++
		}
	}
	return n
}

// firstLine returns the 1-based line of the first code line that carries the
// token, or 0 when the file is gone or the token never appears.
func firstLine(root, rel, tok string) int {
	for i, ln := range lines(root, rel) {
		if containsBounded(ln, tok) {
			return i + 1
		}
	}
	return 0
}

// walkFiles lists the files under dir with one of the given suffixes, skipping
// this prototype's own package and the build output.
func walkFiles(root, dir string, suffixes ...string) []string {
	var out []string
	base := filepath.Join(root, filepath.FromSlash(dir))
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ownPackage || d.Name() == "node_modules" || d.Name() == "dist" ||
				d.Name() == "playwright-report" || d.Name() == "test-results" {
				return filepath.SkipDir
			}
			return nil
		}
		for _, s := range suffixes {
			if strings.HasSuffix(path, s) {
				rel, rerr := filepath.Rel(root, path)
				if rerr != nil {
					return nil
				}
				out = append(out, filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil
	}
	sort.Strings(out)
	return out
}

// readFile is the only file access this package performs besides walking
// directories; it never writes.
func readFile(path string) (string, bool) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: the path comes from the scanned tree.
	if err != nil {
		return "", false
	}
	return string(b), true
}

func ownPackageDir(root string) string {
	return filepath.Join(root, "internal", ownPackage)
}

// parseGo reads one Go file as code. The parser is given the comment-blanked
// source so a doc comment can never contribute a field or a string literal.
func parseGo(root, rel string) (*ast.File, *token.FileSet, bool) {
	src, ok := readFile(filepath.Join(root, filepath.FromSlash(rel)))
	if !ok {
		return nil, nil, false
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, blankComments(src), 0)
	if err != nil {
		return nil, nil, false
	}
	return f, fset, true
}

// stringLiterals returns every string literal in a file with its line, in order.
func stringLiterals(root, rel string) []literal {
	f, fset, ok := parseGo(root, rel)
	if !ok {
		return nil
	}
	var out []literal
	ast.Inspect(f, func(n ast.Node) bool {
		bl, isLit := n.(*ast.BasicLit)
		if !isLit || bl.Kind != token.STRING {
			return true
		}
		out = append(out, literal{
			Text: strings.Trim(bl.Value, "`\""),
			Line: fset.Position(bl.Pos()).Line,
		})
		return true
	})
	return out
}

type literal struct {
	Text string
	Line int
}

// packageImports returns the import paths this package declares, test files
// included, so the read-only promise covers the whole package.
func packageImports(root, dir string) []string {
	set := map[string]bool{}
	fset := token.NewFileSet()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		f, perr := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, parser.ImportsOnly)
		if perr != nil {
			continue
		}
		for _, imp := range f.Imports {
			set[strings.Trim(imp.Path.Value, `"`)] = true
		}
	}
	var out []string
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// importersOutside lists product files that name this package, which must stay
// empty: the study is evidence, not a dependency.
func importersOutside(root string) []string {
	var out []string
	for _, rel := range append(walkFiles(root, "internal", ".go"), walkFiles(root, "cmd", ".go")...) {
		if strings.Contains(filepath.ToSlash(rel), "/"+ownPackage+"/") {
			continue
		}
		src, ok := readFile(filepath.Join(root, filepath.FromSlash(rel)))
		if !ok {
			continue
		}
		if strings.Contains(src, "mwx-isp/internal/"+ownPackage) {
			out = append(out, rel)
		}
	}
	sort.Strings(out)
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

func yesNo(v bool) string {
	if v {
		return "yes"
	}
	return "no"
}

// orNone keeps an empty cell out of a printed table.
func orNone(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
