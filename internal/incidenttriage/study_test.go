package incidenttriage

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func testResult(t *testing.T) *Result {
	t.Helper()
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

func TestParseMirrorsTheExport(t *testing.T) {
	in := Parse([]string{`{"level":"error","msg":"boom"}`, "not json at all", ""})
	if len(in.Lines) != 2 {
		t.Fatalf("parsed %d lines, want 2 (an unstructured line still survives)", len(in.Lines))
	}
	if in.Lines[1].Text != "not json at all" || len(in.Lines[1].Fields) != 0 {
		t.Errorf("unstructured line lost its text: %+v", in.Lines[1])
	}
	if !slices.Contains(in.Bad, 2) {
		t.Error("a blank line is not recorded as unusable")
	}
}

func TestSanitizeRunsTheProductsOwnRuleSet(t *testing.T) {
	in := Parse([]string{`{"level":"error","msg":"bind password=hunter2 for a@example.com at 10.1.2.3",` +
		`"nas_secret":"s3cr3t","customername":"Budi Santoso","framed_ip":"10.20.30.44"}`})
	secrets := []string{"hunter2", "s3cr3t", "Budi Santoso", "10.20.30.44", "a@example.com"}
	out, stats := in.Sanitize(secrets)
	if len(out) != 1 {
		t.Fatalf("got %d lines", len(out))
	}
	text := out[0].Text()
	for _, s := range secrets {
		if strings.Contains(text, s) {
			t.Errorf("%q survived redaction: %s", s, text)
		}
	}
	if stats.SecretKeys != 1 || stats.PIIKeys != 1 || stats.AddressKeys != 1 {
		t.Errorf("classifier touched secret=%d pii=%d address=%d, want 1/1/1",
			stats.SecretKeys, stats.PIIKeys, stats.AddressKeys)
	}
	if !strings.Contains(text, "[REDACTED]") {
		t.Error("the marker the product writes is missing")
	}
}

func TestTheCorpusCarriesSomethingToRedact(t *testing.T) {
	r := testResult(t)
	if r.Stats.SecretKeys == 0 || r.Stats.PIIKeys == 0 || r.Stats.AddressKeys == 0 {
		t.Errorf("the corpus pre-cleaned itself: %+v", r.Stats)
	}
	if r.Totals.SecretValues == 0 {
		t.Error("no secret values were planted, so the leak check proves nothing")
	}
	if r.Totals.Leaked != 0 {
		t.Errorf("triage input leaked %v", r.Leaked)
	}
}

func TestGroupingProducesOneIncidentPerCase(t *testing.T) {
	cases := Dataset()
	incidents := Group(mustSanitize(t, cases), 2*time.Minute)
	if _, err := Pair(cases, incidents); err != nil {
		t.Fatal(err)
	}
	if len(incidents) != len(cases) {
		t.Fatalf("grouping produced %d incidents for %d cases", len(incidents), len(cases))
	}
}

func TestGroupingMergesOnlyLinesInsideTheWindow(t *testing.T) {
	lines := []SanitizedLine{
		{Index: 0, Namespace: "radiusd", Timestamp: "2026-10-29T03:00:00Z"},
		{Index: 1, Namespace: "radiusd", Timestamp: "2026-10-29T03:01:00Z"},
		{Index: 2, Namespace: "radiusd", Timestamp: "2026-10-29T03:30:00Z"},
	}
	got := Group(lines, 2*time.Minute)
	if len(got) != 2 {
		t.Fatalf("grouped into %d incidents, want 2: %+v", len(got), got)
	}
	if len(got[0].Lines) != 2 {
		t.Errorf("the first incident holds %d lines, want 2", len(got[0].Lines))
	}
}

func TestGroupingDoesNotMergeDifferentSubsystems(t *testing.T) {
	lines := []SanitizedLine{
		{Index: 0, Namespace: "radiusd", Timestamp: "2026-10-29T03:00:00Z"},
		{Index: 1, Namespace: "adminapi", Timestamp: "2026-10-29T03:00:30Z"},
	}
	if got := Group(lines, 2*time.Minute); len(got) != 2 {
		t.Errorf("two namespaces grouped into %d incidents", len(got))
	}
}

func TestPairRefusesADatasetThatDoesNotLineUp(t *testing.T) {
	cases := Dataset()
	lines := mustSanitize(t, cases)
	// Losing the final line keeps the incident count right but shortens one
	// incident; the pairing must notice rather than mislabel it.
	if _, err := Pair(cases, Group(lines[:len(lines)-1], 2*time.Minute)); err == nil {
		t.Error("pairing accepted an incident that lost a line")
	}
	if _, err := Pair(cases[:len(cases)-1], Group(lines, 2*time.Minute)); err == nil {
		t.Error("pairing accepted more incidents than the corpus has labels")
	}
}

func TestDatasetIsTheCorpusTheCriterionAsksFor(t *testing.T) {
	cases := Dataset()
	if len(cases) != 20 {
		t.Fatalf("corpus has %d incidents, want 20", len(cases))
	}
	ids := map[string]bool{}
	known := append(CauseIDs(), Unresolved)
	for _, c := range cases {
		if ids[c.ID] {
			t.Errorf("duplicate case id %s", c.ID)
		}
		ids[c.ID] = true
		if len(c.Labels) != 1 {
			t.Errorf("%s declares %d labels; the scorer reads one", c.ID, len(c.Labels))
		}
		if !slices.Contains(known, c.Labels[0]) {
			t.Errorf("%s is labeled %q, which the taxonomy does not define", c.ID, c.Labels[0])
		}
		if len(c.Records) == 0 {
			t.Errorf("%s carries no lines", c.ID)
		}
		if strings.TrimSpace(c.Why) == "" {
			t.Errorf("%s has no stated reason for its label", c.ID)
		}
	}
}

func TestEveryLabeledCaseReachesEveryCauseClass(t *testing.T) {
	r := testResult(t)
	reached := map[string]bool{}
	for _, p := range r.Causes {
		if p.Exact > 0 {
			reached[p.Cause] = true
		}
	}
	for _, id := range CauseIDs() {
		if !reached[id] {
			t.Errorf("class %q is never correctly named by the rules", id)
		}
	}
}

func TestTheRuleEngineIsRightWhereTheLinesSaySo(t *testing.T) {
	r := testResult(t)
	s := scoreOf(t, r, "rule-token-match")
	if s.Hallucinate != 0 {
		t.Errorf("the rule engine hallucinated %d verdicts", s.Hallucinate)
	}
	if s.Wrong != 0 {
		t.Errorf("the rule engine misnamed %d incidents", s.Wrong)
	}
	if s.Correct != r.Totals.Labeled {
		t.Errorf("correct=%d but the corpus labels %d incidents", s.Correct, r.Totals.Labeled)
	}
	if s.Accuracy() != 1 {
		t.Errorf("accuracy %v, want 1 over the named verdicts", s.Accuracy())
	}
}

func TestTheRuleEngineAbstainsRatherThanGuesses(t *testing.T) {
	r := testResult(t)
	s := scoreOf(t, r, "rule-token-match")
	if s.Abstained != r.Totals.AbstainCases {
		t.Fatalf("abstained %d, want %d", s.Abstained, r.Totals.AbstainCases)
	}
	byCase := map[string]Verdict{}
	for i, p := range r.Pairs {
		byCase[p.Case.ID] = r.Verdicts["rule-token-match"][i]
	}
	if v := byCase["c17"]; v.Cause != UnresolvedCause ||
		!strings.Contains(v.Reason, "profile-unavailable") ||
		!strings.Contains(v.Reason, "malformed-packet") ||
		!strings.Contains(v.Reason, "guess") {
		t.Errorf("a tie should refuse and name both classes, got %+v", v)
	}
	for _, id := range []string{"c16", "c18"} {
		if v := byCase[id]; v.Cause != UnresolvedCause {
			t.Errorf("%s should abstain, got %+v", id, v)
		}
	}
}

func TestTheNamespaceFieldIsTooCoarseForTheTaxonomy(t *testing.T) {
	r := testResult(t)
	if len(r.Namespaces) == 0 {
		t.Fatal("no namespace literal was measured, so the coarseness argument has no evidence")
	}
	if len(r.Namespaces) >= len(Causes()) {
		t.Errorf("%d namespace literals against %d cause classes: a subsystem name may now carry the taxonomy",
			len(r.Namespaces), len(Causes()))
	}
	if r.Totals.JobSites != 0 {
		t.Errorf("product code attaches a job field on %d lines; the corpus groups without it", r.Totals.JobSites)
	}
}

func TestEverySignalResolvesToAProductCodeLine(t *testing.T) {
	r := testResult(t)
	if r.Totals.SignalsGrounded != r.Totals.SignalsTotal {
		var missing []string
		for _, s := range allSignals() {
			if r.Sites[s] == "" {
				missing = append(missing, s)
			}
		}
		t.Errorf("%d of %d taxonomy tokens are not written by product code: %v",
			r.Totals.SignalsTotal-r.Totals.SignalsGrounded, r.Totals.SignalsTotal, missing)
	}
}

func TestTheNamespaceBaselineIsBlindOnTheRadiusSide(t *testing.T) {
	r := testResult(t)
	s := scoreOf(t, r, "namespace-baseline")
	if s.Hallucinate == 0 {
		t.Error("a classifier that reads only the subsystem name reported no unsupported answers")
	}
	if s.Accuracy() >= scoreOf(t, r, "rule-token-match").Accuracy() {
		t.Errorf("the namespace baseline (%v) did not lose to the token rules (%v)",
			s.Accuracy(), scoreOf(t, r, "rule-token-match").Accuracy())
	}
}

func TestAlwaysNamingBuysCoverageWithHallucination(t *testing.T) {
	r := testResult(t)
	s := scoreOf(t, r, "always-name")
	if s.Hallucinate == 0 || s.HallucinationRate() == 0 {
		t.Errorf("the never-abstains engine reported no hallucination: %v", s)
	}
	if s.Decided <= scoreOf(t, r, "rule-token-match").Decided {
		t.Errorf("always-naming did not increase coverage: %d vs %d",
			s.Decided, scoreOf(t, r, "rule-token-match").Decided)
	}
	if s.Accuracy() >= scoreOf(t, r, "rule-token-match").Accuracy() {
		t.Error("always-naming looked better than abstaining, which is the whole trap this measures")
	}
}

func TestHallucinationDefinitionNeedsSupportNotJustWardrobe(t *testing.T) {
	inc := Incident{ID: "x", Lines: []SanitizedLine{{Index: 3, Level: "error", Message: "failed to get profile from cache"}}}
	good := Verdict{Cause: "profile-unavailable", Evidence: []Evidence{
		{Cause: "profile-unavailable", Token: "failed to get profile from cache", LineIndex: 3, Site: "internal/domain/radius_user_getters.go:33"}}}
	if hallucinated(good, inc) {
		t.Error("a supported verdict was counted as hallucinated")
	}
	naked := Verdict{Cause: "profile-unavailable"}
	if !hallucinated(naked, inc) {
		t.Error("a named cause with no evidence is a hallucination by definition")
	}
	ghostLine := good
	ghostLine.Evidence[0] = Evidence{Token: "failed to get profile from cache", LineIndex: 99, Site: "a.go:1"}
	if !hallucinated(ghostLine, inc) {
		t.Error("evidence citing a line the incident does not contain was accepted")
	}
	orphan := good
	orphan.Evidence[0] = Evidence{Token: "failed to get profile from cache", LineIndex: 3, Site: ""}
	if !hallucinated(orphan, inc) {
		t.Error("a token with no product code site behind it was accepted as evidence")
	}
	invent := good
	invent.Evidence[0] = Evidence{Token: "totally absent token", LineIndex: 3, Site: "a.go:1"}
	if !hallucinated(invent, inc) {
		t.Error("evidence whose token is not in the lines was accepted")
	}
}

func TestClaimsAllLandOnCodeLines(t *testing.T) {
	r := testResult(t)
	for _, c := range r.Claims {
		if !c.Confirmed {
			t.Errorf("claim %s is %s; file %q token %q", c.ID, statusWord(c), c.File, c.Token)
		}
		for _, h := range c.Hits {
			if strings.HasSuffix(h, ":0") {
				t.Errorf("claim %s cites %s, which is not a line", c.ID, h)
			}
		}
	}
	if r.Totals.ClaimsConfirmed != r.Totals.Claims {
		t.Errorf("%d of %d claims confirmed", r.Totals.ClaimsConfirmed, r.Totals.Claims)
	}
}

func TestGapsStayOpenAndTheScanIgnoresItself(t *testing.T) {
	r := testResult(t)
	for _, g := range r.Gaps {
		if !g.Open {
			t.Errorf("gap %s is vacated by %v; if the product changed, so did this study", g.ID, g.Hits)
		}
		for _, h := range g.Hits {
			if strings.Contains(h, "incidenttriage") {
				t.Errorf("gap %s was refuted by the prototype itself: %s", g.ID, h)
			}
		}
	}
}

func TestThePrototypeWritesNothing(t *testing.T) {
	files, err := productFiles(repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.Contains(filepath.ToSlash(f), "incidenttriage") {
			t.Error("the walk did not exclude this package")
		}
	}
	own, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	banned := []string{"os.WriteFile", "os.Create", "os.MkdirAll", "os.Remove", "net/http",
		"gorm.io", "database/sql", "exec.Command", "http.Post"}
	for _, name := range own {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := readFile(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, b := range banned {
			if strings.Contains(src, b) {
				t.Errorf("%s mentions %q", name, b)
			}
		}
	}
}

func scoreOf(t *testing.T, r *Result, name string) EngineScore {
	t.Helper()
	for _, s := range r.Scores {
		if s.Engine == name {
			return s
		}
	}
	t.Fatalf("engine %q did not run", name)
	return EngineScore{}
}

func statusWord(c ClaimStatus) string {
	switch {
	case c.Broken:
		return "broken (file unreadable)"
	case c.Confirmed:
		return "confirmed"
	default:
		return "unconfirmed"
	}
}

func mustSanitize(t *testing.T, cases []Case) []SanitizedLine {
	t.Helper()
	out, _ := Parse(RawLines(cases)).Sanitize(Secrets(cases))
	return out
}
