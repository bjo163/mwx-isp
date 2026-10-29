package mangaskin

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// themeFile is the single stylesheet-free source of the visual system. FUT-017
// asks whether a skin can be built from tokens that already exist, so this is the
// file every measured color is parsed from rather than copied into Go.
const themeFile = "web/src/theme.ts"

// Palette is one theme mode's token table exactly as web/src/theme.ts writes it.
type Palette struct {
	// Mode is "light" or "dark".
	Mode string
	// Tokens maps a dotted token name ("success.main", "background.paper",
	// "divider") to the literal the file declares for it.
	Tokens map[string]string
}

// Color resolves a token to a color. A translucent token such as the divider
// keeps its alpha so the caller decides what it sits on.
func (p Palette) Color(name string) (Color, error) {
	text, ok := p.Tokens[name]
	if !ok {
		return Color{}, fmt.Errorf("mode %s has no token %q", p.Mode, name)
	}
	c, err := ParseColor(text)
	if err != nil {
		return Color{}, fmt.Errorf("token %s.%s: %w", p.Mode, name, err)
	}
	return c, nil
}

// Surface returns the panel color most status labels are drawn on.
func (p Palette) Surface() (Color, error) { return p.Color("background.paper") }

// OnSurface resolves a token flattened onto the panel color, returning both so a
// report can name the pair it compared.
func (p Palette) OnSurface(name string) (fg, bg Color, err error) {
	fg, err = p.Color(name)
	if err != nil {
		return Color{}, Color{}, err
	}
	bg, err = p.Surface()
	if err != nil {
		return Color{}, Color{}, err
	}
	return fg.Over(bg), bg, nil
}

// On resolves a token flattened onto an explicit surface.
func (p Palette) On(name string, bg Color) (Color, Color, error) {
	fg, err := p.Color(name)
	if err != nil {
		return Color{}, Color{}, err
	}
	return fg.Over(bg), bg, nil
}

// LabelCandidates are the two inks the file itself reaches for when it has to pick
// readable text over a filled surface: primary.contrastText, which the brand
// override computes from black and white, and the body ink. They are the only
// labels a skin built from existing tokens can put on a chip, so no third color
// is introduced here.
func (p Palette) LabelCandidates() []Color {
	var out []Color
	for _, name := range []string{"primary.contrastText", "text.primary"} {
		if c, err := p.Color(name); err == nil && !c.IsTranslucent() {
			out = append(out, c)
		}
	}
	return out
}

// StatusGroups are the colors the skin reserves for operational meaning.
var StatusGroups = []string{"success", "warning", "error", "info"}

// Shades are the three levels each group declares.
var Shades = []string{"main", "light", "dark"}

// declaredGroup matches `name: { key: 'value', ... }` - one palette group.
var declaredGroup = regexp.MustCompile(`([a-zA-Z]+):\s*\{([^{}]*)\}`)

// declaredValue matches `key: 'value'` inside a group or at the top level.
var declaredValue = regexp.MustCompile(`([a-zA-Z]+):\s*'([^']*)'`)

// blockSpan returns the lines of the object literal that opens with the line
// starting at decl, counted by braces so a nested shade object stays inside the
// palette it belongs to.
func blockSpan(lines []string, decl string) ([]string, error) {
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), decl) {
			start = i
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("%s not found in %s", decl, themeFile)
	}
	depth := 0
	for i := start; i < len(lines); i++ {
		depth += strings.Count(lines[i], "{") - strings.Count(lines[i], "}")
		if depth == 0 && i > start {
			return lines[start:i], nil
		}
	}
	return nil, fmt.Errorf("%s block never closes in %s", decl, themeFile)
}

// readPalette parses one of the two palette literals in theme.ts. Group entries
// and top-level entries are read separately so `primary: { main: ... }` on a
// single line still yields the dotted name the components use.
func readPalette(lines []string, decl, mode string) (Palette, error) {
	block, err := blockSpan(lines, decl)
	if err != nil {
		return Palette{}, err
	}
	text := strings.Join(block, "\n")
	p := Palette{Mode: mode, Tokens: map[string]string{}}
	top := declaredGroup.ReplaceAllString(text, "")
	for _, m := range declaredGroup.FindAllStringSubmatch(text, -1) {
		for _, kv := range declaredValue.FindAllStringSubmatch(m[2], -1) {
			p.Tokens[m[1]+"."+kv[1]] = kv[2]
		}
	}
	for _, kv := range declaredValue.FindAllStringSubmatch(top, -1) {
		p.Tokens[kv[1]] = kv[2]
	}
	if len(p.Tokens) == 0 {
		return Palette{}, fmt.Errorf("%s block in %s yielded no tokens", decl, themeFile)
	}
	return p, nil
}

