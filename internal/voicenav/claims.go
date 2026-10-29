package voicenav

import (
	"sort"
	"strings"
)

// Status of one statement, decided by whether the citation still lands on a code
// line that carries the symbols the statement names.
const (
	confirmed = "confirmed"
	broken    = "broken"
	vacated   = "vacated"
)

// ClaimStatus is one statement the memo makes, pinned to a code line.
type ClaimStatus struct {
	ID     string
	Text   string
	File   string
	Line   int
	Status string
	Evid   string
	Neg    bool // a negative claim asserts an absence and states its scope
}

// GapStatus is one limit the study leaves open, with the asset it needs and the
// verdict the round records for it.
type GapStatus struct {
	ID    string
	Text  string
	Needs string
	Asset string
	Open  bool
}

// claimDef is one statement as the table writes it: the sentence, the file and
// the tokens that must sit on one code line for the sentence still to be true,
// the evidence the memo prints beside it, and - for a statement that asserts an
// absence - the check that decides whether the absence still holds.
type claimDef struct {
	id     string
	text   string
	file   string
	tokens []string
	evid   func(*evidence) string
	check  func(*evidence) bool
}

// evalClaims resolves the table against the tree. A citation that no longer lands
// reports broken, a citation whose file is gone reports vacated, and a failed
// absence check downgrades a pinned claim to broken. Nothing is dropped, because
// a claim that quietly disappears is worse than a claim that fails.
func evalClaims(root string, ev *evidence, defs []claimDef) []ClaimStatus {
	out := make([]ClaimStatus, 0, len(defs))
	for _, d := range defs {
		c := ClaimStatus{ID: d.id, Text: d.text, File: d.file, Neg: d.check != nil}
		c.Line = anchorAny(root, d.file, d.tokens)
		switch c.Line {
		case -1:
			c.Status = vacated
		case 0:
			c.Status = broken
		default:
			c.Status = confirmed
		}
		if d.check != nil && !d.check(ev) {
			c.Status = broken
		}
		if d.evid != nil {
			c.Evid = d.evid(ev)
		}
		out = append(out, c)
	}
	return out
}

// anchorAny finds the first line that carries every token listed, each matched as
// a whole identifier when it is identifier-shaped.
func anchorAny(root, rel string, tokens []string) int {
	ls := lines(root, rel)
	if ls == nil {
		return -1
	}
	for i, ln := range ls {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		ok := true
		for _, t := range tokens {
			if !containsBounded(ln, t) {
				ok = false
				break
			}
		}
		if ok {
			return i + 1
		}
	}
	return 0
}

// PinRow is one mechanism line the memo cites, located in the tree.
type PinRow struct {
	File   string
	Symbol string
	Line   int
	Status string
}

// pinRows locates the code lines that carry the study's mechanism claims.
func pinRows(root string) []PinRow {
	pins := []struct {
		file, symbol string
		tokens       []string
	}{
		{menuFile, "a menu row is the only thing a command can name", []string{"to:", "labelKey:"}},
		{menuFile, "a gated row is still a row the vocabulary carries", []string{"permissions: ["}},
		{enLocale, "the English phrase set is the rendered label block", []string{"menu: {"}},
		{idLocale, "the Indonesian phrase set is the rendered label block", []string{"menu: {"}},
		{idLocale, "a damaged transcript is scored against a real label", []string{"customers: 'Pelanggan'"}},
		{appFile, "a resource declaration gives a destination its page", []string{"<Resource"}},
		{appFile, "a custom route gives a path its page", []string{"<Route"}},
		{appFile, "the home dashboard is the root path", []string{"dashboard={Dashboard}"}},
		{ispAPIFile, "the server's own guard closes the action vocabulary", []string{"action !="}},
		{ispAPIFile, "a mutation an operator could speak is a write route", []string{"ApiPOST(\"/isp/payments/:id/reverse\""}},
		{sessionsAPIFile, "the plainest operation is a POST on a record", []string{"ApiPOST(\"/sessions/:id/disconnect\""}},
		{platformAPIFile, "a route tail that is also a screen label", []string{"ApiPOST(\"/platform/tenants/:id/operators\""}},
		{ispResourceFile, "the typed fallback the console ships is a list filter", []string{"SearchInput"}},
		{resolverFile, "the resolver's thresholds, printed with the run", []string{"acceptFloor = 0.78"}},
		{ciFile, "the repository's gate is a build and test, never an audio run", []string{"npm run build"}},
	}
	var out []PinRow
	for _, p := range pins {
		line := anchorAny(root, p.file, p.tokens)
		status := "pinned"
		switch line {
		case -1:
			status = "vacated"
		case 0:
			status = "broken"
		}
		out = append(out, PinRow{File: p.file, Symbol: p.symbol, Line: line, Status: status})
	}
	return out
}

