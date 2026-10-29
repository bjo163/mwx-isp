package incidentcard

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The card itself is the deliverable, and a card is only printable if its rows fit
// the sheet. Every number in this file comes out of web/src/theme.ts or out of A4
// geometry; the one assumption the repository cannot settle is the default page
// margin, which is stated as an assumption and carried into the gap register.

const (
	// A4 is 210 x 297 mm. At the 96 css px per inch the browser prints at, that is
	// the pixel box below before any margin is taken off.
	a4WidthMM, a4HeightMM = 210.0, 297.0
	mmPerInch             = 25.4
	printDPI              = 96.0
	// assumedMarginMM is the printable margin the study assumes because the tree
	// writes zero "@page" rules: 0.5 in, the common browser default. It is an
	// assumption, not a measurement, and it is the reason the fit is reported with
	// its spare room rather than as a pass.
	assumedMarginMM = 12.7
)

// Metrics are the vertical numbers the skin commits to, read from the token file.
type Metrics struct {
	// RootFontPx is the base typography size the theme sets.
	RootFontPx float64
	// CellFontPx, CellLineHeight and CellPaddingVpx come from MuiTableCell.
	CellFontPx     float64
	CellLineHeight float64
	CellPaddingVpx float64
	// HeaderFontPx and HeaderPaddingVpx come from MuiCardHeader.
	HeaderFontPx   float64
	HeaderPaddingV float64
	// ChipFontPx and ChipHeightPx come from MuiChip, the component a status prints.
	ChipFontPx   float64
	ChipHeightPx float64
	// CardBorderPx is the outline every panel draws.
	CardBorderPx float64
}

var (
	reRootFont  = regexp.MustCompile(`fontSize: ([0-9]+),`)
	reCellRoot  = regexp.MustCompile(`root: \{ borderBottom: '1px solid ' \+ palette\.divider, padding: '([0-9]+)px ([0-9]+)px', fontSize: '([0-9.]+)rem', lineHeight: ([0-9.]+) \}`)
	reCardHead  = regexp.MustCompile(`root: \{ padding: '([0-9]+)px ([0-9]+)px', borderBottom`)
	reChipRoot  = regexp.MustCompile(`root: \{ borderRadius: [0-9]+, fontWeight: [0-9]+, fontSize: '([0-9.]+)rem', height: ([0-9]+)`)
	reHeadTitle = regexp.MustCompile(`title: \{ fontFamily: fontStacks\.display, fontWeight: [0-9]+, fontSize: '([0-9.]+)rem'`)
)

// readMetrics parses the skin's own vertical numbers. Anything missing is an error:
// a page budget computed from half the numbers would be a made-up number.
func readMetrics(root string) (Metrics, error) {
	src := blankText(readFileJoin(root, themeFile))
	m := Metrics{CardBorderPx: 1}
	if mm := reRootFont.FindStringSubmatch(src); mm != nil {
		m.RootFontPx = num(mm[1])
	}
	if mm := reCellRoot.FindStringSubmatch(src); mm != nil {
		m.CellPaddingVpx = num(mm[1])
		m.CellFontPx = num(mm[3]) * remBase
		m.CellLineHeight = num(mm[4])
	}
	if mm := reCardHead.FindStringSubmatch(src); mm != nil {
		m.HeaderPaddingV = num(mm[1])
	}
	if mm := reHeadTitle.FindStringSubmatch(src); mm != nil {
		m.HeaderFontPx = num(mm[1]) * remBase
	}
	if mm := reChipRoot.FindStringSubmatch(src); mm != nil {
		m.ChipFontPx = num(mm[1]) * remBase
		m.ChipHeightPx = num(mm[2])
	}
	for _, spec := range []struct {
		name string
		val  float64
	}{
		{"root font size", m.RootFontPx},
		{"table cell size", m.CellFontPx},
		{"table cell leading", m.CellLineHeight},
		{"header title size", m.HeaderFontPx},
		{"chip height", m.ChipHeightPx},
	} {
		if spec.val == 0 {
			return Metrics{}, fmt.Errorf("%s not parsed out of %s", spec.name, themeFile)
		}
	}
	return m, nil
}

func readFileJoin(root, rel string) string {
	src, _ := readFile(joinRoot(root, rel))
	return src
}

func num(text string) float64 {
	v, _ := strconv.ParseFloat(text, 64)
	return v
}

// Sheet is the A4 page the card has to live on.
type Sheet struct {
	WidthPx  float64
	HeightPx float64
	// MarginPx is the assumption the repository cannot settle, printed so a reader
	// can re-run the arithmetic by hand.
	MarginPx float64
}

