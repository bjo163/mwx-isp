package teapbind

import (
	"bytes"
	"crypto"
	"crypto/hmac"
	_ "crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
)

// The two specifications this package implements, read from the copies the repository
// carries rather than from memory.
const (
	rfc9930Path = "../../../../../docs/rfcs/rfc9930-teap-version-1.txt"
	rfc9427Path = "../../../../../docs/rfcs/rfc9427-tls-based-eap-types-tls13.txt"
)

// requestNonce is a Binding Request nonce: 32 octets whose least significant bit is
// clear, as §4.2.13 requires.
func requestNonce() []byte {
	nonce := make([]byte, 32)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	nonce[31] = 0x10
	return nonce
}

// outer is the Phase 1 material the test session exchanged before the tunnel.
func outer() OuterTLVs {
	return OuterTLVs{
		Server: []byte{0x00, 0x01, 0x00, 0x04, 0x00, 0x00, 0x00, 0x2a},
		Peer:   []byte{0x80, 0x09, 0x00, 0x03, 'm', 'w', 'x'},
	}
}

// exchange is a TLS 1.3 session bound with SHA-256 over the TEAP Type the other party
// sent.
func exchange() Exchange {
	return Exchange{Hash: crypto.SHA256, OtherPartyType: teapkeys.Type, Outer: outer()}
}

// secret returns n octets that differ from each other and from zero.
func secret(n int) []byte {
	out := make([]byte, n, n+1)
	for i := range len(out) {
		out[i] = byte(i + 1)
	}
	return out
}

// cmk returns a distinguishable 20-octet Compound MAC key.
func cmk(seed byte) []byte {
	out := make([]byte, compoundMACLen)
	for i := range out {
		out[i] = seed + byte(i)
	}
	return out
}

// filledTLV builds a Crypto-Binding TLV whose announced MAC slots carry recognizable
// non-zero octets, for the tests that must see that region disappear.
func filledTLV(t *testing.T, nonce []byte, flags, subType uint8) teaptlv.TLV {
	t.Helper()
	var emsk, msk []byte
	if flags&teaptlv.FlagEMSKPresent != 0 {
		emsk = cmk(0xa0)
	}
	if flags&teaptlv.FlagMSKPresent != 0 {
		msk = cmk(0x50)
	}
	tlv, err := teaptlv.NewCryptoBinding(teaptlv.Version, flags, subType, nonce, emsk, msk)
	require.NoError(t, err)
	return tlv
}

// zeroSlot returns a run of MAC-length zeros when Flags announces the slot and nothing
// when it does not, which is what teaptlv encodes for a field waiting to be filled.
func zeroSlot(present bool) []byte {
	if !present {
		return nil
	}
	return make([]byte, compoundMACLen)
}

// assertAllBytesZero checks that every octet of a region is zero, which is what §6.3
// asks of a covered TLV and §4.2.13 of a slot Flags does not announce. testify's own
// Zero compares against the zero value of the type, so a filled slice of zeros is not
// "zero" to it.
func assertAllBytesZero(t *testing.T, got []byte, msg string) {
	t.Helper()
	assert.Equal(t, make([]byte, len(got)), got, msg)
}

// readSpec loads one vendored specification.
func readSpec(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // G304: one of the two paths pinned as constants above, under docs/rfcs.
	require.NoError(t, err, "read %s", path)
	return string(raw)
}

// collapseSpace joins the line-wrapped prose of an RFC so a quoted sentence matches
// however the text file breaks its lines.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func TestHashFromCipherSuite(t *testing.T) {
	for _, tc := range []struct {
		name  string
		suite uint16
		want  crypto.Hash
	}{
		{"aes-128-gcm", tls.TLS_AES_128_GCM_SHA256, crypto.SHA256},
		{"aes-256-gcm", tls.TLS_AES_256_GCM_SHA384, crypto.SHA384},
		{"chacha20-poly1305", tls.TLS_CHACHA20_POLY1305_SHA256, crypto.SHA256},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := HashFromCipherSuite(tc.suite)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	// A TLS 1.2 suite negotiates a PRF, not the HKDF hash RFC 9427 §2.2 makes the
	// Compound MAC, so it is refused rather than mapped to a plausible hash.
	for _, suite := range []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA, tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, 0x0000} {
		_, err := HashFromCipherSuite(suite)
		assert.ErrorIs(t, err, ErrUnsupportedCipherSuite, "0x%04x is not a TLS 1.3 cipher suite", suite)
	}
}

