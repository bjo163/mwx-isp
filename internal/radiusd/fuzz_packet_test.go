package radiusd

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/huawei"
	"github.com/bjo163/mwx-isp/internal/radiusd/vendors/mikrotik"
	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

// radiusFrameHeaderSize is the fixed RADIUS header size (Code, Identifier,
// Length) that precedes the 16-octet Authenticator.
const radiusFrameHeaderSize = 4

// radFuzzSecret is the shared secret the corpus is parsed with. It is not a
// credential: it only exercises the keyed Message-Authenticator path.
var radFuzzSecret = []byte("fuzz-shared-secret")

// radiusFrameExpectation is the contract outcome a corpus frame must produce.
// RADIUS Length semantics come from RFC 2865 §3: the minimum length is 20, the
// maximum is 4096, and a packet shorter than its Length field MUST be
// silently discarded.
type radiusFrameExpectation int

const (
	// expectPacket means the frame is a complete, well-formed packet.
	expectPacket radiusFrameExpectation = iota
	// expectMalformedFrame means the length field itself cannot delimit a
	// RADIUS packet, so the connection must be considered desynchronized.
	expectMalformedFrame
	// expectShortRead means the announcement was plausible but the stream
	// ended before the declared body arrived.
	expectShortRead
	// expectBadAttributes means the frame was read completely but the
	// attribute TLV list violates RFC 2865 §5.
	expectBadAttributes
)

type radiusFrameCase struct {
	name  string
	frame []byte
	want  radiusFrameExpectation
	// code and attrCount are asserted only for expectPacket cases.
	code      radius.Code
	attrCount int
}

// radiusFrameLength converts a test-controlled byte count to the 16-bit Length
// field, panicking on input that no corpus fixture should ever produce.
func radiusFrameLength(n int) uint16 {
	if n < 0 || n > 0xFFFF {
		panic("radius test frame length out of range")
	}
	return uint16(n)
}

// radiusAttributeLength converts an attribute TLV size to its one-octet Length
// field, panicking on input that no corpus fixture should ever produce.
func radiusAttributeLength(n int) byte {
	if n < 0 || n > 0xFF {
		panic("radius test attribute length out of range")
	}
	return byte(n)
}

// radiusFrame builds a frame whose declared Length matches len(body).
func radiusFrame(body []byte) []byte {
	frame := make([]byte, radiusFrameHeaderSize+len(body))
	frame[0] = byte(radius.CodeAccessRequest)
	frame[1] = 9
	binary.BigEndian.PutUint16(frame[2:4], radiusFrameLength(len(frame)))
	copy(frame[radiusFrameHeaderSize:], body)
	return frame
}

// radiusFrameWithDeclaredLength builds a frame whose Length field is set
// independently of the bytes actually supplied, so truncation and oversize
// cases can be expressed.
func radiusFrameWithDeclaredLength(declared uint16, body []byte) []byte {
	frame := make([]byte, radiusFrameHeaderSize+len(body))
	frame[0] = byte(radius.CodeAccessRequest)
	frame[1] = 9
	binary.BigEndian.PutUint16(frame[2:4], declared)
	copy(frame[radiusFrameHeaderSize:], body)
	return frame
}

// radiusAttribute encodes one attribute TLV (Type, Length, Value).
func radiusAttribute(typ byte, value []byte) []byte {
	attr := make([]byte, 2+len(value))
	attr[0] = typ
	attr[1] = radiusAttributeLength(len(attr))
	copy(attr[2:], value)
	return attr
}

func radiusAttributeList(attrs ...[]byte) []byte {
	return bytes.Join(attrs, nil)
}

// radiusFrameWithAttributes prepends a zero authenticator to the given
// attributes and frames them as an Access-Request.
func radiusFrameWithAttributes(attrs ...[]byte) []byte {
	body := append(make([]byte, 16), radiusAttributeList(attrs...)...)
	return radiusFrame(body)
}

// huaweiEnterpriseID is Huawei's SMI Network Management Private Enterprise
// Code, the four-octet vendor identifier of a Vendor-Specific attribute.
const huaweiEnterpriseID = 2011

