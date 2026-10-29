// Package teapreadiness reports which mechanisms RFC 7170 (TEAP) expects of an
// authentication server, and which of them this repository's EAP server already
// implements.
//
// It is an ENG-006 research prototype. It only reads, no product code imports
// it, and every verdict is anchored twice: a mechanism counts as implemented
// only when a Go identifier in the product code proves it, and it counts as
// required only when a clause quoted from a specification this repository
// carries demands it. Which supplicants actually ship TEAP, and whether they
// interoperate, is never claimed here — no file in the tree can answer that, so
// the report lists it as unknown instead of guessing.
package teapreadiness

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// The RFC labels cited by the catalog.
const (
	// rfcTEAP is the base TEAP specification, vendored as docs/rfcs/rfc7170-teap.txt.
	rfcTEAP = "RFC 7170"
	// rfcTLSTunnel is the TLS 1.3 transport TEAP builds on.
	rfcTLSTunnel = "RFC 9190"
	// rfcEAPBase is the EAP framework carrying fragmentation rules.
	rfcEAPBase = "RFC 3748"
	// rfcCUI is the chargeable-user-identity specification behind the anonymous
	// outer identity the server already sends.
	rfcCUI = "RFC 4372"
	// rfcTEAPv1 is the revision that adds the AVP and password mechanisms; this
	// repository carries no text for it, so its clauses cannot be quoted locally.
	rfcTEAPv1 = "RFC 9930"
	// rfcCryptoAgility is the TEAP key-derivation update; likewise not vendored.
	rfcCryptoAgility = "RFC 9427"
)

// Status is one mechanism's readiness in the tree that was scanned.
type Status string

// Statuses a mechanism can reach. They are reported verbatim, so the names read
// as the claim they back: ready means an implementer can reuse what is there,
// missing means nothing does it, blocked means the code actively pins the
// opposite, and partial means only some of the markers were found.
const (
	StatusReady   Status = "ready"
	StatusPartial Status = "partial"
	StatusBlocked Status = "blocked"
	StatusMissing Status = "missing"
)

// Source is the specification clause that makes a Mechanic a requirement.
type Source struct {
	// RFC names the specification, e.g. "RFC 7170".
	RFC string
	// Section is the clause number, or "" when the requirement is quoted from
	// the document body rather than a numbered heading.
	Section string
	// Document is the vendored text under docs/rfcs, or "" when this repository
	// carries no text for RFC — in which case Quote must be "" too.
	Document string
	// Quote is a phrase that appears verbatim in Document once whitespace is
	// collapsed. It is what makes the requirement citable instead of asserted.
	Quote string
}

// String renders the citation the way the report and notes use it.
func (s Source) String() string {
	if s.Section != "" {
		return fmt.Sprintf("%s §%s", s.RFC, s.Section)
	}
	return s.RFC
}

// Mechanic is one thing a TEAP server must be able to do.
type Mechanic struct {
	// ID names the mechanism in reports, e.g. "crypto-binding-tlv".
	ID string
	// Title states the requirement in one line.
	Title string
	// Source is the clause that demands it.
	Source Source
	// Markers are Go identifiers whose presence in the product code proves the
	// mechanism exists. A marker found only in a comment or a test file is not
	// evidence, because neither changes what the server does on the wire.
	Markers []string
	// Against is code text that contradicts the mechanism. Any match blocks the
	// mechanism even when its markers are present.
	Against []string
	// Note tells an implementer what the verdict means for the work.
	Note string
}

// Evidence is one line of product code that a verdict points at.
type Evidence struct {
	// File is the repository-relative path.
	File string
	// Line is the one-based line number within the comment-stripped file.
	Line int
	// Text is the trimmed source line.
	Text string
}

// String renders evidence the way the notes and tables use it.
func (e Evidence) String() string {
	return fmt.Sprintf("%s:%d %s", e.File, e.Line, e.Text)
}

// Finding is one Mechanic's verdict for one scanned tree.
type Finding struct {
	// Mechanic is the requirement this finding answers.
	Mechanic Mechanic
	// Status is the verdict.
	Status Status
	// Support is the product code that proves the mechanism.
	Support []Evidence
	// Against is the product code that contradicts it.
	Against []Evidence
	// MarkerHits counts the markers found, out of len(Mechanic.Markers).
	MarkerHits int
	// Note carries Mechanic.Note plus any qualification a citation needs.
	Note string
}

