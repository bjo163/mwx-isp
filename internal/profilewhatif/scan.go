package profilewhatif

import (
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
)

// The product directories the scans walk. One is what an Access-Accept is built
// from, another what a running session is billed from, and the rest carry the
// inheritance contract and its cache.
const (
	enhancerDir  = "internal/radiusd/plugins/auth/enhancers"
	checkerDir   = "internal/radiusd/plugins/auth/checkers"
	accounting   = "internal/radiusd/plugins/accounting"
	adminapiDir  = "internal/adminapi"
	radiusdRoot  = "internal/radiusd"
	gettersFile  = "internal/domain/radius_user_getters.go"
	cacheFile    = "internal/app/profile_cache.go"
	profileAPI   = "internal/adminapi/profiles.go"
	coaFile      = "internal/radiusd/coa_service.go"
	runnerFile   = "internal/radiusd/auth_plugin_runner.go"
	domainRadius = "internal/domain/radius.go"
)

// banRule is one forbidden import. Prefix rules catch a subtree; exact rules catch
// one package, which is how the CoA-declaring package is banned without banning the
// enhancer packages that live beside it.
type banRule struct {
	path  string
	exact bool
}

func (r banRule) hits(imp string) bool {
	if r.exact {
		return imp == r.path
	}
	return imp == r.path || strings.HasPrefix(imp, r.path)
}

// bannedImports is the list this package must never appear in. The test that counts
// them is the proof that the prototype cannot reach a database, a NAS, or an HTTP
// handler.
func banRules() []banRule {
	return []banRule{
		{path: "gorm.io/"},
		{path: "database/sql"},
		{path: "net/http"},
		{path: "github.com/labstack/echo"},
		{path: "github.com/bjo163/mwx-isp/internal/app"},
		{path: "github.com/bjo163/mwx-isp/internal/adminapi"},
		{path: "github.com/bjo163/mwx-isp/internal/service"},
		{path: "github.com/bjo163/mwx-isp/internal/radiusd/accounting"},
		{path: "github.com/bjo163/mwx-isp/internal/radiusd", exact: true},
	}
}

// bannedPaths flattens the rules for the report.
func bannedPaths(rules []banRule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if r.exact {
			out = append(out, r.path+" (exact)")
			continue
		}
		out = append(out, r.path)
	}
	return out
}

// isImportSend checks the one thing the ban list cannot: the wire library itself is
// imported to build packets, so the study must never call its client.
func isImportSend(root string) int {
	return countSubstring(corpus(goFiles(root, "internal/"+ownPackage, true, false)), "radius.NewClient(") +
		countSubstring(corpus(goFiles(root, "internal/"+ownPackage, true, false)), ".Authenticate(")
}

// bannedHit matches one import against the flattened display list, respecting the
// "(exact)" marker the flattener adds.
func bannedHit(imp string, display []string) bool {
	for _, d := range display {
		if exact := strings.HasSuffix(d, " (exact)"); exact {
			if imp == strings.TrimSuffix(d, " (exact)") {
				return true
			}
			continue
		}
		if imp == d || strings.HasPrefix(imp, d) {
			return true
		}
	}
	return false
}

// countBanned reports how many of the package's imports are on the ban list.
func countBanned(imports []string, banned []banRule) int {
	n := 0
	for _, imp := range imports {
		for _, b := range banned {
			if b.hits(imp) {
				n++
				break
			}
		}
	}
	return n
}

// ownPackageDir is this prototype's directory under the given root.
func ownPackageDir(root string) string {
	return filepath.Join(root, "internal", ownPackage)
}

// goFiles lists the .go files under one repo-relative directory. When tests is
// false, _test.go files are left out so a claim is proven by product code. When
// recursive is false, only that directory itself is listed.
func goFiles(root, dir string, tests, recursive bool) []string {
	base := filepath.Join(root, filepath.FromSlash(dir))
	var out []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != base && !recursive {
				return filepath.SkipDir
			}
			if filepath.Base(path) == ownPackage {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !tests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		out = append(out, path)
		return nil
	})
	if err != nil {
		return nil
	}
	sortStrings(out)
	return out
}

