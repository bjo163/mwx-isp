package intentsearch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixturesAreTheTwentyTheCriterionAsksFor(t *testing.T) {
	fs := Fixtures()
	if len(fs) != 20 {
		t.Fatalf("corpus has %d fixtures, the criterion names 20", len(fs))
	}
	seen := map[string]bool{}
	for _, f := range fs {
		if strings.TrimSpace(f.Prompt) == "" {
			t.Error("a fixture has no prompt")
		}
		if seen[f.Prompt] {
			t.Errorf("prompt %q appears twice", f.Prompt)
		}
		seen[f.Prompt] = true
		if strings.TrimSpace(f.Why) == "" {
			t.Errorf("prompt %q is mapped without a reason", f.Prompt)
		}
	}
}

func TestEveryFixtureResolvesAsExpected(t *testing.T) {
	s := testSurface(t)
	for _, res := range Run(s) {
		if !res.Matches {
			t.Errorf("prompt %q: expected %s %s %v, got %s",
				res.Fixture.Prompt, res.Fixture.WantKind, res.Fixture.WantResource,
				res.Fixture.WantParams, actual(res.Resolution))
		}
	}
}

func TestEveryNamedTargetIsAReadRoute(t *testing.T) {
	s := testSurface(t)
	read := map[string]bool{}
	for _, r := range s.ReadRoutes() {
		read[r.Display()] = true
	}
	mutating := map[string]bool{}
	for _, r := range s.Routes {
		if !r.ReadOnly() {
			mutating[r.Display()] = true
		}
	}
	for _, res := range Run(s) {
		targets := res.Resolution.Targets()
		if res.Resolution.Kind == KindRefuse && len(targets) != 0 {
			t.Errorf("refusal for %q still named %v", res.Fixture.Prompt, targets)
			continue
		}
		for _, tg := range targets {
			if !read[tg] {
				t.Errorf("prompt %q named %s, which is not a read route", res.Fixture.Prompt, tg)
			}
			if mutating[tg] {
				t.Errorf("prompt %q named the mutating route %s", res.Fixture.Prompt, tg)
			}
		}
	}
}

func TestAMisreadMutationPromptNamesNoTarget(t *testing.T) {
	s := testSurface(t)
	p, err := NewParser(s)
	if err != nil {
		t.Fatal(err)
	}
	// Every action word in the API, plus one alias, tried both alone and inside a
	// sentence that also names a screen. None of them may produce a target.
	cases := []string{"suspend", "reverse last payment", "pelanggan suspend",
		"reset koneksi nas", "hapus invoice", "bayar tagihan", "disconnect session"}
	for _, c := range cases {
		r := p.Resolve(c)
		if len(r.Targets()) != 0 {
			t.Errorf("%q resolved to %v, a mutation prompt must never resolve", c, r.Targets())
		}
		if r.Kind != KindRefuse {
			t.Errorf("%q came back as %s", c, r.Kind)
		}
	}
	if len(p.MutationVocabulary()) == 0 {
		t.Error("no deny vocabulary")
	}
}

func TestAnAmbiguousPromptRefusesInsteadOfGuessing(t *testing.T) {
	s := testSurface(t)
	p, err := NewParser(s)
	if err != nil {
		t.Fatal(err)
	}
	r := p.Resolve("pelanggan dan user")
	if r.Kind != KindRefuse || len(r.Targets()) != 0 {
		t.Fatalf("tie resolved as %s -> %v", r.Kind, r.Targets())
	}
	if !strings.Contains(strings.Join(r.Refusals, " "), "ambiguous") {
		t.Errorf("refusal did not say why: %v", r.Refusals)
	}
}

func TestAWordThatIsAFilterKeyDoesNotOpenASecondScreen(t *testing.T) {
	s := testSurface(t)
	p, err := NewParser(s)
	if err != nil {
		t.Fatal(err)
	}
	r := p.Resolve("tagihan pelanggan 42")
	if r.Resource != "isp/invoices" || r.IntentID != "invoices" {
		t.Fatalf("got %s (%s): %s", r.Route, r.IntentID, actual(r))
	}
	if v, ok := r.FilterValue("customer_id"); !ok || v != "" {
		t.Errorf("customer_id should be named with no value, got %q ok=%v", v, ok)
	}
}

