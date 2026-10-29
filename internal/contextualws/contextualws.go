// Package contextualws measures the management console's information
// architecture as the source declares it, so that a proposal for an adaptive
// contextual workspace can be compared against the shipped navigation with
// numbers instead of opinion.
//
// The study is read-only: it parses the console's own files, holds the parsed
// tables in memory, and writes nothing. Nothing in cmd/ or internal/ imports
// it, and it imports only the standard library.
//
// # What is measured
//
// The console ships a task-first surface today: the sidebar menu, the
// getting-started checklist, the quick tour, the user guide's chapters and the
// dashboard cards each order a set of destinations for an operator who is doing
// one job. That gives the study a real baseline rather than an imagined one. Two
// quantities are then computed from the parsed source order:
//
//   - keyboard cost of a task, expressed as focus stops inside the navigation
//     region - the same unit the repository's own keyboard test counts with its
//     tabTo helper (web/e2e/keyboard-navigation.spec.ts);
//   - reachability of a task's destinations for each role arm the product can
//     actually produce, because the menu filters and the task surfaces do not
//     filter the same way.
//
// # What counts as evidence
//
// A claim cites a code line, never a comment: comments are blanked to spaces of
// the same length before any symbol is matched, so line numbers survive and
// prose cannot confirm a claim. Identifiers match whole, so MenuItemLink cannot
// be proven by Menu. A claim whose file is gone reads vacated and a claim whose
// line moved reads broken; both are printed, never dropped.
//
// The corpus separates three classes. Product source (web/src, internal) proves
// behavior. The assertion corpus (web/e2e) is only ever used to measure what the
// product has promised to check, never to prove how the product behaves, because
// a test that asserts something is not evidence that it happens.
package contextualws

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Result is one full run of the study: every parsed surface, every derived
// number, and the digest that pins them.
type Result struct {
	Menu     []MenuItem
	Roles    []Role
	MenuCost []MenuRow

	Resources []ResourceView
	Routes    []RouteView

	Tasks []Task
	Edges []EdgeRow

	Costs []CostRow
	Gates []GateRow

	Claims []Claim
	Gaps   []Gap

	Affordances []AffordanceRow
	Pins        []PinRow

	Totals Totals
	Import string

	Digest string
}

// Totals carries the headline counts the memo quotes. Each one is computed by
// the run that reports it, so a table in the memo can never disagree with the
// package that produced it.
type Totals struct {
	Destinations int
	Sections     int
	GatedItems   int
	UngatedItems int
	RoleArms     int

	Resources      int
	ResourceRoutes int
	CustomRoutes   int
	NoLayoutRoutes int

	TaskSurfaces    int
	TaskSteps       int
	TaskEdges       int
	ResolvedEdges   int
	UnresolvedEdges int

	RuntimeBuiltControls int

	GuideChapters  int
	SetupSteps     int
	TourSteps      int
	DashboardCards int

	NativeEdges  int
	MouseEdges   int
	UnknownEdges int

	AdminOnlySteps int
	ManualSteps    int

	EdgesSurfaceButHidden int
	SurfacesWithGating    int
	SurfacesWithoutGating int

	CostRows          int
	StopsBaseline     int
	StopsContextual   int
	StopsSaved        int
	WorstTaskSavings  int
	TasksAllReachable int

	MenuGatesAsserted   int
	MenuGatesUnasserted int

	NavFilterAffordances int
	KeyboardScenarios    int
	KeyboardDestinations int
	CIPlaywrightRuns     int

	PreferenceModels int

	ClaimsConfirmed int
	ClaimsBroken    int
	ClaimsVacated   int
	GapsOpen        int
}

// Run reads the repository at root and produces the whole study. It is
// deterministic: the same tree gives the same digest.
func Run(root string) (*Result, error) {
	menu, err := parseMenu(root)
	if err != nil {
		return nil, err
	}
	resources, routes, err := parseApp(root)
	if err != nil {
		return nil, err
	}
	declared := declaredSet(resources, routes)
	roles := roleArms(root)

	tasks, err := parseTasks(root, declared)
	if err != nil {
		return nil, err
	}

	res := &Result{
		Menu:      menu,
		Roles:     roles,
		Resources: resources,
		Routes:    routes,
		Tasks:     tasks,
	}

	res.MenuCost = menuCostRows(menu, roles)
	res.Edges = edgeRows(tasks)
	res.Costs = costRows(tasks, menu, roles)
	res.Gates = gateRows(tasks, menu, roles)
	res.Affordances = affordanceRows(root)
	res.Pins = pinRows(root)

	ev := gatherEvidence(root, res)
	res.Claims = evalClaims(root, ev, claims(ev))
	res.Gaps = gaps(root, ev)
	res.Totals = totals(res, ev)
	res.Import = importSet(root)
	res.Digest = digest(res)
	return res, nil
}

