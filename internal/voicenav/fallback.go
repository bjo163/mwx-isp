package voicenav

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// FallbackRow is one destination and the typed path the console offers for it.
// The criterion asks for "teks fallback ada", so a fallback has to be a control
// that accepts typed text on the screen the voice command names - not the mouse.
type FallbackRow struct {
	Path        string
	Page        string
	TypedFields int
	FilterBound bool
	VoiceOnly   bool
}

var (
	reImportFrom = regexp.MustCompile(`from\s*'([^']+)'`)
	reImportSpec = regexp.MustCompile(`^\s*import\s+(.*)$`)
	reResource   = regexp.MustCompile(`name="([^"]+)"`)
	reListProp   = regexp.MustCompile(`list=\{([A-Za-z0-9_]+)\}`)
	reRoutePath  = regexp.MustCompile(`path="([^"]+)"`)
	reRouteElem  = regexp.MustCompile(`element=\{<([A-Za-z0-9_]+)`)
	reTypedField = regexp.MustCompile(`<(TextInput|SearchInput)\b`)
	reDashboard  = regexp.MustCompile(`dashboard=\{([A-Za-z0-9_]+)\}`)
)

// componentModules maps a React component name to the file App.tsx imports it
// from, so a destination can be traced to the code that renders it.
func componentModules(root string) map[string]string {
	out := map[string]string{}
	acc := ""
	for _, ln := range lines(root, appFile) {
		if acc == "" && !reImportSpec.MatchString(ln) {
			continue
		}
		if acc == "" {
			m := reImportSpec.FindStringSubmatch(ln)
			acc = m[1]
		} else {
			acc += " " + ln
		}
		from := reImportFrom.FindStringSubmatch(acc)
		if from == nil {
			continue
		}
		mod := from[1]
		for _, name := range importNames(acc) {
			out[name] = resolveModule(root, mod)
		}
		acc = ""
	}
	return out
}

var reIdent = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// importNames pulls the identifiers an import statement binds. For `X as Y` the
// bound name is Y, which is the name App.tsx then uses.
func importNames(stmt string) []string {
	i := strings.Index(stmt, " from ")
	if i < 0 {
		return nil
	}
	head := strings.TrimSpace(stmt[:i])
	head = strings.TrimPrefix(head, "import")
	head = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(head), "type"))
	var out []string
	for _, p := range strings.FieldsFunc(head, func(r rune) bool {
		return r == '{' || r == '}' || r == ',' || r == '"' || r == '\''
	}) {
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		last := fields[len(fields)-1]
		if !reIdent.MatchString(last) {
			continue
		}
		out = append(out, last)
	}
	return out
}

// resolveModule turns a TS import specifier into a repo-relative file, trying the
// extensions the bundler would.
func resolveModule(root, mod string) string {
	base := filepath.ToSlash(filepath.Join("web", "src", strings.TrimPrefix(strings.TrimPrefix(mod, "./"), "/")))
	for _, suffix := range []string{".tsx", ".ts", "/index.tsx", "/index.ts", ".js"} {
		rel := base + suffix
		if _, ok := readFile(filepath.Join(root, filepath.FromSlash(rel))); ok {
			return rel
		}
	}
	return ""
}

// resourceLists reads the `<Resource name="..." list={X}` declarations, including
// the multi-line ones, so a menu path can be traced to its list component.
func resourceLists(root string) map[string]string {
	out := map[string]string{}
	buf := ""
	depth := 0
	for _, ln := range lines(root, appFile) {
		if depth == 0 {
			if !strings.Contains(ln, "<Resource") {
				continue
			}
			buf, depth = "", 1
		}
		buf += " " + ln
		if !strings.Contains(ln, "/>") {
			continue
		}
		depth = 0
		n := reResource.FindStringSubmatch(buf)
		l := reListProp.FindStringSubmatch(buf)
		if n != nil && l != nil {
			out["/"+n[1]] = l[1]
		}
	}
	return out
}

// routeElements reads the `<Route path="..." element={<X` declarations.
func routeElements(root string) map[string]string {
	out := map[string]string{}
	for _, ln := range lines(root, appFile) {
		if !strings.Contains(ln, "<Route") {
			continue
		}
		p := reRoutePath.FindStringSubmatch(ln)
		e := reRouteElem.FindStringSubmatch(ln)
		if p != nil && e != nil {
			out[p[1]] = e[1]
		}
	}
	return out
}

// shellRoutes maps every path the app shell declares to the component that
// renders it: React Admin resources, custom routes, and the home dashboard the
// Admin element is given. A destination outside this set is a menu row that
// speaks a path the application does not have.
func shellRoutes(root string) map[string]string {
	out := map[string]string{}
	for p, c := range resourceLists(root) {
		out[p] = c
	}
	for p, c := range routeElements(root) {
		out[p] = c
	}
	for _, ln := range lines(root, appFile) {
		if m := reDashboard.FindStringSubmatch(ln); m != nil {
			out["/"] = m[1]
			break
		}
	}
	return out
}

// pageFile finds the file that renders one menu destination.
func pageFile(root string, path string, mods map[string]string, shell map[string]string) string {
	if comp, ok := shell[path]; ok {
		if f, ok := mods[comp]; ok {
			return f
		}
	}
	return ""
}

// fallbackRows measures the typed path for every destination.
func fallbackRows(root string, ds []Destination) []FallbackRow {
	mods := componentModules(root)
	shell := shellRoutes(root)
	var out []FallbackRow
	for _, d := range ds {
		page := pageFile(root, d.Path, mods, shell)
		row := FallbackRow{Path: d.Path, Page: page}
		if page != "" {
			row.TypedFields = countPattern(root, page, reTypedField)
			row.FilterBound = countMatches(root, page, "setFilters") > 0
		}
		row.VoiceOnly = row.TypedFields == 0
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// countPattern counts the code lines of one file that match a pattern.
func countPattern(root, rel string, re *regexp.Regexp) int {
	n := 0
	for _, ln := range lines(root, rel) {
		if re.MatchString(ln) {
			n++
		}
	}
	return n
}

// speechHits counts the references to a browser speech API in the frontend, and
// lists the files. The product is expected to read zero: that is what makes a
// voice layer necessarily opt-in instead of something already running.
func speechHits(root string) (int, []string) {
	var files []string
	n := 0
	for _, rel := range append(walkFiles(root, frontendDir, ".ts", ".tsx"),
		walkFiles(root, "web/e2e", ".ts")...) {
		if strings.HasSuffix(rel, ".d.ts") {
			continue
		}
		for _, ln := range lines(root, rel) {
			for _, tok := range speechTokens {
				if strings.Contains(ln, tok) {
					n++
					files = append(files, rel)
				}
			}
		}
	}
	sort.Strings(files)
	return n, dedupe(files)
}

var speechTokens = []string{"SpeechRecognition", "SpeechSynthesis", "getUserMedia", "MediaRecorder", "webkitSpeech"}

// listenerHits counts the server-side entry points that could hold a microphone
// open. The card forbids a permanent backend listener, so the claim is measured
// against the backend's own transports.
func listenerHits(root string) (int, []string) {
	var files []string
	n := 0
	for _, rel := range walkFiles(root, "internal", ".go") {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		for _, ln := range lines(root, rel) {
			for _, tok := range listenerTokens {
				if strings.Contains(ln, tok) {
					n++
					files = append(files, rel)
				}
			}
		}
	}
	return n, dedupe(files)
}

var listenerTokens = []string{"Microphone", "AudioStream", "/ws/voice", "Recognizer"}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
