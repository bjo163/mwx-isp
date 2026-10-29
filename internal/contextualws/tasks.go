package contextualws

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Step is one position in a task the product itself spells out: a checklist
// row, a tour page, a guide chapter's link, a dashboard card.
type Step struct {
	ID         string
	Title      string
	To         string
	AdminOnly  bool
	Manual     bool
	Resolves   bool
	DeclaredBy string
	Line       int
}

// Task is one shipped task surface: an ordered set of destinations an operator
// is meant to visit while doing one job.
type Task struct {
	ID      string
	Surface string
	Name    string
	File    string
	Line    int

	// Gates records whether the surface filters its own steps by role. The
	// checklist does; the guide, the tour and the dashboard do not.
	Gates bool

	Carrier     string
	Keyboard    string
	CarrierLine int

	Steps []Step
}

// stepsFor returns the steps that surface for a role. Only a gating surface
// removes anything.
func (t Task) stepsFor(r Role) []Step {
	var out []Step
	admin := r.Level == "super" || r.Level == "admin"
	for _, s := range t.Steps {
		if t.Gates && s.AdminOnly && !admin {
			continue
		}
		out = append(out, s)
	}
	return out
}

var (
	reGuideChapter = regexp.MustCompile(`\{\s*id:\s*'([^']*)',\s*title:\s*'([^']*)'`)
	reGuideLink    = regexp.MustCompile(`\[\s*'([^']*)'\s*,\s*'(/[^']*)'\s*\]`)
	reStepRoute    = regexp.MustCompile(`route:\s*'([^']*)'`)
	reStepAdmin    = regexp.MustCompile(`admin:\s*(true|false)`)
	reStepManual   = regexp.MustCompile(`manual:\s*(true|false)`)
	reTourPath     = regexp.MustCompile(`path:\s*'([^']*)'`)
	reTourLink     = regexp.MustCompile(`link:\s*'([^']*)'`)
	reCardTo       = regexp.MustCompile(`to:\s*'([^']*)'`)
	reCardLabel    = regexp.MustCompile(`label:\s*'([^']*)'`)
	reJSXTag       = regexp.MustCompile(`<([A-Za-z][A-Za-z0-9_.]*)`)
)

// keyboardNative lists the MUI and react-admin primitives that already take part
// in the tab order. A link written through one of them needs no mouse.
var keyboardNative = map[string]bool{
	"Button": true, "IconButton": true, "CardActionArea": true, "MenuItemLink": true,
	"Link": true, "ListItem": true, "ListItemButton": true, "Tab": true, "Checkbox": true,
	"ToggleThemeButton": true, "MenuItem": true,
}

// keyboardMouseOnly lists the elements that carry a handler but are not focusable
// as written, which is the case a keyboard operator inherits.
var keyboardMouseOnly = map[string]bool{
	"Box": true, "div": true, "span": true, "Stack": true, "Card": true, "Typography": true,
	"Grid": true, "Paper": true,
}

// classifyCarrier names the tag that carries a link on a JSX line and says
// whether that tag is in the tab order as written.
func classifyCarrier(line string) (string, string) {
	idx := strings.Index(line, "to=")
	if idx < 0 {
		idx = strings.Index(line, "onClick=")
	}
	if idx < 0 {
		return "", "unknown"
	}
	head := line[:idx]
	var tag string
	for _, m := range reJSXTag.FindAllStringSubmatch(head, -1) {
		tag = m[1]
	}
	if tag == "" {
		return "", "unknown"
	}
	switch {
	case keyboardNative[tag]:
		return tag, "native"
	case keyboardMouseOnly[tag]:
		return tag, "mouse-only"
	default:
		return tag, "unknown"
	}
}

// linkLine finds the JSX line that renders a surface's links from a marker that
// only that line carries, such as the expression bound to `to`.
func linkLine(root, file, marker string) (string, string, int) {
	for i, ln := range lines(root, file) {
		if strings.Contains(ln, marker) {
			tag, kind := classifyCarrier(ln)
			return tag, kind, i + 1
		}
	}
	return "", "unknown", 0
}

