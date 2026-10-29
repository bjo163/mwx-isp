package incidentcard

import (
	"fmt"
	"sort"
	"strings"
)

// Markdown renders the measured tables. Every number the memo quotes is copied from
// here, so the memo and the run cannot disagree.
func Markdown(res *Result) string {
	var b strings.Builder
	sectionLedger(&b, res)
	sectionNames(&b, res)
	sectionTables(&b, res)
	sectionGray(&b, res)
	sectionPairs(&b, res)
	sectionPaper(&b, res)
	sectionInk(&b, res)
	sectionBudget(&b, res)
	sectionStates(&b, res)
	sectionVisibility(&b, res)
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

func sectionLedger(b *strings.Builder, res *Result) {
	header(b, "Field ledger",
		"Every slot the incident response types serve, with the four checks applied: the shipped classifier, the query that fills it, the prose the server composes into it and - for a key that speaks of a named party - the assignment behind it.")
	b.WriteString("| response type | slot | type | class | source | disposition | why |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, s := range res.Slots {
		src := "-"
		if s.From != "" {
			src = s.From + " @ " + s.Proven
		} else if s.Composed != "" {
			src = "prose @ " + s.Composed
		} else if s.Held != "" {
			src = "name trace @ " + s.Held
		}
		row(b, s.Struct, orNone(s.Key), s.Type, orNone(s.Class), src, s.Disposition, orNone(s.Note))
	}
	fmt.Fprintf(b, "\n%d slots, %d printable, %d refused.\n\n",
		len(res.Slots), len(allowed(res.Slots)), len(refused(res.Slots)))
}

func sectionNames(b *strings.Builder, res *Result) {
	header(b, "Name-shaped slots and the rows behind them",
		"The criterion forbids the customer's name, so every served key that says name is followed past its key to the assignment that fills it. A name the handlers do not account for, or account for with a subscriber row, stays off the sheet.")
	b.WriteString("| slot | assignments | reads | disposition |\n| --- | ---: | --- | --- |\n")
	for _, s := range res.Slots {
		if !isNameShape(s.Key) {
			continue
		}
		fills := fillsFor(res.Fills, s.Struct, s.Go)
		var srcs []string
		for _, fl := range fills {
			srcs = append(srcs, fl.source()+" @ "+fl.where())
		}
		if len(srcs) == 0 {
			srcs = append(srcs, "none")
		}
		row(b, s.Struct+"."+s.Key, itoa(len(fills)), strings.Join(srcs, "; "), s.Disposition)
	}
	fmt.Fprintf(b, "\n%d name-shaped slots served, %d printed with every assignment traced to a row that is not the subscriber's, %d held off.\n\n",
		res.Totals.NameSlots, res.Totals.NamePrint, res.Totals.NameHeld)
}

func sectionTables(b *strings.Builder, res *Result) {
	header(b, "Identity tables the rules cover only partly",
		"A persisted model counts as an identity table when the shipped rules already name one of its columns sensitive. The columns it serves that no rule names are the ones a card can reach without a guard.")
	b.WriteString("| table | model | guarded columns | unguarded columns served |\n")
	b.WriteString("| --- | --- | --- | --- |\n")
	for _, t := range res.Tables {
		row(b, t.Table, t.Struct, orNone(strings.Join(t.Guarded, ", ")), orNone(strings.Join(t.Unguarded, ", ")))
	}
	b.WriteString("\n")
}

func sectionGray(b *strings.Builder, res *Result) {
	header(b, "Greys on the printed sheet",
		"Each operational token converted to the grey its WCAG luminance dries to, then measured against the white sheet the product's own print rule forces. The floor is 4.5:1, the AA requirement for the 10.9px chip label the skin prints.")
	b.WriteString("| mode | token | hex | grey 0-255 | grey vs paper | floor | verdict |\n")
	b.WriteString("| --- | --- | --- | ---: | --- | --- | --- |\n")
	for _, g := range res.Gray {
		row(b, g.Mode, g.Token, g.Color.Text, itoa(int(g.Gray)), pct(g.OnPaper), pct(g.Min), verdict(g.Pass))
	}
	b.WriteString("\n")
}

func sectionPairs(b *strings.Builder, res *Result) {
	header(b, "Status marks told apart without hue",
		"The same tokens measured against each other on the sheet. A pair below 3:1 is two statuses that print as the same grey, so the distinction has to live in the label.")
	b.WriteString("| mode | first | second | grey contrast | distinct at 3:1 |\n")
	b.WriteString("| --- | --- | --- | --- | --- |\n")
	for _, p := range res.Pairs {
		row(b, p.Mode, p.A, p.B, pct(p.Ratio), verdict(p.Distinct))
	}
	b.WriteString("\n")
}

func sectionPaper(b *strings.Builder, res *Result) {
	header(b, "Paper surface census",
		"How many code lines in web/src write each paper declaration. A zero on a required token is the finding, not a missing row.")
	b.WriteString("| declaration | lines in web/src |\n| --- | ---: |\n")
	for _, name := range sortedChannelNames(res.Paper.Counts) {
		row(b, name, itoa(res.Paper.Counts[name]))
	}
	b.WriteString("\n### Files that write a paper rule\n\n")
	b.WriteString("| file | declarations | incident surface |\n| --- | --- | --- |\n")
	for _, s := range res.Paper.Sites {
		row(b, s.File, orNone(strings.Join(s.Tokens, "; ")), yesNo(s.Incident))
	}
	b.WriteString("\n### Incident surfaces\n\n")
	b.WriteString("| file | print path |\n| --- | --- |\n")
	for _, f := range res.Paper.IncidentFiles {
		hit := false
		for _, g := range res.Paper.IncidentWithPrint {
			if g == f {
				hit = true
			}
		}
		row(b, f, yesNo(hit))
	}
	b.WriteString("\n")
}

func sectionInk(b *strings.Builder, res *Result) {
	header(b, "Grunge channels on paper",
		"The manga-ink look is a set of paints. This counts each one in the token file and asks whether any @media print block in the tree names the property it draws on.")
	b.WriteString("| channel | css property | lines in theme.ts | paper rules naming it | disposition |\n")
	b.WriteString("| --- | --- | ---: | ---: | --- |\n")
	for _, ch := range res.Ink {
		row(b, ch.Channel, ch.Property, itoa(ch.Declared), itoa(ch.PaperRules), ch.Disposition)
	}
	b.WriteString("\n")
}

func sectionBudget(b *strings.Builder, res *Result) {
	header(b, "Page budget",
		"One A4 sheet, converted at 96 css px per inch, less the margin the product never declares - the tree writes no @page rule, so the margin below is an assumption and the fit is reported with its spare room.")
	b.WriteString("| measure | value |\n| --- | --- |\n")
	row(b, "A4 sheet", fmt.Sprintf("%.1f x %.1f px", res.Fit.Sheet.WidthPx, res.Fit.Sheet.HeightPx))
	row(b, "assumed margin", fmt.Sprintf("%.1f px (%.1f mm)", res.Fit.Sheet.MarginPx, assumedMarginMM))
	row(b, "printable height", fmt.Sprintf("%.1f px", res.Fit.BudgetPx+res.Fit.HeaderPx+res.Fit.FooterPx+2*res.Metrics.CardBorderPx))
	row(b, "title band", fmt.Sprintf("%.1f px", res.Fit.HeaderPx))
	row(b, "confidence strip", fmt.Sprintf("%.1f px", res.Fit.FooterPx))
	row(b, "one slot row", fmt.Sprintf("%.1f px", res.Fit.RowHeightPx))
	row(b, "rows the sheet holds", itoa(res.Fit.RowsPossible))
	row(b, "rows the ledger allows", itoa(res.Fit.RowsWanted))
	row(b, "spare", fmt.Sprintf("%.1f px", res.Fit.SparePx))
	row(b, "fits one page", verdict(res.Fit.Fits))
	b.WriteString("\n")
}

func sectionStates(b *strings.Builder, res *Result) {
	header(b, "State vocabulary and its non-color channels",
		"The chip colors come from the product's own statusTone map. Where several states share a color, only the label tells them apart once the sheet is grey.")
	b.WriteString("| color | states sharing it | collisions |\n| --- | --- | --- |\n")
	groups := hueCollision(res.Statuses)
	var names []string
	for k := range groups {
		names = append(names, k)
	}
	sort.Strings(names)
	for _, k := range names {
		row(b, k, strings.Join(groups[k], ", "), yesNo(len(groups[k]) > 1))
	}
	b.WriteString("\n| channel | lines | first at |\n| --- | ---: | --- |\n")
	for _, l := range res.Labels {
		row(b, l.Channel, itoa(l.Count), l.Load)
	}
	b.WriteString("\n")
}

func sectionVisibility(b *strings.Builder, res *Result) {
	header(b, "Stale and confidence visibility",
		"The middle clause of the criterion. A field counts as visible on screen when web/src reads it, and on paper only when a print path carries the surface that renders it.")
	b.WriteString("| field | served by | lines in web/src | print paths carrying it | note |\n")
	b.WriteString("| --- | --- | ---: | ---: | --- |\n")
	for _, v := range res.Visible {
		row(b, v.Field, v.Served, itoa(v.Rendered), itoa(v.OnPaper), v.Note)
	}
	b.WriteString("\n")
}

func sectionClaims(b *strings.Builder, res *Result) {
	header(b, "Pinned statements",
		"Each statement is checked against a code line of a named file after comments are blanked. Drift is reported, never dropped.")
	b.WriteString("| id | statement | file | line | status |\n| --- | --- | --- | ---: | --- |\n")
	for _, c := range res.Claims {
		line := "-"
		if c.Line > 0 {
			line = itoa(c.Line)
		}
		row(b, c.ID, c.Text, c.File, line, c.Status)
	}
	ok, broken, vacated := statusCounts(res.Claims)
	fmt.Fprintf(b, "\n%d of %d confirmed, %d broken, %d vacated.\n\n", ok, len(res.Claims), broken, vacated)
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
	header(b, "Totals", "The run's headline numbers, all of them counts of the tables above.")
	b.WriteString("| measure | value |\n| --- | --- |\n")
	t := res.Totals
	row(b, "slots in the ledger", itoa(t.Slots))
	row(b, "slots a card may print", itoa(t.Allowed))
	row(b, "refused by the shipped rules", itoa(t.RefusedRule))
	row(b, "refused by a query provenance", itoa(t.RefusedQuery))
	row(b, "refused by composed prose", itoa(t.RefusedProse))
	row(b, "refused by the name trace", itoa(t.RefusedTrace))
	row(b, "refusals a shipped rule already covers", itoa(t.RefusedGuarded))
	row(b, "refusals only this ledger covers", itoa(t.RefusedPastRules))
	row(b, "name-shaped slots served", itoa(t.NameSlots))
	row(b, "name-shaped slots printed from a traced non-subscriber row", itoa(t.NamePrint))
	row(b, "name-shaped slots held off", itoa(t.NameHeld))
	row(b, "identity tables", itoa(t.Tables))
	row(b, "their guarded columns", itoa(t.Guarded))
	row(b, "their unguarded columns served", itoa(t.Unguarded))
	row(b, "light tokens readable on paper", fmt.Sprintf("%d of %d", t.GrayLightOK, t.GrayLight))
	row(b, "dark tokens readable on paper", fmt.Sprintf("%d of %d", t.GrayDarkOK, t.GrayDark))
	row(b, "status pairs indistinguishable in grey", itoa(t.PairCollide))
	row(b, "files writing a paper rule", itoa(t.PaperFiles))
	row(b, "incident surfaces", itoa(t.Incidents))
	row(b, "incident surfaces with a print path", itoa(t.IncPrintable))
	row(b, "@page rules in web/src", itoa(t.PageRules))
	row(b, "print-color-adjust declarations", itoa(t.ColorAdjust))
	row(b, "grunge channels no paper rule names", itoa(t.InkUnruled))
	row(b, "grunge channels a paper rule erases", itoa(t.InkErased))
	row(b, "states in the vocabulary", itoa(t.Statuses))
	row(b, "colors shared by more than one state", itoa(t.HueGroups))
	row(b, "stale/confidence fields visible on screen", itoa(t.VisibleOn))
	row(b, "stale/confidence fields with no print path carrying them", itoa(t.VisibleOff))
	row(b, "rows the ledger wants against the sheet", fmt.Sprintf("%d of %d", t.FitRows, t.FitPossible))
	row(b, "statements pinned", fmt.Sprintf("%d of %d", t.ClaimsOK, t.Claims))
	row(b, "gaps open", itoa(t.Gaps))
	row(b, "packages this study imports", itoa(t.Imports))
	fmt.Fprintf(b, "\nDigest: `%s`\n", res.Digest)
}

func verdict(pass bool) string {
	if pass {
		return "pass"
	}
	return "FAIL"
}
