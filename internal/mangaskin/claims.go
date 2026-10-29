package mangaskin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Claim is one statement about the shipped skin, pinned to the file that proves it.
type Claim struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the memo asserts about the skin.
	Statement string
	// File is the repository-relative path that must prove it.
	File string
	// Symbol is the text that must appear in that file. It is matched on code
	// lines only, so a doc comment describing a feature cannot prove it exists.
	Symbol string
	// Note carries the qualification the verdict needs.
	Note string
}

// Absence is a negative claim: a symbol that must not appear anywhere in scope.
type Absence struct {
	// ID names the claim in reports.
	ID string
	// Statement is what the absence proves.
	Statement string
	// Symbol must not appear in any file under Scope.
	Symbol string
	// Scope is the repository-relative directory or file the search covers.
	Scope string
	// Gap says what the missing capability would have been worth.
	Gap string
}

// Status is one verification verdict.
type Status string

// The three verdicts a claim can reach. A broken claim is reported, never
// silently dropped, because a memo that keeps its sentence while the code moved
// would be quoting a color nobody draws.
const (
	StatusConfirmed Status = "confirmed"
	StatusBroken    Status = "broken"
	StatusVacated   Status = "vacated"
)

// Finding is one verified claim.
type Finding struct {
	// Claim is the statement checked.
	Claim Claim
	// Status is the verdict.
	Status Status
	// Evidence is the file:line that proves it, or where the search failed.
	Evidence string
	// Note carries Claim.Note plus any qualification the verdict needs.
	Note string
}

// String renders one finding as a single line.
func (f Finding) String() string {
	return fmt.Sprintf("%s %s [%s] %s", f.Claim.ID, f.Status, f.Claim.File, f.Evidence)
}

// AbsenceFinding is one verified negative claim.
type AbsenceFinding struct {
	// Absence is the negative claim checked.
	Absence Absence
	// Status is confirmed when nothing matches, vacated when something does.
	Status Status
	// Hits lists the matching locations, capped for readability.
	Hits []string
}

// String renders one negative verdict.
func (a AbsenceFinding) String() string {
	if len(a.Hits) == 0 {
		return fmt.Sprintf("%s %s (no %q under %s)", a.Absence.ID, a.Status, a.Absence.Symbol, a.Absence.Scope)
	}
	return fmt.Sprintf("%s %s %s", a.Absence.ID, a.Status, strings.Join(a.Hits, ", "))
}

// maxHits caps how many locations one negative verdict reports.
const maxHits = 4

