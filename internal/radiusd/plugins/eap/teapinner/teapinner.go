// Package teapinner enforces the bounds RFC 9930 places on the Inner Methods that
// make up TEAP Phase 2: which methods may run, in what order, and what each one
// owes the Crypto-Binding exchange before the next one starts.
//
// TEAP's reason to chain methods is that one conversation can carry two
// authentications — a machine certificate and then a user password — while EAP
// itself forbids multiple methods in one exchange (RFC 3748 §2.1, quoted at
// RFC 9930 §3.6.2). That freedom is what makes the ordering decisions load
// bearing, so this package holds the rules rather than leaving them to each
// caller: a tunneled method must not be nested inside TEAP (§3.6.5), an inner
// method must not resume (§3.6.5), an inner method that binds no keys to the
// tunnel must not run (§3.6.5), a method that derives an EMSK must be bound by an
// EMSK Compound MAC (§6.2.3), and a password round must keep the username it
// started with (§3.6.3).
//
// Nothing here touches the tunnel, keeps TLS state or computes a key. The
// handshake that owns a TEAP session asks this package what is allowed, feeds
// each method's keys to teapkeys, and lets teapbind compute the Compound MACs
// whose presence this package checks. The TLV encodings are teaptlv's, so a
// password exchange is checked against the same codec that will put it on the
// wire, and the key rules are teapkeys', so a method's contribution to the chain
// is the value the derivation will actually fold in.
package teapinner

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
)

// mskHalfLen is the width of one of the two halves §3.6.4 swaps.
const mskHalfLen = 16

// defaultPasswordRounds is the round-trip limit §3.6.3 asks implementations to set:
// "It is reasonable to use one or two round trips."
const defaultPasswordRounds = 2

var (
	// ErrTunneledMethod rejects PEAP, EAP-TTLS and EAP-FAST as Inner Methods. §3.6.5
	// states the rule and its reason: "There is no reason to have multiple layers of
	// TLS in order to protect a password exchange."
	ErrTunneledMethod = errors.New("teapinner: tunneled EAP method must not be an Inner Method")
	// ErrInnerResumption rejects a resumed Inner Method. §3.6.5 requires a full
	// authentication per method and points anyone who wants resumption at the outer
	// TLS session instead.
	ErrInnerResumption = errors.New("teapinner: Inner Methods must not resume")
	// ErrNoCryptoBinding rejects an Inner Method that binds no keys of its own to the
	// tunnel. §3.6.5 makes it fatal: implementations "MUST NOT permit the use of
	// inner EAP methods that fail to perform crypto-binding".
	ErrNoCryptoBinding = errors.New("teapinner: Inner Method does not crypto-bind")
	// ErrUnboundPassword rejects a plan whose only authentication is a password
	// exchange. §6.2.1 gives the reason: with no MSK and no EMSK, "using an MSK of all
	// zeroes opens up TEAP to on-path attacks", so such a method is usable only in
	// conjunction with one that does derive keys.
	ErrUnboundPassword = errors.New("teapinner: password authentication has no key-deriving method beside it")
	// ErrEMSKFirst rejects a plan that does not start with an EMSK-deriving method when
	// the policy demands it. §6.2.3 lists the check and §6.2.1 the reason: the first
	// binding is what detects an on-path attacker before a password is sent.
	ErrEMSKFirst = errors.New("teapinner: the first Inner Method must derive an EMSK")
	// ErrNoBinding means a successful method was not accompanied by a Crypto-Binding
	// TLV, which §3.6.2 requires on both sides of the tunnel.
	ErrNoBinding = errors.New("teapinner: successful Inner Method without a Crypto-Binding TLV")
	// ErrNoEMSKBinding means a method that derived an EMSK was bound by the MSK Compound
	// MAC only. §6.2.3 lists this as one of the checks an implementation can enforce.
	ErrNoEMSKBinding = errors.New("teapinner: EMSK-deriving Inner Method bound without an EMSK Compound MAC")
	// ErrUnexpectedBinding means a failed method arrived with a Crypto-Binding TLV.
	// §6.2.4 states the other half of the rule it breaks: "If the Inner Method has
	// failed, the server does not send a Crypto-Binding TLV."
	ErrUnexpectedBinding = errors.New("teapinner: failed Inner Method carries a Crypto-Binding TLV")
	// ErrNoPrompt rejects the first Basic-Password-Auth-Req TLV of a session without a
	// prompt, which §3.6.3 makes a MUST because it is what the user is asked for.
	ErrNoPrompt = errors.New("teapinner: first password request must carry a prompt")
	// ErrIdentityChanged rejects a password round whose username differs from the first
	// one. §3.6.3: "There is no reason for a user (or machine) to change identities in
	// the middle of authentication."
	ErrIdentityChanged = errors.New("teapinner: password rounds changed the username")
	// ErrTooManyRounds rejects a password exchange that exceeds the configured limit,
	// which §3.6.3 states as a MUST to bound the number of round trips.
	ErrTooManyRounds = errors.New("teapinner: password round-trip limit exceeded")
	// ErrNotRunning means the handshake asked for the outcome of a method this session
	// never started.
	ErrNotRunning = errors.New("teapinner: no Inner Method is running")
	// ErrMSKLength means SwapMSCHAPv2MSK was handed an MSK that is not the two 16-octet
	// halves §3.6.4 swaps.
	ErrMSKLength = errors.New("teapinner: MSK is not two 16-octet halves")
	// ErrMissingMethodKeys means a method that the configuration says derives keys
	// reached the chain with none, which is a handshake bug rather than a policy
	// choice: the keys exist or the method did not complete.
	ErrMissingMethodKeys = errors.New("teapinner: Inner Method derives keys and supplied none")
)