func TestFilterValuesOnlyComeFromThePageVocabulary(t *testing.T) {
	s := testSurface(t)
	p, err := NewParser(s)
	if err != nil {
		t.Fatal(err)
	}
	off := p.Resolve("pelanggan yang suspended_extra")
	for _, f := range off.Filters {
		if f.Value != "" {
			t.Errorf("%s filled %s from a word the page does not offer", off.Route, f.Param)
		}
	}
	on := p.Resolve("pelanggan yang statusnya suspended")
	if v, ok := on.FilterValue("status"); !ok || v != "suspended" {
		t.Errorf("status should be filled with suspended, got %q ok=%v", v, ok)
	}
}

func TestEveryCatalogedTargetExistsInTheTree(t *testing.T) {
	s := testSurface(t)
	if _, err := NewParser(s); err != nil {
		t.Fatalf("catalog points outside the measured surface: %v", err)
	}
	for _, in := range Catalog() {
		if !s.HasResource(in.Resource) {
			t.Errorf("intent %s names %q, which App.tsx does not declare", in.ID, in.Resource)
		}
		if in.FilterGroup != "" && len(s.FiltersForGroup(in.FilterGroup)) == 0 {
			t.Errorf("intent %s names filter group %q, which no page declares", in.ID, in.FilterGroup)
		}
	}
}

func TestTheCorpusReachesEveryCatalogedIntent(t *testing.T) {
	s := testSurface(t)
	reached, total := Coverage(Run(s))
	if reached != total {
		t.Errorf("the corpus covers %d of %d intents", reached, total)
	}
}

func TestClaimsAllLandOnCodeLines(t *testing.T) {
	r := testReport(t)
	if r.Totals.ClaimsConfirmed != r.Totals.Claims {
		for _, c := range r.Claims {
			if !c.Confirmed {
				t.Errorf("claim %s is %s (hits %v)", c.ID, statusOf(c), c.Hits)
			}
		}
	}
	for _, c := range r.Claims {
		if c.Broken {
			t.Errorf("claim %s cites a missing file %s", c.ID, c.File)
		}
		for _, hit := range c.Hits {
			if !strings.Contains(hit, ":") {
				t.Errorf("claim %s hit %s is not a file:line citation", c.ID, hit)
			}
		}
	}
}

func TestGapsStayOpenAndTheScanIgnoresItself(t *testing.T) {
	r := testReport(t)
	for _, g := range r.Gaps {
		if !g.Open {
			t.Errorf("gap %s got filled: %v", g.ID, g.Hits)
		}
		for _, hit := range g.Hits {
			if strings.HasPrefix(hit, "internal/intentsearch/") {
				t.Errorf("gap %s counted the prototype itself: %s", g.ID, hit)
			}
		}
	}
	if len(r.Gaps) < 5 {
		t.Errorf("only %d gaps cataloged", len(r.Gaps))
	}
}

func TestThePrototypeWritesNothing(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".go") || strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		raw, rerr := os.ReadFile(f.Name())
		if rerr != nil {
			t.Fatal(rerr)
		}
		for _, banned := range []string{"os.WriteFile", "os.Create", "os.MkdirAll", "net/http", "gorm.io", "database/sql", "exec.Command"} {
			if strings.Contains(string(raw), banned) {
				t.Errorf("%s contains %q - the study package must stay read-only", f.Name(), banned)
			}
		}
	}
}

func statusOf(c ClaimStatus) string {
	if c.Broken {
		return "broken"
	}
	return "unconfirmed"
}

func testReport(t *testing.T) Report {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	r, err := Verify(root)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("INTENTSEARCH_DUMP") == "" {
		t.Skip("set INTENTSEARCH_DUMP=1 to print the rendered report")
	}
	fmt.Println(RenderMarkdown(testReport(t)))
}
