package voicenav

import (
	"fmt"
	"sort"
	"strings"
)

// Markdown renders one run as the tables the memo splices. Every number in the
// memo comes from here, so a table can never disagree with the package that made
// it.
func Markdown(r *Result) string {
	var b strings.Builder
	section(&b, "The measured numbers")
	dump(&b, totalsRows(r.Totals))

	section(&b, "The vocabulary: the menu rows and the labels the two locales render")
	dump(&b, vocabularyRows(r.Destinations))

	section(&b, "What the console performs as an operation, and what the refusal list keeps")
	dump(&b, verbRows(r.Verbs))

	section(&b, "The derived refusal set, and exactly what the pruning rules removed")
	dump(&b, banRows(r))

	section(&b, "The transcript corpus, scored")
	dump(&b, verdictRows(r.Verdicts))

	section(&b, "Per error class")
	dump(&b, classTable(r.Classes))

	section(&b, "The typed fallback, destination by destination")
	dump(&b, fallbackTable(r.Fallbacks))

	section(&b, "The absence scan: anything already listening")
	dump(&b, absenceRows(r))

	section(&b, "The mechanism lines")
	dump(&b, pinTable(r.Pins))

	section(&b, "The statements, pinned to code")
	dump(&b, claimRows(r.Claims))

	section(&b, "What this study does not close")
	dump(&b, gapRows(r.Gaps))

	section(&b, "The import set this package builds on")
	dump(&b, importRows(r.Imports))

	section(&b, "What the digest locks")
	digestSection(&b, r)
	return b.String()
}

func section(b *strings.Builder, title string) {
	b.WriteString("\n## " + title + "\n\n")
}

func dump(b *strings.Builder, rows [][]string) {
	if len(rows) == 0 {
		b.WriteString("(no rows)\n")
		return
	}
	for i, ln := range rows {
		b.WriteString("| " + strings.Join(ln, " | ") + " |")
		if i == 0 {
			b.WriteString("\n|")
			for range ln {
				b.WriteString(" --- |")
			}
		}
		b.WriteString("\n")
	}
}

// head starts a table that may have no rows; an empty body still keeps the header,
// so the memo's splice cannot silently lose a column.
func head(cols ...string) [][]string {
	return [][]string{cols}
}

func totalsRows(t Totals) [][]string {
	return [][]string{
		{"number", "what it counts", "value"},
		{"destinations", "menu rows a spoken command may name", itoa(t.Destinations)},
		{"phrases", "labels the two locales render for them", itoa(t.Phrases)},
		{"en-phrases", "labels from the English locale", itoa(t.EnPhrases)},
		{"id-phrases", "labels from the Indonesian locale", itoa(t.IdPhrases)},
		{"unlabelled", "destinations speakable in neither locale", itoa(t.Unlabelled)},
		{"gated", "destinations the menu hides from a lower role", itoa(t.Gated)},
		{"reads", "GET registrations in the admin API", itoa(t.Reads)},
		{"writes", "POST, PUT and DELETE registrations in the admin API", itoa(t.Writes)},
		{"action-words", "operation words the server derives", itoa(t.ActionVerbs)},
		{"ban-raw", "tokens the naive refusal list holds", itoa(t.BanRaw)},
		{"ban-short", "tokens dropped for being under three letters", itoa(t.BanShortDropped)},
		{"ban-screen", "tokens dropped because a screen label uses them", itoa(t.BanScreenDropped)},
		{"ban-kept", "tokens the resolver refuses on", itoa(t.BanKept)},
		{"fixtures", "transcripts scored", itoa(t.Fixtures)},
		{"navigated", "transcripts answered with a screen", itoa(t.Navigated)},
		{"asked", "transcripts answered with a question", itoa(t.Asked)},
		{"refused", "transcripts answered with nothing", itoa(t.Refused)},
		{"matched", "transcripts whose answer is what the study requires", itoa(t.Fixtures - t.Mismatches)},
		{"mismatched", "transcripts whose answer is not what the study requires", itoa(t.Mismatches)},
		{"wrong-screen", "confident answers naming a screen the transcript did not ask for", itoa(t.WrongScreen)},
		{"mutation-reached", "confident answers that performed an operation", itoa(t.MutationReached)},
		{"fallback-pages", "destinations traced to a page file", itoa(t.FallbackPages)},
		{"typed-fields", "TextInput and SearchInput elements across those pages", itoa(t.TypedFields)},
		{"filter-bound", "pages that bind a filter to typed input", itoa(t.FilterBound)},
		{"no-typed-field", "screens reachable only by a whole-label utterance or a click", itoa(t.VoiceOnly)},
		{"speech-refs", "speech-API references in the frontend", itoa(t.SpeechRefs)},
		{"listener-refs", "microphone-listener references in the backend", itoa(t.ListenerRefs)},
		{"claims", "statements pinned to a code line", itoa(t.Claims)},
		{"confirmed", "statements whose citation still lands", itoa(t.ClaimsConfirmed)},
		{"broken", "statements that no longer hold", itoa(t.ClaimsBroken)},
		{"vacated", "statements whose file is gone", itoa(t.ClaimsVacated)},
		{"limits", "open questions this round cannot close", itoa(t.Gaps)},
		{"open", "limits still open after this run", itoa(t.GapsOpen)},
		{"imports", "packages this study builds on", itoa(t.Imports)},
	}
}

