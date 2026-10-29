// Package opstats measures the statistics FUT-020 asks about - per-tenant operational
// aggregates kept private, publishable only where governance is explicit - against the
// surfaces the console already serves.
//
// The study is read-only: it parses the product's own files, holds the parsed tables in
// memory, and writes nothing. Nothing in cmd/ or internal/ imports it, and it imports
// only the standard library.
//
// # What is measured
//
// Three quantities come out of the run:
//
//   - the aggregate census. Every number a statistics response serves, with the
//     statement that produces it, the table it reads, and the cell rule that says how
//     many stored rows one published value covers.
//   - the reconstructability. A deterministic fixture of six tenants, every stored row
//     carrying a value no other row shares, run through each field's aggregate verb. A
//     value over one row is that row's own value, and a count of one over a subscriber
//     table says the tenant has exactly one subscriber.
//   - the permission. What governance exists for letting an aggregate leave its tenant:
//     a config switch, a suppression rule, a redactor reaching a number, the shipped
//     opt-in precedent, and the tenancy contract's own words.
package opstats

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Result is one measured run: the census, the fixture, the exposures, the permission
// table and the statements that pin them, plus the digest that ties them together.
type Result struct {
	Surfaces   []surfaceSpec
	Fields     []fieldRow
	Exps       []exposure
	Floors     []floorRow
	Scope      []scopeRow
	Fixture    *Fixture
	Gov        []governanceRow
	Vocab      []vocabHit
	Fallback   []string
	Precedent  optInPrecedent
	ConfigKeys int
	StatsKeys  int
	Broken     []string
	Claims     []Claim
	Gaps       []Gap
	Imports    []string
	Totals     Totals
	Digest     string
}

// Totals is the run's headline numbers.
type Totals struct {
	Surfaces   int
	Fields     int
	Aggregates int
	// Partition of the aggregate rows, by what the fixture says about them.
	Unmodeled    int
	Impersonal   int
	Safe         int
	ValueExposed int
	Membership   int
	RateDisguise int
	// Personal counts the aggregates that describe a subscriber's or an invoice's row.
	Personal int
	// Distinct counts the aggregates whose per-tenant fixture value is unique, so a
	// published series would identify the tenant by its own number.
	Distinct   int
	CellsOfOne int
	// Scope classes, one per surface.
	ScopeCallback int
	ScopeExplicit int
	ScopeFallback int
	ScopePlatform int
	ScopeUnknown  int
	// CrossTenant counts the aggregates served from a surface that is deliberately not
	// tenant-scoped - the only class that publishes outside the owning tenant today.
	CrossTenant int
	// Floor rows: exposures and coverage at each suppression threshold.
	FloorK2Exposed            int
	FloorK2Blank              int
	ConfigKeys                int
	StatsConfigKeys           int
	RedactReach               int
	FallbackSites             int
	DashboardTenantAssertions int
	PrecedentFields           int
	FixtureTenants            int
	FixtureRows               int
	FixtureTables             int
	BrokenSpecs               int
	ClaimsOK                  int
	ClaimsBroken              int
	ClaimsVacated             int
	Gaps                      int
	Imports                   int
}

// Run performs the whole study against a repository root.
func Run(root string) (*Result, error) {
	res := &Result{}
	res.Surfaces = surfaceSpecs()
	rows, broken := readSurfaces(root)
	res.Broken = broken
	attribute(root, rows)
	res.Fields = rows

	fx := buildFixture()
	res.Fixture = fx

	exps := measure(rows, fx)
	res.Exps = exps
	res.Floors = priceFloors(exps)

	scope := readScoping(root)
	res.Scope = scope
	scoped := map[string]scopeRow{}
	for _, s := range scope {
		scoped[s.Surface] = s
	}
	res.Totals.CrossTenant = len(crossTenantServed(exps, scoped))

	files := map[string]bool{}
	for _, sp := range res.Surfaces {
		files[sp.file] = true
	}
	surfaceFiles := sortedTables(files)
	gov, vocab, fallbacks := buildGovernance(root, surfaceFiles)
	res.Gov = gov
	res.Vocab = vocab
	res.Fallback = fallbacks
	res.Precedent = readOptInPrecedent(root)
	keys, _ := readConfigKeys(root)
	res.ConfigKeys = len(keys)
	res.StatsKeys = len(matchingKeys(keys))

	res.Claims = evalClaims(root)
	res.Gaps = gaps()
	res.Imports = packageImports(root, ownPackageDir(root))
	res.Totals = totals(res)
	res.Digest = digest(res)
	return res, nil
}

// bucketOf puts one aggregate into exactly one class, so the totals add up and a field
// cannot be counted twice or dropped.
//
// A rate is only a count in disguise when the count it divides is served in the same
// response; a bare ratio the study cannot attribute to a table is unmodeled, and says so.
func bucketOf(e exposure) string {
	switch {
	case e.Measure == "rate" && strings.Contains(e.Denominator, " / ") &&
		!strings.Contains(e.Denominator, unresolvedOperand):
		return "rate-disguise"
	case e.Table == "" || e.MinRows == 0:
		return "unmodeled"
	case !e.Personal:
		return "impersonal"
	case e.MinRows == 1 && (e.Measure == "money" || e.Measure == "bytes"):
		return "value-exposed"
	case e.MinRows == 1:
		return "membership"
	default:
		return "safe"
	}
}

