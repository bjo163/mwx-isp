package handlers

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapbind"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapinner"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapmsg"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsfragment"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"layeh.com/radius"
)

// Phase 2 sub-states. A TEAP Phase 2 runs inside the protected tunnel: the
// server writes TLVs as TLS application data on the outer engine and the peer
// answers with TLVs the same way, so each sub-state names which TLV the server
// is waiting for.
const (
	// teapPhase2Identity waits for the peer's Identity-Type and
	// EAP-Response/Identity (§3.6.2: an EAP authentication inside the tunnel
	// begins with an EAP-Request/Identity).
	teapPhase2Identity = "identity"
	// teapPhase2InnerTLS drives the inner EAP-TLS handshake carried in
	// EAP-Payload TLVs.
	teapPhase2InnerTLS = "inner-tls"
	// teapPhase2WaitBinding waits for the peer's Intermediate-Result and
	// Crypto-Binding TLV (§3.6.2).
	teapPhase2WaitBinding = "wait-binding"
	// teapPhase2WaitPassword waits for the Basic-Password-Auth-Resp TLV of
	// §3.6.3.
	teapPhase2WaitPassword = "wait-password"
	// teapPhase2WaitResult waits for the peer's Result TLV and Crypto-Binding TLV
	// of the protected termination exchange (§3.6.6).
	teapPhase2WaitResult = "wait-result"
	// teapPhase2WaitFailure waits for the peer's Result TLV after the server
	// reported a fatal error (§3.9.3).
	teapPhase2WaitFailure = "wait-failure-result"
)

// teapPhase2NonceLen is the Crypto-Binding Nonce length of §4.2.13.
const teapPhase2NonceLen = 32

// Error-Code values of the Error TLV (§4.2.6) this server reports. The list is
// limited to the outcomes Phase 2 can produce.
const (
	teapErrInnerMethod       uint32 = 1001 // Inner Method Error
	teapErrAuthFailure       uint32 = 1003 // Unspecified authentication failure
	teapErrInnerNotSupported uint32 = 1032 // Inner Method not supported
	teapErrTunnelCompromise  uint32 = 2001 // Tunnel Compromise Error
	teapErrUnexpectedTLVs    uint32 = 2002 // Unexpected TLVs Exchanged
	teapErrInvalidBinding    uint32 = 2003 // The Crypto-Binding TLV is invalid
	teapErrMissingMSKCM      uint32 = 2005 // no required MSK Compound MAC
	teapErrBadMSKCM          uint32 = 2006 // the MSK Compound MAC fails verification
	teapErrMissingEMSKCM     uint32 = 2007 // no required EMSK Compound MAC
	teapErrBadEMSKCM         uint32 = 2008 // the EMSK Compound MAC fails verification
	teapErrUnexpectedEMSKCM  uint32 = 2009 // EMSK Compound MAC without an EMSK-deriving method
)

// teapPhase2Session is the Phase 2 exchange of one TEAP session, held on the
// EAP state so it survives the RADIUS round-trips.
//
// It owns the teapinner plan and session, which decide what may run and record
// each method's outcome, and the teapkeys chain, which folds each method's keys
// into the compound keys the Crypto-Binding TLV is computed with. The engines
// are not owned here: the outer engine lives on the EAP state and the inner
// EAP-TLS engine is a field because it exists only while its handshake runs.
type teapPhase2Session struct {
	plan     teapinner.Plan
	methods  []teapinner.Method
	idx      int
	session  *teapinner.Session
	chain    *teapkeys.Chain
	exchange teapbind.Exchange

	// method is the Inner Method currently running; it decides whether a
	// Crypto-Binding TLV must carry an EMSK Compound MAC (§6.2.3).
	method teapinner.Method
	// password is the password exchange of §3.6.3, nil unless one is running.
	password *teapinner.PasswordRounds

	// inner is the Phase 2 EAP-TLS engine and innerID the identifier of the last
	// inner EAP request the server sent.
	inner   *tlsengine.Engine
	innerID uint8
	// innerIdentity is the identity the peer sent in its EAP-Response/Identity
	// and certIdentity the one its certificate proves.
	innerIdentity string
	certIdentity  *tlsengine.PeerIdentity

	// passwordUser is the username the Basic-Password-Auth-Resp carried, which is
	// the identity the session is granted on when a password exchange ran.
	passwordUser string
	// nonce is the Nonce of the last Crypto-Binding request, whose least
	// significant bit the peer must flip in its response (§4.2.13).
	nonce []byte
	// failure is why a Result TLV indicating failure was sent, and the error the
	// coordinator reports once the peer acknowledges it (§3.9.3).
	failure error
}

// phase2Session returns the Phase 2 exchange stored on state, or nil.
func phase2Session(state *eap.EAPState) *teapPhase2Session {
	if state == nil || state.Data == nil {
		return nil
	}
	p2, _ := state.Data[stateKeyTEAPSession].(*teapPhase2Session)
	return p2
}

// closeInner releases the inner EAP-TLS engine if one is running.
func (p *teapPhase2Session) closeInner() {
	if p.inner != nil {
		_ = p.inner.Close()
		p.inner = nil
	}
}

// phase2Plan builds the Inner Method sequence this handler's options ask for:
// the EAP-TLS method always, plus the Basic-Password-Auth exchange when a prompt
// is configured. teapinner.NewPlan applies the §3.6.5 and §6.2 ordering rules, so
// an inadmissible combination is refused before the tunnel has anything to run.
func (h *TEAPHandler) phase2Plan() (teapinner.Plan, error) {
	policy := h.opts.Policy
	if policy == (teapinner.Policy{}) {
		policy = teapinner.DefaultPolicy()
	}
	methods := []teapinner.Method{teapinner.EAPTLS}
	if h.passwordPrompt() != "" {
		methods = append(methods, teapinner.BasicPassword)
	}
	return teapinner.NewPlan(policy, methods...)
}

