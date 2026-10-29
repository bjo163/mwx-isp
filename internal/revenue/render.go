package revenue

import (
	"fmt"
	"sort"
	"strings"
)

// RenderMarkdown turns one audit run into the tables an auditor reads: totals,
// the per-kind counts, every finding with the rule it traces to, the rules that
// were cited, and what the ledger could not answer.
func RenderMarkdown(report Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Findings: %d (%d not quantified) · Exposure: IDR %d · Tenants: %d\n\n",
		report.Totals.Findings, report.Totals.Unquantified, report.Totals.Exposure, len(report.ByTenant))

	out.WriteString("## Findings per kind\n\n")
	out.WriteString("| kind | findings |\n| --- | ---: |\n")
	kinds := make([]string, 0, len(report.Totals.ByKind))
	for kind := range report.Totals.ByKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	for _, kind := range kinds {
		fmt.Fprintf(&out, "| %s | %d |\n", cell(kind), report.Totals.ByKind[kind])
	}

	out.WriteString("\n## Tenants\n\n| tenant | findings | exposure (IDR) |\n| --- | ---: | ---: |\n")
	for _, tenant := range report.ByTenant {
		fmt.Fprintf(&out, "| %d | %d | %d |\n", tenant.TenantID, tenant.Findings, tenant.Exposure)
	}

	out.WriteString("\n## Findings\n\n")
	out.WriteString("| tenant | kind | subject | rule | exposure | evidence |\n")
	out.WriteString("| ---: | --- | --- | --- | --- | --- |\n")
	for _, finding := range report.Findings {
		exposure := "not quantified"
		if finding.Quantified {
			exposure = fmt.Sprintf("IDR %d", finding.Exposure)
		}
		fmt.Fprintf(&out, "| %d | %s | %s | %s | %s | %s |\n",
			finding.TenantID, cell(finding.Kind), cell(finding.Subject),
			cell(finding.Rule.ID), exposure, cell(strings.Join(finding.Evidence, "; ")))
	}

	out.WriteString("\n## Rules cited\n\n| rule | function | source |\n| --- | --- | --- |\n")
	for _, rule := range citedRules(report) {
		fmt.Fprintf(&out, "| %s | %s | `%s` |\n", cell(rule.ID), cell(rule.Function), rule.Source)
	}

	out.WriteString("\n## What the ledger cannot answer\n\n")
	for _, gap := range report.Gaps {
		fmt.Fprintf(&out, "- %s\n", gap)
	}
	return out.String()
}

// citedRules returns the distinct rules a report used, in first-cited order.
func citedRules(report Report) []Rule {
	seen := map[string]bool{}
	var rules []Rule
	for _, finding := range report.Findings {
		if seen[finding.Rule.ID] {
			continue
		}
		seen[finding.Rule.ID] = true
		rules = append(rules, finding.Rule)
	}
	return rules
}

// cell escapes pipes so no stored value can break a markdown table.
func cell(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
