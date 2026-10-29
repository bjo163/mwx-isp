package contextualws

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// CostRow is one (task, role) cell of the navigation cost comparison. Both
// columns count forward focus stops inside the navigation region - the unit the
// repository's own keyboard test counts - over the same deduplicated
// destination sequence. The only difference is which list numbers the stops: the
// shipped menu, or the task's own order.
type CostRow struct {
	Task         string
	Surface      string
	Name         string
	Role         string
	Destinations int
	Blocked      []string

	BaselineStops    int // sum of each destination's index in that role's menu
	ContextualStops  int // sum of each destination's index inside the task
	SavedStops       int
	WorstDestination string
	WorstStops       int
}

// costRows compares the shipped ordering with a task-grouped ordering, per role.
func costRows(tasks []Task, menu []MenuItem, roles []Role) []CostRow {
	stops := menuCostRows(menu, roles)
	var out []CostRow
	for _, t := range tasks {
		for _, r := range roles {
			row := CostRow{Task: t.ID, Surface: t.Surface, Name: t.Name, Role: r.Name}
			seq := orderedDestinations(t, r)
			row.Destinations = len(seq)
			for i, to := range seq {
				n, visible, present := stopsFor(stops, r.Name, to)
				if !present {
					row.Blocked = append(row.Blocked, to+" (no menu entry at all)")
					continue
				}
				if !visible {
					row.Blocked = append(row.Blocked, to+" (filtered away for "+r.Name+")")
					continue
				}
				row.BaselineStops += n
				row.ContextualStops += i + 1
				if n > row.WorstStops {
					row.WorstStops = n
					row.WorstDestination = to
				}
			}
			row.SavedStops = row.BaselineStops - row.ContextualStops
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Task != out[j].Task {
			return out[i].Task < out[j].Task
		}
		return out[i].Role < out[j].Role
	})
	return out
}

// orderedDestinations is the destination sequence for one role: the steps the
// surface shows that role, deduplicated in source order.
func orderedDestinations(t Task, r Role) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range t.stepsFor(r) {
		if s.To == "" || seen[s.To] {
			continue
		}
		seen[s.To] = true
		out = append(out, s.To)
	}
	return out
}

// GateRow is one (task, role) cell of the reachability check: the surface offers
// these steps to this role, and the role's own menu hides some of them.
type GateRow struct {
	Task        string
	Surface     string
	Role        string
	Surfaces    string
	Hidden      []string
	HiddenEdges int
	Offered     int
}

// gateRows measures the disagreement between a task surface and the menu for the
// same role. A surface that does not filter shows every step to every role, so a
// destination the menu gates is still offered as a task action.
func gateRows(tasks []Task, menu []MenuItem, roles []Role) []GateRow {
	stops := menuCostRows(menu, roles)
	var out []GateRow
	for _, t := range tasks {
		for _, r := range roles {
			row := GateRow{Task: t.ID, Surface: t.Surface, Role: r.Name}
			if t.Gates {
				row.Surfaces = "filters its own steps"
			} else {
				row.Surfaces = "shows every step to every role"
			}
			seen := map[string]bool{}
			for _, to := range orderedDestinations(t, r) {
				if seen[to] {
					continue
				}
				seen[to] = true
				row.Offered++
				_, visible, present := stopsFor(stops, r.Name, to)
				switch {
				case !present:
					row.Hidden = append(row.Hidden, to+" (no menu entry at all)")
					row.HiddenEdges++
				case !visible:
					row.Hidden = append(row.Hidden, to+" (hidden by that role's gate)")
					row.HiddenEdges++
				}
			}
			out = append(out, row)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Task != out[j].Task {
			return out[i].Task < out[j].Task
		}
		return out[i].Role < out[j].Role
	})
	return out
}

// AffordanceRow counts the navigation controls of one kind in one console file.
type AffordanceRow struct {
	File    string
	Carrier string
	Count   int
	Kind    string
}

var reRouterLinkProp = regexp.MustCompile(`component=\{RouterLink\}`)

