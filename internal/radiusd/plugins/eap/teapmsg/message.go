// Package teapmsg implements the TEAP message envelope of RFC 9930 §4.1, the
// framing every TEAP packet carries — outside the TLS tunnel during Phase 1 and
// inside it during Phase 2.
//
// A TEAP packet is an EAP message of type 55 whose data field is:
//
//	+--------+---------+----------------+---------------+----------+------------+
//	| Flags  | Version | Message Length | Outer TLV Len | TLS Data | Outer TLVs |
//	| 5 bits | 3 bits  | L flag only    | O flag only   |          |            |
//	+--------+---------+----------------+---------------+----------+------------+
//
// Flags (RFC 9930 §4.1) are:
//
//	0 1 2 3 4
//	+-+-+-+-+-+
//	|L M S O R|
//	+-+-+-+-+-+
//
// L marks the presence of the four-octet Message Length field and is set only on
// the first fragment of a fragmented message; M is set on every fragment but the
// last; S marks the server's TEAP Start; O marks the four-octet Outer TLV Length
// field and is set only in the initial request and response, on its first
// fragment; R is reserved and must be zero.
//
// Message Length counts the TLS Data that the fragments reassemble into, which is
// the payload RFC 9930 §4.1 says "may be fragmented over the data fields of
// multiple packets". The Outer TLVs are counted separately by the Outer TLV
// Length field, because §4.1 delimits them from the end of the packet with that
// field rather than by the message's own length.
//
// The Outer TLVs are Phase 1 clear-text material: §6.3 appends them, in both
// directions, to the BUFFER the Crypto-Binding Compound MAC covers, which is why
// the reassembler hands them back instead of discarding them.
//
// This package frames messages and nothing else. It does not negotiate the
// version it carries (§3.1), interpret the TLS records inside, or decode the
// Outer TLVs — teaptlv owns TLV formats, tlsengine owns the handshake, and the
// handler that owns a TEAP session decides what each message means.
package teapmsg

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
)

// Version is the TEAP protocol version this package frames. RFC 9930 §4.1
// documents version 1 in the three-bit Ver field, and §3.1 requires an
// implementation of that document to send it.
const Version uint8 = 1

// Flag bits of the Flags/Ver octet (RFC 9930 §4.1).
const (
	// FlagLengthIncluded is L: the four-octet Message Length field is present.
	FlagLengthIncluded uint8 = 0x80
	// FlagMoreFragments is M: this packet is not the last fragment of the message.
	FlagMoreFragments uint8 = 0x40
	// FlagStart is S: the server's TEAP Start.
	FlagStart uint8 = 0x20
	// FlagOuterTLVs is O: the four-octet Outer TLV Length field is present.
	FlagOuterTLVs uint8 = 0x10
)

const (
	// flagReserved is R: "MUST be zero and ignored upon receipt" (§4.1). The
	// encoder never sets it; the parser keeps it out of the flags it reports.
	flagReserved uint8 = 0x08
	// versionMask selects the three-bit Ver field.
	versionMask uint8 = 0x07
	// flagsMask selects the five flag bits.
	flagsMask uint8 = 0xF8
)

// DefaultMaxMessageLength is the default upper bound on one reassembled message.
// RFC 9930 §3.10 leaves fragmentation to the rules of RFC 5216 §2.1.5, which
// allows a maximum "to protect against reassembly lockup and denial-of-service
// attacks"; 64 KB is the value that section calls reasonable for a TLS message
// carrying a certificate.
const DefaultMaxMessageLength = 64 * 1024