// printable returns the box the sheet offers after the assumed margin.
func (s Sheet) printable() (float64, float64) {
	return s.WidthPx - 2*s.MarginPx, s.HeightPx - 2*s.MarginPx
}

// newSheet does the unit conversion once, in the open.
func newSheet() Sheet {
	pxPerMM := printDPI / mmPerInch
	return Sheet{
		WidthPx:  a4WidthMM * pxPerMM,
		HeightPx: a4HeightMM * pxPerMM,
		MarginPx: assumedMarginMM * pxPerMM,
	}
}

// Fit is the page budget of one card.
type Fit struct {
	Sheet        Sheet
	RowsWanted   int
	RowHeightPx  float64
	HeaderPx     float64
	FooterPx     float64
	BudgetPx     float64
	RowsPossible int
	SparePx      float64
	Fits         bool
}

// budgetCard costs one card in the skin's own vertical numbers: a title band, one
// line per printed slot, and the confidence strip the criterion insists is visible.
func budgetCard(m Metrics, rows int) Fit {
	s := newSheet()
	_, height := s.printable()
	row := m.CellPaddingVpx*2 + m.CellFontPx*m.CellLineHeight
	header := m.HeaderPaddingV*2 + m.HeaderFontPx*1.2
	footer := m.CellPaddingVpx*2 + m.ChipHeightPx + m.RootFontPx*1.45
	budget := height - header - footer - m.CardBorderPx*2
	f := Fit{
		Sheet: s, RowsWanted: rows, RowHeightPx: row, HeaderPx: header, FooterPx: footer,
		BudgetPx: budget,
	}
	f.RowsPossible = 0
	if row > 0 {
		f.RowsPossible = int(budget / row)
	}
	f.SparePx = budget - float64(rows)*row
	f.Fits = rows <= f.RowsPossible
	return f
}

// statusValue is one operational state a card can print.
type statusValue struct {
	Value string
	Color string
	Line  int
}

var reStatusTone = regexp.MustCompile(`^\s*(\w+):\s*'(success|warning|error|info|default)'`)

// readStatusVocabulary parses the statusTone map the console renders chips from. It
// is the product's own statement of which states get which color, so a collision is
// the product's, not a fixture's.
func readStatusVocabulary(root string) []statusValue {
	rel := "web/src/components/Enterprise.tsx"
	var out []statusValue
	src := lines(root, rel)
	start := -1
	for i, ln := range src {
		if strings.HasPrefix(strings.TrimSpace(ln), "const statusTone") {
			start = i
			break
		}
	}
	if start < 0 {
		return nil
	}
	for i := start + 1; i < len(src) && i <= start+12; i++ {
		for _, part := range strings.Split(src[i], ",") {
			m := reStatusTone.FindStringSubmatch(strings.TrimSpace(part) + ":")
			if m == nil {
				continue
			}
			out = append(out, statusValue{Value: m[1], Color: m[2], Line: i + 1})
		}
		if strings.Contains(src[i], "};") {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Color != out[j].Color {
			return out[i].Color < out[j].Color
		}
		return out[i].Value < out[j].Value
	})
	return out
}

// hueCollision groups the states that share one color, which is the case where a
// printed card must carry the distinction in text.
func hueCollision(values []statusValue) map[string][]string {
	out := map[string][]string{}
	for _, v := range values {
		out[v.Color] = append(out[v.Color], v.Value)
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

// labelChannel counts the ways a state is written that do not depend on ink color.
type labelChannel struct {
	Channel string
	Count   int
	// Load is the line or file the count came from, printed so the number can be
	// checked without reading the code first.
	Load string
}

func measureLabelChannels(root string) []labelChannel {
	var out []labelChannel
	for _, spec := range []struct{ name, file, token string }{
		{"uppercase status label", "web/src/components/Enterprise.tsx", ".toUpperCase()"},
		{"outline-only chip", "web/src/components/Enterprise.tsx", `variant="outlined"`},
		{"translated status text", "web/src/components/Enterprise.tsx", "status.${v}"},
		{"monospaced identifiers", "web/src/components/Enterprise.tsx", "fontStacks.mono"},
	} {
		n := countMatches(root, spec.file, spec.token)
		out = append(out, labelChannel{Channel: spec.name, Count: n,
			Load: spec.file + ":" + itoa(firstLine(root, spec.file, spec.token))})
	}
	return out
}

func blankText(src string) string { return blankComments(src) }

func joinRoot(root, rel string) string { return filepathJoin(root, rel) }
