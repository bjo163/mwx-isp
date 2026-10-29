package mangaskin

import (
	"fmt"
	"sort"
	"strings"
)

// State is one of the operational states the skin has to keep readable. It is
// written as a pair of token names plus the component that draws it, so a contrast
// number can only be quoted for a combination the application actually renders.
type State struct {
	// ID names the state in the report.
	ID string
	// What is the operator-visible meaning of the state.
	What string
	// Fg is a token name, or one of the @-resolved values: @label for the best
	// label ink the file itself declares, @inkLine for the panel outline and
	// @inkShadow for the hard offset shadow.
	Fg string
	// Bg is a token name, @panel for the card surface or @header for the toolbar
	// and table-head surface.
	Bg string
	// Kind is the WCAG requirement the pair is judged against.
	Kind Kind
	// Px and Weight are the size and weight the skin draws the label at, read from
	// the typography block of theme.ts. They decide whether 4.5:1 or 3:1 applies.
	Px     float64
	Weight int
	// File and Symbol pin the state to the code that draws it.
	File   string
	Symbol string
	// Note qualifies the verdict.
	Note string
}

// Measurement is one state resolved into two colors and a ratio.
type Measurement struct {
	// State is the state measured.
	State State
	// Mode is the theme mode the pair was read from.
	Mode string
	// Fg and Bg are the resolved colors, already flattened onto each other.
	Fg, Bg Color
	// Ratio is the WCAG contrast between them.
	Ratio float64
	// Required is the floor State.Kind has to reach.
	Required float64
	// Pass reports whether the shipped skin meets that floor.
	Pass bool
	// Alternates records the other label inks tried for an @label pair, so a
	// "passes" verdict cannot hide which of the two colors made it pass.
	Alternates []string
}

// String renders one measurement as a single report line.
func (m Measurement) String() string {
	return fmt.Sprintf("%-22s %-5s %-14s %s on %-22s %5.2f:1 need %.1f %s",
		m.State.ID, m.Mode, m.State.Kind, m.Fg.Text, m.Bg.Text, m.Ratio, m.Required, verdict(m.Pass))
}

func verdict(pass bool) string {
	if pass {
		return "PASS"
	}
	return "FAIL"
}