func TestBufferConcatenatesTheFieldsInTheSpecifiedOrder(t *testing.T) {
	o := outer()
	tlv := filledTLV(t, requestNonce(), teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)

	buffer, err := exchange().Buffer(tlv)
	require.NoError(t, err)

	wire, err := tlv.Bytes()
	require.NoError(t, err)
	var want bytes.Buffer
	want.Write(wire[:macFieldOffset])
	want.Write(make([]byte, 2*compoundMACLen))
	want.WriteByte(teapkeys.Type)
	want.Write(o.Server)
	want.Write(o.Peer)
	assert.Equal(t, want.Bytes(), buffer, "the BUFFER is the §6.3 fields in the order listed")
	assert.Len(t, buffer, bindingTLVLen+1+len(o.Server)+len(o.Peer))
}

func TestBufferZeroesBothMACFieldsOfATLVThatCarriesThem(t *testing.T) {
	// RFC 9930 §6.3 computes the MAC over the TLV "with both the EMSK and MSK Compound
	// MAC fields zeroed out" and states that during the calculation "the MAC field is
	// filled with zeros", so a received MAC never feeds its own computation, nor the
	// other slot's.
	tlv := filledTLV(t, requestNonce(), teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)
	wire, err := tlv.Bytes()
	require.NoError(t, err)
	require.NotEmpty(t, bytes.Trim(wire[macFieldOffset:bindingTLVLen], "\x00"), "the fixture carries real MAC octets")

	buffer, err := exchange().Buffer(tlv)
	require.NoError(t, err)
	assertAllBytesZero(t, buffer[macFieldOffset:bindingTLVLen], "both Compound MAC fields are zero for the whole computation")
	assert.Equal(t, wire[:macFieldOffset], buffer[:macFieldOffset], "everything before the MAC fields is as encoded")
}

func TestBufferCoversTheTLVAsReceived(t *testing.T) {
	x := exchange()
	tlv := filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)

	plain, err := x.Buffer(tlv)
	require.NoError(t, err)

	// The Mandatory bit is covered octet, not metadata: normalising it would let a peer
	// flip a flag the other side never saw and still agree on the MAC.
	nonMandatory := tlv
	nonMandatory.Mandatory = false
	covered, err := x.Buffer(nonMandatory)
	require.NoError(t, err)
	assert.NotEqual(t, plain, covered)
	assert.Equal(t, plain[1:], covered[1:], "the two buffers differ only in the header octet that carries the bit")
}

func TestBufferRejectsWhatIsNotTheFirstTEAPExchange(t *testing.T) {
	x := exchange()

	_, err := x.Buffer(teaptlv.NewResult(teaptlv.StatusSuccess))
	assert.ErrorIs(t, err, ErrNotCryptoBinding)

	peerSentPEAP := x
	peerSentPEAP.OtherPartyType = 25
	_, err = peerSentPEAP.Buffer(filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest))
	assert.ErrorIs(t, err, ErrWrongEAPType)

	// A Value shorter than §4.2.13 fixes has no defined position for the MAC fields, so
	// the fields cannot be zeroed and the TLV cannot be covered.
	short := filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)
	short.Value = short.Value[:75]
	_, err = x.Buffer(short)
	assert.ErrorIs(t, err, ErrTLVLength)

	// A Value the two-octet Length field cannot even count never reaches the zeroing:
	// the TLV has no encoding, so there are no octets to cover.
	oversized := teaptlv.TLV{Type: teaptlv.TypeCryptoBinding, Mandatory: true, Value: make([]byte, 1<<16)}
	_, err = x.Buffer(oversized)
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV)
}

func TestCompoundMACTruncatesToTheFieldWidth(t *testing.T) {
	// The reference is the plain HMAC, computed here rather than through the package:
	// SHA-256 gives 32 octets and SHA-384 gives 48, and §4.2.13 keeps the first 20 of
	// either.
	buffer := []byte("the §6.3 BUFFER")
	key := cmk(0x10)
	for _, tc := range []struct {
		hash crypto.Hash
		size int
	}{
		{crypto.SHA256, sha256.Size},
		{crypto.SHA384, 48},
	} {
		mac := hmac.New(tc.hash.New, key)
		mac.Write(buffer)
		full := mac.Sum(nil)
		require.Len(t, full, tc.size)

		got, err := CompoundMAC(tc.hash, key, buffer)
		require.NoError(t, err)
		assert.Equal(t, full[:compoundMACLen], got, "%s truncated to the field width", tc.hash)
	}
}

