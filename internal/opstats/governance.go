package opstats

import (
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

// The second half of the criterion is not about numbers, it is about permission: an
// aggregate may be published only where governance is explicit. This file counts the
// permission surfaces that exist, so the answer is a measurement rather than an
// opinion.

// governanceRow is one permission mechanism, whether it exists, and where the search
// looked.
type governanceRow struct {
	ID        string
	Mechanism string
	// Count is the number of code or config lines that carry it.
	Count int
	Files int
	Where string
	// Reading states what the count means for publication.
	Reading string
}

// configKey is one entry of the shipped config registry.
type configKey struct {
	Key   string
	Group string
	Type  string
}

type configSchema struct {
	Schemas []configKey `json:"schemas"`
}

// readConfigKeys parses the registry the system config page renders. The registry is
// the only place an operator-editable switch can live, so a statistics opt-in that is
// not in this file cannot be turned on.
func readConfigKeys(root string) ([]configKey, string) {
	rel := "internal/app/config_schemas.json"
	src, ok := readFile(filepath.Join(root, rel))
	if !ok {
		return nil, rel
	}
	var parsed configSchema
	if err := json.Unmarshal([]byte(src), &parsed); err != nil {
		return nil, rel
	}
	out := append([]configKey(nil), parsed.Schemas...)
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, rel
}

// statsKeyWords are the words a governance switch for aggregate publication would use.
var statsKeyWords = []string{"stat", "benchmark", "aggregat", "anonym", "publish", "share", "min_cell", "cell_floor"}

// matchingKeys returns the registry keys that carry one of those words.
func matchingKeys(keys []configKey) []configKey {
	var out []configKey
	for _, k := range keys {
		lower := strings.ToLower(k.Key)
		for _, w := range statsKeyWords {
			if strings.Contains(lower, w) {
				out = append(out, k)
				break
			}
		}
	}
	return out
}

// vocabTerms are the identifiers a suppression or anonymisation rule would be named with.
// The generic ones are searched and reported too, even though they match unrelated product
// code, so the reader can see which words were examined rather than which were chosen.
var vocabTerms = []struct {
	term    string
	reading string
}{
	{"kAnonymity", "a k-anonymity constant would set the smallest publishable cell"},
	{"minCell", "a minimum cell size would withhold a cell that covers too few rows"},
	{"CellFloor", "a cell floor is the name a suppression rule for a statistic would carry"},
	{"SuppressCell", "a named suppression of one cell is the mechanism the criterion needs"},
	{"anonymiz", "an anonymiser would decide what may leave the tenant"},
	{"suppress", "the generic word, which the tree uses for a RADIUS guard action and for seeding, not for a statistic"},
	{"Pseudonymize", "the shipped pseudonymiser, which is a log and support-bundle tool"},
	{"RiskAcknowledgedAt", "a stored acknowledgement, the governance shape the notification settings already use"},
}

type vocabHit struct {
	Term  string
	Count int
	Files int
	Where string
	// Reading states what the term would have meant for publication.
	Reading string
	// Stats counts the matches inside the files that serve statistics, which is how a
	// generic word elsewhere in the product is kept out of the absence claim.
	Stats int
}

// scanVocab walks the Go tree for each term after comments are blanked, so a proposal
// mentioned in a doc comment cannot be mistaken for a mechanism.
func scanVocab(root string, files []string, term string) vocabHit {
	v := vocabHit{Term: term}
	seen := map[string]bool{}
	for _, rel := range files {
		n := countMatches(root, rel, term)
		if n == 0 {
			continue
		}
		v.Count += n
		v.Files++
		if v.Where == "" {
			v.Where = rel + ":" + itoa(firstLine(root, rel, term))
		}
		seen[rel] = true
	}
	return v
}

// redactReach counts the shipped privacy code inside the files that serve statistics.
func redactReach(root string, files []string) int {
	total := 0
	for _, rel := range files {
		total += countMatches(root, rel, "redact.")
	}
	return total
}

// fallbackSites lists the code lines that read a tenant id and fall back to the default
// tenant when none is in the context. A background aggregation job has no request
// context, so this is where a number computed for one tenant could be filed under
// another.
func fallbackSites(root string, files []string) []string {
	var out []string
	for _, rel := range files {
		for _, ln := range matchLines(root, rel, "DefaultTenantID") {
			out = append(out, rel+":"+itoa(ln))
		}
	}
	sort.Strings(out)
	return out
}

// scopeTests counts the tenant-scope assertions in the test files beside each statistics
// handler, which says what CI actually guards.
func scopeTests(root string, rel string) int {
	if rel == "" {
		return 0
	}
	return countMatches(root, rel, "tenant")
}

// optInPrecedent reads the one shipped per-tenant consent shape: the notification
// settings row, which carries an enable flag, a stored risk acknowledgement and a
// tenant-scoped unique index.
type optInPrecedent struct {
	Type     string
	File     string
	Fields   int
	Enable   string
	AckField string
	Scope    string
	Line     int
}

func readOptInPrecedent(root string) optInPrecedent {
	rel := "internal/domain/network_monitor.go"
	fields, ok := structFields(root, rel, "NotificationSettings")
	p := optInPrecedent{Type: "NotificationSettings", File: rel}
	if !ok {
		return p
	}
	p.Fields = len(fields)
	for _, f := range fields {
		switch f.JSON {
		case "risk_acknowledged_at":
			p.AckField = f.Name
		case "whatsapp_enabled":
			p.Enable = f.Name
		}
		if f.Name == "TenantID" {
			p.Scope = f.Name
		}
	}
	p.Line = firstLine(root, rel, "type NotificationSettings struct")
	return p
}

// researchNoteMarker separates the words of the TR-F033 contract from the measured findings
// a research round appends to the same row. A finding reports what a response already does
// and cannot grant it, so counting its words would price a permission the contract withholds.
const researchNoteMarker = "Research note ("

// featureContractAsks counts the words of aggregate publication inside the TR-F033 row of
// the feature checklist: if the contract does not name it, publication has no scope.
func featureContractAsks(root string, terms []string) map[string]int {
	out := map[string]int{}
	src := lines(root, "docs/feature-checklist.md")
	for _, ln := range src {
		if !strings.Contains(ln, "TR-F033") || !strings.HasPrefix(ln, "|") {
			continue
		}
		if i := strings.Index(ln, researchNoteMarker); i >= 0 {
			ln = ln[:i]
		}
		for _, t := range terms {
			if n := strings.Count(strings.ToLower(ln), strings.ToLower(t)); n > 0 {
				out[t] += n
			}
		}
	}
	return out
}

// buildGovernance assembles the permission table from the scans above.
//
// The vocabulary scan walks only Go files: the mechanism the criterion needs is code, and
// counting the word "suppress" in a prose memo would report a rule that does not exist.
func buildGovernance(root string, surfaceFiles []string) ([]governanceRow, []vocabHit, []string) {
	var search []string
	for _, dir := range []string{"internal", "pkg", "cmd"} {
		search = append(search, walkFiles(root, dir, ".go")...)
	}
	sort.Strings(search)
	adminFiles := walkFiles(root, "internal/adminapi", ".go")
	keys, keyFile := readConfigKeys(root)
	var hits []vocabHit
	index := map[string]vocabHit{}
	for _, v := range vocabTerms {
		hit := scanVocab(root, search, v.term)
		hit.Reading = v.reading
		hit.Stats = countMatchesIn(root, surfaceFiles, v.term)
		hits = append(hits, hit)
		index[v.term] = hit
	}
	fallbacks := fallbackSites(root, adminFiles)
	precedent := readOptInPrecedent(root)
	contract := featureContractAsks(root, []string{"statistic", "aggregate", "benchmark", "publish"})
	floor := index["kAnonymity"]
	cell := index["minCell"]
	bar := index["CellFloor"]
	supp := index["SuppressCell"]
	where := floor.Where
	for _, c := range []vocabHit{cell, bar, supp} {
		if where == "" {
			where = c.Where
		}
	}
	rows := []governanceRow{
		{"P01", "a config switch that turns aggregate publication on",
			len(matchingKeys(keys)), 1, keyFile,
			"the registry the system config page renders is the only operator-editable switch; a mechanism absent from it cannot be enabled"},
		{"P02", "a cell-size floor or suppression rule in the Go tree",
			floor.Count + cell.Count + bar.Count + supp.Count, floor.Files + cell.Files + bar.Files + supp.Files, orNone(where),
			"nothing withholds a statistic that covers one row"},
		{"P03", "an anonymiser applied to a served number",
			index["anonymiz"].Count, index["anonymiz"].Files, index["anonymiz"].Where,
			"the shipped privacy code speaks of keys and text, not of numbers"},
		{"P04", "the shipped redactor reaching a statistics handler",
			redactReach(root, surfaceFiles), len(surfaceFiles), "internal/adminapi statistics files",
			"a statistics handler never calls the redactor, so no served number is filtered by the product's own policy"},
		{"P05", "a per-tenant consent row with a stored risk acknowledgement",
			precedent.Fields, 1, precedent.File + ":" + itoa(precedent.Line),
			"the product already owns the governance shape publication needs: " +
				orNone(precedent.Enable) + ", " + orNone(precedent.AckField) + " and a " + orNone(precedent.Scope) + " scope"},
		{"P06", "TR-F033 naming a statistic, an aggregate, a benchmark or publication",
			contractSum(contract), 1, "docs/feature-checklist.md",
			"the contract wording that governs this data - read up to the research note appended to the same row - never mentions an aggregate leaving a tenant"},
		{"P07", "a fallback to the default tenant when no tenant context is present",
			len(fallbacks), 1, orNone(strings.Join(fallbacks, ", ")),
			"a job with no request context reports the default tenant's numbers rather than refusing, which is how a per-tenant statistic gets mis-attributed"},
		{"P08", "a tenant-scope assertion in the dashboard's own test file",
			scopeTests(root, "internal/adminapi/dashboard_test.go"), 1, "internal/adminapi/dashboard_test.go",
			"the largest aggregate surface has no test that a tenant sees only its own numbers"},
	}
	return rows, hits, fallbacks
}

func contractSum(m map[string]int) int {
	n := 0
	for _, v := range m {
		n += v
	}
	return n
}
