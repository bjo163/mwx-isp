package handlers

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/statemanager"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapmsg"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

func TestTEAPHandler_Metadata(t *testing.T) {
	h := NewTEAPHandler()
	assert.Equal(t, EAPMethodTEAP, h.Name())
	assert.Equal(t, "eap-teap", h.Name())
	assert.Equal(t, uint8(eap.TypeTEAP), h.EAPType())
	assert.Equal(t, uint8(55), h.EAPType())

	assert.True(t, h.CanHandle(&eap.EAPContext{EAPMessage: &eap.EAPMessage{Type: eap.TypeTEAP}}))
	assert.False(t, h.CanHandle(&eap.EAPContext{EAPMessage: &eap.EAPMessage{Type: eap.TypeTLS}}))
	assert.False(t, h.CanHandle(&eap.EAPContext{}))
	assert.False(t, h.CanHandle(nil))
}

// startTEAPExchange answers EAP-Response/Identity through the handler and returns
// the issued State and the TEAP Start it sent.
func startTEAPExchange(t *testing.T, h *TEAPHandler, sm eap.EAPStateManager, username, secret string) (string, *radius.Packet) {
	t.Helper()
	packet := radius.New(radius.CodeAccessRequest, []byte(secret))
	require.NoError(t, rfc2865.UserName_SetString(packet, username))
	writer := &mockResponseWriter{}
	ctx := &eap.EAPContext{
		Request:        &radius.Request{Packet: packet},
		ResponseWriter: writer,
		StateManager:   sm,
		Secret:         secret,
		EAPMessage:     &eap.EAPMessage{Code: eap.CodeResponse, Identifier: 1, Type: eap.TypeIdentity},
	}
	handled, err := h.HandleIdentity(ctx)
	require.NoError(t, err)
	require.True(t, handled)
	require.NotNil(t, writer.response, "the identity phase must answer with a challenge")
	return rfc2865.State_GetString(writer.response), writer.response
}

func TestTEAPHandler_HandleIdentity_SendsStartWithAuthorityID(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID, resp := startTEAPExchange(t, h, sm, "alice@example.org", "secret")
	require.NotEmpty(t, stateID)
	assert.Equal(t, radius.CodeAccessChallenge, resp.Code)

	eapData, err := rfc2869.EAPMessage_Lookup(resp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(eapData), 5)
	assert.Equal(t, byte(eap.CodeRequest), eapData[0])
	assert.Equal(t, byte(eap.TypeTEAP), eapData[4])

	pkt, err := teapmsg.Parse(eapData[5:])
	require.NoError(t, err)
	assert.True(t, pkt.Start(), "the server opens with the Start bit (§3.2)")
	assert.Equal(t, teapmsg.Version, pkt.Version)
	assert.False(t, pkt.More())
	assert.False(t, pkt.HasLength, "an empty TLS payload needs no Message Length")
	assert.True(t, pkt.HasOuterTLVs, "the Start carries the Authority-ID TLV (§3.2)")
	assert.Empty(t, pkt.TLSData)

	tlvs, err := teaptlv.Parse(pkt.OuterTLVs)
	require.NoError(t, err)
	require.Len(t, tlvs, 1)
	assert.Equal(t, teaptlv.TypeAuthorityID, tlvs[0].Type)
	assert.Equal(t, []byte("example.org"), tlvs[0].Value)

	state, err := sm.GetState(stateID)
	require.NoError(t, err)
	assert.Equal(t, EAPMethodTEAP, state.Method)
	assert.Equal(t, "alice@example.org", state.Username)
}

func TestTEAPAuthorityID(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     string
	}{
		{name: "realm", username: "alice@example.org", want: "example.org"},
		{name: "no realm", username: "alice", want: "alice"},
		{name: "trailing at sign", username: "alice@", want: "alice@"},
		{name: "last realm wins", username: "alice@host@example.org", want: "example.org"},
		{name: "empty", username: "", want: ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, teapAuthorityID(tc.username))
		})
	}
}

