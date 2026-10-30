// Package teapreadiness reports which mechanisms RFC 9930 (TEAP Version 1)
// expects of an authentication server, and which of them this repository's EAP
// server already implements.
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
	// rfcTEAP is the current TEAP Version 1 specification, vendored as
	// docs/rfcs/rfc9930-teap-version-1.txt. It obsoletes RFC 7170, which stays in
	// the archive only as the history of the clauses this catalog no longer cites.
	rfcTEAP = "RFC 9930"
	// rfcTLSTunnel is the TLS 1.3 transport TEAP builds on.
	rfcTLSTunnel = "RFC 9190"
	// rfcCUI is the chargeable-user-identity specification behind the anonymous
	// outer identity the server already sends.
	rfcCUI = "RFC 4372"
	// rfcCryptoAgility carries the TLS 1.3 key derivations TEAP uses, vendored as
	// docs/rfcs/rfc9427-tls-based-eap-types-tls13.txt. RFC 9930 updates it.
	rfcCryptoAgility = "RFC 9427"
)

// Status is one mechanism's readiness in the tree that was scanned.
type Status string

// Statuses a mechanism can reach. They are reported verbatim, so the names read
// as the claim they back: ready means an implementer can reuse what is there,
// missing means nothing does it, blocked means the code actively pins the
// opposite, and partial means only some of the markers were found. Partial is
// also what a wire-format codec with no serving-path caller produces, so its
// note must say which half is absent.
const (
	StatusReady   Status = "ready"
	StatusPartial Status = "partial"
	StatusBlocked Status = "blocked"
	StatusMissing Status = "missing"
)

