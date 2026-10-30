// Package teapbind computes the TEAP Crypto-Binding Compound MAC of RFC 9930 §6.3,
// the value that proves the peer on the other side of the TLS tunnel is the same
// peer that ran the Inner Method.
//
// The TLV is teaptlv's wire format and the key is teapkeys' CMK[n]; this package is
// where they meet. A Compound MAC covers the entire Crypto-Binding TLV with both of
// its MAC fields zeroed, the EAP Type the other party sent in the first TEAP
// message, and the Outer TLVs exchanged in both directions before the tunnel was
// built. Those Outer TLVs are Phase 1 clear-text octets the tunnel endpoint never
// sees, so any modification of them in transit makes the two sides compute different
// values — which is the on-path attack RFC 9930 §6.3 exists to detect.
//
// MAC(CMK[n], BUFFER) is HMAC with the hash the TLS 1.3 key schedule negotiated for
// HKDF (RFC 9427 §2.2), and RFC 9930 §4.2.13 fixes each field at 20 octets, so a
// longer MAC is truncated here and only the truncated value may be compared.
//
// Nothing here is session state. Which Inner Method completed, which Compound MACs
// the site policy of RFC 9930 §6.2.4 requires, and when the TLV goes out all belong
// to the TEAP handshake, and the caller refreshes the keys from its teapkeys.Chain
// after every method. A tunnel that negotiated TLS 1.2 must not use this package:
// RFC 9427 §2.2 changes the MAC to the TLS 1.3 HKDF hash, and the BUFFER is the
// same in both versions only because it never depends on the TLS version itself.
package teapbind

import (
	"crypto"
	"crypto/hmac"
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
)

// compoundMACLen is the width of one Compound MAC field. RFC 9930 §4.2.13 states it
// is always 20 octets and that a larger MAC "is simply truncated", with every
// comparison made on the truncated value.
const compoundMACLen = 20

// The offsets inside an encoded Crypto-Binding TLV. §4.2.1 puts a four-octet
// Type/Length header in front of the Value, and §4.2.13 orders the Value as one
// Reserved, one Version, one Received-Ver and one Flags|Sub-Type octet, then the
// 32-octet Nonce, then the two 20-octet MAC fields — so everything from
// macFieldOffset on is MAC material, and §6.3 zeroes all of it. macValueOffset is the
// same boundary counted inside the Value alone, which is where Bind writes to.
const (
	macFieldOffset = teaptlv.HeaderLen + 4 + 32
	macValueOffset = macFieldOffset - teaptlv.HeaderLen
	bindingTLVLen  = macFieldOffset + 2*compoundMACLen
)

var (
	// ErrNotCryptoBinding means the TLV handed to the MAC is not the one §6.3 covers.
	ErrNotCryptoBinding = errors.New("teapbind: not a Crypto-Binding TLV")
	// ErrTLVLength means the TLV is not the 80 octets §4.2.13 fixes, so the two MAC
	// fields have no defined position to zero.
	ErrTLVLength = errors.New("teapbind: Crypto-Binding TLV length mismatch")
	// ErrWrongEAPType means the first TEAP message did not carry TEAP. §6.3 step 2
	// says the octet "MUST be TEAP", and teapkeys.Type is the only value allowed.
	ErrWrongEAPType = errors.New("teapbind: first TEAP message carried a non-TEAP EAP Type")
	// ErrNoCompoundKey means a Compound MAC was asked for while the CMK that covers
	// it is missing, which is the state before any Inner Method has been folded into
	// the chain (RFC 9930 §6.2.2).
	ErrNoCompoundKey = errors.New("teapbind: no Compound MAC key")
	// ErrUnsupportedHash means the hash cannot serve as the TLS 1.3 HKDF MAC: it is
	// not linked in, or its output is shorter than the 20 octets §4.2.13 needs.
	ErrUnsupportedHash = errors.New("teapbind: hash is not usable as the TLS 1.3 HMAC")
	// ErrUnsupportedCipherSuite means the suite is not one of the TLS 1.3 cipher
	// suites, so it negotiates no HKDF hash for the key schedule.
	ErrUnsupportedCipherSuite = errors.New("teapbind: cipher suite negotiates no TLS 1.3 HKDF hash")
	// ErrCompoundMACMismatch is the failure §4.2.13 calls fatal: the MAC received does
	// not equal the MAC computed from the same tunnel and the same Outer TLVs.
	ErrCompoundMACMismatch = errors.New("teapbind: Compound MAC does not verify")
)