// teapResponse builds one EAP-Response/TEAP round bound to stateID.
func teapResponse(t *testing.T, sm eap.EAPStateManager, stateID, secret string, ident uint8, pkt *teapmsg.Packet) (*eap.EAPContext, *mockResponseWriter) {
	t.Helper()
	packet := radius.New(radius.CodeAccessRequest, []byte(secret))
	require.NoError(t, rfc2865.State_SetString(packet, stateID))
	data, err := pkt.Encode()
	require.NoError(t, err)
	writer := &mockResponseWriter{}
	return &eap.EAPContext{
		Request:        &radius.Request{Packet: packet},
		ResponseWriter: writer,
		Response:       packet.Response(radius.CodeAccessAccept),
		StateManager:   sm,
		Secret:         secret,
		EAPMessage:     &eap.EAPMessage{Code: eap.CodeResponse, Identifier: ident, Type: eap.TypeTEAP, Data: data},
	}, writer
}

func newTEAPState(t *testing.T, sm eap.EAPStateManager, stateID string) *eap.EAPState {
	t.Helper()
	state := &eap.EAPState{StateID: stateID, Method: EAPMethodTEAP}
	require.NoError(t, sm.SetState(stateID, state))
	return state
}

func TestTEAPHandler_HandleResponse_RequiresState(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	packet := radius.New(radius.CodeAccessRequest, []byte("secret"))
	data, err := (&teapmsg.Packet{Version: teapmsg.Version}).Encode()
	require.NoError(t, err)
	ctx := &eap.EAPContext{
		Request:        &radius.Request{Packet: packet},
		ResponseWriter: &mockResponseWriter{},
		StateManager:   sm,
		Secret:         "secret",
		EAPMessage:     &eap.EAPMessage{Type: eap.TypeTEAP, Data: data},
	}
	success, err := h.HandleResponse(ctx)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrStateNotFound)
}

func TestTEAPHandler_HandleResponse_RejectsMalformedEnvelope(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	newTEAPState(t, sm, "state-1")

	ctx, _ := teapResponse(t, sm, "state-1", "secret", 2, &teapmsg.Packet{Version: teapmsg.Version})
	ctx.EAPMessage.Data = nil
	success, err := h.HandleResponse(ctx)
	assert.False(t, success)
	assert.ErrorIs(t, err, teapmsg.ErrEmptyPayload)
}

func TestTEAPHandler_HandleResponse_RejectsPeerStart(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	newTEAPState(t, sm, "state-1")

	ctx, _ := teapResponse(t, sm, "state-1", "secret", 2, &teapmsg.Packet{
		Flags:   teapmsg.FlagStart,
		Version: teapmsg.Version,
	})
	success, err := h.HandleResponse(ctx)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrTEAPProtocol)
}

func TestTEAPHandler_HandleResponse_RejectsUnsupportedVersion(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	newTEAPState(t, sm, "state-1")

	ctx, _ := teapResponse(t, sm, "state-1", "secret", 2, &teapmsg.Packet{Version: 2})
	success, err := h.HandleResponse(ctx)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrTEAPVersionUnsupported)
}

func TestTEAPHandler_HandleResponse_RejectsVersionChangeMidConversation(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	// The first message is accepted as a partial fragment, so the conversation
	// continues and the next message can be checked against the negotiated value.
	stateID := "state-1"
	newTEAPState(t, sm, stateID)
	first, writer := teapResponse(t, sm, stateID, "secret", 2, &teapmsg.Packet{
		Flags:   teapmsg.FlagMoreFragments,
		Version: teapmsg.Version,
		TLSData: []byte{0x16, 0x03, 0x01},
	})
	success, err := h.HandleResponse(first)
	require.NoError(t, err)
	require.False(t, success)
	ack, err := rfc2869.EAPMessage_Lookup(writer.response)
	require.NoError(t, err)
	ackPkt, err := teapmsg.Parse(ack[5:])
	require.NoError(t, err)
	assert.True(t, ackPkt.IsACK())

	second, _ := teapResponse(t, sm, stateID, "secret", 3, &teapmsg.Packet{Version: 2})
	success, err = h.HandleResponse(second)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrTEAPProtocol)
}