// Method is one Inner Method as the server's configuration describes it. The
// capability fields are what the §3.6.5 and §6.2 bounds read, so they state what
// the method actually produces, not what its name suggests: a variant that
// reuses an EAP Type but derives different keys is a different Method.
type Method struct {
	// Name identifies the method in a log or an error and carries no meaning here.
	Name string
	// Password marks the Basic-Password-Auth exchange of §3.6.3. It is not an EAP
	// method: the peer's credentials travel in a TLV, and no MSK or EMSK is derived.
	Password bool
	// DerivesMSK and DerivesEMSK state which keys the method hands to the chain. §3.6.5
	// groups the permitted methods by exactly this: methods deriving both, and methods
	// deriving only an MSK.
	DerivesMSK  bool
	DerivesEMSK bool
	// CryptoBinds reports whether the method's own keys are bound to the tunnel by a
	// Crypto-Binding TLV. §3.6.5 admits no method that does not.
	CryptoBinds bool
	// Tunneled marks a method that is itself a TLS tunnel — PEAP, EAP-TTLS, EAP-FAST —
	// which §3.6.5 forbids inside TEAP.
	Tunneled bool
	// Resumable marks a method that can resume its own session, which §3.6.5 forbids as
	// an Inner Method.
	Resumable bool
}

// String returns the name, so a Method prints as itself in an error or a log line.
func (m Method) String() string { return m.Name }

// EAPTLS and EAPFASTMSCHAPv2 are the pair §3.6.5 names as the commonly supported,
// interoperable Inner Methods: "Implementations SHOULD limit the permitted inner
// EAP methods to a small set such as EAP-TLS and the EAP-FAST-MSCHAPv2 variant of
// EAP-MSCHAPv2." EAP-TLS derives both keys; the MSCHAPv2 variant derives an MSK
// only, which is why §3.6.5 says both shapes are known to work.
//
// They carry no EAP Type number: the archive this repository holds assigns numbers
// only to TEAP itself (RFC 9930 §7.5), and a number copied in from elsewhere would
// be a claim this tree cannot check.
var (
	EAPTLS = Method{
		Name:        "EAP-TLS",
		DerivesMSK:  true,
		DerivesEMSK: true,
		CryptoBinds: true,
	}
	EAPFASTMSCHAPv2 = Method{
		Name:        "EAP-FAST-MSCHAPv2",
		DerivesMSK:  true,
		CryptoBinds: true,
	}
	// BasicPassword is the §3.6.3 exchange. It crypto-binds nothing of its own, so it
	// is only ever admissible beside a method that derives keys.
	BasicPassword = Method{
		Name:     "Basic-Password-Auth",
		Password: true,
	}
)

// Policy is the set of configurable checks §6.2.3 describes: "Implementations can
// provide configuration flags, policies, or documented recommendations that
// control the type of Inner Methods used or verify their order." Each flag is one
// of the checks that document lists, so a relaxed flag is a documented deviation
// rather than an undocumented hole.
type Policy struct {
	// RequireEMSKFirst enforces "Verifying that the first Inner Method used is one that
	// derives EMSK."
	RequireEMSKFirst bool
	// ForbidUnboundPassword refuses a plan whose password exchange has no key-deriving
	// method beside it, which §6.2.1 recommends and §3.6.5's binding requirement
	// points at.
	ForbidUnboundPassword bool
	// MaxPasswordRounds bounds the password exchange. Zero selects the default of two,
	// the upper value §3.6.3 calls reasonable.
	MaxPasswordRounds int
}

