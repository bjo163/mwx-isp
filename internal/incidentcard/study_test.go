package incidentcard

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", wd)
		}
		dir = parent
	}
}

func mustRun(t *testing.T, root string) *Result {
	t.Helper()
	res, err := Run(root)
	if err != nil {
		t.Fatalf("Run(%s): %v", root, err)
	}
	return res
}

// TestEveryStatementPinsToCode keeps the memo honest: a claim that lost its file or
// its symbol has to fail the run, not print as a sentence nobody checked.
func TestEveryStatementPinsToCode(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	ok, broken, vacated := statusCounts(res.Claims)
	if broken != 0 || vacated != 0 {
		for _, c := range res.Claims {
			if c.Status != "confirmed" {
				t.Errorf("%s %q: %s in %s", c.ID, c.Text, c.Status, c.File)
			}
		}
	}
	if ok != len(res.Claims) {
		t.Errorf("%d of %d statements confirmed", ok, len(res.Claims))
	}
	if ok < len(claimSpecs()) {
		t.Errorf("expected at least %d statements, got %d", len(claimSpecs()), ok)
	}
}

// TestNoAllowedSlotNamesASubscriber is the third clause of the criterion, checked
// against the ledger rather than asserted about it: nothing a card may print is a
// key the shipped rules call sensitive, nothing is a column an identity table proved
// to reach the response, and no key that speaks of a named party prints unless every
// assignment behind it is traced to a row that is not the subscriber's.
func TestNoAllowedSlotNamesASubscriber(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	card := allowed(res.Slots)
	if len(card) == 0 {
		t.Fatal("the ledger allowed no slots at all, so the check is vacuous")
	}
	for _, s := range card {
		switch s.Class {
		case "Secret", "CustomerPII", "Address":
			t.Errorf("slot %s.%s classifies as %s but is on the card", s.Struct, s.Key, s.Class)
		}
		if s.From != "" {
			t.Errorf("slot %s.%s prints identity column %s", s.Struct, s.Key, s.From)
		}
		if s.Composed != "" {
			t.Errorf("slot %s.%s is prose the server composes at %s", s.Struct, s.Key, s.Composed)
		}
		if isNameShape(s.Key) {
			fills := fillsFor(res.Fills, s.Struct, s.Go)
			if len(fills) == 0 {
				t.Errorf("slot %s.%s prints a name no handler assignment accounts for", s.Struct, s.Key)
			}
			for _, fl := range fills {
				if fl.Subscriber {
					t.Errorf("slot %s.%s prints %s at %s, the subscriber's own row", s.Struct, s.Key, fl.source(), fl.where())
				}
				if !fl.Traced {
					t.Errorf("slot %s.%s prints a name read from %s at %s, which the study cannot trace",
						s.Struct, s.Key, fl.source(), fl.where())
				}
			}
		}
	}
	// The keys that can only ever name the subscriber must be refused, by name.
	for _, forbidden := range []string{"customer_name", "phone", "email", "address", "password", "realname"} {
		var hit *slot
		for i := range res.Slots {
			if res.Slots[i].Key == forbidden && res.Slots[i].Disposition == disprint {
				needle := res.Slots[i]
				hit = &needle
			}
		}
		if hit != nil {
			t.Errorf("%s reached the card from %s", forbidden, hit.Struct)
		}
	}
}

// TestTheNameDecisionRefusesTheSubscriberRow exercises the name clause on the case the
// repository does not contain: a name read straight off isp_customer. Without it the
// passing run would only show that the product happens not to print a customer name.
func TestTheNameDecisionRefusesTheSubscriberRow(t *testing.T) {
	subscriber := []filler{{Struct: "nocImpactScope", Go: "Name", Table: "isp_customer",
		Column: "name", Traced: true, Subscriber: true, File: "internal/adminapi/noc_impact.go", Line: 1}}
	if v := nameDecision(subscriber); v.OK || !strings.Contains(v.Why, "isp_customer.name") || v.Held == "" {
		t.Errorf("a name read from the customer row was accepted: %+v", v)
	}
	equipment := []filler{{Struct: "nocImpactScope", Go: "Name", Table: "net_nas",
		Column: "name", Traced: true, File: "internal/adminapi/noc_impact.go", Line: 329}}
	if v := nameDecision(equipment); !v.OK {
		t.Errorf("a name read from a NAS row was refused: %s", v.Why)
	}
	if v := nameDecision(nil); v.OK || v.Why == "" || v.Held == "" {
		t.Errorf("an unaccounted name was accepted, or refused without a reason: %+v", v)
	}
	untraced := []filler{{Struct: "nocImpactScope", Go: "Name", Table: "nocImpactSubscriptionRow",
		Column: "CustomerName", File: "internal/adminapi/noc_impact.go", Line: 467}}
	if v := nameDecision(untraced); v.OK || v.Held == "" {
		t.Errorf("a name carried in from another response object was accepted: %+v", v)
	}
	// The served run must actually exercise the shape, or these cases are fiction.
	res := mustRun(t, repoRoot(t))
	if res.Totals.NameSlots == 0 {
		t.Error("no name-shaped slot is served, so the name clause was measured against nothing")
	}
	if res.Totals.NamePrint == 0 {
		t.Error("no name-shaped slot printed, so the trace never had a passing case")
	}
}

