package incidentcard

import (
	"sort"
)

// Claim is one statement the run checks against the tree. The check is a whole
// identifier match on a code line of a named file, after comments are blanked, so a
// sentence in a doc comment cannot confirm a claim and a moved symbol breaks one.
type Claim struct {
	ID     string
	Text   string
	File   string
	Symbol string
	Line   int
	Status string
}

// Gap is something the criterion asks for that a repository-only round cannot
// answer, with the reason printed next to it.
type Gap struct {
	ID   string
	Need string
	// Status is always no-go here: the item needs an asset, not more reading.
	Status string
	Note   string
}

// claimSpec is a statement plus the file and symbol that decides it.
type claimSpec struct {
	id, text, file, symbol string
}

func claimSpecs() []claimSpec {
	return []claimSpec{
		{"A01", "the card inherits its privacy policy from the shipped classifier, not from this study",
			redactFile, classifyFuncName},
		{"A02", "customer_name is one of the keys that classifier already treats as subscriber PII",
			redactFile, "customername"},
		{"A03", "the free-text sweep only knows credentials, URL credentials, e-mail and IPv4 - a personal name is not in its vocabulary",
			redactFile, "emailPattern"},
		{"A04", "the NOC impact response is a served type, so its slots are candidates rather than inventions",
			"internal/adminapi/noc_impact.go", "type nocImpactResponse struct"},
		{"A05", "the impacted roster selects the customer's name column straight out of the identity table",
			"internal/adminapi/noc_impact.go", "c.name AS customer_name"},
		{"A06", "confidence is one of three values the server states about its own conclusion",
			"internal/adminapi/noc_impact.go", "nocImpactConfidenceInferred"},
		{"A07", "the stale state is a named value, not an inference the card has to make",
			"internal/adminapi/noc_impact.go", "nocImpactStateLastDownStale"},
		{"A08", "a printed roster is bounded by a 24h lookback the sheet cannot refresh",
			"internal/adminapi/noc_impact.go", "nocImpactLookback"},
		{"A09", "staleness has a floor of five minutes so a fast monitor cannot flap the printed state",
			"internal/adminapi/noc_impact.go", "nocImpactMinStaleAfter"},
		{"A10", "the auto-ticket composes the customer's name into the ticket description before the card ever sees it",
			"internal/adminapi/isp_operations.go", "customer.Name"},
		{"A11", "the ticket the card would print is a persisted row with prose columns",
			"internal/domain/isp_operations.go", "type TroubleTicket struct"},
		{"A12", "the console forces the sheet white and the ink black before anything is printed",
			"web/src/resources/isp.tsx", "background-color: #fff !important"},
		{"A13", "the same print rule erases the panel fill the grunge skin is made of",
			"web/src/resources/isp.tsx", "background-color: transparent !important"},
		{"A14", "a status chip draws an outlined label, so the state survives without a fill",
			"web/src/components/Enterprise.tsx", "variant=\"outlined\""},
		{"A15", "the state vocabulary that decides chip color is a map in the product, not a fixture",
			"web/src/components/Enterprise.tsx", "statusTone"},
		{"A16", "the skin ships dark as the default, so the dark token set is the one most cards would print",
			"web/src/App.tsx", "defaultTheme"},
		{"A17", "the ink outline that draws a panel border is one alpha the theme picks per mode",
			themeFile, "inkLine"},
		{"A18", "the halftone texture is a background image, not a border",
			themeFile, "radial-gradient"},
		{"A19", "the display face that carries the grunge titles is a local font stack, never fetched",
			themeFile, "Arial Narrow"},
		{"A20", "the print action exists on export surfaces the incident screens do not share",
			"web/src/resources/isp.tsx", "window.print()"},
		{"A21", "the scope name the card would print is read off the monitor target row",
			"internal/adminapi/noc_impact.go", "target.Name"},
		{"A22", "the NAS name is assigned to the scope field, not carried in a select list",
			"internal/adminapi/noc_impact.go", "resp.Scope.NasName = nas.Name"},
		{"A23", "the account record stores a person's real name under a key the shipped rules never name",
			"internal/domain/radius.go", "realname"},
		{"A24", "the subscriber's own name is a plain indexed column on the customer row, also unnamed by the rules",
			"internal/domain/isp.go", "index;size:150;not null"},
	}
}

const classifyFuncName = "func ClassifyKey(key string) Class"

func evalClaims(root string, res *Result) []Claim {
	specs := claimSpecs()
	out := make([]Claim, 0, len(specs))
	for _, s := range specs {
		c := Claim{ID: s.id, Text: s.text, File: s.file, Symbol: s.symbol}
		if src := lines(root, s.file); len(src) == 0 {
			c.Status = "vacated"
		} else if ln := findLine(src, s.symbol); ln > 0 {
			c.Line = ln
			c.Status = "confirmed"
		} else {
			c.Status = "broken"
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	// The claim set has to stay the size the memo prints. A dropped row would look
	// like a passing run.
	if len(out) != len(specs) {
		out = append(out, Claim{ID: "Z00", Text: "claim count drifted", Status: "broken"})
	}
	return out
}

// findLine returns the 1-based line of the first code line carrying the symbol.
func findLine(src []string, symbol string) int {
	for i, ln := range src {
		if containsBounded(ln, symbol) {
			return i + 1
		}
	}
	return 0
}

// gaps records the half of the criterion the repository cannot answer.
func gaps(res *Result) []Gap {
	_ = res
	return []Gap{
		{"G01", "a printed proof sheet from a real monochrome laser",
			"no-go", "the greys here are the WCAG luminance conversion a printer approximates; only paper settles it"},
		{"G02", "a page geometry the product declares",
			"no-go", "zero @page rules exist in web/src, so the printable box is the browser's default and the fit is computed against an assumed 12.7mm margin"},
		{"G03", "a decision on background graphics",
			"no-go", "zero print-color-adjust declarations, so whether the halftone and offset shadow reach paper is a driver setting this round cannot read"},
		{"G04", "a display face that ships with the product",
			"no-go", "web/index.html fetches Inter and JetBrains Mono only; the grunge titles fall back to a local Arial Narrow or Impact"},
		{"G05", "an exporter that produces the card",
			"no-go", "no PDF dependency exists in web/package.json or go.mod, so the only export path is the browser print dialog"},
		{"G06", "a redaction rule that covers a name in prose",
			"no-go", "the card's only defense is refusing the prose slots and tracing each name-shaped slot to its assignment; closing the gap means editing internal/redact, a product change outside a RISET round"},
		{"G07", "incident data from a running deployment",
			"no-go", "the ledger measures served shapes, not a real outage's values"},
	}
}

// statusCounts summarizes the claim table for the totals row.
func statusCounts(claims []Claim) (ok, broken, vacated int) {
	for _, c := range claims {
		switch c.Status {
		case "confirmed":
			ok++
		case "broken":
			broken++
		case "vacated":
			vacated++
		}
	}
	return ok, broken, vacated
}

func refusedByReason(slots []slot, has func(slot) bool) []slot {
	var out []slot
	for _, s := range slots {
		if s.Disposition == dRefused && has(s) {
			out = append(out, s)
		}
	}
	return out
}
