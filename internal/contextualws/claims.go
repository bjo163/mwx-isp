package contextualws

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Status of one statement, decided by whether the citation still lands on a code
// line that carries the symbols the statement names.
const (
	confirmed = "confirmed"
	broken    = "broken"
	vacated   = "vacated"
)

// Claim is one statement the memo makes, pinned to a code line.
type Claim struct {
	ID     string
	Text   string
	File   string
	Line   int
	Evid   string
	Status string
	Neg    bool // a negative claim asserts an absence and states its scope
}

// Gap is one limit the study leaves open, with the asset or decision it needs.
type Gap struct {
	ID    string
	Text  string
	Needs string
	Asset string
}

// claimDef is one statement as the table writes it: the sentence, the file and
// the tokens that must sit on one code line for the sentence to still be true,
// the evidence the memo prints next to it, and - for a statement that asserts an
// absence - the check that decides whether the absence still holds.
type claimDef struct {
	id     string
	text   string
	file   string
	tokens []string
	evid   func(*evidence) string
	check  func(*evidence) bool
}

// evalClaims resolves the table against the tree. A citation that no longer
// lands reports broken, a citation whose file is gone reports vacated, and a
// failed absence check downgrades a pinned claim to broken. Nothing is dropped,
// because a claim that quietly disappears is worse than a claim that fails.
func evalClaims(root string, ev *evidence, defs []claimDef) []Claim {
	out := make([]Claim, 0, len(defs))
	for _, d := range defs {
		c := Claim{ID: d.id, Text: d.text, File: d.file, Neg: d.check != nil}
		c.Line = anchorAny(root, d.file, d.tokens)
		switch c.Line {
		case -1:
			c.Status = vacated
		case 0:
			c.Status = broken
		default:
			c.Status = confirmed
		}
		if d.check != nil && !d.check(ev) {
			c.Status = broken
		}
		if d.evid != nil {
			c.Evid = d.evid(ev)
		}
		out = append(out, c)
	}
	return out
}

// evidence holds the measured facts the claims are built from, gathered once.
type evidence struct {
	root string

	menu      []MenuItem
	roles     []Role
	edges     []EdgeRow
	costs     []CostRow
	gates     []GateRow
	resources []ResourceView
	declared  map[string]string

	dynamicNav     []string
	gateMismatch   []GateRow
	parsedTasks    []Task
	guideChapters  int
	setupSteps     int
	tourSteps      int
	dashboardCards int
	adminOnlySteps int
	manualSteps    int

	nativeEdges, pointerEdges, unknownEdges   int
	resolvedEdges, unresolvedEdges            int
	surfacesWithGating, surfacesWithoutGating int

	menuGates       int
	gatesAsserted   int
	gatesUnasserted int
	sections        int
	menuImplements  int

	navFilterAffordances int
	keyboardScenarios    int
	keyboardDestinations int
	ciPlaywrightRuns     int
	ciFrontendSteps      int

	preferenceModels    int
	perOperatorSections int
}

func gatherEvidence(root string, res *Result) *evidence {
	ev := &evidence{root: root}
	ev.menu = res.Menu
	ev.roles = res.Roles
	ev.edges = res.Edges
	ev.costs = res.Costs
	ev.gates = res.Gates
	ev.resources = res.Resources
	ev.parsedTasks = res.Tasks
	ev.declared = declaredSet(res.Resources, res.Routes)
	ev.dynamicNav = dynamicLinks(root, consoleFiles(root))

	for _, g := range ev.gates {
		if g.HiddenEdges > 0 {
			ev.gateMismatch = append(ev.gateMismatch, g)
		}
	}
	for _, t := range res.Tasks {
		if t.Gates {
			ev.surfacesWithGating++
		} else {
			ev.surfacesWithoutGating++
		}
	}

	for _, e := range ev.edges {
		switch e.Keyboard {
		case "native":
			ev.nativeEdges++
		case "mouse-only":
			ev.pointerEdges++
		default:
			ev.unknownEdges++
		}
		if e.Resolves {
			ev.resolvedEdges++
		} else {
			ev.unresolvedEdges++
		}
	}
	for _, m := range ev.menu {
		if len(m.Permissions) > 0 {
			ev.menuGates++
		}
		if m.SectionKey != "" {
			ev.sections++
		}
	}

	ev.gatesAsserted = len(specGateMatrix(root))
	ev.gatesUnasserted = ev.menuGates - ev.gatesAsserted

	ev.navFilterAffordances = countTokens(root, []string{appBarFile, menuFile, layoutFile},
		[]string{"Autocomplete", "Search", "CommandPalette", "input"})
	ev.keyboardScenarios = countMatches(root, keyboardSpec, "test(")
	ev.keyboardDestinations = distinctHashTargets(root, keyboardSpec)
	ev.ciPlaywrightRuns = countTokens(root, []string{ciFile}, []string{"playwright"})
	ev.ciFrontendSteps = countMatches(root, ciFile, "npm run build")

	ev.preferenceModels = countTokens(root, []string{domainDir}, []string{"Preference"})
	ev.perOperatorSections = distinctSections(root)

	ev.guideChapters = countMatches(root, guideFile, "links:")
	ev.setupSteps = countMatches(root, setupFile, "route:")
	ev.tourSteps = countMatches(root, tourFile, "path:")
	ev.dashboardCards = countLiteralCardLinks(root)
	ev.adminOnlySteps = countMatches(root, setupFile, "admin: true")
	ev.manualSteps = countMatches(root, setupFile, "manual: true")
	ev.menuImplements = countMatches(root, layoutFile, "menu={CustomMenu}")
	return ev
}