// Claims returns the statements the FUT-017 argument rests on: that the direction
// the card asks to prototype is the direction already shipped, that it is built
// from tokens rather than a stylesheet, and that rolling it back is a mode change.
func Claims() []Claim {
	return []Claim{
		{
			ID:        "direction-is-already-manga-ink",
			Statement: "The shipped direction is a manga panel language: an ink line draws every panel and a hard offset shadow lifts it, with no elevation blur.",
			File:      "web/src/theme.ts",
			Symbol:    "inkShadow",
			Note:      "elevation is pinned to 0 on Paper, Card, AppBar and Dialog in the same file.",
		},
		{
			ID:        "brushes-are-alpha-steps-of-the-ink",
			Statement: "The outline, the shadow and the hover wash are transparency steps of the body text color, not extra hues the skin introduces.",
			File:      "web/src/theme.ts",
			Symbol:    "inkLine",
			Note:      "measured: inkLine is alpha(ink, 0.28 or 0.7) and hover is alpha(primary.main, 0.07-0.08).",
		},
		{
			ID:        "paper-gets-a-halftone-wash",
			Statement: "The body and the card headers carry a printed-paper halftone, drawn as a radial dot of the same ink.",
			File:      "web/src/theme.ts",
			Symbol:    "halftone",
		},
		{
			ID:        "titles-are-capitalized-display-type",
			Statement: "Panel titles, tabs and headings above h5 are set in a condensed display stack at weight 900 and uppercased.",
			File:      "web/src/theme.ts",
			Symbol:    "fontStacks.display",
		},
		{
			ID:        "corners-stay-square",
			Statement: "The shape radius is 3px, so the brutalist read comes from line and shadow rather than from rounded cards.",
			File:      "web/src/theme.ts",
			Symbol:    "borderRadius: 3",
			Note:      "chips and menu links go to 2px, the tightest the component library still draws as a rectangle.",
		},
		{
			ID:        "density-keeps-every-label-at-normal-text",
			Statement: "Body copy is 13px, so no status label in the skin clears the WCAG large-text cut-off and every one of them is judged at 4.5:1.",
			File:      "web/src/theme.ts",
			Symbol:    "fontSize: 13",
		},
		{
			ID:        "one-accent-per-deployment",
			Statement: "A deployment overrides exactly one brand value, the accent, and the factory recomputes its light and dark shades from it.",
			File:      "web/src/theme.ts",
			Symbol:    "accentColor",
		},
		{
			ID:        "the-accent-override-is-contrast-gated",
			Statement: "A picked accent is stepped until it reaches 4.5:1 against the panel before the theme is built, and falls back to a black or white ink when no step works.",
			File:      "web/src/theme.ts",
			Symbol:    "accessibleAccent",
			Note:      "This is the only contrast gate the product already owns, and it covers the brand color only.",
		},
		{
			ID:        "status-colors-stay-outside-the-brand",
			Statement: "success, warning, error and info are declared as palette groups of their own, so an accent change cannot repaint an operational state.",
			File:      "web/src/theme.ts",
			Symbol:    "warning",
			Note:      "None of the four declares a contrastText, so the label over a filled one is the component library's choice.",
		},
		{
			ID:        "both-modes-come-from-one-factory",
			Statement: "The app builds its light and dark themes from the same factory call, so switching is a state change rather than loading a second stylesheet.",
			File:      "web/src/App.tsx",
			Symbol:    "createAppTheme",
		},
		{
			ID:        "the-deployment-can-return-to-the-declared-defaults",
			Statement: "The branding screen exposes a restore-to-defaults action, so a changed accent can be rolled back by the operator without a code change.",
			File:      "web/src/pages/BrandingPage.tsx",
			Symbol:    "Restore defaults",
		},
		{
			ID:        "the-status-ink-is-a-palette-token-not-a-local-hex",
			Statement: "The live/disabled read is taken from the palette by name, so a skin change reaches it without editing the component.",
			File:      "web/src/pages/AccountSettings.tsx",
			Symbol:    "success.main",
			Note:      "59 components still branch on mode === 'dark' by hand; the token path is the exception, not the rule.",
		},
		{
			ID:        "the-chrome-is-ink-outlined-not-shadowed",
			Statement: "The app bar and the drawer are separated by a 2px ink border instead of a drop shadow.",
			File:      "web/src/theme.ts",
			Symbol:    "MuiAppBar",
		},
		{
			ID:        "motion-is-respected-not-relied-on",
			Statement: "The skin ships a prefers-reduced-motion override, so its hover shift is not the only carrier of feedback.",
			File:      "web/src/theme.ts",
			Symbol:    "prefers-reduced-motion",
		},
	}
}

// Absences returns the negative claims: what must not exist for the criterion's
// "without a CSS fork" half to still be true tomorrow.
func Absences() []Absence {
	return []Absence{
		{
			ID:        "no-stylesheet-to-fork",
			Statement: "No component imports a stylesheet, so a skin cannot be added as a CSS fork by accident.",
			Symbol:    ".css'",
			Scope:     "web/src",
			Gap:       "The entire visual system lives in one theme object; a fork would have to create the file it overrides.",
		},
		{
			ID:        "no-css-custom-properties",
			Statement: "The skin declares no CSS variables, so there is no second source of color that a variant could disagree with.",
			Symbol:    "--",
			Scope:     "web/src/theme.ts",
			Gap:       "Token identity is the palette object alone, which is what makes a re-pointing variant provable.",
		},
		{
			ID:        "no-forced-colors-support",
			Statement: "Nothing honors the forced-colors media feature, so a user on a Windows high-contrast scheme gets the skin as drawn.",
			Symbol:    "forced-colors",
			Scope:     "web/src",
			Gap:       "The neo-brutalist direction leans on a hand-drawn line; a forced-colors pass would have to state what survives.",
		},
		{
			ID:        "no-contrast-assertion-in-the-ui-tests",
			Statement: "No browser test asserts a contrast ratio, so the AA floor is a claim this repository cannot regress.",
			Symbol:    "getContrastRatio",
			Scope:     "web/e2e",
			Gap:       "The only contrast helper in the web tree is the accent gate inside theme.ts.",
		},
		{
			ID:        "no-second-theme-factory",
			Statement: "No alternative theme factory or skin registry exists, so a prototype cannot silently become a supported option.",
			Symbol:    "createAppThemeSkin",
			Scope:     "web/src",
			Gap:       "TR-F029 rules out a theme marketplace; a bounded prototype has to stay outside the product for that reason.",
		},
	}
}

