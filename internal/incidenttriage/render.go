package incidenttriage

import (
	"sort"
	"strconv"
	"strings"
)

// RenderMarkdown emits every table the memo splices in, from the same run that
// produced the numbers in the prose.
func RenderMarkdown(r *Result) string {
	var b strings.Builder
	pct := func(f float64) string { return strconv.FormatFloat(f*100, 'f', 1, 64) + "%" }

	b.WriteString("## Verdict\n\n| Measure | Value |\n| --- | ---: |\n")
	row(&b, "Incidents in the corpus", strconv.Itoa(r.Totals.Cases))
	row(&b, "Log lines in the corpus", strconv.Itoa(r.Totals.Lines))
	row(&b, "Incidents with a single labeled cause", strconv.Itoa(r.Totals.Labeled))
	row(&b, "Incidents labeled unresolved", strconv.Itoa(r.Totals.AbstainCases))
	row(&b, "Cause classes in the taxonomy", strconv.Itoa(len(Causes())))
	row(&b, "Signals in the taxonomy", strconv.Itoa(r.Totals.SignalsTotal))
	row(&b, "Signals resolved to a product code line", strconv.Itoa(r.Totals.SignalsGrounded))
	row(&b, "Incidents the grouping produced", strconv.Itoa(len(r.Pairs)))
	row(&b, "Grouping window", r.Totals.GroupWindow.String())
	row(&b, "Namespace-plus-level keys the baseline fitted", strconv.Itoa(r.Totals.BaselineKeys))
	row(&b, "Of those, keys whose corpus labels disagree", strconv.Itoa(r.Totals.BaselineAmbiguous))
	row(&b, "Namespace literals product code writes", strconv.Itoa(len(r.Namespaces)))
	row(&b, "Log lines that attach a namespace", strconv.Itoa(r.Totals.NamespaceSites))
	row(&b, "Job literals product code writes", strconv.Itoa(len(r.Jobs)))
	row(&b, "Log lines that attach a job", strconv.Itoa(r.Totals.JobSites))
	for _, s := range r.Scores {
		row(&b, s.Engine+": named a cause", strconv.Itoa(s.Decided))
		row(&b, s.Engine+": correct", strconv.Itoa(s.Correct))
		row(&b, s.Engine+": wrong", strconv.Itoa(s.Wrong))
		row(&b, s.Engine+": abstained", strconv.Itoa(s.Abstained))
		row(&b, s.Engine+": hallucinated", strconv.Itoa(s.Hallucinate))
		row(&b, s.Engine+": accuracy", pct(s.Accuracy()))
		row(&b, s.Engine+": hallucination rate", pct(s.HallucinationRate()))
	}
	row(&b, "Secret values the corpus planted", strconv.Itoa(r.Totals.SecretValues))
	row(&b, "Secret values that survived redaction", strconv.Itoa(r.Totals.Leaked))
	row(&b, "Mutating verbs measured from the admin API", strconv.Itoa(len(r.Actions)))
	row(&b, "Verdict fields that could carry an action", strconv.Itoa(r.Totals.ActionFields))
	row(&b, "Verdicts naming one of those verbs", strconv.Itoa(actionNamesIn(r)))
	row(&b, "Claims confirmed against product code",
		strconv.Itoa(r.Totals.ClaimsConfirmed)+" of "+strconv.Itoa(r.Totals.Claims))
	row(&b, "Gaps still open", strconv.Itoa(r.Totals.GapsOpen)+" of "+strconv.Itoa(r.Totals.Gaps))

	b.WriteString("\n## The labeled corpus\n\n")
	b.WriteString("| Label | Incidents |\n| --- | ---: |\n")
	var labeled []Case
	for _, p := range r.Pairs {
		labeled = append(labeled, p.Case)
	}
	tally := LabelCount(labeled)
	keys := make([]string, 0, len(tally))
	for k := range tally {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if tally[keys[i]] != tally[keys[j]] {
			return tally[keys[i]] > tally[keys[j]]
		}
		return keys[i] < keys[j]
	})
	for _, k := range keys {
		row(&b, k, strconv.Itoa(tally[k]))
	}
	b.WriteString("\n| Case | Subsystem | Job | Lines | Label | Why that label |\n")
	b.WriteString("| --- | --- | --- | ---: | --- | --- |\n")
	b.WriteString("<!-- (none) in Subsystem means the code path writing that message attaches no namespace field; no product line attaches a job. -->\n")
	for _, p := range r.Pairs {
		b.WriteString("| " + cell(p.Case.ID) + " | " + cell(orNone(p.Case.Namespace)) + " | " +
			cell(orNone(p.Case.Job)) + " | " + strconv.Itoa(len(p.Incident.Lines)) + " | " +
			cell(p.Case.Labels[0]) + " | " + cell(p.Case.Why) + " |\n")
	}

	b.WriteString("\n## What redaction touched\n\n| Measure | Value |\n| --- | ---: |\n")
	row(&b, "Structured lines", strconv.Itoa(r.Stats.Structured))
	row(&b, "Fields keyed secret", strconv.Itoa(r.Stats.SecretKeys))
	row(&b, "Fields keyed customer PII", strconv.Itoa(r.Stats.PIIKeys))
	row(&b, "Fields keyed address", strconv.Itoa(r.Stats.AddressKeys))
	row(&b, "Messages changed by the free-text sweep", strconv.Itoa(r.Stats.MessagesSwept))
	if len(r.Leaked) > 0 {
		b.WriteString("\nLeaked: " + cell(list(r.Leaked)) + "\n")
	}

	b.WriteString("\n## What the export's own filters carry\n\n")
	b.WriteString("The operator log export lets an operator filter on namespace and job. " +
		"Counted from product code, that is what those two fields actually hold:\n\n")
	b.WriteString("| Field | Literal | Times written |\n| --- | --- | ---: |\n")
	fields := []struct {
		name   string
		counts map[string]int
	}{{"namespace", r.Namespaces}, {"job", r.Jobs}}
	for _, f := range fields {
		names := make([]string, 0, len(f.counts))
		for k := range f.counts {
			names = append(names, k)
		}
		sort.Slice(names, func(i, j int) bool {
			if f.counts[names[i]] != f.counts[names[j]] {
				return f.counts[names[i]] > f.counts[names[j]]
			}
			return names[i] < names[j]
		})
		if len(names) == 0 {
			b.WriteString("| " + cell(f.name) + " | no literal written anywhere | 0 |\n")
			continue
		}
		for _, n := range names {
			b.WriteString("| " + cell(f.name) + " | " + cell(`"`+n+`"`) + " | " +
				strconv.Itoa(f.counts[n]) + " |\n")
		}
	}

	b.WriteString("\n## The taxonomy and where its tokens come from\n\n")
	b.WriteString("| Cause | Signal | First product code line that writes it |\n| --- | --- | --- |\n")
	for _, c := range Causes() {
		for _, s := range c.Signals {
			b.WriteString("| " + cell(c.ID) + " | " + cell(`"`+s.Token+`"`) + " | " +
				cell(signalSite(r, s.Token)) + " |\n")
		}
	}

	b.WriteString("\n## Three engines over the same corpus\n\n")
	b.WriteString("| Engine | Named | Correct | Wrong | Hallucinated | Abstained | Accuracy | Hallucination rate |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, s := range r.Scores {
		b.WriteString("| " + cell(s.Engine) + " | " + strconv.Itoa(s.Decided) + " | " +
			strconv.Itoa(s.Correct) + " | " + strconv.Itoa(s.Wrong) + " | " +
			strconv.Itoa(s.Hallucinate) + " | " + strconv.Itoa(s.Abstained) + " | " +
			pct(s.Accuracy()) + " | " + pct(s.HallucinationRate()) + " |\n")
	}

	b.WriteString("\n## Per-cause detail for the rule engine\n\n")
	b.WriteString("| Cause | Labeled support | Predicted | Exact | Precision | Recall |\n")
	b.WriteString("| --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, p := range r.Causes {
		b.WriteString("| " + cell(p.Cause) + " | " + strconv.Itoa(p.Support) + " | " +
			strconv.Itoa(p.Predicted) + " | " + strconv.Itoa(p.Exact) + " | " +
			pct(p.Precision()) + " | " + pct(p.Recall()) + " |\n")
	}

	b.WriteString("\n## What each engine said, incident by incident\n\n")
	b.WriteString("| Case | Label | " + strings.Join(engineNames(r), " | ") + " |\n")
	b.WriteString("| --- | --- |" + strings.Repeat(" --- |", len(r.Scores)) + "\n")
	for i, p := range r.Pairs {
		b.WriteString("| " + cell(p.Case.ID) + " | " + cell(p.Case.Labels[0]) + " |")
		for _, s := range r.Scores {
			v := r.Verdicts[s.Engine][i]
			b.WriteString(" " + cell(verdictCell(v)) + " |")
		}
		b.WriteString("\n")
	}

	b.WriteString("\n## No automatic remediate\n\n")
	b.WriteString("Verdict fields that could carry an action: " +
		strconv.Itoa(r.Totals.ActionFields) + " - " +
		"`TestVerdictCarriesNoActionField` reads the struct by reflection.\n\n")
	b.WriteString("Mutating verbs the admin API accepts, measured from its own route " +
		"registrations: " + cell(list(r.Actions)) + ".\n\n")
	b.WriteString("Verdicts naming any of them: " + strconv.Itoa(actionNamesIn(r)) + ".\n")

	b.WriteString("\n## Statements the study rests on\n\n")
	b.WriteString("| ID | Statement | Evidence | Status |\n| --- | --- | --- | --- |\n")
	for _, c := range r.Claims {
		status := "unconfirmed"
		switch {
		case c.Broken:
			status = "broken"
		case c.Confirmed:
			status = "confirmed"
		}
		b.WriteString("| " + cell(c.ID) + " | " + cell(c.Statement) + " | " +
			cell(list(c.Hits)) + " | " + cell(status) + " |\n")
	}

	b.WriteString("\n## Gaps this study does not close\n\n")
	b.WriteString("| ID | Gap | Scope | Status | Evidence |\n| --- | --- | --- | --- | --- |\n")
	for _, g := range r.Gaps {
		status := "open"
		if g.Vacated {
			status = "vacated"
		}
		b.WriteString("| " + cell(g.ID) + " | " + cell(g.Statement) + " | " +
			cell(strings.Join(g.Dirs, ", ")) + " | " + cell(status) + " | " +
			cell(list(g.Hits)) + " |\n")
	}
	return b.String()
}

