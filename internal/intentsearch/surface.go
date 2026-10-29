// Package intentsearch measures what a bounded intent search could resolve in the
// operator console today, and pins the one safety property the FUT-001 criterion
// asks for: a prompt the parser misreads must not reach a mutating endpoint.
//
// The package is a research prototype. It reads the repository as text, writes
// nothing, performs no network call, and is imported by no product code.
package intentsearch

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Route is one admin API registration, read out of the source text.
type Route struct {
	// Method is GET, POST, PUT or DELETE, taken from the registration helper name.
	Method string
	// Path is the registered path, e.g. /isp/customers.
	Path string
	// Handler is the Go function the path is bound to.
	Handler string
	// File and Line point at the registration, not at the handler body.
	File string
	Line int
}

// ReadOnly is the whole permission model of a resolution: only a GET can be named.
func (r Route) ReadOnly() bool { return r.Method == "GET" }

// Display is how a route is quoted in a report, e.g. "GET /isp/customers".
func (r Route) Display() string { return r.Method + " " + r.Path }

// Action is the trailing word of an action route - /isp/subscriptions/:id/suspend
// yields "suspend". A path that ends in a parameter or in the collection itself
// has no action word, so it contributes nothing to the deny vocabulary.
func (r Route) Action() string {
	segs := strings.Split(strings.Trim(r.Path, "/"), "/")
	if len(segs) < 2 {
		return ""
	}
	last := segs[len(segs)-1]
	if last == "" || strings.HasPrefix(last, ":") {
		return ""
	}
	if !strings.HasPrefix(segs[len(segs)-2], ":") {
		return ""
	}
	if !actionWordRe.MatchString(last) {
		return ""
	}
	// A plural trailing segment is a collection, not an action: the platform
	// operator route ends in "operators" and must not deny that word.
	if strings.HasSuffix(last, "s") {
		return ""
	}
	return last
}

var actionWordRe = regexp.MustCompile(`^[a-z][a-z_]*$`)

// Resource is one <Resource> entry in web/src/App.tsx, which is the menu the
// operator can actually reach.
type Resource struct {
	Name      string
	HasList   bool
	HasShow   bool
	HasEdit   bool
	HasCreate bool
	File      string
	Line      int
}

// Readable reports whether the resource has a view-only surface to navigate to.
func (r Resource) Readable() bool { return r.HasList || r.HasShow }

// Filter is one filter input a list page declares.
type Filter struct {
	// Group is the array the page passes to <List filters={...}>, e.g. customerFilters.
	Group string
	// Param is the query key the product reads back, e.g. status.
	Param string
	// Component is the widget, e.g. StatusSelectInput.
	Component string
	// Values are the choices a select offers, taken from the page's own vocabulary.
	Values []string
	File   string
	Line   int
}

// Surface is the measured vocabulary a resolution may draw from. Everything in it
// comes from the repository; nothing is declared in this package.
type Surface struct {
	Routes    []Route
	Resources []Resource
	Filters   []Filter
	// ActionSites records where a mutation action word was measured when it is not
	// a route segment - the subscription lifecycle handler takes it as a value.
	ActionSites map[string][]string
	// ParamsByFile lists the query keys each handler file reads.
	ParamsByFile map[string][]string
	// PathMap is the console's own translation from a menu name to an API path,
	// read from resourcePathMap in web/src/providers/dataProvider.ts.
	PathMap map[string]string
	// ScannedFiles counts the product files the scan read.
	ScannedFiles int
	// Root is the repository root the scan started from.
	Root string
}

// Scan reads the admin API registrations, the resource list and the filter
// declarations from a checkout of this repository.
func Scan(root string) (Surface, error) {
	s := Surface{Root: root, ParamsByFile: map[string][]string{}, PathMap: map[string]string{},
		ActionSites: map[string][]string{}}
	apiDir := filepath.Join(root, "internal", "adminapi")
	routes, params, actions, files, err := scanRoutes(apiDir, root)
	if err != nil {
		return Surface{}, fmt.Errorf("scan admin api: %w", err)
	}
	s.Routes = routes
	s.ParamsByFile = params
	s.ActionSites = actions
	s.ScannedFiles += files

	pathMap, err := scanPathMap(filepath.Join(root, "web", "src", "providers", "dataProvider.ts"))
	if err != nil {
		return Surface{}, fmt.Errorf("scan resource path map: %w", err)
	}
	s.PathMap = pathMap

	resources, err := scanResources(filepath.Join(root, "web", "src", "App.tsx"), root)
	if err != nil {
		return Surface{}, fmt.Errorf("scan resources: %w", err)
	}
	s.Resources = resources

	resDir := filepath.Join(root, "web", "src", "resources")
	filters, files, err := scanFilters(resDir, root)
	if err != nil {
		return Surface{}, fmt.Errorf("scan filters: %w", err)
	}
	s.Filters = filters
	s.ScannedFiles += files

	sort.Slice(s.Routes, func(i, j int) bool {
		if s.Routes[i].Path != s.Routes[j].Path {
			return s.Routes[i].Path < s.Routes[j].Path
		}
		return s.Routes[i].Method < s.Routes[j].Method
	})
	sort.Slice(s.Resources, func(i, j int) bool { return s.Resources[i].Name < s.Resources[j].Name })
	sort.Slice(s.Filters, func(i, j int) bool {
		if s.Filters[i].Group != s.Filters[j].Group {
			return s.Filters[i].Group < s.Filters[j].Group
		}
		return s.Filters[i].Param < s.Filters[j].Param
	})
	return s, nil
}