func totals(res *Result, ev *evidence) Totals {
	t := Totals{}
	sections := map[string]bool{}
	for _, m := range res.Menu {
		t.Destinations++
		if m.SectionKey != "" {
			sections[m.SectionKey] = true
		}
		if len(m.Permissions) > 0 {
			t.GatedItems++
		} else {
			t.UngatedItems++
		}
	}
	t.Sections = len(sections)
	t.RoleArms = len(res.Roles)

	t.Resources = len(res.Resources)
	for _, r := range res.Routes {
		switch {
		case r.NoLayout:
			t.NoLayoutRoutes++
		default:
			t.CustomRoutes++
		}
	}
	t.ResourceRoutes = len(res.Resources)

	for _, task := range res.Tasks {
		t.TaskSurfaces++
		t.TaskSteps += len(task.Steps)
	}
	for _, e := range res.Edges {
		t.TaskEdges++
		if e.Resolves {
			t.ResolvedEdges++
		} else {
			t.UnresolvedEdges++
		}
		switch e.Keyboard {
		case "native":
			t.NativeEdges++
		case "mouse-only":
			t.MouseEdges++
		default:
			t.UnknownEdges++
		}
	}
	t.GuideChapters = ev.guideChapters
	t.SetupSteps = ev.setupSteps
	t.TourSteps = ev.tourSteps
	t.DashboardCards = ev.dashboardCards
	t.AdminOnlySteps = ev.adminOnlySteps
	t.ManualSteps = ev.manualSteps

	for _, g := range res.Gates {
		t.EdgesSurfaceButHidden += g.HiddenEdges
	}
	for _, task := range res.Tasks {
		if task.Gates {
			t.SurfacesWithGating++
		} else {
			t.SurfacesWithoutGating++
		}
	}

	t.CostRows = len(res.Costs)
	for _, c := range res.Costs {
		t.StopsBaseline += c.BaselineStops
		t.StopsContextual += c.ContextualStops
		t.StopsSaved += c.SavedStops
		if c.SavedStops > t.WorstTaskSavings {
			t.WorstTaskSavings = c.SavedStops
		}
		if len(c.Blocked) == 0 {
			t.TasksAllReachable++
		}
	}

	t.MenuGatesAsserted = ev.gatesAsserted
	t.MenuGatesUnasserted = t.GatedItems - ev.gatesAsserted
	t.RuntimeBuiltControls = len(ev.dynamicNav)
	t.NavFilterAffordances = ev.navFilterAffordances
	t.KeyboardScenarios = ev.keyboardScenarios
	t.KeyboardDestinations = ev.keyboardDestinations
	t.CIPlaywrightRuns = ev.ciPlaywrightRuns
	t.PreferenceModels = ev.preferenceModels

	for _, c := range res.Claims {
		switch c.Status {
		case confirmed:
			t.ClaimsConfirmed++
		case broken:
			t.ClaimsBroken++
		case vacated:
			t.ClaimsVacated++
		}
	}
	t.GapsOpen = len(res.Gaps)
	return t
}