// huaweiVSA wraps a nested Huawei TLV (Type, Length, Value) in a
// Vendor-Specific attribute (RFC 2865 §5.26, enterprise code 2011).
func huaweiVSA(nested []byte) []byte {
	value := make([]byte, 4+len(nested))
	binary.BigEndian.PutUint32(value[:4], huaweiEnterpriseID)
	copy(value[4:], nested)
	return radiusAttribute(byte(rfc2865.VendorSpecific_Type), value)
}

// radiusMaxLengthFrame returns the largest legal packet: declared Length 4096
// with a full attribute list that exactly fills the body.
func radiusMaxLengthFrame() []byte {
	var attrs [][]byte
	for i := 0; i < 16; i++ {
		attrs = append(attrs, radiusAttribute(byte(rfc2865.UserName_Type), bytes.Repeat([]byte{'a'}, 252)))
	}
	attrs = append(attrs, radiusAttribute(byte(rfc2865.UserName_Type), bytes.Repeat([]byte{'b'}, 10)))
	return radiusFrameWithAttributes(attrs...)
}

func radiusFrameCorpus() []radiusFrameCase {
	maxFrame := radiusMaxLengthFrame()
	if got := len(maxFrame); got != maxRadiusPacketLength {
		panic("radiusMaxLengthFrame must be exactly 4096 bytes")
	}
	return []radiusFrameCase{
		{
			name:  "minimal access request",
			frame: radiusFrame(make([]byte, 16)),
			want:  expectPacket,
			code:  radius.CodeAccessRequest,
		},
		{
			name: "user-name and nas-identifier attributes",
			frame: radiusFrameWithAttributes(
				radiusAttribute(byte(rfc2865.UserName_Type), []byte("alice")),
				radiusAttribute(byte(rfc2865.NASIdentifier_Type), []byte("nas-1")),
			),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 2,
		},
		{
			// RFC 2865 §3: minimum Length is 20; 19 cannot even contain the
			// authenticator.
			name:  "declared length below minimum",
			frame: radiusFrameWithDeclaredLength(19, make([]byte, 15)),
			want:  expectMalformedFrame,
		},
		{
			name:  "declared length zero",
			frame: radiusFrameWithDeclaredLength(0, nil),
			want:  expectMalformedFrame,
		},
		{
			name:  "declared length below header size",
			frame: radiusFrameWithDeclaredLength(3, nil),
			want:  expectMalformedFrame,
		},
		{
			// RFC 2865 §3: maximum Length is 4096.
			name:  "declared length one byte over maximum",
			frame: radiusFrameWithDeclaredLength(4097, make([]byte, 16)),
			want:  expectMalformedFrame,
		},
		{
			// 65535 is the maximum a 16-bit Length field can encode; accepting
			// it would drive an oversized pooled allocation.
			name:  "declared length 65535",
			frame: radiusFrameWithDeclaredLength(0xFFFF, make([]byte, 16)),
			want:  expectMalformedFrame,
		},
		{
			name:      "declared length 4096 with full attribute list",
			frame:     maxFrame,
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 17,
		},
		{
			name:  "truncated authenticator",
			frame: radiusFrameWithDeclaredLength(20, make([]byte, 10)),
			want:  expectShortRead,
		},
		{
			name:  "truncated attribute body",
			frame: radiusFrameWithDeclaredLength(100, make([]byte, 20)),
			want:  expectShortRead,
		},
		{
			name:  "partial header",
			frame: []byte{1, 2, 0},
			want:  expectShortRead,
		},
		{
			name:  "attribute length zero",
			frame: radiusFrameWithAttributes([]byte{1, 0}),
			want:  expectBadAttributes,
		},
		{
			name:  "attribute length one",
			frame: radiusFrameWithAttributes([]byte{1, 1}),
			want:  expectBadAttributes,
		},
		{
			name:  "attribute overruns body",
			frame: radiusFrameWithAttributes([]byte{1, 10, 'a'}),
			want:  expectBadAttributes,
		},
		{
			name:  "attribute header truncated",
			frame: radiusFrameWithAttributes([]byte{1}),
			want:  expectBadAttributes,
		},
		{
			// RFC 2865 §3: octets beyond the Length field are padding and must
			// be ignored, not folded into the packet.
			name:  "trailing padding is ignored",
			frame: radiusFrameWithDeclaredLength(20, append(make([]byte, 16), 9, 9, 9)),
			want:  expectPacket,
			code:  radius.CodeAccessRequest,
		},
		{
			name:  "all bytes set",
			frame: bytes.Repeat([]byte{0xFF}, 64),
			want:  expectMalformedFrame,
		},
		{
			name:      "vendor specific attribute with malformed nested TLV",
			frame:     radiusFrameWithAttributes(radiusAttribute(byte(rfc2865.VendorSpecific_Type), []byte{0, 0, 7, 0xDB, 1, 0})),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 1,
		},
		{
			name: "vendor specific attribute with valid Huawei TLV",
			frame: radiusFrameWithAttributes(huaweiVSA([]byte{
				1, 6, 0, 0, 0x27, 0x10,
			})),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 1,
		},
		{
			name:      "vendor specific attribute without value",
			frame:     radiusFrameWithAttributes(radiusAttribute(byte(rfc2865.VendorSpecific_Type), nil)),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 1,
		},
		{
			name:      "message authenticator with wrong length",
			frame:     radiusFrameWithAttributes(radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, 8))),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 1,
		},
		{
			// RFC 3579 §3.2 permits at most one Message-Authenticator.
			name: "duplicate message authenticator",
			frame: radiusFrameWithAttributes(
				radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, 16)),
				radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, 16)),
			),
			want:      expectPacket,
			code:      radius.CodeAccessRequest,
			attrCount: 2,
		},
	}
}