// String renders one finding as a single line.
func (f Finding) String() string {
	verdict := fmt.Sprintf("%s %s [%s] markers=%d/%d", f.Mechanic.ID, f.Status,
		f.Mechanic.Source, f.MarkerHits, len(f.Mechanic.Markers))
	switch {
	case len(f.Against) > 0:
		return verdict + " contradicted-by=" + f.Against[0].String()
	case len(f.Support) > 0:
		return verdict + " evidence=" + f.Support[0].String()
	default:
		return verdict + " evidence=none"
	}
}

// Unknown is a question about TEAP compatibility that the repository cannot
// answer, named so that a readiness report is not read as a compatibility claim.
type Unknown struct {
	// Topic is the open question.
	Topic string
	// Reason says why no file in the tree can settle it.
	Reason string
}

// Totals summarizes a report.
type Totals struct {
	// Mechanics counts the requirements examined.
	Mechanics int
	// Ready counts mechanisms the tree already implements.
	Ready int
	// Partial counts mechanisms with some markers present.
	Partial int
	// Blocked counts mechanisms contradicted by code that pins the opposite.
	Blocked int
	// Missing counts mechanisms nothing implements.
	Missing int
}

// Report is one readiness run.
type Report struct {
	// Findings is ordered by mechanism ID, so the same tree always renders the
	// same document.
	Findings []Finding
	// Totals summarizes Findings.
	Totals Totals
	// Unknown lists what the scan cannot settle.
	Unknown []Unknown
}

// maxEvidence caps how many matching lines one verdict carries, so a marker that
// appears across the whole tree does not turn the report into a source dump.
const maxEvidence = 3