// passwordPrompt returns the configured prompt, or the empty string.
func (h *TEAPHandler) passwordPrompt() string {
	if h.opts.PasswordPrompt == nil {
		return ""
	}
	return h.opts.PasswordPrompt()
}

// outerEngine returns the Phase 1 TLS engine, building it on first use.
//
// The engine is the same one the handshake ran on, because Phase 2's TLVs and
// the inner methods' records are all application data on it. Its ticket callbacks
// are wired here rather than in the configuration provider: the association of
// §3.11 belongs to this EAP state, and only the handler holding it can record
// which credentials Phase 2 proved.
func (h *TEAPHandler) outerEngine(state *eap.EAPState) (*tlsengine.Engine, error) {
	if state.Data != nil {
		if e, ok := state.Data[stateKeyEngine].(*tlsengine.Engine); ok && e != nil {
			return e, nil
		}
	}

	if h.configProvider == nil {
		return nil, eap.ErrTLSNotConfigured
	}
	cfg, err := h.configProvider()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", eap.ErrTLSNotConfigured, err)
	}
	if cfg == nil {
		return nil, eap.ErrTLSNotConfigured
	}

	holder := ticketHolder(state)
	cfg.WrapSession = holder.wrap
	cfg.UnwrapSession = holder.unwrap

	engine, err := tlsengine.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", eap.ErrTLSNotConfigured, err)
	}
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[stateKeyEngine] = engine
	return engine, nil
}

// innerEngine returns the Phase 2 EAP-TLS engine, building it on first use.
//
// The inner method is a second, complete EAP-TLS conversation whose records
// travel as application data inside the outer tunnel, so it needs its own engine
// and a configuration that authenticates the peer. Tickets stay off: §3.6.5
// requires an Inner Method to run a full authentication, and the inner handshake
// must be TLS 1.3 because only that version defines the EMSK §6.2.3 requires the
// EMSK Compound MAC over.
func (h *TEAPHandler) innerEngine(p2 *teapPhase2Session) (*tlsengine.Engine, error) {
	if p2.inner != nil {
		return p2.inner, nil
	}
	if h.opts.InnerTLSConfigProvider == nil {
		return nil, eap.ErrTEAPPhase2NotConfigured
	}
	cfg, err := h.opts.InnerTLSConfigProvider()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", eap.ErrTEAPPhase2NotConfigured, err)
	}
	if cfg == nil {
		return nil, eap.ErrTEAPPhase2NotConfigured
	}
	cfg.MinVersion = tls.VersionTLS13
	cfg.MaxVersion = tls.VersionTLS13
	cfg.EnableSessionTickets = false
	cfg.WrapSession = nil
	cfg.UnwrapSession = nil

	engine, err := tlsengine.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", eap.ErrTEAPPhase2NotConfigured, err)
	}
	p2.inner = engine
	return engine, nil
}

// startPhase2 opens Phase 2 after the tunnel is up: it derives the session key
// seed and compound-key chain of §6.2, fixes the Crypto-Binding inputs of §6.3
// from the Phase 1 Outer TLVs, and initiates the first Inner Method.
func (h *TEAPHandler) startPhase2(ctx *eap.EAPContext, state *eap.EAPState) (bool, error) {
	engine, err := h.outerEngine(state)
	if err != nil {
		return false, err
	}
	if !engine.HandshakeComplete() {
		return false, eap.ErrTEAPProtocol
	}

	plan, err := h.phase2Plan()
	if err != nil {
		return false, err
	}
	seed, err := teapkeys.SessionKeySeed(engine)
	if err != nil {
		return false, err
	}
	chain, err := teapkeys.NewChain(seed)
	if err != nil {
		return false, err
	}
	hash, err := teapbind.HashFromCipherSuite(engine.NegotiatedCipherSuite())
	if err != nil {
		return false, err
	}

	p2 := &teapPhase2Session{
		plan:    plan,
		methods: plan.Methods(),
		session: teapinner.NewSession(plan),
		chain:   chain,
		exchange: teapbind.Exchange{
			Hash:           hash,
			OtherPartyType: teapkeys.Type,
			Outer: teapbind.OuterTLVs{
				Server: getBytes(state, stateKeyTEAPServerOuter),
				Peer:   getBytes(state, stateKeyTEAPPeerOuter),
			},
		},
	}
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[stateKeyTEAPSession] = p2
	setBool(state, stateKeyTEAPPhase2, true)

	m, ok := p2.session.Next()
	if !ok {
		return false, fmt.Errorf("%w: Phase 2 has no Inner Method to run", eap.ErrTEAPPhase2NotConfigured)
	}
	p2.method = m
	return h.beginMethod(ctx, state, p2)
}

// beginMethod initiates the method teapinner handed out.
func (h *TEAPHandler) beginMethod(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session) (bool, error) {
	if p2.method.Password {
		return h.sendPasswordRequest(ctx, state, p2)
	}
	return h.sendPhase2Identity(ctx, state, p2)
}

// sendPhase2Identity sends the Identity-Type hint and the EAP-Request/Identity
// that opens an inner EAP authentication (§3.6.2).
func (h *TEAPHandler) sendPhase2Identity(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session) (bool, error) {
	p2.innerID++
	request := teapInnerRequest(p2.innerID, eap.TypeIdentity, nil)
	payload, err := teaptlv.NewEAPPayload(request)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}

	setString(state, stateKeyTEAPPhase, teapPhase2Identity)
	return false, h.sendPhase2TLVs(ctx, state, teaptlv.NewIdentityType(teaptlv.IdentityUser), payload)
}

