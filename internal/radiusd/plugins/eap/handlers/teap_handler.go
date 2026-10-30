package handlers

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapmsg"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
	"github.com/bjo163/mwx-isp/pkg/common"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
)

// EAPMethodTEAP is the configuration / handler name for RFC 9930 TEAP.
const EAPMethodTEAP = "eap-teap"

// State keys a TEAP session keeps across EAP rounds.
const (
	// stateKeyTEAPVersion is the uint8 version the peer answered with. Zero means
	// no peer message has been accepted yet: RFC 9930 §3.1 defines no version 0,
	// so the value doubles as "negotiation pending".
	stateKeyTEAPVersion = "teap_version"
	// stateKeyTEAPPeerOuter holds the Outer TLVs of the peer's initial message,
	// which RFC 9930 §6.3 appends to the Crypto-Binding BUFFER.
	stateKeyTEAPPeerOuter = "teap_peer_outer"
	// stateKeyTEAPOutFrags queues []*teapmsg.Packet still to be sent.
	stateKeyTEAPOutFrags = "teap_out_frags"
	// Inbound reassembly, persisted between rounds like the EAP-TLS tunnel does.
	stateKeyTEAPRxBuf      = "teap_rx_buf"
	stateKeyTEAPRxDeclared = "teap_rx_declared"
	stateKeyTEAPRxHasLen   = "teap_rx_has_len"
)

// TEAPHandler is the EAP-TEAP (RFC 9930) authentication handler.
//
// TEAP builds a server-only TLS tunnel (Phase 1) and then runs one or more Inner
// Methods inside it (Phase 2), binding the tunnel to the Inner Method keys with
// the Crypto-Binding TLV. Its outer framing is TEAP's own envelope — a Flags/Ver
// octet plus optional Message Length and Outer TLV Length fields (§4.1), carried
// by teapmsg — rather than the EAP-TLS layout PEAP and EAP-TTLS reuse.
//
// Milestone scope:
//   - M11.8 (current): registered the method (EAP type 55, name "eap-teap"),
//     answers EAP-Response/Identity with a TEAP Start carrying the Authority-ID
//     TLV, negotiates the version (§3.1), captures the peer's Phase 1 Outer TLVs
//     and drives the TLS 1.3 handshake over the TEAP envelope. Once the tunnel is
//     up it rejects with eap.ErrTEAPPhase2NotImplemented, so it can never grant
//     access without the Phase 2 Result TLV.
//   - M11.9: the Phase 2 TLV exchange (Inner Methods, Intermediate-Result,
//     Crypto-Binding, Result) and the MSK/EMSK derivation.
type TEAPHandler struct {
	configProvider TLSConfigProvider
	maxFragment    int
}

// NewTEAPHandler creates a TEAP handler without TLS material configured. It
// answers the Start exchange but rejects handshake attempts safely with
// eap.ErrTLSNotConfigured until a config provider is supplied.
func NewTEAPHandler() *TEAPHandler {
	return &TEAPHandler{maxFragment: defaultMaxTLSFragment}
}

// NewTEAPHandlerWithConfig creates a TEAP handler that drives the outer TLS
// tunnel using the TLS material returned by provider.
func NewTEAPHandlerWithConfig(provider TLSConfigProvider) *TEAPHandler {
	return &TEAPHandler{configProvider: provider, maxFragment: defaultMaxTLSFragment}
}

// Name returns the handler name ("eap-teap").
func (h *TEAPHandler) Name() string {
	return EAPMethodTEAP
}

// EAPType returns the EAP method type code handled (55, TEAP).
func (h *TEAPHandler) EAPType() uint8 {
	return eap.TypeTEAP
}

// CanHandle reports whether this handler can process the EAP message.
func (h *TEAPHandler) CanHandle(ctx *eap.EAPContext) bool {
	if ctx == nil || ctx.EAPMessage == nil {
		return false
	}
	return ctx.EAPMessage.Type == eap.TypeTEAP
}

