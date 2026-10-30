package handlers

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/statemanager"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapbind"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapinner"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapmsg"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsfragment"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2869"
)

// --- peer-side TLS client -------------------------------------------------

// connStateExporter adapts a completed crypto/tls connection to the exporter
// shape teapkeys derives the RFC 9427 §2.2 keys with. The peer needs it because
// its keys come from the same TLS session as the server's, exported from the
// client's ConnectionState.
type connStateExporter struct {
	state *tls.ConnectionState
}

// ExportKey returns length octets of exported keying material (RFC 5705).
func (e connStateExporter) ExportKey(label string, context []byte, length int) ([]byte, error) {
	return e.state.ExportKeyingMaterial(label, context, length)
}

// peerTLS is one crypto/tls client of the test peer. The handler under test runs
// the server side of the connection, so what the client writes is what the peer
// sends and what is written to it is what the server sent.
type peerTLS struct {
	conn     *tls.Conn
	to       *hsStream
	from     *hsStream
	done     chan error
	finished bool
	err      error
}

// newPeerTLS creates the client and closes its streams when the test ends, so a
// client goroutine blocked in Read cannot outlive it.
func newPeerTLS(t *testing.T, cfg *tls.Config) *peerTLS {
	t.Helper()
	to, from := newHSStream(), newHSStream()
	t.Cleanup(func() {
		to.close()
		from.close()
	})
	return &peerTLS{
		conn: tls.Client(&hsConn{rd: to, wr: from}, cfg),
		to:   to,
		from: from,
		done: make(chan error, 1),
	}
}

// poll reports whether the client goroutine has exited, recording its error.
func (c *peerTLS) poll() bool {
	if c.finished {
		return true
	}
	select {
	case c.err = <-c.done:
		c.finished = true
		return true
	default:
		return false
	}
}

// hasOutput reports whether the client has written bytes the server has not read.
func (c *peerTLS) hasOutput() bool {
	c.from.mu.Lock()
	defer c.from.mu.Unlock()
	return len(c.from.buf) > 0
}

// idle reports whether the client has consumed everything the server sent and is
// blocked waiting for more. It is the signal that no further output will arrive
// until the server writes again.
func (c *peerTLS) idle() bool {
	c.to.mu.Lock()
	defer c.to.mu.Unlock()
	return len(c.to.buf) == 0 && c.to.reading
}

// awaitOutput waits until the client either wrote a flight or is idle. An idle
// client is a legitimate answer: for example, the TLS layer consumes a session
// ticket without producing anything the EAP layer must carry.
func (c *peerTLS) awaitOutput() ([]byte, error) {
	deadline := time.After(5 * time.Second)
	for {
		if c.hasOutput() {
			return c.from.drain(), nil
		}
		if c.idle() {
			return nil, nil
		}
		if c.poll() {
			if c.err != nil {
				return nil, c.err
			}
			return c.from.drain(), nil
		}
		select {
		case <-deadline:
			return nil, errors.New("peer TLS client produced neither output nor an idle state")
		case <-time.After(time.Millisecond):
		}
	}
}

// takeFlight waits for one complete client flight. The client is either blocked
// reading again (a flight that expects an answer) or finished (the last flight of
// its handshake).
func (c *peerTLS) takeFlight() ([]byte, error) {
	deadline := time.After(5 * time.Second)
	for {
		if c.hasOutput() && (c.idle() || c.poll()) {
			return c.from.drain(), nil
		}
		if c.poll() {
			if c.err != nil {
				return nil, c.err
			}
			if out := c.from.drain(); len(out) > 0 {
				return out, nil
			}
			return nil, errors.New("peer TLS client finished without producing a flight")
		}
		select {
		case <-deadline:
			return nil, errors.New("timed out waiting for a peer TLS flight")
		case <-time.After(time.Millisecond):
		}
	}
}

// --- the Phase 2 peer -----------------------------------------------------

