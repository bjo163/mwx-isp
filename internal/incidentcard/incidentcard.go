// Package incidentcard measures the incident card FUT-018 asks for - a printable,
// grunge-styled A4 summary of an outage - against the three things the criterion
// requires of it: the page has to be readable in black and white, the stale and
// confidence state has to be visible, and it must not leak a customer's name.
//
// The study is read-only: it parses the product's own files, holds the parsed
// tables in memory, and writes nothing. Nothing in cmd/ or internal/ imports it,
// and it imports only the standard library.
//
// # What is measured
//
// Three quantities come out of the parsed source:
//
//   - the field ledger. A slot is a candidate for the card only because a response
//     type the admin API already serves carries it; it stays on the card only when
//     the shipped redaction rules, the query that fills it, the prose the handler
//     composes into it and the assignment that names it all agree it is not the
//     subscriber's own data.
//   - the greys. Every operational color token in web/src/theme.ts converted to the
//     grey it dries to on the white sheet the product's own print rule forces, and
//     measured against it and against each other.
//   - the paper. What the console's @media print blocks actually declare, which of
//     the incident surfaces has a print path at all, and which grunge channels
//     survive onto the sheet.
package incidentcard

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Result is one measured run: the ledger, the greys, the paper census and the
// statements that pin them, plus the digest that ties them together.
type Result struct {
	// Rules is the shipped classifier, parsed from the product.
	Rules Rules
	// Tables are the persisted identity models and which of their columns the
	// rules cover.
	Tables []identityTable
	// Slots is the card ledger: every candidate field with its four checks.
	Slots []slot
	// Projections are the query columns bound into response keys.
	Projections []projection
	// Taints are the prose slots the product composes out of an identity row.
	Taints []taint
	// Fills are the assignments behind every name-shaped slot the card could print.
	Fills []filler
	// Skin is the token file the colors were parsed from.
	Skin Skin
	// Gray is every operational token as the mark it dries to.
	Gray []GrayReading
	// Pairs is the pairwise separability of those marks on one sheet.
	Pairs []grayPair
	// Paper is the print census of the console.
	Paper PrintCensus
	// Ink is the grunge skin's channels and what the paper rules do to each.
	Ink []inkChannel
	// Metrics are the vertical numbers the skin commits to.
	Metrics Metrics
	// Fit is the page budget of one card carrying the allowed slots.
	Fit Fit
	// Statuses is the state vocabulary a chip can print.
	Statuses []statusValue
	// Labels counts the non-color channels the console writes for a state.
	Labels []labelChannel
	// Visible is the stale/confidence half of the criterion.
	Visible []Visibility
	// Claims are the statements this run proves, each pinned to a code line.
	Claims []Claim
	// Gaps are what a repository-only round cannot close.
	Gaps []Gap
	// Imports lists the packages this study is allowed to read with.
	Imports []string
	Totals  Totals
	Digest  string
}

// Visibility is one stale-or-confidence field, and where it can be seen.
type Visibility struct {
	Field string
	// Served is the response type that carries it.
	Served string
	// Rendered is the number of code lines in web/src that read it.
	Rendered int
	// OnPaper counts the print paths that carry it, which is the honest half.
	OnPaper int
	// Note records the bound that makes the field meaningful.
	Note string
}