// TestEveryRefusalNamesItsReason stops the ledger from refusing on vibes: each
// turned-away slot has to point at a class, a column, or a line of the product.
func TestEveryRefusalNamesItsReason(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	turned := refused(res.Slots)
	if len(turned) == 0 {
		t.Fatal("nothing was refused, so the privacy check proved nothing")
	}
	for _, s := range turned {
		switch {
		case s.Class == "Secret" || s.Class == "CustomerPII" || s.Class == "Address":
		case s.From != "" && s.Proven != "":
		case s.Composed != "":
		case s.Held != "":
		default:
			t.Errorf("%s.%s refused with no reason", s.Struct, s.Key)
		}
		if s.Note == "" {
			t.Errorf("%s.%s refused without a note", s.Struct, s.Key)
		}
		if s.Held != "" && !isNameShape(s.Key) {
			t.Errorf("%s.%s carries name-trace evidence but is not a name-shaped key", s.Struct, s.Key)
		}
		// A trace refusal that blames a line has to blame a line the run read.
		if strings.Contains(s.Held, ":") {
			found := false
			for _, fl := range res.Fills {
				if fl.where() == s.Held {
					found = true
				}
			}
			if !found {
				t.Errorf("%s.%s is held by %s, which is not one of the recorded assignments",
					s.Struct, s.Key, s.Held)
			}
		}
	}
}

// TestTheLedgerCoversEveryIncidentSource keeps the field set the size of the
// response types the product actually serves.
func TestTheLedgerCoversEveryIncidentSource(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	seen := map[string]bool{}
	for _, s := range res.Slots {
		seen[s.Struct] = true
	}
	for _, src := range incidentSources {
		if !seen[src.name] {
			t.Errorf("incident response type %s contributed no slot", src.name)
		}
	}
	if len(res.Slots) < len(incidentSources) {
		t.Errorf("%d slots across %d response types", len(res.Slots), len(incidentSources))
	}
}

// TestTheGrayTableCoversBothModes is the first clause of the criterion: the reading
// is per token per mode, and a mode with no rows would hide a failure.
func TestTheGrayTableCoversBothModes(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	if len(res.Gray) == 0 {
		t.Fatal("no token measured")
	}
	counts := map[string]int{}
	for _, g := range res.Gray {
		counts[g.Mode]++
	}
	for _, mode := range []string{"light", "dark"} {
		if counts[mode] < 18 {
			t.Errorf("mode %s measured %d tokens, expected every status shade", mode, counts[mode])
		}
	}
	if res.Totals.GrayDarkOK >= res.Totals.GrayLightOK {
		t.Errorf("the dark set (%d) did not read worse than the light set (%d); the sheet is the same paper for both",
			res.Totals.GrayDarkOK, res.Totals.GrayLightOK)
	}
}

// TestAnIncidentSurfaceHasNoPrintPath is the second clause's honest half: the
// stale/confidence fields are served and rendered, and no print path carries them.
func TestAnIncidentSurfaceHasNoPrintPath(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	if len(res.Paper.IncidentFiles) == 0 {
		t.Fatal("no incident surface found, so the census is vacuous")
	}
	if len(res.Paper.IncidentWithPrint) != 0 {
		t.Errorf("incident surfaces with a print path: %v", res.Paper.IncidentWithPrint)
	}
	if res.Paper.Counts["page geometry declared"] != 0 {
		t.Errorf("the tree writes @page rules after all")
	}
	if res.Paper.Counts["backgrounds asked of the browser"]+
		res.Paper.Counts["backgrounds asked, webkit spelling"] != 0 {
		t.Errorf("a print-color-adjust rule exists, so the ink census has to be re-read")
	}
	for _, v := range res.Visible {
		if v.OnPaper != 0 {
			t.Errorf("field %s reaches paper through a print path that the census says does not exist", v.Field)
		}
	}
}

