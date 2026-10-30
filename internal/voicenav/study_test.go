package voicenav

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

// repoRoot walks up from the package directory to the module root, so the study
// always reads the tree it lives in.
func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.ToSlash(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above %s", wd)
		}
		dir = parent
	}
}

var (
	runOnce   *Result
	runErr    error
	studyOnce sync.Once
)

func study(t *testing.T) *Result {
	t.Helper()
	studyOnce.Do(func() { runOnce, runErr = Run(repoRoot(t)) })
	if runErr != nil {
		t.Fatalf("Run: %v", runErr)
	}
	return runOnce
}

// TestEveryStatementHolds is the gate the protocol asks for: a memo sentence that no
// longer lands on a code line fails the build instead of staying in print.
func TestEveryStatementHolds(t *testing.T) {
	res := study(t)
	if len(res.Claims) == 0 {
		t.Fatal("no statements to check")
	}
	for _, c := range res.Claims {
		if c.Status != confirmed {
			t.Errorf("%s not confirmed (%s at %s:%d): %s", c.ID, c.Status, c.File, c.Line, c.Text)
		}
	}
}

// TestTheDigestSurvivesATickingClock proves the study is a measurement of the tree,
// not of the moment it ran.
func TestTheDigestSurvivesATickingClock(t *testing.T) {
	root := repoRoot(t)
	first, err := Run(root)
	if err != nil {
		t.Fatalf("first run: %v", err)
	}
	second, err := Run(root)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if first.Digest != second.Digest {
		t.Errorf("digest moved between runs: %s then %s", first.Digest, second.Digest)
	}
	if first.Totals.ClaimsConfirmed != second.Totals.ClaimsConfirmed {
		t.Error("the statement table changed size between runs")
	}
}

// TestRunningTheStudyWritesNothing is the read-only promise as a number: every file
// the study reads is checked for size and mtime before and after the run.
func TestRunningTheStudyWritesNothing(t *testing.T) {
	root := repoRoot(t)
	before := snapshot(t, root)
	if _, err := Run(root); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if after := snapshot(t, root); before != after {
		t.Error("the corpus moved while the study ran")
	}
}

func snapshot(t *testing.T, root string) string {
	t.Helper()
	var b strings.Builder
	for _, rel := range corpusFiles(root) {
		st, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			b.WriteString(rel + " gone\n")
			continue
		}
		b.WriteString(rel + "=" + itoa(int(st.Size())) + "@" + st.ModTime().String() + "\n")
	}
	return b.String()
}

// corpusFiles is everything the study reads, so the proof covers the whole input set
// rather than the handful of files the author happened to think of.
func corpusFiles(root string) []string {
	var out []string
	for _, dir := range []string{"web/src", "web/e2e", "internal/adminapi", ".github/workflows"} {
		out = append(out, walkFiles(root, dir, ".go", ".ts", ".tsx", ".yml")...)
	}
	return out
}

// TestTheStudyImportsNothingThatCanSendOrWrite closes the protocol half of the
// promise: the package may read files and parse them, and nothing else.
func TestTheStudyImportsNothingThatCanSendOrWrite(t *testing.T) {
	allowed := map[string]bool{
		"crypto/sha256": true, "encoding/hex": true, "fmt": true, "go/parser": true,
		"go/token": true, "io/fs": true, "os": true, "path/filepath": true, "reflect": true,
		"regexp": true, "sort": true, "strconv": true, "strings": true, "unicode": true,
		"testing": true, "sync": true,
		// The two non-stdlib paths are the test framework, and a test file is the only
		// place they appear: packageImports reads the whole package, tests included, so
		// the allowlist covers them rather than excluding them.
		"github.com/stretchr/testify/assert":  true,
		"github.com/stretchr/testify/require": true,
	}
	root := repoRoot(t)
	imp := packageImports(root, ownPackageDir(root))
	if len(imp) == 0 {
		t.Fatal("no imports parsed, so the allowlist is vacuous")
	}
	for _, i := range imp {
		if !allowed[i] {
			t.Errorf("import %q is outside the read-only allowlist", i)
		}
	}
}

// TestNothingImportsTheStudy keeps the research package a study, not a feature.
func TestNothingImportsTheStudy(t *testing.T) {
	if got := importersOutside(repoRoot(t)); len(got) > 0 {
		t.Errorf("product code imports the study: %s", strings.Join(got, ", "))
	}
}

