// Package mangaskin measures the shipped manga-ink skin against the two numbers
// the FUT-017 criterion asks for: WCAG AA contrast on the states that carry
// operational meaning, and whether the skin can be rolled back without forking a
// stylesheet.
//
// It is a research prototype. It only reads, no product code imports it, and it
// writes nothing. Every color it measures is parsed out of web/src/theme.ts and
// every state it measures is pinned to an identifier in the product code that
// renders it, so a number cannot be quoted for a pair the application never
// draws. A claim whose file or symbol moved is reported as broken rather than
// dropped, and comment lines are blanked before matching so a sentence in a doc
// comment cannot prove a color.
package mangaskin

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Color is one sRGB color with its alpha channel, held in 0-255 integers so a
// parsed value can be compared verbatim against the source file.
type Color struct {
	R, G, B uint8
	// Alpha is 1 for an opaque color. rgba() colors carry their own channel and
	// must be composited over a surface before they can be compared.
	Alpha float64
	// Text is the token exactly as web/src/theme.ts writes it, so a report can
	// quote the value it measured rather than a re-formatted copy.
	Text string
}

// Thresholds used by WCAG 2.1. Two digits decide the verdicts in this package:
// 4.5:1 for ordinary text and 3:1 for the large-text and non-text cases.
const (
	// MinNormalText is the AA ratio for text below the large-text cut-off.
	MinNormalText = 4.5
	// MinLargeOrUI is the AA ratio for large text (1.4.4) and for the non-text
	// contrast of meaningful interface components and graphical objects (1.4.11).
	MinLargeOrUI = 3.0
	// LargeTextPx is the size at which text stops needing 4.5:1.
	LargeTextPx = 24.0
	// LargeBoldTextPx is the reduced cut-off for bold text.
	LargeBoldTextPx = 18.66
	// BoldWeight is the font-weight WCAG treats as bold for that cut-off.
	BoldWeight = 700
)

// Kind is what a measured pair has to satisfy.
type Kind string

// The three contrast requirements a pair can be judged against.
const (
	// KindText is ordinary text: 4.5:1.
	KindText Kind = "text"
	// KindLargeText is large or bold text: 3:1.
	KindLargeText Kind = "large text"
	// KindUI is a non-text object that carries meaning, such as a focus ring or a
	// panel outline: 3:1.
	KindUI Kind = "UI"
	// KindExempt marks a pair WCAG excludes, such as an inactive control. It is
	// still measured, because a ratio below AA on a status label is a readability
	// problem even when the success criterion does not apply.
	KindExempt Kind = "exempt"
)

// Required returns the ratio a kind has to reach. An exempt pair reports against
// the ordinary text floor so the memo can state how far below it sits.
func (k Kind) Required() float64 {
	switch k {
	case KindLargeText, KindUI:
		return MinLargeOrUI
	default:
		return MinNormalText
	}
}

var (
	hex6  = regexp.MustCompile(`^#([0-9a-fA-F]{6})$`)
	hex3  = regexp.MustCompile(`^#([0-9a-fA-F]{3})$`)
	rgba  = regexp.MustCompile(`^rgba\(\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9]+)\s*,\s*([0-9.]+)\s*\)$`)
	hexIn = regexp.MustCompile(`#[0-9a-fA-F]{6}\b`)
)

// hexByte reads one or two hexadecimal characters as a single 0-255 channel.
func hexByte(text string) (uint8, error) {
	n, err := strconv.ParseUint(text, 16, 8)
	if err != nil {
		return 0, fmt.Errorf("hex channel %q: %w", text, err)
	}
	//nolint:gosec // G115: the bitSize above bounds the value to one byte.
	return uint8(n), nil
}

// rgbChannel reads a decimal rgba() channel and refuses anything the browser could
// not draw, so a malformed token cannot silently measure as a different color.
func rgbChannel(text string) (uint8, error) {
	n, err := strconv.Atoi(text)
	if err != nil {
		return 0, fmt.Errorf("rgba channel %q: %w", text, err)
	}
	if n > 255 {
		return 0, fmt.Errorf("rgba channel %q is out of range", text)
	}
	//nolint:gosec // G115: the bound above keeps the value inside one byte.
	return uint8(n), nil
}

// ParseColor reads a #rgb, #rrggbb or rgba(...) literal. An unrecognized value is
// an error rather than a zero color, because a silently black token would move a
// contrast number without saying so.
func ParseColor(text string) (Color, error) {
	text = strings.TrimSpace(text)
	if m := hex6.FindStringSubmatch(text); m != nil {
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
	if m := hex3.FindStringSubmatch(text); m != nil {
		v := func(i int) (uint8, error) {
			n, err := hexByte(m[1][i : i+1])
			if err != nil {
				return 0, err
			}
			return n * 17, nil
		}
		r, err := v(0)
		if err != nil {
			return Color{}, err
		}
		g, err := v(1)
		if err != nil {
			return Color{}, err
		}
		b, err := v(2)
		if err != nil {
			return Color{}, err
		}
		return Color{R: r, G: g, B: b, Alpha: 1, Text: text}, nil
	}
	if m := rgba.FindStringSubmatch(text); m != nil {
		r, err := rgbChannel(m[1])
		if err != nil {
			return Color{}, err
		}
		g, err := rgbChannel(m[2])
		if err != nil {
			return Color{}, err
		}
		b, err := rgbChannel(m[3])
		if err != nil {
			return Color{}, err
		}
		a, err := strconv.ParseFloat(m[4], 64)
		if err != nil || a < 0 || a > 1 {
			return Color{}, fmt.Errorf("bad rgba alpha %q", m[4])
		}
		return Color{R: r, G: g, B: b, Alpha: a, Text: text}, nil
	}
	return Color{}, fmt.Errorf("unrecognized color %q", text)
}

// IsTranslucent reports whether the value needs a surface to be compared at all.
func (c Color) IsTranslucent() bool { return c.Alpha < 1 }

// Over flattens c onto bg. Opaque colors pass through, so a caller can always
// composite and never has to special-case a hex value.
func (c Color) Over(bg Color) Color {
	if c.Alpha >= 1 {
		return Color{R: c.R, G: c.G, B: c.B, Alpha: 1, Text: c.Text}
	}
	mix := func(f, b uint8) uint8 {
		return uint8(math.Round(float64(f)*c.Alpha + float64(b)*(1-c.Alpha)))
	}
	return Color{R: mix(c.R, bg.R), G: mix(c.G, bg.G), B: mix(c.B, bg.B), Alpha: 1,
		Text: fmt.Sprintf("%s over %s", c.Text, bg.Text)}
}

// Luminance returns the WCAG relative luminance of the color.
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

// Contrast returns the WCAG contrast ratio between two colors, always >= 1.
func Contrast(a, b Color) float64 {
	l1, l2 := a.Luminance(), b.Luminance()
	if l2 > l1 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

// IsLarge reports whether a size and weight clear the WCAG large-text cut-off.
// The shipped skin is a compact console - 13-14px body, 10-11px chips and table
// headers - so this returns false for everything the skin labels with color.
func IsLarge(px float64, weight int) bool {
	return px >= LargeTextPx || (px >= LargeBoldTextPx && weight >= BoldWeight)
}

// KindFor picks the requirement a text pair falls under.
func KindFor(px float64, weight int) Kind {
	if IsLarge(px, weight) {
		return KindLargeText
	}
	return KindText
}

// HexLiterals returns the #rrggbb literals in a source text, in file order and
// without de-duplication, so a count of them is a count of decisions made.
func HexLiterals(text string) []string {
	return hexIn.FindAllString(text, -1)
}