func totals(res *Result) Totals {
	t := Totals{
		Surfaces:    len(res.Surfaces),
		Fields:      len(res.Fields),
		Aggregates:  len(aggregateRows(res.Fields)),
		BrokenSpecs: len(res.Broken),
	}
	for _, e := range res.Exps {
		switch bucketOf(e) {
		case "unmodeled":
			t.Unmodeled++
		case "impersonal":
			t.Impersonal++
		case "safe":
			t.Safe++
		case "value-exposed":
			t.ValueExposed++
		case "membership":
			t.Membership++
		case "rate-disguise":
			t.RateDisguise++
		}
		if e.Personal {
			t.Personal++
		}
		if e.Fingerprint {
			t.Distinct++
		}
		if e.MinRows == 1 {
			t.CellsOfOne++
		}
	}
	for _, s := range res.Scope {
		switch s.Class {
		case "callback":
			t.ScopeCallback++
		case "explicit":
			t.ScopeExplicit++
		case "fallback":
			t.ScopeFallback++
		case "platform":
			t.ScopePlatform++
		default:
			t.ScopeUnknown++
		}
	}
	for _, f := range res.Floors {
		if f.K == 2 {
			t.FloorK2Exposed = f.Exposed
			t.FloorK2Blank = f.BlankAll
		}
	}
	for _, g := range res.Gov {
		switch g.ID {
		case "P04":
			t.RedactReach = g.Count
		case "P07":
			t.FallbackSites = g.Count
		case "P08":
			t.DashboardTenantAssertions = g.Count
		}
	}
	t.ConfigKeys = res.ConfigKeys
	t.StatsConfigKeys = res.StatsKeys
	t.PrecedentFields = res.Precedent.Fields
	t.FixtureTenants = len(res.Fixture.Tenants)
	t.FixtureRows = res.Fixture.rowTotal()
	t.FixtureTables = len(res.Fixture.tablesUsed())
	t.ClaimsOK, t.ClaimsBroken, t.ClaimsVacated = statusCounts(res.Claims)
	t.Gaps = len(res.Gaps)
	t.Imports = len(res.Imports)
	return t
}

// digest locks the tables the memo prints. It reads no clock: a run over the same tree
// returns the same string, and any change to a served field, a fixture population or a
// claim line changes it.
func digest(res *Result) string {
	h := sha256.New()
	write := func(prefix string, rows ...string) {
		for _, r := range rows {
			_, _ = fmt.Fprintf(h, "%s\t%s\n", prefix, r)
		}
	}
	for _, sp := range res.Surfaces {
		write("surface", strings.Join([]string{sp.route, sp.handler, sp.file, sp.typeName, sp.audience}, "\t"))
	}
	for _, f := range res.Fields {
		write("field", strings.Join([]string{f.Surface, f.Key, f.Type, f.Shape, f.Measure, f.Verb, f.Table, f.Cell, f.Proven, itoa(f.Line)}, "\t"))
	}
	for _, e := range res.Exps {
		write("exposure", strings.Join([]string{e.Surface, e.Key, bucketOf(e), e.Verb, e.Table, e.Cell,
			itoa(e.MinRows), itoa(e.Singleton), itoa(e.Blank), yesNo(e.Fingerprint), e.Denominator, e.Verdict}, "\t"))
	}
	for _, fl := range res.Floors {
		write("floor", strings.Join([]string{itoa(fl.K), itoa(fl.Exposed), itoa(fl.Suppressed), itoa(fl.BlankAll), itoa(fl.Fields)}, "\t"))
	}
	for _, s := range res.Scope {
		write("scope", strings.Join([]string{s.Surface, s.Class, s.Evidence, itoa(s.ModelQueries), itoa(s.RawQueries)}, "\t"))
	}
	for _, t := range res.Fixture.tablesUsed() {
		for _, tn := range res.Fixture.Tenants {
			write("fixture", strings.Join([]string{tn, t, itoa(res.Fixture.count(tn, t))}, "\t"))
		}
	}
	for _, g := range res.Gov {
		write("gov", strings.Join([]string{g.ID, itoa(g.Count), itoa(g.Files)}, "\t"))
	}
	for _, v := range res.Vocab {
		write("vocab", strings.Join([]string{v.Term, itoa(v.Count), itoa(v.Files), v.Where, itoa(v.Stats), v.Reading}, "\t"))
	}
	for _, f := range res.Fallback {
		write("fallback", f)
	}
	for _, c := range res.Claims {
		write("claim", strings.Join([]string{c.ID, c.Status, c.File, c.Symbol, itoa(c.Line)}, "\t"))
	}
	for _, g := range res.Gaps {
		write("gap", strings.Join([]string{g.ID, g.Need, g.Status}, "\t"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// scopeClasses is the sorted scope table for the memo.
func scopeBySurface(res *Result, route string) (scopeRow, bool) {
	for _, s := range res.Scope {
		if s.Surface == route {
			return s, true
		}
	}
	return scopeRow{}, false
}