var (
	reSpecGate   = regexp.MustCompile(`\{\s*href:\s*'([^']*)',\s*visibleFor:\s*([A-Za-z]+),`)
	reCardLine   = regexp.MustCompile(`label:\s*'[^']*',\s*value:[\s\S]*?to:\s*'(/[^']*)'`)
	reHashTarget = regexp.MustCompile(`page\.goto\('/#([^']*)'`)
	reSectionKey = regexp.MustCompile(`(?:read|write)Onboarding(?:Preference|Checks)\([^)]*(?:,\s*'([^']*)')?`)
)

// specGateMatrix reads the role-visibility expectations the browser role matrix
// states, as a destination to role-set mapping.
func specGateMatrix(root string) map[string]string {
	out := map[string]string{}
	for _, ln := range lines(root, roleSpec) {
		if g := reSpecGate.FindStringSubmatch(ln); g != nil {
			out[strings.TrimPrefix(g[1], "#/")] = g[2]
		}
	}
	return out
}

// distinctHashTargets counts the destinations the keyboard test opens.
func distinctHashTargets(root, spec string) int {
	seen := map[string]bool{}
	for _, ln := range lines(root, spec) {
		for _, g := range reHashTarget.FindAllStringSubmatch(ln, -1) {
			seen[g[1]] = true
		}
	}
	return len(seen)
}

// countLiteralCardLinks counts the dashboard's card definitions that carry both
// a label and a literal destination.
func countLiteralCardLinks(root string) int {
	src, ok := readRaw(root, dashboardFile)
	if !ok {
		return 0
	}
	n := 0
	for _, ln := range strings.Split(blankComments(src), "\n") {
		if reCardLine.MatchString(ln) {
			n++
		}
	}
	return n
}

// countTokens counts occurrences of any token across the given files or dirs.
func countTokens(root string, paths []string, tokens []string) int {
	n := 0
	for _, p := range paths {
		var files []string
		if strings.Contains(p, "internal/") {
			files = walkFiles(root, p, ".go")
		} else if strings.HasSuffix(p, ".yml") || strings.HasSuffix(p, ".tsx") || strings.HasSuffix(p, ".ts") {
			files = []string{p}
		} else {
			continue
		}
		for _, f := range files {
			for _, ln := range lines(root, f) {
				for _, t := range tokens {
					if containsBounded(ln, t) {
						n++
						break
					}
				}
			}
		}
	}
	return n
}

// distinctSections counts the per-operator storage sections the console writes.
func distinctSections(root string) int {
	seen := map[string]bool{}
	for _, rel := range walkFiles(root, "web/src/components/onboarding", ".ts", ".tsx") {
		for _, ln := range lines(root, rel) {
			for _, g := range reSectionKey.FindAllStringSubmatch(ln, -1) {
				if g[1] != "" {
					seen[g[1]] = true
				}
			}
		}
	}
	// readOnboardingChecks and writeOnboardingChecks have no section argument:
	// they are fixed to "checks", which the key template names in the storage
	// module itself.
	for _, ln := range lines(root, storageFile) {
		for _, g := range regexp.MustCompile(`'([a-z-]+)'`).FindAllStringSubmatch(ln, -1) {
			if g[1] == "checks" {
				seen[g[1]] = true
			}
		}
	}
	return len(seen)
}