func vocabularyRows(ds []Destination) [][]string {
	out := head("menu destination", "label key", "gate", "English label", "Indonesian label", "phrases")
	for _, d := range ds {
		gate := "(none)"
		if d.Gate != "" {
			gate = d.Gate
		}
		n := 0
		if d.English != "" {
			n++
		}
		if d.Indonesian != "" {
			n++
		}
		en := d.English
		if en == "" {
			en = "(not rendered)"
		}
		id := d.Indonesian
		if id == "" {
			id = "(not rendered)"
		}
		out = append(out, []string{d.Path, d.LabelKey, gate, en, id, itoa(n)})
	}
	return out
}

func verbRows(vs []ActionVerb) [][]string {
	out := head("action word", "where it comes from", "code line", "English label the console renders",
		"Indonesian label the console renders")
	for _, v := range vs {
		en := v.English
		if en == "" {
			en = "(none)"
		}
		id := v.Speaking
		if id == "" {
			id = "(none)"
		}
		out = append(out, []string{v.Verb, v.Source, v.File + ":" + itoa(v.Line), en, id})
	}
	return out
}

func banRows(r *Result) [][]string {
	out := head("refusal token", "derived from", "kept or dropped")
	for _, t := range sortedKeys(r.RawBan) {
		verdict := "kept"
		switch {
		case contains(r.Ban.ShortWords, t):
			verdict = "dropped: under three letters"
		case contains(r.Ban.ScreenWords, t):
			verdict = "dropped: a screen label uses it"
		}
		out = append(out, []string{t, r.RawBan[t], verdict})
	}
	return out
}

func verdictRows(vs []Verdict) [][]string {
	out := head("transcript", "error class", "why this transcript is in the corpus", "answer",
		"kind", "why it stopped there", "score", "margin", "the study requires", "match")
	for _, v := range vs {
		var whyParts []string
		for _, s := range []string{v.Reason, v.Detail} {
			if s != "" {
				whyParts = append(whyParts, s)
			}
		}
		why := "-"
		if len(whyParts) > 0 {
			why = strings.Join(whyParts, " - ")
		}
		out = append(out, []string{v.Utterance, v.Class, orDash(v.Note), v.Answer, v.Kind,
			why, pct(v.Score), pct(v.Margin), v.Expect, yesNo(v.Ok)})
	}
	return out
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func classTable(rows []ClassRow) [][]string {
	out := head("error class", "transcripts", "answered with a screen", "asked", "refused", "matched the requirement")
	for _, c := range rows {
		out = append(out, []string{c.Class, itoa(c.Fixtures), itoa(c.Navigated), itoa(c.Asked),
			itoa(c.Refused), itoa(c.Matched) + " of " + itoa(c.Fixtures)})
	}
	return out
}

func fallbackTable(rows []FallbackRow) [][]string {
	out := head("menu destination", "page file it traces to", "typed fields on it", "filter bound to input",
		"reachable by typing")
	for _, f := range rows {
		page := f.Page
		if page == "" {
			page = "(not traced)"
		}
		out = append(out, []string{f.Path, page, itoa(f.TypedFields), yesNo(f.FilterBound), yesNo(!f.VoiceOnly)})
	}
	return out
}

func absenceRows(r *Result) [][]string {
	out := head("scan", "files naming it", "hits")
	out = append(out, []string{"browser speech APIs across web/src and web/e2e",
		orNone(r.SpeechHits), itoa(r.Totals.SpeechRefs)})
	out = append(out, []string{"microphone or audio entry points across internal",
		orNone(r.ListenerHits), itoa(r.Totals.ListenerRefs)})
	return out
}

func pinTable(rows []PinRow) [][]string {
	out := head("code line", "mechanism", "status")
	for _, p := range rows {
		out = append(out, []string{p.File + ":" + itoa(p.Line), p.Symbol, p.Status})
	}
	return out
}

func claimRows(rows []ClaimStatus) [][]string {
	out := head("id", "statement", "code line", "status", "measured", "asserts an absence")
	for _, c := range rows {
		where := c.File
		switch c.Line {
		case -1:
			where += " (file gone)"
		case 0:
			where += " (line moved)"
		default:
			where += ":" + itoa(c.Line)
		}
		out = append(out, []string{c.ID, c.Text, where, c.Status, c.Evid, yesNo(c.Neg)})
	}
	return out
}

func gapRows(rows []GapStatus) [][]string {
	out := head("id", "what the study cannot close", "what it would need", "kind of asset", "verdict")
	for _, g := range rows {
		out = append(out, []string{g.ID, g.Text, g.Needs, g.Asset, noGo(g.Open)})
	}
	return out
}

func importRows(imports []string) [][]string {
	out := head("package import")
	for _, imp := range imports {
		out = append(out, []string{imp})
	}
	return out
}

func digestSection(b *strings.Builder, r *Result) {
	b.WriteString("The digest is the sha256 of the tab-separated rows below, so the memo's tables\n" +
		"are reproducible from the tree alone and move the moment the tree moves.\n\n")
	fmt.Fprintf(b, "- %d vocabulary rows\n- %d action words\n- %d refusal tokens kept of %d derived\n"+
		"- %d scored transcripts\n- %d fallback rows\n- %d mechanism lines\n- %d statements\n- %d limits\n\n",
		len(r.Destinations), len(r.Verbs), len(r.Ban.Kept), len(r.RawBan),
		len(r.Verdicts), len(r.Fallbacks), len(r.Pins), len(r.Claims), len(r.Gaps))
	fmt.Fprintf(b, "- digest: `%s`\n", r.Digest)
}

// helpers

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func contains(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

func orNone(files []string) string {
	if len(files) == 0 {
		return "(none)"
	}
	return strings.Join(files, ", ")
}

func noGo(open bool) string {
	if open {
		return "no-go"
	}
	return "closed"
}