// OuterTLVs are the two message bodies RFC 9930 §6.3 steps 3 and 4 append to the
// BUFFER. The order is fixed by the specification — server first, then peer — and is
// the same for both parties, which is why each side can compute the same value while
// holding its own view of which message it sent and which it received.
//
// A TEAP message fragmented into several packets contributes the Outer TLVs of all
// its fragments, so the caller concatenates the fragments in wire order before
// storing them here.
type OuterTLVs struct {
	// Server holds the Outer TLVs of the first TEAP message the EAP server sent.
	Server []byte
	// Peer holds the Outer TLVs of the first TEAP message the peer sent.
	Peer []byte
}

// Keys are the two Compound MAC keys of the session — CMK[n] of RFC 9930 §6.2.2's
// parallel derivations, which is what teapkeys.Chain.CompoundMACKeys returns. A
// field is empty when no Inner Method has fed that path yet, and §6.2.5's
// carry-forward means the EMSK path stays empty for a session whose methods produce
// no EMSK.
type Keys struct {
	// MSKCMK keys the MSK Compound MAC, the one always available.
	MSKCMK []byte
	// EMSKCMK keys the EMSK Compound MAC, present only once a method derived an EMSK.
	EMSKCMK []byte
}

// Exchange is the input every Compound MAC of one TEAP session shares: the MAC
// algorithm the tunnel negotiated and the Phase 1 material the BUFFER covers. It is
// built once, after the tunnel is established, and reused for the Binding Request
// and the Binding Response of every Inner Method.
type Exchange struct {
	// Hash is the HMAC algorithm the TLS 1.3 key schedule negotiated for HKDF
	// (RFC 9427 §2.2). HashFromCipherSuite derives it from the negotiated suite.
	Hash crypto.Hash
	// OtherPartyType is the EAP Type the other party sent in the first TEAP message;
	// §6.3 requires it to be TEAP and rejects anything else rather than computing a
	// MAC over a session that is not TEAP.
	OtherPartyType uint8
	// Outer holds the Outer TLVs of the first TEAP message in each direction.
	Outer OuterTLVs
}

// HashFromCipherSuite returns the HMAC algorithm the TLS 1.3 key schedule uses for
// HKDF under a negotiated cipher suite, which RFC 9427 §2.2 makes the MAC algorithm
// of the Compound MAC. The hash is fixed by the suite (RFC 8446 §7.1 and §B.4), and
// crypto/tls exposes no accessor for it, so the mapping lives here: a TLS 1.2 suite
// is refused rather than silently substituted, because its PRF is not this MAC.
func HashFromCipherSuite(suiteID uint16) (crypto.Hash, error) {
	switch suiteID {
	case tls.TLS_AES_128_GCM_SHA256, tls.TLS_CHACHA20_POLY1305_SHA256:
		return crypto.SHA256, nil
	case tls.TLS_AES_256_GCM_SHA384:
		return crypto.SHA384, nil
	default:
		return 0, fmt.Errorf("%w: 0x%04x is not a TLS 1.3 cipher suite", ErrUnsupportedCipherSuite, suiteID)
	}
}

// Buffer assembles the octets the Compound MAC covers, in the order RFC 9930 §6.3
// lists them: the whole Crypto-Binding TLV with both MAC fields zeroed, the other
// party's EAP Type as one octet, then the Outer TLVs the server sent and the peer
// sent. The TLV is covered as it was received — the Mandatory bit included — because
// normalising it would let a peer flip a flag the other side did not see and still
// agree on the MAC.
func (x Exchange) Buffer(binding teaptlv.TLV) ([]byte, error) {
	if binding.Type != teaptlv.TypeCryptoBinding {
		return nil, fmt.Errorf("%w: TLV type %d", ErrNotCryptoBinding, binding.Type)
	}
	if x.OtherPartyType != teapkeys.Type {
		return nil, fmt.Errorf("%w: 0x%02x, §6.3 requires the TEAP octet 0x%02x",
			ErrWrongEAPType, x.OtherPartyType, teapkeys.Type)
	}
	wire, err := binding.Bytes()
	if err != nil {
		return nil, err
	}
	if len(wire) != bindingTLVLen {
		return nil, fmt.Errorf("%w: %d octets, §4.2.13 fixes it at %d", ErrTLVLength, len(wire), bindingTLVLen)
	}
	clear(wire[macFieldOffset:])

	buffer := make([]byte, 0, len(wire)+1+len(x.Outer.Server)+len(x.Outer.Peer))
	buffer = append(buffer, wire...)
	buffer = append(buffer, x.OtherPartyType)
	buffer = append(buffer, x.Outer.Server...)
	buffer = append(buffer, x.Outer.Peer...)
	return buffer, nil
}

