// Package voicenav measures what a voice-assisted navigation layer for the
// management console could and could not do, without a microphone, a recognizer,
// or a browser session, so that the accessibility question can be answered with
// numbers instead of opinion.
//
// The study is read-only: it parses the console's own files, holds the parsed
// tables in memory, and writes nothing. Nothing in cmd/ or internal/ imports it,
// and it imports only the standard library.
//
// # What is measured
//
// Three quantities come out of the parsed source:
//
//   - the vocabulary. The only words a command may name are the labels the two
//     shipped locales render for the rows of the sidebar menu
//     (web/src/components/CustomMenu.tsx, web/src/i18n/en-US.ts,
//     web/src/i18n/indonesian-overrides.ts), so a phrase exists only because the
//     console prints it.
//   - the reach of an answer. A verdict carries a menu path, a label and a score.
//     The admin API's write registrations are counted from the backend's own
//     route declarations, and the resolver has no field that could name one of
//     them, which is the "a mis-transcription causes no mutation" criterion as a
//     structural fact rather than a behavior test.
//   - the refusal set. The operation words an utterance may not carry are derived
//     from the server's own action guard and from the trailing segment of its
//     action routes, together with the label each locale renders for them, and
//     then pruned by two rules the corpus forced.
//
// A scored transcript corpus of both locales, with the automatic-speech-recognition
// error classes a mis-transcription is made of, is run through that resolver; every
// number the memo quotes comes out of the same run.
//
// # Reading the tables
//
// VOICE_DUMP=1 go test -run TestReportMarkdownDump ./internal/voicenav/ prints the
// Markdown the memo splices. The digest at the bottom of that output is the sha256
// of the rows, so the tables cannot be quietly retyped.
package voicenav

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
)

// Result is everything the memo quotes. It is produced by one run over the tree
// and rendered by Markdown without a hand-typed number in between.
type Result struct {
	Destinations []Destination
	Phrases      []Phrase
	Writes       []Route
	Reads        []Route
	Verbs        []ActionVerb
	RawBan       map[string]string
	Ban          PrunedBan
	Verdicts     []Verdict
	Classes      []ClassRow
	Fallbacks    []FallbackRow
	SpeechHits   []string
	ListenerHits []string
	Pins         []PinRow
	Claims       []ClaimStatus
	Gaps         []GapStatus
	Totals       Totals
	Imports      []string
	Digest       string
}

// Totals is the table the memo opens with.
type Totals struct {
	Destinations     int
	Phrases          int
	EnPhrases        int
	IdPhrases        int
	Unlabelled       int
	Gated            int
	Reads            int
	Writes           int
	ActionVerbs      int
	BanRaw           int
	BanShortDropped  int
	BanScreenDropped int
	BanKept          int
	Fixtures         int
	Navigated        int
	Asked            int
	Refused          int
	Mismatches       int
	WrongScreen      int
	MutationReached  int
	FallbackPages    int
	VoiceOnly        int
	TypedFields      int
	FilterBound      int
	SpeechRefs       int
	ListenerRefs     int
	Claims           int
	ClaimsConfirmed  int
	ClaimsBroken     int
	ClaimsVacated    int
	Gaps             int
	GapsOpen         int
	Imports          int
}