// HandleIdentity handles EAP-Response/Identity by sending a TEAP Start: an
// EAP-Request of type TEAP whose body is a Flags/Ver octet with the Start bit
// set, no TLS data and an Authority-ID TLV in the Outer TLV field (RFC 9930
// §3.2, framing per §4.1).
func (h *TEAPHandler) HandleIdentity(ctx *eap.EAPContext) (bool, error) {
	username := rfc2865.UserName_GetString(ctx.Request.Packet)
	eapData, err := h.buildStartRequest(ctx.EAPMessage.Identifier, teapAuthorityID(username))
	if err != nil {
		return false, err
	}

	stateID := common.UUID()
	state := &eap.EAPState{
		Username: username,
		StateID:  stateID,
		Method:   EAPMethodTEAP,
		Success:  false,
	}

	if err := ctx.StateManager.SetState(stateID, state); err != nil {
		return false, err
	}

	return true, h.writeChallenge(ctx, stateID, eapData)
}

// HandleResponse drives the TEAP Phase 1 tunnel: version negotiation (§3.1),
// Outer TLV capture, fragment reassembly over the TEAP envelope (§4.1) and the
// TLS handshake. Once the tunnel is up it rejects, because Phase 2 (§3.6) is
// not implemented yet; see eap.ErrTEAPPhase2NotImplemented.
func (h *TEAPHandler) HandleResponse(ctx *eap.EAPContext) (bool, error) {
	stateID := rfc2865.State_GetString(ctx.Request.Packet)
	if stateID == "" {
		return false, eap.ErrStateNotFound
	}

	state, err := ctx.StateManager.GetState(stateID)
	if err != nil {
		return false, err
	}

	pkt, err := teapmsg.Parse(ctx.EAPMessage.Data)
	if err != nil {
		closeEngine(state)
		return false, err
	}

	// RFC 9930 §3.2: only the server sets S, in its Start. A peer that echoes it
	// is answering a message the server never sent.
	if pkt.Start() {
		closeEngine(state)
		return false, fmt.Errorf("%w: peer set the Start bit", eap.ErrTEAPProtocol)
	}

	// §3.1: the peer answers the Start with the version it will use. This server
	// implements version 1 only, so §3.1 step 5 lets it terminate the exchange on
	// any other value rather than guess at a lower one.
	negotiated := getUint8(state, stateKeyTEAPVersion)
	firstMessage := negotiated == 0
	if firstMessage {
		if pkt.Version != teapmsg.Version {
			closeEngine(state)
			return false, fmt.Errorf("%w: peer proposed version %d, this server implements %d",
				eap.ErrTEAPVersionUnsupported, pkt.Version, teapmsg.Version)
		}
		negotiated = pkt.Version
		setUint8(state, stateKeyTEAPVersion, negotiated)
	} else if pkt.Version != negotiated {
		closeEngine(state)
		return false, fmt.Errorf("%w: version changed from %d to %d mid-conversation",
			eap.ErrTEAPProtocol, negotiated, pkt.Version)
	}

	// §4.1 allows the Outer TLVs only in the initial request and response, so they
	// arrive with the peer's first message and nowhere else. §6.3 step 4 appends
	// them to the Crypto-Binding BUFFER, so they are kept verbatim.
	if pkt.HasOuterTLVs {
		if !firstMessage {
			closeEngine(state)
			return false, fmt.Errorf("%w: Outer TLVs outside the initial message", eap.ErrTEAPProtocol)
		}
		setBytes(state, stateKeyTEAPPeerOuter, pkt.OuterTLVs)
		// The peer's initial message may complete the ClientHello in one packet,
		// so the Outer TLVs must survive even when no later round persists them.
		if serr := ctx.StateManager.SetState(stateID, state); serr != nil {
			closeEngine(state)
			return false, serr
		}
	}

	if len(getTEAPFragments(state)) > 0 {
		return h.sendNextFragment(ctx, state, pkt)
	}

	reassembler := loadTEAPReassembler(state)
	complete, err := reassembler.Accept(pkt)
	if err != nil {
		closeEngine(state)
		return false, err
	}
	if !complete {
		saveTEAPReassembler(state, reassembler)
		if serr := ctx.StateManager.SetState(stateID, state); serr != nil {
			closeEngine(state)
			return false, serr
		}
		ack, aerr := teapmsgACK(ctx.EAPMessage.Identifier + 1)
		if aerr != nil {
			closeEngine(state)
			return false, aerr
		}
		return false, h.writeChallenge(ctx, stateID, ack)
	}

	tlsInput := reassembler.Buffer()
	resetTEAPReassembler(state)
	return h.advanceHandshake(ctx, state, tlsInput)
}