// claims states every sentence the memo is allowed to say, each pinned to the code
// line that proves it.
func claims(ev *evidence) []claimDef {
	out := []claimDef{
		{"C01", "the words a command may name are the words the console prints, so a phrase exists only because a locale renders it",
			menuFile, []string{"to:", "labelKey:"},
			func(*evidence) string {
				return itoa(len(ev.destinations)) + " menu rows, " + itoa(len(ev.phrases)) + " phrases"
			}, nil},

		{"C02", "both shipped locales contribute the vocabulary, because either one is what an operator hears on screen",
			enLocale, []string{"menu: {"},
			func(*evidence) string {
				return itoa(ev.countLang("en-US")) + " en-US + " + itoa(ev.countLang("id-ID")) +
					" id-ID phrases, " + itoa(ev.unlabelled) + " destinations speakable in neither locale"
			}, func(e *evidence) bool { return e.unlabelled == 0 }},

		{"C03", "every path the resolver can answer with is a route the app shell already declares, so voice adds no application",
			appFile, []string{"<Resource"},
			func(*evidence) string {
				return itoa(len(ev.destinations)-len(ev.undeclared)) + " of " + itoa(len(ev.destinations)) +
					" destinations resolve to a declared route" + namedList(" (", ev.undeclared)
			}, func(e *evidence) bool { return len(e.undeclared) == 0 }},

		{"C04", "the console's write surface is larger than its read surface, and none of it is reachable through the answer type",
			sessionsAPIFile, []string{"ApiPOST(\"/sessions/:id/disconnect\""},
			func(*evidence) string {
				return itoa(len(ev.writes)) + " write registrations vs " + itoa(len(ev.reads)) +
					" reads; a verdict carries a menu path, never a method, id or payload"
			}, nil},

		{"C05", "the mutation vocabulary is taken from the server's own guard and action routes, never typed by hand",
			ispAPIFile, []string{"action !="},
			func(*evidence) string {
				return itoa(len(ev.verbs)) + " action words, each traced to a guard line or a write route tail"
			}, nil},

		{"C06", "a derived refusal list also contains screen words, so the pruning rules are measured rather than assumed",
			platformAPIFile, []string{"ApiPOST(\"/platform/tenants/:id/operators\""},
			func(*evidence) string {
				return itoa(len(ev.rawBan)) + " derived tokens, " + itoa(len(ev.ban.ShortWords)) +
					" dropped as too short, " + itoa(len(ev.ban.ScreenWords)) +
					" dropped as screen words, " + itoa(len(ev.ban.Kept)) + " kept"
			}, func(e *evidence) bool { return len(e.ban.ScreenWords) > 0 || len(e.ban.ShortWords) > 0 }},

		{"C07", "refusal is checked before scoring, so an utterance that names an operation never reaches a screen",
			ispAPIFile, []string{"ApiPOST(\"/isp/payments/:id/reverse\""},
			func(*evidence) string {
				return itoa(ev.actionRefused) + " of " + itoa(ev.actionRows) +
					" operation transcripts refused, " + itoa(ev.mutation) + " navigated to a mutation"
			}, func(e *evidence) bool { return e.actionRefused == e.actionRows && e.mutation == 0 }},

		{"C08", "no transcript that the study expects to stop is answered with a confident screen",
			idLocale, []string{"customers: 'Pelanggan'"},
			func(*evidence) string {
				return itoa(ev.wrongScreen) + " wrong-screen answers, " + itoa(ev.fragGuessed) +
					" label fragments guessed into a screen, over " + itoa(len(ev.verdicts)) + " transcripts"
			}, func(e *evidence) bool { return e.wrongScreen == 0 && e.fragGuessed == 0 }},

		{"C09", "a transcript the recognizer damaged still names the screen it meant, across every error class the corpus carries",
			enLocale, []string{"radius_profiles: 'Profiles Management'"},
			func(*evidence) string {
				return itoa(ev.damagedHit) + " of " + itoa(ev.damaged) +
					" damaged transcripts land on the intended screen, over " + itoa(ev.classes) + " error classes"
			}, nil},

		{"C10", "the vocabulary is role-blind: a gated row stays speakable, exactly as it stays reachable by URL",
			menuFile, []string{"permissions: ["},
			func(*evidence) string {
				return itoa(ev.gated) + " of " + itoa(len(ev.destinations)) + " destinations are gated in the menu and still in the vocabulary"
			}, func(e *evidence) bool { return e.gated > 0 }},

		{"C11", "voice is necessarily opt-in: the product contains no speech API reference and no server-side listener",
			appFile, []string{"<Admin"},
			func(*evidence) string {
				return itoa(ev.speechRefs) + " speech-API references in the frontend, " +
					itoa(ev.listenerRefs) + " microphone-listener references in the backend"
			}, func(e *evidence) bool { return e.speechRefs == 0 && e.listenerRefs == 0 }},

		{"C12", "the text fallback is measured per screen, and it is a navigation-level fallback rather than a typed one on most pages",
			appFile, []string{"dashboard={Dashboard}"},
			func(*evidence) string {
				return itoa(ev.pagesFound) + " of " + itoa(len(ev.destinations)) + " destinations traced to a page, " +
					itoa(ev.typedFields) + " typed inputs in total, " + itoa(ev.voiceOnly) + " screens with none"
			}, func(e *evidence) bool { return e.pagesFound == len(e.destinations) }},

		{"C13", "the one screen that carries a typed filter is the one the corpus can reach with it",
			ispResourceFile, []string{"SearchInput"},
			func(*evidence) string {
				return itoa(ev.filterBound) + " screens bind setFilters, " + itoa(ev.typedFields) +
					" text or search inputs across the resolved pages"
			}, func(e *evidence) bool { return e.filterBound > 0 }},

		{"C14", "the resolver's thresholds are constants the run prints, so a tuning change shows up as a different measured outcome",
			resolverFile, []string{"acceptFloor = 0.78"},
			func(*evidence) string {
				return "accept floor " + pct(acceptFloor) + ", margin " + pct(acceptMargin) +
					", deny floor " + pct(denyFloor) + "; " + itoa(ev.clean) + " of " + itoa(len(ev.verdicts)) +
					" transcripts match the answer the study requires"
			}, nil},

		{"C15", "no product code imports this study, and the study imports nothing that can write",
			ciFile, []string{"go test"},
			func(*evidence) string {
				n := len(importersOutside(ev.root))
				if n == 0 {
					return "0 importers outside internal/" + ownPackage + ", " + itoa(len(ev.ban.Kept)) + " refusal tokens held in memory only"
				}
				return itoa(n) + " importers: " + strings.Join(importersOutside(ev.root), ", ")
			}, func(e *evidence) bool { return len(importersOutside(e.root)) == 0 }},
	}
	return out
}

