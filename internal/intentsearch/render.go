package intentsearch

import (
	"sort"
	"strconv"
	"strings"
)

// RenderMarkdown turns the study into the tables the memo splices in, so every
// number in the prose comes from the same run that produced the verdict.
func RenderMarkdown(r Report) string {
	var b strings.Builder
	b.WriteString("## Verdict\n\n")
	b.WriteString("| Measure | Value |\n| --- | ---: |\n")
	row(&b, "Fixtures in the corpus", strconv.Itoa(r.Totals.Fixtures))
	row(&b, "Fixtures matching their expectation", strconv.Itoa(r.Totals.Matched))
	row(&b, "Resolved as navigation", strconv.Itoa(r.Totals.Navigations))
	row(&b, "Resolved with a composed filter", strconv.Itoa(r.Totals.Filters))
	row(&b, "Refused", strconv.Itoa(r.Totals.Refusals))
	row(&b, "Targets named that could reach a write", strconv.Itoa(targetWrites(r)))
	row(&b, "Query keys filled from the page's own vocabulary", strconv.Itoa(r.Totals.StatusFills))
	row(&b, "Cataloged intents reached by the corpus",
		strconv.Itoa(r.Totals.IntentsReached)+" of "+strconv.Itoa(r.Totals.IntentsTotal))
	row(&b, "Read routes in the admin API", strconv.Itoa(r.Totals.ReadRoutes))
	row(&b, "Mutating routes in the admin API", strconv.Itoa(r.Totals.MutationRoutes))
	row(&b, "Distinct action words the deny step recognizes", strconv.Itoa(r.Totals.ActionVerbs))
	row(&b, "Of those, words measured from a handler value guard", strconv.Itoa(r.Totals.GuardVerbs))
	row(&b, "Entries in the console's menu-name-to-path table", strconv.Itoa(r.Totals.PathMapEntries))
	row(&b, "Menu entries not in that table, which call their own name", strconv.Itoa(r.Totals.UnmappedNames))
	row(&b, "Hand-written Indonesian hazard aliases", strconv.Itoa(r.Totals.HazardAliases))
	row(&b, "Claims confirmed against product code",
		strconv.Itoa(r.Totals.ClaimsConfirmed)+" of "+strconv.Itoa(r.Totals.Claims))
	row(&b, "Gaps still open", strconv.Itoa(r.Totals.GapsOpen)+" of "+strconv.Itoa(r.Totals.Gaps))
	row(&b, "Menu entries in the console", strconv.Itoa(len(r.Surface.Resources)))
	row(&b, "Filter inputs declared by list pages", strconv.Itoa(len(r.Surface.Filters)))
	row(&b, "Product files scanned", strconv.Itoa(r.Surface.ScannedFiles))

	b.WriteString("\n## What the console exposes today\n\n")
	b.WriteString("| HTTP method | Routes |\n| --- | ---: |\n")
	counts := r.Surface.CountByMethod()
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		row(&b, k, strconv.Itoa(counts[k]))
	}
	b.WriteString("\n| Menu entry | list | show | edit | create |\n| --- | --- | --- | --- | --- |\n")
	for _, res := range r.Surface.Resources {
		b.WriteString("| ")
		b.WriteString(cell(res.Name))
		b.WriteString(" | ")
		for _, ok := range []bool{res.HasList, res.HasShow, res.HasEdit, res.HasCreate} {
			b.WriteString(boolMark(ok))
			b.WriteString(" | ")
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## Filter inputs the parser may compose\n\n")
	b.WriteString("| Group | Query key | Widget | Offered values | Declared at |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, f := range r.Surface.Filters {
		b.WriteString("| ")
		b.WriteString(cell(f.Group) + " | " + cell(f.Param) + " | " + cell(f.Component) + " | ")
		b.WriteString(cell(list(f.Values)) + " | ")
		b.WriteString(cell(f.File+":"+strconv.Itoa(f.Line)) + " |\n")
	}

	b.WriteString("\n## The name the menu shows and the path the API serves\n\n")
	b.WriteString("One table in the data provider rewrites some menu names before the request " +
		"leaves; a resolver has to read it rather than assume the two are the same string.\n\n")
	b.WriteString("| Menu entry | Resource name sent to the API | How |\n| --- | --- | --- |\n")
	for _, res := range r.Surface.Resources {
		path, ok := r.Surface.PathMap[res.Name]
		how := "not in the table"
		if ok && path != res.Name {
			how = "renamed"
		} else if ok {
			how = "declared as itself"
		}
		if !ok {
			path = res.Name
		}
		b.WriteString("| " + cell(res.Name) + " | " + cell(path) + " | " + cell(how) + " |\n")
	}
	var extra []string
	for name, path := range r.Surface.PathMap {
		if !r.Surface.HasResource(name) {
			extra = append(extra, name+" -> "+path)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		b.WriteString("\nTable entries that are not menu entries: " + cell(list(extra)) + ".\n")
	}

	b.WriteString("\n## The deny vocabulary\n\n")
	b.WriteString("Action words are measured from two places: the trailing segment of a " +
		"non-GET registration (`/…/:id/<verb>`) and the value a mutation handler compares " +
		"its action against. The alias list is the only hand-written half.\n\n")
	p, err := NewParser(r.Surface)
	if err == nil {
		b.WriteString("| Measured action word | Routes it closes off |\n| --- | --- |\n")
		for _, v := range p.MutationVocabulary() {
			var disp []string
			for _, x := range r.Surface.RoutesForVerb(v) {
				disp = append(disp, x.Display())
			}
			if len(disp) == 0 {
				sites := r.Surface.ActionSitesFor(v)
				if len(sites) == 0 {
					continue
				}
				disp = []string{"value guard at " + strings.Join(sites, ", ")}
			}
			b.WriteString("| " + cell(v) + " | " + cell(strings.Join(disp, ", ")) + " |\n")
		}
		b.WriteString("\nHand-written aliases: ")
		b.WriteString(cell(list(p.handWrittenAliases())) + ".\n")
	}

	b.WriteString("\n## The 20 prompt fixtures\n\n")
	b.WriteString("| # | Prompt | Expected | Parsed | Match | Reason and refusal |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- |\n")
	for i, res := range r.Results {
		b.WriteString("| " + strconv.Itoa(i+1) + " | ")
		b.WriteString(cell(res.Fixture.Prompt) + " | ")
		b.WriteString(cell(string(res.Fixture.WantKind)) + " | ")
		b.WriteString(cell(actual(res.Resolution)) + " | ")
		b.WriteString(cell(boolMark(res.Matches)) + " | ")
		b.WriteString(cell(res.Fixture.Why) + " |\n")
	}

	b.WriteString("\n### What each refusal said\n\n")
	for _, res := range r.Results {
		if res.Resolution.Kind != KindRefuse {
			continue
		}
		b.WriteString("- `" + res.Fixture.Prompt + "`\n")
		for _, why := range res.Resolution.Refusals {
			b.WriteString("  - " + why + "\n")
		}
	}

	b.WriteString("\n## Statements the proposal rests on\n\n")
	b.WriteString("| ID | Statement | Evidence | Status |\n| --- | --- | --- | --- |\n")
	for _, c := range r.Claims {
		status := "unconfirmed"
		switch {
		case c.Broken:
			status = "broken"
		case c.Confirmed:
			status = "confirmed"
		}
		b.WriteString("| " + cell(c.ID) + " | " + cell(c.Statement) + " | " +
			cell(list(c.Hits)) + " | " + cell(status) + " |\n")
	}

	b.WriteString("\n## Gaps this study does not close\n\n")
	b.WriteString("| ID | Gap | Scope | Status | Evidence |\n| --- | --- | --- | --- | --- |\n")
	for _, g := range r.Gaps {
		status := "open"
		if g.Vacated {
			status = "vacated"
		}
		b.WriteString("| " + cell(g.ID) + " | " + cell(g.Statement) + " | " +
			cell(g.Scope) + " | " + cell(status) + " | " + cell(list(g.Hits)) + " |\n")
	}

	if len(r.StaleFiles) > 0 {
		b.WriteString("\n## Read but not real source\n\n")
		b.WriteString("The scan walks `web/src`, so these copies are read as text and reported:\n\n")
		for _, f := range r.StaleFiles {
			b.WriteString("- `" + f + "`\n")
		}
	}
	return b.String()
}

// actual renders what the parser produced, in one cell.
func actual(r Resolution) string {
	if r.Kind == KindRefuse {
		return "refuse: " + strings.Join(r.Refusals, "; ")
	}
	s := string(r.Kind) + " " + r.Route
	if parts := r.ParamsOf(); len(parts) > 0 {
		var filled []string
		for _, p := range parts {
			if v, ok := r.FilterValue(p); ok {
				filled = append(filled, p+"="+v)
				continue
			}
			filled = append(filled, p+"=?")
		}
		s += " [" + strings.Join(filled, ", ") + "]"
	}
	return s
}

// targetWrites counts named targets that are not a GET route, which the resolver
// cannot produce; the number is asserted to stay zero.
func targetWrites(r Report) int {
	read := map[string]bool{}
	for _, rt := range r.Surface.ReadRoutes() {
		read[rt.Display()] = true
	}
	n := 0
	for _, res := range r.Results {
		for _, t := range res.Resolution.Targets() {
			if !read[t] {
				n++
			}
		}
	}
	return n
}

func (p Parser) handWrittenAliases() []string {
	out := append([]string{}, p.aliases...)
	sort.Strings(out)
	return out
}

func row(b *strings.Builder, k, v string) {
	b.WriteString("| " + cell(k) + " | " + cell(v) + " |\n")
}

func boolMark(ok bool) string {
	if ok {
		return "yes"
	}
	return "no"
}

func list(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func cell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
