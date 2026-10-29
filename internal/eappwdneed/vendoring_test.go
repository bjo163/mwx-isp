// Tests for the docs/rfcs vendoring audit.

package eappwdneed

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCarriedArchiveHasNoNumberMismatch(t *testing.T) {
	v, err := AuditVendoring(repoRoot(t))
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	if len(v.Docs) < 40 {
		t.Fatalf("only %d specification files read; the archive is at docs/rfcs", len(v.Docs))
	}
	for _, d := range v.NumberMismatch {
		t.Errorf("%s is filed under a number it does not declare", d.File)
	}
}

// TestEAPWDSpecIsCarriedUnderTheNameThatPromisesIt pins M12.1. Until 2026-10-29
// the archive had no EAP-PWD text at all: the file named for it was RFC 7542, the
// Network Access Identifier. RFC 5931 is now carried, and nothing else in docs/rfcs
// claims the EAP-PWD name, so a milestone citing "the EAP-PWD spec" cannot silently
// read the wrong document.
func TestEAPWDSpecIsCarriedUnderTheNameThatPromisesIt(t *testing.T) {
	v, err := AuditVendoring(repoRoot(t))
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	got := findSpec(v, "eap-pwd-spec")
	if got == nil {
		t.Fatal("the EAP-PWD citation is not audited")
	}
	if got.State != SpecHeld {
		t.Fatalf("state = %s, want carried; %s", got.State, got.String())
	}
	if filepath.Base(got.HeldBy) != "rfc5931-eap-pwd.txt" {
		t.Errorf("held by %s, want the RFC 5931 text", got.HeldBy)
	}
	if got.HeldBy != got.PromisedBy {
		t.Errorf("held=%s promised=%s; the milestone must be able to cite one file", got.HeldBy, got.PromisedBy)
	}

	var promises []string
	for _, d := range v.Docs {
		if strings.Contains(filepath.Base(d.File), "-"+got.Spec.Slug+".txt") {
			promises = append(promises, d.File)
		}
	}
	if len(promises) != 1 {
		t.Errorf("%d files are named for EAP-PWD (%s), want exactly one", len(promises), strings.Join(promises, ", "))
	}

	// Carrying the right number is not enough: the text has to be the method M12
	// would build, so the phrases an implementer needs are checked in the file.
	// The type code is one of them, because the roadmap schedules `TypePWD = 52`
	// and that number has to be readable here rather than recalled.
	//nolint:gosec // G304: a fixed path inside this repository, resolved under the repo root.
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), filepath.FromSlash(got.HeldBy)))
	if err != nil {
		t.Fatalf("reading the carried text: %v", err)
	}
	for _, want := range []string{"EAP-pwd", "shared password", "Ciphersuite", "EAP method type for EAP-pwd (52)"} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("%s is absent from %s, which is supposed to specify the method", want, got.HeldBy)
		}
	}
}

func TestTEAPSpecIsCarriedAsThePositiveControl(t *testing.T) {
	v, err := AuditVendoring(repoRoot(t))
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	got := findSpec(v, "teap-spec-control")
	if got == nil {
		t.Fatal("the TEAP control citation is not audited")
	}
	if got.State != SpecHeld {
		t.Fatalf("state = %s, want carried; the audit cannot tell a mislabeled file from a missing one unless one of each is checked", got.State)
	}
	if !strings.Contains(got.HeldTitle, "Tunnel Extensible Authentication Protocol") {
		t.Errorf("RFC 7170 read as %q", got.HeldTitle)
	}
	if got.HeldBy != got.PromisedBy {
		t.Errorf("the control should be carried under the name that promises it: held=%s promised=%s", got.HeldBy, got.PromisedBy)
	}
}

func TestEverySpecNumberIsCitedByAProjectDocument(t *testing.T) {
	root := repoRoot(t)
	for _, s := range Specs() {
		//nolint:gosec // G304: a fixed path inside this repository, resolved under the repo root.
		cited, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(s.CitedIn)))
		if err != nil {
			t.Fatalf("reading the citing document: %v", err)
		}
		if !strings.Contains(string(cited), "RFC "+s.Number) {
			t.Errorf("spec %s claims RFC %s is cited, but %s never prints that number", s.ID, s.Number, s.CitedIn)
		}
	}
}

func TestAuditReportsAbsentWhenNoFilePromisesTheTopic(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/rfcs/rfc3748-eap.txt": "Network Working Group\nRequest for Comments: 3748\nCategory: Standards Track\n\n\n                     Extensible Authentication Protocol (EAP)\n\nAbstract\n",
	})
	v, err := AuditVendoring(root)
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	got := findSpec(v, "eap-pwd-spec")
	if got == nil {
		t.Fatal("no verdict for the EAP-PWD citation")
	}
	if got.State != SpecAbsent {
		t.Errorf("state = %s, want absent; nothing in this tree is named for EAP-PWD", got.State)
	}
	if got.PromisedBy != "" {
		t.Errorf("an absent spec should name no file, got %s", got.PromisedBy)
	}
	if !strings.Contains(got.String(), "absent") {
		t.Errorf("String() lost the verdict: %q", got.String())
	}
}