// Run reads the tree once and returns the whole measurement.
func Run(root string) (*Result, error) {
	ds := destinations(root)
	if len(ds) == 0 {
		return nil, fmt.Errorf("no menu rows parsed from %s", menuFile)
	}
	phrases := phraseSet(ds)
	routes := registrations(root)
	verbs := actionVerbs(root)
	raw := banTokens(verbs)
	pruned := pruneBan(raw, phrases)
	fs := corpus()
	vs := verdicts(fs, phrases, pruned.Kept)
	fb := fallbackRows(root, ds)

	res := &Result{
		Destinations: ds,
		Phrases:      phrases,
		Verbs:        verbs,
		RawBan:       raw,
		Ban:          pruned,
		Verdicts:     vs,
		Classes:      classRows(vs),
		Fallbacks:    fb,
		SpeechHits:   []string{},
	}
	for _, r := range routes {
		if isWrite(r) {
			res.Writes = append(res.Writes, r)
			continue
		}
		res.Reads = append(res.Reads, r)
	}
	_, files := speechHits(root)
	res.SpeechHits = files
	_, lfiles := listenerHits(root)
	res.ListenerHits = lfiles

	ev := gatherEvidence(root, res)
	res.Pins = pinRows(root)
	res.Claims = evalClaims(root, ev, claims(ev))
	res.Gaps = gaps(ev)
	res.Imports = packageImports(root, ownPackageDir(root))
	res.Totals = totals(res, ev)
	res.Digest = digest(res)
	return res, nil
}

// evidence is the numbers the claims and gaps need. It is gathered once so a
// statement cannot quote a figure the run did not produce.
type evidence struct {
	root          string
	destinations  []Destination
	phrases       []Phrase
	writes        []Route
	reads         []Route
	verbs         []ActionVerb
	rawBan        map[string]string
	ban           PrunedBan
	verdicts      []Verdict
	fallbacks     []FallbackRow
	classes       int
	speechRefs    int
	listenerRefs  int
	clean         int
	damaged       int
	damagedHit    int
	wrongScreen   int
	mutation      int
	actionRows    int
	actionRefused int
	fragGuessed   int
	unlabelled    int
	gated         int
	typedFields   int
	filterBound   int
	voiceOnly     int
	pagesFound    int
	undeclared    []string
}

func gatherEvidence(root string, res *Result) *evidence {
	ev := &evidence{
		root:         root,
		destinations: res.Destinations,
		phrases:      res.Phrases,
		writes:       res.Writes,
		reads:        res.Reads,
		verbs:        res.Verbs,
		rawBan:       res.RawBan,
		ban:          res.Ban,
		verdicts:     res.Verdicts,
		fallbacks:    res.Fallbacks,
	}
	ev.speechRefs, _ = speechHits(root)
	ev.listenerRefs, _ = listenerHits(root)
	shell := shellRoutes(root)
	for _, d := range ev.destinations {
		if d.English == "" && d.Indonesian == "" {
			ev.unlabelled++
		}
		if d.Gate != "" {
			ev.gated++
		}
		if _, ok := shell[d.Path]; !ok {
			ev.undeclared = append(ev.undeclared, d.Path)
		}
	}
	for _, f := range ev.fallbacks {
		if f.Page != "" {
			ev.pagesFound++
		}
		ev.typedFields += f.TypedFields
		if f.FilterBound {
			ev.filterBound++
		}
		if f.VoiceOnly {
			ev.voiceOnly++
		}
	}
	classOf := map[string]bool{}
	for _, v := range ev.verdicts {
		if v.Class == "action" {
			ev.actionRows++
			if v.Kind == refused {
				ev.actionRefused++
			}
		}
		if v.Class == "truncation" && v.Expect == expectAskOrRefuse && v.Kind == navigate {
			ev.fragGuessed++
		}
		switch v.Class {
		case "homophone", "boundary", "truncation", "deletion", "insertion":
			if v.Expect != expectAskOrRefuse {
				ev.damaged++
				classOf[v.Class] = true
				if v.Kind == navigate && strings.HasPrefix(v.Expect, expectNavigatePfx) &&
					v.Path == strings.TrimPrefix(v.Expect, expectNavigatePfx) {
					ev.damagedHit++
				}
			}
		}
		if v.Kind == navigate && strings.HasPrefix(v.Expect, expectNavigatePfx) &&
			v.Path != strings.TrimPrefix(v.Expect, expectNavigatePfx) {
			ev.wrongScreen++
		}
		if v.Kind == navigate && v.Class != "action-uncovered" &&
			(v.Expect == expectAnySafe || v.Expect == "refused") {
			ev.mutation++
		}
		if v.Ok {
			ev.clean++
		}
	}
	ev.classes = len(classOf)
	return ev
}

