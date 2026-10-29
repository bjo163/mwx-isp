package mangaskin

import (
	"strings"
	"testing"
)

func TestLoadThemeReadsBothPalettesAndTheDerivedValues(t *testing.T) {
	theme, err := LoadTheme(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	for _, mode := range []string{"light", "dark"} {
		p := theme.PaletteFor(mode)
		if len(p.Tokens) < 20 {
			t.Errorf("mode %s yielded only %d tokens; the parse is missing groups", mode, len(p.Tokens))
		}
		for _, name := range []string{"primary.main", "background.paper", "text.primary", "success.main"} {
			if _, err := p.Color(name); err != nil {
				t.Errorf("mode %s: %v", mode, err)
			}
		}
		if theme.Derived.InkLineAlpha[mode] <= 0 || theme.Derived.InkLineAlpha[mode] > 1 {
			t.Errorf("mode %s inkLine alpha %v is out of range", mode, theme.Derived.InkLineAlpha[mode])
		}
		if theme.Derived.InkShadowAlpha[mode] <= 0 || theme.Derived.InkShadowAlpha[mode] > 1 {
			t.Errorf("mode %s inkShadow alpha %v is out of range", mode, theme.Derived.InkShadowAlpha[mode])
		}
		if theme.HeaderSurface(mode).Text == "" {
			t.Errorf("mode %s has no header surface", mode)
		}
	}
	// The nested shade objects must land as dotted names, and the top-level
	// divider must not be swallowed by the group regex.
	if _, err := theme.Dark.Color("primary.light"); err != nil {
		t.Errorf("a nested shade was not read: %v", err)
	}
	if _, err := theme.Light.Color("divider"); err != nil {
		t.Errorf("the top-level divider entry was not read: %v", err)
	}
}

func TestLoadThemeFailsLoudlyOnAThemeItCannotParse(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, themeFile, "export const theme = {};\n")
	if _, err := LoadTheme(root); err == nil {
		t.Error("a theme file with no palette parsed without an error")
	}
}

func TestUndeclaredStatusLabelsNamesTheGroupsTheLibraryHasToDecideFor(t *testing.T) {
	theme, err := LoadTheme(repoRoot(t))
	if err != nil {
		t.Fatalf("LoadTheme: %v", err)
	}
	for _, mode := range []string{"light", "dark"} {
		got := strings.Join(theme.UndeclaredStatusLabels(mode), ",")
		if got != "error,info,success,warning" {
			t.Errorf("mode %s labels: got %q", mode, got)
		}
	}
}

// has reports whether a measured file list contains a path.
func has(list []string, want string) bool {
	for _, v := range list {
		if v == want {
			return true
		}
	}
	return false
}

func TestMeasureSurfaceCountsTheVocabularyTheSkinHas(t *testing.T) {
	s, err := MeasureSurface(repoRoot(t))
	if err != nil {
		t.Fatalf("MeasureSurface: %v", err)
	}
	// The criterion's rollback half rests on this: there is no stylesheet for a
	// skin to fork, and one factory that builds both modes.
	if s.Stylesheets != 0 {
		t.Errorf("expected the stylesheet-free tree the memo argues from, found %d stylesheets", s.Stylesheets)
	}
	if len(s.ThemeFactoryDefs) != 1 || s.ThemeFactoryDefs[0] != themeFile {
		t.Errorf("expected the one factory the rollback argument rests on, got %v", s.ThemeFactoryDefs)
	}
	if !has(s.ThemeFactories, "web/src/App.tsx") {
		t.Errorf("the mode toggle should be a factory call site, got %v", s.ThemeFactories)
	}
	if s.HexInTheme == 0 {
		t.Error("no hex literals read out of the token file")
	}
	if s.TokensInMode["light"] == 0 || s.TokensInMode["dark"] == 0 {
		t.Errorf("token counts per mode came back empty: %v", s.TokensInMode)
	}
	if s.HandPickedModeColors < 10 {
		t.Errorf("the per-component mode branches are the finding that makes the token path worth arguing; found only %d", s.HandPickedModeColors)
	}
	for _, f := range s.OutsideByFile {
		if f.File == themeFile {
			t.Error("the token file was counted as one of its own outliers")
		}
	}
}

func TestBlankCommentsDropsProseButKeepsStrings(t *testing.T) {
	text := "const a = '#111111'; // see #222222\n// #333333\nconst url = 'https://x.test/#frag';\n"
	got := blankComments(text)
	if strings.Contains(got, "#333333") || strings.Contains(got, "see") {
		t.Errorf("comment text survived blanking:\n%s", got)
	}
	if !strings.Contains(got, "#111111") {
		t.Errorf("code was blanked with the comment:\n%s", got)
	}
	if !strings.Contains(got, "https://x.test/#frag") {
		t.Errorf("a URL inside a string was blanked:\n%s", got)
	}
}