// claims states every sentence the memo is allowed to say, each pinned to the
// code line that proves it.
func claims(ev *evidence) []claimDef {
	out := []claimDef{
		{"C01", "the sidebar is a static array the source orders, so a tab-stop number is a property of the code",
			menuFile, []string{"menuItems"},
			func(*evidence) string { return itoa(len(ev.menu)) + " destinations in source order" }, nil},

		{"C02", "the menu filters rows by identity level or by an issued permission",
			menuFile, []string{"permissions.some", "identity?.level"},
			func(*evidence) string {
				return itoa(ev.menuGates) + " gated rows, " + itoa(len(ev.menu)-ev.menuGates) + " shown to every role"
			}, nil},

		{"C03", "platform_admin is the only permission string the server can hand a session",
			authFile, []string{"permissions = append(permissions"},
			func(*evidence) string { return "issued permissions: " + strings.Join(issuedPermissions(ev.root), ", ") }, nil},

		{"C04", "an operator level is one of the three the API validates",
			operatorsGo, []string{"payload.Level !="},
			func(*evidence) string {
				return itoa(len(ev.roles)) + " role arms = levels " + strings.Join(operatorLevels(ev.root), ", ") +
					" x issued permissions"
			}, nil},

		{"C05", "every destination the menu offers is a route the app shell declares",
			appFile, []string{"<Resource"},
			func(*evidence) string {
				return menuDestinationsOrphan(ev)
			}, nil},

		{"C06", "a section label is text, not a focus stop, so grouping never costs the keyboard anything today",
			menuFile, []string{"sectionKey &&"},
			func(*evidence) string {
				return itoa(ev.sections) + " menu rows carry a section label, 0 of them focusable"
			}, nil},

		{"C07", "the navigation region has no type-to-filter affordance, so order and grouping are the only levers",
			menuFile, []string{"MenuItemLink"},
			func(*evidence) string {
				return itoa(ev.navFilterAffordances) + " filter or palette controls in the app bar, menu and layout"
			}, func(e *evidence) bool { return e.navFilterAffordances == 0 }},

		{"C08", "the browser role matrix states a visibility expectation for every gated destination and no ungated one",
			roleSpec, []string{"menuMatrix", "visibleFor"},
			func(*evidence) string {
				return itoa(ev.gatesAsserted) + " gated destinations asserted, " +
					itoa(ev.gatesUnasserted) + " gated destinations no test asserts"
			}, func(e *evidence) bool { return e.gatesUnasserted == 0 }},

		{"C09", "the user guide is already written as tasks: each chapter is an ordered list of destinations",
			guideFile, []string{"links:"},
			func(*evidence) string {
				return itoa(ev.guideChapters) + " chapters, " + itoa(guideLinkCount(ev)) + " destination links"
			}, nil},

		{"C10", "the getting-started checklist is the only surface that filters its own steps by role",
			setupFile, []string{"visibleSteps"},
			func(*evidence) string {
				return itoa(ev.setupSteps) + " steps, " + itoa(ev.adminOnlySteps) + " marked admin, " +
					itoa(ev.surfacesWithGating) + " of " + itoa(ev.surfacesWithGating+ev.surfacesWithoutGating) +
					" surfaces filter"
			}, func(e *evidence) bool { return e.surfacesWithGating == 1 }},

		{"C11", "a checklist step opens its destination through a router-bound button",
			setupFile, []string{"component={RouterLink}", "to={step.route}"},
			func(*evidence) string { return "carrier: " + carrierOf(ev, setupFile) }, nil},

		{"C12", "the quick tour walks five destinations, all of them declared routes",
			tourFile, []string{"TOUR_STEPS"},
			func(*evidence) string {
				return itoa(ev.tourSteps) + " steps, " + itoa(tourUnresolved(ev)) + " unresolved"
			},
			func(e *evidence) bool { return tourUnresolved(e) == 0 }},

		{"C13", "the dashboard cards are click targets that route, and they are action areas rather than plain boxes",
			dashboardFile, []string{"CardActionArea"},
			func(*evidence) string {
				return itoa(ev.dashboardCards) + " cards, carrier " + carrierOf(ev, dashboardFile)
			}, nil},

		{"C14", "a task surface can offer a destination whose menu entry the same role cannot see",
			setupFile, []string{"route: '/system/config'", "admin: true"},
			func(*evidence) string {
				return itoa(len(ev.gateMismatch)) + " task x role cells offer a hidden destination: " +
					strings.Join(distinctHidden(ev), ", ")
			}, func(e *evidence) bool { return len(e.gateMismatch) > 0 }},

		{"C15", "the config page performs no client-side role check, so the menu filter is the only gate the console shows",
			configFile, []string{"export const SystemConfigPage"},
			func(*evidence) string {
				return itoa(countTokens(ev.root, []string{configFile}, []string{"useIsAdmin", "usePermissions", "identity"})) +
					" role checks inside the page"
			}, func(e *evidence) bool {
				return countTokens(e.root, []string{configFile}, []string{"useIsAdmin", "usePermissions", "identity"}) == 0
			}},

		{"C16", "the only per-operator adaptive state the console keeps is browser storage, and it survives neither another browser nor another machine",
			storageFile, []string{"localStorage.setItem"},
			func(*evidence) string {
				return itoa(ev.perOperatorSections) + " storage sections, " +
					itoa(ev.preferenceModels) + " operator UI-preference models in the domain"
			}, func(e *evidence) bool { return e.preferenceModels == 0 }},

		{"C17", "the keyboard cost this study counts is the unit the repository's own keyboard test already counts",
			keyboardSpec, []string{"Tab", "maxPresses"},
			func(*evidence) string {
				return itoa(ev.keyboardScenarios) + " scenarios, " + itoa(ev.keyboardDestinations) +
					" distinct destinations driven to"
			}, nil},

		{"C18", "CI never runs the browser suite, so no gate in this repository observes a real focus order",
			ciFile, []string{"npm run build"},
			func(*evidence) string {
				return itoa(ev.ciPlaywrightRuns) + " playwright invocations in the workflow, " +
					itoa(ev.ciFrontendSteps) + " frontend build steps"
			}, func(e *evidence) bool { return e.ciPlaywrightRuns == 0 }},

		{"C19", "the console mounts exactly one layout, and the menu inside it is the product's own component",
			layoutFile, []string{"menu={CustomMenu}"},
			func(*evidence) string {
				return itoa(ev.menuImplements) + " layout wiring, " + itoa(len(consoleFiles(ev.root))) +
					" console files measured"
			}, nil},

		{"C20", "every destination a task offers is a route the shell already declares, so a workspace adds no application",
			appFile, []string{"<Route"},
			func(*evidence) string {
				return itoa(ev.resolvedEdges) + " of " + itoa(len(ev.edges)) +
					" task edges resolve to a declared route; " + itoa(len(ev.dynamicNav)) +
					" further controls build their target at runtime"
			}, func(e *evidence) bool { return e.unresolvedEdges == 0 }},

		{"C21", "ordering a task's own destinations in front of the menu never costs a stop, and across the corpus it removes most of the walk",
			menuFile, []string{"filteredMenuItems"},
			func(*evidence) string {
				b, c := 0, 0
				for _, row := range ev.costs {
					b += row.BaselineStops
					c += row.ContextualStops
				}
				return itoa(b) + " baseline stops vs " + itoa(c) + " grouped stops over " +
					itoa(len(ev.costs)) + " task x role cells"
			}, func(e *evidence) bool { return groupedNeverCostsMore(e) }},

		{"C22", "the menu's own row count changes with the role, so ordering cost is already adaptive and only in the direction the role cannot control",
			menuFile, []string{"permissions"},
			func(*evidence) string { return roleRowSpread(ev) }, func(e *evidence) bool { return roleRowSpreadPositive(e) }},

		{"C23", "the resources behind the destinations are read-mostly declared: a task that needs a write has to leave the list",
			appFile, []string{"list={"},
			func(*evidence) string { return viewCoverage(ev.resources) }, nil},

		{"C24", "no product code imports this study, and the study imports nothing that can write",
			ciFile, []string{"go test"},
			func(*evidence) string {
				n := len(importersOutside(ev.root))
				if n == 0 {
					return "0 importers outside internal/" + ownPackage
				}
				return itoa(n) + " importers: " + strings.Join(importersOutside(ev.root), ", ")
			}, func(e *evidence) bool { return len(importersOutside(e.root)) == 0 }},
	}
	return out
}

