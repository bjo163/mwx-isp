// Package teapkeys derives the TEAP session keys of RFC 9930 §6.2 for a tunnel
// negotiated with TLS 1.3, using the exporter labels RFC 9427 §2.2 gives the
// method.
//
// TEAP can run more than one Inner Method, so the outer MSK and EMSK are not the
// keys of any single method: each method contributes an Inner Method Session Key
// (IMSK) that is folded into a running compound key, and only the final compound
// key yields the session keys. Two compound keys run in parallel because a method
// may yield an MSK, an EMSK, or neither (RFC 9930 §6.2.2).
//
// The package computes no TLS handshake state, reads no TLVs and verifies no
// Crypto-Binding: the caller is the TEAP handshake, which supplies the exporter
// and the keys each Inner Method produced. RFC 9930 §6.3's Compound MAC is the
// caller's use of the keys returned here, not a service of this package.
//
// Only the TLS 1.3 derivation lives here. RFC 9930 §6.2 defines the TLS 1.2 form
// with the TLS-PRF of RFC 5246, and RFC 9427 §2.2 replaces it label by label; the
// two are not interchangeable, so a tunnel that negotiated TLS 1.2 must not call
// this package.
package teapkeys

import (
	"bytes"
	"errors"
	"fmt"
)

// Type is the EAP Type assigned to TEAP: 55, written 0x37. RFC 9930 §7.5 records
// the IANA allocation, and §6.3 states the octet in the round because the Compound
// MAC BUFFER carries "The EAP Type sent by the other party in the first TEAP
// message, which MUST be TEAP, encoded as one octet of 0x37". It is also the
// context RFC 9427 §2.2 passes for the session key seed.
const Type uint8 = 0x37

// The exporter labels are the five strings RFC 9427 §2.2 registers for TEAP. They
// are spelled here once because a label is a wire-visible input to the key
// schedule: renaming a constant does not fail a test, it fails interoperability.
const (
	labelSessionKeySeed = "EXPORTER: teap session key seed"
	labelInnerMethods   = "EXPORTER: Inner Methods Compound Keys"
	labelMSK            = "EXPORTER: Session Key Generating Function"
	labelEMSK           = "EXPORTER: Extended Session Key Generating Function"
	labelIMSK           = "TEAPbindkey@ietf.org"
)

// Key lengths come from the derivations themselves, and RFC 9427 §2.1 makes them
// load-bearing: "implementations MUST pass the correct length instead of passing a
// large length and truncating the output", because TLS 1.3 exporter output depends
// on the length argument.
const (
	// sIMCKLen is the length of S-IMCK[j]. RFC 9427 §2.2 sets S-IMCK[0] to the
	// session key seed, so the seed has this length too.
	sIMCKLen = 40
	// cmkLen is the length of CMK[j], the Compound MAC key RFC 9930 §6.3 puts in
	// front of the Compound MAC.
	cmkLen = 20
	// imckLen is the exported IMCK[j] that splits into the two keys above.
	imckLen = sIMCKLen + cmkLen
	// imskLen is the length of IMSK[j], and also the length of the all-zero IMSK a
	// method without keys contributes.
	imskLen = 32
	// resultKeyLen is the length RFC 9427 §2.2 derives for each of the outer MSK
	// and EMSK: each is its own export, not a slice of a larger one.
	resultKeyLen = 64
)

var (
	// ErrNoExporter is returned when the handshake supplies no exporter, which
	// means there is no completed TLS session to derive from.
	ErrNoExporter = errors.New("teapkeys: no TLS exporter")
	// ErrKeyLength is returned when the exporter hands back a number of octets
	// other than the one asked for. It is the §2.1 failure mode, so it stops the
	// handshake rather than deriving keys nobody can verify.
	ErrKeyLength = errors.New("teapkeys: exported key length mismatch")
	// ErrSeedLength is returned by NewChain for a session key seed that is not the
	// 40 octets RFC 9427 §2.2 derives.
	ErrSeedLength = errors.New("teapkeys: session key seed is not 40 octets")
	// ErrNoInnerMethod is returned when the session keys are asked for before any
	// Inner Method has been folded into the chain.
	ErrNoInnerMethod = errors.New("teapkeys: no Inner Method has been run")
	// ErrNoEMSK is returned when the EMSK compound key is selected although no
	// Inner Method produced an EMSK, so no EMSK Compound MAC could ever have been
	// verified (RFC 9930 §6.2.2, §6.2.5).
	ErrNoEMSK = errors.New("teapkeys: no Inner Method produced an EMSK")
)

