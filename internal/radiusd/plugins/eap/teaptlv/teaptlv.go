// Package teaptlv encodes and decodes the TEAP TLVs defined by RFC 9930 §4.2,
// the payload format an EAP server exchanges inside the TLS tunnel it builds for
// EAP type 39.
//
// Every TLV is laid out as documented in RFC 9930 §4.2.1: a two-octet field
// holding the Mandatory bit, a reserved bit and the 14-bit TLV Type, then a
// two-octet Length that counts the Value only, then the Value:
//
//	 0                   1                   2                   3
//	 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|M|R|            TLV Type       |            Length             |
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//	|                              Value...
//	+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
//
// This package knows the wire format and the constraints the specification states
// for each TLV it models. It does not compute the Crypto-Binding Compound MACs
// (RFC 9930 §6.3) and it holds no handshake state: those live with the method
// handler. The PAC TLV (type 11) is named here for completeness only — RFC 9930 §2
// deprecated it, so resumption rides on TLS 1.3 NewSessionTicket instead.
package teaptlv

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Type is a TEAP TLV type as it appears in the 14-bit TLV Type field.
type Type uint16

// The TLV types allocated by RFC 9930 §4.2.1. The list is reproduced in full so a
// peer's unexpected TLV is reported by name rather than as an opaque number.
const (
	TypeUnassigned Type = 0
	// TypeAuthorityID identifies the provisioning authority (§4.2.2).
	TypeAuthorityID Type = 1
	// TypeIdentityType tells the peer whether a user or a machine is expected (§4.2.3).
	TypeIdentityType Type = 2
	// TypeResult is the acknowledged success or failure of the exchange (§4.2.4).
	TypeResult Type = 3
	// TypeNAK reports TLVs the peer does not support (§4.2.5).
	TypeNAK Type = 4
	// TypeError carries an error code alongside a failure (§4.2.6).
	TypeError Type = 5
	// TypeChannelBinding is the RFC 5056 channel binding TLV (§4.2.7).
	TypeChannelBinding Type = 6
	// TypeVendorSpecific nests vendor-defined TLVs (§4.2.8).
	TypeVendorSpecific Type = 7
	// TypeRequestAction asks the peer for an action such as a password change (§4.2.9).
	TypeRequestAction Type = 8
	// TypeEAPPayload carries a complete inner EAP packet (§4.2.10).
	TypeEAPPayload Type = 9
	// TypeIntermediateResult reports the result of one inner method without ending
	// the exchange (§4.2.11).
	TypeIntermediateResult Type = 10
	// TypePAC is the Protected Access Credential TLV. RFC 9930 §2 deprecated it;
	// TEAP resumes with TLS 1.3 session tickets instead.
	TypePAC Type = 11
	// TypeCryptoBinding binds the tunnel to the inner method's keys (§4.2.13).
	TypeCryptoBinding Type = 12
	// TypeBasicPasswordAuthReq requests a password without an inner EAP method (§4.2.14).
	TypeBasicPasswordAuthReq Type = 13
	// TypeBasicPasswordAuthResp answers it with a username and password (§4.2.15).
	TypeBasicPasswordAuthResp Type = 14
	// TypePKCS7 carries a PKCS#7 payload (§4.2.16).
	TypePKCS7 Type = 15
	// TypePKCS10 carries a certificate signing request (§4.2.17).
	TypePKCS10 Type = 16
	// TypeTrustedServerRoot carries the trusted server root (§4.2.18).
	TypeTrustedServerRoot Type = 17
	// TypeCSRAttributes carries attributes to add to a signing request (§4.2.19).
	TypeCSRAttributes Type = 18
	// TypeIdentityHint lets the peer name the identity it is proving (§4.2.20).
	TypeIdentityHint Type = 19
)