// parseTasks builds every task surface the console ships.
func parseTasks(root string, declared map[string]string) ([]Task, error) {
	var out []Task

	guide, err := parseGuide(root, declared)
	if err != nil {
		return nil, err
	}
	out = append(out, guide...)

	setup, err := parseSetup(root, declared)
	if err != nil {
		return nil, err
	}
	out = append(out, setup)

	tour, err := parseTour(root, declared)
	if err != nil {
		return nil, err
	}
	out = append(out, tour)

	cards, err := parseDashboardCards(root, declared)
	if err != nil {
		return nil, err
	}
	out = append(out, cards...)

	for _, t := range out {
		if len(t.Steps) == 0 {
			return nil, fmt.Errorf("task %s parsed with no steps", t.ID)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

// parseGuide reads the user guide's chapters. Each chapter is a task: the page
// writes it as an ordered list of destination links.
func parseGuide(root string, declared map[string]string) ([]Task, error) {
	ls := lines(root, guideFile)
	if ls == nil {
		return nil, fmt.Errorf("%s is missing", guideFile)
	}
	carrier, kind, carrierLine := linkLine(root, guideFile, "to={path}")
	var out []Task
	for i, ln := range ls {
		head := reGuideChapter.FindStringSubmatch(ln)
		if head == nil {
			continue
		}
		t := Task{ID: "guide." + head[1], Surface: "user guide", Name: head[2],
			File: guideFile, Line: i + 1, Gates: false, Carrier: carrier, Keyboard: kind, CarrierLine: carrierLine}
		for _, g := range reGuideLink.FindAllStringSubmatch(ln, -1) {
			t.Steps = append(t.Steps, Step{ID: head[1] + "." + strings.TrimPrefix(g[2], "/"),
				Title: g[1], To: g[2], Line: i + 1})
		}
		if len(t.Steps) > 0 {
			out = append(out, t)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no guide chapters with links in %s", guideFile)
	}
	out = resolveTasks(out, declared)
	return out, nil
}

// parseSetup reads the getting-started checklist, the one surface that does
// filter its own steps.
func parseSetup(root string, declared map[string]string) (Task, error) {
	ls := lines(root, setupFile)
	if ls == nil {
		return Task{}, fmt.Errorf("%s is missing", setupFile)
	}
	carrier, kind, carrierLine := linkLine(root, setupFile, "to={step.route}")
	t := Task{ID: "setup.checklist", Surface: "getting started", Name: "Setup checklist",
		File: setupFile, Gates: true, Carrier: carrier, Keyboard: kind, CarrierLine: carrierLine}
	for i, ln := range ls {
		if !strings.Contains(ln, "id:") || !strings.Contains(ln, "route:") {
			continue
		}
		if t.Line == 0 {
			t.Line = i + 1
		}
		step := Step{Line: i + 1}
		if g := reCardLabel.FindStringSubmatch(ln); g != nil {
			step.Title = g[1]
		}
		if g := regexp.MustCompile(`id:\s*'([^']*)'`).FindStringSubmatch(ln); g != nil {
			step.ID = "setup." + g[1]
		}
		if g := reStepRoute.FindStringSubmatch(ln); g != nil {
			step.To = g[1]
		}
		if g := reStepAdmin.FindStringSubmatch(ln); g != nil {
			step.AdminOnly = g[1] == "true"
		}
		if g := reStepManual.FindStringSubmatch(ln); g != nil {
			step.Manual = g[1] == "true"
		}
		if step.ID == "" || step.To == "" {
			continue
		}
		t.Steps = append(t.Steps, step)
	}
	if len(t.Steps) == 0 {
		return t, fmt.Errorf("no setup steps parsed from %s", setupFile)
	}
	t.Steps = resolveSteps(t.Steps, declared)
	return t, nil
}

// parseTour reads the quick tour dialog.
func parseTour(root string, declared map[string]string) (Task, error) {
	ls := lines(root, tourFile)
	if ls == nil {
		return Task{}, fmt.Errorf("%s is missing", tourFile)
	}
	carrier, kind, carrierLine := linkLine(root, tourFile, "to={current.path}")
	t := Task{ID: "tour.quick", Surface: "quick tour", Name: "Quick tour",
		File: tourFile, Gates: false, Carrier: carrier, Keyboard: kind, CarrierLine: carrierLine}
	for i, ln := range ls {
		if !strings.Contains(ln, "path:") || !strings.Contains(ln, "title:") {
			continue
		}
		if t.Line == 0 {
			t.Line = i + 1
		}
		step := Step{Line: i + 1}
		if g := reCardLabel.FindStringSubmatch(ln); g != nil {
			step.Title = g[1]
		}
		if g := reTourPath.FindStringSubmatch(ln); g != nil {
			step.To = g[1]
		}
		if g := reTourLink.FindStringSubmatch(ln); g != nil {
			step.ID = "tour." + g[1]
		}
		if step.To == "" {
			continue
		}
		t.Steps = append(t.Steps, step)
	}
	if len(t.Steps) == 0 {
		return t, fmt.Errorf("no tour steps parsed from %s", tourFile)
	}
	t.Steps = resolveSteps(t.Steps, declared)
	return t, nil
}

// parseDashboardCards reads the dashboard's clickable summary cards. Each card
// is its own one-destination task, so the study can name the card rather than a
// group.
func parseDashboardCards(root string, declared map[string]string) ([]Task, error) {
	ls := lines(root, dashboardFile)
	if ls == nil {
		return nil, fmt.Errorf("%s is missing", dashboardFile)
	}
	carrier, kind, carrierLine := linkLine(root, dashboardFile, "to={to}")
	var out []Task
	for i, ln := range ls {
		if !strings.Contains(ln, "label:") || !reCardTo.MatchString(ln) {
			continue
		}
		label := reCardLabel.FindStringSubmatch(ln)[1]
		to := reCardTo.FindStringSubmatch(ln)[1]
		out = append(out, Task{
			ID: "dashboard." + label, Surface: "dashboard card", Name: label,
			File: dashboardFile, Line: i + 1, Gates: false,
			Carrier: carrier, Keyboard: kind, CarrierLine: carrierLine,
			Steps: []Step{{ID: "dashboard." + label, Title: label, To: to, Line: i + 1}},
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no dashboard cards parsed from %s", dashboardFile)
	}
	out = resolveTasks(out, declared)
	return out, nil
}

// EdgeRow is one measured link: a destination a task surface offers, the
// control that carries it, and whether that destination is a route the shell
// actually declares.
type EdgeRow struct {
	Task       string
	Surface    string
	To         string
	Label      string
	Carrier    string
	Keyboard   string
	Resolves   bool
	DeclaredBy string
	Line       int
}

// resolveTasks marks every step of every task whose destination is a route the
// app shell declares. A step that does not resolve is a task pointing somewhere
// the console cannot navigate to, which is a finding rather than a parsing artifact.
func resolveTasks(tasks []Task, declared map[string]string) []Task {
	out := make([]Task, 0, len(tasks))
	for _, t := range tasks {
		t.Steps = resolveSteps(t.Steps, declared)
		out = append(out, t)
	}
	return out
}

// resolveSteps returns the steps with their destinations priced against the
// declared route set.
func resolveSteps(steps []Step, declared map[string]string) []Step {
	out := make([]Step, 0, len(steps))
	for _, st := range steps {
		if where, ok := declared[st.To]; ok {
			st.Resolves = true
			st.DeclaredBy = where
		} else {
			st.DeclaredBy = "not declared in the app shell"
		}
		out = append(out, st)
	}
	return out
}

// edgeRows flattens every (task, destination) pair with its control and whether
// the destination exists as a declared route.
func edgeRows(tasks []Task) []EdgeRow {
	var out []EdgeRow
	for _, t := range tasks {
		for _, s := range t.Steps {
			out = append(out, EdgeRow{Task: t.ID, Surface: t.Surface, To: s.To, Label: s.Title,
				Carrier: t.Carrier, Keyboard: t.Keyboard, Line: s.Line,
				Resolves: s.Resolves, DeclaredBy: s.DeclaredBy})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Task != out[j].Task {
			return out[i].Task < out[j].Task
		}
		return out[i].Line < out[j].Line
	})
	return out
}

// dynamicLinks counts the navigation controls in the console that a static read
// cannot resolve because their target is built at runtime.
func dynamicLinks(root string, consoleFiles []string) []string {
	var out []string
	for _, rel := range consoleFiles {
		base := filepath.Base(rel)
		if base == "App.tsx" || base == "main.tsx" {
			continue
		}
		for i, ln := range lines(root, rel) {
			if !strings.Contains(ln, "to={") && !strings.Contains(ln, "to={`") {
				continue
			}
			if strings.Contains(ln, "to={step.route}") || strings.Contains(ln, "to={current.path}") ||
				strings.Contains(ln, "to={path}") || strings.Contains(ln, "to={to}") {
				continue // the four task surfaces already measured above
			}
			tag, _ := classifyCarrier(ln)
			out = append(out, fmt.Sprintf("%s:%d <%s>", rel, i+1, tag))
		}
	}
	sort.Strings(out)
	return out
}
