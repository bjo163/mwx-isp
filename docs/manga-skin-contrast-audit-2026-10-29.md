# FUT-017 — Can a second visual direction be prototyped from the tokens that already exist?

Study date: 2026-10-29. Repository: `mwx-isp`, branch `dev`, HEAD `bacd23f`.
Backlog row: `FUT-017` (`TR-F029`, P3, RISET, dependency `ENG-053` delivered) in
[MWX-ISP-audit-backlog-2026-10-27.md](MWX-ISP-audit-backlog-2026-10-27.md).
Prototype: `internal/mangaskin` (read-only, imported by nothing).
Criterion under test: **Contrast WCAG AA pada state penting, toggle rollback ke
theme asli tanpa CSS fork.**

## The question

The card asks for a prototype of one funky/grunge visual direction built with
existing tokens. Reading `web/src/theme.ts` first, as the card's own microtask 1
requires, changes what the question can be: the direction already shipped *is* a
neo-brutalist manga-ink system - an ink line draws every panel, a hard offset
shadow lifts it, elevation is pinned to zero, titles are uppercased condensed
display type, and the body carries a printed-paper halftone. `TR-F029` names that
direction as the sanctioned one ("shared design tokens/components provide a
professional manga-ink direction") and rules out the alternative the card could
otherwise mean: "no arbitrary CSS, theme/plugin marketplace".

So there is no second direction to invent inside this scope, and the card allows
the other outcome: *"consider a no-change outcome if the current implementation
already satisfies the criterion, and document why."* The answerable form of the
criterion is therefore two measurements plus one bounded experiment:

1. Does the shipped skin hold WCAG AA on the states that carry operational
   meaning, in both modes?
2. Can it be rolled back without forking a stylesheet - measured, not asserted?
3. Where it misses, can the miss be closed by re-aiming tokens the file already
   declares, without introducing a single new color?

Everything the criterion asks that is *not* answerable from the tree - a human
preference judgment, a rendered screenshot pass, a font license - is recorded on
the no-go register with the asset it needs, not converted into an inference.

## What counts as evidence

The same three rules the `ENG-006`, `ENG-007` and `ECO-006` studies used, enforced
by tests in `internal/mangaskin`:

- **A citation is a code line.** Comment-only lines are blanked before a symbol is
  matched, so a doc comment naming `inkShadow` proves the comment exists, not the
  shadow (`TestCommentMentionsAreNotEvidence`, `TestEveryCitationLandsOnACodeLine`).
- **A whole identifier.** Word-bounded matching, so `ChipLabel` cannot prove `Chip`
  (`TestBoundedMatchCannotBeProvedByALongerName`).
- **Drift is reported, never dropped.** A moved file is `broken`, a gap that got
  filled is `vacated` (`TestMovedFileIsReportedBrokenNotDropped`,
  `TestVacatedGapIsReportedWithItsHit`).

Three rules specific to a study whose subject is color:

- **No color from outside the file.** Every pair in the contrast table is either a
  literal `web/src/theme.ts` writes or an alpha it writes, resolved through the
  palette parse - a palette study of colors the console never draws would pass
  these tests and fail this one (`TestTheMeasuredPairsComeFromTheTokenFile`).
- **A number is only quoted for a pair a component renders.** Each state names the
  file and symbol that draw it (`TestEveryStateResolvesInBothModes`).
- **Nothing may be claimed as shipped product work.** No statement may read as
  though a skin was deployed, an operator was consulted or a price was set
  (`TestNoClaimPromisesAShippedSkin`, `TestNoClaimPromisesAShippedSkin` also covers
  the negative catalog).

The absence scan skips `internal/mangaskin/` itself, so the prototype cannot
refute or confirm its own gaps (`TestAbsenceScanIgnoresThePrototypeItself`).

## The prototype

| File | Role |
| --- | --- |
| `internal/mangaskin/wcag.go` | sRGB luminance, contrast ratio, the AA floors, the large-text cut-off, alpha compositing |
| `internal/mangaskin/theme.go` | parses `web/src/theme.ts` into palettes and derived values; measures the color vocabulary of `web/` |
| `internal/mangaskin/states.go` | the 22 operational states, each pinned to the component that draws it, and their measurements |
| `internal/mangaskin/claims.go` | the direction catalog (14 claims), the gap catalog (5 absences), verification |
| `internal/mangaskin/variant.go` | token re-aimings, applied to a copy; proves no new color appears |
| `internal/mangaskin/render.go` | the report below, rendered from the same slices the tests assert on |

## Verdict

**Delivered, with the criterion met on both halves and one real miss named.**

- Direction claims verified: **14/14**. Gaps still open: **5/5**.
- Color pairs measured: **44** - 40 the standard judges, 4 it exempts.
- AA violations on the shipped skin: **1** - the dark-mode panel outline at
  **2.34:1** against the 3:1 non-text floor (1.4.11).
- After the two token re-aimings: **0** AA violations, and **0 colors introduced**.
- Rollback: **0** stylesheets exist under `web/` to fork, **1** theme factory is
  defined (3 call sites), both modes come from it, and the branding screen already
  exposes a restore-to-defaults action.

The second half of the criterion is therefore met by construction rather than by
argument: with no stylesheet in the tree, a skin cannot be a CSS fork, and rolling
back is a mode change plus one button.

## The direction is already shipped

The claims table is the evidence for that reading, each row pinned to a line:

| Claim | Verdict | Evidence | Statement |
| --- | --- | --- | --- |
| direction-is-already-manga-ink | confirmed | web/src/theme.ts:73 | The shipped direction is a manga panel language: an ink line draws every panel and a hard offset shadow lifts it, with no elevation blur. |
| brushes-are-alpha-steps-of-the-ink | confirmed | web/src/theme.ts:72 | The outline, the shadow and the hover wash are transparency steps of the body text color, not extra hues the skin introduces. |
| paper-gets-a-halftone-wash | confirmed | web/src/theme.ts:76 | The body and the card headers carry a printed-paper halftone, drawn as a radial dot of the same ink. |
| titles-are-capitalized-display-type | confirmed | web/src/theme.ts:87 | Panel titles, tabs and headings above h5 are set in a condensed display stack at weight 900 and uppercased. |
| corners-stay-square | confirmed | web/src/theme.ts:83 | The shape radius is 3px, so the brutalist read comes from line and shadow rather than from rounded cards. |
| density-keeps-every-label-at-normal-text | confirmed | web/src/theme.ts:86 | Body copy is 13px, so no status label in the skin clears the WCAG large-text cut-off and every one of them is judged at 4.5:1. |
| one-accent-per-deployment | confirmed | web/src/theme.ts:56 | A deployment overrides exactly one brand value, the accent, and the factory recomputes its light and dark shades from it. |
| the-accent-override-is-contrast-gated | confirmed | web/src/theme.ts:42 | A picked accent is stepped until it reaches 4.5:1 against the panel before the theme is built, and falls back to a black or white ink when no step works. |
| status-colors-stay-outside-the-brand | confirmed | web/src/theme.ts:10 | success, warning, error and info are declared as palette groups of their own, so an accent change cannot repaint an operational state. |
| both-modes-come-from-one-factory | confirmed | web/src/App.tsx:24 | The app builds its light and dark themes from the same factory call, so switching is a state change rather than loading a second stylesheet. |
| the-deployment-can-return-to-the-declared-defaults | confirmed | web/src/pages/BrandingPage.tsx:227 | The branding screen exposes a restore-to-defaults action, so a changed accent can be rolled back by the operator without a code change. |
| the-status-ink-is-a-palette-token-not-a-local-hex | confirmed | web/src/pages/AccountSettings.tsx:392 | The live/disabled read is taken from the palette by name, so a skin change reaches it without editing the component. |
| the-chrome-is-ink-outlined-not-shadowed | confirmed | web/src/theme.ts:175 | The app bar and the drawer are separated by a 2px ink border instead of a drop shadow. |
| motion-is-respected-not-relied-on | confirmed | web/src/theme.ts:119 | The skin ships a prefers-reduced-motion override, so its hover shift is not the only carrier of feedback. |

## The gaps that are still gaps

Negative claims, checked the same way. A `confirmed` row here is a fact the
criterion depends on, not a defect.

| Gap | Verdict | Found at | What closing it would mean |
| --- | --- | --- | --- |
| no-stylesheet-to-fork | confirmed | none | The entire visual system lives in one theme object; a fork would have to create the file it overrides. |
| no-css-custom-properties | confirmed | none | Token identity is the palette object alone, which is what makes a re-pointing variant provable. |
| no-forced-colors-support | confirmed | none | The neo-brutalist direction leans on a hand-drawn line; a forced-colors pass would have to state what survives. |
| no-contrast-assertion-in-the-ui-tests | confirmed | none | The only contrast helper in the web tree is the accent gate inside theme.ts. |
| no-second-theme-factory | confirmed | none | TR-F029 rules out a theme marketplace; a bounded prototype has to stay outside the product for that reason. |

## Contrast on the states that matter

22 states, both modes: the account row, the paid invoice, the stale graph, the
delete confirmation, the directory hint, the empty value, the table head, the
selected nav item, the tooltip, the focus ring, the panel outline, the hard
shadow, the brand button, four filled chips and four soft fills. Sizes come from
`theme.ts` itself, and in a 13px console nothing reaches the large-text cut-off,
so every text row is judged at 4.5:1 and every meaningful line at 3:1.

| State | Mode | Foreground | Background | Ratio | Floor | Verdict |
| --- | --- | --- | --- | --- | --- | --- |
| account-enabled | dark | #62d990 | #17171a | 10.09:1 | 4.50:1 | text PASS |
| active-nav | dark | #0d0d0d | #22c55e | 8.53:1 | 4.50:1 | text PASS |
| brand-button | dark | #0d0d0d | #22c55e | 8.53:1 | 4.50:1 | text PASS |
| chip-error | dark | #0d0d0d | #f87171 | 7.03:1 | 4.50:1 | text PASS |
| chip-info | dark | #0d0d0d | #55cbd5 | 10.08:1 | 4.50:1 | text PASS |
| chip-success | dark | #0d0d0d | #62d990 | 10.96:1 | 4.50:1 | text PASS |
| chip-warning | dark | #0d0d0d | #fbbf24 | 11.64:1 | 4.50:1 | text PASS |
| destructive-action | dark | #f87171 | #17171a | 6.47:1 | 4.50:1 | text PASS |
| directory-hint | dark | #55cbd5 | #17171a | 9.28:1 | 4.50:1 | text PASS |
| empty-value | dark | #6f6c65 | #17171a | 3.41:1 | 4.50:1 | exempt FAIL |
| focus-ring | dark | #22c55e | #17171a | 7.85:1 | 3.00:1 | UI PASS |
| hard-shadow | dark | alpha(ink, 0.38) | #17171a | 3.27:1 | 3.00:1 | UI PASS |
| invoice-settled | dark | #62d990 | #17171a | 10.09:1 | 4.50:1 | text PASS |
| metric-stale | dark | #fbbf24 | #17171a | 10.72:1 | 4.50:1 | text PASS |
| panel-outline | dark | alpha(ink, 0.28) | #17171a | 2.34:1 | 3.00:1 | UI FAIL |
| row-divider | dark | rgba(243, 239, 228, 0.12) over #17171a | #17171a | 1.38:1 | 4.50:1 | exempt FAIL |
| table-heading | dark | #b5b1a5 | #1f1f23 | 7.66:1 | 4.50:1 | text PASS |
| tooltip-body | dark | #17171a | #f3efe4 | 15.57:1 | 4.50:1 | text PASS |
| wash-error | dark | #0d0d0d | #fca5a5 | 10.24:1 | 4.50:1 | text PASS |
| wash-info | dark | #0d0d0d | #84e0e7 | 12.77:1 | 4.50:1 | text PASS |
| wash-success | dark | #0d0d0d | #8ae8ad | 13.17:1 | 4.50:1 | text PASS |
| wash-warning | dark | #0d0d0d | #fcd34d | 13.48:1 | 4.50:1 | text PASS |
| account-enabled | light | #13783f | #fffdf6 | 5.45:1 | 4.50:1 | text PASS |
| active-nav | light | #ffffff | #5d7000 | 5.55:1 | 4.50:1 | text PASS |
| brand-button | light | #ffffff | #5d7000 | 5.55:1 | 4.50:1 | text PASS |
| chip-error | light | #ffffff | #b42318 | 6.57:1 | 4.50:1 | text PASS |
| chip-info | light | #ffffff | #087e8b | 4.81:1 | 4.50:1 | text PASS |
| chip-success | light | #ffffff | #13783f | 5.55:1 | 4.50:1 | text PASS |
| chip-warning | light | #ffffff | #925500 | 5.96:1 | 4.50:1 | text PASS |
| destructive-action | light | #b42318 | #fffdf6 | 6.46:1 | 4.50:1 | text PASS |
| directory-hint | light | #087e8b | #fffdf6 | 4.72:1 | 4.50:1 | text PASS |
| empty-value | light | #8a877d | #fffdf6 | 3.53:1 | 4.50:1 | exempt FAIL |
| focus-ring | light | #5d7000 | #fffdf6 | 5.45:1 | 3.00:1 | UI PASS |
| hard-shadow | light | alpha(ink, 0.85) | #fffdf6 | 12.79:1 | 3.00:1 | UI PASS |
| invoice-settled | light | #13783f | #fffdf6 | 5.45:1 | 4.50:1 | text PASS |
| metric-stale | light | #925500 | #fffdf6 | 5.86:1 | 4.50:1 | text PASS |
| panel-outline | light | alpha(ink, 0.7) | #fffdf6 | 7.32:1 | 3.00:1 | UI PASS |
| row-divider | light | rgba(13, 13, 13, 0.16) over #fffdf6 | #fffdf6 | 1.42:1 | 4.50:1 | exempt FAIL |
| table-heading | light | #4a4943 | #e9e4d6 | 7.11:1 | 4.50:1 | text PASS |
| tooltip-body | light | #fffdf6 | #0d0d0d | 19.10:1 | 4.50:1 | text PASS |
| wash-error | light | #0d0d0d | #d9483d | 4.58:1 | 4.50:1 | text PASS |
| wash-info | light | #0d0d0d | #10a6b5 | 6.61:1 | 4.50:1 | text PASS |
| wash-success | light | #0d0d0d | #24a862 | 6.34:1 | 4.50:1 | text PASS |
| wash-warning | light | #0d0d0d | #c27a12 | 5.63:1 | 4.50:1 | text PASS |

### The one violation, and why it is the interesting one

`panel-outline` in dark mode is 2.34:1. It is not a status color and it is not a
typography problem: it is the transparency the skin chose for its own signature.
`inkLine` is `alpha(ink, isDark ? 0.28 : 0.7)` - the light mode draws its panels
with a line at 0.7, the dark mode at 0.28, and only the dark side falls under the
non-text floor. The manga line language is the thing at risk, which is exactly why
the criterion asked for a measurement instead of a mood board.

### Measured, exempt

Four pairs are below 4.5:1 while WCAG 2.1 excludes them, and they come from two
states: the row hairline (1.38:1 dark, 1.42:1 light) is decoration rather than a
control, so 1.4.11 does not apply, and the empty value is inactive content, so
1.4.3 does not apply. They are
reported because "0 violations" and "nothing measured" look identical otherwise -
and the empty value earns its place: `text.disabled` at 3.41:1 is the one token
that has to carry both *"nothing here yet"* and *"you cannot touch this"*, and an
operator scanning for a missing field is not reading exempt content.

## The bounded experiment: two re-aimings, no new color

The prototype skin, `Ink Max`, changes exactly two things, both inside the
vocabulary the file already declares.

| Change | Value | Why |
| --- | --- | --- |
| token text.disabled | takes text.secondary | an empty value is the one thing an operator scans for, and the washed ink cannot carry it |
| alpha inkLine (dark) | alpha(ink, 0.7) | the light mode already draws inkLine at that weight |

Applying them moves three pairs and breaks nothing:

| State | Mode | Shipped | Ink Max | Verdict |
| --- | --- | --- | --- | --- |
| empty-value | dark | 3.41:1 | 8.35:1 | now passes 4.50:1 |
| empty-value | light | 3.53:1 | 8.87:1 | now passes 4.50:1 |
| panel-outline | dark | 2.34:1 | 8.08:1 | now passes 3.00:1 |

`NewColorsIn` compares every value in the applied skin against the token values
the base file declares and returns an empty list, which is the mechanical answer
to "via existing tokens": the direction was reachable without inventing one color,
one file or one rule. It also keeps the two background tokens and the four status
groups distinct, so raising a line does not collapse a meaning
(`TestThePrototypeKeepsEveryDeclaredColorDistinct`).

## Rollback without a CSS fork

| Question | Measured answer |
| --- | --- |
| What would a CSS fork fork? | 0 stylesheets under web/ |
| Theme factories defined | 1 |
| Theme factory call sites | 3 |
| Can a deployment return to the declared defaults? | confirmed |
| Colors the prototype introduces | none |

Two of those rows are the whole argument for `RISET` rather than implementation:
the product defines one factory, and nothing in the tree is a stylesheet. A variant
that stays this small is a token diff; a variant that wanted a fork would first
have to create the file it forks, and then it would be `TR-F029`'s forbidden
arbitrary CSS.

## What the token path does not reach

The measurement that matters most for a skin decision is not the palette, it is
how much of the UI ignores the palette.

| Measure | Value |
| --- | --- |
| Tokens declared per mode | light 26, dark 26 |
| Hex literals in theme.ts | 72 |
| Hex literals outside theme.ts | 170 across 84 files |
| Hand-picked `mode === 'dark' ?` branches | 61 |
| References to a status token | 128 |
| Files that call a contrast helper | web/src/theme.ts |
| Theme factory call sites | web/src/App.tsx, web/src/pages/BrandingPage.tsx, web/src/theme.ts |
| Palette groups with no label color | error, info, success, warning |

Ten files with the most literal colors of their own:

| File | Hex literals |
| --- | --- |
| web/src/pages/LandingPage.tsx | 119 |
| web/src/pages/WebsiteEditorPage.tsx | 21 |
| web/src/components/MrtgTrafficGraph.tsx | 10 |
| web/src/pages/OperationsPage.tsx | 7 |
| web/src/components/Enterprise.tsx | 5 |
| web/src/pages/CustomerPortalPage.tsx | 3 |
| web/src/pages/IPAMPage.tsx | 2 |
| web/src/branding/BrandingContext.tsx | 1 |
| web/src/pages/AuthTest.tsx | 1 |
| web/src/pages/DebugAccountSettings.tsx | 1 |

`web/src/theme.ts` holds 72 literals and 26 tokens per mode. The rest of `web/src`
holds 170 literals of its own across 84 files, and **119 of them are
`LandingPage.tsx` alone**. 61 places branch on `mode === 'dark' ?` by hand instead
of asking the palette. The live/disabled read that the claims table cites is the
good citizen; the landing page and the website editor are a second, unthemed
visual system that no token re-aiming reaches.

Two consequences, both findings rather than defects of this study:

- A skin decision scoped to tokens is a decision about the console, not about the
  public marketing surfaces. Saying "the skin changed" while `LandingPage.tsx`
  keeps its own 119 literals would be the stale claim the criterion warns about.
- `success`, `warning`, `error` and `info` declare no `contrastText`. The label
  over a filled status chip is chosen by MUI, not by the skin, and the measurement
  shows the choice is mode-dependent: in dark mode the paper ink `#f3efe4` sits at
  1.26-2.41:1 on those fills while `#0d0d0d` reaches 7.03-11.64:1; in light mode
  the two inks are 2.96-4.04:1 apart from each other, and only `#ffffff` passes.
  One of the two inks is safe per mode and nothing in the theme records which.

## No-go register

| No-go | Blocking asset | Why it is not answerable here |
| --- | --- | --- |
| Rendering the console and measuring contrast in the browser | a running admin session with seeded accounts, invoices and metrics, plus a screenshot pass | every number here is computed from the token file and the components' declared sizes; MUI resolves the final DOM color, and `web/e2e` asserts no ratio (`no-contrast-assertion-in-the-ui-tests`) |
| Applying either re-aiming to `web/src/theme.ts` | a `TR-F029` scope decision from the owner | the dark panel line at 0.7 is a product change to the sanctioned direction; a RISET row cannot make it |
| "Is the stronger line better?" - operator preference | human respondents, and the instrument to sample them | the study measures floors, not taste; no claim in this memo stands in for an operator |
| A display webfont for the condensed title stack | a licensing and self-hosting decision | `fontStacks.display` falls back to `Arial Narrow` / `Impact`, which differ per OS; a bundled face changes the rendered weights the large-text cut-off depends on |
| A forced-colors / high-contrast pass | a decision that the direction survives Windows high-contrast at all | nothing in the tree honors `forced-colors` (`no-forced-colors-support`), so the honest answer is "unknown", not "works" |

## Decisions

1. **No new skin directory.** `TR-F029` sanctions one manga-ink direction; the
   card's own microtask prefers a documented no-change outcome over an invented
   artifact, and the claims table shows the shipped direction already is the
   funky/grunge one.
2. **The dark `inkLine` 0.28 is the one open defect this study can name.** It is
   below 1.4.11, and the fix is one number the same line already writes for light
   mode. It is not applied here: a research package does not edit product code.
3. **`text.disabled` needs a second token, not a bigger contrast.** The empty-value
   row is exempt but read; the fix is a distinct "unfilled" ink, which is a design
   decision beyond a re-aiming.
4. **Status labels need a declared `contrastText`.** Four groups left to MUI's
   choice is a decision nobody recorded; declaring it is a `TR-F029` question, not
   a skin question.
5. **The landing page is out of skin scope.** 119 literals in one file are a
   separate theming decision; measuring it here keeps a future "the skin changed"
   claim from silently covering `/home`.
6. **`TR-N001`/`TR-N002` untouched.** This is a presentation study of the existing
   operator shell; no new service, endpoint, schema or reporting surface.

## What this means for the portfolio

`FUT-017` closes as delivered-with-scope: the AA half is measured across 44 pairs
with one real violation and one bounded, token-only repair; the rollback half is
met structurally at 0 stylesheets and 1 factory. The preference, browser-verification
and licensing halves stay on the no-go register. `FUT-018` (printable incident-card
visuals) depends on this row and inherits the same posture: measure what the token
file declares, and stop before a print rendering someone has to look at.

## Verification

- `go build -p 1 ./...` - clean.
- `go vet -p 1 ./internal/mangaskin/` - clean.
- `go test -p 1 -count=1 ./internal/mangaskin/` - 33 tests, all passing. The
  tables above are transcribed from `TestReportMarkdownDump`, which renders the
  report and logs it; `TestRenderMarkdownNumbersComeFromTheRun` pins the verdict
  sentence to the measured counts, and `TestRenderIsDeterministic` pins the text
  against map-iteration drift.
- `golangci-lint run ./internal/mangaskin/...` - 0 issues.
- `grep -rn mangaskin --include=*.go internal/ cmd/` outside the package - 0 hits.

## Residual limits

- Ratios are computed from the token file and the sizes `theme.ts` declares. Where
  MUI picks a shade itself (a soft fill, a chip label), the pair measured is the
  declared token, not necessarily the rendered pixel; both declared shades are
  measured so no fill escapes the audit, and the browser pass stays on the no-go
  register.
- The state list is a curated reading of "state penting": 22 rows chosen for
  operational meaning, not an enumeration of every color pair in the app. A new
  status color that is not in `States()` would not be measured - the fix is to add
  the row, and the citation tests would force it to name a component.
- The 0-stylesheet finding is a property of this tree today. A single imported CSS
  file would falsify `no-stylesheet-to-fork`, and the absence scan reports it as
  `vacated` with the location rather than letting the rollback argument go quiet.
- Nothing here measures operator preference, brand fit or whether a stronger panel
  line looks better. Those are exactly the five rows on the no-go register.
