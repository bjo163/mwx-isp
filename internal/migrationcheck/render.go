package migrationcheck

import (
	"strconv"
	"strings"
)

// RenderMarkdown turns one verification run into the artifact ECO-006 asks for:
// the verdict on the criterion, the migration checklist the operator would hand a
// client, and the effort items sized from measured quantities.
func RenderMarkdown(r Report) string {
	var b strings.Builder
	confirmed, held := 0, 0
	for _, f := range r.Findings {
		if f.Status == StatusConfirmed {
			confirmed++
		}
	}
	for _, a := range r.Absences {
		if a.Status == StatusConfirmed {
			held++
		}
	}
	b.WriteString("## Verdict\n\n")
	b.WriteString("Migration claims verified: ")
	b.WriteString(strconv.Itoa(confirmed))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(len(r.Findings)))
	b.WriteString("; gaps still open: ")
	b.WriteString(strconv.Itoa(held))
	b.WriteString("/")
	b.WriteString(strconv.Itoa(len(r.Absences)))
	b.WriteString("; import columns measured: ")
	b.WriteString(strconv.Itoa(len(r.Surface.Columns)))
	b.WriteString("; row rejection rules measured: ")
	b.WriteString(strconv.Itoa(len(r.Surface.Rejections)))
	b.WriteString(".\n\n")

	b.WriteString("## Claims the service rests on\n\n")
	b.WriteString("| Claim | Verdict | Evidence | Why it matters |\n| --- | --- | --- | --- |\n")
	for _, f := range r.Findings {
		b.WriteString("| ")
		b.WriteString(cell(f.Claim.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(f.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(f.Evidence))
		b.WriteString(" | ")
		b.WriteString(cell(f.Claim.Statement))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Gaps a migration service still has to close\n\n")
	b.WriteString("| Gap | Verdict | Found at | What it costs a migration |\n| --- | --- | --- | --- |\n")
	for _, a := range r.Absences {
		b.WriteString("| ")
		b.WriteString(cell(a.Absence.ID))
		b.WriteString(" | ")
		b.WriteString(cell(string(a.Status)))
		b.WriteString(" | ")
		b.WriteString(cell(strings.Join(a.Hits, ", ")))
		b.WriteString(" | ")
		b.WriteString(cell(a.Absence.Gap))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Migration checklist (mock)\n\n")
	b.WriteString("Every step below is one verified claim, in the order the work is\n")
	b.WriteString("performed. A step whose claim failed to verify is listed as unverified\n")
	b.WriteString("rather than removed.\n\n")
	for _, s := range BuildChecklist(r) {
		b.WriteString(strconv.Itoa(s.Sequence))
		b.WriteString(". ")
		b.WriteString(s.Text)
		if s.Verified {
			b.WriteString(" (")
			b.WriteString(s.Evidence)
			b.WriteString(")\n")
			continue
		}
		b.WriteString(" (UNVERIFIED: ")
		b.WriteString(s.Evidence)
		b.WriteString(")\n")
	}

	b.WriteString("\n## Import vocabulary measured from the validator\n\n")
	b.WriteString("| Column | Accepted headers |\n| --- | --- |\n")
	for _, f := range r.Surface.Columns {
		b.WriteString("| ")
		b.WriteString(cell(f.Column))
		b.WriteString(" | ")
		b.WriteString(cell(strings.Join(f.Aliases, ", ")))
		b.WriteString(" |\n")
	}
	b.WriteString("\nFormats accepted: ")
	b.WriteString(strings.Join(r.Surface.Formats, ", "))
	b.WriteString("\nSpreadsheet sheet name required: ")
	b.WriteString(quote(r.Surface.Sheet))
	b.WriteString(".\n\n")
	b.WriteString("Row rejection reasons, in source order:\n\n")
	for _, reason := range r.Surface.Rejections {
		b.WriteString("- ")
		b.WriteString(reason)
		b.WriteString("\n")
	}
	b.WriteString("\nFields the profile supplies when the row leaves them empty: ")
	b.WriteString(list(r.Surface.ConditionalFields))
	b.WriteString(".\n")
	b.WriteString("Fields always copied from the profile: ")
	b.WriteString(list(r.Surface.ProfileFields))
	b.WriteString(".\n")
	b.WriteString("Values filled in when the row omits a column: ")
	b.WriteString(list(r.Surface.Defaults))
	b.WriteString(".\n")

	b.WriteString("\n## Effort items (sizes derived from measured counts)\n\n")
	b.WriteString("Sizes are t-shirt labels. No clock or currency figure is asserted;\n")
	b.WriteString("the master inventory keeps those baselines at UNKNOWN until a scoped\n")
	b.WriteString("engagement measures them.\n\n")
	b.WriteString("| Item | Size | Measured driver |\n| --- | --- | --- |\n")
	for _, w := range WorkItems(r) {
		b.WriteString("| ")
		b.WriteString(cell(w.Title))
		b.WriteString(" | ")
		b.WriteString(cell(w.Size(r.Verdict)))
		b.WriteString(" | ")
		b.WriteString(cell(w.Driver))
		b.WriteString(" |\n")
	}

	b.WriteString("\n## Dry-run and preview posture\n\n")
	b.WriteString("Entity types the product can import from an uploaded file:\n\n")
	if len(r.ImportSurfaces) == 0 {
		b.WriteString("- none found\n")
	}
	for _, s := range r.ImportSurfaces {
		b.WriteString("- ")
		b.WriteString(s)
		b.WriteString("\n")
	}
	b.WriteString("\nNon-mutating preflight already implemented elsewhere:\n\n")
	if len(r.Precedents) == 0 {
		b.WriteString("- none found\n")
	}
	for _, p := range r.Precedents {
		b.WriteString("- ")
		b.WriteString(p)
		b.WriteString("\n")
	}
	b.WriteString("\nTests that pin the import contract:\n\n")
	for _, t := range r.ImportTests {
		b.WriteString("- ")
		b.WriteString(t)
		b.WriteString("\n")
	}
	return b.String()
}

// quote renders a measured string literal, or the word none when the extraction
// found nothing to report.
func quote(s string) string {
	if s == "" {
		return "none"
	}
	return "`" + s + "`"
}

// list joins a measured set for one prose line.
func list(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

// cell makes text safe for a Markdown table cell.
func cell(s string) string {
	return strings.ReplaceAll(s, "|", "\\|")
}