// TestTheStaleAndConfidenceSlotsAreInTheLedger keeps the middle clause from being a
// claim about a UI nobody measured: each field has to be served and counted.
func TestTheStaleAndConfidenceSlotsAreInTheLedger(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	want := []string{"state", "confidence", "confidence_reason", "generated_at",
		"stale", "stale_after_seconds", "last_checked_at", "interval_seconds"}
	have := map[string]int{}
	for _, v := range res.Visible {
		have[v.Field] = v.Rendered
	}
	for _, name := range want {
		if _, ok := have[name]; !ok {
			t.Errorf("%s never reached the visibility table", name)
		}
	}
	total := 0
	for _, n := range have {
		total += n
	}
	if total == 0 {
		t.Error("no stale or confidence field is read anywhere in web/src")
	}
}

// TestTheCardFitsTheSheetItClaims is the PoC's own arithmetic: the page budget has
// to be derived from the skin, and the fit reported whether or not it passes.
func TestTheCardFitsTheSheetItClaims(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	if res.Fit.RowsWanted != len(allowed(res.Slots)) {
		t.Errorf("the budget planned %d rows against %d allowed slots", res.Fit.RowsWanted, len(allowed(res.Slots)))
	}
	if res.Fit.RowHeightPx <= 0 || res.Fit.BudgetPx <= 0 {
		t.Errorf("page budget from parsed metrics: row %.1f, budget %.1f", res.Fit.RowHeightPx, res.Fit.BudgetPx)
	}
	if res.Metrics.RootFontPx != 13 {
		t.Errorf("the skin's root font size moved to %v; the memo's px numbers depend on it", res.Metrics.RootFontPx)
	}
}

// TestTheTotalsAgreeWithTheTables catches a rendered table and a printed count that
// drifted apart.
func TestTheTotalsAgreeWithTheTables(t *testing.T) {
	res := mustRun(t, repoRoot(t))
	t1 := res.Totals
	if t1.Slots != len(res.Slots) || t1.Allowed != len(allowed(res.Slots)) {
		t.Errorf("slot totals %d/%d disagree with the ledger %d/%d",
			t1.Allowed, t1.Slots, len(allowed(res.Slots)), len(res.Slots))
	}
	if t1.RefusedRule+t1.RefusedQuery+t1.RefusedProse+t1.RefusedTrace != len(refused(res.Slots)) {
		t.Errorf("refusal reasons %d+%d+%d+%d do not add up to the %d refused slots",
			t1.RefusedRule, t1.RefusedQuery, t1.RefusedProse, t1.RefusedTrace, len(refused(res.Slots)))
	}
	if t1.RefusedTrace == 0 {
		t.Error("the name trace refused nothing, so the fourth check was never exercised")
	}
	if t1.RefusedGuarded+t1.RefusedPastRules != len(refused(res.Slots)) {
		t.Errorf("rule coverage split %d+%d does not add up to the %d refused slots",
			t1.RefusedGuarded, t1.RefusedPastRules, len(refused(res.Slots)))
	}
	if t1.RefusedPastRules == 0 {
		t.Error("no refusal reaches past a shipped rule, so the ledger is a copy of redact and proves nothing")
	}
	if t1.PaperFiles != len(res.Paper.Sites) || t1.Incidents != len(res.Paper.IncidentFiles) {
		t.Errorf("paper totals %d/%d disagree with %d/%d rows",
			t1.PaperFiles, t1.Incidents, len(res.Paper.Sites), len(res.Paper.IncidentFiles))
	}
	if t1.GrayLight+t1.GrayDark != len(res.Gray) {
		t.Errorf("grey totals %d cover %d rows", t1.GrayLight+t1.GrayDark, len(res.Gray))
	}
	if t1.Claims != len(res.Claims) || t1.Gaps != len(res.Gaps) {
		t.Errorf("statement and gap counts drifted: %d/%d against %d/%d rows",
			t1.Claims, t1.Gaps, len(res.Claims), len(res.Gaps))
	}
}

// TestTheDigestSurvivesATickingClock keeps the memo quotable: the run is a function
// of the tree, so nothing measured here may read the clock.
func TestTheDigestSurvivesATickingClock(t *testing.T) {
	root := repoRoot(t)
	first := mustRun(t, root)
	second := mustRun(t, root)
	if first.Digest != second.Digest {
		t.Errorf("digest moved between runs: %s then %s", first.Digest, second.Digest)
	}
	if len(first.Digest) != 64 {
		t.Errorf("digest %q is not a sha256 hex", first.Digest)
	}
}

// TestTheStudyWritesNothing is the read-only promise, checked by hashing the trees
// it reads before and after a run.
func TestTheStudyWritesNothing(t *testing.T) {
	root := repoRoot(t)
	before := treeHash(t, root)
	mustRun(t, root)
	after := treeHash(t, root)
	if before != after {
		t.Errorf("the run changed the tree: %s then %s", before, after)
	}
}

