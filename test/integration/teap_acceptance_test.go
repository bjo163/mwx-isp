//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"github.com/bjo163/mwx-isp/internal/domain"
	eap "github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	eaphandlers "github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/handlers"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapbind"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapinner"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapmsg"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsfragment"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// The inner EAP-TLS method exports its MSK and EMSK through the RFC 9190 §2.3
// exporter, which is the same label and context standalone EAP-TLS uses.
const (
	teapExportLabel = "EXPORTER_EAP_TLS_Key_Material"
	teapExportLen   = 128
)

// TestTEAPEndToEnd drives a complete RFC 9930 TEAP conversation over live RADIUS
// UDP packets against the running auth server, exercising the acceptance gate of
// milestone M11.10: a hand-built peer opens the TLS 1.3 outer tunnel with the
// TEAP envelope, runs the Phase 2 plan (inner EAP-TLS, then Basic-Password-Auth
// when the prompt is configured), computes its own Compound MACs from the same
// exporters and BUFFER inputs, and checks the session key the server hands back.
//
// It proves:
//   - a valid inner EAP-TLS certificate plus a valid password produce an
//     Access-Accept carrying EAP-Success and MS-MPPE keys equal to the peer's own
//     RFC 9930 §6.4 derivation;
//   - the second Inner Method is bound by a Crypto-Binding TLV, including the
//     EMSK Compound MAC the EAP-TLS method requires (§6.2.3);
//   - a resumption from the TLS 1.3 ticket (§3.5) is granted on the credentials
//     §3.11 associated with it, without running any Inner Method;
//   - a wrong password rejects the session with EAP-Failure and no MPPE keys.
//
// It is intentionally serial (no t.Parallel) because the RADIUS plugin registry
// and dynamic settings are process-global shared state.
func TestTEAPEndToEnd(t *testing.T) {
	const secret = "it-teap-secret"
	suffix := uniqueSuffix()
	nasIP := net.ParseIP(uniqueNASIP())
	nasID := "it-teap-nas-" + suffix

	nas := &domain.NetNas{
		ID:         common.UUIDint64(),
		Identifier: nasID,
		Ipaddr:     nasIP.String(),
		Secret:     secret,
		VendorCode: "0",
		Status:     common.ENABLED,
	}
	require.NoError(t, h.appCtx.DB().Create(nas).Error)

	profileID := seedProfile(t, "it-teap-profile-"+suffix)
	serverAddr := h.radiusServerAddr()

	// Restore the EAP method after this test so later integration cases are not
	// affected by the runtime switch to eap-teap.
	restoreEapMethod(t)

	ca := newEAPTLSTestCA(t, "IT TEAP Root CA "+suffix)
	serverCert := ca.issueServer(t, "radius.example.com")
	const prompt = "Password for the TEAP test realm"

	t.Run("inner EAP-TLS and password authenticate with MPPE keys", func(t *testing.T) {
		username := "it-teap-alice-" + suffix + "@example.com"
		password := "it-teap-pass-" + suffix
		seedTEAPUser(t, profileID, username, password)
		configureTEAP(t, serverCert, ca.certPEM(), prompt)

		peer := newTEAPAcceptancePeer(t, serverAddr, secret, username, password, nasID, nasIP, ca, nil)
		resp := peer.run(t)
		require.Equalf(t, radius.CodeAccessAccept, resp.Code,
			"a valid TEAP peer must authenticate, got %v (%q)", resp.Code, rfc2865.ReplyMessage_GetString(resp))
		assertEAPCode(t, resp, eap.CodeSuccess)

		assert.Equal(t, 2, peer.methodsRun, "both Inner Methods of the plan must be bound")
		assert.Equal(t, prompt, peer.passwordPromptSeen,
			"§3.6.3 requires the first password request to carry the configured prompt")
		require.Len(t, peer.outerMSK, 64, "the peer must derive the §6.4 MSK itself")

		resp.Secret = []byte(secret)
		resp.Authenticator = peer.lastReqAuth
		recvKey, err := microsoft.MSMPPERecvKey_Lookup(resp)
		require.NoError(t, err, "Access-Accept must carry an MS-MPPE-Recv-Key")
		assert.Equal(t, peer.outerMSK[:32], recvKey)
		sendKey, err := microsoft.MSMPPESendKey_Lookup(resp)
		require.NoError(t, err, "Access-Accept must carry an MS-MPPE-Send-Key")
		assert.Equal(t, peer.outerMSK[32:64], sendKey)
	})

	t.Run("ticket resumption skips Phase 2", func(t *testing.T) {
		username := "it-teap-resume-" + suffix + "@example.com"
		password := "it-teap-resume-pass-" + suffix
		seedTEAPUser(t, profileID, username, password)
		// An empty prompt keeps the plan to the inner EAP-TLS method alone, so a
		// session that does resume cannot have run anything to hide behind.
		configureTEAP(t, serverCert, ca.certPEM(), "")
		cache := tls.NewLRUClientSessionCache(4)

		first := newTEAPAcceptancePeer(t, serverAddr, secret, username, password, nasID, nasIP, ca, cache)
		firstResp := first.run(t)
		require.Equalf(t, radius.CodeAccessAccept, firstResp.Code,
			"the first TEAP session must authenticate, got %v", firstResp.Code)
		assert.True(t, first.innerStarted, "the first session runs the Inner Method")
		assert.Equal(t, 1, first.methodsRun)
		require.Len(t, first.outerMSK, 64)

		second := newTEAPAcceptancePeer(t, serverAddr, secret, username, password, nasID, nasIP, ca, cache)
		secondResp := second.run(t)
		require.Equalf(t, radius.CodeAccessAccept, secondResp.Code,
			"a resumed TEAP session is granted without Phase 2, got %v", secondResp.Code)
		assert.False(t, second.innerStarted, "§3.5 bypasses Phase 2 entirely")
		assert.Equal(t, 0, second.methodsRun)

		secondResp.Secret = []byte(secret)
		secondResp.Authenticator = second.lastReqAuth
		recvKey, err := microsoft.MSMPPERecvKey_Lookup(secondResp)
		require.NoError(t, err)
		assert.Equal(t, first.outerMSK[:32], recvKey,
			"§3.11 resumes on the credentials the ticket was bound to")
		sendKey, err := microsoft.MSMPPESendKey_Lookup(secondResp)
		require.NoError(t, err)
		assert.Equal(t, first.outerMSK[32:64], sendKey)
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		username := "it-teap-mallory-" + suffix + "@example.com"
		password := "it-teap-correct-" + suffix
		seedTEAPUser(t, profileID, username, password)
		configureTEAP(t, serverCert, ca.certPEM(), prompt)

		peer := newTEAPAcceptancePeer(t, serverAddr, secret, username, password+"-wrong", nasID, nasIP, ca, nil)
		resp := peer.run(t)
		require.Equalf(t, radius.CodeAccessReject, resp.Code,
			"a wrong TEAP password must be rejected, got %v", resp.Code)
		assertEAPCode(t, resp, eap.CodeFailure)
		assert.Equal(t, uint32(1003), peer.lastErrorCode,
			"§4.2.6 error 1003 generalizes the failure so usernames cannot be enumerated")

		resp.Secret = []byte(secret)
		resp.Authenticator = peer.lastReqAuth
		_, err := microsoft.MSMPPERecvKey_Lookup(resp)
		assert.ErrorIs(t, err, radius.ErrNoAttribute, "rejects must not carry MPPE keys")
	})
}

