package contextualws

import (
	"fmt"
	"strings"
)

// Markdown renders one run as the tables the memo splices. Every number in the
// memo comes from here, so a table can never disagree with the package that
// produced it.
func Markdown(r *Result) string {
	var b strings.Builder
	section(&b, "The measured numbers")
	dump(&b, totalsRows(r.Totals))

	section(&b, "The shipped menu, numbered the way a keyboard numbers it")
	dump(&b, menuRows(r.MenuCost))

	section(&b, "The task surfaces the console already ships")
	dump(&b, taskRows(r.Tasks))

	section(&b, "Navigation cost: the shipped order against the task's own order")
	dump(&b, costTable(r.Costs))

	section(&b, "Where a task surface and the same role's menu disagree")
	dump(&b, gateTable(r.Gates))

	section(&b, "Every destination a task offers, with the control that carries it")
	dump(&b, edgeTable(r.Edges))

	section(&b, "The navigation controls in the console, by element")
	dump(&b, affordanceTable(r.Affordances))

	section(&b, "The mechanism lines")
	dump(&b, pinTable(r.Pins))

	section(&b, "The statements, pinned to code")
	dump(&b, claimRows(r.Claims))

	section(&b, "What this study does not close")
	dump(&b, gapRows(r.Gaps))

	section(&b, "The import set this package builds on")
	dump(&b, importRows(r))

	section(&b, "The lines this study reads")
	dump(&b, readRows(r.Pins))

	section(&b, "What the digest locks")
	digestSection(&b, r)
	return b.String()
}

func section(b *strings.Builder, title string) {
	b.WriteString("\n## " + title + "\n\n")
}

func dump(b *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		b.WriteString("(no rows)\n")
		return
	}
	for i, ln := range rows {
		b.WriteString("| " + strings.Join(ln, " | ") + " |")
		if i == 0 {
			b.WriteString("\n|")
			for range ln {
				b.WriteString(" --- |")
			}
		}
		b.WriteString("\n")
	}
}

// head starts a table that may have no rows; an empty body still keeps the header
// so the memo's splice cannot silently lose a column.
func head(cols ...string) [][]string {
	return [][]string{cols}
}

func menuRows(rows []MenuRow) [][]string {
	out := head("role arm", "menu destination", "label key", "gate", "forward tab stops", "reachable")
	for _, r := range rows {
		stops := "(not reachable)"
		if r.Visible {
			stops = itoa(r.Stops)
		}
		out = append(out, []string{r.Role, r.To, r.Label, r.Gate, stops, yesNo(r.Visible)})
	}
	return out
}

func taskRows(tasks []Task) [][]string {
	out := head("task", "surface", "what it is called", "source line", "filters by role", "carrier", "keyboard")
	for _, t := range tasks {
		out = append(out, []string{t.ID, t.Surface, t.Name, t.File + ":" + itoa(t.Line),
			yesNo(t.Gates), t.Carrier, t.Keyboard + " (line " + itoa(t.CarrierLine) + ")"})
	}
	return out
}

func costTable(rows []CostRow) [][]string {
	out := head("task", "role arm", "destinations", "stops via the menu", "stops via the task",
		"stops saved", "farthest destination", "stops to it", "not in that menu")
	for _, c := range rows {
		farthest := "(none)"
		if c.WorstDestination != "" {
			farthest = c.WorstDestination
		}
		blocked := "(none)"
		if len(c.Blocked) > 0 {
			blocked = strings.Join(c.Blocked, ", ")
		}
		out = append(out, []string{c.Task, c.Role, itoa(c.Destinations), itoa(c.BaselineStops),
			itoa(c.ContextualStops), itoa(c.SavedStops), farthest, itoa(c.WorstStops), blocked})
	}
	return out
}

func gateTable(rows []GateRow) [][]string {
	out := head("task", "role arm", "how the surface decides", "offered", "hidden by the menu", "which")
	any := false
	for _, g := range rows {
		if g.HiddenEdges == 0 {
			continue
		}
		any = true
		out = append(out, []string{g.Task, g.Role, g.Surfaces, itoa(g.Offered),
			itoa(g.HiddenEdges), strings.Join(g.Hidden, ", ")})
	}
	if !any {
		out = append(out, []string{"(every task x role cell agrees with that role's menu)",
			"", "", "", "", ""})
	}
	return out
}

func edgeTable(rows []EdgeRow) [][]string {
	out := head("task", "surface", "destination", "what it is called", "carrier", "keyboard",
		"declared route", "declared by", "source line")
	for _, e := range rows {
		out = append(out, []string{e.Task, e.Surface, e.To, e.Label, e.Carrier, e.Keyboard,
			yesNo(e.Resolves), e.DeclaredBy, itoa(e.Line)})
	}
	return out
}

func affordanceTable(rows []AffordanceRow) [][]string {
	out := head("console file", "element", "kind", "count")
	for _, a := range rows {
		out = append(out, []string{a.File, a.Carrier, a.Kind, itoa(a.Count)})
	}
	return out
}

func pinTable(rows []PinRow) [][]string {
	out := head("code line", "mechanism", "status")
	for _, p := range rows {
		out = append(out, []string{p.File + ":" + itoa(p.Line), p.Symbol, p.Status})
	}
	return out
}

func claimRows(rows []Claim) [][]string {
	out := head("id", "statement", "code line", "status", "measured", "asserts an absence")
	for _, c := range rows {
		where := c.File
		switch c.Line {
		case -1:
			where += " (file gone)"
		case 0:
			where += " (line moved)"
		default:
			where += ":" + itoa(c.Line)
		}
		out = append(out, []string{c.ID, c.Text, where, c.Status, c.Evid, yesNo(c.Neg)})
	}
	return out
}

func gapRows(rows []Gap) [][]string {
	out := head("id", "what the study cannot close", "what it would need", "kind of asset")
	for _, g := range rows {
		out = append(out, []string{g.ID, g.Text, g.Needs, g.Asset})
	}
	return out
}

func importRows(r *Result) [][]string {
	out := head("package import")
	if r.Import == "" {
		return out
	}
	for _, imp := range strings.Split(r.Import, ", ") {
		out = append(out, []string{imp})
	}
	return out
}

func readRows(rows []PinRow) [][]string {
	out := head("code line", "what it carries")
	for _, p := range rows {
		out = append(out, []string{p.File + ":" + itoa(p.Line), p.Symbol})
	}
	return out
}

func digestSection(b *strings.Builder, r *Result) {
	b.WriteString("The digest is the sha256 of the tab-separated rows below, so the memo's tables\n")
	b.WriteString("are reproducible from the tree alone and move the moment the tree moves.\n\n")
	fmt.Fprintf(b, "- %d menu stop rows\n- %d cost rows\n- %d gate rows\n- %d edges\n"+
		"- %d affordance rows\n- %d mechanism lines\n- %d statements\n- %d limits\n\n",
		len(r.MenuCost), len(r.Costs), len(r.Gates), len(r.Edges), len(r.Affordances),
		len(r.Pins), len(r.Claims), len(r.Gaps))
	fmt.Fprintf(b, "- digest: `%s`\n", r.Digest)
}