// modeColors reads `const name = isDark ? '<dark>' : '<light>'`, the shape
// theme.ts uses for surfaces that are not palette entries.
var modeColors = regexp.MustCompile(`const (\w+) = isDark \? '([^']+)' : '([^']+)'`)

// ModePair is a value declared per mode in one expression.
type ModePair struct {
	// Light and Dark are the two values that expression yields.
	Light Color
	Dark  Color
}

// For returns one side of the pair.
func (m ModePair) For(mode string) Color {
	if mode == "dark" {
		return m.Dark
	}
	return m.Light
}

// Derived holds the mode-dependent values the skin computes rather than declares
// in a palette: the header surface, the panel outline and the hard offset shadow.
type Derived struct {
	// Surfaces are the mode-picked color constants, keyed by Go identifier.
	Surfaces map[string]ModePair
	// InkLineAlpha is the alpha applied to the body ink to draw a panel outline.
	InkLineAlpha map[string]float64
	// InkShadowAlpha is the alpha applied to the body ink for the offset shadow.
	InkShadowAlpha map[string]float64
}

// alphaPairs read the two alpha() expressions, in the order inkLine then
// inkShadow, so the report quotes the numbers the file holds.
var inkLineExpr = regexp.MustCompile(`const inkLine = alpha\(ink, isDark \? ([0-9.]+) : ([0-9.]+)\)`)

// inkShadowExpr reads the shadow helper's own alpha pair. The template literal
// between the arrow and the alpha() call is skipped by .*, which cannot cross the
// line it starts on.
var inkShadowExpr = regexp.MustCompile(`const inkShadow = \(n: number\) =>.*alpha\(ink, isDark \? ([0-9.]+) : ([0-9.]+)\)`)

func parseFloat(name, text string) (float64, error) {
	v, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", name, err)
	}
	return v, nil
}

func readDerived(text string) (Derived, error) {
	d := Derived{
		Surfaces:       map[string]ModePair{},
		InkLineAlpha:   map[string]float64{},
		InkShadowAlpha: map[string]float64{},
	}
	for _, m := range modeColors.FindAllStringSubmatch(text, -1) {
		dark, err := ParseColor(m[2])
		if err != nil {
			return Derived{}, fmt.Errorf("surface %s dark: %w", m[1], err)
		}
		light, err := ParseColor(m[3])
		if err != nil {
			return Derived{}, fmt.Errorf("surface %s light: %w", m[1], err)
		}
		d.Surfaces[m[1]] = ModePair{Light: light, Dark: dark}
	}
	if len(d.Surfaces) == 0 {
		return Derived{}, fmt.Errorf("no mode-specific surface found in %s", themeFile)
	}
	for _, spec := range []struct {
		name   string
		re     *regexp.Regexp
		target map[string]float64
	}{{"inkLine", inkLineExpr, d.InkLineAlpha}, {"inkShadow", inkShadowExpr, d.InkShadowAlpha}} {
		m := spec.re.FindStringSubmatch(text)
		if m == nil {
			return Derived{}, fmt.Errorf("%s expression not found in %s", spec.name, themeFile)
		}
		dark, err := parseFloat(spec.name+" dark alpha", m[1])
		if err != nil {
			return Derived{}, err
		}
		light, err := parseFloat(spec.name+" light alpha", m[2])
		if err != nil {
			return Derived{}, err
		}
		spec.target["dark"] = dark
		spec.target["light"] = light
	}
	return d, nil
}

// Theme is the skin as the product writes it.
type Theme struct {
	// Light and Dark are the two palettes the app bar toggles between.
	Light Palette
	Dark  Palette
	// Derived are the computed surfaces.
	Derived Derived
	// SourceText is theme.ts verbatim, kept so a literal count and a token lookup
	// read the same bytes.
	SourceText string
}

// PaletteFor returns the palette of a mode.
func (t Theme) PaletteFor(mode string) Palette {
	if mode == "dark" {
		return t.Dark
	}
	return t.Light
}