// seedTEAPUser creates an enabled RADIUS user whose plaintext password backs the
// Basic-Password-Auth exchange (the default provider returns it as-is).
func seedTEAPUser(t *testing.T, profileID int64, username, password string) {
	t.Helper()
	user := &domain.RadiusUser{
		ID:         common.UUIDint64(),
		ProfileId:  profileID,
		Username:   username,
		Password:   password,
		Status:     common.ENABLED,
		ExpireTime: time.Now().AddDate(1, 0, 0),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	require.NoError(t, h.appCtx.DB().Create(user).Error)
}

// configureTEAP points the server at the managed TEAP certificates and switches
// it to eap-teap. The inherited outer and inner configurations both resolve the
// same managed server certificate; the client CA is what lets the inner EAP-TLS
// method authenticate the peer. An empty prompt keeps Basic-Password-Auth out of
// the Phase 2 plan.
func configureTEAP(t *testing.T, serverCert tls.Certificate, caPEM []byte, prompt string) {
	t.Helper()
	serverName := seedManagedServerCert(t, serverCert)
	caName := seedManagedCA(t, caPEM)

	cm := h.appCtx.ConfigMgr()
	require.NoError(t, cm.Set("radius", eaphandlers.SettingEapTlsServerCert, serverName))
	require.NoError(t, cm.Set("radius", eaphandlers.SettingEapTlsClientCa, caName))
	require.NoError(t, cm.Set("radius", "EapMethod", "eap-teap"))
	require.NoError(t, cm.Set("radius", eaphandlers.SettingEapTeapPasswordPrompt, prompt))
	t.Cleanup(func() {
		_ = cm.Set("radius", eaphandlers.SettingEapTeapPasswordPrompt, "")
	})
}

// --- peer-side TLS client -------------------------------------------------

// teapExporter adapts a completed crypto/tls connection to the exporter shape
// teapkeys derives the RFC 9427 §2.2 keys with.
type teapExporter struct {
	state *tls.ConnectionState
}

// ExportKey returns length octets of exported keying material (RFC 5705).
func (e teapExporter) ExportKey(label string, context []byte, length int) ([]byte, error) {
	return e.state.ExportKeyingMaterial(label, context, length)
}

// teapPeerTLS is one crypto/tls client of the acceptance peer: what it writes is
// what the peer sends and what is written to it is what the server sent.
type teapPeerTLS struct {
	conn     *tls.Conn
	to       *eapStream
	from     *eapStream
	done     chan error
	finished bool
	err      error
}

// newTEAPPeerTLS creates the client and closes its streams when the test ends, so
// a client goroutine blocked in Read cannot outlive it.
func newTEAPPeerTLS(t *testing.T, cfg *tls.Config) *teapPeerTLS {
	t.Helper()
	to, from := newEAPStream(), newEAPStream()
	t.Cleanup(func() {
		to.close()
		from.close()
	})
	return &teapPeerTLS{
		conn: tls.Client(&eapConn{rd: to, wr: from}, cfg),
		to:   to,
		from: from,
		done: make(chan error, 1),
	}
}

// poll reports whether the client goroutine has exited, recording its error.
func (c *teapPeerTLS) poll() bool {
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

// hasOutput reports whether the client wrote bytes the server has not read.
func (c *teapPeerTLS) hasOutput() bool {
	c.from.mu.Lock()
	defer c.from.mu.Unlock()
	return len(c.from.buf) > 0
}

// idle reports whether the client consumed everything the server sent and is
// blocked waiting for more.
func (c *teapPeerTLS) idle() bool {
	c.to.mu.Lock()
	defer c.to.mu.Unlock()
	return len(c.to.buf) == 0 && c.to.reading
}

// awaitOutput waits until the client either wrote a flight or is idle. An idle
// client is a legitimate answer: the TLS layer may consume a session ticket
// without producing anything the EAP layer must carry.
func (c *teapPeerTLS) awaitOutput() ([]byte, error) {
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

// takeFlight waits for one complete client flight.
func (c *teapPeerTLS) takeFlight() ([]byte, error) {
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

// --- the TEAP peer --------------------------------------------------------

// teapAcceptancePeer drives a complete TEAP conversation over RADIUS UDP: the
// outer TLS 1.3 tunnel inside the TEAP envelope, the Inner Methods the server's
// plan asks for, and the Intermediate-Result, Crypto-Binding and Result TLVs
// that close them. It computes its own Compound MACs from the same exporters and
// BUFFER inputs the server uses, so a successful run is evidence the two sides
// agree on RFC 9930 §6.3.
type teapAcceptancePeer struct {
	t          *testing.T
	serverAddr string
	secret     string
	username   string
	password   string
	nasID      string
	nasIP      net.IP

	outerCfg *tls.Config
	innerCfg *tls.Config
	outer    *teapPeerTLS
	inner    *teapPeerTLS

	// innerStarted records whether the Inner Method ran at all, which a resumed
	// session (§3.5) must not.
	innerStarted bool

	state       []byte
	respIdent   uint8
	lastReqAuth [16]byte

	peerOuter   []byte
	serverOuter []byte

	chain      *teapkeys.Chain
	binding    teapbind.Exchange
	exporter   teapkeys.Exporter
	methodsRun int
	// outerMSK is the peer's own §6.4 derivation of the granted MSK, computed when
	// the final Crypto-Binding exchange closes the plan.
	outerMSK []byte
	// passwordPromptSeen is the Prompt of the last Basic-Password-Auth-Req.
	passwordPromptSeen string
	// lastErrorCode is the Error-Code the server reported through an Error TLV.
	lastErrorCode uint32

	outQueue []*teapmsg.Packet
	inBuf    []byte
}

// newTEAPAcceptancePeer builds a peer that authenticates as username with
// password. The inner certificate carries the username as its CommonName so the
// RFC 5216 §5.2 identity binding matches the inner EAP identity. A non-nil cache
// lets successive peers resume the TLS 1.3 session (§3.5).
func newTEAPAcceptancePeer(t *testing.T, serverAddr, secret, username, password, nasID string, nasIP net.IP, ca *eapTLSTestCA, cache tls.ClientSessionCache) *teapAcceptancePeer {
	t.Helper()

	peerOuter, err := teaptlv.Marshal(teaptlv.NewAuthorityID([]byte("peer-hint")))
	require.NoError(t, err)

	return &teapAcceptancePeer{
		t:          t,
		serverAddr: serverAddr,
		secret:     secret,
		username:   username,
		password:   password,
		nasID:      nasID,
		nasIP:      nasIP,
		outerCfg: &tls.Config{
			RootCAs:            ca.pool(),
			ServerName:         "radius.example.com",
			MinVersion:         tls.VersionTLS13,
			ClientSessionCache: cache,
		},
		innerCfg: &tls.Config{
			Certificates: []tls.Certificate{ca.issueClient(t, username, username)},
			RootCAs:      ca.pool(),
			ServerName:   "radius.example.com",
			MinVersion:   tls.VersionTLS13,
			MaxVersion:   tls.VersionTLS13,
		},
		peerOuter: peerOuter,
	}
}

// run performs the EAP-Response/Identity round, then drives the conversation
// until the server grants or rejects, returning the final RADIUS reply.
func (p *teapAcceptancePeer) run(t *testing.T) *radius.Packet {
	t.Helper()

	p.sendIdentity(t)

	p.outer = newTEAPPeerTLS(t, p.outerCfg)
	go func() { p.outer.done <- p.outerSession() }()

	hello, err := p.outer.takeFlight()
	require.NoError(t, err)
	require.NotEmpty(t, hello, "the TLS client must produce a ClientHello")
	p.queueTLS(t, hello, p.peerOuter)

	for round := 0; round < 128; round++ {
		resp := p.exchange(t, p.nextOutbound(t))
		if resp.Code == radius.CodeAccessAccept || resp.Code == radius.CodeAccessReject {
			return resp
		}
		require.Equalf(t, radius.CodeAccessChallenge, resp.Code,
			"unexpected RADIUS code %v during the TEAP exchange", resp.Code)

		msg, err := eap.ParseEAPMessage(resp)
		require.NoError(t, err)
		require.Equalf(t, uint8(eap.TypeTEAP), msg.Type, "expected EAP-TEAP, got EAP type %d", msg.Type)
		p.respIdent = msg.Identifier
		if st := rfc2865.State_Get(resp); len(st) > 0 {
			p.state = st
		}

		pkt, err := teapmsg.Parse(msg.Data)
		require.NoError(t, err)
		p.inBuf = append(p.inBuf, pkt.TLSData...)
		if pkt.More() {
			p.queueACK()
			continue
		}

		flight := p.inBuf
		p.inBuf = nil
		if len(flight) == 0 {
			// A server ACK of one of our fragments; the rest is already queued.
			if len(p.outQueue) == 0 {
				p.queueACK()
			}
			continue
		}
		_, err = p.outer.to.Write(flight)
		require.NoError(t, err)
		out, err := p.outer.awaitOutput()
		require.NoError(t, err)
		if len(out) == 0 {
			p.queueACK()
			continue
		}
		p.queueTLS(t, out, nil)
	}
	t.Fatal("the TEAP exchange did not finish within the round budget")
	return nil
}

// sendIdentity performs the opening EAP-Response/Identity round and captures the
// PEAP-style Start: its identifier, State, and the Outer TLVs §6.3 puts in the
// Crypto-Binding BUFFER.
func (p *teapAcceptancePeer) sendIdentity(t *testing.T) {
	t.Helper()
	identity := &eap.EAPMessage{
		Code:       eap.CodeResponse,
		Identifier: 1,
		Type:       eap.TypeIdentity,
		Data:       []byte(p.username),
	}
	packet := p.newAccessRequest()
	eap.SetEAPMessageAndAuth(packet, identity.Encode(), p.secret)
	p.lastReqAuth = packet.Authenticator

	resp, err := p.exchangeRaw(packet)
	require.NoError(t, err)
	require.Equalf(t, radius.CodeAccessChallenge, resp.Code,
		"expected Access-Challenge after identity, got %v", resp.Code)

	msg, err := eap.ParseEAPMessage(resp)
	require.NoError(t, err)
	require.Equalf(t, uint8(eap.TypeTEAP), msg.Type, "expected EAP-TEAP Start, got EAP type %d", msg.Type)

	start, err := teapmsg.Parse(msg.Data)
	require.NoError(t, err)
	require.True(t, start.Start(), "the server opens with a TEAP Start")
	p.serverOuter = append([]byte(nil), start.OuterTLVs...)
	require.NotEmpty(t, p.serverOuter, "the Start must carry the Authority-ID Outer TLV")

	p.state = rfc2865.State_Get(resp)
	require.NotEmpty(t, p.state, "the Start challenge is missing the State attribute")
	p.respIdent = msg.Identifier
}

// exchange sends one TEAP packet as an EAP-Response and records the Request
// Authenticator needed to decrypt the MS-MPPE keys on the final reply.
func (p *teapAcceptancePeer) exchange(t *testing.T, outbound *teapmsg.Packet) *radius.Packet {
	t.Helper()
	data, err := outbound.Encode()
	require.NoError(t, err)

	msg := &eap.EAPMessage{
		Code:       eap.CodeResponse,
		Identifier: p.respIdent,
		Type:       eap.TypeTEAP,
		Data:       data,
	}
	packet := p.newAccessRequest()
	require.NoError(t, rfc2865.State_Set(packet, p.state))
	eap.SetEAPMessageAndAuth(packet, msg.Encode(), p.secret)
	p.lastReqAuth = packet.Authenticator

	resp, err := p.exchangeRaw(packet)
	require.NoError(t, err)
	return resp
}

func (p *teapAcceptancePeer) newAccessRequest() *radius.Packet {
	packet := radius.New(radius.CodeAccessRequest, []byte(p.secret))
	_ = rfc2865.UserName_SetString(packet, p.username)   //nolint:errcheck
	_ = rfc2865.NASIdentifier_SetString(packet, p.nasID) //nolint:errcheck
	_ = rfc2865.NASIPAddress_Set(packet, p.nasIP)        //nolint:errcheck
	return packet
}

func (p *teapAcceptancePeer) exchangeRaw(packet *radius.Packet) (*radius.Packet, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return exchangeFromNAS(ctx, packet, p.serverAddr, p.nasIP.String())
}

// queueTLS fragments one peer message into TEAP packets.
func (p *teapAcceptancePeer) queueTLS(t *testing.T, data, outer []byte) {
	t.Helper()
	pkts, err := teapmsg.Fragment(teapmsg.Version, data, outer, 0)
	require.NoError(t, err)
	p.outQueue = append(p.outQueue, pkts...)
}

// queueACK queues the bare acknowledgment §4.1 uses between fragments.
func (p *teapAcceptancePeer) queueACK() {
	p.outQueue = append(p.outQueue, &teapmsg.Packet{Version: teapmsg.Version})
}

// nextOutbound returns the peer's next TEAP packet.
func (p *teapAcceptancePeer) nextOutbound(t *testing.T) *teapmsg.Packet {
	t.Helper()
	require.NotEmpty(t, p.outQueue, "the peer must always have a packet to answer with")
	pkt := p.outQueue[0]
	p.outQueue = p.outQueue[1:]
	return pkt
}

// outerSession is the outer client goroutine: the TLS handshake, then one
// decrypted TLV message at a time, answered with its TLVs.
func (p *teapAcceptancePeer) outerSession() error {
	c := p.outer
	if err := c.conn.Handshake(); err != nil {
		return fmt.Errorf("peer outer handshake: %w", err)
	}

	state := c.conn.ConnectionState()
	p.exporter = teapExporter{state: &state}
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
	p.binding = teapbind.Exchange{
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
func (p *teapAcceptancePeer) answerPhase2(tlvs []teaptlv.TLV) ([]byte, bool, error) {
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
func (p *teapAcceptancePeer) answerInner(payload *teaptlv.TLV) ([]byte, bool, error) {
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
func (p *teapAcceptancePeer) answerIdentity(msg *eap.EAPMessage) ([]byte, bool, error) {
	response := (&eap.EAPMessage{
		Code:       eap.CodeResponse,
		Identifier: msg.Identifier,
		Type:       eap.TypeIdentity,
		Data:       []byte(p.username),
	}).Encode()
	payload, err := teaptlv.NewEAPPayload(response)
	if err != nil {
		return nil, false, err
	}
	out, err := teaptlv.Marshal(teaptlv.NewIdentityType(teaptlv.IdentityUser), payload)
	return out, false, err
}

// answerInnerTLS drives the inner EAP-TLS client one flight at a time.
func (p *teapAcceptancePeer) answerInnerTLS(msg *eap.EAPMessage) ([]byte, bool, error) {
	frag, err := tlsfragment.Parse(msg.Data)
	if err != nil {
		return nil, false, err
	}
	switch {
	case frag.Start():
		if !p.innerStarted {
			p.inner = newTEAPPeerTLS(p.t, p.innerCfg)
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
func (p *teapAcceptancePeer) answerPassword(request *teaptlv.TLV) ([]byte, bool, error) {
	prompt, err := request.Prompt()
	if err != nil {
		return nil, false, err
	}
	p.passwordPromptSeen = prompt
	response, err := teaptlv.NewBasicPasswordAuthResp([]byte(p.username), []byte(p.password))
	if err != nil {
		return nil, false, err
	}
	out, err := teaptlv.Marshal(response)
	return out, false, err
}

// answerBinding replies to a Crypto-Binding request with the Intermediate-Result,
// the Binding Response and, when the server also sent a Result, the Result TLV
// (§3.6.2, §3.6.6).
func (p *teapAcceptancePeer) answerBinding(tlvs []teaptlv.TLV, bindingTLV *teaptlv.TLV) ([]byte, bool, error) {
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
	answer, err := p.binding.Bind(
		teapbind.Keys{MSKCMK: mskCMK, EMSKCMK: emskCMK},
		flags, teaptlv.SubTypeBindingResponse, nonce,
	)
	if err != nil {
		return nil, false, err
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
func (p *teapAcceptancePeer) advanceChain() error {
	if p.methodsRun == 0 {
		if err := p.awaitInnerHandshake(); err != nil {
			return err
		}
		state := p.inner.conn.ConnectionState()
		keyMaterial, err := state.ExportKeyingMaterial(teapExportLabel, []byte{eap.TypeTLS}, teapExportLen)
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

// awaitInnerHandshake waits for the inner TLS client to finish.
func (p *teapAcceptancePeer) awaitInnerHandshake() error {
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

// parseTEAPInnerEAP decodes the EAP packet carried inside an EAP-Payload TLV
// (RFC 3748 §4: Code, Identifier, Length, Type, Data).
func parseTEAPInnerEAP(packet []byte) (*eap.EAPMessage, error) {
	if len(packet) < 5 {
		return nil, fmt.Errorf("inner EAP payload is only %d octets", len(packet))
	}
	if length := int(binary.BigEndian.Uint16(packet[2:4])); length != len(packet) {
		return nil, fmt.Errorf("inner EAP length %d does not match %d octets", length, len(packet))
	}
	msg := &eap.EAPMessage{Code: packet[0], Identifier: packet[1], Type: packet[4]}
	if len(packet) > 5 {
		msg.Data = packet[5:]
	}
	if msg.Code != eap.CodeRequest {
		return nil, errors.New("inner EAP payload is a response, not a request")
	}
	return msg, nil
}
