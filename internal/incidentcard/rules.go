package incidentcard

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// The redaction rule set is read out of internal/redact/redact.go rather than
// copied into this package. A card may only print a slot the shipped rules already
// call harmless, so the rules are evidence that has to move with the product: when
// redact.go gains or loses a fragment, the card's allowed field set changes with
// it, and a claim that no longer matches says so.
const (
	redactFile      = "internal/redact/redact.go"
	classifyFunc    = "func ClassifyKey(key string) Class"
	normalizerCall  = "strings.NewReplacer"
	fragmentMatcher = "containsAnyFragment"
)

// Predicate kinds, named after the source call that produced them.
const (
	predList     = "list"
	predExact    = "=="
	predSuffix   = "HasSuffix"
	predPrefix   = "HasPrefix"
	predContains = "Contains"
)

type predicate struct {
	kind  string
	value string
}

// ruleGroup is one "if ... { return Class }" branch of the shipped classifier,
// with the source line of its return so a table can name where the rule lives.
type ruleGroup struct {
	class      string
	predicates []predicate
	line       int
}

// Rules is the classification program ClassifyKey runs, parsed from the product.
type Rules struct {
	// Groups are in source order, which is the precedence the function applies.
	Groups []ruleGroup
	// Lists holds the named fragment lists the groups reference.
	Lists map[string][]string
	// Strip is the character set the normalizer removes before matching.
	Strip []string
}

var (
	reListCall = regexp.MustCompile(fragmentMatcher + `\(\w+,\s*(\w+)\)`)
	reExact    = regexp.MustCompile(`normalized == "([^"]+)"`)
	reSuffix   = regexp.MustCompile(`strings\.HasSuffix\(\w+, "([^"]+)"\)`)
	rePrefix   = regexp.MustCompile(`strings\.HasPrefix\(\w+, "([^"]+)"\)`)
	reContains = regexp.MustCompile(`strings\.Contains\(\w+, "([^"]+)"\)`)
	reReturn   = regexp.MustCompile(`^return ([A-Z]\w*)$`)
	reListDecl = regexp.MustCompile(`^(\w+)\s*=\s*\[\]string\{`)
	reQuoted   = regexp.MustCompile(`"([^"]*)"`)
)

// loadRules parses the shipped classifier out of redact.go. Every piece of the
// rule set - the precedence, the fragment lists, the separator strip - comes from
// the file, so the study cannot quietly invent a friendlier policy.
func loadRules(root string) (Rules, error) {
	all := lines(root, redactFile)
	if len(all) == 0 {
		return Rules{}, fmt.Errorf("%s not readable", redactFile)
	}
	start := -1
	for i, ln := range all {
		if strings.HasPrefix(strings.TrimSpace(ln), classifyFunc) {
			start = i
			break
		}
	}
	if start < 0 {
		return Rules{}, fmt.Errorf("%s not found in %s", classifyFunc, redactFile)
	}
	depth := 0
	end := -1
	for i := start; i < len(all); i++ {
		depth += strings.Count(all[i], "{") - strings.Count(all[i], "}")
		if depth == 0 && i > start {
			end = i
			break
		}
	}
	if end < 0 {
		return Rules{}, fmt.Errorf("%s never closes in %s", classifyFunc, redactFile)
	}
	body := all[start : end+1]
	text := strings.Join(body, "\n")

	r := Rules{Lists: map[string][]string{}}
	if !strings.Contains(text, normalizerCall) {
		return Rules{}, fmt.Errorf("normalizer %s not found in %s", normalizerCall, redactFile)
	}
	for _, ln := range body {
		if !strings.Contains(ln, normalizerCall) {
			continue
		}
		for _, m := range reQuoted.FindAllStringSubmatch(ln, -1) {
			if m[1] != "" {
				r.Strip = append(r.Strip, m[1])
			}
		}
	}

	// The fragment lists are declared above the function; keep only the ones this
	// function actually references.
	for i, ln := range all {
		m := reListDecl.FindStringSubmatch(strings.TrimSpace(ln))
		if m == nil || !strings.Contains(text, m[1]) {
			continue
		}
		items := reQuoted.FindAllStringSubmatch(blockText(all, i), -1)
		var frags []string
		for _, q := range items {
			if q[1] != "" {
				frags = append(frags, q[1])
			}
		}
		r.Lists[m[1]] = frags
	}

	var pending []predicate
	for i, ln := range body {
		code := strings.TrimSpace(ln)
		if !strings.Contains(code, fragmentMatcher) {
			for _, spec := range []struct {
				re   *regexp.Regexp
				kind string
			}{
				{reExact, predExact}, {reSuffix, predSuffix},
				{rePrefix, predPrefix}, {reContains, predContains},
			} {
				for _, m := range spec.re.FindAllStringSubmatch(code, -1) {
					pending = append(pending, predicate{kind: spec.kind, value: m[1]})
				}
			}
		}
		for _, m := range reListCall.FindAllStringSubmatch(code, -1) {
			pending = append(pending, predicate{kind: predList, value: m[1]})
		}
		if rm := reReturn.FindStringSubmatch(code); rm != nil {
			r.Groups = append(r.Groups, ruleGroup{class: rm[1], predicates: pending, line: start + i + 1})
			pending = nil
		}
	}
	if len(r.Groups) == 0 {
		return Rules{}, fmt.Errorf("%s yielded no rule branches", redactFile)
	}
	for _, g := range r.Groups {
		for _, p := range g.predicates {
			if p.kind == predList && len(r.Lists[p.value]) == 0 {
				return Rules{}, fmt.Errorf("classifier uses %s but that list parsed empty in %s", p.value, redactFile)
			}
		}
	}
	if len(r.Strip) == 0 {
		return Rules{}, fmt.Errorf("%s normalizer parsed no separator", redactFile)
	}
	return r, nil
}