func TestTEAPHandler_HandleResponse_RejectsOuterTLVsOutsideInitialMessage(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := "state-1"
	newTEAPState(t, sm, stateID)
	first, _ := teapResponse(t, sm, stateID, "secret", 2, &teapmsg.Packet{
		Flags:   teapmsg.FlagMoreFragments,
		Version: teapmsg.Version,
		TLSData: []byte{0x16, 0x03, 0x01},
	})
	success, err := h.HandleResponse(first)
	require.NoError(t, err)
	require.False(t, success)

	outer, err := teaptlv.Marshal(teaptlv.NewAuthorityID([]byte("late")))
	require.NoError(t, err)
	second, _ := teapResponse(t, sm, stateID, "secret", 3, &teapmsg.Packet{
		Version:      teapmsg.Version,
		HasOuterTLVs: true,
		OuterTLVs:    outer,
	})
	success, err = h.HandleResponse(second)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrTEAPProtocol)
}

func TestTEAPHandler_HandleResponse_FragmentQueue(t *testing.T) {
	t.Run("a non-ack is refused while fragments are queued", func(t *testing.T) {
		h := NewTEAPHandler()
		sm := statemanager.NewMemoryStateManager()
		defer sm.Close()

		stateID := "state-1"
		state := newTEAPState(t, sm, stateID)
		setUint8(state, stateKeyTEAPVersion, teapmsg.Version)
		setTEAPFragments(state, []*teapmsg.Packet{{Version: teapmsg.Version, TLSData: []byte{0x01}}})
		// SetState copies the state, so re-persist after mutating it.
		require.NoError(t, sm.SetState(stateID, state))

		ctx, _ := teapResponse(t, sm, stateID, "secret", 2, &teapmsg.Packet{
			Version: teapmsg.Version,
			TLSData: []byte{0x02},
		})
		success, err := h.HandleResponse(ctx)
		assert.False(t, success)
		assert.ErrorIs(t, err, eap.ErrTEAPUnexpectedFragment)
	})

	t.Run("an ack releases the next fragment", func(t *testing.T) {
		h := NewTEAPHandler()
		sm := statemanager.NewMemoryStateManager()
		defer sm.Close()

		stateID := "state-1"
		state := newTEAPState(t, sm, stateID)
		setUint8(state, stateKeyTEAPVersion, teapmsg.Version)
		setTEAPFragments(state, []*teapmsg.Packet{
			{Version: teapmsg.Version, TLSData: []byte{0x01, 0x02}, Flags: teapmsg.FlagMoreFragments},
			{Version: teapmsg.Version, TLSData: []byte{0x03}},
		})
		// SetState copies the state, so re-persist after mutating it.
		require.NoError(t, sm.SetState(stateID, state))

		ctx, writer := teapResponse(t, sm, stateID, "secret", 2, &teapmsg.Packet{Version: teapmsg.Version})
		success, err := h.HandleResponse(ctx)
		require.NoError(t, err)
		require.False(t, success)

		eapData, err := rfc2869.EAPMessage_Lookup(writer.response)
		require.NoError(t, err)
		sent, err := teapmsg.Parse(eapData[5:])
		require.NoError(t, err)
		assert.Equal(t, []byte{0x01, 0x02}, sent.TLSData)
		assert.True(t, sent.More())

		// Read the persisted state: the handler works on its own copy.
		persisted, err := sm.GetState(stateID)
		require.NoError(t, err)
		queued := getTEAPFragments(persisted)
		require.Len(t, queued, 1)
		assert.Equal(t, []byte{0x03}, queued[0].TLSData)
	})
}

func TestTEAPHandler_HandshakeWithoutTLSMaterialRejects(t *testing.T) {
	h := NewTEAPHandler()
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID, _ := startTEAPExchange(t, h, sm, "alice@example.org", "secret")
	ctx, _ := teapResponse(t, sm, stateID, "secret", 2, &teapmsg.Packet{
		Version: teapmsg.Version,
		TLSData: []byte{0x16, 0x03, 0x01, 0x00, 0x00},
	})
	success, err := h.HandleResponse(ctx)
	assert.False(t, success)
	assert.ErrorIs(t, err, eap.ErrTLSNotConfigured)
}

// --- end-to-end Phase 1 handshake over the TEAP envelope ------------------

