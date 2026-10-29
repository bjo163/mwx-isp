package incidenttriage

import (
	"sort"
	"strings"
)

// Evidence is one backlink: the token that fired, the log line it was found on,
// and the product code line that writes that token.
type Evidence struct {
	Cause     string
	Token     string
	LineIndex int
	Site      string
}

// Verdict is what an engine says about one incident. There is deliberately no
// action, command, route or verb in this struct: "no automatic remediate" is a
// property of the type, and TestVerdictCarriesNoActionField checks it by
// reflection rather than by promise.
type Verdict struct {
	Engine     string
	Incident   string
	Cause      string
	Reason     string
	Evidence   []Evidence
	Unresolved bool
}

// Engine triages one incident at a time. Every engine here is deterministic:
// no model runs in this package, and the study says so.
type Engine interface {
	Name() string
	Triage(inc Incident) Verdict
}

// UnresolvedCause is what an abstention says.
const UnresolvedCause = "unresolved"

// sites maps each signal token to the first product code line that writes it.
type sites map[string]string

// buildSites walks the product tree once and records where each token appears
// on a code line. A token with no site is reported by claims.go, not dropped.
func buildSites(root string) (sites, error) {
	files, err := productFiles(root)
	if err != nil {
		return nil, err
	}
	out := sites{}
	for _, want := range Causes() {
		for _, s := range want.Signals {
			out[s.Token] = ""
		}
	}
	for _, path := range files {
		src, err := readFile(path)
		if err != nil {
			continue
		}
		lines := strings.Split(blankComments(src), "\n")
		for token := range out {
			if out[token] != "" || !strings.Contains(src, token) {
				continue
			}
			if n := locate(lines, token); n > 0 {
				rel := relToRoot(path)
				out[token] = rel + ":" + itoa(n)
			}
		}
	}
	return out, nil
}

// RuleEngine is the non-AI option the card asks for: a token table with
// evidence, no model, and an explicit abstention.
type RuleEngine struct {
	sites sites
}

// NewRuleEngine wires the rules to the code lines that prove their tokens.
func NewRuleEngine(root string) (*RuleEngine, error) {
	s, err := buildSites(root)
	if err != nil {
		return nil, err
	}
	return &RuleEngine{sites: s}, nil
}

// Name implements Engine.
func (e *RuleEngine) Name() string { return "rule-token-match" }

// Triage scores every cause by the number of its signals that appear in the
// incident. A tie abstains: naming one of two equal causes would be a guess,
// and a guess is what the criterion calls a hallucination.
func (e *RuleEngine) Triage(inc Incident) Verdict {
	type scored struct {
		cause Cause
		hits  []Evidence
	}
	var best, runner scored
	for _, c := range Causes() {
		var hits []Evidence
		for _, sig := range c.Signals {
			for _, l := range inc.Lines {
				if containsBounded(l.Text(), sig.Token) {
					hits = append(hits, Evidence{
						Cause: c.ID, Token: sig.Token, LineIndex: l.Index, Site: e.sites[sig.Token],
					})
					break
				}
			}
		}
		if len(hits) == 0 {
			continue
		}
		sort.SliceStable(hits, func(i, j int) bool { return hits[i].Token < hits[j].Token })
		cur := scored{cause: c, hits: hits}
		if len(cur.hits) > len(best.hits) {
			runner = best
			best = cur
		} else if len(cur.hits) > len(runner.hits) {
			runner = cur
		}
	}
	if len(best.hits) == 0 {
		return Verdict{Engine: e.Name(), Incident: inc.ID, Cause: UnresolvedCause,
			Reason: "no token from the taxonomy appears in these lines", Unresolved: true}
	}
	if len(runner.hits) == len(best.hits) {
		return Verdict{Engine: e.Name(), Incident: inc.ID, Cause: UnresolvedCause,
			Reason: "two causes match the same number of tokens: " + best.cause.ID +
				" and " + runner.cause.ID + "; naming one would be a guess",
			Evidence: best.hits, Unresolved: true}
	}
	return Verdict{Engine: e.Name(), Incident: inc.ID, Cause: best.cause.ID,
		Reason: describe(best.hits), Evidence: best.hits}
}

func describe(hits []Evidence) string {
	var toks []string
	for _, h := range hits {
		toks = append(toks, `"`+h.Token+`" on line `+itoa(h.LineIndex))
	}
	return "matched " + strings.Join(toks, ", ")
}

