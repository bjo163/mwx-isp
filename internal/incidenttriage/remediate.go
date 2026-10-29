package incidenttriage

import (
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// actionRouteRe matches a mutating registration in the admin API. Both helper
// spellings the tree uses are covered.
var actionRouteRe = regexp.MustCompile(`webserver\.(?:Api)?(POST|PUT|DELETE)\(\s*"([^"]+)"`)

// identRe keeps a trailing segment that reads like an action word.
var identRe = regexp.MustCompile(`^[a-z][a-z_]*$`)

// measuredActions returns the verbs the product accepts on a mutating route:
// the trailing segment of a path that addresses one record. They are collected
// from the routes, not listed by hand, so "no automatic remediate" is checked
// against the console's real action vocabulary.
func measuredActions(root string) []string {
	base := filepath.Join(root, "internal", "adminapi")
	seen := map[string]bool{}
	_ = filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
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
		src, err := readFile(path)
		if err != nil {
			return nil
		}
		for _, m := range actionRouteRe.FindAllStringSubmatch(blankComments(src), -1) {
			if seg := actionSegment(m[2]); seg != "" {
				seen[seg] = true
			}
		}
		return nil
	})
	var out []string
	for v := range seen {
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

// actionSegment extracts the verb from "/sessions/:id/disconnect" and refuses a
// plural, because a plural tail names a collection rather than an action.
func actionSegment(path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 {
		return ""
	}
	addressed := false
	for _, p := range parts[:len(parts)-1] {
		if strings.HasPrefix(p, ":") {
			addressed = true
		}
	}
	if !addressed {
		return ""
	}
	last := parts[len(parts)-1]
	if !identRe.MatchString(last) || strings.HasSuffix(last, "s") {
		return ""
	}
	return last
}

// namedActions reports which measured verbs appear in a rendered verdict.
func namedActions(text string, verbs []string) []string {
	var out []string
	for _, v := range verbs {
		if containsBounded(text, v) {
			out = append(out, v)
		}
	}
	return out
}

// actionableFields lists the Verdict fields a remediation could be carried on.
// A verdict struct that grows an action, command, verb or route field shows up
// here, and the test that expects an empty list fails.
func actionableFields() []string {
	t := reflect.TypeOf(Verdict{})
	var out []string
	for i := 0; i < t.NumField(); i++ {
		name := strings.ToLower(t.Field(i).Name)
		for _, bad := range []string{"action", "command", "verb", "route", "method", "body", "exec", "remedy"} {
			if strings.Contains(name, bad) {
				out = append(out, t.Field(i).Name)
			}
		}
	}
	return out
}

func itoa(n int) string { return strconv.Itoa(n) }

// relToRoot turns an absolute scanned path back into a repository-relative one.
func relToRoot(path string) string {
	normalized := filepath.ToSlash(path)
	for _, marker := range []string{"/internal/", "/pkg/", "/cmd/", "/test/"} {
		if i := strings.LastIndex(normalized, marker); i >= 0 {
			return normalized[i+1:]
		}
	}
	return normalized
}