// Mechanics returns the requirement catalog in the order RFC 7170 introduces it.
// Every marker names an identifier an implementer would have to write, and every
// citation is quoted from a document under docs/rfcs — except the two clauses
// whose specifications this repository does not carry, which say so in Note.
func Mechanics() []Mechanic {
	return []Mechanic{
		{
			ID:      "teap-eap-type",
			Title:   "TEAP has an assigned EAP method type the server can send",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc7170-teap.txt", Quote: "TEAP Message Format"},
			Markers: []string{"TypeTEAP"},
			Note:    "interfaces.go declares types 1-33 and stops at EAP-TLV; type 39 is the first one missing.",
		},
		{
			ID:      "teap-method-name",
			Title:   "The method has a name the registry and configuration can address",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc7170-teap.txt", Quote: "TEAP Message Format"},
			Markers: []string{"EAPMethodTEAP"},
			Note:    "Handlers export EAPMethodTLS, EAPMethodPEAP, EAPMethodTTLS and EAPMethodOTP; there is no TEAP name.",
		},
		{
			ID:      "teap-handler-wiring",
			Title:   "A TEAP handler is registered and reachable from the identity phase",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc7170-teap.txt", Quote: "In the first EAP-Request sent with EAP type=TEAP"},
			Markers: []string{"NewTEAPHandler", "NewTEAPHandlerWithConfig"},
			Note:    "Registration is central in plugins/init.go, and the identity-phase dispatch in coordinator.go is a name switch, so a new method needs both a constructor and a case there.",
		},
		{
			ID:      "teap-config-choice",
			Title:   "Operators can select TEAP as radius.EapMethod",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc7170-teap.txt", Quote: "TEAP Message Format"},
			Markers: []string{"eap-teap"},
			Note:    "The config enum lists eap-md5, eap-mschapv2, eap-tls, eap-peap and eap-ttls; the allowlist would drop an unregistered name anyway.",
		},
		{
			ID:      "teap-version-negotiation",
			Title:   "The TEAP version field is negotiated and re-verified",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.13", Document: "rfc7170-teap.txt", Quote: "The receiver of the Crypto-Binding TLV MUST verify that the version"},
			Markers: []string{"teapVersion", "TEAPVersion"},
			Note:    "Nothing parses a TEAP header today, so there is no version octet to negotiate.",
		},
		{
			ID:      "result-tlv",
			Title:   "Result TLV is read to finish phase 2",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.4", Document: "rfc7170-teap.txt", Quote: "Result TLV"},
			Markers: []string{"TypeTLV", "tlvTypeResult"},
			Note:    "PEAPv0 already decodes a Result TLV of type 3, which is the same encoding TEAP requires.",
		},
		{
			ID:      "crypto-binding-tlv",
			Title:   "Crypto-Binding TLV is constructed and verified",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.13", Document: "rfc7170-teap.txt", Quote: "Phase 2 MUST always end with a Crypto-Binding TLV"},
			Markers: []string{"CryptoBinding", "cryptobinding"},
			Note:    "The only mentions in the tree are comments: peap_inner.go says unknown TLVs including Cryptobinding are skipped, so the tunneled methods today cannot close a TEAP exchange.",
		},
		{
			ID:      "eap-payload-tlv",
			Title:   "Inner EAP runs are carried in EAP-Payload TLVs",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.10", Document: "rfc7170-teap.txt", Quote: "EAP-Payload TLV"},
			Markers: []string{"eapPayloadTLV", "EAPPayload"},
			Note:    "PEAP and TTLS put a bare inner EAP packet in the tunnel; TEAP wraps it in a TLV, which is a new encoder rather than a reuse.",
		},
		{
			ID:      "intermediate-result-tlv",
			Title:   "Intermediate-Result TLV reports progress before the final result",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.11", Document: "rfc7170-teap.txt", Quote: "Intermediate-Result TLV"},
			Markers: []string{"IntermediateResult", "CodeIntermediate"},
			Note:    "The code knows EAP Request, Response, Success and Failure only; the chaining design needs the intermediate form to continue after the first inner method.",
		},
		{
			ID:      "pac-tlv",
			Title:   "PAC TLV provisioning and reuse is implemented",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.12", Document: "rfc7170-teap.txt", Quote: "PAC TLV Format"},
			Markers: []string{"pacOpaque", "PACInfo", "pacTLV"},
			Note:    "Session resumption inside TEAP depends on PAC-Key, PAC-Opaque and PAC-Info; none of them exists, so every TEAP handshake would be a full handshake.",
		},
		{
			ID:      "inner-method-chaining",
			Title:   "Two inner methods chain, machine then user",
			Source:  Source{RFC: rfcTEAP, Document: "rfc7170-teap.txt", Quote: "TEAPv1 meets this requirement by supporting inner EAP method chaining"},
			Markers: []string{"InnerMethods", "methodChain"},
			Note:    "The tunnel state machines run exactly one inner method per session, which is the M11 requirement the roadmap calls certificate plus password.",
		},
		{
			ID:      "identity-type-tlv",
			Title:   "Identity-Type TLV distinguishes machine from user identity",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.3", Document: "rfc7170-teap.txt", Quote: "Identity-Type TLV"},
			Markers: []string{"identityType", "IdentityType"},
			Note:    "Chaining is meaningless without the TLV that says whose credentials are being proved.",
		},
		{
			ID:      "authority-id-tlv",
			Title:   "Authority-ID TLV identifies the provisioning authority",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.2", Document: "rfc7170-teap.txt", Quote: "Authority-ID TLV"},
			Markers: []string{"authorityID", "AuthorityID"},
			Note:    "Only needed if PAC provisioning is built; it is listed so the build set is complete rather than implied.",
		},
		{
			ID:      "basic-password-auth-tlv",
			Title:   "Basic-Password-Auth TLVs carry a password without an inner EAP method",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.14", Document: "rfc7170-teap.txt", Quote: "Basic-Password-Auth-Req TLV"},
			Markers: []string{"basicPasswordAuth", "PasswordAuthTLV"},
			Note:    "This is the certificate-plus-password path the roadmap wants, and it is a TLV exchange, not an inner method.",
		},
		{
			ID:      "tls13-outer-tunnel",
			Title:   "The outer tunnel can negotiate TLS 1.3",
			Source:  Source{RFC: rfcTLSTunnel, Document: "rfc9190-eap-tls13.txt", Quote: "use of EAP-TLS with TLS 1.3"},
			Markers: []string{"VersionTLS13"},
			Against: []string{"MaxVersion: tls.VersionTLS12"},
			Note:    "EAP-TLS can reach TLS 1.3 through the configured minimum, but the tunneled providers cap the session at TLS 1.2, so the TEAP tunnel needs a new provider rather than the TTLS one.",
		},
		{
			ID:      "rfc9427-key-derivation",
			Title:   "Session keys derive through the RFC 9427 KDF",
			Source:  Source{RFC: rfcCryptoAgility},
			Markers: []string{"cfs4", "KdfLabel"},
			Note:    "docs/rfcs carries no RFC 9427 text, so this requirement is named but cannot be quoted from a document in this repository; the existing exporter call sites use the RFC 5705 labels of the inner methods.",
		},
		{
			ID:      "exported-keying-material",
			Title:   "The tunnel exports keying material for the inner method",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.13", Document: "rfc7170-teap.txt", Quote: "Crypto-Binding TLV"},
			Markers: []string{"ExportKeyingMaterial"},
			Note:    "RFC 5705 export already happens inside the established tunnel, which is where a Crypto-Binding calculation would hook in.",
		},
		{
			ID:      "tunneled-eap-framing",
			Title:   "Inner EAP packets are parsed and normalized inside the tunnel",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.10", Document: "rfc7170-teap.txt", Quote: "EAP-Payload TLV"},
			Markers: []string{"parseInnerEAP", "normalizeInnerEAP"},
			Note:    "PEAP's inner framing already does the header normalization a TEAP EAP-Payload TLV would reuse.",
		},
		{
			ID:      "inner-avp-codec",
			Title:   "Vendor AVPs are encoded and decoded inside the tunnel",
			Source:  Source{RFC: rfcTEAPv1},
			Markers: []string{"parseTTLSAVPs", "encodeTTLSAVP"},
			Note:    "The AVP codec is EAP-TTLS's and is reusable, but docs/rfcs carries no RFC 9930 text, so its use inside TEAP is named here without a local citation.",
		},
		{
			ID:      "tls-fragmentation",
			Title:   "Oversized tunnel records are fragmented and reassembled",
			Source:  Source{RFC: rfcEAPBase, Document: "rfc3748-eap.txt", Quote: "a fragmented EAP packet will require"},
			Markers: []string{"tlsfragment.Reassembler", "tlsfragment.Packet"},
			Note:    "Fragmentation is shared by every tunneled method and needs no TEAP-specific work.",
		},
		{
			ID:      "protected-success",
			Title:   "Success is confirmed inside the tunnel",
			Source:  Source{RFC: rfcTLSTunnel, Document: "rfc9190-eap-tls13.txt", Quote: "EAP-TLS with TLS 1.3"},
			Markers: []string{"protectedSuccess"},
			Note:    "EAP-TLS already sends the TLS 1.3 protected-success octet; TEAP's equivalent is the Crypto-Binding plus Result exchange, so this reuse is partial by design.",
		},
		{
			ID:      "anonymous-outer-identity",
			Title:   "The outer identity can be anonymous",
			Source:  Source{RFC: rfcCUI, Document: "rfc4372-chargeable-user-identity.txt", Quote: "contains an anonymous identity"},
			Markers: []string{"IsAnonymousNAI"},
			Note:    "Already required and implemented for EAP-TLS, so a TEAP deployment inherits credential hygiene for the outer exchange.",
		},
	}
}