// sendPasswordRequest sends the Basic-Password-Auth-Req TLV of §3.6.3. The first
// request of an exchange must carry the prompt the user is asked for, which is
// why a handler with no prompt never plans this method.
func (h *TEAPHandler) sendPasswordRequest(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session) (bool, error) {
	rounds := teapinner.NewPasswordRounds(p2.plan.PasswordRounds())
	p2.password = rounds
	request, err := rounds.Request(h.passwordPrompt())
	if err != nil {
		return h.failPhase2(ctx, state, teapErrAuthFailure, fmt.Errorf("%w: %v", eap.ErrTEAPPassword, err))
	}

	setString(state, stateKeyTEAPPhase, teapPhase2WaitPassword)
	return false, h.sendPhase2TLVs(ctx, state, request)
}

// handlePhase2Packet consumes one Phase 2 message: the peer's TLS records are
// reassembled, decrypted on the outer engine and dispatched to the handler of the
// current sub-state.
func (h *TEAPHandler) handlePhase2Packet(ctx *eap.EAPContext, state *eap.EAPState, pkt *teapmsgPacket) (bool, error) {
	p2 := phase2Session(state)
	if p2 == nil {
		return false, fmt.Errorf("%w: Phase 2 state is missing", eap.ErrTEAPProtocol)
	}
	// §4.1 allows the O flag only in the initial request and response, which the
	// peer sent long before the tunnel was up.
	if pkt.HasOuterTLVs {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: Outer TLVs inside Phase 2", eap.ErrTEAPProtocol))
	}

	reassembler := loadTEAPReassembler(state)
	complete, err := reassembler.Accept(pkt)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}
	if !complete {
		saveTEAPReassembler(state, reassembler)
		if serr := ctx.StateManager.SetState(state.StateID, state); serr != nil {
			return false, serr
		}
		ack, aerr := teapmsgACK(ctx.EAPMessage.Identifier + 1)
		if aerr != nil {
			return false, aerr
		}
		return false, h.writeChallenge(ctx, state.StateID, ack)
	}

	records := append([]byte(nil), reassembler.Buffer()...)
	resetTEAPReassembler(state)
	if len(records) == 0 {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: empty Phase 2 message", eap.ErrTEAPInnerProtocol))
	}

	engine, err := h.outerEngine(state)
	if err != nil {
		return false, err
	}
	plaintext, err := engine.ReadApplication(records)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrTunnelCompromise, fmt.Errorf("%w: %v", eap.ErrTEAPInnerProtocol, err))
	}
	tlvs, err := teaptlv.Parse(plaintext)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}

	switch getString(state, stateKeyTEAPPhase) {
	case teapPhase2Identity:
		return h.handlePhase2Identity(ctx, state, p2, tlvs)
	case teapPhase2InnerTLS:
		return h.handlePhase2InnerTLS(ctx, state, p2, tlvs)
	case teapPhase2WaitBinding:
		return h.handleBindingResponse(ctx, state, p2, tlvs, false)
	case teapPhase2WaitPassword:
		return h.handlePasswordResponse(ctx, state, p2, tlvs)
	case teapPhase2WaitResult:
		return h.handleBindingResponse(ctx, state, p2, tlvs, true)
	case teapPhase2WaitFailure:
		return h.handleFailureResult(ctx, state, p2, tlvs)
	default:
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: unknown Phase 2 sub-state", eap.ErrTEAPProtocol))
	}
}

// handlePhase2Identity consumes the peer's answer to the Identity exchange.
func (h *TEAPHandler) handlePhase2Identity(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlvs []teaptlv.TLV) (bool, error) {
	// §4.2.3 asks the peer to echo the Identity-Type; it SHOULD answer, so an
	// absent TLV is tolerated but an answer with another type is not.
	if identityType := teaptlv.Find(tlvs, teaptlv.TypeIdentityType); identityType != nil {
		identity, err := identityType.IdentityType()
		if err != nil {
			return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
		}
		if identity != teaptlv.IdentityUser {
			return h.failPhase2(ctx, state, teapErrUnexpectedTLVs,
				fmt.Errorf("%w: peer answered Identity-Type %d, the server asked for a user", eap.ErrTEAPInnerProtocol, identity))
		}
	}

	payload := teaptlv.Find(tlvs, teaptlv.TypeEAPPayload)
	if payload == nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: no EAP-Payload in the identity response", eap.ErrTEAPInnerProtocol))
	}
	packet, _, err := payload.EAPPayload()
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}
	msg, err := parseTEAPInnerEAP(packet)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}
	if msg.Code != eap.CodeResponse || msg.Type != eap.TypeIdentity {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs,
			fmt.Errorf("%w: expected an inner EAP-Response/Identity", eap.ErrTEAPInnerProtocol))
	}
	p2.innerIdentity = string(msg.Data)

	// EAP-TLS Start: an EAP-Request/EAP-TLS with the S flag and no TLS data
	// (RFC 5216 §3.1), carried in an EAP-Payload TLV.
	p2.innerID++
	start := teapInnerRequest(p2.innerID, eap.TypeTLS, (&tlsfragment.Packet{Flags: tlsfragment.FlagStart}).Encode())
	startPayload, err := teaptlv.NewEAPPayload(start)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}

	setString(state, stateKeyTEAPPhase, teapPhase2InnerTLS)
	return false, h.sendPhase2TLVs(ctx, state, startPayload)
}