// TLV field sizes.
const (
	// HeaderLen is the TLV Type/Length header in octets.
	HeaderLen = 4
	// maxType is the largest value the 14-bit TLV Type field can hold.
	maxType = 0x3fff
	// maxValueLen is the largest Value the two-octet Length field can count.
	maxValueLen = 0xffff
	// cryptoBindingLen is the Crypto-Binding Value length RFC 9930 §4.2.13 fixes at 76.
	cryptoBindingLen = 76
	// nonceLen is the Crypto-Binding Nonce length (§4.2.13).
	nonceLen = 32
	// compoundMACLen is the width of each Compound MAC field, which RFC 9930 §4.2.13
	// truncates to 20 octets regardless of the MAC algorithm.
	compoundMACLen = 20
	// passwordFieldLen is the width of the Userlen and Passlen octets (§4.2.15).
	passwordFieldLen = 1
	// maxPasswordField is the longest username or password a one-octet length can count.
	maxPasswordField = 0xff
)

// Status is the two-octet Status field shared by the Result TLV (§4.2.4) and the
// Intermediate-Result TLV (§4.2.11).
type Status uint16

// The Status values RFC 9930 defines. A Status outside this set is a fatal error:
// §4.2.4 requires the receiver to treat it as "Unexpected TLVs Exchanged".
const (
	StatusSuccess Status = 1
	StatusFailure Status = 2
)

// Identity values carried by the Identity-Type TLV (§4.2.3).
const (
	IdentityUser    uint16 = 1
	IdentityMachine uint16 = 2
)

// Crypto-Binding flag bits (§4.2.13). Every other combination is invalid.
const (
	flagEMSKPresent uint8 = 1
	flagMSKPresent  uint8 = 2
)

// Crypto-Binding sub-types (§4.2.13).
const (
	SubTypeBindingRequest  uint8 = 0
	SubTypeBindingResponse uint8 = 1
)

// Version is the Crypto-Binding TLV version TEAPv1 sends, and the only Received-Ver
// value a TEAPv1 receiver accepts (§4.2.13, §3.1).
const Version uint8 = 1

// Errors reported for a TLV that cannot be encoded or decoded. RFC 9930 §4.2.1 is
// explicit that a malformed TLV is invalid and cannot be used, so callers reject it
// rather than recovering a partial value.
var (
	// ErrTruncated means the octets ran out before the TLV the header described.
	ErrTruncated = errors.New("teaptlv: truncated TLV")
	// ErrInvalidTLV means a field holds a value the specification does not allow.
	ErrInvalidTLV = errors.New("teaptlv: invalid TLV")
)

// TLV is one Type-Length-Value record. Value is the raw octets counted by the
// wire-format Length field; the typed constructors and accessors below interpret it.
type TLV struct {
	// Type is the TLV type.
	Type Type
	// Mandatory is the M bit: the sender requires the receiver to recognize it.
	Mandatory bool
	// Value holds the TLV Value field.
	Value []byte
}