// codeLinesOf returns one file's code lines, comments blanked.
func codeLinesOf(path string) []string {
	src, ok := readFile(path)
	if !ok {
		return nil
	}
	return blankLines(src)
}

// corpus reads a set of files into code lines, one entry per file.
func corpus(paths []string) [][]string {
	out := make([][]string, 0, len(paths))
	for _, p := range paths {
		out = append(out, codeLinesOf(p))
	}
	return out
}

// countToken counts code lines across a corpus that carry every token as a whole
// identifier.
func countToken(files [][]string, tokens ...string) int {
	n := 0
	for _, lines := range files {
		for _, ln := range lines {
			if strings.TrimSpace(ln) == "" {
				continue
			}
			hit := true
			for _, t := range tokens {
				if !containsBounded(ln, t) {
					hit = false
					break
				}
			}
			if hit {
				n++
			}
		}
	}
	return n
}

// countSubstring counts code lines carrying a substring, used for accessor names
// that only ever appear inside a longer identifier.
func countSubstring(files [][]string, sub string) int {
	n := 0
	for _, lines := range files {
		for _, ln := range lines {
			if strings.Contains(ln, sub) {
				n++
			}
		}
	}
	return n
}

// getterCount is how many inheritance getters the domain exposes.
func getterCount(root string) int {
	return countSubstring(corpus([]string{filepath.Join(root, filepath.FromSlash(gettersFile))}),
		"func (u *RadiusUser) Get")
}

// enhancerFiles is the product code that builds an Access-Accept.
func enhancerFiles(root string) []string { return goFiles(root, enhancerDir, false, true) }

// checkerFiles is the product code that decides whether a session is accepted at
// all.
func checkerFiles(root string) []string { return goFiles(root, checkerDir, false, true) }

// readsColumnInEnhancers counts code lines in the enhancer tree that touch one
// user or profile column by selector, for example ".ActiveNum".
func readsColumnInEnhancers(root, selector string) int {
	return countSubstring(corpus(enhancerFiles(root)), selector)
}

// readsColumnInCheckers counts the same for the policy-checker tree.
func readsColumnInCheckers(root, selector string) int {
	return countSubstring(corpus(checkerFiles(root)), selector)
}

// accountingCacheReads counts where the accounting pipeline asks for a profile.
func accountingCacheReads(root string) int {
	return countSubstring(corpus(goFiles(root, accounting, false, true)), "profile_cache")
}

// cacheMutators lists the exported mutators the product's own cache exposes
// besides the read path.
func cacheMutators(root string) []string {
	var out []string
	for _, ln := range productLines(root, cacheFile) {
		if !strings.HasPrefix(ln, "func (pc *ProfileCache)") {
			continue
		}
		name := between(ln, "func (pc *ProfileCache) ", "(")
		if name == "" || !unicode.IsUpper(rune(name[0])) || strings.HasPrefix(name, "Get") {
			continue
		}
		out = append(out, name)
	}
	sortStrings(out)
	return out
}

// invalidateSites counts the production call sites that drop a profile from the
// cache after a write.
func invalidateSites(root string) int {
	return countSubstring(corpus(goFiles(root, adminapiDir, false, false)), "ProfileCache().Invalidate")
}

// rateSymbols are the accessor fragments that mean "this test asserts a rate".
var rateSymbols = []string{"RateLimit", "AverageRate", "PeakRate", "SpeedLimit", "RateCtrl"}