// handlePhase2InnerTLS reassembles one inner EAP-TLS fragment flight, drives the
// inner handshake and reports its outcome.
func (h *TEAPHandler) handlePhase2InnerTLS(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlvs []teaptlv.TLV) (bool, error) {
	payload := teaptlv.Find(tlvs, teaptlv.TypeEAPPayload)
	if payload == nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: no EAP-Payload in the inner TLS message", eap.ErrTEAPInnerProtocol))
	}
	packet, _, err := payload.EAPPayload()
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}
	msg, err := parseTEAPInnerEAP(packet)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}
	if msg.Code != eap.CodeResponse || msg.Type != eap.TypeTLS {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs,
			fmt.Errorf("%w: expected an inner EAP-Response/EAP-TLS", eap.ErrTEAPInnerProtocol))
	}
	frag, err := tlsfragment.Parse(msg.Data)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}

	reassembler := loadInnerReassembler(state)
	complete, err := reassembler.Accept(frag)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	if !complete {
		// RFC 5216 §2.1.5: a fragment is acknowledged with an EAP-Request/EAP-TLS
		// that carries no TLS data.
		saveInnerReassembler(state, reassembler)
		p2.innerID++
		ack := teapInnerRequest(p2.innerID, eap.TypeTLS, (&tlsfragment.Packet{}).Encode())
		ackPayload, aerr := teaptlv.NewEAPPayload(ack)
		if aerr != nil {
			return h.failPhase2(ctx, state, teapErrInnerMethod, aerr)
		}
		return false, h.sendPhase2TLVs(ctx, state, ackPayload)
	}

	records := append([]byte(nil), reassembler.Buffer()...)
	resetInnerReassembler(state)

	engine, err := h.innerEngine(p2)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	out, done, hsErr := engine.Process(records)
	if hsErr != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("%w: %v", eap.ErrTLSHandshakeFailed, hsErr))
	}
	if !done {
		return h.sendInnerFlight(ctx, state, p2, out)
	}
	return h.completeInnerMethod(ctx, state, p2, out)
}

// sendInnerFlight wraps one server flight of the inner handshake in an
// EAP-Payload TLV. The flight is sent unfragmented at the EAP-TLS level; the
// outer envelope fragments the records if they exceed one TEAP packet.
func (h *TEAPHandler) sendInnerFlight(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlsData []byte) (bool, error) {
	if len(tlsData) == 0 {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("%w: inner TLS flight is empty", eap.ErrTEAPInnerProtocol))
	}
	p2.innerID++
	frag := &tlsfragment.Packet{
		HasLength:        true,
		TLSMessageLength: uint32(len(tlsData)), //nolint:gosec // G115: EAP packets are far below 4 GiB
		Data:             tlsData,
	}
	packet := teapInnerRequest(p2.innerID, eap.TypeTLS, frag.Encode())
	payload, err := teaptlv.NewEAPPayload(packet)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	return false, h.sendPhase2TLVs(ctx, state, payload)
}

// completeInnerMethod runs once the inner EAP-TLS handshake finishes: it binds
// the proven certificate identity, exports the method's MSK and EMSK
// (RFC 9190 §2.3), folds them into the chain and starts the Crypto-Binding
// exchange.
func (h *TEAPHandler) completeInnerMethod(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, out []byte) (bool, error) {
	engine := p2.inner
	if engine == nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("%w: inner TLS engine is missing", eap.ErrTEAPInnerProtocol))
	}
	if version := engine.NegotiatedVersion(); version != tls.VersionTLS13 {
		return h.failPhase2(ctx, state, teapErrInnerMethod,
			fmt.Errorf("%w: inner TLS negotiated version %#04x", eap.ErrTEAPInnerProtocol, version))
	}

	identity, err := engine.Identity()
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("%w: %v", eap.ErrTLSNoIdentity, err))
	}
	if identity.Name == "" {
		return h.failPhase2(ctx, state, teapErrInnerMethod, eap.ErrTLSNoIdentity)
	}
	// RFC 5216 §5.2: the identity the peer claimed in EAP must be the one its
	// certificate proves.
	if p2.innerIdentity != "" && !identity.Matches(p2.innerIdentity) {
		return h.failPhase2(ctx, state, teapErrInnerMethod,
			fmt.Errorf("%w: certificate identity %q does not match the inner identity %q",
				eap.ErrTLSIdentityMismatch, identity.Name, p2.innerIdentity))
	}
	p2.certIdentity = identity

	keyMaterial, err := engine.ExportKey(eapTLS13KeyLabel, []byte{eap.TypeTLS}, eapTLS13KeyLength)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("TEAP inner EAP-TLS exporter: %w", err))
	}
	if len(keyMaterial) != eapTLS13KeyLength {
		return h.failPhase2(ctx, state, teapErrInnerMethod,
			fmt.Errorf("%w: inner EAP-TLS exporter returned %d octets, §2.3 fixes it at %d",
				eap.ErrTEAPInnerProtocol, len(keyMaterial), eapTLS13KeyLength))
	}

	keys, err := teapinner.KeysFor(p2.method, keyMaterial[:64], keyMaterial[64:])
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	outer, err := h.outerEngine(state)
	if err != nil {
		return false, err
	}
	if err := p2.chain.Advance(outer, keys); err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}

	p2.closeInner()
	if err := bindInnerIdentity(state, identity.Name); err != nil {
		return h.failPhase2(ctx, state, teapErrAuthFailure, err)
	}
	return h.finishMethod(ctx, state, p2, out)
}

// finishMethod sends the Crypto-Binding exchange that closes the method that
// just succeeded. When it was the last method in the plan the Result TLV rides
// along, which §3.6.6 allows the server to do to announce the end of the
// conversation; otherwise the peer's answer starts the next method.
func (h *TEAPHandler) finishMethod(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, extra []byte) (bool, error) {
	final := p2.idx >= len(p2.methods)-1
	return false, h.sendBinding(ctx, state, p2, extra, final)
}