func TestCompoundMACRefusesAKeyOrHashItCannotUse(t *testing.T) {
	_, err := CompoundMAC(crypto.SHA256, nil, []byte("buffer"))
	assert.ErrorIs(t, err, ErrNoCompoundKey, "no CMK means no Compound MAC")

	// A hash that is not linked in cannot serve, and one whose output cannot fill the
	// field either: §4.2.13 needs 20 octets and a 16-octet MAC cannot be stretched.
	_, err = CompoundMAC(crypto.Hash(99), cmk(0x10), []byte("buffer"))
	assert.ErrorIs(t, err, ErrUnsupportedHash)
	_, err = CompoundMAC(crypto.MD5, cmk(0x10), []byte("buffer"))
	assert.ErrorIs(t, err, ErrUnsupportedHash)
}

func TestBindFillsExactlyTheSlotsTheFlagsAnnounce(t *testing.T) {
	x := exchange()
	nonce := requestNonce()
	keys := Keys{MSKCMK: cmk(0x10), EMSKCMK: cmk(0x20)}

	for _, tc := range []struct {
		name    string
		flags   uint8
		hasEMSK bool
		hasMSK  bool
	}{
		{"emsk only", teaptlv.FlagEMSKPresent, true, false},
		{"msk only", teaptlv.FlagMSKPresent, false, true},
		{"both", teaptlv.FlagEMSKPresent | teaptlv.FlagMSKPresent, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tlv, err := x.Bind(keys, tc.flags, teaptlv.SubTypeBindingRequest, nonce)
			require.NoError(t, err)
			cb, err := tlv.CryptoBinding()
			require.NoError(t, err)
			assert.Equal(t, tc.flags, cb.Flags)
			assert.Equal(t, nonce, cb.Nonce)

			// The BUFFER is the same TLV with the announced slots empty, so recomposing
			// it here checks Bind's output without reusing its code.
			empty, err := teaptlv.NewCryptoBinding(teaptlv.Version, tc.flags, teaptlv.SubTypeBindingRequest, nonce,
				zeroSlot(tc.hasEMSK), zeroSlot(tc.hasMSK))
			require.NoError(t, err)
			buffer, err := x.Buffer(empty)
			require.NoError(t, err)

			if tc.hasEMSK {
				want, err := CompoundMAC(x.Hash, keys.EMSKCMK, buffer)
				require.NoError(t, err)
				assert.Equal(t, want, cb.EMSKCompoundMAC)
			} else {
				assert.Nil(t, cb.EMSKCompoundMAC)
			}
			if tc.hasMSK {
				want, err := CompoundMAC(x.Hash, keys.MSKCMK, buffer)
				require.NoError(t, err)
				assert.Equal(t, want, cb.MSKCompoundMAC)
			} else {
				assert.Nil(t, cb.MSKCompoundMAC)
			}

			// The slot Flags does not announce stays zero on the wire rather than
			// carrying a MAC nobody asked for.
			wire, err := tlv.Bytes()
			require.NoError(t, err)
			if !tc.hasEMSK {
				assertAllBytesZero(t, wire[macFieldOffset:macFieldOffset+compoundMACLen], "the unannounced EMSK slot must stay zero")
			}
			if !tc.hasMSK {
				assertAllBytesZero(t, wire[macFieldOffset+compoundMACLen:], "the unannounced MSK slot must stay zero")
			}
		})
	}
}

func TestBindNeedsOneKeyPerAnnouncedSlot(t *testing.T) {
	x := exchange()
	nonce := requestNonce()

	_, err := x.Bind(Keys{MSKCMK: cmk(0x10)}, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent,
		teaptlv.SubTypeBindingRequest, nonce)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCompoundKey)
	assert.Contains(t, err.Error(), "EMSK Compound MAC", "the failure must name the slot that has no key")

	_, err = x.Bind(Keys{EMSKCMK: cmk(0x20)}, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, nonce)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNoCompoundKey)
	assert.Contains(t, err.Error(), "MSK Compound MAC")

	// No key at all is the state before any Inner Method ran, and it fails the same way
	// rather than binding a session whose keys do not exist yet.
	_, err = x.Bind(Keys{}, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, nonce)
	assert.ErrorIs(t, err, ErrNoCompoundKey)
}