// States returns the operational states the criterion's "state penting" refers
// to: the pairs a support ticket or an NOC glance is actually made of.
//
// Every row cites a component that draws it, so the list cannot drift into a
// palette study of colors no screen uses. Sizes come from theme.ts typography:
// this is a compact console, so no status label clears the large-text cut-off and
// every text row is judged at 4.5:1.
func States() []State {
	return []State{
		{ID: "account-enabled", What: "an account line reads as live", Fg: "success.main", Bg: "@panel",
			Kind: KindText, Px: 14, Weight: 400, File: "web/src/pages/AccountSettings.tsx", Symbol: "success.main"},
		{ID: "invoice-settled", What: "a paid invoice total", Fg: "success.main", Bg: "@panel",
			Kind: KindText, Px: 13, Weight: 700, File: "web/src/pages/CustomerPortalPage.tsx", Symbol: "success.main"},
		{ID: "metric-stale", What: "a graph whose data is stale", Fg: "warning.main", Bg: "@panel",
			Kind: KindText, Px: 13, Weight: 400, File: "web/src/components/MrtgTrafficGraph.tsx", Symbol: "warning.main"},
		{ID: "destructive-action", What: "a delete confirmation", Fg: "error.main", Bg: "@panel",
			Kind: KindText, Px: 13, Weight: 400, File: "web/src/components/ConfirmDialog.tsx", Symbol: "error.main"},
		{ID: "directory-hint", What: "a directory integration hint", Fg: "info.main", Bg: "@panel",
			Kind: KindText, Px: 14, Weight: 400, File: "web/src/pages/AccountSettings.tsx", Symbol: "info.main"},
		{ID: "empty-value", What: "a field the operator has not filled", Fg: "text.disabled", Bg: "@panel",
			Kind: KindExempt, Px: 12, Weight: 400, File: "web/src/components/detail/EmptyValue.tsx", Symbol: "text.disabled",
			Note: "WCAG 1.4.3 exempts inactive content, but an empty value is still read by an operator scanning for gaps."},
		{ID: "table-heading", What: "a data-grid column title", Fg: "text.secondary", Bg: "@header",
			Kind: KindText, Px: 10.4, Weight: 600, File: "web/src/theme.ts", Symbol: "headerBg"},
		{ID: "active-nav", What: "the selected item in the side nav", Fg: "primary.contrastText", Bg: "primary.main",
			Kind: KindText, Px: 13, Weight: 700, File: "web/src/theme.ts", Symbol: "RaMenuItemLink-active"},
		{ID: "tooltip-body", What: "a tooltip over a dense table", Fg: "background.paper", Bg: "text.primary",
			Kind: KindText, Px: 11.5, Weight: 400, File: "web/src/theme.ts", Symbol: "MuiTooltip"},
		{ID: "focus-ring", What: "the keyboard focus outline", Fg: "primary.main", Bg: "@panel",
			Kind: KindUI, Px: 0, Weight: 0, File: "web/src/theme.ts", Symbol: "outline",
			Note: "1.4.11 - the ring is how a keyboard operator sees where they are."},
		{ID: "panel-outline", What: "the ink line that draws every panel", Fg: "@inkLine", Bg: "@panel",
			Kind: KindUI, Px: 0, Weight: 0, File: "web/src/theme.ts", Symbol: "inkLine",
			Note: "the manga line language: if the outline disappears, the panel becomes a wash."},
		{ID: "hard-shadow", What: "the hard offset shadow under a card", Fg: "@inkShadow", Bg: "@panel",
			Kind: KindUI, Px: 0, Weight: 0, File: "web/src/theme.ts", Symbol: "inkShadow",
			Note: "measured at 1px of ink, the narrowest the skin ever draws it."},
		{ID: "row-divider", What: "the hairline between two data-grid rows", Fg: "divider", Bg: "@panel",
			Kind: KindExempt, Px: 0, Weight: 0, File: "web/src/theme.ts", Symbol: "divider",
			Note: "decoration, not a control: 1.4.11 does not require a row hairline to reach 3:1, so it is measured to keep the report from hiding it."},
		{ID: "brand-button", What: "a contained primary button", Fg: "@label", Bg: "primary.main",
			Kind: KindText, Px: 13, Weight: 700, File: "web/src/theme.ts", Symbol: "contained"},
		{ID: "chip-success", What: "a filled status chip", Fg: "@label", Bg: "success.main",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/pages/PlatformTenantsPage.tsx", Symbol: "Chip"},
		{ID: "chip-warning", What: "a filled warning chip", Fg: "@label", Bg: "warning.main",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/components/MrtgTrafficGraph.tsx", Symbol: "warning.main"},
		{ID: "chip-error", What: "a filled error chip", Fg: "@label", Bg: "error.main",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/components/CustomError.tsx", Symbol: "error.main"},
		{ID: "chip-info", What: "a filled info chip", Fg: "@label", Bg: "info.main",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/pages/AccountSettings.tsx", Symbol: "info.main"},
		{ID: "wash-success", What: "the lighter shade a component library may pick for a soft fill",
			Fg: "@label", Bg: "success.light", Kind: KindText, Px: 10.9, Weight: 700,
			File: "web/src/theme.ts", Symbol: "success",
			Note: "shade choice belongs to the component library; both declared shades are measured so no fill escapes the audit."},
		{ID: "wash-warning", What: "the soft warning fill", Fg: "@label", Bg: "warning.light",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/theme.ts", Symbol: "warning"},
		{ID: "wash-error", What: "the soft error fill", Fg: "@label", Bg: "error.light",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/theme.ts", Symbol: "error"},
		{ID: "wash-info", What: "the soft info fill", Fg: "@label", Bg: "info.light",
			Kind: KindText, Px: 10.9, Weight: 700, File: "web/src/theme.ts", Symbol: "info"},
	}
}

// resolveSurface maps a background name to a color.
func (t Theme) resolveSurface(mode, name string) (Color, error) {
	switch name {
	case "@panel":
		return t.PaletteFor(mode).Surface()
	case "@header":
		return t.HeaderSurface(mode), nil
	}
	return t.PaletteFor(mode).Color(name)
}

// resolveForeground maps a foreground name to one or more candidate colors.
func (t Theme) resolveForeground(mode, name string, bg Color) ([]Color, error) {
	p := t.PaletteFor(mode)
	switch name {
	case "@label":
		return p.LabelCandidates(), nil
	case "@inkLine":
		fg, _, err := t.Outline(mode)
		if err != nil {
			return nil, err
		}
		return []Color{fg}, nil
	case "@inkShadow":
		ink, err := p.Color("text.primary")
		if err != nil {
			return nil, err
		}
		a, ok := t.Derived.InkShadowAlpha[mode]
		if !ok {
			return nil, fmt.Errorf("no inkShadow alpha for mode %q", mode)
		}
		shadow := Color{R: ink.R, G: ink.G, B: ink.B, Alpha: a}
		flat := shadow.Over(bg)
		flat.Text = fmt.Sprintf("alpha(ink, %g)", a)
		return []Color{flat}, nil
	}
	c, err := p.Color(name)
	if err != nil {
		return nil, err
	}
	return []Color{c.Over(bg)}, nil
}

// MeasureOne resolves a state in one mode against a theme.
func MeasureOne(theme Theme, st State, mode string) (Measurement, error) {
	m := Measurement{State: st, Mode: mode, Required: st.Kind.Required()}
	bg, err := theme.resolveSurface(mode, st.Bg)
	if err != nil {
		return Measurement{}, err
	}
	m.Bg = bg
	fgs, err := theme.resolveForeground(mode, st.Fg, bg)
	if err != nil {
		return Measurement{}, err
	}
	if len(fgs) == 0 {
		return Measurement{}, fmt.Errorf("state %s has no candidate label in mode %s", st.ID, mode)
	}
	best := fgs[0]
	m.Ratio = Contrast(best, bg)
	for _, alt := range fgs[1:] {
		r := Contrast(alt, bg)
		if st.Fg == "@label" {
			m.Alternates = append(m.Alternates, fmt.Sprintf("%s %5.2f:1", alt.Text, r))
		}
		if r > m.Ratio {
			best, m.Ratio = alt, r
		}
	}
	m.Fg = best
	m.Pass = m.Ratio >= m.Required-0.005
	return m, nil
}

// Measure resolves every state in both modes.
func Measure(theme Theme, states []State) ([]Measurement, error) {
	var out []Measurement
	for _, mode := range []string{"light", "dark"} {
		for _, st := range states {
			m, err := MeasureOne(theme, st, mode)
			if err != nil {
				return nil, fmt.Errorf("state %s: %w", st.ID, err)
			}
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Mode != out[j].Mode {
			return out[i].Mode == "dark"
		}
		return out[i].State.ID < out[j].State.ID
	})
	return out, nil
}

// Summary is the count of states that hold and the list of the ones that do not.
//
// A pair the standard leaves out of AA is still measured, but it is counted
// separately: a report that quietly drops the row hairline reads as a clean audit,
// while one that mixes it in with a failing outline reads as three defects when
// only one of them is a violation.
type Summary struct {
	// Measured counts the pairs evaluated.
	Measured int
	// Held counts the pairs AA actually judges.
	Held int
	// Exempt counts pairs measured but left out of the AA verdict by the standard.
	Exempt int
	// Pass counts the pairs that reach their floor.
	Pass int
	// Fail is every pair below its floor, worst first.
	Fail []Measurement
	// FailHeld is the subset of Fail that is an AA violation, so the memo can name
	// the defects a fix has to answer instead of a count that includes decoration.
	FailHeld []Measurement
	// FailByMode counts failures per mode, exempt pairs included.
	FailByMode map[string]int
}

// Summarize reduces measurements to the verdict the memo quotes.
func Summarize(ms []Measurement) Summary {
	s := Summary{FailByMode: map[string]int{}}
	for _, m := range ms {
		s.Measured++
		if m.State.Kind == KindExempt {
			s.Exempt++
		} else {
			s.Held++
		}
		if m.Pass {
			s.Pass++
			continue
		}
		s.Fail = append(s.Fail, m)
		s.FailByMode[m.Mode]++
		if m.State.Kind != KindExempt {
			s.FailHeld = append(s.FailHeld, m)
		}
	}
	sortFailures := func(xs []Measurement) {
		sort.Slice(xs, func(i, j int) bool {
			if xs[i].Ratio != xs[j].Ratio {
				return xs[i].Ratio < xs[j].Ratio
			}
			return xs[i].State.ID < xs[j].State.ID
		})
	}
	sortFailures(s.Fail)
	sortFailures(s.FailHeld)
	return s
}

// Table renders the measurements as fixed-width lines, one per pair.
func Table(ms []Measurement) string {
	var b strings.Builder
	for _, m := range ms {
		b.WriteString(m.String())
		b.WriteString("\n")
	}
	return b.String()
}