// sendBinding sends the Intermediate-Result TLV with a Crypto-Binding request
// computed over the §6.3 BUFFER.
func (h *TEAPHandler) sendBinding(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, extra []byte, final bool) error {
	mskCMK, emskCMK := p2.chain.CompoundMACKeys()
	flags := teaptlv.FlagMSKPresent
	// §6.2.5: a method that derives no EMSK leaves the EMSK Compound MAC out, even
	// though the chain still carries an EMSK from an earlier method.
	if p2.method.DerivesEMSK && len(emskCMK) > 0 {
		flags |= teaptlv.FlagEMSKPresent
	}
	nonce, err := newTEAPNonce()
	if err != nil {
		return err
	}
	binding, err := p2.exchange.Bind(teapbind.Keys{MSKCMK: mskCMK, EMSKCMK: emskCMK}, flags, teaptlv.SubTypeBindingRequest, nonce)
	if err != nil {
		return err
	}
	intermediate, err := teaptlv.NewIntermediateResult(teaptlv.StatusSuccess)
	if err != nil {
		return err
	}

	tlvs := make([]teaptlv.TLV, 0, 4)
	if len(extra) > 0 {
		frag := &tlsfragment.Packet{
			HasLength:        true,
			TLSMessageLength: uint32(len(extra)), //nolint:gosec // G115: EAP packets are far below 4 GiB
			Data:             extra,
		}
		packet := teapInnerRequest(p2.innerID+1, eap.TypeTLS, frag.Encode())
		p2.innerID++
		payload, perr := teaptlv.NewEAPPayload(packet)
		if perr != nil {
			return perr
		}
		tlvs = append(tlvs, payload)
	}
	tlvs = append(tlvs, intermediate, binding)
	if final {
		tlvs = append(tlvs, teaptlv.NewResult(teaptlv.StatusSuccess))
		setString(state, stateKeyTEAPPhase, teapPhase2WaitResult)
	} else {
		setString(state, stateKeyTEAPPhase, teapPhase2WaitBinding)
	}
	p2.nonce = nonce
	return h.sendPhase2TLVs(ctx, state, tlvs...)
}

// handleBindingResponse validates the peer's Crypto-Binding TLV, records the
// method's outcome and moves on to the next method or to the grant.
//
// final marks the protected termination exchange, whose message also carries the
// peer's Result TLV (§3.6.6).
func (h *TEAPHandler) handleBindingResponse(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlvs []teaptlv.TLV, final bool) (bool, error) {
	bindingTLV := teaptlv.Find(tlvs, teaptlv.TypeCryptoBinding)
	if bindingTLV == nil {
		return h.failPhase2(ctx, state, teapErrInvalidBinding, fmt.Errorf("%w: no Crypto-Binding TLV", eap.ErrTEAPCryptoBinding))
	}
	binding, err := bindingTLV.CryptoBinding()
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInvalidBinding, fmt.Errorf("%w: %v", eap.ErrTEAPCryptoBinding, err))
	}
	if binding.SubType != teaptlv.SubTypeBindingResponse {
		return h.failPhase2(ctx, state, teapErrInvalidBinding,
			fmt.Errorf("%w: Sub-Type %d is not a Binding Response", eap.ErrTEAPCryptoBinding, binding.SubType))
	}
	wantNonce, err := teaptlv.ResponseNonce(p2.nonce)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInvalidBinding, err)
	}
	if !bytes.Equal(binding.Nonce, wantNonce) {
		return h.failPhase2(ctx, state, teapErrInvalidBinding, fmt.Errorf("%w: response Nonce does not match the request", eap.ErrTEAPCryptoBinding))
	}

	mskCMK, emskCMK := p2.chain.CompoundMACKeys()
	buffer, err := p2.exchange.Buffer(*bindingTLV)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInvalidBinding, err)
	}
	if binding.Flags&teaptlv.FlagMSKPresent == 0 {
		return h.failPhase2(ctx, state, teapErrMissingMSKCM, fmt.Errorf("%w: §6.2.4 requires the MSK Compound MAC", eap.ErrTEAPCryptoBinding))
	}
	if err := verifyTEAPCompoundMAC(p2, mskCMK, buffer, binding.MSKCompoundMAC); err != nil {
		return h.failPhase2(ctx, state, teapErrBadMSKCM, fmt.Errorf("%w: %v", eap.ErrTEAPCryptoBinding, err))
	}
	if p2.method.DerivesEMSK {
		if binding.Flags&teaptlv.FlagEMSKPresent == 0 {
			return h.failPhase2(ctx, state, teapErrMissingEMSKCM,
				fmt.Errorf("%w: %q derived an EMSK and the peer sent no EMSK Compound MAC", eap.ErrTEAPCryptoBinding, p2.method.Name))
		}
		if err := verifyTEAPCompoundMAC(p2, emskCMK, buffer, binding.EMSKCompoundMAC); err != nil {
			return h.failPhase2(ctx, state, teapErrBadEMSKCM, err)
		}
	} else if binding.Flags&teaptlv.FlagEMSKPresent != 0 {
		return h.failPhase2(ctx, state, teapErrUnexpectedEMSKCM,
			fmt.Errorf("%w: %q derives no EMSK but the peer sent an EMSK Compound MAC", eap.ErrTEAPCryptoBinding, p2.method.Name))
	}

	peerStatus := teaptlv.StatusSuccess
	if intermediate := teaptlv.Find(tlvs, teaptlv.TypeIntermediateResult); intermediate != nil {
		status, _, ierr := intermediate.IntermediateResult()
		if ierr != nil {
			return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, ierr)
		}
		peerStatus = status
	} else if !final {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs,
			fmt.Errorf("%w: no Intermediate-Result TLV for %q", eap.ErrTEAPInnerProtocol, p2.method.Name))
	}

	if final {
		result := teaptlv.Find(tlvs, teaptlv.TypeResult)
		if result == nil {
			return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: no Result TLV in the termination exchange", eap.ErrTEAPInnerProtocol))
		}
		status, rerr := result.Status()
		if rerr != nil {
			return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, rerr)
		}
		if status != teaptlv.StatusSuccess {
			return h.failPhase2(ctx, state, teapErrAuthFailure, eap.ErrTEAPPeerFailure)
		}
	}

	if peerStatus != teaptlv.StatusSuccess {
		return h.failPhase2(ctx, state, teapErrAuthFailure,
			fmt.Errorf("%w: peer reported failure for %q", eap.ErrTEAPPeerFailure, p2.method.Name))
	}
	if err := p2.session.Record(teaptlv.StatusSuccess, peerStatus, binding.Flags); err != nil {
		return h.failPhase2(ctx, state, teapBindingErrorCode(err), err)
	}

	if final {
		return h.finishPhase2(ctx, state, p2)
	}

	p2.idx++
	next, ok := p2.session.Next()
	if !ok {
		return h.failPhase2(ctx, state, teapErrInnerMethod, fmt.Errorf("%w: no Inner Method to run", eap.ErrTEAPPhase2NotConfigured))
	}
	p2.method = next
	p2.password = nil
	return h.beginMethod(ctx, state, p2)
}

