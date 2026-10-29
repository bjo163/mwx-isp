package intentsearch

import (
	"path/filepath"
	"strings"
	"testing"
)

func testSurface(t *testing.T) Surface {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	s, err := Scan(root)
	if err != nil {
		t.Fatalf("scan %s: %v", root, err)
	}
	if len(s.Routes) == 0 {
		t.Fatalf("no routes found in %s - wrong checkout root", root)
	}
	return s
}

func TestScanSeparatesReadFromMutation(t *testing.T) {
	s := testSurface(t)
	if s.CountByMethod()["GET"] == 0 {
		t.Fatal("no read route found")
	}
	for _, r := range s.Routes {
		if len(r.Path) < 2 || r.Path[0] != '/' {
			t.Errorf("route %q is not a registered path", r.Path)
		}
		if r.Handler == "" || r.File == "" || r.Line <= 0 {
			t.Errorf("route %v has no anchored registration", r)
		}
	}
	if len(s.ReadRoutes()) != s.CountByMethod()["GET"] {
		t.Error("read route list disagrees with the method count")
	}
}

func TestResourceScanFindsTheMenu(t *testing.T) {
	s := testSurface(t)
	if len(s.Resources) < 13 {
		t.Fatalf("found %d resources, expected the 13 the console declares", len(s.Resources))
	}
	for _, r := range s.Resources {
		if !r.HasList {
			t.Errorf("resource %s has no list page, so a navigation could not land on it", r.Name)
		}
		if r.File != "web/src/App.tsx" {
			t.Errorf("resource %s declared outside App.tsx: %s", r.Name, r.File)
		}
	}
	if !s.HasResource("isp/customers") {
		t.Error("the customer register is not in the scanned menu")
	}
}

func TestFilterScanReadsTheDeclaredInputs(t *testing.T) {
	s := testSurface(t)
	var found []Filter
	for _, f := range s.FiltersForGroup("customerFilters") {
		if f.Param == "status" {
			found = append(found, f)
		}
	}
	if len(found) != 1 {
		t.Fatalf("customerFilters should declare exactly one status input, got %v", found)
	}
	status := found[0]
	if status.Component != "StatusSelectInput" {
		t.Errorf("status input is a %s", status.Component)
	}
	want := []string{"pending", "active", "inactive", "suspended", "terminated"}
	if len(status.Values) != len(want) {
		t.Fatalf("status vocabulary = %v, want the five the page declares", status.Values)
	}
	for i, v := range want {
		if status.Values[i] != v {
			t.Errorf("status value %d = %s, want %s", i, status.Values[i], v)
		}
	}
}

func TestMutationVerbsComeFromRoutes(t *testing.T) {
	s := testSurface(t)
	verbs := s.MutationVerbs()
	if len(verbs) == 0 {
		t.Fatal("the admin API has no action route at all?")
	}
	for _, v := range verbs {
		rt := s.RoutesForVerb(v)
		sites := s.ActionSitesFor(v)
		if len(rt) == 0 && len(sites) == 0 {
			t.Errorf("verb %q has neither a route nor a handler guard behind it", v)
		}
		for _, r := range rt {
			if r.ReadOnly() {
				t.Errorf("verb %q resolved to a read route %s", v, r.Display())
			}
		}
	}
	// The measured set must contain the word an operator uses most, and must not
	// contain a collection noun that happens to sit at the end of a route.
	if !contains(verbs, "suspend") || !contains(verbs, "disconnect") {
		t.Errorf("measured verbs are %v, which lost suspend or disconnect", verbs)
	}
	for _, v := range verbs {
		if strings.HasSuffix(v, "s") {
			t.Errorf("verb %q is a collection noun, not an action", v)
		}
	}
}

func TestParamsAreReadFromTheHandlerFile(t *testing.T) {
	s := testSurface(t)
	r, ok := s.RouteFor("isp/customers")
	if !ok {
		t.Fatal("no read route for isp/customers")
	}
	params := s.ParamsFor(r)
	if !contains(params, "q") || !contains(params, "status") {
		t.Errorf("%s reads %v, which cannot serve the customer filter the page declares", r.Display(), params)
	}
}