func TestBindRejectsAnInvalidTLVBeforeComputingAnything(t *testing.T) {
	x := exchange()
	keys := Keys{MSKCMK: cmk(0x10)}

	// A Binding Response must echo the request nonce with its least significant bit set.
	// teaptlv owns that rule and Bind applies it before spending a MAC.
	_, err := x.Bind(keys, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingResponse, requestNonce())
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV)

	_, err = x.Bind(keys, 0, teaptlv.SubTypeBindingRequest, requestNonce())
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV, "Flags 0 is not one of the values §4.2.13 defines")

	// §6.3 step 2 binds a TEAP session, so a sender that does not hold the TEAP octet
	// from the other party's first message never gets to the MAC either - the check
	// fires on the sending side exactly as it does on the verifying one.
	_, err = Exchange{Hash: crypto.SHA256, OtherPartyType: 0x36, Outer: outer()}.
		Bind(keys, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	assert.ErrorIs(t, err, ErrWrongEAPType)
}

func TestVerifyAcceptsTheBindingRequestAndResponseOfOneSession(t *testing.T) {
	x := exchange()
	keys := Keys{MSKCMK: cmk(0x10), EMSKCMK: cmk(0x20)}

	// RFC 9930 §6.2.4's two steps, and the reason both sides can compute the same
	// values: each holds the first TEAP message in both directions, so the Outer TLVs
	// and the other party's EAP Type agree without any of them being re-sent.
	request, err := x.Bind(keys, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)
	require.NoError(t, x.Verify(keys, request))

	nonce, err := teaptlv.ResponseNonce(requestNonce())
	require.NoError(t, err)
	response, err := x.Bind(keys, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingResponse, nonce)
	require.NoError(t, err)
	require.NoError(t, x.Verify(keys, response))

	cb, err := response.CryptoBinding()
	require.NoError(t, err)
	assert.Equal(t, nonce, cb.Nonce)
	assert.False(t, bytes.Equal(request.Value, response.Value), "the two directions must not carry the same MAC")
}

func TestVerifyFollowsTheFlagsAndIgnoresWhatTheyDoNotAnnounce(t *testing.T) {
	x := exchange()
	keys := Keys{MSKCMK: cmk(0x10), EMSKCMK: cmk(0x20)}

	mskOnly, err := x.Bind(keys, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)
	require.NoError(t, x.Verify(keys, mskOnly))

	// §6.2.4: "No further action is taken by the server if an unexpected Compound MAC is
	// present." A peer that fills the EMSK slot while announcing only the MSK slot is
	// therefore still verified - the slot is outside what Flags claims, and it is zeroed
	// for the BUFFER anyway.
	wire, err := mskOnly.Bytes()
	require.NoError(t, err)
	require.Equal(t, byte(0), wire[macFieldOffset], "the unannounced slot starts empty")
	junk := append([]byte(nil), wire...)
	junk[macFieldOffset] = 0xff
	parsed, err := teaptlv.Parse(junk)
	require.NoError(t, err)
	require.Len(t, parsed, 1)
	require.NoError(t, x.Verify(keys, parsed[0]))

	// Announcing a slot this session has no key for is a fatal failure, not a pass.
	both, err := x.Bind(keys, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)
	require.NoError(t, x.Verify(keys, both))
	assert.ErrorIs(t, x.Verify(Keys{MSKCMK: cmk(0x10)}, both), ErrNoCompoundKey)
}