// affordanceRows walks the console's own components and pages, excluding the two
// routes the shell renders outside the layout, and counts the navigation
// controls by the element that carries them. This is the keyboard-parity
// evidence: a workspace built from these controls inherits their tab behavior,
// and a control outside the native set is a path a keyboard operator cannot
// follow.
func affordanceRows(root string) []AffordanceRow {
	type key struct{ file, carrier, kind string }
	counts := map[key]int{}
	for _, rel := range consoleFiles(root) {
		for _, ln := range lines(root, rel) {
			if !strings.Contains(ln, "to=") && !strings.Contains(ln, "onClick=") {
				continue
			}
			tag, kind := classifyCarrier(ln)
			if tag == "" || kind == "unknown" {
				continue
			}
			if !strings.Contains(ln, "to=") {
				kind += " (handler)"
			}
			if reRouterLinkProp.FindStringIndex(ln) != nil {
				kind += " router"
			}
			counts[key{rel, tag, kind}]++
		}
	}
	var out []AffordanceRow
	for k, c := range counts {
		out = append(out, AffordanceRow{File: k.file, Carrier: k.carrier, Kind: k.kind, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Carrier != out[j].Carrier {
			return out[i].Carrier < out[j].Carrier
		}
		return out[i].Kind < out[j].Kind
	})
	return out
}

// consoleFiles lists the frontend files that render inside the layout, so the
// public landing page and the customer portal are excluded by derivation from
// the app shell rather than by a hand-kept list.
func consoleFiles(root string) []string {
	_, routes, err := parseApp(root)
	if err != nil {
		return nil
	}
	public := map[string]bool{}
	for _, r := range routes {
		if r.NoLayout {
			public[r.Component] = true
		}
	}
	out := map[string]bool{}
	for _, rel := range append(walkFiles(root, "web/src/components", ".tsx", ".ts"),
		walkFiles(root, "web/src/pages", ".tsx", ".ts")...) {
		base := filepath.Base(rel)
		if strings.HasSuffix(base, ".test.tsx") || strings.HasSuffix(base, ".test.ts") {
			continue
		}
		// A public page may import a shared component; the component is still
		// part of the console, so exclusion happens by file name only.
		if public[strings.TrimSuffix(base, filepath.Ext(base))] {
			continue
		}
		out[rel] = true
	}
	var files []string
	for f := range out {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// PinRow is one mechanism line the memo cites. The runner finds the line, so a
// citation that moves shows up as drift instead of quietly pointing elsewhere.
type PinRow struct {
	File   string
	Symbol string
	Line   int
	Status string
}

// pinRows locates the code lines that carry the study's mechanism claims.
func pinRows(root string) []PinRow {
	pins := []struct {
		file, symbol string
		tokens       []string
	}{
		{menuFile, "the menu is a static source array", []string{"menuItems", "["}},
		{menuFile, "the menu filters by identity level or issued permission", []string{"filteredMenuItems", "filter"}},
		{menuFile, "a filtered row renders as a focusable menu link", []string{"MenuItemLink"}},
		{menuFile, "a section label is not a focusable element", []string{"sectionKey", "<Typography"}},
		{appFile, "a resource declaration makes a route", []string{"<Resource"}},
		{appFile, "a custom route registers a page", []string{"<Route"}},
		{appBarFile, "the app bar carries the only chrome above the menu", []string{"IconButton"}},
		{setupFile, "the checklist hides admin steps from a lower level", []string{"visibleSteps", "filter"}},
		{setupFile, "a checklist step opens its route through the router link", []string{"to={step.route}"}},
		{guideFile, "a guide chapter carries an ordered link list", []string{"links:"}},
		{guideFile, "a guide link renders as a button bound to the router", []string{"to={path}"}},
		{tourFile, "a tour step renders as a button bound to the router", []string{"to={current.path}"}},
		{dashboardFile, "a dashboard card is a clickable action area", []string{"to={to}"}},
		{storageFile, "the only per-operator state is browser storage", []string{"localStorage.setItem"}},
		{keyboardSpec, "the repo's own keyboard cost unit is a Tab press", []string{"Tab"}},
		{customerTimelineFile, "a record page links to a destination built from the record", []string{"to={event.source_path}"}},
	}
	var out []PinRow
	for _, p := range pins {
		line := anchorAny(root, p.file, p.tokens)
		status := "pinned"
		switch line {
		case -1:
			status = "vacated"
		case 0:
			status = "broken"
		}
		out = append(out, PinRow{File: p.file, Symbol: p.symbol, Line: line, Status: status})
	}
	return out
}

// anchorAny finds the first line that carries every token listed, each matched
// as a whole identifier when it is identifier-shaped.
func anchorAny(root, rel string, tokens []string) int {
	ls := lines(root, rel)
	if ls == nil {
		return -1
	}
	for i, ln := range ls {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		ok := true
		for _, t := range tokens {
			if !containsBounded(ln, t) {
				ok = false
				break
			}
		}
		if ok {
			return i + 1
		}
	}
	return 0
}
