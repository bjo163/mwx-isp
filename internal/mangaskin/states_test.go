package mangaskin

import (
	"math"
	"strconv"
	"strings"
	"testing"
)

func TestParsesHexShorthandAndRGBA(t *testing.T) {
	c, err := ParseColor("#0f0")
	if err != nil {
		t.Fatalf("short hex: %v", err)
	}
	if c.R != 0 || c.G != 255 || c.B != 0 {
		t.Errorf("#0f0 expanded to %+v", c)
	}
	r, err := ParseColor("rgba(13, 13, 13, 0.16)")
	if err != nil {
		t.Fatalf("rgba: %v", err)
	}
	if r.Alpha != 0.16 || r.R != 13 {
		t.Errorf("rgba read as %+v", r)
	}
	if _, err := ParseColor("palette.error.main"); err == nil {
		t.Error("a token reference must not parse as a color")
	}
	for _, bad := range []string{"rgba(300, 0, 0, 1)", "rgba(0, 0, 0, 1.5)"} {
		if _, err := ParseColor(bad); err == nil {
			t.Errorf("%s parsed instead of being refused", bad)
		}
	}
}

func TestContrastMatchesThePublishedWCAGRatios(t *testing.T) {
	cases := []struct {
		a, b string
		want float64
	}{
		{"#000000", "#ffffff", 21.0},
		{"#ffffff", "#ffffff", 1.0},
		{"#777777", "#ffffff", 4.48},
		{"#767676", "#ffffff", 4.54},
	}
	for _, c := range cases {
		fg, err := ParseColor(c.a)
		if err != nil {
			t.Fatalf("%s: %v", c.a, err)
		}
		bg, err := ParseColor(c.b)
		if err != nil {
			t.Fatalf("%s: %v", c.b, err)
		}
		got := Contrast(fg, bg)
		if math.Abs(got-c.want) > 0.02 {
			t.Errorf("contrast(%s, %s) = %.3f, want %.2f", c.a, c.b, got, c.want)
		}
	}
}

func TestTranslucentColorIsFlattenedBeforeItIsCompared(t *testing.T) {
	ink, err := ParseColor("rgba(13, 13, 13, 0.16)")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	paper, err := ParseColor("#fffdf6")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	flat := ink.Over(paper)
	if flat.Alpha != 1 {
		t.Errorf("composited color still carries alpha %v", flat.Alpha)
	}
	if flat.R == ink.R {
		t.Errorf("compositing left the ink unchanged: %+v", flat)
	}
	if got := Contrast(flat, paper); got > 1.6 {
		t.Errorf("a 16%% ink line over paper measured %.2f:1, far too high for a hairline", got)
	}
	opaque, err := ParseColor("#5d7000")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	flatOpaque := opaque.Over(paper)
	if flatOpaque.R != opaque.R || flatOpaque.G != opaque.G || flatOpaque.B != opaque.B {
		t.Errorf("an opaque color must pass through compositing unchanged, got %+v", flatOpaque)
	}
}

func TestLargeTextCutoffKeepsEveryStatusLabelAtNormalText(t *testing.T) {
	// The skin's largest body label is 14px and its chips are 10.9px bold; none of
	// them reaches 18.66px, so none of them earns the 3:1 floor.
	for _, px := range []float64{10.4, 10.9, 11.5, 13, 14} {
		if KindFor(px, 700) != KindText {
			t.Errorf("%.1fpx at weight 700 was promoted out of the 4.5:1 floor", px)
		}
	}
	if KindFor(19, 700) != KindLargeText {
		t.Error("19px bold should reach the large-text floor")
	}
	if KindFor(24, 400) != KindLargeText {
		t.Error("24px should reach the large-text floor")
	}
	if KindUI.Required() != 3 || KindExempt.Required() != 4.5 {
		t.Errorf("floor map wrong: UI %v exempt %v", KindUI.Required(), KindExempt.Required())
	}
}

func TestEveryStateResolvesInBothModes(t *testing.T) {
	theme, err := LoadTheme(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	ms, err := Measure(theme, States())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	if want := len(States()) * 2; len(ms) != want {
		t.Fatalf("got %d measurements, want %d", len(ms), want)
	}
	for _, m := range ms {
		if m.Bg.Text == "" || m.Fg.Text == "" {
			t.Errorf("state %s resolved to an unnamed color: %+v", m.State.ID, m)
		}
	}
}

// TestTheMeasuredPairsComeFromTheTokenFile is the rule that keeps this out of
// being a palette study: a color may only appear in the table because theme.ts
// writes that exact literal or derives it from one it writes.
func TestTheMeasuredPairsComeFromTheTokenFile(t *testing.T) {
	root := repoRoot(t)
	theme, err := LoadTheme(root)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	literals := map[string]bool{}
	for _, m := range hexLiteralRe.FindAllString(theme.SourceText, -1) {
		literals[strings.ToLower(m)] = true
	}
	ms, err := Measure(theme, States())
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	for _, m := range ms {
		for _, c := range []Color{m.Fg, m.Bg} {
			text := strings.ToLower(c.Text)
			switch {
			case strings.HasPrefix(text, "alpha(ink,"):
				for _, a := range append(alphaList(theme.Derived.InkLineAlpha), alphaList(theme.Derived.InkShadowAlpha)...) {
					if !strings.Contains(theme.SourceText, a) {
						t.Errorf("state %s used an alpha the file never writes: %s", m.State.ID, a)
					}
				}
			case strings.Contains(text, " over "):
				continue
			default:
				if !literals[text] {
					t.Errorf("state %s measured a color theme.ts does not declare: %q", m.State.ID, c.Text)
				}
			}
		}
	}
}

// alphaList renders the alphas of one derived expression the way theme.ts writes
// them, so a report can prove the number came from the file.
func alphaList(m map[string]float64) []string {
	var out []string
	for _, v := range m {
		out = append(out, strconv.FormatFloat(v, 'f', -1, 64))
	}
	return out
}

func TestReportTheContrastTable(t *testing.T) {
	root := repoRoot(t)
	theme, err := LoadTheme(root)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	states := States()
	ms, err := Measure(theme, states)
	if err != nil {
		t.Fatalf("Measure: %v", err)
	}
	t.Logf("shipped skin:\n%s", Table(ms))
	if len(ms) > 0 {
		var alt strings.Builder
		for _, m := range ms {
			if len(m.Alternates) > 0 {
				alt.WriteString(m.State.ID + " " + m.Mode + ": " + strings.Join(m.Alternates, ", ") + "\n")
			}
		}
		t.Logf("losing label candidates:\n%s", alt.String())
	}
	sum := Summarize(ms)
	t.Logf("shipped: %d/%d pass, failures by mode %v", sum.Pass, sum.Measured, sum.FailByMode)
}