// locate returns the 1-based line a symbol first appears on in code, or 0. The
// match is bounded: Chip cannot be proved by ChipLabel, and a symbol that starts
// or ends with punctuation is matched as written.
func locate(lines []string, symbol string) int {
	needle := strings.TrimSpace(symbol)
	for i, l := range lines {
		if containsBounded(l, needle) {
			return i + 1
		}
	}
	return 0
}

// isIdent reports whether b can continue a JavaScript identifier.
func isIdent(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || b == '_' || b == '$'
}

// containsBounded reports whether text holds needle with identifier edges.
func containsBounded(text, needle string) bool {
	if needle == "" {
		return false
	}
	needLeft := isIdent(needle[0])
	needRight := isIdent(needle[len(needle)-1])
	for at := 0; at <= len(text)-len(needle); {
		j := strings.Index(text[at:], needle)
		if j < 0 {
			return false
		}
		pos := at + j
		leftOK := !needLeft || pos == 0 || !isIdent(text[pos-1])
		rightOK := !needRight || pos+len(needle) >= len(text) || !isIdent(text[pos+len(needle)])
		if leftOK && rightOK {
			return true
		}
		at = pos + 1
	}
	return false
}

// VerifyClaims checks the catalog only - the statements and the negative claims -
// without touching the palette, so a partial tree still reports instead of
// failing the way a build would.
func VerifyClaims(root string) (Report, error) {
	var report Report
	for _, claim := range Claims() {
		finding := Finding{Claim: claim, Note: claim.Note}
		//nolint:gosec // G304: paths from this package's own catalog, under the repository root.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(claim.File)))
		if err != nil {
			finding.Status = StatusBroken
			finding.Evidence = "file unreadable"
			report.Findings = append(report.Findings, finding)
			continue
		}
		lines := strings.Split(blankComments(string(raw)), "\n")
		if at := locate(lines, claim.Symbol); at > 0 {
			finding.Status = StatusConfirmed
			finding.Evidence = fmt.Sprintf("%s:%d", claim.File, at)
		} else {
			finding.Status = StatusBroken
			finding.Evidence = fmt.Sprintf("symbol %q not found on a code line in %s", claim.Symbol, claim.File)
		}
		report.Findings = append(report.Findings, finding)
	}

	for _, absence := range Absences() {
		found, err := scanFor(root, absence.Scope, absence.Symbol)
		if err != nil {
			return Report{}, err
		}
		status := StatusConfirmed
		if len(found) > 0 {
			status = StatusVacated
		}
		report.Absences = append(report.Absences, AbsenceFinding{Absence: absence, Status: status, Hits: found})
	}
	return report, nil
}

// Verify reads the files the claims cite and checks each one, then measures the
// skin and the states it draws.
func Verify(root string) (Report, error) {
	report, err := VerifyClaims(root)
	if err != nil {
		return Report{}, err
	}
	theme, err := LoadTheme(root)
	if err != nil {
		return Report{}, err
	}
	report.Theme = theme
	ms, err := Measure(theme, States())
	if err != nil {
		return Report{}, err
	}
	report.States = ms
	report.Summary = Summarize(ms)

	variant := InkMax()
	applied, err := Apply(theme, variant)
	if err != nil {
		return Report{}, err
	}
	report.Variant = applied
	report.Skin = variant
	vm, err := Measure(applied, States())
	if err != nil {
		return Report{}, err
	}
	report.VariantStates = vm
	report.VariantSummary = Summarize(vm)
	report.NewColors = NewColorsIn(applied, theme)

	report.Surface, err = MeasureSurface(root)
	if err != nil {
		return Report{}, err
	}
	report.Totals = Totals{
		Claims:          len(report.Findings),
		Absences:        len(report.Absences),
		States:          len(report.States),
		Pass:            report.Summary.Pass,
		Fail:            len(report.Summary.Fail),
		Held:            report.Summary.Held,
		Exempt:          report.Summary.Exempt,
		FailHeld:        len(report.Summary.FailHeld),
		VariantPass:     report.VariantSummary.Pass,
		VariantFail:     len(report.VariantSummary.Fail),
		VariantFailHeld: len(report.VariantSummary.FailHeld),
		Repoints:        len(variant.Repoints),
		NewColors:       len(report.NewColors),
		TokensLight:     len(theme.Light.Tokens),
		TokensDark:      len(theme.Dark.Tokens),
		HexInTheme:      report.Surface.HexInTheme,
		HexOutsideTheme: report.Surface.HexOutsideTheme,
		Stylesheets:     report.Surface.Stylesheets,
		Undecorated:     len(theme.UndeclaredStatusLabels("light")),
	}
	return report, nil
}