// Exporter is the TLS keying-material export this package needs. It is the shape of
// tlsengine.Engine.ExportKey and, underneath, of crypto/tls
// ConnectionState.ExportKeyingMaterial: a label, a context, and the exact length.
type Exporter interface {
	ExportKey(label string, context []byte, length int) ([]byte, error)
}

// export asks for length octets and refuses anything else.
func export(e Exporter, label string, context []byte, length int) ([]byte, error) {
	if e == nil {
		return nil, ErrNoExporter
	}
	out, err := e.ExportKey(label, context, length)
	if err != nil {
		return nil, fmt.Errorf("teapkeys: export %q: %w", label, err)
	}
	if len(out) != length {
		return nil, fmt.Errorf("%w: %q asked for %d octets and got %d", ErrKeyLength, label, length, len(out))
	}
	return out, nil
}

// SessionKeySeed derives S-IMCK[0], the key that bootstraps the compound chain as
// soon as the TLS tunnel is established and before any Inner Method runs
// (RFC 9427 §2.2: session_key_seed = TLS-Exporter("EXPORTER: teap session key
// seed", Type, 40)). The context is the TEAP Type octet; with TLS 1.2 the same
// derivation used no context at all, which is why the two versions do not
// interoperate.
func SessionKeySeed(e Exporter) ([]byte, error) {
	return export(e, labelSessionKeySeed, []byte{Type}, sIMCKLen)
}

// InnerMethodKeys are the keys one Inner Method hands to the chain. Either may be
// empty: an inner method may derive an MSK only, an EMSK as well, or neither (the
// Basic-Password-Auth TLVs, which RFC 9930 §4.2.8 says set both to zero).
type InnerMethodKeys struct {
	MSK  []byte
	EMSK []byte
}

// chain is one of the two compound-key derivations.
type chain struct {
	sIMCK []byte
	cmk   []byte
	// advanced counts the Inner Methods folded into sIMCK, so the handshake can
	// tell "no method yet" from a method that contributed nothing.
	advanced int
}

// Chain carries both IMCK derivations RFC 9930 §6.2.2 requires: one built from the
// MSK of each Inner Method, one from the EMSK. Which one yields the session keys is
// decided at the end by what the peer put in its Crypto-Binding TLV.
//
// A Chain is not safe for concurrent use; one TEAP session owns one Chain.
type Chain struct {
	msk  chain
	emsk chain
}

// NewChain starts both derivations from the session key seed, which RFC 9427 §2.2
// spells S-IMCK[0] = session_key_seed. The seed is copied, so the caller may reuse
// its buffer.
func NewChain(sessionKeySeed []byte) (*Chain, error) {
	if len(sessionKeySeed) != sIMCKLen {
		return nil, fmt.Errorf("%w: got %d octets", ErrSeedLength, len(sessionKeySeed))
	}
	seed := bytes.Clone(sessionKeySeed)
	return &Chain{
		msk:  chain{sIMCK: seed},
		emsk: chain{sIMCK: bytes.Clone(seed)},
	}, nil
}

// zeroIMSK is the IMSK of an Inner Method that produced no keying material.
// RFC 9930 §6.2.1 and §6.3 both spell it as 32 octets of zero and warn that it
// leaves the exchange without cryptographic binding, so it is a legitimate value
// rather than an error — RFC 9930 §6.2.3 recommends ordering the methods so that
// the first one derives an EMSK and therefore still detects an on-path attacker.
func zeroIMSK() []byte { return make([]byte, imskLen) }

// innerKey derives IMSK[j] from one method key
// (RFC 9427 §2.2: IMSK[j] = TLS-Exporter("TEAPbindkey@ietf.org", secret, 32)),
// substituting the all-zero IMSK when the method produced no key of that kind.
//
// RFC 9930 §6.2.1 prefers the EMSK and allows the MSK because peers are known to
// differ, so which secret the caller passes is a method-policy choice; the
// derivation is the same either way.
func innerKey(e Exporter, secret []byte) ([]byte, error) {
	if len(secret) == 0 {
		return zeroIMSK(), nil
	}
	return export(e, labelIMSK, secret, imskLen)
}

