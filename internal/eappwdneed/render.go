// Markdown rendering for the EAP-PWD need verification.

package eappwdneed

import (
	"strconv"
	"strings"
)

// RenderMarkdown turns a verification run into the tables the M12 memo quotes,
// so the memo's numbers come from the same code path as its claims.
func RenderMarkdown(r Report) string {
	var b strings.Builder
	t := r.Totals
	b.WriteString("Verdict: ")
	b.WriteString(strconv.Itoa(t.Confirmed))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(t.Claims))
	b.WriteString(" need claims confirmed, ")
	b.WriteString(strconv.Itoa(t.AbsencesHeld))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(t.Absences))
	b.WriteString(" gaps still open, ")
	b.WriteString(strconv.Itoa(t.RFCFindings))
	b.WriteString(" vendoring finding(s) across ")
	b.WriteString(strconv.Itoa(t.DocsCarried))
	b.WriteString(" carried specification(s).\n\n")

	b.WriteString("## Need claims\n\n")
	b.WriteString("| Claim | Verdict | Evidence |\n| --- | --- | --- |\n")
	for _, f := range r.Findings {
		b.WriteString("| ")
		b.WriteString(cell(f.Claim.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(f.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(f.Evidence))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Gaps that would be duplicated\n\n")
	b.WriteString("| Negative claim | Verdict | Locations |\n| --- | --- | --- |\n")
	for _, a := range r.Absences {
		b.WriteString("| ")
		b.WriteString(cell(a.Absence.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(a.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(strings.Join(a.Hits, ", ")))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Cited specifications\n\n")
	b.WriteString("| Spec | Verdict | Text carried in | File named for the topic | What that file is |\n| --- | --- | --- | --- | --- |\n")
	for _, s := range r.Vendoring.Specs {
		b.WriteString("| ")
		b.WriteString(cell(s.Spec.ID + " (RFC " + s.Spec.Number + " " + s.Spec.Topic + ")"))
		b.WriteString(" | ")
		b.WriteString(cell(string(s.State)))
		b.WriteString(" | ")
		b.WriteString(cell(s.HeldBy))
		b.WriteString(" | ")
		b.WriteString(cell(s.PromisedBy))
		b.WriteString(" | ")
		b.WriteString(cell(s.PromisedTitle))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Where each citation comes from\n\n")
	for _, s := range r.Vendoring.Specs {
		b.WriteString("- ")
		b.WriteString(s.Spec.ID)
		b.WriteString(": ")
		b.WriteString(s.Spec.CitedBy)
		b.WriteString("\n")
	}

	b.WriteString("\n## Number mismatches\n\n")
	if len(r.Vendoring.NumberMismatch) == 0 {
		b.WriteString("None - every carried filename matches the number the document prints about itself.\n")
	}
	for _, d := range r.Vendoring.NumberMismatch {
		b.WriteString("- ")
		b.WriteString(d.String())
		b.WriteString("\n")
	}

	b.WriteString("\n## What each claim implies\n\n")
	for _, f := range r.Findings {
		if f.Status != StatusConfirmed {
			continue
		}
		b.WriteString("- ")
		b.WriteString(f.Claim.ID)
		b.WriteString(": ")
		b.WriteString(f.Note)
		b.WriteString("\n")
	}
	return b.String()
}

// cell keeps a code fragment from breaking a Markdown table cell.
func cell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
