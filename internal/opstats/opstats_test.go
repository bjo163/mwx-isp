package opstats

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", wd)
		}
		dir = parent
	}
}

func runOnce(t *testing.T) *Result {
	t.Helper()
	res, err := Run(repoRoot(t))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	return res
}

// TestNothingImportsTheStudy keeps the study out of the product's dependency graph.
func TestNothingImportsTheStudy(t *testing.T) {
	root := repoRoot(t)
	got := importersOutside(root)
	if len(got) != 0 {
		t.Fatalf("product code imports the study: %v", got)
	}
}

// TestTheStudyWritesNothing snapshots the trees it reads before and after a run.
func TestTheStudyWritesNothing(t *testing.T) {
	root := repoRoot(t)
	before := snapshot(t, root, "internal", "docs", "config")
	_, _ = Run(root)
	after := snapshot(t, root, "internal", "docs", "config")
	if before != after {
		t.Fatalf("the run changed the tree: %s -> %s", before, after)
	}
}

func snapshot(t *testing.T, root string, dirs ...string) string {
	t.Helper()
	h := sha256.New()
	var names []string
	for _, d := range dirs {
		base := filepath.Join(root, d)
		if _, err := os.Stat(base); err != nil {
			continue
		}
		err := filepath.WalkDir(base, func(path string, e fs.DirEntry, err error) error {
			if err != nil || e == nil || e.IsDir() {
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return nil
			}
			names = append(names, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	sort.Strings(names)
	for _, rel := range names {
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		_, _ = h.Write([]byte(rel + ":" + itoa(int(st.Size())) + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TestTheStudyReadsNoDatabase pins the promise that no connection, driver or ORM is
// imported: the fixture is the only data this study holds.
func TestTheStudyReadsNoDatabase(t *testing.T) {
	root := repoRoot(t)
	for _, imp := range packageImports(root, ownPackageDir(root)) {
		for _, banned := range []string{"gorm.io", "database/sql", "lib/pq", "pgx", "redis"} {
			if strings.Contains(imp, banned) {
				t.Fatalf("the study imports %s, which would make it a client rather than a measurement", imp)
			}
		}
	}
}

// TestTheImportAllowlistIsNotVacuous fails if the parse finds nothing, so an empty
// import set cannot masquerade as a passing standard-library check.
func TestTheImportAllowlistIsNotVacuous(t *testing.T) {
	root := repoRoot(t)
	imps := packageImports(root, ownPackageDir(root))
	if len(imps) == 0 {
		t.Fatal("no imports parsed; the stdlib-only claim is unmeasured")
	}
	for _, imp := range imps {
		if strings.Contains(imp, "/") && !strings.HasPrefix(imp, "go/") && !strings.Contains(imp, "golang.org") &&
			!strings.HasPrefix(imp, "crypto/") && !strings.HasPrefix(imp, "encoding/") &&
			!strings.HasPrefix(imp, "text/") && !strings.HasPrefix(imp, "math/") {
			if strings.HasPrefix(imp, "github.com/bjo163/mwx-isp") {
				t.Fatalf("the study imports product package %s", imp)
			}
		}
	}
}

// TestEverySurfaceSpecResolves refuses a table that quietly shrank: each route, handler
// and served type must be found in the file that names it.
func TestEverySurfaceSpecResolves(t *testing.T) {
	res := runOnce(t)
	if len(res.Broken) != 0 {
		t.Fatalf("surface specs did not resolve: %v", res.Broken)
	}
	if res.Totals.Surfaces != len(surfaceSpecs()) {
		t.Fatalf("surface count drifted: %d vs %d", res.Totals.Surfaces, len(surfaceSpecs()))
	}
	if res.Totals.Fields == 0 {
		t.Fatal("no served fields parsed; the census is measuring nothing")
	}
}

// TestEveryAggregateIsAttributed requires each scored number to carry the attribution
// rule that found its statement, so no field is classified from its name alone.
func TestEveryAggregateIsAttributed(t *testing.T) {
	res := runOnce(t)
	missing := []string{}
	for _, f := range aggregateRows(res.Fields) {
		if f.Proven == "" {
			missing = append(missing, f.Surface+" "+f.Key)
		}
	}
	if len(missing) != 0 {
		t.Fatalf("aggregate fields with no attribution: %v", missing)
	}
}

// TestTheFixtureMarksEveryRowUniquely is the fixture's own contract: a shared marker
// would make a reconstruction claim ambiguous.
func TestTheFixtureMarksEveryRowUniquely(t *testing.T) {
	fx := buildFixture()
	if !fx.markersUnique() {
		t.Fatal("two stored rows share a marker, so a published value could not be traced to one row")
	}
	if fx.rowTotal() == 0 || len(fx.Tenants) < 2 {
		t.Fatalf("fixture too thin to measure: %d tenants, %d rows", len(fx.Tenants), fx.rowTotal())
	}
}

// TestTheFixtureHoldsACellOfOne makes sure the small end of the tenant base is present:
// without a one-customer tenant there is nothing to reconstruct and nothing to price.
func TestTheFixtureHoldsACellOfOne(t *testing.T) {
	fx := buildFixture()
	for _, table := range personalTables {
		if !fx.hasTable(table) {
			t.Fatalf("the fixture does not populate %s, which an aggregate field maps onto", table)
		}
		if fx.minRows(table, "tenant") != 1 {
			t.Fatalf("no tenant holds exactly one %s row, so the exposure case is unmeasured", table)
		}
	}
}

// TestTheHarnessDetectsAPlantedExposure is the positive control: the money sum the ISP
// dashboard publishes is a single stored balance in the solo tenant, and the run has to
// say so. A harness that cannot find a planted exposure cannot be trusted with a null.
func TestTheHarnessDetectsAPlantedExposure(t *testing.T) {
	res := runOnce(t)
	var found *exposure
	for i := range res.Exps {
		e := &res.Exps[i]
		if e.Key == "outstanding" && e.Measure == "money" {
			found = e
		}
	}
	if found == nil {
		t.Fatal("the run lost the outstanding money sum")
	}
	if bucketOf(*found) != "value-exposed" {
		t.Fatalf("the planted exposure reads as %q, verdict %q", bucketOf(*found), found.Verdict)
	}
	if found.MinRows != 1 || found.Table != "isp_invoice" {
		t.Fatalf("planted exposure mis-attributed: table %q, smallest cell %d", found.Table, found.MinRows)
	}
}

// TestNoPersonalAggregateLeavesItsTenantToday states the shipped answer: every personal
// number is served to the tenant that owns the rows, so the reconstruction risk is about
// a future publication, not a current leak - and the check is only worth anything because
// the run scores aggregates at all.
func TestNoPersonalAggregateLeavesItsTenantToday(t *testing.T) {
	res := runOnce(t)
	if res.Totals.Aggregates == 0 {
		t.Fatal("no aggregates scored; the null result proves nothing")
	}
	leaked := []string{}
	for _, e := range res.Exps {
		if !e.Personal {
			continue
		}
		s, ok := scopeBySurface(res, e.Surface)
		if ok && s.Class == "platform" {
			leaked = append(leaked, e.Surface+" "+e.Key)
		}
	}
	if len(leaked) != 0 {
		t.Fatalf("a subscriber aggregate is served from a cross-tenant surface: %v", leaked)
	}
	if res.Totals.CrossTenant != 0 && len(leaked) == 0 {
		t.Fatalf("the cross-tenant counter disagrees with the per-field scan: %d", res.Totals.CrossTenant)
	}
}

// TestRatesBesideTheirDenominatorAreCounts pins the composition finding. A rate is not a
// softer statistic than the counts it divides, so the run has to read the division and
// name both operands - and a claim about the operands is only worth anything if each one
// is a field the same response serves, found again on the cited line.
func TestRatesBesideTheirDenominatorAreCounts(t *testing.T) {
	res := runOnce(t)
	if res.Totals.RateDisguise == 0 {
		t.Fatal("no rate was found beside its own operands; the composition claim is unmeasured")
	}
	served := map[string]map[string]bool{}
	for _, f := range res.Fields {
		if served[f.Surface] == nil {
			served[f.Surface] = map[string]bool{}
		}
		served[f.Surface][f.Key] = true
	}
	for _, e := range res.Exps {
		if e.Measure != "rate" {
			continue
		}
		resolved := strings.Contains(e.Denominator, " / ") &&
			!strings.Contains(e.Denominator, unresolvedOperand)
		if bucketOf(e) == "rate-disguise" != resolved {
			t.Fatalf("rate %s %s ratio %q bucketed %q", e.Surface, e.Key, e.Denominator, bucketOf(e))
		}
		if !resolved {
			continue
		}
		parts := strings.Split(e.Denominator, " / ")
		if len(parts) != 2 {
			t.Fatalf("rate %s %s has a ratio that is not two operands: %q", e.Surface, e.Key, e.Denominator)
		}
		src := lines(repoRoot(t), surfaceFile(res, e.Surface))
		if e.Line < 1 || e.Line > len(src) {
			t.Fatalf("rate %s %s cites line %d, which %s does not have", e.Surface, e.Key, e.Line, surfaceFile(res, e.Surface))
		}
		for _, operand := range parts {
			if !served[e.Surface][operand] {
				t.Fatalf("rate %s divides %q, which %s does not serve", e.Surface, operand, e.Surface)
			}
			goName := fieldGo(res, e.Surface, operand)
			if !hasDivisionInto(src[e.Line-1], e.GoName, goName, src) {
				t.Fatalf("rate %s %s does not divide %q on its own line: %q",
					e.Surface, e.Key, operand, src[e.Line-1])
			}
		}
	}
}

// surfaceFile is the handler file of one surveyed surface, which is where a rate's
// assignment is written.
func surfaceFile(res *Result, route string) string {
	for _, sp := range res.Surfaces {
		if sp.route == route {
			return sp.file
		}
	}
	return ""
}

// fieldGo is the Go identifier behind one served key.
func fieldGo(res *Result, surface, key string) string {
	for _, f := range res.Fields {
		if f.Surface == surface && f.Key == key {
			return f.Go
		}
	}
	return ""
}

// hasDivisionInto re-reads the product's own assignment and asks whether it is a division
// naming the field the study cites on one of its sides - directly, or under the local name
// a helper parameter carries it in under. A renamed alias breaks this gate, not the ratio.
func hasDivisionInto(ln, goName, operandGo string, src []string) bool {
	if goName == "" || operandGo == "" {
		return false
	}
	at := strings.Index(ln, "."+goName+" =")
	if at < 0 {
		return false
	}
	expr := ln[at+len(goName)+3:]
	slash := strings.Index(expr, "/")
	if slash < 0 {
		return false
	}
	for _, side := range []string{expr[:slash], expr[slash+1:]} {
		for _, tok := range identTokens(side) {
			if tok == operandGo || paramOrigin(src, tok) == operandGo {
				return true
			}
		}
	}
	return false
}

// TestTheFloorTradeOffIsMonotone keeps the suppression table honest: raising the floor
// cannot expose more fields and cannot suppress fewer.
func TestTheFloorTradeOffIsMonotone(t *testing.T) {
	res := runOnce(t)
	for i := 1; i < len(res.Floors); i++ {
		prev, next := res.Floors[i-1], res.Floors[i]
		if next.Exposed > prev.Exposed {
			t.Fatalf("floor %d exposes more fields (%d) than floor %d (%d)", next.K, next.Exposed, prev.K, prev.Exposed)
		}
		if next.Suppressed < prev.Suppressed {
			t.Fatalf("floor %d suppresses fewer fields (%d) than floor %d (%d)", next.K, next.Suppressed, prev.K, prev.Suppressed)
		}
		if next.K == 2 && next.Exposed != 0 {
			t.Fatalf("floor 2 still reconstructs %d fields, so a single stored row survived it", next.Exposed)
		}
	}
	if res.Floors[0].Exposed == 0 {
		t.Fatal("no field reconstructs at floor 1; the fixture priced nothing")
	}
}

// TestEveryExposureNamesItsVerdict keeps the memo's table readable: no row may be an
// empty cell.
func TestEveryExposureNamesItsVerdict(t *testing.T) {
	res := runOnce(t)
	for _, e := range res.Exps {
		if e.Verdict == "" || e.Cell == "" {
			t.Fatalf("exposure %s %s has no verdict or no cell rule", e.Surface, e.Key)
		}
	}
}

// TestGovernanceCountsWhatItClaims is the ratchet: the permission mechanisms the criterion
// needs are absent, and the test fails if the tree grows one without the memo changing.
func TestGovernanceCountsWhatItClaims(t *testing.T) {
	res := runOnce(t)
	if res.Totals.StatsConfigKeys != 0 {
		t.Fatalf("the config registry now carries %d statistics or sharing keys; the memo must be re-read", res.Totals.StatsConfigKeys)
	}
	if res.Totals.ConfigKeys == 0 {
		t.Fatal("the config registry parsed to zero keys; the absence claim is unmeasured")
	}
	gov := map[string]governanceRow{}
	for _, g := range res.Gov {
		gov[g.ID] = g
	}
	for _, id := range []string{"P01", "P02", "P03", "P04", "P06"} {
		if gov[id].Count != 0 {
			t.Fatalf("%s (%s) counts %d lines: this round's absence claim needs re-writing", id, gov[id].Mechanism, gov[id].Count)
		}
	}
	if gov["P05"].Count == 0 {
		t.Fatal("the shipped opt-in precedent vanished, so the governance recommendation has no shape to copy")
	}
}

// TestTheContractRowMarksItsResearchNote keeps the P06 exclusion honest. The words a research
// round appends to the TR-F033 row report what a response already does and cannot grant a
// permission, so they sit behind a marker and only the wording before it is counted - which
// only holds while the marker really is a study citation and the contract stays silent.
func TestTheContractRowMarksItsResearchNote(t *testing.T) {
	marked := 0
	rows := 0
	for _, ln := range lines(repoRoot(t), "docs/feature-checklist.md") {
		if !strings.HasPrefix(ln, "|") || !strings.Contains(ln, "TR-F033") {
			continue
		}
		rows++
		i := strings.Index(ln, researchNoteMarker)
		if i < 0 {
			i = len(ln)
		} else {
			marked++
			if !strings.HasPrefix(ln[i:], researchNoteMarker+"FUT-020): per-tenant operational statistics are measured in [per-tenant-operational-statistics-study-2026-10-29.md]") {
				t.Fatalf("what follows a research-note marker is not this round's study citation: %s", ln[i:])
			}
		}
		for _, term := range []string{"statistic", "aggregate", "benchmark", "publish"} {
			if n := strings.Count(strings.ToLower(ln[:i]), term); n != 0 {
				t.Fatalf("the contract wording before the marker names %q %d times: that is a scope change, not a finding, and P06 must stop reading zero", term, n)
			}
		}
	}
	if rows == 0 {
		t.Fatal("no TR-F033 row in docs/feature-checklist.md, so P06 measured nothing")
	}
	if marked != 1 {
		t.Fatalf("%d of %d TR-F033 rows carry a research-note marker; the exclusion must apply to exactly one", marked, rows)
	}
}

// TestThePrecedentShapeExists names the three parts a publication switch would copy.
func TestThePrecedentShapeExists(t *testing.T) {
	res := runOnce(t)
	p := res.Precedent
	if p.Enable == "" || p.AckField == "" || p.Scope == "" || p.Line == 0 {
		t.Fatalf("the consent precedent is incomplete: %+v", p)
	}
}

// TestNoSuppressionVocabularyReachesAStatisticsFile keeps the generic words honest: the
// product does use "suppress" - for a RADIUS guard action and for seeding - and none of
// those uses sits in a file that serves a number.
func TestNoSuppressionVocabularyReachesAStatisticsFile(t *testing.T) {
	res := runOnce(t)
	generic := 0
	for _, v := range res.Vocab {
		if v.Count > 0 {
			generic++
		}
		if v.Stats != 0 {
			t.Fatalf("%q appears %d times inside a statistics file; the absence row P02/P03 has to be re-read", v.Term, v.Stats)
		}
	}
	if generic == 0 {
		t.Fatal("no vocabulary term was found anywhere; the scan is matching nothing")
	}
}

// TestEveryScopeRowCitesWhatItsClassClaims checks the class column against the line it
// cites: a surface labeled fallback must point at a line that substitutes the default
// tenant, and one labeled explicit at a written tenant comparison. A ranking bug would
// otherwise relabel a surface without changing any counted number.
func TestEveryScopeRowCitesWhatItsClassClaims(t *testing.T) {
	root := repoRoot(t)
	res := runOnce(t)
	want := map[string]string{
		"fallback": "DefaultTenantID",
		"explicit": "tenant",
		"platform": "context.Background()",
		"callback": "GetDB(c)",
	}
	for _, s := range res.Scope {
		if s.Class == "unattributed" {
			t.Fatalf("surface %s reaches the database in no way the study can name", s.Surface)
		}
		rel, lineStr, ok := strings.Cut(s.Evidence, ":")
		if !ok {
			t.Fatalf("scope row %s cites %q without a line", s.Surface, s.Evidence)
		}
		src := lines(root, rel)
		n := 0
		if _, err := fmt.Sscanf(lineStr, "%d", &n); err != nil || n <= 0 || n > len(src) {
			t.Fatalf("scope row %s cites an unusable line %q", s.Surface, s.Evidence)
		}
		ln := src[n-1]
		token := want[s.Class]
		if s.Class == "explicit" {
			if !strings.Contains(ln, "tenant") {
				t.Fatalf("explicit row %s cites %q, which names no tenant", s.Surface, ln)
			}
			continue
		}
		if !strings.Contains(ln, token) {
			t.Fatalf("%s row %s cites %q, which does not carry %q", s.Class, s.Surface, ln, token)
		}
	}
}

// TestTheFallbackSitesAreCited requires every counted site to be a real file and line.
func TestTheFallbackSitesAreCited(t *testing.T) {
	res := runOnce(t)
	if len(res.Fallback) == 0 {
		t.Fatal("no default-tenant fallback found; the misattribution claim is unmeasured")
	}
	for _, site := range res.Fallback {
		file, line, ok := strings.Cut(site, ":")
		if !ok || firstLine(repoRoot(t), file, "DefaultTenantID") == 0 || line == "" {
			t.Fatalf("fallback site %s does not cite a line that reads the default tenant", site)
		}
	}
}

// TestTheDashboardScopeGapIsReal proves the untested-surface finding is an absence in the
// tree rather than an absence in the scan.
func TestTheDashboardScopeGapIsReal(t *testing.T) {
	root := repoRoot(t)
	rel := "internal/adminapi/dashboard_test.go"
	src := lines(root, rel)
	if len(src) == 0 {
		t.Fatalf("%s is gone; the finding has to be re-read", rel)
	}
	res := runOnce(t)
	if res.Totals.DashboardTenantAssertions != 0 {
		t.Fatalf("the dashboard test now names a tenant %d times; the gap row is closed", res.Totals.DashboardTenantAssertions)
	}
	if n := countMatches(root, "internal/adminapi/noc_impact_test.go", "Tenant"); n == 0 {
		t.Fatal("the impact test's tenant assertions vanished; the comparison is unmeasured")
	}
}

// TestTheTotalsPartition keeps the six buckets a partition of the scored aggregates, so a
// number cannot be dropped or double counted.
func TestTheTotalsPartition(t *testing.T) {
	res := runOnce(t)
	tot := res.Totals
	sum := tot.Unmodeled + tot.Impersonal + tot.Safe + tot.ValueExposed + tot.Membership + tot.RateDisguise
	if sum != tot.Aggregates || len(res.Exps) != tot.Aggregates {
		t.Fatalf("buckets %d do not partition %d aggregates over %d exposures", sum, tot.Aggregates, len(res.Exps))
	}
	if tot.Fields < tot.Aggregates || tot.Aggregates == 0 {
		t.Fatalf("field counts make no sense: %d fields, %d aggregates", tot.Fields, tot.Aggregates)
	}
	scoped := tot.ScopeCallback + tot.ScopeExplicit + tot.ScopeFallback + tot.ScopePlatform + tot.ScopeUnknown
	if scoped != tot.Surfaces {
		t.Fatalf("scope classes %d do not cover %d surfaces", scoped, tot.Surfaces)
	}
}

// TestTheDigestSurvivesATickingClock proves the digest is a function of the tree alone.
func TestTheDigestSurvivesATickingClock(t *testing.T) {
	root := repoRoot(t)
	first, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1100 * time.Millisecond)
	second, err := Run(root)
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest {
		t.Fatalf("the digest moved on an idle tree: %s -> %s", first.Digest, second.Digest)
	}
}

// TestEveryStatementIsConfirmed fails on drift in the pinned statements.
func TestEveryStatementIsConfirmed(t *testing.T) {
	res := runOnce(t)
	for _, c := range res.Claims {
		if c.Status != "confirmed" {
			t.Fatalf("statement %s is %s (%s %q)", c.ID, c.Status, c.File, c.Symbol)
		}
	}
	if len(res.Claims) != len(claimSpecs()) {
		t.Fatalf("claim table drifted: %d vs %d", len(res.Claims), len(claimSpecs()))
	}
}

// TestEveryGapIsANoGo keeps the limit register in the form the memo prints it.
func TestEveryGapIsANoGo(t *testing.T) {
	res := runOnce(t)
	if len(res.Gaps) < 6 {
		t.Fatalf("the round closed too few limits to be honest: %d", len(res.Gaps))
	}
	for _, g := range res.Gaps {
		if g.Status != "no-go" || g.Note == "" {
			t.Fatalf("gap %s is %q without a reason", g.ID, g.Status)
		}
	}
}

// TestReportMarkdownDump prints the memo's tables. It is off by default because the run is
// only evidence when the numbers are copied, not pasted into a log.
func TestReportMarkdownDump(t *testing.T) {
	if os.Getenv("OPSTATS_DUMP") != "1" {
		t.Skip("set OPSTATS_DUMP=1 to print the measured tables")
	}
	res := runOnce(t)
	fmt.Println(Markdown(res))
	fmt.Println("Digest: `" + res.Digest + "`")
}