// advanceHandshake feeds one reassembled client flight to the TLS engine and
// sends whatever the engine answers as TEAP fragments. A completed handshake
// ends Phase 1; Phase 2 is not implemented yet, so it rejects.
func (h *TEAPHandler) advanceHandshake(ctx *eap.EAPContext, state *eap.EAPState, tlsInput []byte) (bool, error) {
	engine, err := engineForState(state, h.configProvider)
	if err != nil {
		return false, err
	}

	out, done, hsErr := engine.Process(tlsInput)
	if hsErr != nil {
		closeEngine(state)
		return false, fmt.Errorf("%w: %v", eap.ErrTLSHandshakeFailed, hsErr)
	}

	if done {
		// teapkeys implements the RFC 9427 §2.2 TLS 1.3 exporter schedule only.
		// RFC 9930 §6.2 defines a TLS 1.2 form over the RFC 5246 TLS-PRF, which
		// would derive different keys from the same inputs, so a 1.2 tunnel must
		// not reach Phase 2 on this server.
		if version := engine.NegotiatedVersion(); version != tls.VersionTLS13 {
			closeEngine(state)
			return false, fmt.Errorf("%w: negotiated TLS version %#04x", eap.ErrTEAPRequiresTLS13, version)
		}
		closeEngine(state)
		return false, eap.ErrTEAPPhase2NotImplemented
	}

	if len(out) == 0 {
		closeEngine(state)
		return false, eap.ErrTEAPUnexpectedFragment
	}

	packets, err := teapmsg.Fragment(teapmsg.Version, out, nil, h.maxFragment)
	if err != nil {
		closeEngine(state)
		return false, err
	}

	first, rest := packets[0], packets[1:]
	if len(rest) > 0 {
		setTEAPFragments(state, rest)
	}
	if err := ctx.StateManager.SetState(state.StateID, state); err != nil {
		closeEngine(state)
		return false, err
	}
	eapData, err := h.buildRequest(ctx.EAPMessage.Identifier+1, first)
	if err != nil {
		closeEngine(state)
		return false, err
	}
	return false, h.writeChallenge(ctx, state.StateID, eapData)
}

// sendNextFragment answers a peer fragment acknowledgement with the next queued
// server fragment (RFC 9930 §4.1).
func (h *TEAPHandler) sendNextFragment(ctx *eap.EAPContext, state *eap.EAPState, pkt *teapmsg.Packet) (bool, error) {
	if !pkt.IsACK() {
		closeEngine(state)
		return false, eap.ErrTEAPUnexpectedFragment
	}

	queue := getTEAPFragments(state)
	next := queue[0]
	queue = queue[1:]
	if len(queue) > 0 {
		setTEAPFragments(state, queue)
	} else {
		clearKey(state, stateKeyTEAPOutFrags)
	}

	if err := ctx.StateManager.SetState(state.StateID, state); err != nil {
		closeEngine(state)
		return false, err
	}
	eapData, err := h.buildRequest(ctx.EAPMessage.Identifier+1, next)
	if err != nil {
		closeEngine(state)
		return false, err
	}
	return false, h.writeChallenge(ctx, state.StateID, eapData)
}

// buildStartRequest constructs the TEAP Start: a Start bit, version 1, no TLS
// data and one Authority-ID TLV (§4.2.2) in the Outer TLV field.
func (h *TEAPHandler) buildStartRequest(identifier uint8, authorityID string) ([]byte, error) {
	outer, err := teaptlv.Marshal(teaptlv.NewAuthorityID([]byte(authorityID)))
	if err != nil {
		return nil, err
	}

	pkt := &teapmsg.Packet{
		Flags:        teapmsg.FlagStart,
		Version:      teapmsg.Version,
		HasOuterTLVs: true,
		OuterTLVs:    outer,
	}
	return h.buildRequest(identifier, pkt)
}