// unknowns names what no file in this repository can answer about TEAP
// compatibility, so a report that finds several ready mechanisms is not read as
// a statement that TEAP would work with real clients.
var unknowns = []Unknown{
	{
		Topic:  "Which supplicants offer TEAP, and behind which options",
		Reason: "No file in this repository records client behavior; answering it needs the supplicant matrix that blocks ENG-002.",
	},
	{
		Topic:  "Whether a real TEAP peer completes the Crypto-Binding and PAC exchange",
		Reason: "The handshake harness in the tree drives a crypto/tls client through EAP-TLS, PEAP and TTLS framing only; a TEAP conversation needs a peer that speaks it.",
	},
	{
		Topic:  "Whether the deployed NAS fleet would carry a TEAP session",
		Reason: "No stored NAS capability data exists; the pilot that collects it is blocked as ENG-013.",
	},
	{
		Topic:  "Whether machine-then-user chaining is accepted by real peers",
		Reason: "Chaining behavior is client-specific, and this prototype claims no client result; it needs supplicant hardware.",
	},
	{
		Topic:  "What RFC 9427 and RFC 9930 require of TEAP key derivation and inner AVPs",
		Reason: "docs/rfcs carries no text for either specification, so those two mechanisms are named from their RFC numbers but not quoted; the local EAP-TLS/TTLS documents do not cover them.",
	},
}