func TestVerifyRejectsAnyChangeToTheCoveredOctets(t *testing.T) {
	server := exchange()
	keys := Keys{MSKCMK: cmk(0x10), EMSKCMK: cmk(0x20)}
	request, err := server.Bind(keys, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent,
		teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)

	tamperedServerTLVs := outer()
	tamperedServerTLVs.Server = bytes.Replace(tamperedServerTLVs.Server, []byte{0x2a}, []byte{0x2b}, 1)
	tamperedPeerTLVs := outer()
	tamperedPeerTLVs.Peer = bytes.Replace(tamperedPeerTLVs.Peer, []byte("mwx"), []byte("Mwx"), 1)

	for _, tc := range []struct {
		name string
		peer Exchange
		want error
		slot string
	}{
		{
			name: "an Outer TLV the server sent changed in transit",
			peer: Exchange{Hash: crypto.SHA256, OtherPartyType: teapkeys.Type, Outer: tamperedServerTLVs},
			want: ErrCompoundMACMismatch,
			slot: "EMSK Compound MAC",
		},
		{
			name: "an Outer TLV the peer sent changed in transit",
			peer: Exchange{Hash: crypto.SHA256, OtherPartyType: teapkeys.Type, Outer: tamperedPeerTLVs},
			want: ErrCompoundMACMismatch,
			slot: "EMSK Compound MAC",
		},
		{
			// §6.3 step 2 requires the octet to be TEAP, so a session that is not TEAP
			// is refused before any MAC is computed rather than bound.
			name: "the first TEAP message carried another EAP Type",
			peer: Exchange{Hash: crypto.SHA256, OtherPartyType: 0x36, Outer: outer()},
			want: ErrWrongEAPType,
		},
		{
			// RFC 9427 §2.2 makes the MAC the TLS 1.3 HKDF hash, so a peer that picked a
			// different one from the same tunnel cannot be bound either.
			name: "the MAC algorithm is not the one the tunnel negotiated",
			peer: Exchange{Hash: crypto.SHA384, OtherPartyType: teapkeys.Type, Outer: outer()},
			want: ErrCompoundMACMismatch,
			slot: "EMSK Compound MAC",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.peer.Verify(keys, request)
			require.Error(t, err)
			assert.ErrorIs(t, err, tc.want)
			if tc.slot != "" {
				assert.Contains(t, err.Error(), tc.slot, "the failure must name the slot that disagreed")
			}
		})
	}

	// A key from another session is the other way the same check fires: the covered
	// octets match, and only the CMK disagrees.
	otherSession := Keys{MSKCMK: cmk(0x10), EMSKCMK: cmk(0x20)}
	otherSession.MSKCMK[0] ^= 0x01
	err = server.Verify(otherSession, request)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrCompoundMACMismatch)
	assert.Contains(t, err.Error(), "MSK Compound MAC")
}

func TestVerifyRefusesATLVThatFailsTheWireChecks(t *testing.T) {
	x := exchange()
	keys := Keys{MSKCMK: cmk(0x10)}

	// §4.2.13 lists the checks that make the TLV valid, and a MAC over an invalid TLV
	// proves nothing, so they run first.
	err := x.Verify(keys, teaptlv.NewResult(teaptlv.StatusSuccess))
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV)

	wrongVersion := filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)
	wrongVersion.Value[1] = 2
	assert.ErrorIs(t, x.Verify(keys, wrongVersion), teaptlv.ErrInvalidTLV)

	// A Sub-Type that does not match the nonce is the same class of defect: teaptlv
	// refuses to encode it, so editing the octet afterwards is the only way to hand one
	// to Verify. It must not spend a MAC on it either.
	misLabelled := filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)
	misLabelled.Value[3] = teaptlv.FlagMSKPresent<<4 | teaptlv.SubTypeBindingResponse
	assert.ErrorIs(t, x.Verify(keys, misLabelled), teaptlv.ErrInvalidTLV)

	// A TLV covered by a BUFFER this package cannot build is refused for that reason,
	// not reported as a mismatch. §4.2.13 fixes the Value at 76 octets, so a shorter one
	// has no MAC fields to read and fails the wire check before reaching Buffer.
	truncated := filledTLV(t, requestNonce(), teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest)
	truncated.Value = truncated.Value[:75]
	assert.ErrorIs(t, x.Verify(keys, truncated), teaptlv.ErrInvalidTLV)
}

