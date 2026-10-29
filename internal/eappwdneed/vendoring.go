// Vendoring audit for the EAP-PWD argument.
//
// A roadmap line cites a specification by number. This file checks that the
// cited number is actually carried under docs/rfcs, that each carried file's
// own header agrees with the number in its name, and that the file named for a
// topic really is that topic. It reads the header the RFC itself prints, so a
// renamed or mis-numbered document is reported as what it says about itself
// rather than what the project filed it as.

package eappwdneed

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// rfcsDir is where this project keeps the protocol text it cites.
const rfcsDir = "docs/rfcs"

// Doc is one specification carried under docs/rfcs, as read from the file.
type Doc struct {
	// File is the repository-relative path.
	File string
	// FilenameNumber is the four-digit number the filename claims.
	FilenameNumber string
	// HeaderNumber is the number the document prints in its own
	// "Request for Comments" header, or empty when it prints none.
	HeaderNumber string
	// Title is the document title taken from the same header block.
	Title string
}

// String renders one carried document.
func (d Doc) String() string {
	header := "RFC " + d.HeaderNumber
	if d.HeaderNumber == "" {
		header = "no header number"
	}
	return fmt.Sprintf("%s declares %s title=%q", d.File, header, d.Title)
}

// Spec is a standard a project document cites for the M12 argument.
type Spec struct {
	// ID names the spec in reports.
	ID string
	// Number is the RFC number as the citing document writes it.
	Number string
	// Topic is the phrase that must appear in the specification text itself.
	Topic string
	// Slug is the filename fragment the project uses for this topic.
	Slug string
	// CitedIn is the repository document that makes the citation.
	CitedIn string
	// CitedBy quotes enough of the citation to find it again.
	CitedBy string
	// Why says what the milestone needs the document for.
	Why string
}

// SpecState is one audit verdict about a cited specification.
type SpecState string

// The four states a cited specification can be in.
const (
	// SpecHeld means the file named for the topic is the cited document.
	SpecHeld SpecState = "carried"
	// SpecCarriedElsewhere means the cited text is in the archive, but under a
	// different filename than the topic promises.
	SpecCarriedElsewhere SpecState = "carried-elsewhere"
	// SpecAbsent means no carried file declares this number.
	SpecAbsent SpecState = "absent"
	// SpecMislabeled means the file named for this topic is a different
	// document and the cited number is carried nowhere.
	SpecMislabeled SpecState = "mislabeled"
)

// SpecStatus is one specification audit finding.
type SpecStatus struct {
	// Spec is the cited standard.
	Spec Spec
	// State is the verdict.
	State SpecState
	// HeldBy is the file whose own header declares the cited number.
	HeldBy string
	// HeldTitle is that file's title.
	HeldTitle string
	// PromisedBy is the file the archive names for this topic.
	PromisedBy string
	// PromisedNumber is the number that promised file declares about itself.
	PromisedNumber string
	// PromisedTitle is what that promised file actually is.
	PromisedTitle string
	// Note explains what the verdict means for the milestone.
	Note string
}

// String renders one specification verdict.
func (s SpecStatus) String() string {
	switch s.State {
	case SpecHeld:
		return fmt.Sprintf("%s %s %s", s.Spec.ID, s.State, s.HeldBy)
	case SpecCarriedElsewhere:
		return fmt.Sprintf("%s %s cited text in %s, but %s is %q", s.Spec.ID, s.State, s.HeldBy, s.PromisedBy, s.PromisedTitle)
	case SpecMislabeled:
		return fmt.Sprintf("%s %s %s declares RFC %s %q", s.Spec.ID, s.State, s.PromisedBy, s.PromisedNumber, s.PromisedTitle)
	default:
		return fmt.Sprintf("%s %s (no file named *%s* under %s)", s.Spec.ID, s.State, s.Spec.Slug, rfcsDir)
	}
}

// Specs returns the specifications the M12 argument cites, plus one carried
// correctly as a control. Every Number here is copied from a project document,
// not from the memory of whoever wrote the audit.
func Specs() []Spec {
	return []Spec{
		{
			ID:      "eap-pwd-spec",
			Number:  "5931",
			Topic:   "EAP-PWD",
			Slug:    "eap-pwd",
			CitedIn: "docs/feature-checklist.md",
			CitedBy: "docs/feature-checklist.md TR-F004 row: \"Per RFC 5931, authenticate with a shared password\"",
			Why:     "M12 is specified as implementing EAP-PWD, so the method's message format and key derivation have to be readable here.",
		},
		{
			ID:      "eap-md5-spec",
			Number:  "3748",
			Topic:   "EAP-MD5",
			Slug:    "eap-md5",
			CitedIn: "docs-site/src/en/rfc-index.md",
			CitedBy: "docs-site/src/en/rfc-index.md catalog accuracy note: \"EAP-MD5 is defined by RFC 3748 §5.4, not RFC 3851\"",
			Why:     "A second citation the archive resolves differently from its filename, kept here so the audit distinguishes a missing document from a misnamed one.",
		},
		{
			ID:      "teap-spec-control",
			Number:  "7170",
			Topic:   "TEAP",
			Slug:    "teap",
			CitedIn: "docs/feature-checklist.md",
			CitedBy: "docs/feature-checklist.md TR-F004 TEAP row: \"Per RFC 7170 / RFC 9930 (TEAPv1)\"",
			Why:     "A correctly filed specification, included so the audit shows the difference between a mislabeled file and a missing one.",
		},
	}
}

