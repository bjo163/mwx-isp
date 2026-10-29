package contextualws

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// This file repeats the three evidence rules the earlier studies use: a citation
// must land on a code line, an identifier must match as a whole, and drift is
// reported instead of dropped. It is a package-local copy on purpose - two
// research packages that import each other stop being independent evidence.

// ownPackage names this prototype, so a scan can never be closed by the study's
// own fixtures.
const ownPackage = "contextualws"

// The corpus. Product source proves behavior; the assertion corpus is only ever
// measured for what it promises to check.
const (
	menuFile             = "web/src/components/CustomMenu.tsx"
	appFile              = "web/src/App.tsx"
	layoutFile           = "web/src/components/CustomLayout.tsx"
	appBarFile           = "web/src/components/CustomAppBar.tsx"
	guideFile            = "web/src/pages/UserGuidePage.tsx"
	setupFile            = "web/src/components/onboarding/GettingStartedCard.tsx"
	tourFile             = "web/src/components/onboarding/QuickTourDialog.tsx"
	storageFile          = "web/src/components/onboarding/onboardingStorage.ts"
	dashboardFile        = "web/src/pages/Dashboard.tsx"
	configFile           = "web/src/pages/SystemConfigPage.tsx"
	customerTimelineFile = "web/src/components/CustomerTimeline.tsx"

	keyboardSpec = "web/e2e/keyboard-navigation.spec.ts"
	roleSpec     = "web/e2e/role-matrix.spec.ts"
	ciFile       = ".github/workflows/ci.yml"

	authFile     = "internal/adminapi/auth.go"
	operatorsGo  = "internal/adminapi/operators.go"
	domainDir    = "internal/domain"
	frontendDir  = "web/src"
	assertionDir = "web/e2e"
)

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

// blankFor picks the comment dialect by file name. Using the TypeScript rules on
// a YAML workflow would read the "# " of the first comment as code and blank the
// rest of the file, so the corpus is read in the syntax it is written in.
func blankFor(rel, src string) string {
	switch filepath.Ext(filepath.ToSlash(rel)) {
	case ".yml", ".yaml":
		return blankHashComments(src)
	default:
		return blankComments(src)
	}
}

// blankHashComments turns every "#" comment into spaces of the same length. A
// "#" that is not at a line start or after whitespace is part of a scalar, such
// as a color or a version anchor, so it stays code.
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

// countMatches counts the code lines in one file that carry the token.
func countMatches(root, rel, token string) int {
	n := 0
	for _, ln := range lines(root, rel) {
		if containsBounded(ln, token) {
			n++
		}
	}
	return n
}

// firstLineOf returns the 1-based line of the first occurrence of a phrase.
func firstLineOf(root, rel, phrase string) int {
	for i, ln := range lines(root, rel) {
		if strings.Contains(ln, phrase) {
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
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ownPackage || d.Name() == "node_modules" || d.Name() == "dist" {
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

// blankComments turns every comment into spaces of the same length, so a symbol
// mentioned only in prose cannot confirm a claim. It is quote-aware, including
// backticks, because a // inside a URL or a template string is not a comment.
// blankComments turns every comment into spaces of the same length, so a symbol
// mentioned only in prose cannot confirm a claim. It is quote-aware, including
// backticks, because a // inside a URL or a template string is not a comment.
// Newlines are written back exactly: a citation is a line number, so a blanker
// that swallowed a line break would move every claim below it.
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
		if strings.HasSuffix(rel, "_test.go") && strings.Contains(rel, "/"+ownPackage+"/") {
			continue
		}
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
