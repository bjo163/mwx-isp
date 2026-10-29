package voicenav

import (
	"regexp"
	"sort"
	"strings"
)

// Destination is one console screen a spoken command could name, together with
// the exact words the console itself uses for it in each shipped locale. Nothing
// in this study invents a phrase: a phrase exists only because a locale file
// renders it on screen.
type Destination struct {
	Path       string
	LabelKey   string
	SectionKey string
	Gate       string
	English    string
	Indonesian string
	EnLine     int
	IdLine     int
}

var (
	reTo       = regexp.MustCompile(`to:\s*'([^']+)'`)
	reLabelKey = regexp.MustCompile(`labelKey:\s*'([^']+)'`)
	reSection  = regexp.MustCompile(`sectionKey:\s*'([^']+)'`)
	rePerms    = regexp.MustCompile(`permissions:\s*\[([^\]]*)\]`)
	reLocale   = regexp.MustCompile(`^([A-Za-z_0-9]+):\s*'([^']*)'`)
)

// destinations reads the shipped menu array and joins each row to the two label
// sets the console renders. A row whose key is in neither locale is reported, not
// dropped: it is a screen that cannot be said out loud in any shipped language.
func destinations(root string) []Destination {
	ls := lines(root, menuFile)
	en := localeMenuBlock(root, enLocale)
	id := localeMenuBlock(root, idLocale)
	var out []Destination
	for _, ln := range ls {
		m := reTo.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		d := Destination{Path: m[1]}
		if k := reLabelKey.FindStringSubmatch(ln); k != nil {
			d.LabelKey = strings.TrimPrefix(k[1], "menu.")
		}
		if s := reSection.FindStringSubmatch(ln); s != nil {
			d.SectionKey = strings.TrimPrefix(s[1], "menu.")
		}
		if p := rePerms.FindStringSubmatch(ln); p != nil {
			d.Gate = strings.NewReplacer("'", "", " ", "").Replace(p[1])
		}
		d.English = en[d.LabelKey]
		d.Indonesian = id[d.LabelKey]
		d.EnLine = localeLeafLine(root, enLocale, d.LabelKey)
		d.IdLine = localeLeafLine(root, idLocale, d.LabelKey)
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// localeMenuBlock reads the flat `menu: { key: 'label', ... }` block of one
// locale file.
func localeMenuBlock(root, rel string) map[string]string {
	out := map[string]string{}
	inBlock := false
	for _, ln := range lines(root, rel) {
		t := strings.TrimSpace(ln)
		if !inBlock {
			inBlock = t == "menu: {"
			continue
		}
		if strings.HasPrefix(t, "}") {
			break
		}
		if m := reLocale.FindStringSubmatch(t); m != nil {
			out[m[1]] = m[2]
		}
	}
	return out
}

// localeLeafLine finds the first line of one locale file that renders the given
// key as a label. The search is deliberately whole-file: an action verb lives
// under a resource section, not under `menu`.
func localeLeafLine(root, rel, key string) int {
	if key == "" {
		return 0
	}
	for i, ln := range lines(root, rel) {
		if m := reLocale.FindStringSubmatch(strings.TrimSpace(ln)); m != nil && m[1] == key {
			return i + 1
		}
	}
	return 0
}

// localeLeafValue returns the label a locale file renders for a key.
func localeLeafValue(root, rel, key string) string {
	for _, ln := range lines(root, rel) {
		if m := reLocale.FindStringSubmatch(strings.TrimSpace(ln)); m != nil && m[1] == key {
			return m[2]
		}
	}
	return ""
}

// Route is one admin API registration, read from the method helper the backend
// actually calls.
type Route struct {
	Method string
	Path   string
	Files  string
	Line   int
}

var reReg = regexp.MustCompile(`Api(GET|POST|PUT|DELETE)\(\s*"([^"]+)"`)

// registrations walks the admin API and returns every route it declares.
func registrations(root string) []Route {
	var out []Route
	for _, rel := range walkFiles(root, adminapiDir, ".go") {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		for i, ln := range lines(root, rel) {
			m := reReg.FindStringSubmatch(ln)
			if m == nil {
				continue
			}
			out = append(out, Route{Method: m[1], Path: m[2], Files: rel, Line: i + 1})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Method < out[j].Method
	})
	return out
}

func isWrite(r Route) bool { return r.Method != "GET" }

// ActionVerb is one word the server accepts as an operation, and the place it is
// accepted. The vocabulary is derived, never typed by hand: the guard that
// rejects everything else is the source of the list.
type ActionVerb struct {
	Verb     string
	Source   string
	File     string
	Line     int
	English  string
	Speaking string
}

var reQuoted = regexp.MustCompile(`"([A-Za-z][A-Za-z_-]*)"`)

// actionVerbs collects the mutation vocabulary from three measured places: the
// server's own parameterised-action guard, the trailing literal segment of every
// write route that addresses a record, and the label each locale renders for
// those words - which is what an operator would actually say.
func actionVerbs(root string) []ActionVerb {
	seen := map[string]ActionVerb{}
	add := func(v, source, file string, line int) {
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = ActionVerb{Verb: v, Source: source, File: file, Line: line,
			English: localeLeafValue(root, enLocale, v), Speaking: localeLeafValue(root, idLocale, v)}
	}

	for _, rel := range walkFiles(root, adminapiDir, ".go") {
		if strings.HasSuffix(rel, "_test.go") {
			continue
		}
		for i, ln := range lines(root, rel) {
			if strings.Contains(ln, "action !=") {
				for _, q := range reQuoted.FindAllStringSubmatch(ln, -1) {
					add(strings.ToLower(q[1]), "the server's action guard", rel, i+1)
				}
			}
		}
	}

	for _, r := range registrations(root) {
		if !isWrite(r) || !strings.Contains(r.Path, ":") {
			continue
		}
		segs := strings.Split(strings.Trim(r.Path, "/"), "/")
		last := segs[len(segs)-1]
		if strings.HasPrefix(last, ":") {
			continue
		}
		add(strings.ToLower(last), "an action route", r.Files, r.Line)
	}

	var out []ActionVerb
	for _, v := range seen {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Verb < out[j].Verb })
	return out
}

// banTokens is the spoken refusal set: each action word with every word its two
// locale labels render. This is how "putuskan paksa" refuses without the
// transcript ever containing the English identifier.
func banTokens(vs []ActionVerb) map[string]string {
	out := map[string]string{}
	for _, v := range vs {
		// A slice, not a map literal: a token can arrive from the route word and from
		// a locale label at once, and which source is recorded must not depend on
		// iteration order.
		sources := []struct{ word, from string }{
			{v.Verb, "route or guard word"},
			{v.English, "English label"},
			{v.Speaking, "Indonesian label"},
		}
		for _, s := range sources {
			for _, t := range tokens(normalize(s.word)) {
				if _, ok := out[t]; !ok {
					out[t] = v.Verb + " (" + s.from + ")"
				}
			}
		}
	}
	return out
}

// PrunedBan is the refusal set after the two rules that keep it from eating the
// navigation vocabulary, plus what each rule removed. Both removals are counted:
// a derived list is only trustworthy if the study can show what it threw away.
type PrunedBan struct {
	Kept        map[string]string
	ShortWords  []string
	ScreenWords []string
}

// pruneBan drops (1) any derived token shorter than three runes, because the
// locale labels of action words carry the prepositions they were written with -
// "ke" and "to" would refuse ordinary commands; and (2) any token that is also a
// word of a shipped screen label, because the write route that ends in
// `/operators` would otherwise refuse the Operators screen. Refusal can only
// ever remove an answer, so pruning a token cannot introduce an operation: the
// safety property does not depend on this list being right.
func pruneBan(ban map[string]string, phrases []Phrase) PrunedBan {
	nav := map[string]bool{}
	for _, p := range phrases {
		for _, t := range tokens(normalize(p.Label)) {
			nav[t] = true
		}
	}
	out := PrunedBan{Kept: map[string]string{}}
	for token, who := range ban {
		switch {
		case len([]rune(token)) < 3:
			out.ShortWords = append(out.ShortWords, token)
		case nav[token]:
			out.ScreenWords = append(out.ScreenWords, token)
		default:
			out.Kept[token] = who
		}
	}
	sort.Strings(out.ShortWords)
	sort.Strings(out.ScreenWords)
	return out
}