// Vendoring is one audit of docs/rfcs.
type Vendoring struct {
	// Docs lists every carried file, in path order.
	Docs []Doc
	// NumberMismatch lists files whose name and own header disagree.
	NumberMismatch []Doc
	// Specs lists the citation verdicts, in the order Specs returns.
	Specs []SpecStatus
}

// Findings counts everything the audit had to report about the archive.
func (v Vendoring) Findings() int {
	n := len(v.NumberMismatch)
	for _, s := range v.Specs {
		if s.State != SpecHeld {
			n++
		}
	}
	return n
}

// headerNumberRE matches the "Request for Comments: NNNN" line an RFC prints.
var headerNumberRE = regexp.MustCompile(`Request for Comments:\s*(\d+)`)

// filenameNumberRE matches the number a carried filename claims.
var filenameNumberRE = regexp.MustCompile(`^rfc(\d{4})-([a-z0-9.-]+)\.txt$`)

// AuditVendoring reads docs/rfcs and checks the cited specifications against it.
// It fails only when the directory itself cannot be read: an unreadable archive
// would make every citation look absent, which is the wrong answer to report.
func AuditVendoring(root string) (Vendoring, error) {
	base := filepath.Join(root, filepath.FromSlash(rfcsDir))
	entries, err := os.ReadDir(base)
	if err != nil {
		return Vendoring{}, fmt.Errorf("eappwdneed: reading %s: %w", rfcsDir, err)
	}
	var docs []Doc
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
			continue
		}
		m := filenameNumberRE.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(rfcsDir, e.Name()))
		//nolint:gosec // G304: filenames matched by the rfcNNNN-slug.txt pattern above.
		raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			return Vendoring{}, fmt.Errorf("eappwdneed: reading %s: %w", rel, err)
		}
		text := string(raw)
		docs = append(docs, Doc{
			File:           rel,
			FilenameNumber: m[1],
			HeaderNumber:   headerNumber(text),
			Title:          titleOf(text),
		})
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].File < docs[j].File })

	v := Vendoring{Docs: docs}
	for _, d := range docs {
		if d.HeaderNumber != d.FilenameNumber {
			v.NumberMismatch = append(v.NumberMismatch, d)
		}
	}

	for _, spec := range Specs() {
		status := SpecStatus{Spec: spec, Note: spec.Why}
		var promised []Doc
		for _, d := range docs {
			if strings.Contains(filepath.Base(d.File), "-"+spec.Slug+".txt") {
				promised = append(promised, d)
			}
			if d.HeaderNumber == spec.Number {
				status.HeldBy = d.File
				status.HeldTitle = d.Title
			}
		}
		if len(promised) > 0 {
			status.PromisedBy = promised[0].File
			status.PromisedNumber = promised[0].HeaderNumber
			status.PromisedTitle = promised[0].Title
		}
		switch {
		case status.HeldBy != "" && status.HeldBy == status.PromisedBy:
			status.State = SpecHeld
		case status.HeldBy != "":
			status.State = SpecCarriedElsewhere
		case status.PromisedBy != "":
			status.State = SpecMislabeled
		default:
			status.State = SpecAbsent
		}
		v.Specs = append(v.Specs, status)
	}
	return v, nil
}

// headerNumber returns the number an RFC prints about itself, or "" when the
// file carries no such line.
func headerNumber(text string) string {
	m := headerNumberRE.FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return m[1]
}

// titleOf returns the title an RFC prints in its header block. The title is the
// last unbroken run of lines that is not an author, number or category line; the
// IETF text format closes the title with a blank line before Abstract, so the run
// before that blank is the one being looked for.
func titleOf(text string) string {
	var run, last []string
	closeRun := func() {
		if len(run) > 0 {
			last = run
			run = nil
		}
	}
	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if isSectionStop(trimmed) {
			break
		}
		switch {
		case trimmed == "", isHeaderField(trimmed):
			closeRun()
		default:
			run = append(run, trimmed)
		}
	}
	closeRun()
	return strings.Join(last, " ")
}

// isSectionStop reports whether a line begins the block that follows the title:
// the status note, the copyright notice, the abstract or the table of contents.
// Pre-2008 RFCs put those between the title and the abstract, so the scan has to
// stop at the first of them rather than at Abstract alone.
func isSectionStop(trimmed string) bool {
	lower := strings.ToLower(trimmed)
	for _, stop := range []string{"abstract", "table of contents", "status of this", "copyright"} {
		if strings.HasPrefix(lower, stop) {
			return true
		}
	}
	return false
}

// isHeaderField reports whether a line belongs to the RFC header block rather
// than to the document title.
func isHeaderField(trimmed string) bool {
	for _, label := range []string{
		"Request for Comments", "Obsoletes", "Updates", "Category", "ISSN",
		"Internet Engineering", "Streaming Alliance", "Internet-Draft",
		"Expires", "Published", "Network Working Group",
	} {
		if strings.Contains(trimmed, label) {
			return true
		}
	}
	return false
}