// teapAuthorityID derives the Authority-ID hint from the outer identity the peer
// sent: the realm it addressed, or the identity itself when it carries no realm.
// §4.2.2 leaves the value opaque and asks only that it be unique across the
// deployment, and the realm is the part of an NAI that names the server (§3.4
// has the peer validate that realm against the server certificate).
func teapAuthorityID(username string) string {
	if i := strings.LastIndex(username, "@"); i >= 0 && i+1 < len(username) {
		return username[i+1:]
	}
	return username
}

// buildRequest encodes one TEAP packet as an EAP-Request of type TEAP.
func (h *TEAPHandler) buildRequest(identifier uint8, pkt *teapmsg.Packet) ([]byte, error) {
	data, err := pkt.Encode()
	if err != nil {
		return nil, err
	}
	msg := &eap.EAPMessage{
		Code:       eap.CodeRequest,
		Identifier: identifier,
		Type:       eap.TypeTEAP,
		Data:       data,
	}
	return msg.Encode(), nil
}

// teapmsgACK builds the bare acknowledgment §4.1 describes: Flags and Version
// fields with zero-length TLS data.
func teapmsgACK(identifier uint8) ([]byte, error) {
	pkt := &teapmsg.Packet{Version: teapmsg.Version}
	data, err := pkt.Encode()
	if err != nil {
		return nil, err
	}
	msg := &eap.EAPMessage{Code: eap.CodeRequest, Identifier: identifier, Type: eap.TypeTEAP, Data: data}
	return msg.Encode(), nil
}

// writeChallenge sends eapData inside a RADIUS Access-Challenge, echoing the
// handshake State attribute and protecting the response with a
// Message-Authenticator (RFC 3579 §3.2).
func (h *TEAPHandler) writeChallenge(ctx *eap.EAPContext, stateID string, eapData []byte) error {
	response := ctx.Request.Response(radius.CodeAccessChallenge)
	_ = rfc2865.State_SetString(response, stateID) //nolint:errcheck
	eap.SetEAPMessageAndAuth(response, eapData, ctx.Secret)
	return ctx.ResponseWriter.Write(response)
}

// --- TEAP state helpers ---------------------------------------------------

func getTEAPFragments(state *eap.EAPState) []*teapmsg.Packet {
	if state.Data == nil {
		return nil
	}
	if v, ok := state.Data[stateKeyTEAPOutFrags].([]*teapmsg.Packet); ok {
		return v
	}
	return nil
}

func setTEAPFragments(state *eap.EAPState, frags []*teapmsg.Packet) {
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[stateKeyTEAPOutFrags] = frags
}

func getBytes(state *eap.EAPState, key string) []byte {
	if state.Data == nil {
		return nil
	}
	v, _ := state.Data[key].([]byte)
	return v
}

func setBytes(state *eap.EAPState, key string, val []byte) {
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	state.Data[key] = val
}

// loadTEAPReassembler restores the inbound reassembly state persisted by
// saveTEAPReassembler. The peer's Outer TLVs are not restored here: HandleResponse
// copies them onto stateKeyTEAPPeerOuter once, and they must outlive the
// reassembly buffer that resetTEAPReassembler discards.
func loadTEAPReassembler(state *eap.EAPState) *teapmsg.Reassembler {
	declared, _ := state.Data[stateKeyTEAPRxDeclared].(uint32)
	hasLen, _ := state.Data[stateKeyTEAPRxHasLen].(bool)
	return teapmsg.LoadReassembler(getBytes(state, stateKeyTEAPRxBuf), nil, declared, hasLen, 0)
}

func saveTEAPReassembler(state *eap.EAPState, r *teapmsg.Reassembler) {
	if state.Data == nil {
		state.Data = make(map[string]interface{})
	}
	declared, hasLen := r.Declared()
	state.Data[stateKeyTEAPRxBuf] = r.Buffer()
	state.Data[stateKeyTEAPRxDeclared] = declared
	state.Data[stateKeyTEAPRxHasLen] = hasLen
}

func resetTEAPReassembler(state *eap.EAPState) {
	clearKey(state, stateKeyTEAPRxBuf)
	clearKey(state, stateKeyTEAPRxDeclared)
	clearKey(state, stateKeyTEAPRxHasLen)
}