// TestTheVocabularyIsTheShippedMenu pins the corpus itself: a menu row the parser
// loses would silently shrink the only set of answers the resolver can give.
func TestTheVocabularyIsTheShippedMenu(t *testing.T) {
	res := study(t)
	if len(res.Destinations) != 24 {
		t.Errorf("menu parsed to %d rows, the shipped array has 24", len(res.Destinations))
	}
	if res.Totals.Destinations != len(res.Destinations) {
		t.Error("the totals and the parsed menu disagree about the row count")
	}
	seen := map[string]bool{}
	for _, d := range res.Destinations {
		if seen[d.Path] {
			t.Errorf("duplicate destination %q, so a phrase would belong to two rows", d.Path)
		}
		seen[d.Path] = true
	}
	if res.Totals.EnPhrases+res.Totals.IdPhrases != res.Totals.Phrases {
		t.Errorf("phrase set does not split by locale: %d + %d != %d",
			res.Totals.EnPhrases, res.Totals.IdPhrases, res.Totals.Phrases)
	}
}

// TestTheAnswerTypeCarriesNoOperation is the criterion as a structural fact: the
// resolver's answer has a path, a label and a score, and no field that could name a
// method, a record or a payload. A mis-transcription therefore cannot reach a write
// route however badly it is scored.
func TestTheAnswerTypeCarriesNoOperation(t *testing.T) {
	forbidden := []string{"method", "verb", "body", "payload", "id", "action", "params", "request"}
	rt := reflect.TypeOf(Verdict{})
	for i := 0; i < rt.NumField(); i++ {
		name := strings.ToLower(rt.Field(i).Name)
		for _, f := range forbidden {
			if name == f || strings.HasSuffix(name, f) {
				t.Errorf("Verdict carries field %q, which could address an operation", rt.Field(i).Name)
			}
		}
	}
}

// TestEveryAnswerIsAShippedScreen states the same half of the criterion
// measurably: whatever the resolver says yes to is a path the menu itself prints.
func TestEveryAnswerIsAShippedScreen(t *testing.T) {
	res := study(t)
	vocab := map[string]bool{}
	for _, d := range res.Destinations {
		vocab[d.Path] = true
	}
	n := 0
	for _, v := range res.Verdicts {
		if v.Kind != navigate {
			continue
		}
		n++
		if !vocab[v.Path] {
			t.Errorf("%q answered with %q, which is not a menu destination", v.Utterance, v.Path)
		}
	}
	if n == 0 {
		t.Fatal("nothing navigates, so the study has no navigation result")
	}
	if res.Totals.Navigated != n {
		t.Errorf("totals say %d navigations, the verdict table has %d", res.Totals.Navigated, n)
	}
}

// TestNoOperationSentenceMovesAScreen is the deny-first half of the criterion: every
// transcript that names an operation the server performs stops before scoring.
func TestNoOperationSentenceMovesAScreen(t *testing.T) {
	res := study(t)
	rows := 0
	for _, v := range res.Verdicts {
		if v.Class != "action" {
			continue
		}
		rows++
		if v.Kind != refused {
			t.Errorf("%q named an operation and was answered %q", v.Utterance, v.Answer)
		}
	}
	if rows == 0 {
		t.Fatal("the corpus has no operation transcripts, so the criterion is untested")
	}
	if res.Totals.MutationReached != 0 {
		t.Errorf("%d transcripts reached a mutation", res.Totals.MutationReached)
	}
}

// TestNoFragmentBecomesAConfidentScreen prices the other half of the criterion: a
// label fragment must be refused or asked about, never guessed into a page.
func TestNoFragmentBecomesAConfidentScreen(t *testing.T) {
	res := study(t)
	for _, v := range res.Verdicts {
		if v.Expect != expectAskOrRefuse {
			continue
		}
		if v.Kind == navigate {
			t.Errorf("fragment %q was guessed into %q", v.Utterance, v.Path)
		}
	}
	if res.Totals.WrongScreen != 0 {
		t.Errorf("%d confident answers named a screen the transcript did not ask for",
			res.Totals.WrongScreen)
	}
}

// TestTheCorpusSpeaksOnlyShippedScreens keeps the authored transcripts honest: an
// expectation naming a screen the menu does not have would pass nothing.
func TestTheCorpusSpeaksOnlyShippedScreens(t *testing.T) {
	res := study(t)
	vocab := map[string]bool{}
	for _, d := range res.Destinations {
		vocab[d.Path] = true
	}
	for _, f := range corpus() {
		if !strings.HasPrefix(f.Expect, expectNavigatePfx) {
			continue
		}
		p := strings.TrimPrefix(f.Expect, expectNavigatePfx)
		if !vocab[p] {
			t.Errorf("fixture %q expects %q, which the shipped menu does not have", f.Utterance, p)
		}
	}
}