func totals(res *Result, ev *evidence) Totals {
	t := Totals{
		Destinations:     len(res.Destinations),
		Phrases:          len(res.Phrases),
		Writes:           len(res.Writes),
		Reads:            len(res.Reads),
		ActionVerbs:      len(res.Verbs),
		BanRaw:           len(res.RawBan),
		BanShortDropped:  len(res.Ban.ShortWords),
		BanScreenDropped: len(res.Ban.ScreenWords),
		BanKept:          len(res.Ban.Kept),
		Fixtures:         len(res.Verdicts),
		Claims:           len(res.Claims),
		Gaps:             len(res.Gaps),
		Imports:          len(res.Imports),
		WrongScreen:      ev.wrongScreen,
		MutationReached:  ev.mutation,
		Unlabelled:       ev.unlabelled,
		Gated:            ev.gated,
		TypedFields:      ev.typedFields,
		FilterBound:      ev.filterBound,
		VoiceOnly:        ev.voiceOnly,
		FallbackPages:    ev.pagesFound,
		SpeechRefs:       ev.speechRefs,
		ListenerRefs:     ev.listenerRefs,
	}
	for _, v := range res.Verdicts {
		switch v.Kind {
		case navigate:
			t.Navigated++
		case ambiguous:
			t.Asked++
		default:
			t.Refused++
		}
		if !v.Ok {
			t.Mismatches++
		}
	}
	for _, p := range res.Phrases {
		if p.Lang == "en-US" {
			t.EnPhrases++
		} else {
			t.IdPhrases++
		}
	}
	for _, c := range res.Claims {
		switch c.Status {
		case confirmed:
			t.ClaimsConfirmed++
		case broken:
			t.ClaimsBroken++
		case vacated:
			t.ClaimsVacated++
		}
	}
	for _, g := range res.Gaps {
		if g.Open {
			t.GapsOpen++
		}
	}
	return t
}

// digest content-addresses the rows the memo prints, so the tables move the
// moment the tree moves and cannot be quietly retyped.
func digest(r *Result) string {
	var b strings.Builder
	for _, d := range r.Destinations {
		fmt.Fprintf(&b, "dest\t%s\t%s\t%s\t%s\t%s\n", d.Path, d.LabelKey, d.Gate, d.English, d.Indonesian)
	}
	for _, v := range r.Verbs {
		fmt.Fprintf(&b, "verb\t%s\t%s\t%s:%d\t%s\t%s\n", v.Verb, v.Source, v.File, v.Line, v.English, v.Speaking)
	}
	kept := make([]string, 0, len(r.Ban.Kept))
	for k := range r.Ban.Kept {
		kept = append(kept, k)
	}
	sort.Strings(kept)
	for _, k := range kept {
		fmt.Fprintf(&b, "ban\t%s\t%s\n", k, r.Ban.Kept[k])
	}
	for _, v := range r.Verdicts {
		fmt.Fprintf(&b, "verdict\t%s\t%s\t%s\t%s\t%s\t%.3f\t%.3f\t%t\n",
			v.Utterance, v.Class, v.Kind, v.Path, v.Expect, v.Score, v.Margin, v.Ok)
	}
	for _, f := range r.Fallbacks {
		fmt.Fprintf(&b, "fallback\t%s\t%s\t%d\t%t\t%t\n", f.Path, f.Page, f.TypedFields, f.FilterBound, f.VoiceOnly)
	}
	for _, p := range r.Pins {
		fmt.Fprintf(&b, "pin\t%s\t%s\t%d\n", p.File, p.Symbol, p.Line)
	}
	for _, c := range r.Claims {
		fmt.Fprintf(&b, "claim\t%s\t%s\t%s:%d\n", c.ID, c.Status, c.File, c.Line)
	}
	for _, g := range r.Gaps {
		fmt.Fprintf(&b, "gap\t%s\t%s\t%t\n", g.ID, g.Needs, g.Open)
	}
	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