// handlePasswordResponse verifies the Basic-Password-Auth-Resp TLV of §3.6.3.
func (h *TEAPHandler) handlePasswordResponse(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlvs []teaptlv.TLV) (bool, error) {
	// §3.6.3: a peer that does not want to run password authentication answers
	// with a NAK, which is a refusal rather than a protocol error.
	if nak := teaptlv.Find(tlvs, teaptlv.TypeNAK); nak != nil {
		return h.failPhase2(ctx, state, teapErrInnerNotSupported, fmt.Errorf("%w: peer refused Basic-Password-Auth", eap.ErrTEAPPassword))
	}
	response := teaptlv.Find(tlvs, teaptlv.TypeBasicPasswordAuthResp)
	if response == nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, fmt.Errorf("%w: no Basic-Password-Auth-Resp TLV", eap.ErrTEAPPassword))
	}
	username, password, err := response.Credentials()
	if err != nil {
		return h.failPhase2(ctx, state, teapErrUnexpectedTLVs, err)
	}

	if err := h.verifyTEAPPassword(ctx, username, password); err != nil {
		// §4.2.6 warns that error 1031 (credentials incorrect) lets an attacker
		// enumerate usernames, so a mismatch reports the general failure.
		return h.failPhase2(ctx, state, teapErrAuthFailure, fmt.Errorf("%w: %v", eap.ErrTEAPPassword, err))
	}
	if err := bindInnerIdentity(state, string(username)); err != nil {
		return h.failPhase2(ctx, state, teapErrAuthFailure, err)
	}
	p2.passwordUser = string(username)

	outer, err := h.outerEngine(state)
	if err != nil {
		return false, err
	}
	keys, err := teapinner.KeysFor(p2.method, nil, nil)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	if err := p2.chain.Advance(outer, keys); err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	return h.finishMethod(ctx, state, p2, nil)
}

// verifyTEAPPassword checks the cleartext password of a Basic-Password-Auth
// exchange: against an external directory by binding when a CredentialVerifier
// is active, otherwise in constant time against the stored password.
func (h *TEAPHandler) verifyTEAPPassword(ctx *eap.EAPContext, username, password []byte) error {
	if ctx.Verifier != nil && ctx.Verifier.Active() {
		bindCtx := ctx.Context
		if bindCtx == nil {
			bindCtx = context.Background()
		}
		return ctx.Verifier.VerifyCleartext(bindCtx, string(username), string(password))
	}
	if ctx.PwdProvider == nil {
		return fmt.Errorf("%w: no password provider is configured", eap.ErrTEAPPhase2NotConfigured)
	}
	if ctx.User == nil {
		return fmt.Errorf("%w: no user record for the password exchange", eap.ErrTEAPPassword)
	}
	expected, err := ctx.PwdProvider.GetPassword(ctx.User, ctx.IsMacAuth)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(password, []byte(expected)) != 1 {
		return eap.ErrPasswordMismatch
	}
	return nil
}

// finishPhase2 derives the outer MSK and EMSK of §6.4 from the chain, adds the
// MS-MPPE keys to the Access-Accept and grants the session.
func (h *TEAPHandler) finishPhase2(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session) (bool, error) {
	outer, err := h.outerEngine(state)
	if err != nil {
		return false, err
	}
	msk, _, err := p2.chain.SessionKeys(outer, p2.method.DerivesEMSK)
	if err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}
	if err := addTEAPMPPEKeys(ctx.Response, msk[:64]); err != nil {
		return h.failPhase2(ctx, state, teapErrInnerMethod, err)
	}

	// §3.11: the credentials Phase 2 proved replace the Phase 1 identity, and they
	// are associated with the ticket so a later resumption authorizes the same
	// principal.
	ticketHolder(state).associate(state.Username, msk[:64])

	state.Success = true
	p2.closeInner()
	closeEngine(state)
	clearKey(state, stateKeyTEAPSession)
	clearKey(state, stateKeyTEAPPhase)
	clearKey(state, stateKeyTEAPPhase2)
	if err := ctx.StateManager.SetState(state.StateID, state); err != nil {
		return false, err
	}
	return true, nil
}