// Totals is the run's headline numbers.
type Totals struct {
	Slots        int
	Allowed      int
	RefusedRule  int
	RefusedQuery int
	RefusedProse int
	// RefusedTrace counts the name-shaped slots held off because the handlers' own
	// assignments do not account for them - the only refusal channel no shipped rule
	// or query reaches.
	RefusedTrace int
	// RefusedGuarded counts the refusals a shipped rule already covers, and
	// RefusedPastRules the ones only this ledger covers: a card that prints them
	// would not break a rule, it would break the criterion.
	RefusedGuarded   int
	RefusedPastRules int
	// NameSlots counts the served keys that speak of a named party, NamePrint the
	// ones whose assignments the study can account for, NameHeld the ones that stay
	// off the sheet.
	NameSlots    int
	NamePrint    int
	NameHeld     int
	Tables       int
	Guarded      int
	Unguarded    int
	GrayLight    int
	GrayLightOK  int
	GrayDark     int
	GrayDarkOK   int
	PairCollide  int
	PaperFiles   int
	Incidents    int
	IncPrintable int
	PageRules    int
	ColorAdjust  int
	InkKept      int
	InkErased    int
	InkUnruled   int
	Statuses     int
	HueGroups    int
	// VisibleOn counts the stale/confidence slots web/src reads; VisibleOff counts
	// those with no print path carrying the surface that renders them.
	VisibleOn   int
	VisibleOff  int
	FitRows     int
	FitPossible int
	Claims      int
	ClaimsOK    int
	Gaps        int
	Imports     int
}

// Run reads the repository and answers the criterion. Nothing is written.
func Run(root string) (*Result, error) {
	res := &Result{}
	rules, err := loadRules(root)
	if err != nil {
		return nil, err
	}
	res.Rules = rules

	tables, err := identityTables(root, rules)
	if err != nil {
		return nil, err
	}
	res.Tables = tables

	res.Projections = readProjections(root, tables)
	prose := proseAssembly(root, tables)
	res.Taints = composeTaints(root, prose)

	defs, err := incidentDefs(root)
	if err != nil {
		return nil, err
	}
	watch := newWatch(defs)
	fills, err := nameFillers(root, tables, watch)
	if err != nil {
		return nil, err
	}
	res.Fills = fills

	slots, err := buildLedger(root, rules, tables, res.Projections, res.Taints, fills)
	if err != nil {
		return nil, err
	}
	res.Slots = slots

	skin, err := loadSkin(root)
	if err != nil {
		return nil, err
	}
	res.Skin = skin

	metrics, err := readMetrics(root)
	if err != nil {
		return nil, err
	}
	res.Metrics = metrics

	// The sheet is white because the product's own print rule says so, in both
	// modes: body background is forced to #fff, so the dark palette prints on the
	// same paper the light palette does.
	paper := Color{R: 255, G: 255, B: 255, Alpha: 1, Text: "#ffffff"}
	res.Gray = measureGray(skin, paper, minNormalText)
	res.Pairs = measureGrayPairs(res.Gray)

	res.Paper = measurePrint(root)
	res.Ink = measureInk(root, res.Paper)
	res.Fit = budgetCard(metrics, len(allowed(slots)))
	res.Statuses = readStatusVocabulary(root)
	res.Labels = measureLabelChannels(root)
	res.Visible = measureVisibility(root, slots, res.Paper)

	res.Claims = evalClaims(root, res)
	res.Gaps = gaps(res)
	res.Imports = packageImports(root, ownPackageDir(root))
	res.Totals = totals(res)
	res.Digest = digest(res)
	return res, nil
}

var rePanelField = regexp.MustCompile(`data\.(confidence|confidence_reason|state|generated_at|warnings)|monitoring\.(stale|stale_after_seconds|last_checked_at|interval_seconds)`)