var routeRe = regexp.MustCompile(`webserver\.(Api(GET|POST|PUT|PATCH|DELETE))\(\s*"([^"]*)"\s*,\s*([A-Za-z0-9_]+)`)

func scanRoutes(dir, root string) ([]Route, map[string][]string, map[string][]string, int, error) {
	routes := []Route{}
	params := map[string][]string{}
	actions := map[string][]string{}
	count := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel := relPath(root, p)
		if filepath.Ext(rel) != ".go" || strings.HasSuffix(rel, "_test.go") {
			return nil
		}
		//nolint:gosec // G304: path comes from the walk over this directory only.
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		count++
		code := blankComments(string(raw))
		for _, m := range routeRe.FindAllStringSubmatch(code, -1) {
			method := strings.TrimPrefix(m[1], "Api")
			routes = append(routes, Route{
				Method:  method,
				Path:    m[3],
				Handler: m[4],
				File:    rel,
				Line:    lineOf(code, strings.Index(code, m[0])),
			})
		}
		for _, m := range queryParamRe.FindAllStringSubmatch(code, -1) {
			if !contains(params[rel], m[1]) {
				params[rel] = append(params[rel], m[1])
			}
		}
		for _, m := range actionGuardRe.FindAllStringSubmatch(code, -1) {
			site := rel + ":" + strconv.Itoa(lineOf(code, strings.Index(code, m[0])))
			if !contains(actions[m[1]], site) {
				actions[m[1]] = append(actions[m[1]], site)
			}
		}
		for f := range params {
			sort.Strings(params[f])
		}
		return nil
	})
	if err != nil {
		return nil, nil, nil, 0, err
	}
	for k := range actions {
		sort.Strings(actions[k])
	}
	return routes, params, actions, count, nil
}

// actionGuardRe catches the mutation verbs the API takes as a value instead of a
// path segment: the subscription lifecycle guard rejects anything but activate,
// suspend, reactivate, disconnect or terminate.
var actionGuardRe = regexp.MustCompile(`(?i)action !=\s*"([a-z]+)"`)

var queryParamRe = regexp.MustCompile(`QueryParam\("([^"]+)"\)`)

var resourceRe = regexp.MustCompile(`<Resource\b[^>]*>`)

var (
	pathMapRe   = regexp.MustCompile(`(?s)const resourcePathMap[^=]*= \{(.*?)\};`)
	pathEntryRe = regexp.MustCompile(`'([^']+)':\s*'([^']+)'`)
)

// scanPathMap reads the console's menu-name to API-path table. Without it, a menu
// entry named radius/users would look for a route named /radius/users and find
// none, which is the drift this study is built to catch.
func scanPathMap(path string) (map[string]string, error) {
	//nolint:gosec // G304: path is a fixed file inside the checkout the scan reads.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	body := pathMapRe.FindStringSubmatch(blankComments(string(raw)))
	if body == nil {
		return nil, fmt.Errorf("no resourcePathMap in %s", path)
	}
	for _, e := range pathEntryRe.FindAllStringSubmatch(body[1], -1) {
		out[e[1]] = e[2]
	}
	return out, nil
}

func scanResources(path, root string) ([]Resource, error) {
	//nolint:gosec // G304: path is a fixed file inside the checkout the scan reads.
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	code := blankComments(string(raw))
	rel := relPath(root, path)
	var out []Resource
	for _, m := range resourceRe.FindAllString(code, -1) {
		name := firstGroup(nameAttrRe, m)
		if name == "" {
			continue
		}
		out = append(out, Resource{
			Name:      name,
			HasList:   strings.Contains(m, "list="),
			HasShow:   strings.Contains(m, "show="),
			HasEdit:   strings.Contains(m, "edit="),
			HasCreate: strings.Contains(m, "create="),
			File:      rel,
			Line:      lineOf(code, strings.Index(code, m)),
		})
	}
	return out, nil
}

var nameAttrRe = regexp.MustCompile(`name="([^"]+)"`)

var (
	filterGroupRe = regexp.MustCompile(`(?s)const (\w*[Ff]ilters?)\s*(?::[^=]*)?= \[(.*?)\n?\];`)
	inputRe       = regexp.MustCompile(`<([A-Za-z]+(?:Input|Search))\b([^>]*)>`)
	sourceRe      = regexp.MustCompile(`source="([^"]+)"`)
	valuesRe      = regexp.MustCompile(`values=\{(\w+)\}`)
	statusVocabRe = regexp.MustCompile(`const (\w+):\s*StatusValue\[\]\s*=\s*\[([^\]]*)\]`)
	stringLitRe   = regexp.MustCompile(`'([^']+)'`)
)