// Totals are the numbers the memo quotes, counted from the slices rather than
// written down beside them.
type Totals struct {
	// Claims is how many statements were checked.
	Claims int
	// Absences is how many negative claims were checked.
	Absences int
	// States is how many color pairs were measured.
	States int
	// Pass and Fail split the shipped skin at their WCAG floors.
	Pass, Fail int
	// Held and Exempt split the measured pairs by whether AA judges them at all.
	Held, Exempt int
	// FailHeld counts the shipped pairs that are actual AA violations, and
	// VariantFailHeld counts the ones the prototype skin still misses.
	FailHeld, VariantFailHeld int
	// VariantPass and VariantFail split the prototype skin the same way.
	VariantPass, VariantFail int
	// Repoints is how many tokens the prototype skin re-aims.
	Repoints int
	// NewColors is how many colors the prototype skin introduces.
	NewColors int
	// TokensLight and TokensDark count the declared tokens per mode.
	TokensLight, TokensDark int
	// HexInTheme and HexOutsideTheme count literal colors inside and outside the
	// token file.
	HexInTheme, HexOutsideTheme int
	// Stylesheets counts the files a CSS fork would have to live in.
	Stylesheets int
	// Undecorated counts the palette groups that declare no label color.
	Undecorated int
}

// Report is everything one verification pass learned.
type Report struct {
	// Findings holds each claim verdict.
	Findings []Finding
	// Absences holds each negative verdict.
	Absences []AbsenceFinding
	// Theme is the skin as shipped.
	Theme Theme
	// States are the measurements of the shipped skin.
	States []Measurement
	// Summary reduces States.
	Summary Summary
	// Variant is the prototype skin, built from existing tokens only.
	Variant Theme
	// Skin is the prototype as declared, kept beside the applied theme so a report
	// can name what it asked for rather than only what resulted.
	Skin Skin
	// VariantStates are the same measurements against the prototype skin.
	VariantStates []Measurement
	// VariantSummary reduces VariantStates.
	VariantSummary Summary
	// NewColors lists colors the prototype skin introduced; it is expected to be
	// empty, and a non-empty list is the finding.
	NewColors []string
	// Surface is the measured color vocabulary of web/.
	Surface Surface
	// Totals are the quoted numbers.
	Totals Totals
}

// Verdict returns the finding of one claim ID.
func (r Report) Verdict(id string) Finding {
	for _, f := range r.Findings {
		if f.Claim.ID == id {
			return f
		}
	}
	return Finding{Claim: Claim{ID: id}, Status: StatusBroken, Evidence: "claim not checked"}
}

// scanFor searches a scope for a symbol on code lines, skipping this package so a
// prototype cannot refute or confirm itself.
func scanFor(root, scope, symbol string) ([]string, error) {
	base := filepath.Join(root, filepath.FromSlash(scope))
	if stat, err := os.Stat(base); err == nil && !stat.IsDir() {
		//nolint:gosec // G304: scope from this package's own catalog, resolved under the repository root.
		raw, err := os.ReadFile(base)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(blankComments(string(raw)), "\n")
		if locate(lines, symbol) > 0 {
			return []string{fmt.Sprintf("%s:%d", scope, locate(lines, symbol))}, nil
		}
		return nil, nil
	}
	var hits []string
	err := filepath.WalkDir(base, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if strings.HasPrefix(rel, "internal/mangaskin/") {
			return nil
		}
		if filepath.Ext(rel) != ".ts" && filepath.Ext(rel) != ".tsx" && filepath.Ext(rel) != ".go" {
			return nil
		}
		//nolint:gosec // G304: path handed to the walk callback by filepath.WalkDir itself.
		raw, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		lines := strings.Split(blankComments(string(raw)), "\n")
		if at := locate(lines, symbol); at > 0 {
			hits = append(hits, fmt.Sprintf("%s:%d", rel, at))
			sort.Strings(hits)
			if len(hits) >= maxHits {
				return filepath.SkipAll
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hits, nil
}