// checkRadiusFrameContract asserts the framing invariants shared by every
// parse outcome. A framing error is only legal when the announced Length is
// outside RFC 2865 §3's 20..4096 window; every returned packet must have come
// from a complete, in-range frame and must consume exactly that many bytes.
func checkRadiusFrameContract(t *testing.T, frame []byte, reader *bytes.Reader, packet *radius.Packet, err error) {
	t.Helper()

	if len(frame) < radiusFrameHeaderSize {
		if packet != nil || err == nil {
			t.Fatalf("partial header must produce an error and no packet: packet=%v err=%v", packet, err)
		}
		return
	}

	declared := binary.BigEndian.Uint16(frame[2:4])
	if packet == nil {
		if err == nil {
			t.Fatalf("no packet and no error for declared length %d", declared)
		}
		if errors.Is(err, errMalformedFrame) && declared >= 20 && declared <= maxRadiusPacketLength {
			t.Fatalf("framing error for in-range declared length %d: %v", declared, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("packet returned together with error: %v", err)
	}
	if declared < 20 || declared > maxRadiusPacketLength {
		t.Fatalf("packet accepted from out-of-range declared length %d", declared)
	}
	if len(frame) < int(declared) {
		t.Fatalf("packet accepted from a frame shorter than its declared length %d", declared)
	}
	if remaining := reader.Len(); remaining != len(frame)-int(declared) {
		t.Fatalf("parser consumed %d bytes, want exactly the declared %d", len(frame)-remaining, declared)
	}
}

// TestRadiusPacketCorpusContract is the deterministic, fixed-budget corpus
// regression required by ENG-011. It runs every named fixture exactly once and
// asserts the rejection contract, so CI catches parsing regressions without
// depending on fuzzing time.
func TestRadiusPacketCorpusContract(t *testing.T) {
	for _, tc := range radiusFrameCorpus() {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader(tc.frame)
			packet, err := parseTcpPacket(reader, radFuzzSecret)
			switch tc.want {
			case expectPacket:
				if err != nil {
					t.Fatalf("expected a packet, got error: %v", err)
				}
				if packet == nil {
					t.Fatal("expected a packet, got nil")
				}
				if packet.Code != tc.code {
					t.Fatalf("code = %d, want %d", packet.Code, tc.code)
				}
				if got := len(packet.Attributes); got != tc.attrCount {
					t.Fatalf("attribute count = %d, want %d", got, tc.attrCount)
				}
			case expectMalformedFrame:
				if !errors.Is(err, errMalformedFrame) {
					t.Fatalf("expected errMalformedFrame, got %v", err)
				}
			case expectShortRead:
				if !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
					t.Fatalf("expected a short read, got %v", err)
				}
			case expectBadAttributes:
				if err == nil {
					t.Fatal("expected attribute parsing to fail")
				}
				if errors.Is(err, errMalformedFrame) || errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
					t.Fatalf("expected an attribute error, got framing/stream error: %v", err)
				}
			}
			checkRadiusFrameContract(t, tc.frame, reader, packet, err)
		})
	}
}