// Bytes encodes one TLV into its wire form. It fails when the type does not fit the
// 14-bit field or the Value is longer than the two-octet Length can count.
func (t TLV) Bytes() ([]byte, error) {
	if t.Type > maxType {
		return nil, fmt.Errorf("%w: type %d does not fit the 14-bit TLV Type field", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) > maxValueLen {
		return nil, fmt.Errorf("%w: value of %d octets exceeds the two-octet Length field", ErrInvalidTLV, len(t.Value))
	}
	out := make([]byte, HeaderLen+len(t.Value))
	word := uint16(t.Type)
	if t.Mandatory {
		word |= 0x8000
	}
	binary.BigEndian.PutUint16(out[0:2], word)
	binary.BigEndian.PutUint16(out[2:4], uint16(len(t.Value))) //nolint:gosec // G115: bounded by maxValueLen above
	copy(out[HeaderLen:], t.Value)
	return out, nil
}

// Marshal encodes a sequence of TLVs into the octet string that becomes an EAP-TLV
// payload.
func Marshal(tlvs ...TLV) ([]byte, error) {
	var out []byte
	for _, t := range tlvs {
		b, err := t.Bytes()
		if err != nil {
			return nil, err
		}
		out = append(out, b...)
	}
	return out, nil
}

// Parse decodes the TLVs in an EAP-TLV payload. It stops at the end of the octets,
// so zero TLVs is a valid result; a header that does not fit, or a Length that
// overruns the payload, is ErrTruncated because the remaining octets can no longer
// be attributed to a TLV.
func Parse(data []byte) ([]TLV, error) {
	var out []TLV
	for len(data) > 0 {
		if len(data) < HeaderLen {
			return nil, fmt.Errorf("%w: %d octets left, a TLV header needs %d", ErrTruncated, len(data), HeaderLen)
		}
		word := binary.BigEndian.Uint16(data[0:2])
		n := int(binary.BigEndian.Uint16(data[2:4]))
		if n > len(data)-HeaderLen {
			return nil, fmt.Errorf("%w: TLV type %d declares %d octets, %d remain",
				ErrTruncated, word&maxType, n, len(data)-HeaderLen)
		}
		out = append(out, TLV{
			Type:      Type(word & maxType),
			Mandatory: word&0x8000 != 0,
			Value:     data[HeaderLen : HeaderLen+n : HeaderLen+n],
		})
		data = data[HeaderLen+n:]
	}
	return out, nil
}

// Find returns the first TLV of the given type, or nil when the payload carries none.
func Find(tlvs []TLV, typ Type) *TLV {
	for i := range tlvs {
		if tlvs[i].Type == typ {
			return &tlvs[i]
		}
	}
	return nil
}

// NewResult builds the Result TLV of §4.2.4, which is marked mandatory and carries
// the two-octet Status.
func NewResult(status Status) TLV {
	return TLV{Type: TypeResult, Mandatory: true, Value: statusValue(status)}
}

// Status reads the Status field of a Result TLV. RFC 9930 fixes the Value at two
// octets and defines only Success (1) and Failure (2); anything else is a fatal
// error rather than an unknown outcome.
func (t TLV) Status() (Status, error) {
	if t.Type != TypeResult {
		return 0, fmt.Errorf("%w: TLV type %d is not a Result TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) != 2 {
		return 0, fmt.Errorf("%w: Result Length is %d, §4.2.4 fixes it at 2", ErrInvalidTLV, len(t.Value))
	}
	return readStatus(t.Value)
}

// NewIdentityType builds the Identity-Type TLV of §4.2.3, used by a server to say
// whether a machine or a user credential is expected and echoed back by the peer.
func NewIdentityType(identity uint16) TLV {
	value := make([]byte, 2)
	binary.BigEndian.PutUint16(value, identity)
	return TLV{Type: TypeIdentityType, Mandatory: true, Value: value}
}

// IdentityType reads the two-octet Identity-Type field. RFC 9930 §4.2.3 defines User
// (1) and Machine (2); a peer may answer with a type the server did not request, so
// the caller decides what a mismatch means rather than this method.
func (t TLV) IdentityType() (uint16, error) {
	if t.Type != TypeIdentityType {
		return 0, fmt.Errorf("%w: TLV type %d is not an Identity-Type TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) != 2 {
		return 0, fmt.Errorf("%w: Identity-Type Length is %d, §4.2.3 fixes it at 2", ErrInvalidTLV, len(t.Value))
	}
	switch identity := binary.BigEndian.Uint16(t.Value); identity {
	case IdentityUser, IdentityMachine:
		return identity, nil
	default:
		return 0, fmt.Errorf("%w: Identity-Type %d is not one of the values §4.2.3 defines", ErrInvalidTLV, identity)
	}
}

// NewAuthorityID builds the Authority-ID TLV of §4.2.2, an optional hint at the
// identity of the server so the peer can pick the credentials it holds for it.
func NewAuthorityID(id []byte) TLV {
	return TLV{Type: TypeAuthorityID, Value: append([]byte(nil), id...)}
}

// NewEAPPayload builds the EAP-Payload TLV of §4.2.10, which carries a complete
// inner EAP packet followed by zero or more TLVs. The packet is passed through
// untouched: its own Length field bounds it, so this package does not parse it.
func NewEAPPayload(packet []byte, nested ...TLV) (TLV, error) {
	if len(packet) < 4 {
		return TLV{}, fmt.Errorf("%w: an EAP packet header is 4 octets, got %d", ErrInvalidTLV, len(packet))
	}
	if declared := int(binary.BigEndian.Uint16(packet[2:4])); declared != len(packet) {
		return TLV{}, fmt.Errorf("%w: EAP packet declares %d octets but is %d", ErrInvalidTLV, declared, len(packet))
	}
	rest, err := Marshal(nested...)
	if err != nil {
		return TLV{}, err
	}
	value := make([]byte, 0, len(packet)+len(rest))
	value = append(value, packet...)
	return TLV{Type: TypeEAPPayload, Mandatory: true, Value: append(value, rest...)}, nil
}

// EAPPayload splits an EAP-Payload TLV into the inner EAP packet it carries and the
// TLVs that follow it. The packet is bounded by its own Length field (§4.2.10), so
// a packet that claims more octets than the TLV holds is a malformed TLV.
func (t TLV) EAPPayload() (packet []byte, nested []TLV, err error) {
	if t.Type != TypeEAPPayload {
		return nil, nil, fmt.Errorf("%w: TLV type %d is not an EAP-Payload TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) < 4 {
		return nil, nil, fmt.Errorf("%w: EAP-Payload Value is %d octets, too short for an EAP header", ErrTruncated, len(t.Value))
	}
	declared := int(binary.BigEndian.Uint16(t.Value[2:4]))
	if declared < 4 || declared > len(t.Value) {
		return nil, nil, fmt.Errorf("%w: inner EAP packet declares %d octets, %d available", ErrTruncated, declared, len(t.Value))
	}
	packet = t.Value[:declared]
	nested, err = Parse(t.Value[declared:])
	if err != nil {
		return nil, nil, err
	}
	return packet, nested, nil
}

// NewIntermediateResult builds the Intermediate-Result TLV of §4.2.11: the outcome
// of one inner method, followed by the TLVs associated with it. The embedded TLVs
// must all be optional, which is what lets a server mark the Intermediate-Result
// itself mandatory without forcing the peer to understand what rides along.
func NewIntermediateResult(status Status, nested ...TLV) (TLV, error) {
	for _, n := range nested {
		if n.Mandatory {
			return TLV{}, fmt.Errorf("%w: §4.2.11 forbids the mandatory bit on TLVs embedded in an Intermediate-Result TLV (type %d)", ErrInvalidTLV, n.Type)
		}
	}
	rest, err := Marshal(nested...)
	if err != nil {
		return TLV{}, err
	}
	value := append(statusValue(status), rest...)
	return TLV{Type: TypeIntermediateResult, Mandatory: true, Value: value}, nil
}

// IntermediateResult reads the Status of an Intermediate-Result TLV and the TLVs
// embedded after it.
func (t TLV) IntermediateResult() (Status, []TLV, error) {
	if t.Type != TypeIntermediateResult {
		return 0, nil, fmt.Errorf("%w: TLV type %d is not an Intermediate-Result TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) < 2 {
		return 0, nil, fmt.Errorf("%w: Intermediate-Result Value is %d octets, too short for a Status field", ErrTruncated, len(t.Value))
	}
	status, err := readStatus(t.Value[:2])
	if err != nil {
		return 0, nil, err
	}
	nested, err := Parse(t.Value[2:])
	if err != nil {
		return 0, nil, err
	}
	for _, n := range nested {
		if n.Mandatory {
			return 0, nil, fmt.Errorf("%w: §4.2.11 forbids the mandatory bit on TLVs embedded in an Intermediate-Result TLV (type %d)", ErrInvalidTLV, n.Type)
		}
	}
	return status, nested, nil
}

// NewVendorSpecific builds the Vendor-Specific TLV of §4.2.8, which carries a
// Vendor-Id and the vendor's own TLVs. The vendor payload is passed through as
// opaque octets: its format belongs to the vendor, and §4.2.8 only adds that a
// vendor TLV carrying an authentication protocol MUST define MSK and EMSK values.
func NewVendorSpecific(vendorID uint32, mandatory bool, vendorTLVs []byte) TLV {
	return TLV{
		Type:      TypeVendorSpecific,
		Mandatory: mandatory,
		Value:     vendorSpecificValue(vendorID, vendorTLVs),
	}
}

// VendorSpecific reads the Vendor-Id and the vendor-defined payload of a
// Vendor-Specific TLV. RFC 9930 states the Value length as four plus the cumulative
// length of the included Vendor TLVs, so a Value shorter than the Vendor-Id is
// malformed.
func (t TLV) VendorSpecific() (vendorID uint32, vendorTLVs []byte, err error) {
	if t.Type != TypeVendorSpecific {
		return 0, nil, fmt.Errorf("%w: TLV type %d is not a Vendor-Specific TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) < 4 {
		return 0, nil, fmt.Errorf("%w: Vendor-Specific Value is %d octets, shorter than the 4-octet Vendor-Id", ErrTruncated, len(t.Value))
	}
	return binary.BigEndian.Uint32(t.Value[:4]), t.Value[4:], nil
}

// NewNAK builds the NAK TLV of §4.2.5: the type the peer does not support, the
// Vendor-Id that type belongs to (zero unless it is a vendor type), and optional
// TLVs explaining the rejection. A NAK is never sent for a vendor type without its
// Vendor-Id, and every TLV it embeds must be optional.
func NewNAK(refused Type, vendorID uint32, optional ...TLV) (TLV, error) {
	if len(optional) > 0 {
		for _, o := range optional {
			if o.Mandatory {
				return TLV{}, fmt.Errorf("%w: §4.2.5 forbids the mandatory bit on TLVs embedded in a NAK TLV (type %d)", ErrInvalidTLV, o.Type)
			}
		}
	}
	rest, err := Marshal(optional...)
	if err != nil {
		return TLV{}, err
	}
	value := make([]byte, 6, 6+len(rest))
	binary.BigEndian.PutUint32(value[0:4], vendorID)
	binary.BigEndian.PutUint16(value[4:6], uint16(refused))
	return TLV{Type: TypeNAK, Mandatory: true, Value: append(value, rest...)}, nil
}

// NAK reads a NAK TLV: the Vendor-Id and type of the TLV the peer refused, plus the
// optional TLVs that came with it. RFC 9930 fixes the Vendor-Id at zero for types
// that are not vendor-specific.
func (t TLV) NAK() (refused Type, vendorID uint32, optional []TLV, err error) {
	if t.Type != TypeNAK {
		return 0, 0, nil, fmt.Errorf("%w: TLV type %d is not a NAK TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) < 6 {
		return 0, 0, nil, fmt.Errorf("%w: NAK Value is %d octets, §4.2.5 requires at least 6", ErrTruncated, len(t.Value))
	}
	vendorID = binary.BigEndian.Uint32(t.Value[0:4])
	refused = Type(binary.BigEndian.Uint16(t.Value[4:6]))
	optional, err = Parse(t.Value[6:])
	if err != nil {
		return 0, 0, nil, err
	}
	for _, o := range optional {
		if o.Mandatory {
			return 0, 0, nil, fmt.Errorf("%w: §4.2.5 forbids the mandatory bit on TLVs embedded in a NAK TLV (type %d)", ErrInvalidTLV, o.Type)
		}
	}
	return refused, vendorID, optional, nil
}

// NewBasicPasswordAuthReq builds the Basic-Password-Auth-Req TLV of §4.2.14. The
// prompt is an optional UTF-8 message shown to the user, so an empty string means
// the TLV carries no prompt.
func NewBasicPasswordAuthReq(prompt string) TLV {
	return TLV{Type: TypeBasicPasswordAuthReq, Mandatory: true, Value: []byte(prompt)}
}

// Prompt reads the optional UTF-8 prompt of a Basic-Password-Auth-Req TLV.
func (t TLV) Prompt() (string, error) {
	if t.Type != TypeBasicPasswordAuthReq {
		return "", fmt.Errorf("%w: TLV type %d is not a Basic-Password-Auth-Req TLV", ErrInvalidTLV, t.Type)
	}
	return string(t.Value), nil
}

// NewBasicPasswordAuthResp builds the Basic-Password-Auth-Resp TLV of §4.2.15: a
// one-octet length, the UTF-8 username, then the same pair for the password.
// RFC 9930 forbids a zero Userlen or Passlen, and a password needs no human-readable
// bytes, so both fields are taken as octets and must be non-empty and at most 255
// octets each.
func NewBasicPasswordAuthResp(username, password []byte) (TLV, error) {
	if len(username) == 0 || len(username) > maxPasswordField {
		return TLV{}, fmt.Errorf("%w: §4.2.15 requires a Userlen between 1 and 255, got %d", ErrInvalidTLV, len(username))
	}
	if len(password) == 0 || len(password) > maxPasswordField {
		return TLV{}, fmt.Errorf("%w: §4.2.15 requires a Passlen between 1 and 255, got %d", ErrInvalidTLV, len(password))
	}
	value := make([]byte, 0, 2*passwordFieldLen+len(username)+len(password))
	value = append(value, byte(len(username))) //nolint:gosec // G115: bounded by maxPasswordField above
	value = append(value, username...)
	value = append(value, byte(len(password))) //nolint:gosec // G115: bounded by maxPasswordField above
	value = append(value, password...)
	return TLV{Type: TypeBasicPasswordAuthResp, Mandatory: true, Value: value}, nil
}

// Credentials reads the username and password of a Basic-Password-Auth-Resp TLV. The
// password is returned as octets because §4.2.15 states it may contain values below
// 0x20, including 0x00.
func (t TLV) Credentials() (username, password []byte, err error) {
	if t.Type != TypeBasicPasswordAuthResp {
		return nil, nil, fmt.Errorf("%w: TLV type %d is not a Basic-Password-Auth-Resp TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) < 2*passwordFieldLen {
		return nil, nil, fmt.Errorf("%w: Basic-Password-Auth-Resp Value is %d octets, too short for both length octets", ErrTruncated, len(t.Value))
	}
	userLen := int(t.Value[0])
	if userLen == 0 {
		return nil, nil, fmt.Errorf("%w: §4.2.15 forbids a Userlen of zero", ErrInvalidTLV)
	}
	if 1+userLen >= len(t.Value) {
		return nil, nil, fmt.Errorf("%w: username declares %d octets, %d remain", ErrTruncated, userLen, len(t.Value)-1)
	}
	username = t.Value[1 : 1+userLen]
	passLen := int(t.Value[1+userLen])
	if passLen == 0 {
		return nil, nil, fmt.Errorf("%w: §4.2.15 forbids a Passlen of zero", ErrInvalidTLV)
	}
	if 2+userLen+passLen > len(t.Value) {
		return nil, nil, fmt.Errorf("%w: password declares %d octets, %d remain", ErrTruncated, passLen, len(t.Value)-2-userLen)
	}
	password = t.Value[2+userLen : 2+userLen+passLen]
	return username, password, nil
}

// CryptoBinding is the decoded Value of a Crypto-Binding TLV (§4.2.13). The
// specification fixes it at 76 octets: one reserved octet, the version sent, the
// version received, the Flags and Sub-Type octet, a 32-octet Nonce, and the two
// 20-octet Compound MAC fields the Flags say are present.
type CryptoBinding struct {
	// ReceivedVersion is the TEAP version the sender received during version
	// negotiation; RFC 9930 requires TEAPv1 implementations to set and check one (1).
	ReceivedVersion uint8
	// Flags reports which Compound MAC fields are present: flagEMSKPresent,
	// flagMSKPresent, or both.
	Flags uint8
	// SubType is SubTypeBindingRequest or SubTypeBindingResponse.
	SubType uint8
	// Nonce is the 32-octet value used for the Compound MAC key derivation.
	Nonce []byte
	// EMSKCompoundMAC is present when Flags includes flagEMSKPresent.
	EMSKCompoundMAC []byte
	// MSKCompoundMAC is present when Flags includes flagMSKPresent.
	MSKCompoundMAC []byte
}

// NewCryptoBinding builds a Crypto-Binding TLV from its parts; Value validates them
// against §4.2.13. The MACs are supplied by the caller because computing them is
// RFC 9930 §6.3's job, not this package's.
func NewCryptoBinding(receivedVersion uint8, flags, subType uint8, nonce, emskMAC, mskMAC []byte) (TLV, error) {
	value, err := (&CryptoBinding{
		ReceivedVersion: receivedVersion,
		Flags:           flags,
		SubType:         subType,
		Nonce:           nonce,
		EMSKCompoundMAC: emskMAC,
		MSKCompoundMAC:  mskMAC,
	}).Value()
	if err != nil {
		return TLV{}, err
	}
	return TLV{Type: TypeCryptoBinding, Mandatory: true, Value: value}, nil
}

// CryptoBinding decodes and validates a Crypto-Binding TLV. RFC 9930 §4.2.13 lists
// the checks that make the TLV valid — a known Version, a Received-Ver matching the
// negotiated version, a correct Sub-Type, a known Flags value and MACs that verify —
// and states that failing any of them is a fatal error, so this method reports the
// ones the wire format alone can decide. Verifying the MACs needs the session keys
// and is the caller's job.
func (t TLV) CryptoBinding() (*CryptoBinding, error) {
	if t.Type != TypeCryptoBinding {
		return nil, fmt.Errorf("%w: TLV type %d is not a Crypto-Binding TLV", ErrInvalidTLV, t.Type)
	}
	if len(t.Value) != cryptoBindingLen {
		return nil, fmt.Errorf("%w: Crypto-Binding Length is %d, §4.2.13 fixes it at %d", ErrInvalidTLV, len(t.Value), cryptoBindingLen)
	}
	if t.Value[0] != 0 {
		return nil, fmt.Errorf("%w: the Reserved octet is %d, §4.2.13 requires zero", ErrInvalidTLV, t.Value[0])
	}
	if v := t.Value[1]; v != Version {
		return nil, fmt.Errorf("%w: Version is %d, TEAPv1 sends %d", ErrInvalidTLV, v, Version)
	}
	if rv := t.Value[2]; rv != Version {
		return nil, fmt.Errorf("%w: Received-Ver is %d, §4.2.13 requires TEAPv1 to set it to %d", ErrInvalidTLV, rv, Version)
	}
	flags, subType := t.Value[3]>>4, t.Value[3]&0x0f
	cb := &CryptoBinding{ReceivedVersion: t.Value[2], Flags: flags, SubType: subType}
	if err := checkCryptoBindingFlags(flags); err != nil {
		return nil, err
	}
	if err := checkCryptoBindingSubType(subType); err != nil {
		return nil, err
	}
	cb.Nonce = t.Value[4 : 4+nonceLen]
	// The two Compound MAC fields keep their positions from the §4.2.13 layout
	// regardless of which one the Flags announce: RFC 9930 §6.3 computes the MAC over
	// "the entire Crypto-Binding TLV attribute with both the EMSK and MSK Compound MAC
	// fields zeroed out", which only holds if each field has one fixed slot.
	if flags&flagEMSKPresent != 0 {
		cb.EMSKCompoundMAC = t.Value[4+nonceLen : 4+nonceLen+compoundMACLen]
	}
	if flags&flagMSKPresent != 0 {
		cb.MSKCompoundMAC = t.Value[4+nonceLen+compoundMACLen : cryptoBindingLen]
	}
	if err := cb.checkNonce(); err != nil {
		return nil, err
	}
	return cb, nil
}

// Value encodes the Crypto-Binding fields into the 76-octet Value of §4.2.13.
func (cb *CryptoBinding) Value() ([]byte, error) {
	if cb.ReceivedVersion != Version {
		return nil, fmt.Errorf("%w: Received-Ver is %d, §4.2.13 requires TEAPv1 to set it to %d", ErrInvalidTLV, cb.ReceivedVersion, Version)
	}
	if len(cb.Nonce) != nonceLen {
		return nil, fmt.Errorf("%w: the Nonce is %d octets, §4.2.13 fixes it at %d", ErrInvalidTLV, len(cb.Nonce), nonceLen)
	}
	if err := checkCryptoBindingFlags(cb.Flags); err != nil {
		return nil, err
	}
	if err := checkCryptoBindingSubType(cb.SubType); err != nil {
		return nil, err
	}
	if err := cb.checkNonce(); err != nil {
		return nil, err
	}
	value := make([]byte, cryptoBindingLen)
	value[1] = Version
	value[2] = cb.ReceivedVersion
	value[3] = cb.Flags<<4 | cb.SubType
	copy(value[4:4+nonceLen], cb.Nonce)
	if cb.Flags&flagEMSKPresent != 0 {
		if len(cb.EMSKCompoundMAC) != compoundMACLen {
			return nil, fmt.Errorf("%w: the EMSK Compound MAC is %d octets, §4.2.13 fixes it at %d", ErrInvalidTLV, len(cb.EMSKCompoundMAC), compoundMACLen)
		}
		copy(value[4+nonceLen:], cb.EMSKCompoundMAC)
	}
	if cb.Flags&flagMSKPresent != 0 {
		if len(cb.MSKCompoundMAC) != compoundMACLen {
			return nil, fmt.Errorf("%w: the MSK Compound MAC is %d octets, §4.2.13 fixes it at %d", ErrInvalidTLV, len(cb.MSKCompoundMAC), compoundMACLen)
		}
		copy(value[4+nonceLen+compoundMACLen:], cb.MSKCompoundMAC)
	}
	return value, nil
}

// ResponseNonce returns the nonce a peer must echo in the Binding Response: RFC 9930
// §4.2.13 requires the same value as the request with its least significant bit set
// to one, so a server can derive the expected response nonce from the request.
func ResponseNonce(request []byte) ([]byte, error) {
	if len(request) != nonceLen {
		return nil, fmt.Errorf("%w: the Nonce is %d octets, §4.2.13 fixes it at %d", ErrInvalidTLV, len(request), nonceLen)
	}
	if request[nonceLen-1]&0x01 != 0 {
		return nil, fmt.Errorf("%w: a Binding Request nonce must end with its least significant bit clear", ErrInvalidTLV)
	}
	out := append([]byte(nil), request...)
	out[nonceLen-1] |= 0x01
	return out, nil
}

// checkNonce applies the least-significant-bit rule §4.2.13 states for the nonce of
// a request and of a response.
func (cb *CryptoBinding) checkNonce() error {
	lsb := cb.Nonce[nonceLen-1] & 0x01
	switch cb.SubType {
	case SubTypeBindingRequest:
		if lsb != 0 {
			return fmt.Errorf("%w: a Binding Request nonce must end with its least significant bit clear", ErrInvalidTLV)
		}
	case SubTypeBindingResponse:
		if lsb != 1 {
			return fmt.Errorf("%w: a Binding Response nonce must end with its least significant bit set", ErrInvalidTLV)
		}
	}
	return nil
}

// checkCryptoBindingFlags rejects the combinations §4.2.13 leaves undefined.
func checkCryptoBindingFlags(flags uint8) error {
	switch flags {
	case flagEMSKPresent, flagMSKPresent, flagEMSKPresent | flagMSKPresent:
		return nil
	default:
		return fmt.Errorf("%w: Flags %d is not one of the values §4.2.13 defines", ErrInvalidTLV, flags)
	}
}

// checkCryptoBindingSubType rejects sub-types outside Binding Request and Binding
// Response.
func checkCryptoBindingSubType(subType uint8) error {
	switch subType {
	case SubTypeBindingRequest, SubTypeBindingResponse:
		return nil
	default:
		return fmt.Errorf("%w: Sub-Type %d is not one of the values §4.2.13 defines", ErrInvalidTLV, subType)
	}
}

// statusValue encodes the two-octet Status field.
func statusValue(status Status) []byte {
	value := make([]byte, 2)
	binary.BigEndian.PutUint16(value, uint16(status))
	return value
}

// readStatus decodes the two-octet Status field shared by the Result and
// Intermediate-Result TLVs. Both callers bound the field before passing it.
func readStatus(value []byte) (Status, error) {
	switch s := Status(binary.BigEndian.Uint16(value[:2])); s {
	case StatusSuccess, StatusFailure:
		return s, nil
	default:
		return 0, fmt.Errorf("%w: Status %d is not one of the values §4.2.4 defines", ErrInvalidTLV, s)
	}
}

// vendorSpecificValue lays out a Vendor-Id followed by the vendor's octets.
func vendorSpecificValue(vendorID uint32, vendorTLVs []byte) []byte {
	value := make([]byte, 4+len(vendorTLVs))
	binary.BigEndian.PutUint32(value[:4], vendorID)
	copy(value[4:], vendorTLVs)
	return value
}