// CompoundMAC returns the value of one Compound MAC field: the first 20 octets of
// HMAC(cmk, buffer), truncated as §4.2.13 requires. The truncation is the one place
// TEAP mandates cutting a key down — RFC 9427 §2.1 forbids it for the exporter
// calls, which teapkeys enforces.
func CompoundMAC(h crypto.Hash, cmk, buffer []byte) ([]byte, error) {
	if len(cmk) == 0 {
		return nil, ErrNoCompoundKey
	}
	if !h.Available() {
		return nil, fmt.Errorf("%w: %s is not linked in", ErrUnsupportedHash, h)
	}
	if size := h.Size(); size < compoundMACLen {
		return nil, fmt.Errorf("%w: %s yields %d octets, §4.2.13 needs %d", ErrUnsupportedHash, h, size, compoundMACLen)
	}
	mac := hmac.New(h.New, cmk)
	mac.Write(buffer)
	return mac.Sum(nil)[:compoundMACLen], nil
}

// Bind encodes a Crypto-Binding TLV with the Compound MAC fields the Flags announce
// filled in. It first lets teaptlv build the TLV with those slots zero — "During the
// calculation of the Compound MAC, the MAC field is filled with zeros" (§6.3) — takes
// the MAC over that octet string, then writes the results into the positions the
// layout fixes. The version fields and the nonce rules are teaptlv's checks, so a TLV
// that would be invalid on the wire never reaches a MAC.
func (x Exchange) Bind(keys Keys, flags, subType uint8, nonce []byte) (teaptlv.TLV, error) {
	zeroed := make([]byte, compoundMACLen)
	var emsk, msk []byte
	if flags&teaptlv.FlagEMSKPresent != 0 {
		emsk = zeroed
	}
	if flags&teaptlv.FlagMSKPresent != 0 {
		msk = zeroed
	}
	tlv, err := teaptlv.NewCryptoBinding(teaptlv.Version, flags, subType, nonce, emsk, msk)
	if err != nil {
		return teaptlv.TLV{}, err
	}
	buffer, err := x.Buffer(tlv)
	if err != nil {
		return teaptlv.TLV{}, err
	}
	if flags&teaptlv.FlagEMSKPresent != 0 {
		emsk, err = CompoundMAC(x.Hash, keys.EMSKCMK, buffer)
		if err != nil {
			return teaptlv.TLV{}, fmt.Errorf("%w (EMSK Compound MAC)", err)
		}
	}
	if flags&teaptlv.FlagMSKPresent != 0 {
		msk, err = CompoundMAC(x.Hash, keys.MSKCMK, buffer)
		if err != nil {
			return teaptlv.TLV{}, fmt.Errorf("%w (MSK Compound MAC)", err)
		}
	}
	// The validated TLV needs nothing else than its own MAC slots written in: a slot
	// Flags does not announce stays at the zero the codec left there, and re-encoding
	// would only repeat checks that have already passed.
	copy(tlv.Value[macValueOffset:], emsk)
	copy(tlv.Value[macValueOffset+compoundMACLen:], msk)
	return tlv, nil
}

// Verify checks the Compound MAC fields a peer announced in the TLV it sent. Each
// slot the Flags present is recomputed over the TLV as received and compared in
// constant time; a slot the Flags do not present is not examined, which is the
// behavior §6.2.4 states for both directions.
//
// Whether the set of slots is the one the session's policy requires — §6.2.3's rule
// that a method which derived an EMSK must be bound by the EMSK Compound MAC, and
// §6.2.4's Flags checks — is a handshake decision and is not made here.
func (x Exchange) Verify(keys Keys, received teaptlv.TLV) error {
	cb, err := received.CryptoBinding()
	if err != nil {
		return err
	}
	buffer, err := x.Buffer(received)
	if err != nil {
		return err
	}
	if cb.EMSKCompoundMAC != nil {
		if err := verifyOne(x.Hash, keys.EMSKCMK, buffer, cb.EMSKCompoundMAC); err != nil {
			return fmt.Errorf("%w (EMSK Compound MAC)", err)
		}
	}
	if cb.MSKCompoundMAC != nil {
		if err := verifyOne(x.Hash, keys.MSKCMK, buffer, cb.MSKCompoundMAC); err != nil {
			return fmt.Errorf("%w (MSK Compound MAC)", err)
		}
	}
	return nil
}

// verifyOne recomputes one field and compares it without leaking how far the two
// values agreed.
func verifyOne(h crypto.Hash, cmk, buffer, got []byte) error {
	want, err := CompoundMAC(h, cmk, buffer)
	if err != nil {
		return err
	}
	if !hmac.Equal(want, got) {
		return fmt.Errorf("%w: computed % x, received % x", ErrCompoundMACMismatch, want, got)
	}
	return nil
}
