package contextualws

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MenuItem is one row of the shipped sidebar menu, parsed from the array the
// product renders it from. Source order is the tab order, so a row's index in
// the parsed slice is the measurement and not a decoration.
type MenuItem struct {
	To          string
	LabelKey    string
	SectionKey  string
	Permissions []string
	Line        int
}

// ResourceView is one Resource declaration in the app shell, with the views it
// registers. A resource is what makes a route: react-admin mounts /<name>.
type ResourceView struct {
	Name      string
	Path      string
	HasList   bool
	HasCreate bool
	HasEdit   bool
	HasShow   bool
	HasIcon   bool
	Line      int
}

// RouteView is one route the shell registers by hand.
type RouteView struct {
	Path        string
	Component   string
	NoLayout    bool
	IsDashboard bool
	Line        int
}

// Role is one arm of the menu filter: the identity level the server can issue
// crossed with the one permission string it can issue.
type Role struct {
	Name       string
	Level      string
	Permission string
}

// Role arms come from the two sources that define the vocabulary, not from a
// list someone typed here: the operator levels the API accepts, and the single
// permission value the API appends.
func roleArms(root string) []Role {
	levels := operatorLevels(root)
	perms := issuedPermissions(root)
	var out []Role
	for _, level := range levels {
		out = append(out, Role{Name: level, Level: level})
		for _, p := range perms {
			out = append(out, Role{Name: level + "+" + p, Level: level, Permission: p})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// visible reproduces the filter the menu component runs, in the same order the
// component reads it: an ungated row is always visible; a gated row is visible
// when the gate names the identity level or one of the issued permissions.
func (m MenuItem) visible(r Role) bool {
	if len(m.Permissions) == 0 {
		return true
	}
	for _, p := range m.Permissions {
		if p == r.Level || (r.Permission != "" && p == r.Permission) {
			return true
		}
	}
	return false
}

var (
	reMenuTo       = regexp.MustCompile(`to:\s*'([^']*)'`)
	reMenuLabel    = regexp.MustCompile(`labelKey:\s*'([^']*)'`)
	reMenuSection  = regexp.MustCompile(`sectionKey:\s*'([^']*)'`)
	reMenuPerms    = regexp.MustCompile(`permissions:\s*\[([^\]]*)\]`)
	reQuotedItem   = regexp.MustCompile(`'([^']*)'`)
	reQuotedGo     = regexp.MustCompile(`"([^"]*)"`)
	reResourceBlk  = regexp.MustCompile(`<Resource\b[\s\S]*?/>`)
	reAttrName     = regexp.MustCompile(`name="([^"]*)"`)
	reViewProp     = regexp.MustCompile(`\b(list|create|edit|show|icon)=\{`)
	reRouteBlk     = regexp.MustCompile(`<Route\s+path="([^"]*)"[^>]*element=\{<\s*([A-Za-z0-9_.]+)`)
	reCustomRoutes = regexp.MustCompile(`<CustomRoutes(\s+noLayout)?>`)
	reDashboardPrp = regexp.MustCompile(`dashboard=\{([A-Za-z0-9_.]+)\}`)
)

// parseMenu reads the menu array. It fails loudly when the file or the array is
// missing, because every number in the study hangs off these rows.
func parseMenu(root string) ([]MenuItem, error) {
	ls := lines(root, menuFile)
	if ls == nil {
		return nil, fmt.Errorf("%s is missing: the menu cannot be measured", menuFile)
	}
	start := -1
	for i, ln := range ls {
		if strings.Contains(ln, "menuItems") && strings.Contains(ln, "=") && strings.Contains(ln, "[") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no menuItems array in %s", menuFile)
	}
	var out []MenuItem
	for i := start + 1; i < len(ls); i++ {
		ln := ls[i]
		trimmed := strings.TrimSpace(ln)
		if trimmed == "];" || trimmed == "]" {
			break
		}
		if !strings.Contains(trimmed, "to:") {
			continue
		}
		item := MenuItem{Line: i + 1}
		if g := reMenuTo.FindStringSubmatch(ln); g != nil {
			item.To = g[1]
		} else {
			continue
		}
		if g := reMenuLabel.FindStringSubmatch(ln); g != nil {
			item.LabelKey = g[1]
		}
		if g := reMenuSection.FindStringSubmatch(ln); g != nil {
			item.SectionKey = g[1]
		}
		if g := reMenuPerms.FindStringSubmatch(ln); g != nil {
			for _, q := range reQuotedItem.FindAllStringSubmatch(g[1], -1) {
				item.Permissions = append(item.Permissions, q[1])
			}
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("menuItems array in %s parsed to zero rows", menuFile)
	}
	return out, nil
}

// parseApp reads the app shell: the resources that make a route, the custom
// routes, and whether each route renders inside the layout that carries the menu.
func parseApp(root string) ([]ResourceView, []RouteView, error) {
	src, ok := readRaw(root, appFile)
	if !ok {
		return nil, nil, fmt.Errorf("%s is missing: routes cannot be measured", appFile)
	}
	body := blankComments(src)

	var resources []ResourceView
	for _, blk := range reResourceBlk.FindAllString(body, -1) {
		name := ""
		if g := reAttrName.FindStringSubmatch(blk); g != nil {
			name = g[1]
		}
		if name == "" {
			continue
		}
		rv := ResourceView{Name: name, Path: "/" + name}
		for _, g := range reViewProp.FindAllStringSubmatch(blk, -1) {
			switch g[1] {
			case "list":
				rv.HasList = true
			case "create":
				rv.HasCreate = true
			case "edit":
				rv.HasEdit = true
			case "show":
				rv.HasShow = true
			case "icon":
				rv.HasIcon = true
			}
		}
		rv.Line = firstLineOf(root, appFile, `name="`+name+`"`)
		resources = append(resources, rv)
	}

	// Routes inherit the layout of the CustomRoutes block that wraps them; a
	// noLayout route never shows the menu, so it is not part of the navigation
	// region the study costs.
	var routes []RouteView
	noLayout := false
	opened := false
	for i, ln := range strings.Split(body, "\n") {
		if g := reCustomRoutes.FindStringSubmatch(ln); g != nil {
			opened = true
			noLayout = strings.Contains(g[1], "noLayout")
			continue
		}
		if strings.Contains(ln, "</CustomRoutes>") {
			opened = false
			noLayout = false
			continue
		}
		if !opened {
			continue
		}
		if g := reRouteBlk.FindStringSubmatch(ln); g != nil {
			routes = append(routes, RouteView{
				Path:      g[1],
				Component: g[2],
				NoLayout:  noLayout,
				Line:      i + 1,
			})
		}
	}

	dashComponent := "Dashboard"
	if g := reDashboardPrp.FindStringSubmatch(body); g != nil {
		dashComponent = g[1]
	}
	routes = append([]RouteView{{Path: "/", Component: dashComponent, IsDashboard: true,
		Line: firstLineOf(root, appFile, "dashboard=")}}, routes...)

	if len(resources) == 0 {
		return nil, nil, fmt.Errorf("no Resource declarations parsed from %s", appFile)
	}
	return resources, routes, nil
}

// declaredSet is the set a task edge may resolve into: every route the shell
// registers, plus the dashboard.
func declaredSet(resources []ResourceView, routes []RouteView) map[string]string {
	set := map[string]string{}
	for _, r := range resources {
		set[r.Path] = "resource " + r.Name
	}
	for _, rt := range routes {
		where := "custom route"
		if rt.IsDashboard {
			where = "dashboard"
		}
		if rt.NoLayout {
			where += " (no layout)"
		}
		set[rt.Path] = where
	}
	return set
}

// MenuRow is one (role, destination) cell of the navigation cost table.
type MenuRow struct {
	Role     string
	To       string
	Label    string
	Section  string
	Gate     string
	Stops    int
	Visible  bool
	LineMenu int
}

// menuCostRows numbers the menu the way a keyboard does: the first focusable
// entry in the region is one stop, and a row the filter removes is not reachable
// from the region at all.
func menuCostRows(menu []MenuItem, roles []Role) []MenuRow {
	var out []MenuRow
	for _, r := range roles {
		stop := 0
		for _, m := range menu {
			if !m.visible(r) {
				out = append(out, MenuRow{Role: r.Name, To: m.To, Label: m.LabelKey,
					Section: m.SectionKey, Gate: gateText(m), Visible: false, LineMenu: m.Line})
				continue
			}
			stop++
			out = append(out, MenuRow{Role: r.Name, To: m.To, Label: m.LabelKey,
				Section: m.SectionKey, Gate: gateText(m), Stops: stop, Visible: true, LineMenu: m.Line})
		}
	}
	return out
}

func gateText(m MenuItem) string {
	if len(m.Permissions) == 0 {
		return "none"
	}
	return strings.Join(m.Permissions, "|")
}

// stopsFor reads one destination out of the cost table for one role. The three
// answers are different facts and the callers need to tell them apart: a
// destination can be absent from the menu entirely, present but filtered away
// for this role, or present and reachable at a given stop number.
func stopsFor(rows []MenuRow, role, to string) (stops int, visible bool, present bool) {
	for _, r := range rows {
		if r.Role == role && r.To == to {
			return r.Stops, r.Visible, true
		}
	}
	return 0, false, false
}

// operatorLevels reads the level vocabulary the API will accept for an operator.
func operatorLevels(root string) []string {
	ls := lines(root, operatorsGo)
	if ls == nil {
		return []string{"operator"}
	}
	seen := map[string]bool{}
	var out []string
	for _, ln := range ls {
		if !strings.Contains(ln, "payload.Level") || !strings.Contains(ln, "!=") {
			continue
		}
		for _, g := range reQuotedGo.FindAllStringSubmatch(ln, -1) {
			if g[1] == "" {
				continue
			}
			if !seen[g[1]] {
				seen[g[1]] = true
				out = append(out, g[1])
			}
		}
	}
	if len(out) == 0 {
		return []string{"operator"}
	}
	sort.Strings(out)
	return out
}

// issuedPermissions reads the permission strings the server can hand a session.
func issuedPermissions(root string) []string {
	ls := lines(root, authFile)
	if ls == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, ln := range ls {
		if !strings.Contains(ln, "permissions") || !strings.Contains(ln, "append") {
			continue
		}
		for _, g := range reQuotedGo.FindAllStringSubmatch(ln, -1) {
			if !seen[g[1]] {
				seen[g[1]] = true
				out = append(out, g[1])
			}
		}
	}
	sort.Strings(out)
	return out
}

func readRaw(root, rel string) (string, bool) {
	return readFile(filepath.Join(root, filepath.FromSlash(rel)))
}