// TestAuditReportsMislabeledUnderAFixture keeps the mislabeled verdict reachable
// now that the real archive no longer triggers it: a file named for the topic that
// declares a different number must still be reported as mislabeled rather than
// absent, because the difference is what tells an implementer to rename a file
// instead of downloading a specification.
func TestAuditReportsMislabeledUnderAFixture(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/rfcs/rfc7542-eap-pwd.txt": "Request for Comments: 7542\nCategory: Standards Track\n\n\n                     The Network Access Identifier\n\nAbstract\n",
	})
	v, err := AuditVendoring(root)
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	got := findSpec(v, "eap-pwd-spec")
	if got == nil {
		t.Fatal("no verdict for the EAP-PWD citation")
	}
	if got.State != SpecMislabeled {
		t.Fatalf("state = %s, want mislabeled; %s", got.State, got.String())
	}
	if got.PromisedNumber != "7542" || !strings.Contains(got.PromisedTitle, "Network Access Identifier") {
		t.Errorf("the verdict lost what the file really is: number=%s title=%q", got.PromisedNumber, got.PromisedTitle)
	}
}

func TestAuditReportsNumberMismatch(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		"docs/rfcs/rfc1234-eap-demo.txt": "Network Working Group\nRequest for Comments: 9999\nCategory: Standards Track\n\n\n                     Demo Protocol\n\nAbstract\n",
	})
	v, err := AuditVendoring(root)
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	if len(v.NumberMismatch) != 1 {
		t.Fatalf("mismatches = %d, want 1", len(v.NumberMismatch))
	}
	if v.NumberMismatch[0].FilenameNumber != "1234" || v.NumberMismatch[0].HeaderNumber != "9999" {
		t.Errorf("mismatch reported the wrong pair: %+v", v.NumberMismatch[0])
	}
	// One misnumbered file plus both cited specifications, which this synthetic
	// archive carries neither of.
	if v.Findings() != 1+len(Specs()) {
		t.Errorf("findings = %d, want the mismatch plus every citation verdict, since none of them is carried here", v.Findings())
	}
}

func TestAuditFailsWhenTheArchiveIsMissing(t *testing.T) {
	if _, err := AuditVendoring(t.TempDir()); err == nil {
		t.Fatal("the audit accepted a tree with no docs/rfcs; an empty archive would make every citation look absent")
	}
}

func TestTitleOfReadsTheHeaderBlock(t *testing.T) {
	text := "Network Working Group                                   A. Author\n" +
		"Request for Comments: 5931                        Example University\n" +
		"Category: Standards Track\n" +
		"\n" +
		"\n" +
		"          Extensible Authentication Protocol (EAP)\n" +
		"          Password Authenticated Connection Establishment\n" +
		"\n" +
		"Abstract\n" +
		"\n" +
		"   This document describes a protocol.\n"
	if got := titleOf(text); !strings.Contains(got, "Password Authenticated Connection Establishment") || !strings.Contains(got, "Extensible") {
		t.Errorf("titleOf() = %q, want both title lines", got)
	}
	if got := headerNumber(text); got != "5931" {
		t.Errorf("headerNumber() = %q, want 5931", got)
	}
	if got := headerNumber("no header here\nAbstract\n"); got != "" {
		t.Errorf("headerNumber() on a file with no header = %q, want empty", got)
	}
}

func TestVerifyCarriesTheVendoringAudit(t *testing.T) {
	report, err := Verify(repoRoot(t))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if report.Totals.DocsCarried != len(report.Vendoring.Docs) {
		t.Errorf("DocsCarried = %d but %d documents were read", report.Totals.DocsCarried, len(report.Vendoring.Docs))
	}
	if report.Totals.RFCFindings == 0 {
		t.Error("the real tree has a cited specification that is not carried; the total should not be zero")
	}
}

// findSpec returns one specification verdict by ID.
func findSpec(v Vendoring, id string) *SpecStatus {
	for i := range v.Specs {
		if v.Specs[i].Spec.ID == id {
			return &v.Specs[i]
		}
	}
	return nil
}

func TestEAPMD5SpecIsCarriedUnderADifferentName(t *testing.T) {
	v, err := AuditVendoring(repoRoot(t))
	if err != nil {
		t.Fatalf("AuditVendoring: %v", err)
	}
	got := findSpec(v, "eap-md5-spec")
	if got == nil {
		t.Fatal("the EAP-MD5 citation is not audited")
	}
	// The archive carries RFC 3748, so this is not a missing document; the file
	// named for EAP-MD5 is RFC 3851, which is S/MIME.
	if got.State != SpecCarriedElsewhere {
		t.Fatalf("state = %s, want carried-elsewhere (got %s)", got.State, got.String())
	}
	if filepath.Base(got.HeldBy) != "rfc3748-eap.txt" {
		t.Errorf("held by %s, want the EAP core document", got.HeldBy)
	}
	if filepath.Base(got.PromisedBy) != "rfc3851-eap-md5.txt" || got.PromisedNumber != "3851" {
		t.Errorf("promised file = %s declaring %s, want rfc3851-eap-md5.txt declaring 3851", got.PromisedBy, got.PromisedNumber)
	}
	if !strings.Contains(got.PromisedTitle, "S/MIME") {
		t.Errorf("the file named for EAP-MD5 read as %q", got.PromisedTitle)
	}
}