// NamespaceBaseline is the second non-AI option: read only which subsystem
// complained, never what it said. It is fitted on the corpus labels so the
// comparison is honest - it gets the same chance to learn that the rule table
// already had by hand.
type NamespaceBaseline struct {
	byKey map[string]string
	// Ambiguous counts the namespace-plus-level keys whose corpus labels do not
	// agree, which is the baseline's whole problem and a number the memo quotes.
	Ambiguous int
}

// FitNamespaceBaseline learns the most frequent label per namespace plus level,
// ties broken by the label name so the result does not depend on map order.
func FitNamespaceBaseline(pairs []Pairing) *NamespaceBaseline {
	counts := map[string]map[string]int{}
	for _, p := range pairs {
		if len(p.Case.Labels) != 1 {
			continue
		}
		key := p.Incident.Namespace + "|" + p.Incident.Lines[0].Level
		if counts[key] == nil {
			counts[key] = map[string]int{}
		}
		counts[key][p.Case.Labels[0]]++
	}
	b := &NamespaceBaseline{byKey: map[string]string{}, Ambiguous: 0}
	for key, per := range counts {
		labels := make([]string, 0, len(per))
		for l := range per {
			labels = append(labels, l)
		}
		sort.Slice(labels, func(i, j int) bool {
			if per[labels[i]] != per[labels[j]] {
				return per[labels[i]] > per[labels[j]]
			}
			return labels[i] < labels[j]
		})
		if len(per) > 1 {
			b.Ambiguous++
		}
		b.byKey[key] = labels[0]
	}
	return b
}

// Name implements Engine.
func (b *NamespaceBaseline) Name() string { return "namespace-baseline" }

// Triage answers from the subsystem name alone and carries no evidence, which
// is exactly the shape the scorer counts as a hallucination.
func (b *NamespaceBaseline) Triage(inc Incident) Verdict {
	key := inc.Namespace + "|" + inc.Lines[0].Level
	cause, ok := b.byKey[key]
	if !ok {
		return Verdict{Engine: b.Name(), Incident: inc.ID, Cause: UnresolvedCause,
			Reason: "this namespace and level pair was not in the corpus", Unresolved: true}
	}
	return Verdict{Engine: b.Name(), Incident: inc.ID, Cause: cause,
		Reason: "namespace " + inc.Namespace + " and level " + inc.Lines[0].Level +
			" map to this label in the corpus"}
}

// AlwaysName is the counterfactual: the same token rules, but it never abstains.
// It stands in for a classifier that fills a gap with its most likely class
// instead of saying "I do not know" - the failure mode the criterion wants
// measured. It is not a language model and is never described as one.
type AlwaysName struct {
	rules  *RuleEngine
	fallbc string
}

// NewAlwaysName wraps the rule engine with a most-frequent-label fallback.
func NewAlwaysName(root string, pairs []Pairing) (*AlwaysName, error) {
	r, err := NewRuleEngine(root)
	if err != nil {
		return nil, err
	}
	tally := map[string]int{}
	for _, p := range pairs {
		if len(p.Case.Labels) == 1 && p.Case.Labels[0] != Unresolved {
			tally[p.Case.Labels[0]]++
		}
	}
	labels := make([]string, 0, len(tally))
	for l := range tally {
		labels = append(labels, l)
	}
	sort.Slice(labels, func(i, j int) bool {
		if tally[labels[i]] != tally[labels[j]] {
			return tally[labels[i]] > tally[labels[j]]
		}
		return labels[i] < labels[j]
	})
	fallback := UnresolvedCause
	if len(labels) > 0 {
		fallback = labels[0]
	}
	return &AlwaysName{rules: r, fallbc: fallback}, nil
}

// Name implements Engine.
func (a *AlwaysName) Name() string { return "always-name" }

// Triage runs the token rules and, on any abstention, names the most frequent
// class in the corpus with no evidence behind it.
func (a *AlwaysName) Triage(inc Incident) Verdict {
	v := a.rules.Triage(inc)
	if !v.Unresolved {
		v.Engine = a.Name()
		return v
	}
	return Verdict{Engine: a.Name(), Incident: inc.ID, Cause: a.fallbc,
		Reason: "no rule matched, so the most frequent class in the corpus was named: " +
			a.fallbc}
}