// TestRadiusCorpusVendorAndMessageAuthenticatorDecoding asserts the semantic
// outcome of the corpus entries whose attributes carry security-relevant
// payloads: vendor TLVs and Message-Authenticator occurrences.
func TestRadiusCorpusVendorAndMessageAuthenticatorDecoding(t *testing.T) {
	parse := func(t *testing.T, frame []byte) *radius.Packet {
		t.Helper()
		packet, err := parseTcpPacket(bytes.NewReader(frame), radFuzzSecret)
		if err != nil {
			t.Fatalf("parseTcpPacket: %v", err)
		}
		return packet
	}

	t.Run("valid Huawei VSA decoded", func(t *testing.T) {
		packet := parse(t, radiusFrameWithAttributes(huaweiVSA([]byte{1, 6, 0, 0, 0x27, 0x10})))
		values, err := huawei.HuaweiInputBurstSize_Gets(packet)
		if err != nil {
			t.Fatalf("HuaweiInputBurstSize_Gets: %v", err)
		}
		if len(values) != 1 || uint32(values[0]) != 10000 {
			t.Fatalf("values = %v, want one value 10000", values)
		}
	})

	t.Run("malformed nested VSA TLV is ignored", func(t *testing.T) {
		packet := parse(t, radiusFrameWithAttributes(radiusAttribute(byte(rfc2865.VendorSpecific_Type), []byte{0, 0, 7, 0xDB, 1, 0})))
		values, err := huawei.HuaweiInputBurstSize_Gets(packet)
		if err != nil {
			t.Fatalf("HuaweiInputBurstSize_Gets: %v", err)
		}
		if len(values) != 0 {
			t.Fatalf("values = %v, want none", values)
		}
	})

	t.Run("service sees a zeroed message authenticator as invalid", func(t *testing.T) {
		packet := parse(t, radiusFrameWithAttributes(radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, md5.Size))))
		if got := (&RadiusService{}).verifyMessageAuthenticator(packet, radFuzzSecret); got != msgAuthInvalid {
			t.Fatalf("verifyMessageAuthenticator = %d, want msgAuthInvalid", got)
		}
	})

	t.Run("short message authenticator is invalid", func(t *testing.T) {
		packet := parse(t, radiusFrameWithAttributes(radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, 8))))
		if got := (&RadiusService{}).verifyMessageAuthenticator(packet, radFuzzSecret); got != msgAuthInvalid {
			t.Fatalf("verifyMessageAuthenticator = %d, want msgAuthInvalid", got)
		}
	})

	t.Run("duplicate message authenticator is invalid", func(t *testing.T) {
		packet := parse(t, radiusFrameWithAttributes(
			radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, md5.Size)),
			radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, md5.Size)),
		))
		if got := (&RadiusService{}).verifyMessageAuthenticator(packet, radFuzzSecret); got != msgAuthInvalid {
			t.Fatalf("verifyMessageAuthenticator = %d, want msgAuthInvalid", got)
		}
	})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// FuzzParseTcpPacket fuzzes the RadSec frame reader. `go test` runs the seed