// teapPhase2Peer drives a complete TEAP conversation against the handler: the
// outer TLS tunnel, the Inner Methods the plan asks for, and the
// Intermediate-Result, Crypto-Binding and Result TLVs that close them. It
// computes its own Compound MACs from the same exporters and BUFFER inputs the
// server uses, so a successful run is evidence the two sides agree on §6.3.
type teapPhase2Peer struct {
	t       *testing.T
	h       *TEAPHandler
	sm      eap.EAPStateManager
	stateID string
	secret  string
	ident   uint8

	// What the peer presents.
	eapIdentity string
	password    string
	user        *domain.RadiusUser
	pwdProvider eap.PasswordProvider

	outerCfg *tls.Config
	innerCfg *tls.Config

	outer *peerTLS
	inner *peerTLS
	// innerStarted records whether the Server Initiated Inner Method ran at all,
	// which a resumed session (§3.5) must not.
	innerStarted bool

	peerOuter   []byte
	serverOuter []byte

	chain      *teapkeys.Chain
	exchange   teapbind.Exchange
	exporter   teapkeys.Exporter
	methodsRun int

	// wrongMSKCM corrupts the MSK Compound MAC of every response, so the server's
	// §6.3 verification must fail.
	wrongMSKCM bool

	// outerMSK is the peer's own §6.4 derivation of the granted MSK, computed when
	// the final Crypto-Binding exchange closes the plan.
	outerMSK []byte
	// passwordPromptSeen is the Prompt of the last Basic-Password-Auth-Req.
	passwordPromptSeen string
	// lastErrorCode is the Error-Code the server reported through an Error TLV.
	lastErrorCode uint32
	// lastResultCode is the code of the last RADIUS packet the handler produced.
	lastResultCode radius.Code
	// lastResponse is the RADIUS response of the final round.
	lastResponse *radius.Packet

	outQueue []*teapmsg.Packet
	inBuf    []byte
}

// start launches the outer client. The peer's first EAP response is the
// ClientHello, which carries the Outer TLVs of §4.1.
func (p *teapPhase2Peer) start() {
	p.outer = newPeerTLS(p.t, p.outerCfg)
	go func() { p.outer.done <- p.outerSession() }()
}

// outerSession is the outer client goroutine: the TLS handshake, then one
// decrypted TLV message at a time, answered with its TLVs.
func (p *teapPhase2Peer) outerSession() error {
	c := p.outer
	if err := c.conn.Handshake(); err != nil {
		return fmt.Errorf("peer outer handshake: %w", err)
	}

	state := c.conn.ConnectionState()
	p.exporter = connStateExporter{state: &state}
	seed, err := teapkeys.SessionKeySeed(p.exporter)
	if err != nil {
		return fmt.Errorf("peer session key seed: %w", err)
	}
	chain, err := teapkeys.NewChain(seed)
	if err != nil {
		return err
	}
	hash, err := teapbind.HashFromCipherSuite(state.CipherSuite)
	if err != nil {
		return err
	}
	p.chain = chain
	p.exchange = teapbind.Exchange{
		Hash:           hash,
		OtherPartyType: teapkeys.Type,
		Outer:          teapbind.OuterTLVs{Server: p.serverOuter, Peer: p.peerOuter},
	}

	buf := make([]byte, 16*1024)
	for round := 0; round < 32; round++ {
		n, err := c.conn.Read(buf)
		if err != nil {
			return fmt.Errorf("peer reads the tunnel: %w", err)
		}
		tlvs, err := teaptlv.Parse(buf[:n])
		if err != nil {
			return fmt.Errorf("peer parses the Phase 2 TLVs: %w", err)
		}
		answer, final, err := p.answerPhase2(tlvs)
		if err != nil {
			return err
		}
		if len(answer) > 0 {
			if _, err := c.conn.Write(answer); err != nil {
				return fmt.Errorf("peer writes its Phase 2 answer: %w", err)
			}
		}
		if final {
			return nil
		}
	}
	return errors.New("peer saw more Phase 2 messages than a plan can have")
}