// Tree is the comment-free view of the product code a scan reads. Values are the
// stripped lines of each file, kept in order so line numbers still point at the
// source and comments never supply evidence.
type Tree struct {
	// Files maps a repository-relative path to its comment-stripped lines.
	Files map[string][]string
}

// paths returns the file paths of the tree in lexical order.
func (t Tree) paths() []string {
	paths := make([]string, 0, len(t.Files))
	for path := range t.Files {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// exprCache holds one compiled marker pattern per pattern. A readiness scan runs
// every marker against every product file, so compiling once keeps the survey a
// unit-test-sized job instead of a minute of regexp setup.
var exprCache sync.Map

// match returns the lines of path in which pattern appears. Identifier patterns
// are matched at word boundaries, so a longer name cannot prove a shorter one;
// literal fragments are matched as text.
func match(path string, lines []string, pattern string, literal bool) []Evidence {
	re := compiled(pattern, literal)
	var out []Evidence
	for i, line := range lines {
		if re.MatchString(line) {
			out = append(out, Evidence{File: path, Line: i + 1, Text: collapseSpace(line)})
		}
	}
	return out
}

// compiled builds (or reuses) the regexp for one pattern.
func compiled(pattern string, literal bool) *regexp.Regexp {
	key := pattern
	if !literal {
		key = "\x00" + pattern
	}
	if cached, ok := exprCache.Load(key); ok {
		return cached.(*regexp.Regexp)
	}
	expr := `\b` + regexp.QuoteMeta(pattern) + `\b`
	if literal {
		expr = regexp.QuoteMeta(pattern)
	}
	re := regexp.MustCompile(expr)
	exprCache.Store(key, re)
	return re
}

// collapseSpace trims a source line and squeezes its inner whitespace, so the
// evidence a report prints is readable without losing the token it proves.
func collapseSpace(line string) string {
	return strings.Join(strings.Fields(line), " ")
}

// Analyze judges every cataloged mechanism against tree. Absence is only ever
// claimed against a tree that was actually read: an empty tree makes everything
// missing, so callers use Scan, which fails when a product directory is absent.
func Analyze(tree Tree) Report {
	mechanics := Mechanics()
	sort.SliceStable(mechanics, func(i, j int) bool { return mechanics[i].ID < mechanics[j].ID })

	report := Report{Unknown: append([]Unknown(nil), unknowns...)}
	for _, m := range mechanics {
		finding := Finding{Mechanic: m, Note: m.Note}
		var support, against []Evidence
		matched := map[string]bool{}
		for _, path := range tree.paths() {
			lines := tree.Files[path]
			for _, marker := range m.Markers {
				found := match(path, lines, marker, false)
				if len(found) == 0 {
					continue
				}
				matched[marker] = true
				support = append(support, found...)
			}
			for _, blocker := range m.Against {
				against = append(against, match(path, lines, blocker, true)...)
			}
		}
		// One marker proves one mechanism once, however many files use it.
		finding.MarkerHits = len(matched)

		switch {
		case len(against) > 0:
			finding.Status = StatusBlocked
		case finding.MarkerHits == 0:
			finding.Status = StatusMissing
		case finding.MarkerHits == len(m.Markers):
			finding.Status = StatusReady
		default:
			finding.Status = StatusPartial
		}
		// Keep the earliest lines: a verdict is proved by the first definitions
		// an implementer would find, not by every later use.
		finding.Support = firstEvidence(support)
		finding.Against = firstEvidence(against)
		report.Findings = append(report.Findings, finding)
	}
	report.Totals = totals(report.Findings)
	return report
}

// firstEvidence caps one side of a verdict at maxEvidence entries.
func firstEvidence(in []Evidence) []Evidence {
	if len(in) <= maxEvidence {
		return in
	}
	return append([]Evidence(nil), in[:maxEvidence]...)
}

// totals counts findings per status.
func totals(findings []Finding) Totals {
	var t Totals
	for _, f := range findings {
		t.Mechanics++
		switch f.Status {
		case StatusReady:
			t.Ready++
		case StatusPartial:
			t.Partial++
		case StatusBlocked:
			t.Blocked++
		case StatusMissing:
			t.Missing++
		}
	}
	return t
}
