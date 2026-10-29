package incidentcard

import (
	"fmt"
	"math"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// A black-and-white laser printer has one channel: the grey a mark dries to. This
// file measures the shipped skin against that channel. Color values are parsed out
// of web/src/theme.ts, the grey is the WCAG relative luminance of the same color,
// and the sheet is white because the product's own print rule says so.

const themeFile = "web/src/theme.ts"

// WCAG 2.1 thresholds. 4.5:1 is the floor for the 10.9px chip label the skin prints;
// 3:1 is the floor used here for two marks that have to be told apart on paper.
const (
	minNormalText = 4.5
	minPairMark   = 3.0
	remBase       = 16.0
)

// Color is one sRGB color with its alpha, in 0-255 integers, so a parsed value can
// be compared verbatim against the source file.
type Color struct {
	R, G, B uint8
	Alpha   float64
	Text    string
}

var (
	reHex6 = regexp.MustCompile(`^#([0-9a-fA-F]{6})$`)
	reHex3 = regexp.MustCompile(`^#([0-9a-fA-F]{3})$`)
	reRGBA = regexp.MustCompile(`^rgba\(\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9.]+)\s*\)$`)
)

// ParseColor reads a #rgb, #rrggbb or rgba(...) literal. An unrecognized value is an
// error rather than a zero color, because a silently black token would move a
// contrast number without saying so.
func ParseColor(text string) (Color, error) {
	text = strings.TrimSpace(text)
	if m := reHex6.FindStringSubmatch(text); m != nil {
		r, err := hexByte(m[1][0:2])
		if err != nil {
			return Color{}, err
		}
		g, err := hexByte(m[1][2:4])
		if err != nil {
			return Color{}, err
		}
		b, err := hexByte(m[1][4:6])
		if err != nil {
			return Color{}, err
		}
		return Color{R: r, G: g, B: b, Alpha: 1, Text: text}, nil
	}
	if m := reHex3.FindStringSubmatch(text); m != nil {
		var chans [3]uint8
		for i := 0; i < 3; i++ {
			n, err := hexByte(m[1][i : i+1])
			if err != nil {
				return Color{}, err
			}
			chans[i] = n * 17
		}
		return Color{R: chans[0], G: chans[1], B: chans[2], Alpha: 1, Text: text}, nil
	}
	if m := reRGBA.FindStringSubmatch(text); m != nil {
		var chans [3]uint8
		for i := 0; i < 3; i++ {
			n, err := strconv.Atoi(m[i+1])
			if err != nil || n > 255 {
				return Color{}, fmt.Errorf("rgba channel %q out of range", m[i+1])
			}
			//nolint:gosec // G115: the bound above keeps the value inside one byte.
			chans[i] = uint8(n)
		}
		a, err := strconv.ParseFloat(m[4], 64)
		if err != nil || a < 0 || a > 1 {
			return Color{}, fmt.Errorf("bad rgba alpha %q", m[4])
		}
		return Color{R: chans[0], G: chans[1], B: chans[2], Alpha: a, Text: text}, nil
	}
	return Color{}, fmt.Errorf("unrecognized color %q", text)
}

func hexByte(text string) (uint8, error) {
	n, err := strconv.ParseUint(text, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("hex channel %q: %w", text, err)
	}
	//nolint:gosec // G115: the bitSize above bounds the value to one byte.
	return uint8(n), nil
}

// Luminance is the WCAG relative luminance of the color.
func (c Color) Luminance() float64 {
	lum := func(v uint8) float64 {
		s := float64(v) / 255
		if s <= 0.04045 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lum(c.R) + 0.7152*lum(c.G) + 0.0722*lum(c.B)
}

// Contrast is the WCAG ratio between two colors, always >= 1.
func Contrast(a, b Color) float64 {
	l1, l2 := a.Luminance(), b.Luminance()
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// Over flattens a translucent color onto a surface.
func (c Color) Over(bg Color) Color {
	if c.Alpha >= 1 {
		return Color{R: c.R, G: c.G, B: c.B, Alpha: 1, Text: c.Text}
	}
	mix := func(f, b uint8) uint8 {
		//nolint:gosec // G115: the mix of two bounded channels is rounded back into range.
		return uint8(math.Round(float64(f)*c.Alpha + float64(b)*(1-c.Alpha)))
	}
	return Color{R: mix(c.R, bg.R), G: mix(c.G, bg.G), B: mix(c.B, bg.B), Alpha: 1,
		Text: fmt.Sprintf("%s over %s", c.Text, bg.Text)}
}

// GrayLevel is what a color becomes on a monochrome laser printer: the greyscale a
// device produces from a color source is its luminance, so the conversion has to be
// the standard one rather than an average of three channels.
func (c Color) GrayLevel() uint8 {
	//nolint:gosec // G115: Luminance is bounded to 0..1, so the scaled value fits a byte.
	return uint8(math.Round(c.Luminance() * 255))
}

// Palette is one theme mode's token table exactly as theme.ts writes it.
type Palette struct {
	Mode   string
	Tokens map[string]string
}

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

var (
	reGroup = regexp.MustCompile(`([a-zA-Z]+):\s*\{([^{}]*)\}`)
	reValue = regexp.MustCompile(`([a-zA-Z]+):\s*'([^']*)'`)
)

// readPalette parses one palette literal by brace depth so a nested shade object
// stays inside the palette that declares it.
func readPalette(text, decl, mode string) (Palette, error) {
	start := strings.Index(text, decl)
	if start < 0 {
		return Palette{}, fmt.Errorf("%s not found in %s", decl, themeFile)
	}
	lines := strings.Split(text[start:], "\n")
	depth := 0
	end := -1
	for i, l := range lines {
		depth += strings.Count(l, "{") - strings.Count(l, "}")
		if depth == 0 && i > 0 {
			end = i
			break
		}
	}
	if end < 0 {
		return Palette{}, fmt.Errorf("%s block never closes in %s", decl, themeFile)
	}
	block := strings.Join(lines[:end], "\n")
	p := Palette{Mode: mode, Tokens: map[string]string{}}
	top := reGroup.ReplaceAllString(block, "")
	for _, m := range reGroup.FindAllStringSubmatch(block, -1) {
		for _, kv := range reValue.FindAllStringSubmatch(m[2], -1) {
			p.Tokens[m[1]+"."+kv[1]] = kv[2]
		}
	}
	for _, kv := range reValue.FindAllStringSubmatch(top, -1) {
		p.Tokens[kv[1]] = kv[2]
	}
	if len(p.Tokens) == 0 {
		return Palette{}, fmt.Errorf("%s block in %s yielded no tokens", decl, themeFile)
	}
	return p, nil
}

// Skin is the shipped visual system, read from the token file rather than assumed.
type Skin struct {
	Light  Palette
	Dark   Palette
	Source string
}

func loadSkin(root string) (Skin, error) {
	src, ok := readFile(filepath.Join(root, filepath.FromSlash(themeFile)))
	if !ok {
		return Skin{}, fmt.Errorf("%s not readable", themeFile)
	}
	code := blankComments(src)
	light, err := readPalette(code, "const lightPalette = {", "light")
	if err != nil {
		return Skin{}, err
	}
	dark, err := readPalette(code, "const darkPalette = {", "dark")
	if err != nil {
		return Skin{}, err
	}
	return Skin{Light: light, Dark: dark, Source: code}, nil
}

func filepathJoin(root, rel string) string {
	return filepath.Join(root, filepath.FromSlash(rel))
}

// statusTokens are the colors the skin reserves for operational meaning - the ones
// a printed status chip can be drawn with.
func statusTokens() []string {
	var out []string
	for _, g := range []string{"success", "warning", "error", "info", "primary", "secondary"} {
		for _, s := range []string{"main", "light", "dark"} {
			out = append(out, g+"."+s)
		}
	}
	sort.Strings(out)
	return out
}

// GrayReading is one token measured as the mark it dries to.
type GrayReading struct {
	Mode  string
	Token string
	Color Color
	Gray  uint8
	// OnPaper is the contrast of that grey against the white sheet the print rule
	// forces, which is the only reading that matters once the page is on a desk.
	OnPaper float64
	Min     float64
	Pass    bool
}

// measureGray reads every operational token in both modes against the printed sheet.
func measureGray(skin Skin, paper Color, floor float64) []GrayReading {
	var out []GrayReading
	for _, mode := range []string{"light", "dark"} {
		p := skin.Light
		if mode == "dark" {
			p = skin.Dark
		}
		for _, tok := range statusTokens() {
			c, err := p.Color(tok)
			if err != nil {
				continue
			}
			flat := c.Over(paper)
			g := Color{R: flat.GrayLevel(), G: flat.GrayLevel(), B: flat.GrayLevel(), Alpha: 1,
				Text: fmt.Sprintf("gray(%d)", flat.GrayLevel())}
			ratio := Contrast(g, paper)
			out = append(out, GrayReading{Mode: mode, Token: tok, Color: c, Gray: g.R,
				OnPaper: ratio, Min: floor, Pass: ratio >= floor})
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// grayPair is one pair of status marks and how far apart they dry.
type grayPair struct {
	Mode     string
	A, B     string
	Ratio    float64
	Distinct bool
}

// measureGrayPairs asks the second half of the legibility question: even if each
// label can be read, can two statuses be told apart without hue?
func measureGrayPairs(readings []GrayReading) []grayPair {
	var out []grayPair
	for _, m := range []string{"light", "dark"} {
		var main []GrayReading
		for _, r := range readings {
			if r.Mode == m && strings.HasSuffix(r.Token, ".main") {
				main = append(main, r)
			}
		}
		for i := 0; i < len(main); i++ {
			for j := i + 1; j < len(main); j++ {
				a := Color{R: main[i].Gray, G: main[i].Gray, B: main[i].Gray, Alpha: 1}
				b := Color{R: main[j].Gray, G: main[j].Gray, B: main[j].Gray, Alpha: 1}
				out = append(out, grayPair{Mode: m, A: main[i].Token, B: main[j].Token,
					Ratio: Contrast(a, b), Distinct: Contrast(a, b) >= minPairMark})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Mode != out[j].Mode {
			return out[i].Mode < out[j].Mode
		}
		if out[i].Ratio != out[j].Ratio {
			return out[i].Ratio < out[j].Ratio
		}
		return out[i].A+"|"+out[i].B < out[j].A+"|"+out[j].B
	})
	return out
}