// Errors reported for a packet that is not valid TEAP framing.
var (
	// ErrEmptyPayload means the EAP data field carries no Flags/Ver octet.
	ErrEmptyPayload = errors.New("teap: empty payload (missing the flags octet)")
	// ErrShortLength means the L flag is set but the four-octet Message Length
	// field is truncated.
	ErrShortLength = errors.New("teap: length flag set but the Message Length field is truncated")
	// ErrShortOuterTLVs means the O flag is set but the four-octet Outer TLV
	// Length field, or the Outer TLVs it counts, do not fit the packet.
	ErrShortOuterTLVs = errors.New("teap: outer TLV length flag set but the Outer TLVs are truncated")
	// ErrLengthOnContinuation means the L flag appeared on a fragment that is not
	// the first one, which §4.1 forbids.
	ErrLengthOnContinuation = errors.New("teap: length flag set on a continuation fragment")
	// ErrOuterTLVsOnContinuation means the O flag appeared on a fragment that is
	// not the first one, which §4.1 forbids.
	ErrOuterTLVsOnContinuation = errors.New("teap: outer TLV length flag set on a continuation fragment")
	// ErrInvalidVersion means the version does not fit the three-bit Ver field.
	ErrInvalidVersion = errors.New("teap: protocol version does not fit the three-bit field")
	// ErrDeclaredLengthExceeded means the reassembled TLS Data grew past the
	// Message Length the first fragment advertised.
	ErrDeclaredLengthExceeded = errors.New("teap: reassembled data exceeds the declared Message Length")
	// ErrMaxLengthExceeded means a declared or reassembled message exceeds the
	// configured maximum, guarding against reassembly denial of service.
	ErrMaxLengthExceeded = errors.New("teap: message exceeds the maximum allowed length")
	// ErrFinalLengthMismatch means the last fragment arrived but the reassembled
	// TLS Data is not the length the first fragment declared.
	ErrFinalLengthMismatch = errors.New("teap: reassembled length does not match the declared Message Length")
)

// fitsMessage reports whether a field length can be carried by the four-octet
// length fields §4.1 defines.
func fitsMessage(n uint64) bool { return n <= math.MaxUint32 }

// Packet is one parsed TEAP packet: the EAP data field after the Type octet,
// split into the fields RFC 9930 §4.1 defines.
type Packet struct {
	// Flags holds the five flag bits, with R cleared. Version is kept apart.
	Flags uint8
	// Version is the three-bit Ver field. A parser does not judge it: §3.1
	// negotiation is the handler's, and a peer may legitimately answer with the
	// lower version it supports.
	Version uint8
	// HasLength reports whether L was set, so MessageLength is meaningful.
	HasLength bool
	// MessageLength is the total TLS Data length a fragmented message will
	// reassemble to; it counts TLS Data only, not the Outer TLVs.
	MessageLength uint32
	// HasOuterTLVs reports whether O was set, so OuterTLVs is the packet's
	// Outer TLV field rather than empty.
	HasOuterTLVs bool
	// TLSData is the TLS record material carried by this packet.
	TLSData []byte
	// OuterTLVs is the Phase 1 Outer TLV material §6.3 covers, in wire order.
	OuterTLVs []byte
}

// More reports whether the M flag is set, so further fragments follow.
func (p *Packet) More() bool { return p.Flags&FlagMoreFragments != 0 }

// Start reports whether the S flag is set, marking the server's TEAP Start.
func (p *Packet) Start() bool { return p.Flags&FlagStart != 0 }

// IsACK reports whether this is a TEAP acknowledgment: §4.1 describes it as "A
// TEAP packet with Flags and Version fields, but with zero length TLS Data
// field", and this predicate narrows that to a packet that also asks nothing of
// the receiver (no L, M, S or O flag) and carries no Outer TLVs.
func (p *Packet) IsACK() bool {
	return len(p.TLSData) == 0 && len(p.OuterTLVs) == 0 &&
		p.Flags&(FlagLengthIncluded|FlagMoreFragments|FlagStart|FlagOuterTLVs) == 0
}