// failPhase2 reports a fatal Phase 2 error the way §3.9.3 requires: a Result TLV
// indicating failure accompanied by an Error TLV, then wait for the peer's
// failure Result before the cleartext EAP-Failure is sent.
func (h *TEAPHandler) failPhase2(ctx *eap.EAPContext, state *eap.EAPState, code uint32, cause error) (bool, error) {
	p2 := phase2Session(state)
	if p2 == nil {
		return false, cause
	}
	if p2.failure == nil {
		p2.failure = cause
	}
	setString(state, stateKeyTEAPPhase, teapPhase2WaitFailure)

	if err := h.sendPhase2TLVs(ctx, state, teaptlv.NewResult(teaptlv.StatusFailure), teapErrorTLV(code)); err != nil {
		p2.closeInner()
		closeEngine(state)
		return false, cause
	}
	return false, nil
}

// handleFailureResult consumes the peer's answer to a failure Result TLV. The
// peer MUST answer with a failure Result; whatever it sends, the exchange is over
// and the failure the server recorded is reported to the coordinator.
func (h *TEAPHandler) handleFailureResult(ctx *eap.EAPContext, state *eap.EAPState, p2 *teapPhase2Session, tlvs []teaptlv.TLV) (bool, error) {
	if result := teaptlv.Find(tlvs, teaptlv.TypeResult); result != nil {
		if status, err := result.Status(); err == nil && status == teaptlv.StatusSuccess {
			if p2.failure == nil {
				p2.failure = fmt.Errorf("%w: peer acknowledged a failure with a success Result", eap.ErrTEAPInnerProtocol)
			}
		}
	}
	failure := p2.failure
	if failure == nil {
		failure = eap.ErrTEAPPeerFailure
	}
	p2.closeInner()
	closeEngine(state)
	clearKey(state, stateKeyTEAPSession)
	clearKey(state, stateKeyTEAPPhase)
	clearKey(state, stateKeyTEAPPhase2)
	return false, failure
}

// grantResumed completes a resumed TEAP session. RFC 9930 §3.5 bypasses Phase 2
// entirely, so the grant comes from the credentials cached with the ticket
// (RFC 9190 §5.7).
func (h *TEAPHandler) grantResumed(ctx *eap.EAPContext, state *eap.EAPState) (bool, error) {
	holder := ticketHolder(state)
	creds := holder.resumedCredentials()
	if creds == nil || creds.username == "" || len(creds.msk) != 64 {
		closeEngine(state)
		return false, fmt.Errorf("%w: resumed session has no cached credentials", eap.ErrTEAPProtocol)
	}
	if state.Username != "" && !eap.IsAnonymousNAI(state.Username) && !strings.EqualFold(state.Username, creds.username) {
		closeEngine(state)
		return false, fmt.Errorf("%w: ticket identity %q does not match user %q",
			eap.ErrTLSIdentityMismatch, creds.username, state.Username)
	}

	state.Username = creds.username
	if err := addTEAPMPPEKeys(ctx.Response, creds.msk); err != nil {
		closeEngine(state)
		return false, err
	}
	// §3.11: the ticket this resumption issues carries the same credentials.
	holder.associate(creds.username, creds.msk)
	state.Success = true
	closeEngine(state)
	if err := ctx.StateManager.SetState(state.StateID, state); err != nil {
		return false, err
	}
	return true, nil
}

// runPending executes the server step that waited for the peer to acknowledge
// the last queued flight. A TEAP packet carries one message per direction, so a
// flight the peer has not yet seen cannot carry the next request. A nil packet
// means there was no flight to acknowledge and the step runs immediately.
func (h *TEAPHandler) runPending(ctx *eap.EAPContext, state *eap.EAPState, pending string, pkt *teapmsgPacket) (bool, error) {
	clearKey(state, stateKeyTEAPPending)
	if pkt != nil && len(pkt.TLSData) > 0 {
		return false, fmt.Errorf("%w: data arrived before %s", eap.ErrTEAPUnexpectedFragment, pending)
	}
	switch pending {
	case teapPendingPhase2:
		return h.startPhase2(ctx, state)
	case teapPendingGrant:
		return h.grantResumed(ctx, state)
	default:
		return false, fmt.Errorf("%w: unknown pending step %q", eap.ErrTEAPProtocol, pending)
	}
}

// sendPhase2TLVs marshals the TLVs, encrypts them on the outer tunnel and sends
// the resulting TLS records as a TEAP flight.
func (h *TEAPHandler) sendPhase2TLVs(ctx *eap.EAPContext, state *eap.EAPState, tlvs ...teaptlv.TLV) error {
	payload, err := teaptlv.Marshal(tlvs...)
	if err != nil {
		return err
	}
	engine, err := h.outerEngine(state)
	if err != nil {
		return err
	}
	records, err := engine.WriteApplication(payload)
	if err != nil {
		return err
	}
	return h.sendTEAPFlight(ctx, state, records)
}

// --- small helpers ---------------------------------------------------------

// teapmsgPacket aliases the envelope packet the Phase 2 state machine inspects:
// the TLS Data is already decrypted, so only the Outer TLVs matter here.
type teapmsgPacket = teapmsg.Packet

// bindInnerIdentity binds the identity an Inner Method proved to the session.
// An anonymous outer NAI names only the realm, so the proven identity replaces it;
// otherwise the two must agree (RFC 9930 §3.11: authorization runs on the Phase 2
// credentials, not on the Phase 1 identity).
func bindInnerIdentity(state *eap.EAPState, identity string) error {
	identity = strings.TrimSpace(identity)
	if identity == "" {
		return eap.ErrTLSNoIdentity
	}
	if state.Username == "" || eap.IsAnonymousNAI(state.Username) {
		state.Username = identity
		return nil
	}
	if !strings.EqualFold(state.Username, identity) {
		return fmt.Errorf("%w: authenticated identity %q does not match the outer identity %q",
			eap.ErrTLSIdentityMismatch, identity, state.Username)
	}
	return nil
}

