package mangaskin

import (
	"fmt"
	"sort"
	"strings"
)

// Repoint re-aims one token at the value another token already declares. It is the
// only change a prototype skin is allowed to make, which is what "via existing
// tokens" means in practice: no new hue, no new file, no forked rule.
type Repoint struct {
	// Token is the palette entry the skin changes the meaning of.
	Token string
	// To is the palette entry whose value it takes. Both must exist in theme.ts.
	To string
	// Why records the measurement that asked for it.
	Why string
}

// Skin is a prototype visual direction expressed entirely as re-pointings and as
// transparency values the token file already writes.
type Skin struct {
	// ID names the skin in reports.
	ID string
	// Name is the direction, in the words the design system would use.
	Name string
	// Repoints is the token re-aiming list.
	Repoints []Repoint
	// Alphas overrides one derived expression's alpha for one mode. The value must
	// already appear in theme.ts, so a stronger line is still an existing decision.
	Alphas map[string]map[string]float64
	// Read states what the direction is trying to look like, in one line.
	Read string
}

// Apply returns a copy of the theme with the skin laid over it. It never mutates
// the theme it is given, so the shipped skin and the prototype can be measured
// side by side from one parse.
func Apply(base Theme, s Skin) (Theme, error) {
	out := Theme{
		Light:      clonePalette(base.Light),
		Dark:       clonePalette(base.Dark),
		Derived:    cloneDerived(base.Derived),
		SourceText: base.SourceText,
	}
	for _, r := range s.Repoints {
		for _, p := range []*Palette{&out.Light, &out.Dark} {
			value, ok := p.Tokens[r.To]
			if !ok {
				return Theme{}, fmt.Errorf("skin %s: token %q does not exist in mode %s", s.ID, r.To, p.Mode)
			}
			if _, ok := p.Tokens[r.Token]; !ok {
				return Theme{}, fmt.Errorf("skin %s: token %q does not exist in mode %s", s.ID, r.Token, p.Mode)
			}
			p.Tokens[r.Token] = value
		}
	}
	for name, perMode := range s.Alphas {
		target, ok := alphaField(out.Derived, name)
		if !ok {
			return Theme{}, fmt.Errorf("skin %s: no alpha expression %q", s.ID, name)
		}
		for mode, value := range perMode {
			if _, ok := target[mode]; !ok {
				return Theme{}, fmt.Errorf("skin %s: %s has no %s mode", s.ID, name, mode)
			}
			target[mode] = value
		}
	}
	return out, nil
}

func alphaField(d Derived, name string) (map[string]float64, bool) {
	switch name {
	case "inkLine":
		return d.InkLineAlpha, true
	case "inkShadow":
		return d.InkShadowAlpha, true
	}
	return nil, false
}

func clonePalette(p Palette) Palette {
	out := Palette{Mode: p.Mode, Tokens: make(map[string]string, len(p.Tokens))}
	for k, v := range p.Tokens {
		out.Tokens[k] = v
	}
	return out
}

func cloneDerived(d Derived) Derived {
	out := Derived{
		Surfaces:       make(map[string]ModePair, len(d.Surfaces)),
		InkLineAlpha:   make(map[string]float64, len(d.InkLineAlpha)),
		InkShadowAlpha: make(map[string]float64, len(d.InkShadowAlpha)),
	}
	for k, v := range d.Surfaces {
		out.Surfaces[k] = v
	}
	for k, v := range d.InkLineAlpha {
		out.InkLineAlpha[k] = v
	}
	for k, v := range d.InkShadowAlpha {
		out.InkShadowAlpha[k] = v
	}
	return out
}

// NewColorsIn lists the color values the applied skin introduced that the base
// skin does not already declare. An empty list is the proof that the direction was
// built from existing tokens; anything in it is the finding.
func NewColorsIn(applied, base Theme) []string {
	known := map[string]bool{}
	for _, p := range []Palette{base.Light, base.Dark} {
		for _, v := range p.Tokens {
			known[strings.ToLower(v)] = true
		}
	}
	var out []string
	for _, p := range []Palette{applied.Light, applied.Dark} {
		for name, v := range p.Tokens {
			if !known[strings.ToLower(v)] {
				out = append(out, fmt.Sprintf("%s.%s=%s", p.Mode, name, v))
			}
		}
	}
	sort.Strings(out)
	return out
}

// InkMax is the prototype direction this round measures: the manga-ink language
// taken to its readable limit, with the wash tones pulled up to the ink the file
// already declares and the dark panel line strengthened to an alpha the light mode
// already uses. It is data inside a research package, not a shipped theme.
func InkMax() Skin {
	return Skin{
		ID:   "ink-max",
		Name: "Ink Max",
		Read: "same panel language, every line and every status label carried at full ink",
		Repoints: []Repoint{
			{Token: "text.disabled", To: "text.secondary",
				Why: "an empty value is the one thing an operator scans for, and the washed ink cannot carry it"},
		},
		Alphas: map[string]map[string]float64{
			"inkLine": {"dark": 0.7},
		},
	}
}