func TestChainSuppliesTheKeysThisPackageNeeds(t *testing.T) {
	// M11.6 and M11.4 have to compose: the two CMKs teapkeys.Chain reports after an
	// Inner Method are exactly the Keys this package consumes, and a method that
	// produced both keys binds both slots.
	e := exporter{}
	seed, err := teapkeys.SessionKeySeed(e)
	require.NoError(t, err)
	chain, err := teapkeys.NewChain(seed)
	require.NoError(t, err)
	require.NoError(t, chain.Advance(e, teapkeys.InnerMethodKeys{MSK: secret(64), EMSK: secret(96)}))

	mskCMK, emskCMK := chain.CompoundMACKeys()
	require.Len(t, mskCMK, compoundMACLen)
	require.Len(t, emskCMK, compoundMACLen)
	assert.NotEqual(t, mskCMK, emskCMK, "the two derivations must not share a key")

	x := exchange()
	keys := Keys{MSKCMK: mskCMK, EMSKCMK: emskCMK}
	tlv, err := x.Bind(keys, teaptlv.FlagEMSKPresent|teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)
	require.NoError(t, x.Verify(keys, tlv))

	// A method that derives only an MSK leaves the EMSK path carried forward, so the
	// handshake announces the MSK slot alone and Bind has no EMSK key to want (§6.2.5).
	mskOnly, err := teapkeys.NewChain(seed)
	require.NoError(t, err)
	require.NoError(t, mskOnly.Advance(e, teapkeys.InnerMethodKeys{MSK: secret(64)}))
	mskCMK, emskCMK = mskOnly.CompoundMACKeys()
	require.Len(t, mskCMK, compoundMACLen)
	assert.Nil(t, emskCMK, "no method produced an EMSK, so no EMSK Compound MAC key exists")

	mskBound, err := x.Bind(Keys{MSKCMK: mskCMK}, teaptlv.FlagMSKPresent, teaptlv.SubTypeBindingRequest, requestNonce())
	require.NoError(t, err)
	require.NoError(t, x.Verify(Keys{MSKCMK: mskCMK}, mskBound))

	// Before any method ran the chain has no keys, and Bind says so instead of binding
	// an empty session.
	fresh, err := teapkeys.NewChain(seed)
	require.NoError(t, err)
	noneMSK, noneEMSK := fresh.CompoundMACKeys()
	assert.Nil(t, noneMSK)
	assert.Nil(t, noneEMSK)
	assert.ErrorIs(t, x.Verify(Keys{}, tlv), ErrNoCompoundKey)
}

// exporter stands in for the TLS 1.3 exporter: it expands label, context and length
// deterministically, so a chain that threads one derivation into the next yields
// different octets at every step.
type exporter struct{}

func (exporter) ExportKey(label string, context []byte, length int) ([]byte, error) {
	out := make([]byte, 0, length+sha256.Size)
	for counter := byte(0); len(out) < length; counter++ {
		h := sha256.New()
		h.Write([]byte(label))
		h.Write(context)
		h.Write([]byte{byte(length), counter}) //nolint:gosec // G115: these lengths are at most 96, so one octet holds them exactly.
		out = append(out, h.Sum(nil)...)
	}
	return out[:length], nil
}

// TestTheComputationMatchesTheVendoredClauses is the anti-fabrication gate: every
// field, offset and truncation rule this package implements is quoted from the text the
// repository carries, so a reorder of the BUFFER or a widened MAC field stops matching
// the clause it came from.
func TestTheComputationMatchesTheVendoredClauses(t *testing.T) {
	rfc9930 := collapseSpace(readSpec(t, rfc9930Path))
	for _, clause := range []string{
		"Compound MAC = the first 20 octets of MAC( CMK[n], BUFFER )",
		"During the calculation of the Compound MAC, the MAC field is filled with zeros.",
		"The entire Crypto-Binding TLV attribute with both the EMSK and MSK Compound MAC fields zeroed out.",
		"The EAP Type sent by the other party in the first TEAP message, which MUST be TEAP, encoded as one octet of 0x37.",
		"All the Outer TLVs from the first TEAP message sent by the EAP server to the peer.",
		"All the Outer TLVs from the first TEAP message sent by the peer to the EAP server.",
		"The Nonce field is 32 octets.",
		"The EMSK Compound MAC field is 20 octets.",
		"this field is always 20 octets in length. Any larger MAC is simply truncated.",
		"No further action is taken by the server if an unexpected Compound MAC is present.",
	} {
		assert.Contains(t, rfc9930, collapseSpace(clause), "RFC 9930 no longer states %q", clause)
	}

	rfc9427 := collapseSpace(readSpec(t, rfc9427Path))
	for _, clause := range []string{
		"Compound-MAC = MAC( CMK[n], BUFFER )",
		"negotiated for the HMAC-based Key Derivation Function (HKDF) in the key schedule",
		"CMK[j] = last 20 octets of IMCK[j]",
	} {
		assert.Contains(t, rfc9427, collapseSpace(clause), "RFC 9427 no longer states %q", clause)
	}

	// The offsets the clauses fix, restated as one invariant so a silent change of a
	// field width breaks here first.
	assert.Equal(t, 80, bindingTLVLen, "§4.2.13 fixes the TLV at 4 octets of header plus the 76-octet Value")
	assert.Equal(t, 40, macFieldOffset, "the MAC fields begin after the header, four octets and the 32-octet Nonce")
	assert.Equal(t, teapkeys.Type, uint8(0x37), "the BUFFER carries the TEAP octet §6.3 names")
}