// newTEAPNonce returns a Crypto-Binding Nonce whose least significant bit is
// clear, the shape §4.2.13 fixes for a Binding Request.
func newTEAPNonce() ([]byte, error) {
	nonce := make([]byte, teapPhase2NonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("TEAP Crypto-Binding nonce: %w", err)
	}
	nonce[len(nonce)-1] &^= 0x01
	return nonce, nil
}

// verifyTEAPCompoundMAC recomputes one Compound MAC field and compares it in
// constant time (§6.3).
func verifyTEAPCompoundMAC(p2 *teapPhase2Session, cmk, buffer, received []byte) error {
	want, err := teapbind.CompoundMAC(p2.exchange.Hash, cmk, buffer)
	if err != nil {
		return err
	}
	if !hmac.Equal(want, received) {
		return fmt.Errorf("%w: computed % x, received % x", teapbind.ErrCompoundMACMismatch, want, received)
	}
	return nil
}

// teapBindingErrorCode maps a teapinner binding failure to the Error-Code §4.2.6
// defines for it.
func teapBindingErrorCode(err error) uint32 {
	switch {
	case errors.Is(err, teapinner.ErrNoEMSKBinding):
		return teapErrMissingEMSKCM
	case errors.Is(err, teapinner.ErrNoBinding):
		return teapErrMissingMSKCM
	default:
		return teapErrInvalidBinding
	}
}

// teapErrorTLV builds the Error TLV of §4.2.6: a mandatory TLV with a four-octet
// Error-Code.
func teapErrorTLV(code uint32) teaptlv.TLV {
	value := make([]byte, 4)
	binary.BigEndian.PutUint32(value, code)
	return teaptlv.TLV{Type: teaptlv.TypeError, Mandatory: true, Value: value}
}

// teapInnerRequest encodes an inner EAP-Request packet.
func teapInnerRequest(identifier, typ uint8, data []byte) []byte {
	return (&eap.EAPMessage{
		Code:       eap.CodeRequest,
		Identifier: identifier,
		Type:       typ,
		Data:       data,
	}).Encode()
}

// parseTEAPInnerEAP parses a complete inner EAP packet. Unlike the PEAP helper it
// accepts no compressed or padded framing: §3.6.2 carries a full EAP state
// machine, so the packet has to be one.
func parseTEAPInnerEAP(packet []byte) (*eap.EAPMessage, error) {
	if len(packet) < 5 {
		return nil, fmt.Errorf("%w: inner EAP packet is %d octets", eap.ErrInvalidEAPMessage, len(packet))
	}
	declared := int(binary.BigEndian.Uint16(packet[2:4]))
	if declared != len(packet) {
		return nil, fmt.Errorf("%w: inner EAP length %d != %d", eap.ErrInvalidEAPMessage, declared, len(packet))
	}
	return &eap.EAPMessage{
		Code:       packet[0],
		Identifier: packet[1],
		Length:     binary.BigEndian.Uint16(packet[2:4]),
		Type:       packet[4],
		Data:       packet[5:],
	}, nil
}

// addTEAPMPPEKeys adds the MS-MPPE-Recv-Key, MS-MPPE-Send-Key and encryption
// policy attributes derived from a 64-octet TEAP MSK (RFC 2548). Without them an
// authenticator cannot derive the PMK, so a rejection is reported rather than a
// keyless grant.
func addTEAPMPPEKeys(response *radius.Packet, msk []byte) error {
	if len(msk) != 64 {
		return fmt.Errorf("%w: TEAP MSK is %d octets, §6.4 derives 64", eap.ErrTEAPProtocol, len(msk))
	}
	_ = microsoft.MSMPPERecvKey_Add(response, msk[:32]) //nolint:errcheck
	_ = microsoft.MSMPPESendKey_Add(response, msk[32:64]) //nolint:errcheck
	_ = microsoft.MSMPPEEncryptionPolicy_Add(response, //nolint:errcheck
		microsoft.MSMPPEEncryptionPolicy_Value_EncryptionAllowed)
	_ = microsoft.MSMPPEEncryptionTypes_Add(response, //nolint:errcheck
		microsoft.MSMPPEEncryptionTypes_Value_RC440or128BitAllowed)
	return nil
}

// loadInnerReassembler restores the inner EAP-TLS reassembly state.
func loadInnerReassembler(state *eap.EAPState) *tlsfragment.Reassembler {
	declared, _ := state.Data[stateKeyTEAPInnerRxDeclared].(uint32)
	hasLen, _ := state.Data[stateKeyTEAPInnerRxHasLen].(bool)
	return tlsfragment.LoadReassembler(getBytes(state, stateKeyTEAPInnerRxBuf), declared, hasLen, tlsfragment.DefaultMaxMessageLength)
}

// saveInnerReassembler persists the inner EAP-TLS reassembly progress.
func saveInnerReassembler(state *eap.EAPState, r *tlsfragment.Reassembler) {
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	declared, hasLen := r.Declared()
	state.Data[stateKeyTEAPInnerRxBuf] = r.Buffer()
	state.Data[stateKeyTEAPInnerRxDeclared] = declared
	state.Data[stateKeyTEAPInnerRxHasLen] = hasLen
}

// resetInnerReassembler clears the inner EAP-TLS reassembly state.
func resetInnerReassembler(state *eap.EAPState) {
	clearKey(state, stateKeyTEAPInnerRxBuf)
	clearKey(state, stateKeyTEAPInnerRxDeclared)
	clearKey(state, stateKeyTEAPInnerRxHasLen)
}