func engineNames(r *Result) []string {
	var out []string
	for _, s := range r.Scores {
		out = append(out, s.Engine)
	}
	return out
}

func verdictCell(v Verdict) string {
	if v.Cause == UnresolvedCause {
		return "abstain"
	}
	if len(v.Evidence) == 0 {
		return v.Cause + " (no evidence)"
	}
	return v.Cause + " (" + strconv.Itoa(len(v.Evidence)) + " links)"
}

// actionNamesIn counts rendered verdicts that name a mutating verb the console
// accepts. It is expected to be zero and the report says so out loud.
func actionNamesIn(r *Result) int {
	n := 0
	for _, s := range r.Scores {
		for _, v := range r.Verdicts[s.Engine] {
			if len(namedActions(v.Cause+" "+v.Reason+" "+evidenceTokens(v), r.Actions)) > 0 {
				n++
			}
		}
	}
	return n
}

func evidenceTokens(v Verdict) string {
	var toks []string
	for _, e := range v.Evidence {
		toks = append(toks, e.Token)
	}
	sort.Strings(toks)
	return strings.Join(toks, " ")
}

func signalSite(r *Result, token string) string {
	return cellOrNone(r.Sites[token])
}

// cellOrNone names a token with no product code line behind it instead of
// printing an empty cell, because that is a finding rather than a gap.
func cellOrNone(s string) string {
	if s == "" {
		return "not found in product code"
	}
	return s
}

// orNone marks an absent corpus value as deliberate rather than missing.
func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func row(b *strings.Builder, k, v string) {
	b.WriteString("| " + cell(k) + " | " + cell(v) + " |\n")
}

func list(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

func cell(s string) string { return strings.ReplaceAll(s, "|", "\\|") }