// Parse decodes an EAP data field (EAPMessage.Data) into a Packet. It validates
// the framing §4.1 defines but does not interpret the TLS records or the TLVs.
func Parse(payload []byte) (*Packet, error) {
	if len(payload) == 0 {
		return nil, ErrEmptyPayload
	}

	p := &Packet{
		Flags:   payload[0] & flagsMask &^ flagReserved,
		Version: payload[0] & versionMask,
	}
	rest := payload[1:]

	if p.Flags&FlagLengthIncluded != 0 {
		if len(rest) < 4 {
			return nil, ErrShortLength
		}
		p.HasLength = true
		p.MessageLength = binary.BigEndian.Uint32(rest[:4])
		rest = rest[4:]
	}

	if p.Flags&FlagOuterTLVs != 0 {
		if len(rest) < 4 {
			return nil, ErrShortOuterTLVs
		}
		outerLen := binary.BigEndian.Uint32(rest[:4])
		rest = rest[4:]
		if uint64(outerLen) > uint64(len(rest)) {
			return nil, ErrShortOuterTLVs
		}
		p.HasOuterTLVs = true
		split := len(rest) - int(outerLen)
		p.TLSData = rest[:split]
		p.OuterTLVs = rest[split:]
		return p, nil
	}

	p.TLSData = rest
	return p, nil
}

// Encode serializes the Packet into an EAP data field (EAPMessage.Data). The L
// and O flags and their length fields are emitted from HasLength and
// HasOuterTLVs; the reserved bit is never set, and a version wider than the
// three-bit field is refused rather than truncated.
func (p *Packet) Encode() ([]byte, error) {
	if p.Version&^versionMask != 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidVersion, p.Version)
	}
	if !fitsMessage(uint64(len(p.TLSData))) || !fitsMessage(uint64(len(p.OuterTLVs))) {
		return nil, ErrMaxLengthExceeded
	}

	flags := p.Flags & flagsMask &^ flagReserved
	size := 1 + len(p.TLSData) + len(p.OuterTLVs)
	if p.HasLength {
		flags |= FlagLengthIncluded
		size += 4
	} else {
		flags &^= FlagLengthIncluded
	}
	if p.HasOuterTLVs {
		flags |= FlagOuterTLVs
		size += 4
	} else {
		flags &^= FlagOuterTLVs
	}

	buf := make([]byte, 0, size)
	buf = append(buf, flags|p.Version)
	if p.HasLength {
		buf = binary.BigEndian.AppendUint32(buf, p.MessageLength)
	}
	if p.HasOuterTLVs {
		buf = binary.BigEndian.AppendUint32(buf, uint32(len(p.OuterTLVs))) //nolint:gosec // G115: bounded above by the MaxUint32 check
	}
	buf = append(buf, p.TLSData...)
	buf = append(buf, p.OuterTLVs...)
	return buf, nil
}

// Reassembler accumulates inbound TEAP fragments into one TLS message and keeps
// the Outer TLVs that traveled with them. The zero value is not ready for use;
// create one with NewReassembler.
type Reassembler struct {
	buf       []byte
	outer     []byte
	declared  uint32
	hasLength bool
	started   bool
	maxLength int
}

// NewReassembler creates a Reassembler. A non-positive maxLength selects
// DefaultMaxMessageLength.
func NewReassembler(maxLength int) *Reassembler {
	if maxLength <= 0 {
		maxLength = DefaultMaxMessageLength
	}
	return &Reassembler{maxLength: maxLength}
}

// LoadReassembler reconstructs a Reassembler from state persisted between EAP
// rounds. buf is the TLS Data accumulated so far, outer the Outer TLVs that came
// with the first fragment, declared the Message Length advertised by that
// fragment (0 when unknown) and hasLength whether it was advertised.
func LoadReassembler(buf, outer []byte, declared uint32, hasLength bool, maxLength int) *Reassembler {
	r := NewReassembler(maxLength)
	r.buf = buf
	r.outer = outer
	r.declared = declared
	r.hasLength = hasLength
	r.started = len(buf) > 0 || len(outer) > 0 || hasLength
	return r
}

// Buffer returns the TLS Data accumulated so far. The slice belongs to the
// Reassembler and must not be modified by the caller.
func (r *Reassembler) Buffer() []byte { return r.buf }

// OuterTLVs returns the Outer TLVs collected from the message's fragments, in
// wire order. They are the Phase 1 material RFC 9930 §6.3 puts in the
// Crypto-Binding BUFFER.
func (r *Reassembler) OuterTLVs() []byte { return r.outer }