func treeHash(t *testing.T, root string) string {
	t.Helper()
	h := sha256.New()
	for _, dir := range []string{"internal", "web/src", "docs"} {
		base := filepath.Join(root, filepath.FromSlash(dir))
		err := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d == nil || d.IsDir() {
				return nil
			}
			if d.Name() == "node_modules" || d.Name() == "dist" {
				return filepath.SkipDir
			}
			rel, rerr := filepath.Rel(root, p)
			if rerr != nil {
				return nil
			}
			info, serr := d.Info()
			if serr != nil {
				return nil
			}
			_, _ = fmt.Fprintf(h, "%s\t%d\t%v\n", filepath.ToSlash(rel), info.Size(), info.Mode().Perm())
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return hexOf(h.Sum(nil))
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	var out strings.Builder
	for _, c := range b {
		out.WriteByte(digits[c>>4])
		out.WriteByte(digits[c&0xf])
	}
	return out.String()
}

// TestNothingImportsTheStudy keeps the prototype out of the product.
func TestNothingImportsTheStudy(t *testing.T) {
	root := repoRoot(t)
	res := mustRun(t, root)
	if len(res.Imports) == 0 {
		t.Fatal("no imports parsed, so the read-only check is vacuous")
	}
	got := importersOutside(root)
	if len(got) != 0 {
		t.Errorf("product files import the study: %v", got)
	}
}

// TestTheImportAllowlistIsNotVacuous is the read-only promise in one line: this
// package may parse, hash and read, and nothing else.
func TestTheImportAllowlistIsNotVacuous(t *testing.T) {
	root := repoRoot(t)
	imp := packageImports(root, ownPackageDir(root))
	if len(imp) == 0 {
		t.Fatal("no imports parsed, so the allowlist is vacuous")
	}
	allowedSet := map[string]bool{
		"crypto/sha256": true, "encoding/hex": true, "fmt": true,
		"go/ast": true, "go/parser": true, "go/token": true,
		"io/fs": true, "math": true, "os": true, "path/filepath": true,
		"regexp": true, "sort": true, "strconv": true, "strings": true,
		"testing": true,
	}
	for _, i := range imp {
		if !allowedSet[i] {
			t.Errorf("unexpected import %s", i)
		}
	}
}

// TestTheCommentBlankerKeepsLineNumbers protects every pin in the study: a blanker
// that swallowed a line would move every citation below it.
func TestTheCommentBlankerKeepsLineNumbers(t *testing.T) {
	src := "// one\nconst x = 1 /* two\nthree */ + 2 // four\nfunc main() {}\n"
	got := blankComments(src)
	if n := len(strings.Split(src, "\n")); len(strings.Split(got, "\n")) != n {
		t.Fatalf("line count changed: %d to %d", n, len(strings.Split(got, "\n")))
	}
	for i, ln := range strings.Split(got, "\n") {
		if strings.Contains(ln, "//") || strings.Contains(ln, "/*") || strings.Contains(ln, "one") ||
			strings.Contains(ln, "four") {
			t.Errorf("line %d still carries a comment: %q", i+1, ln)
		}
	}
	if !strings.Contains(got, "const x = 1") || !strings.Contains(got, "func main()") {
		t.Errorf("code was blanked with the comments: %q", got)
	}
}

// TestTheRedactionRulesAreParsedNotCopied proves the study inherits the product's
// policy: a key the shipped rules name must classify, and one they never name must
// fall through to the default branch rather than to a friendlier answer.
func TestTheRedactionRulesAreParsedNotCopied(t *testing.T) {
	root := repoRoot(t)
	rules, err := loadRules(root)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]string{
		"customer_name":    "CustomerPII",
		"technician_phone": "CustomerPII",
		"email":            "CustomerPII",
		"home_address":     "CustomerPII",
		"password":         "Secret",
		"nas_ip":           "Address",
		"generated_at":     "Plain",
		"odp_code":         "Plain",
	}
	for key, want := range cases {
		if got := rules.Classify(key); got != want {
			t.Errorf("Classify(%q) = %s, want %s", key, got, want)
		}
	}
	for _, name := range rules.listNames() {
		if len(rules.Lists[name]) == 0 {
			t.Errorf("fragment list %s parsed empty", name)
		}
	}
	if len(rules.Groups) < 3 {
		t.Errorf("only %d classifier branches parsed", len(rules.Groups))
	}
}

// TestTheStudyParsesNoCustomerData keeps the prototype honest about its own inputs:
// nothing in this package holds a subscriber value, because it never reads one.
func TestTheStudyReadsNoDatabase(t *testing.T) {
	root := repoRoot(t)
	for _, rel := range walkFiles(root, "internal/"+ownPackage, ".go") {
		for _, ln := range lines(root, rel) {
			for _, banned := range []string{"gorm.Open", "sql.Open", "os.WriteFile", "http.Get"} {
				if strings.Contains(ln, banned) {
					t.Errorf("%s calls %s", rel, banned)
				}
			}
		}
	}
}