// answerPhase2 produces the peer's answer to one server Phase 2 message.
func (p *teapPhase2Peer) answerPhase2(tlvs []teaptlv.TLV) ([]byte, bool, error) {
	// §3.9.3: a fatal error arrives as a failure Result with an Error TLV. The
	// peer acknowledges the failure Result and the exchange is over.
	if errTLV := teaptlv.Find(tlvs, teaptlv.TypeError); errTLV != nil {
		if len(errTLV.Value) != 4 {
			return nil, false, fmt.Errorf("peer reads an Error TLV of %d octets", len(errTLV.Value))
		}
		p.lastErrorCode = binary.BigEndian.Uint32(errTLV.Value)
		answer, err := teaptlv.Marshal(teaptlv.NewResult(teaptlv.StatusFailure))
		return answer, true, err
	}

	if payload := teaptlv.Find(tlvs, teaptlv.TypeEAPPayload); payload != nil {
		return p.answerInner(payload)
	}
	if request := teaptlv.Find(tlvs, teaptlv.TypeBasicPasswordAuthReq); request != nil {
		return p.answerPassword(request)
	}
	if binding := teaptlv.Find(tlvs, teaptlv.TypeCryptoBinding); binding != nil {
		return p.answerBinding(tlvs, binding)
	}
	if teaptlv.Find(tlvs, teaptlv.TypeResult) != nil {
		answer, err := teaptlv.Marshal(teaptlv.NewResult(teaptlv.StatusFailure))
		return answer, true, err
	}
	return nil, false, errors.New("peer cannot answer the Phase 2 message it received")
}

// answerInner answers one inner EAP request carried in an EAP-Payload TLV.
func (p *teapPhase2Peer) answerInner(payload *teaptlv.TLV) ([]byte, bool, error) {
	packet, _, err := payload.EAPPayload()
	if err != nil {
		return nil, false, err
	}
	msg, err := parseTEAPInnerEAP(packet)
	if err != nil {
		return nil, false, err
	}
	switch msg.Type {
	case eap.TypeIdentity:
		return p.answerIdentity(msg)
	case eap.TypeTLS:
		return p.answerInnerTLS(msg)
	default:
		return nil, false, fmt.Errorf("peer cannot answer inner EAP type %d", msg.Type)
	}
}

// answerIdentity answers the inner EAP-Request/Identity (§3.6.2).
func (p *teapPhase2Peer) answerIdentity(msg *eap.EAPMessage) ([]byte, bool, error) {
	response := (&eap.EAPMessage{
		Code:       eap.CodeResponse,
		Identifier: msg.Identifier,
		Type:       eap.TypeIdentity,
		Data:       []byte(p.eapIdentity),
	}).Encode()
	payload, err := teaptlv.NewEAPPayload(response)
	if err != nil {
		return nil, false, err
	}
	out, err := teaptlv.Marshal(teaptlv.NewIdentityType(teaptlv.IdentityUser), payload)
	return out, false, err
}

// answerInnerTLS drives the inner EAP-TLS client one flight at a time.
func (p *teapPhase2Peer) answerInnerTLS(msg *eap.EAPMessage) ([]byte, bool, error) {
	frag, err := tlsfragment.Parse(msg.Data)
	if err != nil {
		return nil, false, err
	}
	switch {
	case frag.Start():
		if !p.innerStarted {
			p.inner = newPeerTLS(p.t, p.innerCfg)
			p.innerStarted = true
			go func() { p.inner.done <- p.inner.conn.Handshake() }()
		}
	case len(frag.Data) > 0:
		if p.inner == nil {
			return nil, false, errors.New("server sent inner EAP-TLS data before its Start")
		}
		if _, err := p.inner.to.Write(frag.Data); err != nil {
			return nil, false, err
		}
	default:
		return nil, false, errors.New("peer cannot answer an inner EAP-TLS ACK")
	}

	flight, err := p.inner.takeFlight()
	if err != nil {
		return nil, false, err
	}
	response := (&eap.EAPMessage{
		Code:       eap.CodeResponse,
		Identifier: msg.Identifier,
		Type:       eap.TypeTLS,
		Data:       (&tlsfragment.Packet{Data: flight}).Encode(),
	}).Encode()
	payload, err := teaptlv.NewEAPPayload(response)
	if err != nil {
		return nil, false, err
	}
	out, err := teaptlv.Marshal(payload)
	return out, false, err
}