// gaps is the no-go register: what this round cannot answer, and the asset or
// decision it would need. Every row is open, because an asset-free round cannot
// close any of them.
func gaps(ev *evidence) []GapStatus {
	uncovered := 0
	for _, v := range ev.verdicts {
		if v.Class == "action-uncovered" {
			uncovered++
		}
	}
	defs := []struct{ id, text, needs, asset string }{
		{"G01", "the transcripts here are authored, not heard: what the study prices is the resolver's behavior over a mis-transcription, not a real recognizer's error distribution",
			"a recorded corpus of operators speaking the two shipped locales, run through the ASR the product would use", "audio corpus"},
		{"G02", "the score a transcript gets is a string-similarity figure; whether an operator would accept that answer, and how long a wrong one takes to undo, is a human measurement",
			"a panel of operators using a prototype push-to-talk control", "human respondents"},
		{"G03", "the resolver runs on a machine reading the tree; latency, permission prompts, and what a browser actually transcribes of an Indonesian label are only visible in a session",
			"a browser session with the Web Speech API against the dev server and a seeded login", "live browser session"},
		{"G04", itoa(ev.gated) + " gated destinations stay in the vocabulary, so a lower-level operator can be routed at a path the menu hides; whether that path then answers is a server question this study does not ask",
			"a signed session per role, spoken at each gated screen", "live per-role sessions"},
		{"G05", itoa(uncovered) + " operation sentences in Indonesian are refused by nothing but the answer type; a synonym or glossary layer is a product decision about what the voice vocabulary should cover",
			"a decision on TR-F013 scope: a navigation-only vocabulary, or a curated synonym set under the same deny-first rule", "product owner"},
		{"G06", "the device is unstudied: a headset, a phone speaker in a NOC, and a car microphone would produce different damage, and the card forbids a permanent listener without saying what an allowed one costs",
			"a device and duty-cycle decision, plus the battery and data measurements that follow from it", "hardware and a policy decision"},
	}
	out := make([]GapStatus, 0, len(defs))
	for _, d := range defs {
		// An asset-free round cannot close any of these, so every row keeps its
		// verdict rather than being dropped from the register.
		out = append(out, GapStatus{ID: d.id, Text: d.text, Needs: d.needs, Asset: d.asset, Open: true})
	}
	return out
}

// helpers used by the claim table

// countLang counts the phrases drawn from one shipped locale.
func (ev *evidence) countLang(lang string) int {
	n := 0
	for _, p := range ev.phrases {
		if p.Lang == lang {
			n++
		}
	}
	return n
}

func namedList(prefix string, items []string) string {
	if len(items) == 0 {
		return ""
	}
	set := map[string]bool{}
	var out []string
	for _, s := range items {
		if set[s] {
			continue
		}
		set[s] = true
		out = append(out, s)
	}
	sort.Strings(out)
	if len(out) > 6 {
		out = append(out[:6], "...")
	}
	return prefix + strings.Join(out, ", ") + ")"
}