// HeaderSurface returns the toolbar and table-head color for a mode.
func (t Theme) HeaderSurface(mode string) Color { return t.Derived.Surfaces["headerBg"].For(mode) }

// Outline returns the panel outline for a mode flattened on that mode's panel
// color, which is the only way a translucent line can be compared.
func (t Theme) Outline(mode string) (fg, bg Color, err error) {
	p := t.PaletteFor(mode)
	ink, err := p.Color("text.primary")
	if err != nil {
		return Color{}, Color{}, err
	}
	bg, err = p.Surface()
	if err != nil {
		return Color{}, Color{}, err
	}
	a, ok := t.Derived.InkLineAlpha[mode]
	if !ok {
		return Color{}, Color{}, fmt.Errorf("no inkLine alpha for mode %q", mode)
	}
	expr := fmt.Sprintf("alpha(ink, %g)", a)
	line := Color{R: ink.R, G: ink.G, B: ink.B, Alpha: a, Text: expr}
	flat := line.Over(bg)
	flat.Text = expr
	return flat, bg, nil
}

// LoadTheme reads and parses web/src/theme.ts.
func LoadTheme(root string) (Theme, error) {
	//nolint:gosec // G304: path from this package's own catalog, under the repository root.
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(themeFile)))
	if err != nil {
		return Theme{}, err
	}
	text := string(raw)
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	light, err := readPalette(lines, "const lightPalette = {", "light")
	if err != nil {
		return Theme{}, err
	}
	dark, err := readPalette(lines, "const darkPalette = {", "dark")
	if err != nil {
		return Theme{}, err
	}
	derived, err := readDerived(text)
	if err != nil {
		return Theme{}, err
	}
	return Theme{Light: light, Dark: dark, Derived: derived, SourceText: text}, nil
}

// UndeclaredStatusLabels lists the groups whose palette entry carries no
// contrastText, so the label drawn on a filled one is chosen by the component
// library rather than decided by the skin.
func (t Theme) UndeclaredStatusLabels(mode string) []string {
	p := t.PaletteFor(mode)
	var out []string
	for _, group := range append(append([]string{}, StatusGroups...), "primary", "secondary") {
		if _, ok := p.Tokens[group+".contrastText"]; !ok {
			out = append(out, group)
		}
	}
	sort.Strings(out)
	return out
}

// HexInSource returns the #rrggbb literals a source text declares.
func (t Theme) HexInSource() int { return len(HexLiterals(t.SourceText)) }

// FileCount is one measured tally of literals a file writes for itself.
type FileCount struct {
	// File is the repository-relative path.
	File string
	// Count is how many hex literals that file holds outside theme.ts.
	Count int
}

// Surface is what the color vocabulary of the admin UI looks like, measured
// rather than assumed.
type Surface struct {
	// TokensInMode counts the color tokens theme.ts declares per mode.
	TokensInMode map[string]int
	// HexInTheme counts the hex literals inside theme.ts.
	HexInTheme int
	// HexOutsideTheme counts them across the rest of web/src.
	HexOutsideTheme int
	// OutsideByFile breaks that count down, largest first.
	OutsideByFile []FileCount
	// Stylesheets counts .css, .scss and .less files under web/, which is the
	// surface a forked stylesheet would have to live in.
	Stylesheets int
	// HandPickedModeColors counts `mode === 'dark' ?` decisions taken inside a
	// component instead of in the token file.
	HandPickedModeColors int
	// StatusTokenRefs counts references to a status color in web/src.
	StatusTokenRefs int
	// ContrastHelpers lists the files that call a contrast function at all.
	ContrastHelpers []string
	// ThemeFactories lists the files that call a theme factory.
	ThemeFactories []string
	// ThemeFactoryDefs lists the files that define one, which is the number that
	// decides whether a variant could be added without touching the product.
	ThemeFactoryDefs []string
}

// walkWeb runs fn over the source under web/, skipping build output.
func walkWeb(root string, fn func(path, text string) error) error {
	base := filepath.Join(root, "web")
	if _, err := os.Stat(base); err != nil {
		return err
	}
	return filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case "node_modules", "dist", "playwright-report", "test-results":
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if !strings.HasPrefix(rel, "web/src/") && rel != "web/index.html" {
			return nil
		}
		//nolint:gosec // G304: path from the walk over web/, filtered to web/src before reading.
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		return fn(rel, string(raw))
	})
}