// DefaultPolicy returns the strict reading: both §6.2.3 checks enforced and the
// two-round password limit. A server that wants a looser Phase 2 says so through
// this struct, so the deviation is visible in configuration.
func DefaultPolicy() Policy {
	return Policy{RequireEMSKFirst: true, ForbidUnboundPassword: true}
}

// passwordRounds is the configured limit, with the §3.6.3 default applied.
func (p Policy) passwordRounds() int {
	if p.MaxPasswordRounds <= 0 {
		return defaultPasswordRounds
	}
	return p.MaxPasswordRounds
}

// Plan is the ordered sequence of Inner Methods one TEAP session will run. TEAP
// runs them serially — §3.6.2: "Inner Methods are executed serially in a
// sequence. This version of TEAP does not support initiating multiple Inner
// Methods simultaneously in parallel" — so the order is the plan, and the same
// method may appear twice, which is how §3.6.2's own example runs EAP-TLS once for
// the machine and once for the user.
//
// An empty plan is legal: §3.6.6 states that "A TEAP server may initiate the
// Crypto-Binding TLV and Result TLV exchange without initiating any Inner Methods
// in TEAP Phase 2."
type Plan struct {
	policy  Policy
	methods []Method
}

// NewPlan validates an ordered sequence against a policy and returns the plan the
// session will run.
func NewPlan(policy Policy, methods ...Method) (Plan, error) {
	plan := Plan{policy: policy, methods: make([]Method, 0, len(methods))}
	plan.methods = append(plan.methods, methods...)
	var derivedKeys bool
	for _, m := range plan.methods {
		if err := checkMethod(m); err != nil {
			return Plan{}, err
		}
		if m.DerivesMSK || m.DerivesEMSK {
			derivedKeys = true
		}
	}
	if policy.ForbidUnboundPassword && !derivedKeys {
		for _, m := range plan.methods {
			if m.Password {
				return Plan{}, fmt.Errorf("%w: %q is the only Inner Method", ErrUnboundPassword, m.Name)
			}
		}
	}
	if policy.RequireEMSKFirst && len(plan.methods) > 0 && !plan.methods[0].DerivesEMSK {
		return Plan{}, fmt.Errorf("%w: %q runs first", ErrEMSKFirst, plan.methods[0].Name)
	}
	return plan, nil
}

// checkMethod applies the three admissions §3.6.5 states as MUST NOT.
func checkMethod(m Method) error {
	if m.Tunneled {
		return fmt.Errorf("%w: %q tunnels EAP itself", ErrTunneledMethod, m.Name)
	}
	if m.Resumable {
		return fmt.Errorf("%w: %q can resume", ErrInnerResumption, m.Name)
	}
	// The password exchange is not an inner EAP method, so the §3.6.5 crypto-binding
	// requirement reaches it through §6.2.1's conjunction rule, which the plan checks.
	if !m.CryptoBinds && !m.Password {
		return fmt.Errorf("%w: %q derives no key to bind", ErrNoCryptoBinding, m.Name)
	}
	return nil
}

// Methods returns the sequence in the order it will run.
func (p Plan) Methods() []Method { return append([]Method(nil), p.methods...) }

// PasswordRounds returns the limit this plan's policy sets for a password exchange.
func (p Plan) PasswordRounds() int { return p.policy.passwordRounds() }

// Session is one Plan in progress. It hands out the next method, then takes the
// result of the exchange that ran it, because the two are the same decision: §3.6.2
// makes each method an Intermediate-Result exchange plus a Crypto-Binding TLV, and
// its outcome decides whether the sequence continues.
//
// A Session is not safe for concurrent use; one TEAP session owns one Session.
type Session struct {
	plan Plan
	// idx counts the methods handed out, so Next only ever returns each one once.
	idx int
	// running holds the method whose outcome is still owed.
	running Method
	// pending reports that such a method exists.
	pending bool
	// ok lists the methods both sides accepted, in the order they ran.
	ok []Method
}

// NewSession starts a Plan at its first method.
func NewSession(plan Plan) *Session { return &Session{plan: plan} }

