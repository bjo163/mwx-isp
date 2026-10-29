package incidenttriage

import (
	"fmt"
	"sort"
	"time"
)

// EngineScore is one engine's record over the corpus.
type EngineScore struct {
	Engine      string
	Decided     int
	Correct     int
	Wrong       int
	Abstained   int
	Hallucinate int
}

// Accuracy is measured over named verdicts only, so an abstention is not paid
// for as a wrong answer - and not hidden either, because Abstained is reported.
func (s EngineScore) Accuracy() float64 {
	if s.Decided == 0 {
		return 0
	}
	return float64(s.Correct) / float64(s.Decided)
}

// HallucinationRate is the criterion's number: of the incidents the engine
// named a cause for, how many of those names had no support in the lines.
func (s EngineScore) HallucinationRate() float64 {
	if s.Decided == 0 {
		return 0
	}
	return float64(s.Hallucinate) / float64(s.Decided)
}

// String renders the headline numbers for a failure message.
func (s EngineScore) String() string {
	return fmt.Sprintf("%s: %d named, %d correct, %d wrong, %d hallucinated, %d abstained",
		s.Engine, s.Decided, s.Correct, s.Wrong, s.Hallucinate, s.Abstained)
}

// PerCause is the rule engine's confusion detail, so a class the rules never
// reach is visible instead of averaged away.
type PerCause struct {
	Cause     string
	Support   int
	Predicted int
	Exact     int
}

// Precision reads Exact over the class's predictions.
func (p PerCause) Precision() float64 {
	if p.Predicted == 0 {
		return 0
	}
	return float64(p.Exact) / float64(p.Predicted)
}

// Recall reads Exact over the class's labeled support.
func (p PerCause) Recall() float64 {
	if p.Support == 0 {
		return 0
	}
	return float64(p.Exact) / float64(p.Support)
}

// Totals carries the numbers the memo quotes.
type Totals struct {
	Cases           int
	Lines           int
	Labeled         int
	AbstainCases    int
	AbstainsTaken   int
	SignalsTotal    int
	SignalsGrounded int
	SecretValues    int
	Leaked          int
	GroupWindow     time.Duration
	Claims          int
	ClaimsConfirmed int
	Gaps            int
	GapsOpen        int
	// BaselineKeys and BaselineAmbiguous describe the namespace-plus-level rule
	// the baseline learns: how many keys it fitted, and how many of them saw two
	// different labels in the same corpus. Ambiguity is the reason a subsystem
	// name cannot carry a cause.
	BaselineKeys      int
	BaselineAmbiguous int
	ActionFields      int
	NamespaceSites    int
	JobSites          int
}

// Result is one full run of the study.
type Result struct {
	Totals   Totals
	Stats    SanitizeStats
	Scores   []EngineScore
	Causes   []PerCause
	Verdicts map[string][]Verdict
	Pairs    []Pairing
	Sites    map[string]string
	// Namespaces and Jobs are, per literal, what product code attaches to the two
	// fields the log export filters on.
	Namespaces map[string]int
	Jobs       map[string]int
	Claims     []ClaimStatus
	Gaps       []GapStatus
	Actions    []string
	Leaked     []string
}

