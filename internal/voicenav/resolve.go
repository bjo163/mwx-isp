package voicenav

import (
	"sort"
	"strings"
	"unicode"
)

// The tuning the resolver runs with. They are stated here, printed in the memo,
// and the corpus is scored against them, so a threshold change shows up as a
// different measured outcome instead of a silent behavior change.
const (
	// acceptFloor is how close a transcript must get to a shipped label before
	// the resolver is willing to call it that screen.
	acceptFloor = 0.78
	// acceptMargin is how far the winner must lead the runner-up. Below it the
	// transcript is genuinely two screens, and the answer is to ask.
	acceptMargin = 0.06
	// denyFloor is how close a transcript word must get to a mutation word
	// before the whole utterance is refused. It is looser than acceptFloor on
	// purpose: refusing one screen too many costs a click, accepting one
	// mutation too many does not.
	denyFloor = 0.82
)

// Verdict kinds. A navigate verdict is the only kind that carries a path, and a
// path can only come from the shipped menu - the type has no verb, no id and no
// payload field for the same reason the resolver has no way to reach a write
// route.
const (
	navigate  = "navigate"
	ambiguous = "ambiguous"
	refused   = "refused"
)

// The refusal reasons, printed in the memo so a reader can see why a transcript
// stopped rather than moved.
const (
	reasonEmpty        = "silence or noise: nothing to compare"
	reasonMutation     = "names an operation the server performs with a write route"
	reasonNoVocabulary = "no shipped label resembles it"
	reasonFragment     = "fewer words than the label has, so it names more than one screen"
	reasonBelowFloor   = "a label was nearly recognized, but not closely enough to act"
	reasonTwoScreens   = "two screens are equally close, so the console asks"
)

// Phrase is one thing the console says on screen, in one shipped locale.
type Phrase struct {
	Path  string
	Label string
	Lang  string
	Line  int
}

// utterance is a transcript prepared for comparison.
type utterance struct {
	raw    string
	tokens []string
	forms  []string // tokens and short joins, so a split word can be recovered
}

// normalize lower-cases a transcript and turns everything that is not a letter
// or a digit into a separator. It keeps Unicode letters, because the labels are
// Indonesian and English and neither is ASCII-only.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteRune(' ')
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