func scanFilters(dir, root string) ([]Filter, int, error) {
	var out []Filter
	count := 0
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		rel := relPath(root, p)
		ext := filepath.Ext(rel)
		if ext != ".tsx" && ext != ".ts" {
			return nil
		}
		//nolint:gosec // G304: path comes from the walk over this directory only.
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		count++
		code := blankComments(string(raw))
		vocab := map[string][]string{}
		for _, m := range statusVocabRe.FindAllStringSubmatch(code, -1) {
			var vals []string
			for _, lit := range stringLitRe.FindAllStringSubmatch(m[2], -1) {
				vals = append(vals, lit[1])
			}
			vocab[m[1]] = vals
		}
		for _, g := range filterGroupRe.FindAllStringSubmatch(code, -1) {
			group, body := g[1], g[2]
			at := strings.Index(code, g[0])
			for _, in := range inputRe.FindAllStringSubmatch(body, -1) {
				comp, attrs := in[1], in[2]
				param := firstGroup(sourceRe, attrs)
				if param == "" && comp == "StatusSelectInput" {
					// The widget defaults its source to status; isp.tsx declares it.
					param = "status"
				}
				if param == "" {
					continue
				}
				f := Filter{
					Group:     group,
					Param:     param,
					Component: comp,
					Values:    vocab[firstGroup(valuesRe, attrs)],
					File:      rel,
					Line:      lineOf(code, at),
				}
				if !hasFilter(out, f) {
					out = append(out, f)
				}
			}
		}
		return nil
	})
	if err != nil {
		return nil, 0, err
	}
	return out, count, nil
}

// MutationVerbs lists the action words the API itself uses for a state change.
// They come from two places: the trailing segment of an action route, and the
// value a mutation handler compares its action against. Measuring only the first
// would miss suspend, which is the one an operator asks for most often.
func (s Surface) MutationVerbs() []string {
	seen := map[string]bool{}
	var out []string
	for _, r := range s.Routes {
		if r.ReadOnly() {
			continue
		}
		a := r.Action()
		if a == "" || seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	for a := range s.ActionSites {
		if seen[a] {
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// ActionSitesFor returns where a value-shaped action word was measured.
func (s Surface) ActionSitesFor(verb string) []string { return s.ActionSites[verb] }

// RoutesForVerb returns every mutating registration carrying that action word.
func (s Surface) RoutesForVerb(verb string) []Route {
	var out []Route
	for _, r := range s.Routes {
		if !r.ReadOnly() && r.Action() == verb {
			out = append(out, r)
		}
	}
	return out
}

// ReadRoutes returns the only targets a resolution is allowed to name.
func (s Surface) ReadRoutes() []Route {
	var out []Route
	for _, r := range s.Routes {
		if r.ReadOnly() {
			out = append(out, r)
		}
	}
	return out
}

// Endpoint is the API path the console asks for when a menu entry is opened. The
// map is the product's own: a name it does not list resolves to itself.
func (s Surface) Endpoint(resource string) string {
	if v, ok := s.PathMap[resource]; ok {
		return v
	}
	return resource
}

// RouteFor finds the list route a resource name resolves to, which is the chain
// from a menu entry to the query keys the product reads for it.
func (s Surface) RouteFor(resource string) (Route, bool) {
	want := "/" + s.Endpoint(resource)
	for _, r := range s.Routes {
		if r.ReadOnly() && r.Path == want {
			return r, true
		}
	}
	return Route{}, false
}

// ParamsFor lists the query keys read by the file that registers the route.
func (s Surface) ParamsFor(r Route) []string {
	return s.ParamsByFile[r.File]
}

// FiltersForGroup returns the inputs one list page declares.
func (s Surface) FiltersForGroup(group string) []Filter {
	var out []Filter
	for _, f := range s.Filters {
		if f.Group == group {
			out = append(out, f)
		}
	}
	return out
}

// ResourceNames lists the menu entries in sort order.
func (s Surface) ResourceNames() []string {
	out := make([]string, 0, len(s.Resources))
	for _, r := range s.Resources {
		out = append(out, r.Name)
	}
	return out
}

// HasResource reports whether a name is a menu entry the scan found.
func (s Surface) HasResource(name string) bool {
	for _, r := range s.Resources {
		if r.Name == name {
			return true
		}
	}
	return false
}

// CountByMethod returns the route totals per HTTP method, for the report.
func (s Surface) CountByMethod() map[string]int {
	out := map[string]int{}
	for _, r := range s.Routes {
		out[r.Method]++
	}
	return out
}

// hasFilter keeps a duplicate declaration - a stale .backup copy of a page - from
// appearing twice as the same input.
func hasFilter(xs []Filter, f Filter) bool {
	for _, v := range xs {
		if v.Group == f.Group && v.Param == f.Param {
			return true
		}
	}
	return false
}

func firstGroup(re *regexp.Regexp, text string) string {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func relPath(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return filepath.ToSlash(p)
	}
	return filepath.ToSlash(rel)
}

// lineOf converts a byte offset into a one-based line number.
func lineOf(text string, at int) int {
	if at <= 0 {
		return 1
	}
	return strings.Count(text[:at], "\n") + 1
}