// TestVoiceHasNothingToAttachTo measures the optionality claim: the product contains
// no speech-API reference and no backend entry point that could hold a microphone
// open, so a voice layer is necessarily a client the operator turns on.
func TestVoiceHasNothingToAttachTo(t *testing.T) {
	res := study(t)
	if res.Totals.SpeechRefs != 0 {
		t.Errorf("the frontend now references a speech API (%d hits): %s",
			res.Totals.SpeechRefs, strings.Join(res.SpeechHits, ", "))
	}
	if res.Totals.ListenerRefs != 0 {
		t.Errorf("the backend now references an audio entry point (%d hits): %s",
			res.Totals.ListenerRefs, strings.Join(res.ListenerHits, ", "))
	}
}

// TestEveryDestinationTracesToAPage pins the fallback half of the criterion: a
// spoken screen and a clicked screen are the same page, so switching to text always
// lands where the voice command was going.
func TestEveryDestinationTracesToAPage(t *testing.T) {
	res := study(t)
	if res.Totals.FallbackPages != len(res.Destinations) {
		t.Errorf("%d of %d destinations traced to a page file",
			res.Totals.FallbackPages, len(res.Destinations))
	}
	for _, f := range res.Fallbacks {
		if f.Page == "" {
			t.Errorf("%q resolved to no page, so the fallback claim cannot be checked there", f.Path)
		}
	}
}

// TestRefusalOnlyRemovesAnswers states why the pruning rules are safe: dropping a
// token from the refusal set can never add a capability, because the answer type has
// no way to express one. The set arithmetic is checked, not assumed.
func TestRefusalOnlyRemovesAnswers(t *testing.T) {
	res := study(t)
	dropped := res.Totals.BanShortDropped + res.Totals.BanScreenDropped
	if res.Totals.BanKept+dropped != res.Totals.BanRaw {
		t.Errorf("refusal set arithmetic: %d kept + %d dropped != %d derived",
			res.Totals.BanKept, dropped, res.Totals.BanRaw)
	}
	nav := map[string]bool{}
	for _, p := range res.Phrases {
		for _, tok := range tokens(normalize(p.Label)) {
			nav[tok] = true
		}
	}
	for tok := range res.Ban.Kept {
		if nav[tok] {
			t.Errorf("refusal token %q is also a screen word, so it would refuse a real command", tok)
		}
	}
	for _, tok := range res.Ban.ShortWords {
		if len([]rune(tok)) >= 3 {
			t.Errorf("%q dropped as too short but has %d letters", tok, len([]rune(tok)))
		}
	}
}

// TestTotalsAgreeWithTheTables is the splice guard: the memo's opening numbers must
// be the tables' own counts.
func TestTotalsAgreeWithTheTables(t *testing.T) {
	res := study(t)
	if res.Totals.Navigated+res.Totals.Asked+res.Totals.Refused != res.Totals.Fixtures {
		t.Errorf("verdict kinds %d+%d+%d do not cover the %d transcripts",
			res.Totals.Navigated, res.Totals.Asked, res.Totals.Refused, res.Totals.Fixtures)
	}
	if res.Totals.Reads+res.Totals.Writes != len(res.Writes)+len(res.Reads) {
		t.Error("totals disagree with the route tables")
	}
	if res.Totals.GapsOpen != len(res.Gaps) {
		t.Errorf("%d limits open of %d: an asset-free round cannot close one, so a closed row needs a reason",
			res.Totals.GapsOpen, len(res.Gaps))
	}
}

// TestTheCommentBlankerKeepsLineNumbers guards the evidence rule itself: prose must
// disappear without moving the code, or every citation below a block comment would
// point at the wrong line.
func TestTheCommentBlankerKeepsLineNumbers(t *testing.T) {
	nl := "\n"
	root := repoRoot(t)
	for _, rel := range []string{appFile, menuFile, enLocale, idLocale, ispResourceFile,
		ispAPIFile, sessionsAPIFile, platformAPIFile, ciFile} {
		src, ok := readFile(filepath.Join(root, filepath.FromSlash(rel)))
		if !ok {
			t.Fatalf("%s unreadable", rel)
		}
		raw := strings.Split(src, nl)
		kept := strings.Split(blankFor(rel, src), nl)
		if len(raw) != len(kept) {
			t.Errorf("%s: blanking changed the line count from %d to %d", rel, len(raw), len(kept))
			continue
		}
		for i := range raw {
			if len([]rune(raw[i])) != len([]rune(kept[i])) {
				t.Errorf("%s:%d blanking changed the width of the line", rel, i+1)
				break
			}
			trimmed := strings.TrimSpace(raw[i])
			marker := "//"
			if ext := filepath.Ext(rel); ext == ".yml" || ext == ".yaml" {
				marker = "#"
			}
			if strings.HasPrefix(trimmed, marker) && strings.TrimSpace(kept[i]) != "" {
				t.Errorf("%s:%d a comment line survived blanking", rel, i+1)
			}
		}
	}
}