// Next returns the Inner Method to run now, or false when the sequence is complete
// or the previous method has not been recorded yet.
func (s *Session) Next() (Method, bool) {
	if s.pending || s.idx >= len(s.plan.methods) {
		return Method{}, false
	}
	m := s.plan.methods[s.idx]
	s.idx++
	s.running, s.pending = m, true
	return m, true
}

// Running returns the method whose outcome is still owed, and false when the
// handshake is between methods.
func (s *Session) Running() (Method, bool) {
	if !s.pending {
		return Method{}, false
	}
	return s.running, true
}

// Record closes the exchange that ran the current method.
//
// serverStatus and peerStatus are the Intermediate-Result TLV statuses of §3.6.2,
// one per side, and flags is the Flags the peer announced in its Crypto-Binding TLV.
// Both sides have to say success for the method to count: §3.6.2 states that "If
// either indicates failure, then the authentication is considered failed."
//
// A failed method does not end the session. §3.6.2 is explicit that one failure
// does not imply the overall failure — "the server may attempt to authenticate the
// peer with a different method (EAP or password)" — so the caller simply asks for
// the next method.
//
// Record is atomic: an error means the values handed to it do not describe one
// exchange, and the method stays running so the caller can record it correctly.
func (s *Session) Record(serverStatus, peerStatus teaptlv.Status, flags uint8) error {
	if !s.pending {
		return ErrNotRunning
	}
	if serverStatus != teaptlv.StatusSuccess || peerStatus != teaptlv.StatusSuccess {
		// §6.2.4: a method that did not succeed is not bound, so a Crypto-Binding TLV
		// in that exchange is not something the peer should have sent.
		if flags != 0 {
			return fmt.Errorf("%w: %q failed with Flags %d", ErrUnexpectedBinding, s.running.Name, flags)
		}
		s.pending = false
		return nil
	}
	if err := s.recordBinding(s.running, flags); err != nil {
		return err
	}
	s.pending = false
	s.ok = append(s.ok, s.running)
	return nil
}

// recordBinding checks what a successful method owes the binding, which is where
// §6.2.3's second configurable check lives: "Verifying that if any Inner Method
// derives EMSK, the received Crypto-Binding TLV for that method contains an EMSK
// Compound MAC."
func (s *Session) recordBinding(m Method, flags uint8) error {
	if m.Password {
		// The password exchange derives nothing, so there is nothing for it to bind and
		// §6.2.1 puts zeros in its place. The TLV that follows it still has to bind
		// whatever the session has derived up to that point.
		if flags == 0 {
			return fmt.Errorf("%w: %q succeeded without a Crypto-Binding TLV", ErrNoBinding, m.Name)
		}
		return nil
	}
	if flags == 0 {
		return fmt.Errorf("%w: %q succeeded without a Crypto-Binding TLV", ErrNoBinding, m.Name)
	}
	if m.DerivesEMSK && flags&teaptlv.FlagEMSKPresent == 0 {
		return fmt.Errorf("%w: %q derived an EMSK and Flags is %d", ErrNoEMSKBinding, m.Name, flags)
	}
	return nil
}

// Authenticated returns the Inner Methods both sides accepted, in order. The
// handshake compares this against what its own policy demands — the machine
// certificate and the user password, for instance — because which combinations are
// sufficient is a site decision, not a §3.6 one.
func (s *Session) Authenticated() []Method { return append([]Method(nil), s.ok...) }

// Complete reports whether every method in the plan has been run and none is
// waiting for its outcome. A session that still has methods left has not yet
// reached the §3.6.6 protected termination.
func (s *Session) Complete() bool {
	return !s.pending && s.idx >= len(s.plan.methods)
}

// KeysFor turns one method's keys into the input teapkeys.Chain.Advance takes,
// which is where §6.2.1's rule for the password exchange is applied rather than
// left to each caller: "If the Inner Method does not provide either MSK or EMSK,
// such as when basic password authentication is used or when no Inner Method has
// been run, then both MSK and IMSK[j] are set to all zeroes."
//
// A method that claims to derive keys but hands over none is a handshake bug, so it
// is reported instead of silently folded into the chain as zeros.
func KeysFor(m Method, msk, emsk []byte) (teapkeys.InnerMethodKeys, error) {
	if m.Password {
		return teapkeys.InnerMethodKeys{}, nil
	}
	if !m.DerivesMSK && !m.DerivesEMSK {
		return teapkeys.InnerMethodKeys{}, fmt.Errorf(
			"%w: %q claims no MSK and no EMSK", ErrNoCryptoBinding, m.Name)
	}
	if m.DerivesMSK && len(msk) == 0 {
		return teapkeys.InnerMethodKeys{}, fmt.Errorf(
			"%w: %q derives an MSK and none was supplied", ErrMissingMethodKeys, m.Name)
	}
	if m.DerivesEMSK && len(emsk) == 0 {
		return teapkeys.InnerMethodKeys{}, fmt.Errorf(
			"%w: %q derives an EMSK and none was supplied", ErrMissingMethodKeys, m.Name)
	}
	return teapkeys.InnerMethodKeys{MSK: msk, EMSK: emsk}, nil
}