// Run executes the whole study against the repository rooted at root.
func Run(root string) (*Result, error) {
	cases := Dataset()
	intake := Parse(RawLines(cases))
	secrets := Secrets(cases)
	sanitized, stats := intake.Sanitize(secrets)

	incidents := Group(sanitized, 2*time.Minute)
	pairs, err := Pair(cases, incidents)
	if err != nil {
		return nil, err
	}
	rules, err := NewRuleEngine(root)
	if err != nil {
		return nil, err
	}
	baseline := FitNamespaceBaseline(pairs)
	always, err := NewAlwaysName(root, pairs)
	if err != nil {
		return nil, err
	}
	engines := []Engine{rules, baseline, always}

	res := &Result{
		Stats:    stats,
		Verdicts: map[string][]Verdict{},
		Pairs:    pairs,
		Sites:    rules.sites,
		Leaked:   stats.LeakedValues,
		Totals: Totals{
			Cases:        len(cases),
			Lines:        len(sanitized),
			SecretValues: len(secrets),
			Leaked:       len(stats.LeakedValues),
			GroupWindow:  2 * time.Minute,
		},
	}
	for _, c := range cases {
		if c.Labels[0] == Unresolved {
			res.Totals.AbstainCases++
			continue
		}
		res.Totals.Labeled++
	}

	for _, e := range engines {
		score := EngineScore{Engine: e.Name()}
		for _, p := range pairs {
			v := e.Triage(p.Incident)
			res.Verdicts[e.Name()] = append(res.Verdicts[e.Name()], v)
			want := p.Case.Labels[0]
			switch {
			case v.Cause == UnresolvedCause:
				score.Abstained++
				if want == Unresolved {
					res.Totals.AbstainsTaken++
				}
			case hallucinated(v, p.Incident):
				score.Decided++
				score.Hallucinate++
				if want == v.Cause {
					score.Correct++
				} else {
					score.Wrong++
				}
			default:
				score.Decided++
				if want == v.Cause {
					score.Correct++
				} else {
					score.Wrong++
				}
			}
		}
		res.Scores = append(res.Scores, score)
	}

	res.Causes = perCause(pairs, res.Verdicts[rules.Name()])
	res.Actions = measuredActions(root)
	res.Claims = verifyClaims(root)
	res.Gaps = verifyGaps(root)
	res.Namespaces = logFieldLiterals(root, "namespace")
	res.Jobs = logFieldLiterals(root, "job")
	res.Totals.NamespaceSites = sumValues(res.Namespaces)
	res.Totals.JobSites = sumValues(res.Jobs)
	res.Totals.BaselineKeys = len(baseline.byKey)
	res.Totals.BaselineAmbiguous = baseline.Ambiguous
	res.Totals.ActionFields = len(actionableFields())
	res.Totals.SignalsTotal = len(allSignals())
	res.Totals.SignalsGrounded = groundedSignals(rules)
	for _, c := range res.Claims {
		if c.Confirmed {
			res.Totals.ClaimsConfirmed++
		}
	}
	res.Totals.Claims = len(res.Claims)
	for _, g := range res.Gaps {
		if g.Open {
			res.Totals.GapsOpen++
		}
	}
	res.Totals.Gaps = len(res.Gaps)
	return res, nil
}

// hallucinated is the measured definition: a named cause whose evidence does
// not hold. Empty evidence, a token that is not in the incident, a line index
// the incident does not contain, or a token with no product code site behind it
// all count, because each one is a claim nothing in the log supports.
func hallucinated(v Verdict, inc Incident) bool {
	if v.Cause == UnresolvedCause {
		return false
	}
	if len(v.Evidence) == 0 {
		return true
	}
	seen := map[int]bool{}
	for _, l := range inc.Lines {
		seen[l.Index] = true
	}
	text := inc.Text()
	for _, e := range v.Evidence {
		if !seen[e.LineIndex] || e.Site == "" || !containsBounded(text, e.Token) {
			return true
		}
	}
	return false
}

func perCause(pairs []Pairing, verdicts []Verdict) []PerCause {
	byCause := map[string]*PerCause{}
	get := func(id string) *PerCause {
		if byCause[id] == nil {
			byCause[id] = &PerCause{Cause: id}
		}
		return byCause[id]
	}
	for _, p := range pairs {
		if p.Case.Labels[0] != Unresolved {
			get(p.Case.Labels[0]).Support++
		}
	}
	for i, p := range pairs {
		if i >= len(verdicts) {
			continue
		}
		v := verdicts[i]
		if v.Cause == UnresolvedCause {
			continue
		}
		pc := get(v.Cause)
		pc.Predicted++
		if p.Case.Labels[0] == v.Cause {
			pc.Exact++
		}
	}
	out := make([]PerCause, 0, len(byCause))
	for _, pc := range byCause {
		out = append(out, *pc)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Support != out[j].Support {
			return out[i].Support > out[j].Support
		}
		return out[i].Cause < out[j].Cause
	})
	return out
}

func allSignals() []string {
	var out []string
	for _, c := range Causes() {
		for _, s := range c.Signals {
			out = append(out, s.Token)
		}
	}
	sort.Strings(out)
	return out
}

func groundedSignals(e *RuleEngine) int {
	n := 0
	for _, s := range allSignals() {
		if e.sites[s] != "" {
			n++
		}
	}
	return n
}