// teapSplitter drives a real crypto/tls client through the handler, speaking the
// TEAP envelope in both directions: it fragments what the client writes, ACKs each
// server fragment, and reassembles the server's flights before delivering them.
type teapSplitter struct {
	t       *testing.T
	h       *TEAPHandler
	sm      eap.EAPStateManager
	stateID string
	secret  string
	ident   uint8

	fragSize int
	// outer is sent with the peer's first message, as §4.1 allows the O field there.
	outer []byte

	toClient   *hsStream
	fromClient *hsStream
	clientDone chan error
	finished   bool
	clientErr  error

	outQueue []*teapmsg.Packet
	inBuf    []byte
}

func newTEAPSplitter(t *testing.T, h *TEAPHandler, sm eap.EAPStateManager, stateID, secret string, clientCfg *tls.Config, fragSize int) *teapSplitter {
	t.Helper()
	toClient := newHSStream()
	fromClient := newHSStream()
	client := tls.Client(&hsConn{rd: toClient, wr: fromClient}, clientCfg)
	done := make(chan error, 1)
	go func() { done <- client.Handshake() }()
	t.Cleanup(func() {
		_ = client.Close()
		toClient.close()
		fromClient.close()
	})

	return &teapSplitter{
		t: t, h: h, sm: sm, stateID: stateID, secret: secret, ident: 20,
		fragSize: fragSize, toClient: toClient, fromClient: fromClient, clientDone: done,
	}
}

func (s *teapSplitter) queueTLS(data, outer []byte) {
	pkts, err := teapmsg.Fragment(teapmsg.Version, data, outer, s.fragSize)
	require.NoError(s.t, err)
	s.outQueue = append(s.outQueue, pkts...)
}

func (s *teapSplitter) queueACK() {
	s.outQueue = append(s.outQueue, &teapmsg.Packet{Version: teapmsg.Version})
}

func (s *teapSplitter) nextOutbound() *teapmsg.Packet {
	require.NotEmpty(s.t, s.outQueue, "the peer must always have a packet to answer with")
	pkt := s.outQueue[0]
	s.outQueue = s.outQueue[1:]
	return pkt
}

func (s *teapSplitter) waitSettled() {
	if s.finished {
		return
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case err := <-s.clientDone:
			s.finished = true
			s.clientErr = err
			return
		case <-deadline:
			s.t.Fatal("timed out waiting for the TLS client to settle")
		default:
		}
		s.toClient.mu.Lock()
		settled := len(s.toClient.buf) == 0 && s.toClient.reading
		s.toClient.mu.Unlock()
		if settled {
			return
		}
		select {
		case err := <-s.clientDone:
			s.finished = true
			s.clientErr = err
			return
		case <-time.After(2 * time.Millisecond):
		}
	}
}

func (s *teapSplitter) nextClientFlight() []byte {
	s.waitSettled()
	return s.fromClient.drain()
}

// run drives the exchange until the handler rejects, and returns that error.
func (s *teapSplitter) run() error {
	hello := s.nextClientFlight()
	require.NotEmpty(s.t, hello, "the TLS client must produce a ClientHello")
	s.queueTLS(hello, s.outer)

	for round := 0; round < 128; round++ {
		writer := &mockResponseWriter{}
		ctx := s.responseCtx(writer, s.nextOutbound())
		ok, err := s.h.HandleResponse(ctx)
		if err != nil {
			return err
		}
		require.False(s.t, ok, "Phase 1 must never grant access")
		require.NotNil(s.t, writer.response, "the handler must answer with a challenge")
		require.Equal(s.t, radius.CodeAccessChallenge, writer.response.Code)

		pkt := s.parseChallenge(writer.response)
		s.inBuf = append(s.inBuf, pkt.TLSData...)
		if pkt.More() {
			s.queueACK()
			continue
		}
		flight := s.inBuf
		s.inBuf = nil
		if len(flight) == 0 {
			// A server ACK of one of our fragments asks nothing back: the rest of
			// our flight is already queued.
			continue
		}
		_, werr := s.toClient.Write(flight)
		require.NoError(s.t, werr)
		next := s.nextClientFlight()
		if len(next) == 0 {
			s.queueACK()
			continue
		}
		s.queueTLS(next, nil)
	}
	s.t.Fatal("the TEAP Phase 1 exchange did not finish within the round budget")
	return nil
}