// answerPassword answers the Basic-Password-Auth-Req of §3.6.3.
func (p *teapPhase2Peer) answerPassword(request *teaptlv.TLV) ([]byte, bool, error) {
	prompt, err := request.Prompt()
	if err != nil {
		return nil, false, err
	}
	p.passwordPromptSeen = prompt
	response, err := teaptlv.NewBasicPasswordAuthResp([]byte(p.eapIdentity), []byte(p.password))
	if err != nil {
		return nil, false, err
	}
	out, err := teaptlv.Marshal(response)
	return out, false, err
}

// answerBinding replies to a Crypto-Binding request with the Intermediate-Result,
// the Binding Response and, when the server also sent a Result, the Result TLV
// (§3.6.2, §3.6.6).
func (p *teapPhase2Peer) answerBinding(tlvs []teaptlv.TLV, bindingTLV *teaptlv.TLV) ([]byte, bool, error) {
	binding, err := bindingTLV.CryptoBinding()
	if err != nil {
		return nil, false, err
	}
	if err := p.advanceChain(); err != nil {
		return nil, false, err
	}
	mskCMK, emskCMK := p.chain.CompoundMACKeys()

	flags := teaptlv.FlagMSKPresent
	if binding.Flags&teaptlv.FlagEMSKPresent != 0 {
		flags |= teaptlv.FlagEMSKPresent
	}
	nonce, err := teaptlv.ResponseNonce(binding.Nonce)
	if err != nil {
		return nil, false, err
	}
	answer, err := p.exchange.Bind(
		teapbind.Keys{MSKCMK: mskCMK, EMSKCMK: emskCMK},
		flags, teaptlv.SubTypeBindingResponse, nonce,
	)
	if err != nil {
		return nil, false, err
	}
	if p.wrongMSKCM {
		// The MSK Compound MAC occupies the last 20 octets of the fixed §4.2.13
		// layout, whichever flags announce it.
		answer.Value[len(answer.Value)-1] ^= 0xff
	}

	intermediate, err := teaptlv.NewIntermediateResult(teaptlv.StatusSuccess)
	if err != nil {
		return nil, false, err
	}
	out := []teaptlv.TLV{intermediate, answer}
	if teaptlv.Find(tlvs, teaptlv.TypeResult) != nil {
		out = append(out, teaptlv.NewResult(teaptlv.StatusSuccess))
		useEMSK := binding.Flags&teaptlv.FlagEMSKPresent != 0
		msk, _, kerr := p.chain.SessionKeys(p.exporter, useEMSK)
		if kerr != nil {
			return nil, false, kerr
		}
		p.outerMSK = msk
	}
	payload, err := teaptlv.Marshal(out...)
	return payload, false, err
}

// advanceChain folds the Inner Method that just finished into the peer's copy of
// the §6.2 compound-key chain, after the server has finished the same method.
func (p *teapPhase2Peer) advanceChain() error {
	if p.methodsRun == 0 {
		if err := p.awaitInnerHandshake(); err != nil {
			return err
		}
		state := p.inner.conn.ConnectionState()
		keyMaterial, err := state.ExportKeyingMaterial(eapTLS13KeyLabel, []byte{eap.TypeTLS}, eapTLS13KeyLength)
		if err != nil {
			return fmt.Errorf("peer inner EAP-TLS exporter: %w", err)
		}
		keys, err := teapinner.KeysFor(teapinner.EAPTLS, keyMaterial[:64], keyMaterial[64:])
		if err != nil {
			return err
		}
		if err := p.chain.Advance(p.exporter, keys); err != nil {
			return err
		}
	} else {
		keys, err := teapinner.KeysFor(teapinner.BasicPassword, nil, nil)
		if err != nil {
			return err
		}
		if err := p.chain.Advance(p.exporter, keys); err != nil {
			return err
		}
	}
	p.methodsRun++
	return nil
}

// awaitInnerHandshake waits for the inner TLS client to finish, which must have
// happened by the time the server asks for its Crypto-Binding TLV.
func (p *teapPhase2Peer) awaitInnerHandshake() error {
	if p.inner == nil {
		return errors.New("the inner EAP-TLS method never started")
	}
	deadline := time.After(5 * time.Second)
	for !p.inner.poll() {
		select {
		case <-deadline:
			return errors.New("inner EAP-TLS handshake did not finish")
		case <-time.After(time.Millisecond):
		}
	}
	if p.inner.err != nil {
		return fmt.Errorf("peer inner EAP-TLS handshake: %w", p.inner.err)
	}
	return nil
}