// Source is the specification clause that makes a Mechanic a requirement.
type Source struct {
	// RFC names the specification, e.g. "RFC 9930".
	RFC string
	// Section is the clause number, or "" when the requirement is quoted from
	// the document body rather than a numbered heading.
	Section string
	// Document is the vendored text under docs/rfcs that carries the clause. Every
	// catalog entry names one: a mechanism whose specification is not in the
	// repository cannot be cataloged as a requirement.
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
	// AgainstIn narrows where Against counts: each entry is a repository path
	// prefix, and a match in a file outside every prefix is not this mechanism's
	// contradiction. It exists because the same literal can be right for one method
	// and wrong for another - EAP-TTLS pins its tunnel to TLS 1.2 because its
	// phase-2 framing needs that handshake, which says nothing about whether TEAP
	// may. Empty means the whole scanned tree, the rule for a literal that is wrong
	// wherever it appears.
	AgainstIn []string
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

// Mechanics returns the requirement catalog in the order RFC 9930 introduces it.
// Every marker names an identifier an implementer would have to write, and every
// citation is quoted from a document this repository carries under docs/rfcs.
func Mechanics() []Mechanic {
	return []Mechanic{
		{
			ID:      "teap-eap-type",
			Title:   "TEAP has an assigned EAP method type the server can send",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc9930-teap-version-1.txt", Quote: "TEAP Message Format"},
			Markers: []string{"TypeTEAP"},
			Note:    "interfaces.go declares TypeTEAP = 55, the IANA assignment RFC 9930 §7.5 records, and the handler builds every EAP-Request with it. The same value is what teapkeys.Type carries as the TLS exporter context octet.",
		},
		{
			ID:      "teap-method-name",
			Title:   "The method has a name the registry and configuration can address",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc9930-teap-version-1.txt", Quote: "TEAP Message Format"},
			Markers: []string{"EAPMethodTEAP"},
			Note:    "teap_handler.go exports EAPMethodTEAP = \"eap-teap\" alongside the EAPMethodTLS, EAPMethodPEAP, EAPMethodTTLS and EAPMethodOTP names, which is the string the registry and the configuration use to address the method.",
		},
		{
			ID:      "teap-handler-wiring",
			Title:   "A TEAP handler is registered and reachable from the identity phase",
			Source:  Source{RFC: rfcTEAP, Section: "3.1", Document: "rfc9930-teap-version-1.txt", Quote: "In the first EAP-Request sent with EAP type=TEAP"},
			Markers: []string{"NewTEAPHandler", "NewTEAPHandlerWithConfig"},
			Note:    "plugins/init.go registers NewTEAPHandlerWithConfig with the settings-backed TEAP provider (and the bare constructor when no application context exists), and coordinator.go routes \"eap-teap\" to the handler registered under TypeTEAP. A peer that selects the method therefore reaches HandleIdentity and the Phase 1 tunnel.",
		},
		{
			ID:      "teap-config-choice",
			Title:   "Operators can select TEAP as radius.EapMethod",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc9930-teap-version-1.txt", Quote: "TEAP Message Format"},
			Markers: []string{"eap-teap"},
			Note:    "The radius.EapMethod enum now lists eap-md5, eap-mschapv2, eap-tls, eap-peap, eap-ttls and eap-teap, and the schema description explains that the method runs its Phase 1 tunnel and then the RFC 9930 Phase 2 exchange: inner EAP-TLS, and the Basic-Password-Auth prompt when the operator sets one.",
		},
		{
			ID:      "teap-version-negotiation",
			Title:   "The TEAP version field is negotiated and re-verified",
			Source:  Source{RFC: rfcTEAP, Section: "3.1", Document: "rfc9930-teap-version-1.txt", Quote: "The receiver of the Crypto-Binding TLV MUST verify that the version"},
			Markers: []string{"stateKeyTEAPVersion", "handleBindingResponse"},
			Note:    "The handler performs §3.1 negotiation: it answers the Start with version 1, accepts the peer's first message only when it carries version 1 (otherwise eap.ErrTEAPVersionUnsupported terminates the exchange) and refuses a version that changes mid-conversation. The second half of the row is now served too: handleBindingResponse decodes the peer's Crypto-Binding TLV through teaptlv.CryptoBinding, which refuses a TLV whose Received-Ver is not the negotiated TEAPv1, so the re-verification §3.1 asks of the TLV's receiver happens before any Compound MAC is checked.",
		},
		{
			ID:      "result-tlv",
			Title:   "Result TLV is read to finish phase 2",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.4", Document: "rfc9930-teap-version-1.txt", Quote: "Result TLV"},
			Markers: []string{"TypeTLV", "tlvTypeResult"},
			Note:    "PEAPv0 already decodes a Result TLV of type 3, which is the same encoding TEAP requires.",
		},
		{
			ID:      "crypto-binding-tlv",
			Title:   "Crypto-Binding TLV is constructed and verified",
			Source:  Source{RFC: rfcTEAP, Section: "3.6", Document: "rfc9930-teap-version-1.txt", Quote: "Phase 2 MUST always end with a Crypto-Binding TLV"},
			Markers: []string{"CryptoBinding", "CompoundMAC", "verifyTEAPCompoundMAC"},
			Note:    "teaptlv encodes and decodes the 76-octet Value of §4.2.13 — Reserved octet, version pair, Flags and Sub-Type enumerations, the nonce least-significant-bit rule and the Binding Response nonce derived from the request — and teapbind builds the §6.3 BUFFER over the TLV with both MAC fields zeroed, the other party's TEAP octet and the Phase 1 Outer TLVs of the first TEAP message in each direction, HMACs it with the TLS 1.3 HKDF hash, truncates to 20 octets and verifies in constant time. Phase 2 now sends and checks it: sendBinding puts the server's Crypto-Binding TLV and the Intermediate-Result — plus the Result TLV in the termination exchange — into the tunnel message, and handleBindingResponse applies §6.2.4's Flags rule, verifies each Compound MAC with verifyTEAPCompoundMAC and maps a failure to the §4.2.6 code that names it (2005–2009).",
		},
		{
			ID:      "eap-payload-tlv",
			Title:   "Inner EAP runs are carried in EAP-Payload TLVs",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.10", Document: "rfc9930-teap-version-1.txt", Quote: "EAP-Payload TLV"},
			Markers: []string{"NewEAPPayload", "EAPPayload"},
			Note:    "teaptlv encodes the TLV and bounds the inner packet by its own Length field. Phase 2 carries every inner EAP message that way: sendPhase2Identity and sendInnerFlight wrap the request in NewEAPPayload, and handlePhase2Packet reads the peer's with EAPPayload before handing it to the inner method, so the bare inner packet PEAP and EAP-TTLS put in their tunnels is never what TEAP puts on the wire.",
		},
		{
			ID:      "intermediate-result-tlv",
			Title:   "Intermediate-Result TLV reports progress before the final result",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.11", Document: "rfc9930-teap-version-1.txt", Quote: "Intermediate-Result TLV"},
			Markers: []string{"NewIntermediateResult", "IntermediateResult"},
			Note:    "teaptlv encodes the TLV, and §4.2.11's rule that its embedded TLVs stay optional is enforced by the codec. Phase 2 now uses the intermediate form: sendBinding sends an Intermediate-Result TLV beside every Crypto-Binding TLV, and handleBindingResponse reads the peer's status — both sides must report success before the method counts, while a failure still lets the plan move on to the next method as §3.6.2 allows.",
		},
		{
			ID:      "session-resumption",
			Title:   "Phase 1 resumption runs on TLS 1.3 session tickets, not on the deprecated PAC",
			Source:  Source{RFC: rfcTEAP, Section: "2", Document: "rfc9930-teap-version-1.txt", Quote: "the PAC has been deprecated"},
			Markers: []string{"EnableSessionTickets", "SessionTicketsDisabled", "teapTicketHolder"},
			Against: []string{"SessionTicketsDisabled: true"},
			// PEAP and EAP-TTLS may keep tickets off; only a method that means to
			// resume may not pin them off, and only these two files decide that.
			AgainstIn: []string{
				"internal/radiusd/plugins/eap/handlers/teap_config.go",
				"internal/radiusd/plugins/eap/tlsengine/engine.go",
			},
			Note: "RFC 9930 deprecated the PAC that RFC 7170 borrowed from EAP-FAST and resumes a TEAP session with the TLS 1.3 NewSessionTicket message instead. The engine keeps tickets behind an opt-in that carries a caller-supplied key, so a ticket survives the handshake that issued it, and NewSettingsTEAPConfigProvider is the method that opts in; the contradiction is scoped to those two files because PEAP and EAP-TTLS are entitled to issue none. The TEAP handler opens its tunnel through that provider and now also enforces §3.11: teapTicketHolder remembers the identities this handshake issued, associate records the credentials Phase 2 proved against every one of them, unwrap refuses — and deletes — a ticket whose Phase 2 never completed, and grantResumed grants from the cached MSK when Phase 1 resumed, the §3.5 path that skips Phase 2.",
		},
		{
			ID:      "inner-method-chaining",
			Title:   "Two inner methods chain, machine then user",
			Source:  Source{RFC: rfcTEAP, Section: "A.26", Document: "rfc9930-teap-version-1.txt", Quote: "TEAPv1 meets this requirement by supporting inner EAP method chaining"},
			Markers: []string{"NewPlan", "NewSession", "startPhase2"},
			Note:    "The tunnel state machine runs a chain, which is the M11 requirement the roadmap calls certificate plus password. RFC 9930 §3.6.2 defines the chained sequence and §3.6.5 bounds what may be chained; teapinner holds both halves of that. NewPlan refuses a tunneled method nested inside TEAP, an Inner Method that resumes, and one that binds no keys of its own, and applies the two §6.2.3 ordering checks the policy selects. NewSession hands the methods out one at a time - §3.6.2 allows no two running in parallel - and will not accept the next method's start until the current one's Intermediate-Result statuses and Crypto-Binding Flags are recorded, which is where §6.2.3's rule that an EMSK-deriving method must be bound by an EMSK Compound MAC is applied. KeysFor is §6.2.1's password rule and SwapMSCHAPv2MSK is §3.6.4's. Phase 2 is now the caller: startPhase2 builds the plan phase2Plan describes - inner EAP-TLS always, plus Basic-Password-Auth when a prompt is configured - owns its Session and key chain, and beginMethod runs one method at a time inside the tunnel.",
		},
		{
			ID:      "identity-type-tlv",
			Title:   "Identity-Type TLV distinguishes machine from user identity",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.3", Document: "rfc9930-teap-version-1.txt", Quote: "Identity-Type TLV"},
			Markers: []string{"identityType", "IdentityType"},
			Note:    "Chaining is meaningless without the TLV that says whose credentials are being proved. teaptlv encodes §4.2.3's two values and refuses any others; Phase 2 now asks: sendPhase2Identity opens the inner exchange with a user Identity-Type TLV, and handlePhase2Identity rejects an answer that names another type while tolerating an absent one, since §4.2.3 makes the echo a SHOULD rather than a MUST.",
		},
		{
			ID:      "authority-id-tlv",
			Title:   "Authority-ID TLV identifies the provisioning authority",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.2", Document: "rfc9930-teap-version-1.txt", Quote: "Authority-ID TLV"},
			Markers: []string{"NewAuthorityID", "teapAuthorityID", "provisionTEAPAuthority"},
			Note:    "teaptlv encodes the opaque Authority-ID value of §4.2.2 and the handler sends one in its Start, deriving it from the realm the peer addressed so §3.4's realm-versus-certificate check has something to compare. RFC 9930 keeps the TLV for certificate provisioning inside the tunnel (§3.11) as well, not for the deprecated PAC, and provisionTEAPAuthority names that step: a peer that returns an Authority-ID is only relevant to provisioning, which this server does not offer, so normal authentication never consumes one.",
		},
		{
			ID:      "basic-password-auth-tlv",
			Title:   "Basic-Password-Auth TLVs carry a password without an inner EAP method",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.14", Document: "rfc9930-teap-version-1.txt", Quote: "Basic-Password-Auth-Req TLV"},
			Markers: []string{"NewBasicPasswordAuthReq", "NewBasicPasswordAuthResp", "PasswordRounds", "sendPasswordRequest", "handlePasswordResponse"},
			Note:    "This is the certificate-plus-password path the roadmap wants, and it is a TLV exchange, not an inner method. The §4.2.14 prompt and the §4.2.15 Userlen/Username/Passlen/Password layout are encodable now - teaptlv refuses a zero length on either field - and teapinner.PasswordRounds enforces the §3.6.3 rules around them: the first request of a session carries a prompt, a later round keeps the username it started with, and the exchange stops at the configured round-trip limit. Phase 2 now serves it: sendPasswordRequest puts the prompt TLV in the tunnel, handlePasswordResponse reads the response and checks it against the verifier or the configured password provider in constant time, and finishMethod places the exchange in the chain. RFC 9930 §4.2.8 names Basic-Password-Auth-Req as the case whose MSK and EMSK cannot come from cryptographic primitives and MUST be set to zero, and RFC 9427 §2.2 turns that into 32 zero octets for the inner session key; teapinner.KeysFor is what applies that to the chain.",
		},
		{
			ID:      "tls13-outer-tunnel",
			Title:   "The outer tunnel can negotiate TLS 1.3",
			Source:  Source{RFC: rfcTLSTunnel, Document: "rfc9190-eap-tls13.txt", Quote: "use of EAP-TLS with TLS 1.3"},
			Markers: []string{"VersionTLS13", "NewSettingsTEAPConfigProvider", "NewTEAPHandler"},
			Against: []string{"MaxVersion: tls.VersionTLS12"},
			// EAP-TTLS caps its own tunnel by design; that decision is not a
			// contradiction of what TEAP must do, so only the TEAP provider counts.
			AgainstIn: []string{"internal/radiusd/plugins/eap/handlers/teap_config.go"},
			Note:      "NewSettingsTEAPConfigProvider builds the TEAP tunnel with no version cap, so TEAP negotiates TLS 1.3 whenever the configured floor allows it, which is the transport RFC 9930's definitions are written for; the TLS 1.2 pin EAP-TTLS keeps in tls_config.go stays correct for that method's phase-2 framing and is outside this row's scope. The handler is now the caller: it builds its engine from that provider and, once the handshake reports done, reads NegotiatedVersion and terminates with eap.ErrTEAPRequiresTLS13 unless it is TLS 1.3, because teapkeys implements only the RFC 9427 TLS 1.3 exporter form.",
		},
		{
			ID:      "rfc9427-key-derivation",
			Title:   "Session keys derive through the RFC 9427 KDF",
			Source:  Source{RFC: rfcCryptoAgility, Section: "2.2", Document: "rfc9427-tls-based-eap-types-tls13.txt", Quote: `IMSK[j] = TLS-Exporter("TEAPbindkey@ietf.org", secret, 32)`},
			Markers: []string{"SessionKeySeed", "NewChain"},
			Note:    "teapkeys derives every key RFC 9427 §2.2 names for the method: the session key seed exported with the TEAP Type as its context, IMSK[j] from the Inner Method's EMSK or MSK with 32 zero octets when it produced neither, the two IMCK chains that split into S-IMCK[j] and CMK[j], and the outer MSK and EMSK as two separate 64-octet exports. §2.1's warning that TLS 1.3 export output depends on the length is enforced by refusing any other octet count rather than truncating. Phase 2 now carries the chain: startPhase2 seeds it with teapkeys.SessionKeySeed and NewChain over the completed tunnel, completeInnerMethod advances it with the Inner Method's exported keys, handleBindingResponse takes CompoundMACKeys from it, and finishPhase2 turns its SessionKeys into the MS-MPPE attributes of the Access-Accept. Only the TLS 1.3 form lives there - RFC 9930 §6.2 gives the TLS 1.2 TLS-PRF version, and the two derive different keys from the same inputs.",
		},
		{
			ID:      "exported-keying-material",
			Title:   "The tunnel exports keying material for the inner method",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.13", Document: "rfc9930-teap-version-1.txt", Quote: "Crypto-Binding TLV"},
			Markers: []string{"ExportKeyingMaterial"},
			Note:    "RFC 5705 export already happens inside the established tunnel, which is where a Crypto-Binding calculation would hook in.",
		},
		{
			ID:      "tunneled-eap-framing",
			Title:   "Inner EAP packets are parsed and normalized inside the tunnel",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.10", Document: "rfc9930-teap-version-1.txt", Quote: "EAP-Payload TLV"},
			Markers: []string{"parseInnerEAP", "normalizeInnerEAP"},
			Note:    "PEAP's inner framing already does the header normalization a TEAP EAP-Payload TLV would reuse.",
		},
		{
			ID:      "vendor-specific-tlv",
			Title:   "Vendor attributes reach the peer inside a Vendor-Specific TLV",
			Source:  Source{RFC: rfcTEAP, Section: "4.2.8", Document: "rfc9930-teap-version-1.txt", Quote: "The Vendor-Specific TLV is available to allow vendors to support"},
			Markers: []string{"vendorSpecificTLV", "encodeVendorTLV", "parseVendorTLVs", "NewVendorSpecific"},
			Note:    "RFC 9930 nests vendor-defined TLVs in a Vendor-Specific TLV of its own. teaptlv now supplies that envelope - Vendor-Id plus the vendor's octets, which it passes through untouched because their format belongs to the vendor - while the ttls_avp.go reader remains EAP-TTLS AVPs, the closest precedent but not the encoding TEAP requires. §4.2.8 still adds the rule that a vendor TLV carrying an authentication protocol MUST define MSK and EMSK values, and that is the vendor's own contract, not something this codec can check.",
		},
		{
			ID:      "tls-fragmentation",
			Title:   "Oversized tunnel records are fragmented and reassembled",
			Source:  Source{RFC: rfcTEAP, Section: "4.1", Document: "rfc9930-teap-version-1.txt", Quote: "More fragments; set on all but the last fragment."},
			Markers: []string{"FlagOuterTLVs", "Reassembler"},
			Note:    "§3.10 defers fragmentation to RFC 5216 §2.1.5, but TEAP carries it in its own envelope: §4.1 adds the Ver field, the four-octet Message Length and the Outer TLV Length, none of which the EAP-TLS layout has. teapmsg frames and reassembles that envelope, including the Outer TLVs §6.3 puts in the Crypto-Binding BUFFER, and the handler drives both halves: every outbound flight is cut by teapmsg.Fragment into a queue released one ACK at a time, and inbound fragments pass through a Reassembler whose partial state is persisted between rounds.",
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
		Topic:  "Whether a real TEAP peer completes the Crypto-Binding and Result exchange",
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
		Topic:  "Whether inner-method combinations beyond those RFC 9930 §5.1 names behave the same",
		Reason: "The specification limits its own claims to the combinations it documents and attributes the rest to how a given implementation encodes the Compound MAC; settling it needs a real TEAP peer, which is the ENG-002 gate.",
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

// inScope reports whether path falls under one of the prefixes. No prefixes means
// every path in the tree is in scope.
func inScope(path string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
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
				if !inScope(path, m.AgainstIn) {
					continue
				}
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
