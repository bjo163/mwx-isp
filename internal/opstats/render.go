package opstats

import (
	"fmt"
	"strings"
)

// Markdown renders the measured tables. Every number the memo quotes is copied from
// here, so the memo and the run cannot disagree.
func Markdown(res *Result) string {
	var b strings.Builder
	sectionSurfaces(&b, res)
	sectionFixture(&b, res)
	sectionCensus(&b, res)
	sectionExposures(&b, res)
	sectionFloors(&b, res)
	sectionGovernance(&b, res)
	sectionVocab(&b, res)
	sectionFallbacks(&b, res)
	sectionClaims(&b, res)
	sectionGaps(&b, res)
	sectionTotals(&b, res)
	return b.String()
}

func header(b *strings.Builder, title, blurb string) {
	fmt.Fprintf(b, "## %s\n\n%s\n\n", title, blurb)
}

func row(b *strings.Builder, cells ...string) {
	b.WriteString("| ")
	b.WriteString(strings.Join(cells, " | "))
	b.WriteString(" |\n")
}

func sectionSurfaces(b *strings.Builder, res *Result) {
	header(b, "Statistics surfaces",
		"Every aggregate surface the console registers, how its query reaches the database, and how many statements go through a domain model (where the tenant callback can reach) versus a raw table (where it cannot).")
	b.WriteString("| surface | audience | handler | served type | scope | evidence | model queries | raw queries |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | ---: |\n")
	for _, sp := range res.Surfaces {
		s, ok := scopeBySurface(res, sp.route)
		class, ev, mq, rq := "unattributed", "-", "0", "0"
		if ok {
			class, ev, mq, rq = s.Class, orNone(s.Evidence), itoa(s.ModelQueries), itoa(s.RawQueries)
		}
		row(b, sp.route, sp.audience, sp.handler, sp.typeName, class, ev, mq, rq)
	}
	fmt.Fprintf(b, "\n%d surfaces, %d scope classes: %d callback, %d explicit, %d fallback, %d platform, %d unattributed.\n\n",
		res.Totals.Surfaces, 5, res.Totals.ScopeCallback, res.Totals.ScopeExplicit,
		res.Totals.ScopeFallback, res.Totals.ScopePlatform, res.Totals.ScopeUnknown)
}

func sectionFixture(b *strings.Builder, res *Result) {
	header(b, "The fixture",
		"The synthetic installation the criterion names: six tenants, every stored row carrying a value no other row shares. These are the populations a statistic is measured over, not real customers.")
	b.WriteString("| tenant | " + strings.Join(res.Fixture.tablesUsed(), " | ") + " |\n")
	b.WriteString("| --- ")
	for range res.Fixture.tablesUsed() {
		b.WriteString("| ---: ")
	}
	b.WriteString("|\n")
	for _, t := range res.Fixture.Tenants {
		cells := []string{t}
		for _, table := range res.Fixture.tablesUsed() {
			cells = append(cells, itoa(res.Fixture.count(t, table)))
		}
		row(b, cells...)
	}
	fmt.Fprintf(b, "\n%d tenants, %d stored rows across %d tables, markers unique: %s.\n\n",
		res.Totals.FixtureTenants, res.Totals.FixtureRows, res.Totals.FixtureTables,
		yesNo(res.Fixture.markersUnique()))
}

func sectionCensus(b *strings.Builder, res *Result) {
	header(b, "Aggregate census",
		"Every field a statistics response serves, with the statement that produces it: the verb the handler writes, the table the verb reads, and the cell rule that says how many stored rows one published value covers.")
	b.WriteString("| surface | field | type | shape | measure | verb | table | cell | attribution | line |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- | --- | --- | ---: |\n")
	for _, f := range res.Fields {
		row(b, f.Surface, f.Key, f.Type, f.Shape, f.Measure, orNone(f.Verb), orNone(f.Table), f.Cell, orNone(f.Proven), itoa(f.Line))
	}
	fmt.Fprintf(b, "\n%d served fields, %d of them aggregates this study scores.\n\n",
		res.Totals.Fields, res.Totals.Aggregates)
}

