package profilewhatif

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// This file repeats the three evidence rules the earlier studies use: a citation
// must land on a code line, an identifier must match as a whole, and drift is
// reported instead of dropped. It is a package-local copy on purpose - two
// research packages that import each other stop being independent evidence.

// ownPackage names this prototype. Every scan of the product tree skips it, so
// no claim or gap is settled by the study's own fixtures.
const ownPackage = "profilewhatif"

// productDirs are the trees a claim may be proven from.
var productDirs = []string{"internal", "pkg"}

// productFiles walks the product directories, skipping this prototype's own
// package and every _test.go, so a claim is proven by product code and never by
// a fixture next to it.
func productFiles(root string) []string {
	var out []string
	for _, dir := range productDirs {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if filepath.Base(path) == ownPackage {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			out = append(out, path)
			return nil
		})
		if err != nil {
			return nil
		}
	}
	sortStrings(out)
	return out
}

// productLines returns the code lines of one repo-relative file with comments
// blanked, or nil when the file is gone. A nil return is what makes a citation
// read "vacated" rather than silently false.
func productLines(root, rel string) []string {
	src, ok := readFile(filepath.Join(root, filepath.FromSlash(rel)))
	if !ok {
		return nil
	}
	return strings.Split(blankComments(src), "\n")
}

// anchor finds the 1-based line in rel where every token appears, on a code
// line and as a whole identifier. It returns 0 when the citation has drifted.
func anchor(root, rel string, tokens ...string) int {
	lines := productLines(root, rel)
	if lines == nil {
		return -1 // file gone: drift, not absence
	}
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue // blank after comment blanking: prose cannot confirm a claim
		}
		found := true
		for _, t := range tokens {
			if !containsBounded(ln, t) {
				found = false
				break
			}
		}
		if found {
			return i + 1
		}
	}
	return 0
}

// blankComments turns every comment into spaces of the same length, so a symbol
// mentioned only in prose cannot confirm a claim. Quote-aware: a // inside a
// string literal does not end the line.
func blankComments(src string) string {
	var b strings.Builder
	b.Grow(len(src))
	runes := []rune(src)
	for i := 0; i < len(runes); {
		switch {
		case runes[i] == '"' || runes[i] == '\'':
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
				runes[i] = ' '
				i++
			}
		case runes[i] == '/' && i+1 < len(runes) && runes[i+1] == '*':
			for i < len(runes) {
				if runes[i] == '*' && i+1 < len(runes) && runes[i+1] == '/' {
					break
				}
				if runes[i] != '\n' {
					runes[i] = ' '
				}
				i++
			}
			if i < len(runes) {
				runes[i] = ' '
				i++
				if i < len(runes) {
					runes[i] = ' '
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
	return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}

func isIdent(token string) bool {
	for _, r := range token {
		if !isIdentRune(r) {
			return false
		}
	}
	return len(token) > 0
}

// containsBounded matches token as a whole word when it is identifier-shaped,
// and as a phrase otherwise, refusing an adjacent word character either way.
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

// readFile is the only file access in this package; it is read-only.
func readFile(path string) (string, bool) {
	b, err := os.ReadFile(path) //nolint:gosec // G304: caller passes a path from the scanned tree.
	if err != nil {
		return "", false
	}
	return string(b), true
}

// packageImports returns the import paths declared by the .go files in dir,
// including the test files, so the ban list covers the whole package.
func packageImports(dir string) []string {
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
	sortStrings(out)
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