var (
	hexLiteralRe   = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
	modePickRe     = regexp.MustCompile(`mode === 'dark' \?`)
	statusPropRe   = regexp.MustCompile(`'(success|warning|error|info)\.(main|light|dark)'`)
	statusPathRe   = regexp.MustCompile(`palette\.(success|warning|error|info)\.(main|light|dark)`)
	contrastFnRe   = regexp.MustCompile(`getContrastRatio|getTextContrast|useColor`)
	themeFactoryRe = regexp.MustCompile(`createAppTheme\(`)
	// themeFactoryDefRe matches the factory's own declaration, not its callers.
	themeFactoryDefRe = regexp.MustCompile(`const createAppTheme\s*=`)
)

// MeasureSurface walks web/ and counts the vocabulary the skin actually has to
// work with. Comment lines are blanked first, so a hex value named in prose is
// not counted as a color a component declared.
func MeasureSurface(root string) (Surface, error) {
	s := Surface{TokensInMode: map[string]int{}}
	if theme, err := LoadTheme(root); err == nil {
		s.TokensInMode["light"] = len(theme.Light.Tokens)
		s.TokensInMode["dark"] = len(theme.Dark.Tokens)
		s.HexInTheme = theme.HexInSource()
	}
	counts := map[string]int{}
	css := 0
	err := walkWeb(root, func(rel, text string) error {
		switch filepath.Ext(rel) {
		case ".css", ".scss", ".less":
			css++
		}
		blob := blankComments(text)
		if rel != themeFile {
			counts[rel] += len(hexLiteralRe.FindAllString(blob, -1))
		}
		s.HandPickedModeColors += len(modePickRe.FindAllString(blob, -1))
		s.StatusTokenRefs += len(statusPropRe.FindAllString(blob, -1))
		s.StatusTokenRefs += len(statusPathRe.FindAllString(blob, -1))
		if contrastFnRe.MatchString(blob) {
			s.ContrastHelpers = append(s.ContrastHelpers, rel)
		}
		if themeFactoryRe.MatchString(blob) {
			s.ThemeFactories = append(s.ThemeFactories, rel)
		}
		if themeFactoryDefRe.MatchString(blob) {
			s.ThemeFactoryDefs = append(s.ThemeFactoryDefs, rel)
		}
		return nil
	})
	if err != nil {
		return Surface{}, err
	}
	s.Stylesheets = css
	for f, n := range counts {
		s.HexOutsideTheme += n
		s.OutsideByFile = append(s.OutsideByFile, FileCount{File: f, Count: n})
	}
	sort.Slice(s.OutsideByFile, func(i, j int) bool {
		if s.OutsideByFile[i].Count != s.OutsideByFile[j].Count {
			return s.OutsideByFile[i].Count > s.OutsideByFile[j].Count
		}
		return s.OutsideByFile[i].File < s.OutsideByFile[j].File
	})
	sort.Strings(s.ContrastHelpers)
	sort.Strings(s.ThemeFactories)
	sort.Strings(s.ThemeFactoryDefs)
	if s.HexInTheme == 0 {
		return Surface{}, fmt.Errorf("%s yielded no hex literals", themeFile)
	}
	return s, nil
}

// SourceFiles lists the files measured outside the token file, largest first.
func (s Surface) SourceFiles() int { return len(s.OutsideByFile) }

// blankComments drops comment-only lines and trailing comments so a color named
// in prose cannot be counted as one the skin declares. A // inside a quoted string
// is left alone, so a URL in a source file survives.
func blankComments(text string) string {
	var out []string
	inBlock := false
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if inBlock {
			if strings.Contains(trimmed, "*/") {
				inBlock = false
			}
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "/*") {
			inBlock = !strings.Contains(trimmed, "*/")
			out = append(out, "")
			continue
		}
		if strings.HasPrefix(trimmed, "*/") || strings.HasPrefix(trimmed, "*") {
			out = append(out, "")
			continue
		}
		// A // only ends a line when it sits outside a string. TypeScript in this
		// tree quotes with single quotes, so both quote styles have to balance.
		if i := strings.Index(trimmed, "//"); i >= 0 &&
			strings.Count(trimmed[:i], "\"")%2 == 0 && strings.Count(trimmed[:i], "'")%2 == 0 {
			trimmed = strings.TrimRight(trimmed[:i], " \t")
			if trimmed == "" {
				out = append(out, "")
				continue
			}
		}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n")
}