func sectionExposures(b *strings.Builder, res *Result) {
	header(b, "Reconstructability",
		"The fixture run through each aggregate verb. A cell of one stored row is the case that matters: the published number is then that row's own value, or a statement that the tenant has exactly one.")
	b.WriteString("| surface | field | class | verb | table | cell | smallest cell | tenants with a cell of one | tenants blanked at floor 2 | verdict |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | ---: | ---: | ---: | --- |\n")
	for _, e := range res.Exps {
		row(b, e.Surface, e.Key, bucketOf(e), orNone(e.Verb), orNone(e.Table), e.Cell,
			itoa(e.MinRows), itoa(e.Singleton), itoa(e.Blank), e.Verdict)
	}
	fmt.Fprintf(b, "\n%d aggregates scored: %d publish a stored value verbatim, %d say the tenant has exactly one, "+
		"%d are a count in disguise, %d cover a thing the operator owns, %d stay above a cell of one, %d have no source table attributed.\n\n",
		res.Totals.Aggregates, res.Totals.ValueExposed, res.Totals.Membership, res.Totals.RateDisguise,
		res.Totals.Impersonal, res.Totals.Safe, res.Totals.Unmodeled)
}

func sectionFloors(b *strings.Builder, res *Result) {
	header(b, "Suppression floors priced",
		"How high the cell floor has to be before no fixture value survives, and what that costs the tenants a benchmark is meant to describe.")
	b.WriteString("| floor | fields still reconstructing | fields suppressed somewhere | fields blank for at least one tenant | fields scored |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
	for _, f := range res.Floors {
		row(b, itoa(f.K), itoa(f.Exposed), itoa(f.Suppressed), itoa(f.BlankAll), itoa(f.Fields))
	}
	b.WriteString("\nA floor that removes an exposure also withholds the tenant that caused it, which is why the smallest tenants in the fixture - the RT/RW Net end - are the ones a benchmark would lose.\n\n")
}

func sectionGovernance(b *strings.Builder, res *Result) {
	header(b, "Governance",
		"The permission half of the criterion. Each row is a mechanism a publication would need, counted in the tree after comments are blanked.")
	b.WriteString("| id | mechanism | lines | files | where | reading |\n")
	b.WriteString("| --- | --- | ---: | ---: | --- | --- |\n")
	for _, g := range res.Gov {
		row(b, g.ID, g.Mechanism, itoa(g.Count), itoa(g.Files), g.Where, g.Reading)
	}
	fmt.Fprintf(b, "\n%d config keys in the registry, %d of them about statistics, benchmarks, aggregates, anonymisation or sharing.\n\n",
		res.Totals.ConfigKeys, res.Totals.StatsConfigKeys)
}

func sectionVocab(b *strings.Builder, res *Result) {
	header(b, "Vocabulary searched",
		"The identifiers a suppression or anonymisation rule would be named with, and where the tree actually uses them. The last column counts the matches inside the files that serve statistics - a word used by the RADIUS guard is not a rule about a cell.")
	b.WriteString("| term | lines | files | first | in statistics files | searched for |\n| --- | ---: | ---: | --- | ---: | --- |\n")
	for _, v := range res.Vocab {
		row(b, v.Term, itoa(v.Count), itoa(v.Files), orNone(v.Where), itoa(v.Stats), v.Reading)
	}
	b.WriteString("\n")
}

func sectionFallbacks(b *strings.Builder, res *Result) {
	header(b, "Default-tenant fallback sites",
		"Code that reads a tenant id and substitutes the default tenant when no context is present. A background aggregation job has no request context, so this is where one tenant's number can be filed under another.")
	if len(res.Fallback) == 0 {
		b.WriteString("none found\n\n")
		return
	}
	b.WriteString("| file | line |\n| --- | ---: |\n")
	for _, f := range res.Fallback {
		file, line, _ := strings.Cut(f, ":")
		row(b, file, line)
	}
	b.WriteString("\n")
}

func sectionClaims(b *strings.Builder, res *Result) {
	header(b, "Pinned statements",
		"Each statement is checked against a code line of a named file after comments are blanked. Drift is reported, never dropped.")
	b.WriteString("| id | statement | file | line | status |\n| --- | --- | --- | ---: | --- |\n")
	for _, c := range res.Claims {
		row(b, c.ID, c.Text, c.File, itoa(c.Line), c.Status)
	}
	fmt.Fprintf(b, "\n%d of %d confirmed, %d broken, %d vacated.\n\n",
		res.Totals.ClaimsOK, len(res.Claims), res.Totals.ClaimsBroken, res.Totals.ClaimsVacated)
}

func sectionGaps(b *strings.Builder, res *Result) {
	header(b, "What this round cannot answer",
		"Each row needs something the repository does not have. The status is the deliverable: an explicit no-go, with the reason.")
	b.WriteString("| id | needs | status | why |\n| --- | --- | --- | --- |\n")
	for _, g := range res.Gaps {
		row(b, g.ID, g.Need, g.Status, g.Note)
	}
	b.WriteString("\n")
}

func sectionTotals(b *strings.Builder, res *Result) {
	header(b, "Totals",
		"The run's headline numbers, all of them counts of the tables above.")
	b.WriteString("| measure | value |\n| --- | --- |\n")
	pairs := [][2]string{
		{"surfaces surveyed", itoa(res.Totals.Surfaces)},
		{"surface specs that did not resolve", itoa(res.Totals.BrokenSpecs)},
		{"served fields", itoa(res.Totals.Fields)},
		{"aggregate fields scored", itoa(res.Totals.Aggregates)},
		{"publishing a stored value verbatim", itoa(res.Totals.ValueExposed)},
		{"saying the tenant has exactly one", itoa(res.Totals.Membership)},
		{"rates beside their own denominator", itoa(res.Totals.RateDisguise)},
		{"aggregates over a thing the operator owns", itoa(res.Totals.Impersonal)},
		{"aggregates above a cell of one", itoa(res.Totals.Safe)},
		{"aggregates with no source table attributed", itoa(res.Totals.Unmodeled)},
		{"aggregates over a personal table", itoa(res.Totals.Personal)},
		{"fields whose per-tenant fixture value is distinct", itoa(res.Totals.Distinct)},
		{"fields with a cell of one in the fixture", itoa(res.Totals.CellsOfOne)},
		{"cross-tenant aggregates served today", itoa(res.Totals.CrossTenant)},
		{"surfaces scoped by the tenant callback", itoa(res.Totals.ScopeCallback)},
		{"surfaces scoped by a written comparison", itoa(res.Totals.ScopeExplicit)},
		{"surfaces with a default-tenant fallback", itoa(res.Totals.ScopeFallback)},
		{"surfaces meant to cross tenants", itoa(res.Totals.ScopePlatform)},
		{"config keys in the registry", itoa(res.Totals.ConfigKeys)},
		{"config keys about statistics or sharing", itoa(res.Totals.StatsConfigKeys)},
		{"statistics files calling the shipped redactor", itoa(res.Totals.RedactReach)},
		{"default-tenant fallback sites", itoa(res.Totals.FallbackSites)},
		{"tenant assertions in the dashboard test", itoa(res.Totals.DashboardTenantAssertions)},
		{"fields in the shipped opt-in precedent", itoa(res.Totals.PrecedentFields)},
		{"fixture tenants", itoa(res.Totals.FixtureTenants)},
		{"fixture stored rows", itoa(res.Totals.FixtureRows)},
		{"exposures left at floor 2", itoa(res.Totals.FloorK2Exposed)},
		{"statements pinned", itoa(len(res.Claims))},
		{"gaps open", itoa(res.Totals.Gaps)},
		{"packages this study imports", itoa(res.Totals.Imports)},
	}
	for _, p := range pairs {
		row(b, p[0], p[1])
	}
	b.WriteString("\n")
}