// SwapMSCHAPv2MSK returns the EAP-FAST-MSCHAPv2 form of an EAP-MSCHAPv2 MSK, which
// §3.6.4 requires TEAP to use as the Inner Method Session Key: "the first and the
// second 16 octets of the EAP-MSCHAPv2 Master Session Key (MSK) are swapped when it
// is used as the Inner Method Session Keys (IMSKs) for TEAP."
//
// It is a swap of halves, not a truncation or a re-derivation, so the result keeps
// every octet the MSK had.
func SwapMSCHAPv2MSK(msk []byte) ([]byte, error) {
	if len(msk) != 2*mskHalfLen {
		return nil, fmt.Errorf("%w: %d octets, §3.6.4 swaps two runs of %d", ErrMSKLength, len(msk), mskHalfLen)
	}
	out := make([]byte, 0, 2*mskHalfLen)
	out = append(out, msk[mskHalfLen:]...)
	out = append(out, msk[:mskHalfLen]...)
	return out, nil
}

// PasswordRounds is one Basic-Password-Auth exchange in progress: the request and
// response TLVs of §4.2.14 and §4.2.15, guarded by the §3.6.3 rules about prompts,
// identities and round trips.
//
// The exchange belongs to one Inner Method of a Session, not to the session as a
// whole, so a plan that runs password authentication twice has one of these per
// method.
type PasswordRounds struct {
	limit    int
	rounds   int
	username []byte
}

// NewPasswordRounds starts an exchange with the round-trip limit §3.6.3 makes
// implementations set. A limit of zero selects the default of two.
func NewPasswordRounds(limit int) *PasswordRounds {
	if limit <= 0 {
		limit = defaultPasswordRounds
	}
	return &PasswordRounds{limit: limit}
}

// Request builds the next Basic-Password-Auth-Req TLV. The first one of a session
// MUST carry the prompt the peer shows the user (§3.6.3); a later one may send an
// empty prompt, in which case the peer is only asked for a password.
func (r *PasswordRounds) Request(prompt string) (teaptlv.TLV, error) {
	if r.rounds >= r.limit {
		return teaptlv.TLV{}, fmt.Errorf("%w: %d rounds configured, %d already sent", ErrTooManyRounds, r.limit, r.rounds)
	}
	if r.rounds == 0 && prompt == "" {
		return teaptlv.TLV{}, fmt.Errorf("%w: the first request has nothing to show the user", ErrNoPrompt)
	}
	r.rounds++
	return teaptlv.NewBasicPasswordAuthReq(prompt), nil
}

// Respond builds the Basic-Password-Auth-Resp TLV the peer sends back and is where
// the server's own obligations are enforced: the username of the first round is
// copied into every later round, and an identity that changes mid-authentication is
// rejected rather than authenticated (§3.6.3). teaptlv already refuses a zero-length
// username or password, so a malformed response never becomes a TLV.
func (r *PasswordRounds) Respond(username, password []byte) (teaptlv.TLV, error) {
	tlv, err := teaptlv.NewBasicPasswordAuthResp(username, password)
	if err != nil {
		return teaptlv.TLV{}, err
	}
	if len(r.username) == 0 {
		r.username = append([]byte(nil), username...)
		return tlv, nil
	}
	if !bytes.Equal(r.username, username) {
		return teaptlv.TLV{}, fmt.Errorf("%w: round %d sent %q instead of %q",
			ErrIdentityChanged, r.rounds, username, r.username)
	}
	return tlv, nil
}

// Rounds returns how many requests have been sent.
func (r *PasswordRounds) Rounds() int { return r.rounds }

// Username returns the identity the exchange is bound to, or nil before the first
// response. The handshake keeps this to compare against its own expectation of who
// is authenticating.
func (r *PasswordRounds) Username() []byte { return append([]byte(nil), r.username...) }
