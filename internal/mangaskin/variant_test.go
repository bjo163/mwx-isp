package mangaskin

import (
	"strings"
	"testing"
)

func TestApplyNeverMutatesTheShippedTheme(t *testing.T) {
	root := repoRoot(t)
	base, err := LoadTheme(root)
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	before := base.Dark.Tokens["text.disabled"]
	beforeLine := base.Derived.InkLineAlpha["dark"]
	got, err := Apply(base, InkMax())
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if base.Dark.Tokens["text.disabled"] != before {
		t.Error("Apply rewrote the token file's palette in place")
	}
	if base.Derived.InkLineAlpha["dark"] != beforeLine {
		t.Error("Apply rewrote the derived alpha in place")
	}
	if got.Dark.Tokens["text.disabled"] == before {
		t.Error("the re-pointing did not reach the applied skin")
	}
	if got.Derived.InkLineAlpha["dark"] == beforeLine {
		t.Error("the alpha override did not reach the applied skin")
	}
	if got.Light.Tokens["text.primary"] != base.Light.Tokens["text.primary"] {
		t.Error("Apply changed a token no repoint named")
	}
}

func TestApplyRefusesATokenThatDoesNotExist(t *testing.T) {
	base, err := LoadTheme(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	for _, s := range []Skin{
		{ID: "ghost-token", Repoints: []Repoint{{Token: "text.disabled", To: "text.tertiary"}}},
		{ID: "ghost-target", Repoints: []Repoint{{Token: "text.hint", To: "text.secondary"}}},
		{ID: "ghost-alpha", Alphas: map[string]map[string]float64{"inkGlow": {"dark": 0.5}}},
		{ID: "ghost-mode", Alphas: map[string]map[string]float64{"inkLine": {"sepia": 0.5}}},
	} {
		if _, err := Apply(base, s); err == nil {
			t.Errorf("skin %s applied without an error", s.ID)
		} else if !strings.Contains(err.Error(), s.ID) {
			t.Errorf("skin %s failed with an error that does not name it: %v", s.ID, err)
		}
	}
}

// TestInkMaxClosesTheHeldViolationsWithoutIntroducingAColor is the criterion read
// mechanically: the AA misses the standard judges have to go, and no color the
// token file does not already declare may appear.
func TestInkMaxClosesTheHeldViolationsWithoutIntroducingAColor(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(report.NewColors) > 0 {
		t.Errorf("the prototype introduced colors the token file does not declare: %v", report.NewColors)
	}
	if len(report.Summary.FailHeld) == 0 {
		t.Skip("the shipped skin already holds AA on every judged pair; the prototype has nothing to close")
	}
	if report.Totals.VariantFailHeld != 0 {
		for _, m := range report.VariantSummary.FailHeld {
			t.Errorf("prototype still misses %s %s at %.2f:1 (floor %.1f)", m.State.ID, m.Mode, m.Ratio, m.Required)
		}
		t.Fatalf("the re-aimings were expected to close every judged miss")
	}
	if report.VariantSummary.Pass <= report.Summary.Pass {
		t.Errorf("prototype pass count %d did not improve on the shipped %d", report.VariantSummary.Pass, report.Summary.Pass)
	}
}

// TestThePrototypeKeepsEveryDeclaredColorDistinct guards the one thing a
// re-pointing can break by accident: two tokens that mean different things ending
// up with the same value.
func TestThePrototypeKeepsEveryDeclaredColorDistinct(t *testing.T) {
	base, err := LoadTheme(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	applied, err := Apply(base, InkMax())
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	for _, mode := range []string{"light", "dark"} {
		p := applied.PaletteFor(mode)
		for _, pair := range [][2]string{
			{"background.default", "background.paper"},
			{"text.primary", "text.secondary"},
			{"success.main", "warning.main"},
			{"error.main", "info.main"},
		} {
			if p.Tokens[pair[0]] == p.Tokens[pair[1]] {
				t.Errorf("%s mode collapsed %s and %s onto %q", mode, pair[0], pair[1], p.Tokens[pair[0]])
			}
		}
	}
}