// --- driving the handler --------------------------------------------------

// run drives the conversation until the handler grants or rejects, and returns
// the grant flag and the rejection error.
func (p *teapPhase2Peer) run() (bool, error) {
	p.start()
	hello, err := p.outer.takeFlight()
	if err != nil {
		return false, err
	}
	if len(hello) == 0 {
		return false, errors.New("peer produced no ClientHello")
	}
	p.queueTLS(hello, p.peerOuter)

	for round := 0; round < 1024; round++ {
		ctx, writer := teapResponse(p.t, p.sm, p.stateID, p.secret, p.ident, p.nextOutbound())
		p.ident++
		ctx.PwdProvider = p.pwdProvider
		ctx.User = p.user

		ok, err := p.h.HandleResponse(ctx)
		p.lastResponse = ctx.Response
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
		if writer.response == nil {
			return false, errors.New("handler neither granted nor answered the round")
		}
		p.lastResultCode = writer.response.Code

		pkt, err := p.parseChallenge(writer.response)
		if err != nil {
			return false, err
		}
		p.inBuf = append(p.inBuf, pkt.TLSData...)
		if pkt.More() {
			p.queueACK()
			continue
		}
		flight := p.inBuf
		p.inBuf = nil
		if len(flight) == 0 {
			// A server ACK of one of our fragments; the rest is already queued.
			continue
		}
		if _, err := p.outer.to.Write(flight); err != nil {
			return false, err
		}
		out, err := p.outer.awaitOutput()
		if err != nil {
			return false, err
		}
		if len(out) == 0 {
			p.queueACK()
			continue
		}
		p.queueTLS(out, nil)
	}
	return false, errors.New("the TEAP exchange did not finish within the round budget")
}

// queueTLS fragments one peer message into TEAP packets.
func (p *teapPhase2Peer) queueTLS(data, outer []byte) {
	pkts, err := teapmsg.Fragment(teapmsg.Version, data, outer, 0)
	require.NoError(p.t, err)
	p.outQueue = append(p.outQueue, pkts...)
}

// queueACK queues the bare acknowledgment §4.1 uses between fragments.
func (p *teapPhase2Peer) queueACK() {
	p.outQueue = append(p.outQueue, &teapmsg.Packet{Version: teapmsg.Version})
}

// nextOutbound returns the peer's next TEAP packet.
func (p *teapPhase2Peer) nextOutbound() *teapmsg.Packet {
	require.NotEmpty(p.t, p.outQueue, "the peer must always have a packet to answer with")
	pkt := p.outQueue[0]
	p.outQueue = p.outQueue[1:]
	return pkt
}

// parseChallenge decodes the EAP-Request/TEAP inside a RADIUS challenge.
func (p *teapPhase2Peer) parseChallenge(resp *radius.Packet) (*teapmsg.Packet, error) {
	eapData, err := rfc2869.EAPMessage_Lookup(resp)
	if err != nil {
		return nil, err
	}
	if len(eapData) < 6 || eapData[4] != eap.TypeTEAP {
		return nil, fmt.Errorf("challenge carries %d octets of EAP data", len(eapData))
	}
	return teapmsg.Parse(eapData[5:])
}

// --- fixture --------------------------------------------------------------

// teapPhase2Fixture builds a handler that runs the full Phase 2 over a TLS 1.3
// tunnel whose tickets are enabled, like the production settings provider.
type teapPhase2Fixture struct {
	t        *testing.T
	ca       *hsTestCA
	h        *TEAPHandler
	outerTpl *tlsengine.Config
	innerTpl *tlsengine.Config
	prompt   string
}