// corpus under a fixed budget; `go test -fuzz=FuzzParseTcpPacket -fuzztime=30s`
// explores beyond it. Either way the target asserts two properties: no panic,
// and every rejection matches the RFC 2865 §3 framing contract.
func FuzzParseTcpPacket(f *testing.F) {
	for _, tc := range radiusFrameCorpus() {
		f.Add(tc.frame)
	}

	f.Fuzz(func(t *testing.T, frame []byte) {
		// Only the first 4100 bytes can influence parsing: a legal frame is at
		// most 4096 bytes and an illegal Length is rejected from the header.
		if len(frame) > maxRadiusPacketLength+radiusFrameHeaderSize {
			frame = frame[:maxRadiusPacketLength+radiusFrameHeaderSize]
		}

		reader := bytes.NewReader(frame)
		packet, err := parseTcpPacket(reader, radFuzzSecret)
		checkRadiusFrameContract(t, frame, reader, packet, err)

		// Parsing must be deterministic for identical input: the shared buffer
		// pool must not leak state between frames.
		replay, replayErr := parseTcpPacket(bytes.NewReader(frame), radFuzzSecret)
		if errString(err) != errString(replayErr) || (packet == nil) != (replay == nil) {
			t.Fatalf("non-deterministic parse: %v / %v", err, replayErr)
		}
	})
}

// FuzzParseAttributesAndVendorDecoding fuzzes attribute TLV parsing together
// with the vendor and Message-Authenticator readers built on top of it.
func FuzzParseAttributesAndVendorDecoding(f *testing.F) {
	for _, tc := range radiusFrameCorpus() {
		if len(tc.frame) > 20 {
			f.Add(tc.frame[20:])
		}
	}
	f.Add(radiusAttribute(byte(rfc2865.VendorSpecific_Type), []byte{0, 0, 7, 0xDB, 1, 6, 0, 0, 0x27, 0x10}))
	f.Add(radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, md5.Size)))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxRadiusPacketLength {
			data = data[:maxRadiusPacketLength]
		}

		attrs, err := radius.ParseAttributes(data)
		if err != nil {
			if attrs != nil {
				t.Fatalf("attributes returned together with error: %v", err)
			}
			return
		}

		packet := &radius.Packet{
			Code:       radius.CodeAccessRequest,
			Identifier: 1,
			Secret:     radFuzzSecret,
			Attributes: attrs,
		}

		// The vendor TLV walkers and the Message-Authenticator reader must
		// tolerate every parseable attribute graph, including nested lengths
		// that exceed their parent TLV.
		_, _ = huawei.HuaweiInputBurstSize_Gets(packet)
		_, _ = huawei.HuaweiOutputAverageRate_Gets(packet)
		_, _ = mikrotik.MikrotikRateLimit_Gets(packet)
		_, _ = rfc2869.MessageAuthenticator_Gets(packet)

		if wire, marshalErr := packet.MarshalBinary(); marshalErr == nil {
			if mac := computeMessageAuthenticator(wire, radFuzzSecret); len(mac) != md5.Size {
				t.Fatalf("Message-Authenticator length = %d, want %d", len(mac), md5.Size)
			}
		}

		switch got := (&RadiusService{}).verifyMessageAuthenticator(packet, radFuzzSecret); got {
		case msgAuthAbsent, msgAuthValid, msgAuthInvalid:
		default:
			t.Fatalf("unexpected messageAuthResult %d", got)
		}
	})
}

// FuzzMessageAuthenticator fuzzes the HMAC-MD5 computation (RFC 3579 §3.2) over
// arbitrary wire bytes: it must always return one MD5-sized value, must never
// mutate its input, and must be deterministic.
func FuzzMessageAuthenticator(f *testing.F) {
	f.Add([]byte{}, []byte{})
	f.Add(make([]byte, 20), []byte("secret"))
	wire := radiusFrameWithAttributes(radiusAttribute(byte(rfc2869.MessageAuthenticator_Type), make([]byte, md5.Size)))
	f.Add(wire, []byte("secret"))

	f.Fuzz(func(t *testing.T, wire, secret []byte) {
		if len(wire) > maxRadiusPacketLength {
			wire = wire[:maxRadiusPacketLength]
		}
		if len(secret) > 512 {
			secret = secret[:512]
		}

		original := append([]byte(nil), wire...)
		mac := computeMessageAuthenticator(wire, secret)
		if len(mac) != md5.Size {
			t.Fatalf("Message-Authenticator length = %d, want %d", len(mac), md5.Size)
		}
		if !bytes.Equal(wire, original) {
			t.Fatal("computeMessageAuthenticator mutated its input")
		}
		if !bytes.Equal(mac, computeMessageAuthenticator(wire, secret)) {
			t.Fatal("computeMessageAuthenticator is not deterministic")
		}
	})
}