// Declared returns the advertised Message Length and whether it is known.
func (r *Reassembler) Declared() (uint32, bool) { return r.declared, r.hasLength }

// Accept folds one fragment into the reassembly buffer. It returns complete=true
// once the final fragment (M flag clear) has arrived, at which point Buffer holds
// the whole TLS Data and OuterTLVs the Outer TLVs that came with it.
func (r *Reassembler) Accept(p *Packet) (complete bool, err error) {
	if p.HasLength {
		if r.started {
			return false, ErrLengthOnContinuation
		}
		if int64(p.MessageLength) > int64(r.maxLength) {
			return false, fmt.Errorf("%w: declared %d > max %d", ErrMaxLengthExceeded, p.MessageLength, r.maxLength)
		}
		r.declared = p.MessageLength
		r.hasLength = true
	}
	if p.HasOuterTLVs {
		if r.started {
			return false, ErrOuterTLVsOnContinuation
		}
		if len(r.outer)+len(p.OuterTLVs) > r.maxLength {
			return false, fmt.Errorf("%w: accumulated %d > max %d", ErrMaxLengthExceeded, len(r.outer)+len(p.OuterTLVs), r.maxLength)
		}
		r.outer = append(r.outer, p.OuterTLVs...)
	}

	r.started = true

	if len(p.TLSData) > 0 {
		if len(r.buf)+len(p.TLSData) > r.maxLength {
			return false, fmt.Errorf("%w: accumulated %d > max %d", ErrMaxLengthExceeded, len(r.buf)+len(p.TLSData), r.maxLength)
		}
		if r.hasLength && uint64(len(r.buf)+len(p.TLSData)) > uint64(r.declared) {
			return false, ErrDeclaredLengthExceeded
		}
		r.buf = append(r.buf, p.TLSData...)
	}

	if p.More() {
		return false, nil
	}
	if r.hasLength && uint64(len(r.buf)) != uint64(r.declared) {
		return false, fmt.Errorf("%w: reassembled %d != declared %d", ErrFinalLengthMismatch, len(r.buf), r.declared)
	}
	return true, nil
}

// Fragment splits one outbound message into ordered TEAP packets carrying at
// most maxFragment octets of TLS Data each. The first packet declares L and the
// Message Length, and carries outerTLVs with the O flag when there are any; every
// packet except the last sets M (§4.1). A non-positive maxFragment selects
// DefaultMaxMessageLength.
//
// Empty tlsData yields a single packet with no TLS Data, no L flag (there is
// nothing to reassemble) and no M flag — the shape a Start takes, whose Outer
// TLVs travel on their own. outerTLVs are only ever attached to the first
// packet, because §4.1 allows the O field only there.
func Fragment(version uint8, tlsData, outerTLVs []byte, maxFragment int) ([]*Packet, error) {
	if version&^versionMask != 0 {
		return nil, fmt.Errorf("%w: %d", ErrInvalidVersion, version)
	}
	if !fitsMessage(uint64(len(tlsData))) || !fitsMessage(uint64(len(outerTLVs))) {
		return nil, ErrMaxLengthExceeded
	}
	if maxFragment <= 0 {
		maxFragment = DefaultMaxMessageLength
	}

	total := len(tlsData)
	if total == 0 {
		return []*Packet{{
			Version:      version,
			HasOuterTLVs: len(outerTLVs) > 0,
			OuterTLVs:    append([]byte(nil), outerTLVs...),
		}}, nil
	}

	var packets []*Packet
	for off := 0; off < total; off += maxFragment {
		end := min(off+maxFragment, total)
		p := &Packet{
			Version: version,
			TLSData: append([]byte(nil), tlsData[off:end]...),
		}
		if off == 0 {
			p.HasLength = true
			p.MessageLength = uint32(total) //nolint:gosec // G115: bounded above by the MaxUint32 check
			if len(outerTLVs) > 0 {
				p.HasOuterTLVs = true
				p.OuterTLVs = append([]byte(nil), outerTLVs...)
			}
		}
		if end < total {
			p.Flags |= FlagMoreFragments
		}
		packets = append(packets, p)
	}
	return packets, nil
}