// totalsRows flattens the headline numbers for the dump.
func totalsRows(t Totals) [][]string {
	rows := [][]string{
		{"menu destinations parsed from the shipped menu", itoa(t.Destinations)},
		{"section labels the menu draws", itoa(t.Sections)},
		{"destinations the menu gates by role", itoa(t.GatedItems)},
		{"destinations the menu shows to every role", itoa(t.UngatedItems)},
		{"role arms the product can produce", itoa(t.RoleArms)},
		{"resources declared in the app shell", itoa(t.Resources)},
		{"in-layout custom routes", itoa(t.CustomRoutes)},
		{"routes rendered outside the layout", itoa(t.NoLayoutRoutes)},
		{"task surfaces that already order destinations", itoa(t.TaskSurfaces)},
		{"steps across those surfaces", itoa(t.TaskSteps)},
		{"destination edges across those surfaces", itoa(t.TaskEdges)},
		{"edges that resolve to a declared route", itoa(t.ResolvedEdges)},
		{"edges whose destination is not a declared route", itoa(t.UnresolvedEdges)},
		{"controls whose destination is built at runtime", itoa(t.RuntimeBuiltControls)},
		{"user-guide chapters", itoa(t.GuideChapters)},
		{"getting-started steps", itoa(t.SetupSteps)},
		{"getting-started steps restricted to an admin level", itoa(t.AdminOnlySteps)},
		{"getting-started steps a checklist cannot verify", itoa(t.ManualSteps)},
		{"quick-tour steps", itoa(t.TourSteps)},
		{"dashboard quick links", itoa(t.DashboardCards)},
		{"edges whose control is keyboard-native as written", itoa(t.NativeEdges)},
		{"edges whose control needs a pointer", itoa(t.MouseEdges)},
		{"edges whose control the parser could not classify", itoa(t.UnknownEdges)},
		{"task surfaces that filter by role", itoa(t.SurfacesWithGating)},
		{"task surfaces that do not", itoa(t.SurfacesWithoutGating)},
		{"edges shown to a role whose menu hides the destination", itoa(t.EdgesSurfaceButHidden)},
		{"task x role cost rows measured", itoa(t.CostRows)},
		{"task rows whose every destination is in that role's menu", itoa(t.TasksAllReachable)},
		{"baseline focus stops over all cost rows", itoa(t.StopsBaseline)},
		{"task-grouped focus stops over the same rows", itoa(t.StopsContextual)},
		{"focus stops saved by grouping", itoa(t.StopsSaved)},
		{"largest saving in one task", itoa(t.WorstTaskSavings)},
		{"menu gates the browser role matrix asserts", itoa(t.MenuGatesAsserted)},
		{"menu gates no test asserts", itoa(t.MenuGatesUnasserted)},
		{"type-to-filter affordances in the navigation region", itoa(t.NavFilterAffordances)},
		{"keyboard scenarios in the shipped keyboard test", itoa(t.KeyboardScenarios)},
		{"destinations those scenarios cover", itoa(t.KeyboardDestinations)},
		{"CI jobs that run Playwright", itoa(t.CIPlaywrightRuns)},
		{"operator UI-preference models in the domain", itoa(t.PreferenceModels)},
		{"statements pinned to a code line", itoa(t.ClaimsConfirmed)},
		{"statements whose line moved", itoa(t.ClaimsBroken)},
		{"statements whose file is gone", itoa(t.ClaimsVacated)},
		{"limits this study leaves open", itoa(t.GapsOpen)},
	}
	return append([][]string{{"measure", "value"}}, rows...)
}

// digest hashes everything the study promises. Only values a run can repeat are
// included, so the digest is the reproducibility statement the criterion asks
// for.
func digest(r *Result) string {
	var b strings.Builder
	for _, row := range r.MenuCost {
		fmt.Fprintf(&b, "menustop\t%s\t%s\t%d\t%t\n", row.Role, row.To, row.Stops, row.Visible)
	}
	for _, row := range r.Costs {
		fmt.Fprintf(&b, "cost\t%s\t%s\t%d\t%d\t%d\t%s\n",
			row.Task, row.Role, row.BaselineStops, row.ContextualStops, row.SavedStops,
			strings.Join(row.Blocked, ","))
	}
	for _, row := range r.Gates {
		fmt.Fprintf(&b, "gate\t%s\t%s\t%s\t%s\n", row.Task, row.Role, row.Surfaces, strings.Join(row.Hidden, ","))
	}
	for _, row := range r.Edges {
		fmt.Fprintf(&b, "edge\t%s\t%s\t%s\t%s\t%s\t%t\n",
			row.Task, row.To, row.Label, row.Carrier, row.Keyboard, row.Resolves)
	}
	for _, row := range r.Affordances {
		fmt.Fprintf(&b, "affordance\t%s\t%s\t%d\n", row.File, row.Carrier, row.Count)
	}
	for _, row := range r.Pins {
		fmt.Fprintf(&b, "pin\t%s\t%s\t%d\n", row.File, row.Symbol, row.Line)
	}
	for _, c := range r.Claims {
		fmt.Fprintf(&b, "claim\t%s\t%s\t%s:%d\n", c.ID, c.Status, c.File, c.Line)
	}
	for _, g := range r.Gaps {
		fmt.Fprintf(&b, "gap\t%s\t%s\n", g.ID, g.Needs)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

// importSet lists what this package builds on, so the read-only promise is a
// checked number rather than a sentence.
func importSet(root string) string {
	return strings.Join(packageImports(root, ownPackageDir(root)), ", ")
}