func gaps(root string, ev *evidence) []Gap {
	_ = root
	return []Gap{
		{"G01", "task usability compared against the baseline is a human measurement: what this study can price is focus stops, not the time or the errors an operator makes while using them",
			"a panel of operators on a seeded console", "human respondents"},
		{"G02", "the stop counts here are derived from source order; the rendered focus order, including everything the component library injects above the menu, is only visible in a browser",
			"a Playwright run against the dev server with a seeded session", "live browser session"},
		{"G03", "a destination the menu hides may still work when reached, or may fail at the API; the study reads the menu, the server answers the request",
			"a signed session per role and a page-load probe", "live per-role sessions"},
		{"G04", "putting a task group in front of the menu is a product decision about TR-F013, not a finding this study can make",
			"a scope decision on TR-F013", "product owner"},
		{"G05", "adaptive in the sense of learning from an operator's own work needs state the product does not have; the study measured that the state is browser-local",
			"a schema and API decision for per-operator workspace state", "schema change"},
		{"G06", itoa(len(ev.dynamicNav)) + " navigation controls in the console build their target at runtime, so a workspace cannot enumerate them from source",
			"seeded records that exercise those controls", "runtime fixtures"},
	}
}

// helpers used by the claim table

func menuDestinationsOrphan(ev *evidence) string {
	var orphans []string
	for _, m := range ev.menu {
		if _, ok := ev.declared[m.To]; !ok {
			orphans = append(orphans, m.To)
		}
	}
	if len(orphans) == 0 {
		return itoa(len(ev.menu)) + " menu destinations, 0 of them without a declared route"
	}
	return itoa(len(orphans)) + " menu destinations with no declared route: " + strings.Join(orphans, ", ")
}

