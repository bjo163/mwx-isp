package handlers

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/statemanager"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/microsoft"
	"layeh.com/radius/rfc2759"
)

// --- end-to-end inner CHAP tests (RFC 5281 §11.2.2 / RFC 1994) --------------

// encodeTTLSCHAP builds the inner CHAP AVP flight an EAP-TTLS client tunnels:
// User-Name, CHAP-Challenge and CHAP-Password. The challenge and Identifier are
// derived from the peer's TLS session exactly as the server derives them.
func encodeTTLSCHAP(conn *tls.Conn, username, password string, tamperChallenge, tamperIdent bool) []byte {
	cs := conn.ConnectionState()
	chal, err := cs.ExportKeyingMaterial(ttlsChallengeLabel, nil, ttlsCHAPChallengeLen)
	if err != nil {
		panic(err)
	}
	challenge, ident := chal[:16], chal[16]
	sentChallenge := challenge
	if tamperChallenge {
		sentChallenge = make([]byte, 16)
	}
	sentIdent := ident
	if tamperIdent {
		sentIdent++
	}
	value := append([]byte{sentIdent}, chapResponse(sentIdent, []byte(password), challenge)...)

	var flight []byte
	flight = append(flight, buildTTLSAVP(ttlsAVPCodeUserName, true, 0, []byte(username))...)
	flight = append(flight, buildTTLSAVP(ttlsAVPCodeCHAPChallenge, true, 0, sentChallenge)...)
	flight = append(flight, buildTTLSAVP(ttlsAVPCodeCHAPPassword, true, 0, value)...)
	return flight
}

func TestTTLSHandler_FullCHAPAuth_Succeeds(t *testing.T) {
	ca := newHSTestCA(t, "TTLS CHAP Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSCHAP(conn, "ttlsuser", "S3cr3t!", false, false)
	}
	success, err := peer.run()
	require.NoError(t, err)
	assert.True(t, success, "EAP-TTLS inner CHAP with a correct password must authenticate")

	require.NotNil(t, peer.acceptResponse)
	_, rerr := microsoft.MSMPPERecvKey_Lookup(peer.acceptResponse)
	assert.NoError(t, rerr, "MS-MPPE-Recv-Key must be present on the Access-Accept")
	_, serr := microsoft.MSMPPESendKey_Lookup(peer.acceptResponse)
	assert.NoError(t, serr, "MS-MPPE-Send-Key must be present on the Access-Accept")
}

func TestTTLSHandler_CHAPAuth_WrongPasswordRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS CHAP WrongPwd Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSCHAP(conn, "ttlsuser", "Wr0ng-Passw0rd", false, false)
	}
	success, err := peer.run()
	assert.False(t, success, "a wrong CHAP response must not authenticate")
	assert.ErrorIs(t, err, eap.ErrPasswordMismatch)
}

func TestTTLSHandler_CHAPAuth_TamperedChallengeRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS CHAP Tamper Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSCHAP(conn, "ttlsuser", "S3cr3t!", true, false)
	}
	success, err := peer.run()
	assert.False(t, success, "a CHAP-Challenge that differs from the derived material must reject")
	assert.ErrorIs(t, err, eap.ErrTTLSInnerProtocol)
}

func TestTTLSHandler_CHAPAuth_LDAPVerifierRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS CHAP LDAP Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.verifier = &fakeTTLSVerifier{active: true}
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSCHAP(conn, "ttlsuser", "S3cr3t!", false, false)
	}
	success, err := peer.run()
	assert.False(t, success, "a directory bind cannot verify CHAP")
	assert.ErrorIs(t, err, eap.ErrTTLSInnerNotImplemented)
}

// --- end-to-end inner MS-CHAP tests (RFC 5281 §11.2.3 / RFC 2433) -----------

