package teapreadiness

import (
	"fmt"
	"strings"
)

// RenderMarkdown turns a report into the document a reviewer reads: the verdict
// per mechanism with the clause behind it, the reuse list, the build list, and
// the questions the scan cannot answer.
func RenderMarkdown(report Report) string {
	var out strings.Builder
	fmt.Fprintf(&out, "Mechanics: %d · ready %d · partial %d · blocked %d · missing %d\n\n",
		report.Totals.Mechanics, report.Totals.Ready, report.Totals.Partial,
		report.Totals.Blocked, report.Totals.Missing)

	out.WriteString("## Reusable now\n\n")
	out.WriteString("| mechanism | clause | evidence |\n | --- | --- | --- |\n")
	written := 0
	for _, f := range report.Findings {
		if f.Status != StatusReady && f.Status != StatusPartial {
			continue
		}
		fmt.Fprintf(&out, "| %s | %s | %s |\n", cell(f.Mechanic.ID), cell(f.Mechanic.Source.String()), cell(evidenceText(f.Support)))
		written++
	}
	if written == 0 {
		out.WriteString("| (none) |  |  |\n")
	}

	out.WriteString("\n## Blocked by existing choices\n\n")
	out.WriteString("| mechanism | clause | contradicted by |\n | --- | --- | --- |\n")
	written = 0
	for _, f := range report.Findings {
		if f.Status != StatusBlocked {
			continue
		}
		fmt.Fprintf(&out, "| %s | %s | %s |\n", cell(f.Mechanic.ID), cell(f.Mechanic.Source.String()), cell(evidenceDetail(f.Against)))
		written++
	}
	if written == 0 {
		out.WriteString("| (none) |  |  |\n")
	}

	out.WriteString("\n## To build\n\n")
	for _, f := range report.Findings {
		if f.Status != StatusMissing {
			continue
		}
		fmt.Fprintf(&out, "- `%s` — %s · %s\n", cell(f.Mechanic.ID), cell(f.Mechanic.Title), cell(f.Mechanic.Source.String()))
	}

	out.WriteString("\n## Notes per mechanism\n\n")
	for _, f := range report.Findings {
		fmt.Fprintf(&out, "- `%s` (%s): %s\n", cell(f.Mechanic.ID), f.Status, cell(f.Note))
	}

	out.WriteString("\n## What this scan cannot answer\n\n")
	for _, u := range report.Unknown {
		fmt.Fprintf(&out, "- %s — %s\n", cell(u.Topic), cell(u.Reason))
	}
	return out.String()
}

// evidenceText joins the locations a verdict points at.
func evidenceText(items []Evidence) string {
	parts := make([]string, 0, len(items))
	for _, e := range items {
		parts = append(parts, fmt.Sprintf("%s:%d", e.File, e.Line))
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// evidenceDetail adds the source line, used where the exact code is the claim —
// a blocked mechanism is only convincing if the pin that blocks it is visible.
func evidenceDetail(items []Evidence) string {
	parts := make([]string, 0, len(items))
	for _, e := range items {
		parts = append(parts, e.String())
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " · ")
}

// cell makes a value safe inside a Markdown table row.
func cell(in string) string {
	return strings.ReplaceAll(in, "|", "\\|")
}