func tokens(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func newUtterance(raw string) utterance {
	u := utterance{raw: raw, tokens: tokens(normalize(raw))}
	for i := 0; i < len(u.tokens); i++ {
		u.forms = append(u.forms, u.tokens[i])
		for j := i + 1; j < len(u.tokens) && j <= i+2; j++ {
			u.forms = append(u.forms, strings.Join(u.tokens[i:j+1], ""))
		}
	}
	return u
}

// similarity is 1 - edit distance over the longer rune length, so 1.0 means the
// two words are the same word and 0.0 means nothing in common.
func similarity(a, b string) float64 {
	if a == "" || b == "" {
		return 0
	}
	ra, rb := []rune(a), []rune(b)
	d := levenshtein(ra, rb)
	longer := len(ra)
	if len(rb) > longer {
		longer = len(rb)
	}
	return 1 - float64(d)/float64(longer)
}

func levenshtein(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min3(cur[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

// phraseScore compares one label against a transcript word by word. Every word
// of the label must be recognized, either exactly (a transcript that carried the
// whole label plus lead-in words scores 1.0) or as the closest thing actually
// heard. Words added by the recognizer cost nothing; words it lost or mangled
// cost the whole margin.
func phraseScore(p Phrase, u utterance) float64 {
	want := tokens(normalize(p.Label))
	if len(want) == 0 || len(u.forms) == 0 {
		return 0
	}
	total := 0.0
	for _, w := range want {
		best := 0.0
		for _, f := range u.forms {
			if s := similarity(w, f); s > best {
				best = s
			}
		}
		total += best
	}
	return total / float64(len(want))
}

// Candidate is one screen the transcript could mean.
type Candidate struct {
	Path  string
	Label string
	Lang  string
	Line  int
	Score float64
}

// rank scores every phrase the console ships and returns the best per path. A
// label the transcript cannot cover word for word is not a candidate at all: the
// speaker said fewer words than the screen has, which is a fragment, and a
// fragment names more than one screen. This rule came out of the corpus rather
// than in front of it - the single word "jaringan" scored well enough for the
// Operations label to be answered, and that answer is not defensible.
func rank(u utterance, phrases []Phrase) ([]Candidate, []Candidate) {
	best := map[string]Candidate{}
	for _, p := range phrases {
		s := phraseScore(p, u)
		if s <= 0 {
			continue
		}
		if cur, ok := best[p.Path]; !ok || s > cur.Score {
			best[p.Path] = Candidate{Path: p.Path, Label: p.Label, Lang: p.Lang, Line: p.Line, Score: s}
		}
	}
	all := make([]Candidate, 0, len(best))
	for _, c := range best {
		all = append(all, c)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Score != all[j].Score {
			return all[i].Score > all[j].Score
		}
		return all[i].Path < all[j].Path
	})
	var eligible []Candidate
	for _, c := range all {
		if len(u.tokens) >= len(tokens(normalize(c.Label))) {
			eligible = append(eligible, c)
		}
	}
	return all, eligible
}

// Verdict is the resolver's whole answer to one transcript.
type Verdict struct {
	Utterance string
	Class     string
	Reason    string
	Note      string
	Expect    string

	Kind   string
	Path   string
	Label  string
	Lang   string
	Score  float64
	Margin float64
	Detail string
	Answer string
	Ok     bool
}

// Resolve turns a transcript into at most one navigation target, after the
// refusal checks. Order matters: an empty transcript, then a mutation word, then
// ambiguity, then acceptance, then refusal of whatever cleared nothing.
func Resolve(raw string, phrases []Phrase, ban map[string]string) Verdict {
	u := newUtterance(raw)
	v := Verdict{Utterance: raw, Kind: refused}

	if len(u.tokens) == 0 {
		v.Reason = reasonEmpty
		v.Detail = "no word recognized"
		v.Answer = "(nothing: the console asks again)"
		return v
	}

	for _, f := range u.forms {
		if who, ok := bannedIn(f, ban); ok {
			v.Kind = refused
			v.Reason = reasonMutation
			v.Detail = who + ", heard as " + f
			v.Answer = "(refused: navigation only - " + who + " is performed by a write route)"
			return v
		}
	}

	all, ranked := rank(u, phrases)
	if len(ranked) == 0 {
		if len(all) == 0 {
			v.Reason = reasonNoVocabulary
			v.Detail = "nothing in the shipped labels resembles it"
			v.Answer = "(refused: no screen named)"
			return v
		}
		v.Reason = reasonFragment
		v.Score = all[0].Score
		v.Detail = "the closest label, " + all[0].Label + ", has " +
			itoa(len(tokens(normalize(all[0].Label)))) + " words; the transcript has " + itoa(len(u.tokens))
		v.Answer = "(refused: a fragment of a label names more than one screen)"
		return v
	}

	v.Score = ranked[0].Score
	v.Margin = 1.0
	if len(ranked) > 1 {
		v.Margin = ranked[0].Score - ranked[1].Score
	}
	if v.Score < acceptFloor {
		v.Kind = refused
		v.Reason = reasonBelowFloor
		v.Detail = "best candidate " + ranked[0].Path + " at " + pct(ranked[0].Score) + fragmentNote(u, all, ranked)
		v.Answer = "(refused: too far from any label - " + ranked[0].Label + " at " + pct(ranked[0].Score) + ")"
		return v
	}
	if len(ranked) > 1 && v.Margin < acceptMargin && ranked[1].Score >= acceptFloor {
		v.Kind = ambiguous
		v.Reason = reasonTwoScreens
		v.Detail = ranked[0].Path + " " + pct(ranked[0].Score) + " vs " + ranked[1].Path + " " + pct(ranked[1].Score)
		v.Answer = "(asks: " + ranked[0].Label + " or " + ranked[1].Label + "?)"
		return v
	}

	v.Kind = navigate
	v.Path = ranked[0].Path
	v.Label = ranked[0].Label
	v.Lang = ranked[0].Lang
	v.Reason = ""
	v.Detail = ranked[0].Lang + " label, " + pct(ranked[0].Score)
	v.Answer = v.Path
	return v
}

// bannedIn fuzzy-matches one heard word against the mutation vocabulary. The
// closest token wins, and ties break on the token itself, so the printed reason
// does not depend on map order.
func bannedIn(form string, ban map[string]string) (string, bool) {
	best, bestScore := "", 0.0
	for token, who := range ban {
		s := similarity(form, token)
		if s >= denyFloor && (best == "" || s > bestScore || (s == bestScore && token < best)) {
			best, bestScore = token, s
			if token == form {
				return who, true
			}
		}
	}
	if best == "" {
		return "", false
	}
	return ban[best], true
}

func pct(f float64) string { return itoa(int(f*100+0.5)) + "%" }

// fragmentNote reports how many of the labels that resemble the transcript were
// dropped before scoring because the transcript has fewer words than they do -
// the rule from rank, shown at the row where it changed the outcome.
func fragmentNote(u utterance, all, ranked []Candidate) string {
	if len(ranked) == 0 || len(all) == len(ranked) {
		return ""
	}
	return "; " + itoa(len(all)-len(ranked)) + " of the " + itoa(len(all)) +
		" labels that resemble it were dropped as fragments, since the transcript has only " +
		itoa(len(u.tokens)) + " word(s)"
}

// phraseSet flattens the destinations into the phrases a transcript can match,
// one per shipped locale, with the locale line that renders it.
func phraseSet(ds []Destination) []Phrase {
	var out []Phrase
	for _, d := range ds {
		if d.English != "" {
			out = append(out, Phrase{Path: d.Path, Label: d.English, Lang: "en-US", Line: d.EnLine})
		}
		if d.Indonesian != "" {
			out = append(out, Phrase{Path: d.Path, Label: d.Indonesian, Lang: "id-ID", Line: d.IdLine})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Lang < out[j].Lang
	})
	return out
}