func newTEAPPhase2Fixture(t *testing.T, prompt string) *teapPhase2Fixture {
	t.Helper()
	ca := newHSTestCA(t, "TEAP Phase 2 Test Root CA")

	ticketKey := make([]byte, tlsengine.SessionTicketKeySize)
	_, err := rand.Read(ticketKey)
	require.NoError(t, err)

	outer := &tlsengine.Config{
		ServerCertificate: ca.issue(t, "radius.example.com", func(c *x509.Certificate) {
			c.DNSNames = []string{"radius.example.com"}
		}),
		ServerOnly:           true,
		MinVersion:           tls.VersionTLS13,
		EnableSessionTickets: true,
		SessionTicketKey:     ticketKey,
		HandshakeTimeout:     5 * time.Second,
	}
	inner := &tlsengine.Config{
		ServerCertificate: ca.issue(t, "radius.example.com", func(c *x509.Certificate) {
			c.DNSNames = []string{"radius.example.com"}
		}),
		ClientCAs:        ca.pool,
		MinVersion:       tls.VersionTLS13,
		MaxVersion:       tls.VersionTLS13,
		HandshakeTimeout: 5 * time.Second,
	}
	h := NewTEAPHandlerWithOptions(
		func() (*tlsengine.Config, error) { copy := *outer; return &copy, nil },
		TEAPOptions{
			InnerTLSConfigProvider: func() (*tlsengine.Config, error) { copy := *inner; return &copy, nil },
			PasswordPrompt:         func() string { return prompt },
		},
	)
	return &teapPhase2Fixture{t: t, ca: ca, h: h, outerTpl: outer, innerTpl: inner, prompt: prompt}
}

// startSession answers EAP-Response/Identity and returns the issued State plus the
// Outer TLVs of the TEAP Start, which §6.3 puts in the Crypto-Binding BUFFER.
func (f *teapPhase2Fixture) startSession(t *testing.T, sm eap.EAPStateManager, username string) (string, []byte) {
	t.Helper()
	stateID, resp := startTEAPExchange(t, f.h, sm, username, "secret")
	eapData, err := rfc2869.EAPMessage_Lookup(resp)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(eapData), 6)
	start, err := teapmsg.Parse(eapData[5:])
	require.NoError(t, err)
	require.True(t, start.Start(), "the server opens with a Start")
	return stateID, append([]byte(nil), start.OuterTLVs...)
}

// newPeer builds the peer for one session. certCN names the certificate the inner
// EAP-TLS client presents; an empty value uses the EAP identity.
func (f *teapPhase2Fixture) newPeer(t *testing.T, sm eap.EAPStateManager, stateID string, serverOuter []byte, cache tls.ClientSessionCache, certCN string) *teapPhase2Peer {
	t.Helper()
	peerOuter, err := teaptlv.Marshal(teaptlv.NewAuthorityID([]byte("peer-hint")))
	require.NoError(t, err)
	if certCN == "" {
		certCN = "alice"
	}

	return &teapPhase2Peer{
		t: t, h: f.h, sm: sm, stateID: stateID, secret: "secret", ident: 20,
		eapIdentity: "alice",
		password:    "s3cret",
		user:        &domain.RadiusUser{Username: "alice", Password: "s3cret"},
		pwdProvider: &mockPasswordProvider{},
		outerCfg: &tls.Config{
			RootCAs:            f.ca.pool,
			ServerName:         "radius.example.com",
			MinVersion:         tls.VersionTLS13,
			ClientSessionCache: cache,
		},
		innerCfg: &tls.Config{
			Certificates: []tls.Certificate{f.ca.issue(t, certCN, nil)},
			RootCAs:      f.ca.pool,
			ServerName:   "radius.example.com",
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
		},
		peerOuter:   peerOuter,
		serverOuter: serverOuter,
	}
}

// --- tests ----------------------------------------------------------------

func TestTEAPPhase2_EAPTLSOnlyGrants(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "")
	ok, err := peer.run()
	require.NoError(t, err)
	require.True(t, ok, "§3.6.6 requires the protected termination to grant the session")

	state, err := sm.GetState(stateID)
	require.NoError(t, err)
	assert.True(t, state.Success)
	assert.Equal(t, "alice", state.Username, "§3.11 authorizes on the Phase 2 credentials")
	assert.Empty(t, getTEAPFragments(state), "no server fragment may be left queued")
	assert.False(t, getBool(state, stateKeyTEAPPhase2), "Phase 2 must be closed on success")

	assert.Equal(t, 1, peer.methodsRun)
	require.Len(t, peer.outerMSK, 64, "the peer must derive the §6.4 MSK itself")
	assert.Equal(t, peer.outerMSK[:32], microsoft.MSMPPERecvKey_Get(peer.lastResponse))
	assert.Equal(t, peer.outerMSK[32:64], microsoft.MSMPPESendKey_Get(peer.lastResponse))
}