// Advance folds one completed Inner Method into both derivations.
//
// The MSK path always advances: a method that produced no MSK contributes the
// all-zero IMSK. The EMSK path advances only when the method produced an EMSK;
// otherwise it carries forward unchanged, which RFC 9930 §6.2.5 records as the
// behavior interoperable implementations settled on ("IMCK_EMSK[j] =
// IMCK_EMSK[j - 1]") rather than a defect to be papered over here.
//
// Both paths step for every method including the last: RFC 9427 §2.2 derives the
// outer MSK and EMSK from S-IMCK[n], the value only that last step produces.
func (c *Chain) Advance(e Exporter, keys InnerMethodKeys) error {
	if err := c.mskStep(e, keys.MSK); err != nil {
		return fmt.Errorf("%w (MSK path)", err)
	}
	if len(keys.EMSK) == 0 {
		return nil
	}
	if err := c.emskStep(e, keys.EMSK); err != nil {
		return fmt.Errorf("%w (EMSK path)", err)
	}
	return nil
}

// mskStep folds one method's MSK into the MSK derivation.
func (c *Chain) mskStep(e Exporter, msk []byte) error {
	imsk, err := innerKey(e, msk)
	if err != nil {
		return err
	}
	return c.msk.step(e, imsk)
}

// emskStep folds one method's EMSK into the EMSK derivation.
func (c *Chain) emskStep(e Exporter, emsk []byte) error {
	imsk, err := innerKey(e, emsk)
	if err != nil {
		return err
	}
	return c.emsk.step(e, imsk)
}

// step computes IMCK[j] = TLS-Exporter("EXPORTER: Inner Methods Compound Keys",
// S-IMCK[j-1] || IMSK[j], 60) and splits it the way RFC 9427 §2.2 says: the first
// 40 octets are S-IMCK[j], the last 20 are CMK[j].
//
// The concatenation order is the TLS 1.3 form. With TLS 1.2 the seed was IMSK[j]
// alone and S-IMCK[j-1] was the PRF secret, so the two versions derive different
// keys from the same inputs (RFC 9427 §2.2 notes the change).
func (ch *chain) step(e Exporter, imsk []byte) error {
	context := append(bytes.Clone(ch.sIMCK), imsk...)
	out, err := export(e, labelInnerMethods, context, imckLen)
	if err != nil {
		return err
	}
	// Copy before the slice so neither key shares the exporter's buffer.
	ch.sIMCK = bytes.Clone(out[:sIMCKLen])
	ch.cmk = bytes.Clone(out[sIMCKLen:])
	ch.advanced++
	return nil
}

// CompoundMACKeys returns CMK[n] of each derivation, the key RFC 9930 §6.3 puts in
// front of the Compound MAC. A nil CMK means that derivation has not advanced with
// a usable key yet: the MSK path is nil before the first Inner Method, the EMSK
// path also when no method has produced an EMSK.
func (c *Chain) CompoundMACKeys() (mskCMK, emskCMK []byte) {
	return bytes.Clone(c.msk.cmk), bytes.Clone(c.emsk.cmk)
}

// SessionKeys derives the outer MSK and EMSK from the S-IMCK of the last
// successfully executed Inner Method, from the path the peer's Crypto-Binding TLV
// selected: useEMSK is true when that TLV carried an EMSK Compound MAC, false when
// it carried only the MSK one (RFC 9930 §6.2.2).
//
// RFC 9427 §2.2 fixes both exports at 64 octets and gives each its own label, so
// requesting 128 from one export and slicing it would produce keys no peer can
// reproduce.
func (c *Chain) SessionKeys(e Exporter, useEMSK bool) (msk, emsk []byte, err error) {
	if c.msk.advanced == 0 {
		return nil, nil, ErrNoInnerMethod
	}
	ch := &c.msk
	if useEMSK {
		if c.emsk.advanced == 0 {
			return nil, nil, ErrNoEMSK
		}
		ch = &c.emsk
	}
	msk, err = export(e, labelMSK, bytes.Clone(ch.sIMCK), resultKeyLen)
	if err != nil {
		return nil, nil, fmt.Errorf("%w (outer MSK)", err)
	}
	emsk, err = export(e, labelEMSK, bytes.Clone(ch.sIMCK), resultKeyLen)
	if err != nil {
		return nil, nil, fmt.Errorf("%w (outer EMSK)", err)
	}
	return msk, emsk, nil
}