func (s *teapSplitter) responseCtx(writer *mockResponseWriter, pkt *teapmsg.Packet) *eap.EAPContext {
	s.ident++
	ctx, _ := teapResponse(s.t, s.sm, s.stateID, s.secret, s.ident, pkt)
	ctx.ResponseWriter = writer
	return ctx
}

func (s *teapSplitter) parseChallenge(resp *radius.Packet) *teapmsg.Packet {
	s.t.Helper()
	eapData, err := rfc2869.EAPMessage_Lookup(resp)
	require.NoError(s.t, err)
	require.GreaterOrEqual(s.t, len(eapData), 6)
	require.Equal(s.t, byte(eap.TypeTEAP), eapData[4])
	pkt, err := teapmsg.Parse(eapData[5:])
	require.NoError(s.t, err)
	return pkt
}

func teapServerConfig(t *testing.T, ca *hsTestCA, minVersion, maxVersion uint16) *tlsengine.Config {
	t.Helper()
	return &tlsengine.Config{
		ServerCertificate: ca.issue(t, "radius.example.com", func(c *x509.Certificate) {
			c.DNSNames = []string{"radius.example.com"}
		}),
		ServerOnly:       true,
		MinVersion:       minVersion,
		MaxVersion:       maxVersion,
		HandshakeTimeout: 5 * time.Second,
	}
}

func teapClientConfig(ca *hsTestCA, minVersion, maxVersion uint16) *tls.Config {
	return &tls.Config{
		RootCAs:    ca.pool,
		ServerName: "radius.example.com",
		MinVersion: minVersion,
		MaxVersion: maxVersion,
	}
}

func TestTEAPHandler_FullHandshake_TLS13ReachesPhase2Gate(t *testing.T) {
	ca := newHSTestCA(t, "TEAP Test Root CA")
	cfg := teapServerConfig(t, ca, tls.VersionTLS13, 0)
	h := NewTEAPHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })
	h.maxFragment = 64

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID, _ := startTEAPExchange(t, h, sm, "alice@example.org", "secret")

	peerOuter, err := teaptlv.Marshal(teaptlv.NewAuthorityID([]byte("peer-hint")))
	require.NoError(t, err)

	splitter := newTEAPSplitter(t, h, sm, stateID, "secret", teapClientConfig(ca, tls.VersionTLS13, 0), 64)
	splitter.outer = peerOuter

	err = splitter.run()
	assert.ErrorIs(t, err, eap.ErrTEAPPhase2NotImplemented,
		"a completed TLS 1.3 tunnel must stop at the Phase 2 gate, not grant access")

	state, serr := sm.GetState(stateID)
	require.NoError(t, serr)
	assert.False(t, state.Success)
	assert.Equal(t, teapmsg.Version, getUint8(state, stateKeyTEAPVersion))
	assert.Equal(t, peerOuter, getBytes(state, stateKeyTEAPPeerOuter),
		"§6.3 appends the peer's Phase 1 Outer TLVs to the Crypto-Binding BUFFER")
	assert.Empty(t, getTEAPFragments(state), "no server fragment may be left queued")
}

func TestTEAPHandler_FullHandshake_TLS12Rejected(t *testing.T) {
	ca := newHSTestCA(t, "TEAP Test Root CA")
	cfg := teapServerConfig(t, ca, tls.VersionTLS12, tls.VersionTLS12)
	h := NewTEAPHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID, _ := startTEAPExchange(t, h, sm, "alice@example.org", "secret")
	splitter := newTEAPSplitter(t, h, sm, stateID, "secret", teapClientConfig(ca, tls.VersionTLS12, tls.VersionTLS12), 0)

	err := splitter.run()
	assert.ErrorIs(t, err, eap.ErrTEAPRequiresTLS13,
		"teapkeys implements the RFC 9427 §2.2 TLS 1.3 exporter only, so a 1.2 tunnel cannot reach Phase 2")
}