// measureVisibility asks the middle clause of the criterion where each field can be
// seen: served by the API, rendered by the console, and carried onto paper.
func measureVisibility(root string, slots []slot, paper PrintCensus) []Visibility {
	rendered := map[string]int{}
	for _, rel := range walkFiles(root, webDir, ".tsx", ".ts") {
		for _, ln := range lines(root, rel) {
			for _, m := range rePanelField.FindAllStringSubmatch(ln, -1) {
				rendered[m[1]]++
			}
			for _, name := range []string{"confidence", "stale", "generated_at"} {
				if strings.Contains(ln, name+":") || strings.Contains(ln, name+"?") {
					rendered[name]++
				}
			}
		}
	}
	// A field is on paper only if a print path carries the surface that renders it.
	onPaper := 0
	for _, f := range paper.IncidentWithPrint {
		_ = f
		onPaper++
	}
	notes := map[string]string{
		"state":               "six values, from operational to not_monitored",
		"confidence":          "three values; inferred means the roster is a guess bounded by the 24h lookback",
		"confidence_reason":   "the sentence that says why the level is what it is",
		"generated_at":        "the as-of stamp a printed sheet cannot refresh",
		"warnings":            "what the server refused to conclude",
		"stale":               "the monitor stopped answering, not that the link came back",
		"stale_after_seconds": "the bound that decides it, floored at five minutes",
		"last_checked_at":     "when the last reading was taken",
		"interval_seconds":    "how often a reading is due",
	}
	var out []Visibility
	for _, s := range slots {
		if s.Disposition != disprint {
			continue
		}
		switch s.Key {
		case "state", "confidence", "confidence_reason", "generated_at", "warnings",
			"stale", "stale_after_seconds", "last_checked_at", "interval_seconds":
		default:
			continue
		}
		out = append(out, Visibility{
			Field:    s.Key,
			Served:   s.Struct,
			Rendered: rendered[s.Key],
			OnPaper:  onPaper,
			Note:     orNone(notes[s.Key]),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Field != out[j].Field {
			return out[i].Field < out[j].Field
		}
		return out[i].Served < out[j].Served
	})
	if len(out) == 0 {
		out = append(out, Visibility{Field: "-", Note: "no stale or confidence slot reached the ledger"})
	}
	return out
}

func totals(res *Result) Totals {
	t := Totals{
		Slots:        len(res.Slots),
		Allowed:      len(allowed(res.Slots)),
		Tables:       len(res.Tables),
		PaperFiles:   len(res.Paper.Sites),
		Incidents:    len(res.Paper.IncidentFiles),
		IncPrintable: len(res.Paper.IncidentWithPrint),
		PageRules:    res.Paper.Counts["page geometry declared"],
		ColorAdjust: res.Paper.Counts["backgrounds asked of the browser"] +
			res.Paper.Counts["backgrounds asked, webkit spelling"],
		Statuses:    len(res.Statuses),
		FitRows:     res.Fit.RowsWanted,
		FitPossible: res.Fit.RowsPossible,
		Claims:      len(res.Claims),
		Gaps:        len(res.Gaps),
		Imports:     len(res.Imports),
	}
	for _, s := range res.Slots {
		if s.Disposition != dRefused {
			continue
		}
		switch {
		case s.Class == "Secret" || s.Class == "CustomerPII" || s.Class == "Address":
			t.RefusedRule++
		case s.From != "":
			t.RefusedQuery++
		case s.Composed != "":
			t.RefusedProse++
		case s.Held != "":
			t.RefusedTrace++
		}
	}
	t.RefusedGuarded = len(refusedByReason(res.Slots, func(s slot) bool { return s.Guarded }))
	t.RefusedPastRules = len(refusedByReason(res.Slots, func(s slot) bool { return !s.Guarded }))
	for _, s := range res.Slots {
		if !isNameShape(s.Key) {
			continue
		}
		t.NameSlots++
		if s.Disposition == disprint {
			t.NamePrint++
		} else {
			t.NameHeld++
		}
	}
	for _, tb := range res.Tables {
		t.Guarded += len(tb.Guarded)
		t.Unguarded += len(tb.Unguarded)
	}
	for _, g := range res.Gray {
		if g.Mode == "light" {
			t.GrayLight++
			if g.Pass {
				t.GrayLightOK++
			}
		} else {
			t.GrayDark++
			if g.Pass {
				t.GrayDarkOK++
			}
		}
	}
	for _, p := range res.Pairs {
		if !p.Distinct {
			t.PairCollide++
		}
	}
	for _, ch := range res.Ink {
		switch ch.Disposition {
		case "a paper rule names the property":
			t.InkErased++
		case "no paper rule names the property":
			t.InkUnruled++
		}
		if ch.Disposition != "the skin never writes it" && ch.PaperRules == 0 {
			t.InkKept++
		}
	}
	collisions := map[string]int{}
	for _, v := range res.Statuses {
		collisions[v.Color]++
	}
	for _, n := range collisions {
		if n > 1 {
			t.HueGroups++
		}
	}
	for _, v := range res.Visible {
		if v.Rendered > 0 {
			t.VisibleOn++
		}
		if v.OnPaper == 0 {
			t.VisibleOff++
		}
	}
	for _, c := range res.Claims {
		if c.Status == "confirmed" {
			t.ClaimsOK++
		}
	}
	return t
}

// digest hashes every measured row, so a memo can quote a run and a reader can tell
// whether the tree still says the same thing.
func digest(res *Result) string {
	h := sha256.New()
	write := func(prefix string, rows ...string) {
		for _, r := range rows {
			_, _ = fmt.Fprintf(h, "%s\t%s\n", prefix, r)
		}
	}
	for _, s := range res.Slots {
		write("slot", strings.Join([]string{s.Struct, s.Key, s.Type, s.Class, s.From, s.Composed, s.Held, s.Disposition, yesNo(s.Guarded)}, "\t"))
	}
	for _, fl := range res.Fills {
		write("fill", strings.Join([]string{fl.Struct, fl.Go, fl.source(), yesNo(fl.Traced), yesNo(fl.Subscriber), yesNo(fl.Guarded), fl.File, itoa(fl.Line)}, "\t"))
	}
	for _, tb := range res.Tables {
		write("table", strings.Join([]string{tb.Table, itoa(len(tb.Guarded)), itoa(len(tb.Unguarded))}, "\t"))
	}
	for _, g := range res.Gray {
		write("gray", strings.Join([]string{g.Mode, g.Token, g.Color.Text, itoa(int(g.Gray)), pct(g.OnPaper), yesNo(g.Pass)}, "\t"))
	}
	for _, p := range res.Pairs {
		write("pair", strings.Join([]string{p.Mode, p.A, p.B, pct(p.Ratio), yesNo(p.Distinct)}, "\t"))
	}
	for _, name := range sortedChannelNames(res.Paper.Counts) {
		write("paper", name+"\t"+itoa(res.Paper.Counts[name]))
	}
	for _, f := range res.Paper.IncidentFiles {
		write("incident", f)
	}
	for _, ch := range res.Ink {
		write("ink", strings.Join([]string{ch.Channel, ch.Property, itoa(ch.Declared), itoa(ch.PaperRules), ch.Disposition}, "\t"))
	}
	write("fit", itoa(res.Fit.RowsWanted), fmt.Sprintf("%.1f", res.Fit.RowHeightPx), fmt.Sprintf("%.1f", res.Fit.BudgetPx), itoa(res.Fit.RowsPossible), yesNo(res.Fit.Fits))
	for _, v := range res.Statuses {
		write("status", strings.Join([]string{v.Value, v.Color, itoa(v.Line)}, "\t"))
	}
	for _, l := range res.Labels {
		write("label", strings.Join([]string{l.Channel, itoa(l.Count), l.Load}, "\t"))
	}
	for _, v := range res.Visible {
		write("visible", strings.Join([]string{v.Field, v.Served, itoa(v.Rendered), itoa(v.OnPaper)}, "\t"))
	}
	for _, c := range res.Claims {
		write("claim", strings.Join([]string{c.ID, c.Status, c.File, c.Symbol, itoa(c.Line)}, "\t"))
	}
	for _, g := range res.Gaps {
		write("gap", strings.Join([]string{g.ID, g.Need, g.Status}, "\t"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func sortedChannelNames(counts map[string]int) []string {
	var out []string
	for k := range counts {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func pct(v float64) string { return fmt.Sprintf("%.2f:1", v) }