func TestTEAPPhase2_EAPTLSThenPasswordGrants(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "Password for the test realm")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "")
	ok, err := peer.run()
	require.NoError(t, err)
	require.True(t, ok)

	state, err := sm.GetState(stateID)
	require.NoError(t, err)
	assert.True(t, state.Success)
	assert.Equal(t, "alice", state.Username)

	assert.Equal(t, 2, peer.methodsRun, "both Inner Methods of the plan must be bound")
	assert.Equal(t, f.prompt, peer.passwordPromptSeen, "§3.6.3 requires the first request to carry the prompt")
	require.Len(t, peer.outerMSK, 64)
	assert.Equal(t, peer.outerMSK[:32], microsoft.MSMPPERecvKey_Get(peer.lastResponse))
	assert.Equal(t, peer.outerMSK[32:64], microsoft.MSMPPESendKey_Get(peer.lastResponse))
}

func TestTEAPPhase2_FragmentedEnvelope(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "")
	f.h.maxFragment = 64
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "")
	ok, err := peer.run()
	require.NoError(t, err)
	require.True(t, ok)

	state, err := sm.GetState(stateID)
	require.NoError(t, err)
	assert.True(t, state.Success)
	assert.Empty(t, getTEAPFragments(state))
}

func TestTEAPPhase2_PasswordMismatchRejects(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "Password")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "")
	peer.password = "not-the-password"
	ok, err := peer.run()
	require.False(t, ok)
	assert.ErrorIs(t, err, eap.ErrTEAPPassword)
	assert.Equal(t, teapErrAuthFailure, peer.lastErrorCode,
		"§4.2.6 error 1003 generalizes the failure so usernames cannot be enumerated")
	assert.Equal(t, radius.CodeAccessChallenge, peer.lastResultCode)

	state, err := sm.GetState(stateID)
	require.NoError(t, err)
	assert.False(t, state.Success)
}

func TestTEAPPhase2_CertificateIdentityMismatchRejects(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	// The peer claims "alice" in EAP but presents mallory's certificate.
	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "mallory")
	ok, err := peer.run()
	require.False(t, ok)
	assert.ErrorIs(t, err, eap.ErrTLSIdentityMismatch)
	assert.Equal(t, teapErrInnerMethod, peer.lastErrorCode)
}

func TestTEAPPhase2_WrongCompoundMACRejects(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	stateID, serverOuter := f.startSession(t, sm, "anonymous@example.org")

	peer := f.newPeer(t, sm, stateID, serverOuter, nil, "")
	peer.wrongMSKCM = true
	ok, err := peer.run()
	require.False(t, ok)
	assert.ErrorIs(t, err, eap.ErrTEAPCryptoBinding)
	assert.Equal(t, teapErrBadMSKCM, peer.lastErrorCode,
		"§4.2.6 error 2006 reports a failing MSK Compound MAC")
}

// TestTEAPPhase2_ResumptionSkipsPhase2 covers RFC 9930 §3.5 together with §3.11:
// the first session associates its Phase 2 credentials with the issued ticket, and
// a second session that resumes it is granted on those cached credentials without
// running any Inner Method.
func TestTEAPPhase2_ResumptionSkipsPhase2(t *testing.T) {
	f := newTEAPPhase2Fixture(t, "")
	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()
	cache := tls.NewLRUClientSessionCache(4)

	firstID, firstOuter := f.startSession(t, sm, "anonymous@example.org")
	first := f.newPeer(t, sm, firstID, firstOuter, cache, "")
	ok, err := first.run()
	require.NoError(t, err)
	require.True(t, ok)
	require.True(t, first.innerStarted)
	require.Len(t, first.outerMSK, 64)

	secondID, secondOuter := f.startSession(t, sm, "anonymous@example.org")
	second := f.newPeer(t, sm, secondID, secondOuter, cache, "")
	ok, err = second.run()
	require.NoError(t, err)
	require.True(t, ok, "a resumed session is granted without Phase 2")

	state, err := sm.GetState(secondID)
	require.NoError(t, err)
	assert.True(t, state.Success)
	assert.Equal(t, "alice", state.Username, "§3.11 resumes on the credentials the ticket was bound to")
	assert.False(t, second.innerStarted, "§3.5 bypasses Phase 2 entirely")
	assert.Equal(t, 0, second.methodsRun)
	assert.Equal(t, first.outerMSK[:32], microsoft.MSMPPERecvKey_Get(second.lastResponse))
	assert.Equal(t, first.outerMSK[32:64], microsoft.MSMPPESendKey_Get(second.lastResponse))
}

