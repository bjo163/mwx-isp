package mangaskin

import (
	"sort"
	"strconv"
	"strings"
)

// RenderMarkdown turns one verification run into the artifact FUT-017 asks for:
// the verdict on each half of the criterion, the contrast table behind it, the one
// token re-aiming that closes the misses it can close, and the vocabulary the skin
// had to work with.
//
// Every number in it is counted from the slices the same run produced. Nothing in
// the document is a color nobody measured.
func RenderMarkdown(r Report) string {
	var b strings.Builder
	confirmed := 0
	for _, f := range r.Findings {
		if f.Status == StatusConfirmed {
			confirmed++
		}
	}
	held := 0
	for _, a := range r.Absences {
		if a.Status == StatusConfirmed {
			held++
		}
	}

	b.WriteString("## Verdict\n\n")
	b.WriteString("Direction claims verified: ")
	b.WriteString(strconv.Itoa(confirmed))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(len(r.Findings)))
	b.WriteString("; gaps still open: ")
	b.WriteString(strconv.Itoa(held))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(len(r.Absences)))
	b.WriteString(".\n\n")
	b.WriteString("Color pairs measured: ")
	b.WriteString(strconv.Itoa(r.Totals.States))
	b.WriteString(" (AA-judged ")
	b.WriteString(strconv.Itoa(r.Totals.Held))
	b.WriteString(", exempt by the standard ")
	b.WriteString(strconv.Itoa(r.Totals.Exempt))
	b.WriteString("). AA violations in the shipped skin: ")
	b.WriteString(strconv.Itoa(r.Totals.FailHeld))
	b.WriteString("; after the prototype re-aiming: ")
	b.WriteString(strconv.Itoa(r.Totals.VariantFailHeld))
	b.WriteString(". Colors the prototype introduces: ")
	b.WriteString(strconv.Itoa(r.Totals.NewColors))
	b.WriteString(". Stylesheets under web/: ")
	b.WriteString(strconv.Itoa(r.Totals.Stylesheets))
	b.WriteString(".\n\n")

	b.WriteString("## Claims the direction rests on\n\n")
	b.WriteString("| Claim | Verdict | Evidence | Statement |\n| --- | --- | --- | --- |\n")
	for _, f := range r.Findings {
		b.WriteString("| ")
		b.WriteString(cell(f.Claim.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(f.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(f.Evidence))
		b.WriteString(" | ")
		b.WriteString(cell(f.Claim.Statement))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Gaps the criterion depends on\n\n")
	b.WriteString("| Gap | Verdict | Found at | What closing it would mean |\n| --- | --- | --- | --- |\n")
	for _, a := range r.Absences {
		b.WriteString("| ")
		b.WriteString(cell(a.Absence.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(a.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(list(a.Hits)))
		b.WriteString(" | ")
		b.WriteString(cell(a.Absence.Gap))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Contrast of the states the console is made of\n\n")
	b.WriteString("| State | Mode | Foreground | Background | Ratio | Floor | Verdict |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, m := range r.States {
		b.WriteString("| ")
		b.WriteString(cell(m.State.ID))
		b.WriteString(" | ")
		b.WriteString(cell(m.Mode))
		b.WriteString(" | ")
		b.WriteString(cell(m.Fg.Text))
		b.WriteString(" | ")
		b.WriteString(cell(m.Bg.Text))
		b.WriteString(" | ")
		b.WriteString(cell(ratio(m.Ratio)))
		b.WriteString(" | ")
		b.WriteString(cell(ratio(m.Required)))
		b.WriteString(" | ")
		b.WriteString(cell(string(m.State.Kind) + " " + verdict(m.Pass)))
		b.WriteString(" |\n")
	}

	b.WriteString("\n### AA violations\n\n")
	if len(r.Summary.FailHeld) == 0 {
		b.WriteString("None.\n")
	}
	for _, m := range r.Summary.FailHeld {
		b.WriteString("- ")
		b.WriteString(m.String())
		b.WriteString("\n")
	}
	b.WriteString("\n### Measured, exempt from AA\n\n")
	b.WriteString("Reported so a clean violation count cannot hide them.\n\n")
	for _, m := range r.Summary.Fail {
		if m.State.Kind == KindExempt {
			b.WriteString("- ")
			b.WriteString(m.String())
			b.WriteString("\n")
		}
	}

	b.WriteString("\n## The prototype skin, as token re-aimings\n\n")
	b.WriteString("Name: ")
	b.WriteString(cell(r.Skin.Name))
	b.WriteString(". Read: ")
	b.WriteString(cell(r.Skin.Read))
	b.WriteString(".\n\n")
	b.WriteString("Every entry below either points one existing token at another existing\n")
	b.WriteString("token's value, or uses a transparency number the token file already writes\n")
	b.WriteString("for the other mode.\n\n")
	b.WriteString("| Change | Value | Why |\n| --- | --- | --- |\n")
	for _, rp := range r.Skin.Repoints {
		b.WriteString("| ")
		b.WriteString(cell("token " + rp.Token))
		b.WriteString(" | ")
		b.WriteString(cell("takes " + rp.To))
		b.WriteString(" | ")
		b.WriteString(cell(rp.Why))
		b.WriteString(" |\n")
	}
	for _, row := range r.AlphaRows() {
		b.WriteString("| ")
		b.WriteString(cell("alpha " + row.Name + " (" + row.Mode + ")"))
		b.WriteString(" | ")
		b.WriteString(cell("alpha(ink, " + row.Value + ")"))
		b.WriteString(" | ")
		b.WriteString(cell(row.Why))
		b.WriteString(" |\n")
	}

	b.WriteString("\n### What changes when they are applied\n\n")
	b.WriteString("| State | Mode | Shipped | Ink Max | Verdict |\n| --- | --- | --- | --- | --- |\n")
	for _, d := range Deltas(r.States, r.VariantStates) {
		b.WriteString("| ")
		b.WriteString(cell(d.State))
		b.WriteString(" | ")
		b.WriteString(cell(d.Mode))
		b.WriteString(" | ")
		b.WriteString(cell(ratio(d.Before)))
		b.WriteString(" | ")
		b.WriteString(cell(ratio(d.After)))
		b.WriteString(" | ")
		b.WriteString(cell(d.Verdict()))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Color vocabulary measured under web/\n\n")
	b.WriteString("| Measure | Value |\n| --- | --- |\n")
	b.WriteString("| Tokens declared per mode | light ")
	b.WriteString(strconv.Itoa(r.Totals.TokensLight))
	b.WriteString(", dark ")
	b.WriteString(strconv.Itoa(r.Totals.TokensDark))
	b.WriteString(" |\n")
	b.WriteString("| Hex literals in theme.ts | ")
	b.WriteString(strconv.Itoa(r.Totals.HexInTheme))
	b.WriteString(" |\n")
	b.WriteString("| Hex literals outside theme.ts | ")
	b.WriteString(strconv.Itoa(r.Totals.HexOutsideTheme))
	b.WriteString(" across ")
	b.WriteString(strconv.Itoa(r.Surface.SourceFiles()))
	b.WriteString(" files |\n")
	b.WriteString("| Hand-picked `mode === 'dark' ?` branches | ")
	b.WriteString(strconv.Itoa(r.Surface.HandPickedModeColors))
	b.WriteString(" |\n")
	b.WriteString("| References to a status token | ")
	b.WriteString(strconv.Itoa(r.Surface.StatusTokenRefs))
	b.WriteString(" |\n")
	b.WriteString("| Files that call a contrast helper | ")
	b.WriteString(cell(list(r.Surface.ContrastHelpers)))
	b.WriteString(" |\n")
	b.WriteString("| Theme factory call sites | ")
	b.WriteString(cell(list(r.Surface.ThemeFactories)))
	b.WriteString(" |\n")
	b.WriteString("| Palette groups with no label color | ")
	b.WriteString(cell(list(r.Theme.UndeclaredStatusLabels("dark"))))
	b.WriteString(" |\n")

	b.WriteString("\nTen files with the most literal colors of their own:\n\n")
	b.WriteString("| File | Hex literals |\n| --- | --- |\n")
	for i, f := range r.Surface.OutsideByFile {
		if i >= 10 {
			break
		}
		b.WriteString("| ")
		b.WriteString(cell(f.File))
		b.WriteString(" | ")
		b.WriteString(strconv.Itoa(f.Count))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Rollback posture\n\n")
	b.WriteString("| Question | Measured answer |\n| --- | --- |\n")
	b.WriteString("| What would a CSS fork fork? | ")
	b.WriteString(strconv.Itoa(r.Totals.Stylesheets))
	b.WriteString(" stylesheets under web/ |\n")
	b.WriteString("| Theme factories defined | ")
	b.WriteString(strconv.Itoa(len(r.Surface.ThemeFactoryDefs)))
	b.WriteString(" |\n")
	b.WriteString("| Theme factory call sites | ")
	b.WriteString(strconv.Itoa(len(r.Surface.ThemeFactories)))
	b.WriteString(" |\n")
	b.WriteString("| Can a deployment return to the declared defaults? | ")
	b.WriteString(cell(string(r.Verdict("the-deployment-can-return-to-the-declared-defaults").Status)))
	b.WriteString(" |\n")
	b.WriteString("| Colors the prototype introduces | ")
	b.WriteString(cell(list(r.NewColors)))
	b.WriteString(" |\n")
	return b.String()
}

// Delta is one state whose ratio moved between two skins.
type Delta struct {
	// State is the state ID that moved.
	State string
	// Mode is the theme mode it moved in.
	Mode string
	// Before and After are the ratios in each skin.
	Before, After float64
	// Kind records the floor both numbers are judged against.
	Kind Kind
	// ChangedTo records whether the pair crossed a floor, so a memo cannot present
	// a ratio bump as an accessibility fix or a regression.
	ChangedTo string
}

// Verdict renders the delta as the report line.
func (d Delta) Verdict() string {
	if d.ChangedTo != "" {
		return d.ChangedTo
	}
	return "no floor crossed"
}

// Deltas lists the measurements whose ratio moved between two skins.
func Deltas(before, after []Measurement) []Delta {
	key := func(m Measurement) string { return m.State.ID + "/" + m.Mode }
	by := map[string]Measurement{}
	for _, m := range after {
		by[key(m)] = m
	}
	var out []Delta
	for _, m := range before {
		a, ok := by[key(m)]
		if !ok || a.Ratio == m.Ratio {
			continue
		}
		d := Delta{State: m.State.ID, Mode: m.Mode, Before: m.Ratio, After: a.Ratio, Kind: m.State.Kind}
		if !m.Pass && a.Pass {
			d.ChangedTo = "now passes " + ratio(m.Required)
		} else if m.Pass && !a.Pass {
			d.ChangedTo = "now FAILS " + ratio(m.Required)
		}
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].State != out[j].State {
			return out[i].State < out[j].State
		}
		return out[i].Mode < out[j].Mode
	})
	return out
}

// AlphaRow is one transparency override, resolved against the variant skin so the
// table can show the number actually used rather than the request.
type AlphaRow struct {
	// Name is the derived expression, e.g. inkLine.
	Name string
	// Mode is the theme mode the override lands in.
	Mode string
	// Value is the alpha applied.
	Value string
	// Why records where that number already comes from.
	Why string
}

// AlphaRows lists the skin's alpha overrides in a stable order.
func (r Report) AlphaRows() []AlphaRow {
	var out []AlphaRow
	for _, name := range sortedKeys(r.Skin.Alphas) {
		for _, mode := range sortedKeys(r.Skin.Alphas[name]) {
			field, ok := alphaField(r.Variant.Derived, name)
			if !ok {
				continue
			}
			other := "light"
			if mode != "dark" {
				other = "dark"
			}
			out = append(out, AlphaRow{
				Name:  name,
				Mode:  mode,
				Value: strconv.FormatFloat(field[mode], 'f', -1, 64),
				Why:   "the " + other + " mode already draws " + name + " at that weight",
			})
		}
	}
	return out
}

// sortedKeys renders a map's keys in lexical order so a report never depends on Go
// map iteration.
func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func ratio(v float64) string { return strconv.FormatFloat(v, 'f', 2, 64) + ":1" }

func list(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func cell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