// checkerOnlyFields are the inherited columns no enhancer reads, measured rather
// than asserted: the getter name appears in the checker tree and never in the
// enhancer tree.
func checkerOnlyFields(root string) []string {
	var out []string
	for _, f := range Fields() {
		enh := readsColumnInEnhancers(root, "."+f.Reaches)
		chk := readsColumnInCheckers(root, "."+f.Reaches)
		if enh == 0 && chk > 0 {
			out = append(out, f.Name)
		}
	}
	return out
}

// columnsReadByEnhancers lists the user columns a direct field read (not a getter)
// reaches in the enhancer tree, with the file that reads them.
func columnsReadByEnhancers(root string) map[string]string {
	out := map[string]string{}
	for _, path := range enhancerFiles(root) {
		for _, ln := range codeLinesOf(path) {
			for _, col := range userColumns {
				if strings.Contains(ln, "user."+col) {
					if _, seen := out[col]; !seen {
						out[col] = filepath.Base(path)
					}
				}
			}
		}
	}
	return out
}

// inheritedColumnsReadDirect lists the columns this study can change that an
// enhancer still reads as a struct field, bypassing the inheritance getter.
func inheritedColumnsReadDirect(root string) []string {
	direct := columnsReadByEnhancers(root)
	inherited := map[string]bool{}
	for _, f := range Fields() {
		inherited[f.Name] = true
	}
	var out []string
	for col := range direct {
		if inherited[col] {
			out = append(out, col)
		}
	}
	sortStrings(out)
	return out
}

// userColumns is the list of subscriber columns the study asks about.
var userColumns = []string{
	"UpRate", "DownRate", "ActiveNum", "AddrPool", "Domain", "RadiusClass",
	"IPv6PrefixPool", "DelegatedIpv6PrefixPool", "BindMac", "BindVlan",
	"Vlanid1", "Vlanid2", "IpAddr", "IpV6Addr", "MacAddr", "DelegatedIpv6Prefix",
	"ProfileId", "ProfileLinkMode",
}

// rateTestFilesWithoutCache finds the enhancer tests that assert a rate attribute
// while never handing the getter a profile cache.
func rateTestFilesWithoutCache(root string) []string {
	var out []string
	for _, path := range goFiles(root, enhancerDir, true, true) {
		if !strings.HasSuffix(path, "_test.go") {
			continue
		}
		lines := corpus([]string{path})
		rate := false
		for _, sub := range rateSymbols {
			if countSubstring(lines, sub) > 0 {
				rate = true
				break
			}
		}
		if !rate {
			continue
		}
		if countToken(lines, "profile_cache") > 0 {
			continue
		}
		out = append(out, filepath.Base(path))
	}
	sortStrings(out)
	return out
}

// coaDeclaringPackage names the package that owns the CoA and Disconnect clients,
// found by reading the product source rather than by being told.
func coaDeclaringPackage(root string) string {
	for _, path := range goFiles(root, radiusdRoot, false, false) {
		pkg := ""
		for i, ln := range strings.Split(mustRead(path), "\n") {
			if pkg == "" && strings.HasPrefix(strings.TrimSpace(ln), "package ") {
				pkg = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(ln), "package "))
			}
			code := blankComments(ln)
			if strings.HasPrefix(code, "func ") && (strings.Contains(code, ") CoA(") ||
				strings.Contains(code, ") Disconnect(")) {
				return pkg + " (" + relToRoot(root, path) + ":" + strconv.Itoa(i+1) + ")"
			}
		}
	}
	return ""
}

func mustRead(path string) string {
	src, _ := readFile(path)
	return src
}

func relToRoot(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

// profileAPIHandlers lists the HTTP handlers the profile endpoints expose.
func profileAPIHandlers(root string) []string {
	var out []string
	for _, ln := range productLines(root, profileAPI) {
		if !strings.HasPrefix(ln, "func ") {
			continue
		}
		name := between(ln, "func ", "(")
		if name != "" && !strings.HasPrefix(name, "to") && !strings.HasPrefix(name, "register") {
			out = append(out, name)
		}
	}
	sortStrings(out)
	return out
}