func guideLinkCount(ev *evidence) int {
	n := 0
	for _, t := range ev.tasks() {
		if strings.HasPrefix(t.ID, "guide.") {
			n += len(t.Steps)
		}
	}
	return n
}

func tourUnresolved(ev *evidence) int {
	n := 0
	for _, e := range ev.edges {
		if strings.HasPrefix(e.Task, "tour.") && !e.Resolves {
			n++
		}
	}
	return n
}

func distinctHidden(ev *evidence) []string {
	seen := map[string]bool{}
	var out []string
	for _, g := range ev.gateMismatch {
		for _, h := range g.Hidden {
			if !seen[h] {
				seen[h] = true
				out = append(out, g.Task+" -> "+h)
			}
		}
	}
	sort.Strings(out)
	if len(out) > 6 {
		out = append(out[:6], "...")
	}
	return out
}

func carrierOf(ev *evidence, file string) string {
	for _, t := range ev.tasks() {
		if t.File == file && t.Carrier != "" {
			return t.Carrier + " (" + t.Keyboard + "), line " + itoa(t.CarrierLine)
		}
	}
	return "not found"
}

// groupedNeverCostsMore is the invariant the comparison rests on: a task group
// can only shorten the walk, because its own destinations are numbered 1..k
// instead of scattered through the menu. A task already in menu order saves
// nothing, which is why the claim is about the corpus total, not each cell.
func groupedNeverCostsMore(ev *evidence) bool {
	total := 0
	for _, row := range ev.costs {
		if row.SavedStops < 0 {
			return false
		}
		total += row.SavedStops
	}
	return total > 0
}

func roleRowSpread(ev *evidence) string {
	low, high := "", ""
	lowN, highN := -1, -1
	for _, r := range ev.roles {
		n := 0
		for _, m := range ev.menu {
			if m.visible(r) {
				n++
			}
		}
		if lowN < 0 || n < lowN {
			lowN, low = n, r.Name
		}
		if n > highN {
			highN, high = n, r.Name
		}
	}
	return fmt.Sprintf("%s sees %d rows, %s sees %d rows", low, lowN, high, highN)
}

func roleRowSpreadPositive(ev *evidence) bool {
	counts := map[int]bool{}
	for _, r := range ev.roles {
		n := 0
		for _, m := range ev.menu {
			if m.visible(r) {
				n++
			}
		}
		counts[n] = true
	}
	return len(counts) > 1
}

func viewCoverage(resources []ResourceView) string {
	var list, edit, create, show int
	for _, r := range resources {
		if r.HasList {
			list++
		}
		if r.HasEdit {
			edit++
		}
		if r.HasCreate {
			create++
		}
		if r.HasShow {
			show++
		}
	}
	return itoa(len(resources)) + " resources: " + itoa(list) + " list, " + itoa(create) +
		" create, " + itoa(edit) + " edit, " + itoa(show) + " show"
}

// tasks is a small accessor so a claim can read the parsed surfaces without the
// evidence struct carrying a second copy of them.
func (ev *evidence) tasks() []Task { return ev.parsedTasks }