// --- ticket store ---------------------------------------------------------

func TestTEAPTicketStore_Lifecycle(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	store := newTEAPTicketStore()
	store.now = func() time.Time { return now }

	store.put("pending", []byte("session-state"))
	rec, ok := store.get("pending")
	require.True(t, ok)
	assert.Equal(t, []byte("session-state"), rec.state)
	assert.False(t, rec.complete)

	assert.True(t, store.complete("pending", "alice", bytes.Repeat([]byte{1}, 64)))
	assert.False(t, store.complete("missing", "alice", nil))

	rec, ok = store.get("pending")
	require.True(t, ok)
	assert.True(t, rec.complete)
	assert.Equal(t, "alice", rec.username)
	assert.Len(t, rec.msk, 64)

	// An uncompleted ticket that outlives its Phase 2 window is not usable.
	store.put("abandoned", []byte("state"))
	now = now.Add(teapTicketPendingTTL + time.Second)
	_, ok = store.get("abandoned")
	assert.False(t, ok)
	// The completed record survived the pending window and lives for the ticket
	// lifetime.
	_, ok = store.get("pending")
	assert.True(t, ok)
	now = now.Add(teapTicketCompleteTTL)
	_, ok = store.get("pending")
	assert.False(t, ok)

	store.put("explicit", []byte("state"))
	store.delete("explicit")
	_, ok = store.get("explicit")
	assert.False(t, ok)
}

func TestTEAPTicketStore_EvictsWhenFull(t *testing.T) {
	store := newTEAPTicketStore()
	for i := 0; i < teapTicketMaxEntries; i++ {
		store.put(fmt.Sprintf("id-%d", i), []byte("state"))
	}
	store.put("one-too-many", []byte("state"))

	store.mu.Lock()
	size := len(store.records)
	_, kept := store.records["one-too-many"]
	store.mu.Unlock()
	assert.Equal(t, teapTicketMaxEntries, size, "the store must stay bounded")
	assert.True(t, kept)
}

func TestTEAPTicketHolder_RefusesPendingTicket(t *testing.T) {
	store := newTEAPTicketStore()
	holder := &teapTicketHolder{store: store}
	// A ticket whose Phase 2 never completed carries state but no credentials.
	store.put("pending", []byte("not a serialized session"))

	session, err := holder.unwrap([]byte("pending"), tls.ConnectionState{})
	require.NoError(t, err)
	assert.Nil(t, session, "crypto/tls must fall back to a full handshake")
	assert.Nil(t, holder.resumedCredentials())

	_, ok := store.get("pending")
	assert.False(t, ok, "§3.11 invalidates a ticket it cannot associate")
}

func TestTEAPTicketHolder_AssociatesIssuedTickets(t *testing.T) {
	store := newTEAPTicketStore()
	holder := &teapTicketHolder{store: store}
	holder.issued = []string{"first", "second"}
	store.put("first", []byte("state-a"))
	store.put("second", []byte("state-b"))

	msk := bytes.Repeat([]byte{7}, 64)
	holder.associate("alice", msk)

	for _, id := range holder.issued {
		rec, ok := store.get(id)
		require.True(t, ok)
		assert.True(t, rec.complete)
		assert.Equal(t, "alice", rec.username)
		assert.Equal(t, msk, rec.msk)
	}

	// A credentials copy is handed to the handshake, never the stored slice.
	holder.resumed = &teapTicketCredentials{username: "alice", msk: msk}
	creds := holder.resumedCredentials()
	require.NotNil(t, creds)
	creds.msk[0] = 0
	assert.Equal(t, byte(7), holder.resumed.msk[0])
}