// blockText reads the brace-delimited block that starts on line i.
func blockText(all []string, i int) string {
	depth := 0
	var out []string
	for j := i; j < len(all); j++ {
		depth += strings.Count(all[j], "{") - strings.Count(all[j], "}")
		out = append(out, all[j])
		if depth == 0 && j >= i {
			break
		}
	}
	return strings.Join(out, "\n")
}

// fallback is the class the shipped function returns when no branch matched.
func (r Rules) fallback() string {
	if len(r.Groups) == 0 {
		return ""
	}
	last := r.Groups[len(r.Groups)-1]
	if len(last.predicates) == 0 {
		return last.class
	}
	return ""
}

// normalize applies the same lowercase-and-strip the shipped classifier does.
func (r Rules) normalize(key string) string {
	return strings.NewReplacer(pairs(r.Strip)...).Replace(strings.ToLower(key))
}

// pairs maps every stripped separator to the empty string, which is what
// strings.NewReplacer expects: an (old, new) pair per separator.
func pairs(items []string) []string {
	var out []string
	for _, s := range items {
		out = append(out, s, "")
	}
	return out
}

// Classify runs the parsed program over a JSON key and returns the class name the
// shipped rules give it. A key the rules never name falls through to the
// classifier's own default branch, exactly as the product does.
func (r Rules) Classify(key string) string {
	if key == "" || key == "-" {
		return ""
	}
	norm := r.normalize(key)
	for _, g := range r.Groups {
		for _, p := range g.predicates {
			if p.matches(norm, r) {
				return g.class
			}
		}
	}
	return r.fallback()
}

func (p predicate) matches(norm string, r Rules) bool {
	switch p.kind {
	case predExact:
		return norm == p.value
	case predSuffix:
		return strings.HasSuffix(norm, p.value)
	case predPrefix:
		return strings.HasPrefix(norm, p.value)
	case predContains:
		return strings.Contains(norm, p.value)
	case predList:
		for _, frag := range r.Lists[p.value] {
			if strings.Contains(norm, frag) {
				return true
			}
		}
	}
	return false
}

// listNames returns the parsed fragment lists, sorted by name so a printed table
// never depends on map order.
func (r Rules) listNames() []string {
	var out []string
	for name := range r.Lists {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