// encodeTTLSMSCHAP builds the inner MS-CHAP AVP flight: User-Name,
// MS-CHAP-Challenge and MS-CHAP-Response. lmOnly clears the NT-Response flag
// and zeroes the NT-Response, as a client that only computed the LM response.
func encodeTTLSMSCHAP(conn *tls.Conn, username, password string, lmOnly, tamperIdent bool) []byte {
	cs := conn.ConnectionState()
	chal, err := cs.ExportKeyingMaterial(ttlsChallengeLabel, nil, ttlsMSCHAPChallengeLen)
	if err != nil {
		panic(err)
	}
	challenge, ident := chal[:8], chal[8]
	sentIdent := ident
	if tamperIdent {
		sentIdent++
	}

	value := make([]byte, ttlsMSCHAPResponseLen)
	value[0] = sentIdent
	if lmOnly {
		value[1] = 0
	} else {
		value[1] = ttlsMSCHAPFlagUseNTResponse
		ucs2Password, err := rfc2759.ToUTF16([]byte(password))
		if err != nil {
			panic(err)
		}
		copy(value[26:50], rfc2759.ChallengeResponse(challenge, rfc2759.NTPasswordHash(ucs2Password)))
	}

	var flight []byte
	flight = append(flight, buildTTLSAVP(ttlsAVPCodeUserName, true, 0, []byte(username))...)
	flight = append(flight, buildTTLSAVP(ttlsMSCHAPChallengeCode, true, ttlsVendorMicrosoft, challenge)...)
	flight = append(flight, buildTTLSAVP(ttlsMSCHAPResponseCode, true, ttlsVendorMicrosoft, value)...)
	return flight
}

func TestTTLSHandler_FullMSCHAPAuth_Succeeds(t *testing.T) {
	ca := newHSTestCA(t, "TTLS MSCHAP Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSMSCHAP(conn, "ttlsuser", "S3cr3t!", false, false)
	}
	success, err := peer.run()
	require.NoError(t, err)
	assert.True(t, success, "EAP-TTLS inner MS-CHAP with a correct password must authenticate")

	require.NotNil(t, peer.acceptResponse)
	_, rerr := microsoft.MSMPPERecvKey_Lookup(peer.acceptResponse)
	assert.NoError(t, rerr, "MS-MPPE-Recv-Key must be present on the Access-Accept")
}

func TestTTLSHandler_MSCHAPAuth_WrongPasswordRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS MSCHAP WrongPwd Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSMSCHAP(conn, "ttlsuser", "Wr0ng-Passw0rd", false, false)
	}
	success, err := peer.run()
	assert.False(t, success, "a wrong MS-CHAP NT-Response must not authenticate")
	assert.ErrorIs(t, err, eap.ErrPasswordMismatch)
}

func TestTTLSHandler_MSCHAPAuth_LMOnlyRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS MSCHAP LM Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSMSCHAP(conn, "ttlsuser", "S3cr3t!", true, false)
	}
	success, err := peer.run()
	assert.False(t, success, "an LM-only MS-CHAP response is not verifiable here")
	assert.ErrorIs(t, err, eap.ErrTTLSInnerNotImplemented)
}

func TestTTLSHandler_MSCHAPAuth_TamperedIdentRejected(t *testing.T) {
	ca := newHSTestCA(t, "TTLS MSCHAP Tamper Root CA")
	cfg := ttlsServerEngineConfig(t, ca)
	h := NewTTLSHandlerWithConfig(func() (*tlsengine.Config, error) { return cfg, nil })

	sm := statemanager.NewMemoryStateManager()
	defer sm.Close()

	stateID := startHandshake(t, h, sm, "ttlsuser", "secret")
	peer := newTTLSPapPeer(t, h, ca, sm, stateID, "secret", "ttlsuser", "S3cr3t!")
	peer.buildFlight = func(conn *tls.Conn) []byte {
		return encodeTTLSMSCHAP(conn, "ttlsuser", "S3cr3t!", false, true)
	}
	success, err := peer.run()
	assert.False(t, success, "an Ident that differs from the derived material must reject")
	assert.ErrorIs(t, err, eap.ErrTTLSInnerProtocol)
}
