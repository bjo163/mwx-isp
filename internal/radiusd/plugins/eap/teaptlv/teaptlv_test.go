package teaptlv

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eapPacketBytes lays out an inner EAP packet, Code | Identifier | Length | Type |
// Data, with the Length counting the whole packet as RFC 9930 §4.2.10 requires an
// EAP-Payload TLV to carry.
func eapPacketBytes(identifier, code, typ byte, body []byte) []byte {
	packet := make([]byte, 4+1+len(body))
	packet[0] = code
	packet[1] = identifier
	binary.BigEndian.PutUint16(packet[2:4], uint16(len(packet))) //nolint:gosec // G115: test bodies are a handful of octets
	packet[4] = typ
	copy(packet[5:], body)
	return packet
}

// eapPayloadTLV is a test-side shorthand for an EAP-Payload TLV with no embedded TLVs.
func eapPayloadTLV(t *testing.T) TLV {
	t.Helper()
	tlv, err := NewEAPPayload(eapPacketBytes(1, 2, 43, []byte("id")))
	require.NoError(t, err)
	return tlv
}

// mustMarshal encodes TLVs and fails the test when the codec refuses them.
func mustMarshal(t *testing.T, tlvs ...TLV) []byte {
	t.Helper()
	data, err := Marshal(tlvs...)
	require.NoError(t, err)
	return data
}

// nonceWithLSB returns nonce with its least significant bit set to the given value,
// which §4.2.13 uses to tell a Binding Request nonce from a Binding Response nonce.
func nonceWithLSB(nonce []byte, lsb byte) []byte {
	out := append([]byte(nil), nonce...)
	out[len(out)-1] &= 0xfe
	out[len(out)-1] |= lsb
	return out
}

func TestTLVBytesHeaderLayout(t *testing.T) {
	t.Run("the M and R bits share the first octet with the type", func(t *testing.T) {
		tests := []struct {
			name string
			typ  Type
			mand bool
		}{
			{"optional authority id", TypeAuthorityID, false},
			{"mandatory authority id", TypeAuthorityID, true},
			{"highest type the field holds", Type(maxType), true},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				b, err := TLV{Type: tc.typ, Mandatory: tc.mand}.Bytes()
				require.NoError(t, err)
				require.Len(t, b, HeaderLen)
				word := binary.BigEndian.Uint16(b[0:2])
				assert.Equal(t, uint16(tc.typ), word&maxType)
				assert.Zero(t, word&0x4000, "the reserved bit is set to zero")
				if tc.mand {
					assert.NotZero(t, word&0x8000)
				} else {
					assert.Zero(t, word&0x8000)
				}
				assert.Equal(t, uint16(0), binary.BigEndian.Uint16(b[2:4]))
			})
		}
	})

	t.Run("the length counts the value only", func(t *testing.T) {
		b, err := TLV{Type: TypeEAPPayload, Value: []byte{1, 2, 3, 4, 5}}.Bytes()
		require.NoError(t, err)
		assert.Equal(t, uint16(5), binary.BigEndian.Uint16(b[2:4]))
		assert.Equal(t, []byte{1, 2, 3, 4, 5}, b[HeaderLen:])
	})

	t.Run("rejects what the header cannot hold", func(t *testing.T) {
		_, err := TLV{Type: maxType + 1}.Bytes()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "14-bit")

		_, err = TLV{Type: TypeAuthorityID, Value: make([]byte, maxValueLen+1)}.Bytes()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "two-octet Length")
	})
}

func TestMarshalAndParseRoundTrip(t *testing.T) {
	nested, err := NewIntermediateResult(StatusSuccess, NewAuthorityID([]byte("prod.example")))
	require.NoError(t, err)

	in := []TLV{
		NewResult(StatusFailure),
		NewIdentityType(IdentityMachine),
		NewAuthorityID([]byte("example")),
		nested,
		NewVendorSpecific(2011, false, []byte{0, 0, 0, 1}),
		NewBasicPasswordAuthReq("Passcode?"),
	}

	out, err := Parse(mustMarshal(t, in...))
	require.NoError(t, err)
	require.Len(t, out, len(in))
	for i := range in {
		assert.Equal(t, in[i].Type, out[i].Type)
		assert.Equal(t, in[i].Mandatory, out[i].Mandatory)
		assert.Equal(t, in[i].Value, out[i].Value)
	}
}

func TestParseRejections(t *testing.T) {
	t.Run("truncated header", func(t *testing.T) {
		_, err := Parse([]byte{0x80, 0x03, 0x00})
		require.ErrorIs(t, err, ErrTruncated)
		assert.Contains(t, err.Error(), "3 octets left, a TLV header needs 4")
	})

	t.Run("length overruns the payload", func(t *testing.T) {
		_, err := Parse([]byte{0x80, 0x03, 0x00, 0x03, 0x01})
		require.ErrorIs(t, err, ErrTruncated)
		assert.Contains(t, err.Error(), "TLV type 3 declares 3 octets, 1 remain")
	})

	t.Run("no octets yields no TLVs", func(t *testing.T) {
		tlvs, err := Parse(nil)
		require.NoError(t, err)
		assert.Empty(t, tlvs)
	})

	t.Run("zero-length value", func(t *testing.T) {
		tlvs, err := Parse([]byte{0x80, 0x01, 0x00, 0x00})
		require.NoError(t, err)
		require.Len(t, tlvs, 1)
		assert.Equal(t, TypeAuthorityID, tlvs[0].Type)
		assert.True(t, tlvs[0].Mandatory)
		assert.Empty(t, tlvs[0].Value)
	})

	t.Run("reserved bit is ignored on receipt", func(t *testing.T) {
		// A peer that sets the reserved bit must not be read as a different type.
		tlvs, err := Parse([]byte{0x40, 0x03, 0x00, 0x02, 0x00, 0x01})
		require.NoError(t, err)
		require.Len(t, tlvs, 1)
		assert.Equal(t, TypeResult, tlvs[0].Type)
		assert.False(t, tlvs[0].Mandatory)
	})

	t.Run("marshal propagates a member error", func(t *testing.T) {
		_, err := Marshal(TLV{Type: TypeResult}, TLV{Type: Type(maxType + 1)})
		require.ErrorIs(t, err, ErrInvalidTLV)
	})
}

func TestFind(t *testing.T) {
	tlvs, err := Parse(mustMarshal(t, NewAuthorityID([]byte("a")), NewResult(StatusSuccess)))
	require.NoError(t, err)
	require.NotNil(t, Find(tlvs, TypeResult))
	assert.Nil(t, Find(tlvs, TypeCryptoBinding))
	assert.Nil(t, Find(nil, TypeResult))
}

func TestResultTLV(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		for _, status := range []Status{StatusSuccess, StatusFailure} {
			tlvs, err := Parse(mustMarshal(t, NewResult(status)))
			require.NoError(t, err)
			require.Len(t, tlvs, 1)
			assert.True(t, tlvs[0].Mandatory, "§4.2.4 sets M to one")
			assert.Equal(t, uint16(2), binary.BigEndian.Uint16(mustMarshal(t, NewResult(status))[2:4]))
			got, err := tlvs[0].Status()
			require.NoError(t, err)
			assert.Equal(t, status, got)
		}
	})

	t.Run("unknown status is fatal", func(t *testing.T) {
		// §4.2.4: a Status that is not one of the known values is the fatal error
		// "Unexpected TLVs Exchanged", not an unknown outcome.
		_, err := TLV{Type: TypeResult, Mandatory: true, Value: []byte{0x00, 0x03}}.Status()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "§4.2.4")
	})

	t.Run("malformed", func(t *testing.T) {
		tests := []struct {
			name string
			tlv  TLV
			msg  string
		}{
			{"wrong type", TLV{Type: TypeIdentityType, Value: []byte{0, 1}}, "is not a Result TLV"},
			{"one octet", TLV{Type: TypeResult, Value: []byte{1}}, "Result Length is 1"},
			{"three octets", TLV{Type: TypeResult, Value: []byte{0, 0, 1}}, "Result Length is 3"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := tc.tlv.Status()
				require.ErrorIs(t, err, ErrInvalidTLV)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestIdentityTypeTLV(t *testing.T) {
	for _, identity := range []uint16{IdentityUser, IdentityMachine} {
		tlvs, err := Parse(mustMarshal(t, NewIdentityType(identity)))
		require.NoError(t, err)
		got, err := tlvs[0].IdentityType()
		require.NoError(t, err)
		assert.Equal(t, identity, got)
	}

	tests := []struct {
		name string
		tlv  TLV
		msg  string
	}{
		{"unassigned value", TLV{Type: TypeIdentityType, Value: []byte{0x00, 0x00}}, "not one of the values"},
		{"value above machine", TLV{Type: TypeIdentityType, Value: []byte{0x00, 0x03}}, "not one of the values"},
		{"one octet", TLV{Type: TypeIdentityType, Value: []byte{0x01}}, "fixes it at 2"},
		{"three octets", TLV{Type: TypeIdentityType, Value: []byte{0, 1, 0}}, "fixes it at 2"},
		{"wrong type", TLV{Type: TypeResult, Value: []byte{0x00, 0x01}}, "is not an Identity-Type"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.tlv.IdentityType()
			require.ErrorIs(t, err, ErrInvalidTLV)
			assert.Contains(t, err.Error(), tc.msg)
		})
	}
}

func TestEAPPayloadTLV(t *testing.T) {
	packet := eapPacketBytes(7, 2, 1, []byte{0, 1, 2, 3, 4})

	t.Run("round trip with a nested TLV", func(t *testing.T) {
		tlv, err := NewEAPPayload(packet, NewAuthorityID([]byte("hint")))
		require.NoError(t, err)
		require.True(t, tlv.Mandatory)

		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		gotPacket, nested, err := tlvs[0].EAPPayload()
		require.NoError(t, err)
		assert.Equal(t, packet, gotPacket, "the inner packet is passed through untouched")
		require.Len(t, nested, 1)
		assert.Equal(t, TypeAuthorityID, nested[0].Type)
	})

	t.Run("packet without nested TLVs", func(t *testing.T) {
		tlv, err := NewEAPPayload(packet)
		require.NoError(t, err)
		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		p, nested, err := tlvs[0].EAPPayload()
		require.NoError(t, err)
		assert.Equal(t, packet, p)
		assert.Empty(t, nested)
	})

	t.Run("nested TLV that cannot be encoded", func(t *testing.T) {
		_, err := NewEAPPayload(packet, TLV{Type: Type(maxType + 1)})
		require.ErrorIs(t, err, ErrInvalidTLV)
	})

	t.Run("encode rejects a packet that cannot be bounded", func(t *testing.T) {
		_, err := NewEAPPayload([]byte{1, 0, 0})
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "EAP packet header is 4 octets, got 3")

		_, err = NewEAPPayload(append([]byte(nil), packet[:len(packet)-1]...))
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "declares 10 octets but is 9")
	})

	t.Run("decode rejects a malformed TLV", func(t *testing.T) {
		tests := []struct {
			name string
			tlv  TLV
			want error
			msg  string
		}{
			{"shorter than an EAP header", TLV{Type: TypeEAPPayload, Value: []byte{1, 0, 0}}, ErrTruncated, "too short for an EAP header"},
			{"zero declared length", TLV{Type: TypeEAPPayload, Value: []byte{1, 0, 0, 0}}, ErrTruncated, "declares 0 octets"},
			{"length overruns the TLV", TLV{Type: TypeEAPPayload, Value: []byte{1, 0, 0, 9, 0}}, ErrTruncated, "declares 9 octets, 5 available"},
			{"trailing TLV header cut", TLV{Type: TypeEAPPayload, Value: append(eapPacketBytes(1, 2, 1, nil), 0x80, 0x03)}, ErrTruncated, "a TLV header needs"},
			{"wrong type", TLV{Type: TypeResult, Value: []byte{1, 0, 0, 4}}, ErrInvalidTLV, "is not an EAP-Payload"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := tc.tlv.EAPPayload()
				require.ErrorIs(t, err, tc.want)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestIntermediateResultTLV(t *testing.T) {
	t.Run("round trip with optional embedded TLVs", func(t *testing.T) {
		tlv, err := NewIntermediateResult(StatusSuccess,
			NewAuthorityID([]byte("a")), NewVendorSpecific(2011, false, nil))
		require.NoError(t, err)

		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		status, nested, err := tlvs[0].IntermediateResult()
		require.NoError(t, err)
		assert.Equal(t, StatusSuccess, status)
		require.Len(t, nested, 2)
		assert.Equal(t, TypeAuthorityID, nested[0].Type)
		assert.Equal(t, TypeVendorSpecific, nested[1].Type)
	})

	t.Run("no embedded TLVs", func(t *testing.T) {
		tlv, err := NewIntermediateResult(StatusFailure)
		require.NoError(t, err)
		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		status, nested, err := tlvs[0].IntermediateResult()
		require.NoError(t, err)
		assert.Equal(t, StatusFailure, status)
		assert.Empty(t, nested)
	})

	t.Run("a mandatory embedded TLV is refused on the way in", func(t *testing.T) {
		_, err := NewIntermediateResult(StatusSuccess, NewResult(StatusSuccess))
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "forbids the mandatory bit")

		_, err = NewIntermediateResult(StatusSuccess, NewAuthorityID(nil), NewResult(StatusFailure))
		require.ErrorIs(t, err, ErrInvalidTLV)
	})

	t.Run("an embedded TLV that cannot be encoded", func(t *testing.T) {
		_, err := NewIntermediateResult(StatusSuccess, TLV{Type: Type(maxType + 1)})
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "14-bit")
	})

	t.Run("a peer that sets M on an embedded TLV is refused on the way out", func(t *testing.T) {
		// Status=Success followed by an Authority-ID TLV with its M bit set.
		value := append([]byte{0x00, 0x01}, mustMarshal(t, TLV{Type: TypeAuthorityID, Mandatory: true})...)
		_, _, err := TLV{Type: TypeIntermediateResult, Value: value}.IntermediateResult()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "forbids the mandatory bit")
	})

	t.Run("malformed", func(t *testing.T) {
		tests := []struct {
			name string
			tlv  TLV
			want error
			msg  string
		}{
			{"wrong type", TLV{Type: TypeResult, Value: []byte{0, 1}}, ErrInvalidTLV, "is not an Intermediate-Result"},
			{"one octet", TLV{Type: TypeIntermediateResult, Value: []byte{1}}, ErrTruncated, "too short for a Status"},
			{"unknown status", TLV{Type: TypeIntermediateResult, Value: []byte{0, 9}}, ErrInvalidTLV, "§4.2.4"},
			{"cut embedded TLV", TLV{Type: TypeIntermediateResult, Value: []byte{0, 1, 0x80}}, ErrTruncated, "a TLV header needs"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := tc.tlv.IntermediateResult()
				require.ErrorIs(t, err, tc.want)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestVendorSpecificTLV(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		inner, err := Marshal(TLV{Type: TypeIdentityHint, Value: []byte{1, 2}})
		require.NoError(t, err)
		tlv := NewVendorSpecific(2011, true, inner)
		require.True(t, tlv.Mandatory, "a vendor may mark its TLV mandatory")

		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		vendorID, payload, err := tlvs[0].VendorSpecific()
		require.NoError(t, err)
		assert.Equal(t, uint32(2011), vendorID)
		assert.Equal(t, inner, payload)
	})

	t.Run("vendor id only", func(t *testing.T) {
		_, payload, err := NewVendorSpecific(1, false, nil).VendorSpecific()
		require.NoError(t, err)
		assert.Empty(t, payload)
	})

	t.Run("malformed", func(t *testing.T) {
		_, _, err := TLV{Type: TypeVendorSpecific, Value: []byte{0, 0, 1}}.VendorSpecific()
		require.ErrorIs(t, err, ErrTruncated)
		assert.Contains(t, err.Error(), "shorter than the 4-octet Vendor-Id")

		_, _, err = TLV{Type: TypeNAK, Value: []byte{0, 0, 0, 1}}.VendorSpecific()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "is not a Vendor-Specific")
	})

	t.Run("the caller's octets are copied", func(t *testing.T) {
		src := []byte{0x00, 0x00, 0x00, 0x01}
		tlv := NewVendorSpecific(2011, false, src)
		src[3] = 0x99
		_, payload, err := tlv.VendorSpecific()
		require.NoError(t, err)
		assert.Equal(t, byte(0x01), payload[3])
	})
}

func TestNAKTLV(t *testing.T) {
	t.Run("round trip with an explanation", func(t *testing.T) {
		tlv, err := NewNAK(TypeCryptoBinding, 0, NewAuthorityID([]byte("why")))
		require.NoError(t, err)
		require.True(t, tlv.Mandatory)

		tlvs, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		refused, vendorID, optional, err := tlvs[0].NAK()
		require.NoError(t, err)
		assert.Equal(t, TypeCryptoBinding, refused)
		assert.Equal(t, uint32(0), vendorID)
		require.Len(t, optional, 1)
		assert.Equal(t, TypeAuthorityID, optional[0].Type)
	})

	t.Run("bare NAK is six octets", func(t *testing.T) {
		tlv, err := NewNAK(Type(0xffff), 2011)
		require.NoError(t, err)
		wire := mustMarshal(t, tlv)
		assert.Equal(t, uint16(6), binary.BigEndian.Uint16(wire[2:4]), "§4.2.5 sets Length to at least 6")

		tlvs, err := Parse(wire)
		require.NoError(t, err)
		refused, vendorID, optional, err := tlvs[0].NAK()
		require.NoError(t, err)
		assert.Equal(t, Type(0xffff), refused, "the NAK-Type field is two octets, wider than the 14-bit TLV Type")
		assert.Equal(t, uint32(2011), vendorID)
		assert.Empty(t, optional)
	})

	t.Run("a mandatory embedded TLV is refused", func(t *testing.T) {
		_, err := NewNAK(TypeCryptoBinding, 0, NewResult(StatusFailure))
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "forbids the mandatory bit")

		value := append([]byte{0, 0, 0, 0, 0, 12}, mustMarshal(t, TLV{Type: TypeAuthorityID, Mandatory: true})...)
		_, _, _, err = TLV{Type: TypeNAK, Value: value}.NAK()
		require.ErrorIs(t, err, ErrInvalidTLV)
	})

	t.Run("malformed", func(t *testing.T) {
		_, _, _, err := TLV{Type: TypeNAK, Value: []byte{0, 0, 0, 0, 0}}.NAK()
		require.ErrorIs(t, err, ErrTruncated)
		assert.Contains(t, err.Error(), "requires at least 6")

		_, _, _, err = TLV{Type: TypeNAK, Value: append([]byte{0, 0, 0, 0, 0, 12}, 0x80)}.NAK()
		require.ErrorIs(t, err, ErrTruncated)

		_, _, _, err = TLV{Type: TypeResult, Value: []byte{0, 0, 0, 0, 0, 12}}.NAK()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "is not a NAK")

		_, err = NewNAK(TypeResult, 0, TLV{Type: Type(maxType + 1)})
		require.ErrorIs(t, err, ErrInvalidTLV)
	})
}

func TestBasicPasswordAuthTLVs(t *testing.T) {
	t.Run("request prompt", func(t *testing.T) {
		tlvs, err := Parse(mustMarshal(t, NewBasicPasswordAuthReq("Kata sandi")))
		require.NoError(t, err)
		require.True(t, tlvs[0].Mandatory)
		prompt, err := tlvs[0].Prompt()
		require.NoError(t, err)
		assert.Equal(t, "Kata sandi", prompt)

		empty, err := Parse(mustMarshal(t, NewBasicPasswordAuthReq("")))
		require.NoError(t, err)
		prompt, err = empty[0].Prompt()
		require.NoError(t, err)
		assert.Empty(t, prompt, "§4.2.14 makes the prompt optional")

		_, err = TLV{Type: TypeResult}.Prompt()
		require.ErrorIs(t, err, ErrInvalidTLV)
		assert.Contains(t, err.Error(), "is not a Basic-Password-Auth-Req")
	})

	t.Run("response credentials", func(t *testing.T) {
		// A password may hold octets below 0x20 including NUL, so it travels as octets,
		// not as a string (§4.2.15).
		tlv, err := NewBasicPasswordAuthResp([]byte("user@example"), []byte{0x00, 'p', 0x01})
		require.NoError(t, err)
		out, err := Parse(mustMarshal(t, tlv))
		require.NoError(t, err)
		user, pass, err := out[0].Credentials()
		require.NoError(t, err)
		assert.Equal(t, "user@example", string(user))
		assert.Equal(t, []byte{0x00, 'p', 0x01}, pass)
	})

	t.Run("encode enforces the length octets", func(t *testing.T) {
		tests := []struct {
			name string
			user []byte
			pass []byte
			want string
		}{
			{"empty username", nil, []byte{1}, "Userlen between 1 and 255, got 0"},
			{"empty password", []byte{1}, nil, "Passlen between 1 and 255, got 0"},
			{"username too long", make([]byte, maxPasswordField+1), []byte{1}, "Userlen between 1 and 255, got 256"},
			{"password too long", []byte{1}, make([]byte, maxPasswordField+1), "Passlen between 1 and 255, got 256"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := NewBasicPasswordAuthResp(tc.user, tc.pass)
				require.ErrorIs(t, err, ErrInvalidTLV)
				assert.Contains(t, err.Error(), tc.want)
			})
		}

		tlv, err := NewBasicPasswordAuthResp(make([]byte, maxPasswordField), make([]byte, maxPasswordField))
		require.NoError(t, err)
		user, pass, err := tlv.Credentials()
		require.NoError(t, err)
		assert.Len(t, user, maxPasswordField)
		assert.Len(t, pass, maxPasswordField)
	})

	t.Run("decode enforces the length octets", func(t *testing.T) {
		tests := []struct {
			name string
			tlv  TLV
			want error
			msg  string
		}{
			{"wrong type", TLV{Type: TypeResult, Value: []byte{1, 'a', 1, 'b'}}, ErrInvalidTLV, "is not a Basic-Password-Auth-Resp"},
			{"no octets", TLV{Type: TypeBasicPasswordAuthResp}, ErrTruncated, "too short for both length octets"},
			{"one octet", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{1}}, ErrTruncated, "too short for both length octets"},
			{"zero userlen", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{0, 1, 'b'}}, ErrInvalidTLV, "Userlen of zero"},
			{"username overruns", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{5, 'a', 'b'}}, ErrTruncated, "username declares 5"},
			{"username fills the TLV", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{2, 'a', 'b'}}, ErrTruncated, "username declares 2"},
			{"zero passlen", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{1, 'a', 0}}, ErrInvalidTLV, "Passlen of zero"},
			{"password overruns", TLV{Type: TypeBasicPasswordAuthResp, Value: []byte{1, 'a', 4, 'b'}}, ErrTruncated, "password declares 4"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, _, err := tc.tlv.Credentials()
				require.ErrorIs(t, err, tc.want)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestCryptoBindingEncodeDecode(t *testing.T) {
	nonce := make([]byte, nonceLen)
	for i := range nonce {
		nonce[i] = byte(i)
	}
	request := nonceWithLSB(nonce, 0)
	response := nonceWithLSB(nonce, 1)
	emsk := bytes.Repeat([]byte{0xe}, compoundMACLen)
	msk := bytes.Repeat([]byte{0x4d}, compoundMACLen)

	t.Run("76 octets with both MACs", func(t *testing.T) {
		tlv, err := NewCryptoBinding(Version, FlagEMSKPresent|FlagMSKPresent, SubTypeBindingRequest, request, emsk, msk)
		require.NoError(t, err)
		require.True(t, tlv.Mandatory)
		require.Len(t, tlv.Value, cryptoBindingLen)

		wire := mustMarshal(t, tlv)
		assert.Equal(t, uint16(cryptoBindingLen), binary.BigEndian.Uint16(wire[2:4]), "§4.2.13 sets Length to 76")
		assert.Zero(t, wire[4], "the Value opens with the Reserved octet")
		assert.Equal(t, Version, wire[5])
		assert.Equal(t, Version, wire[6])
		assert.Equal(t, FlagEMSKPresent|FlagMSKPresent, wire[7]>>4, "Flags is the high nibble")
		assert.Equal(t, SubTypeBindingRequest, wire[7]&0x0f, "Sub-Type is the low nibble")

		tlvs, err := Parse(wire)
		require.NoError(t, err)
		cb, err := tlvs[0].CryptoBinding()
		require.NoError(t, err)
		assert.Equal(t, Version, cb.ReceivedVersion)
		assert.Equal(t, FlagEMSKPresent|FlagMSKPresent, cb.Flags)
		assert.Equal(t, SubTypeBindingRequest, cb.SubType)
		assert.Equal(t, request, cb.Nonce)
		assert.Equal(t, emsk, cb.EMSKCompoundMAC)
		assert.Equal(t, msk, cb.MSKCompoundMAC)
	})

	t.Run("a decoded binding re-encodes byte for byte", func(t *testing.T) {
		// M11.4 will compare a peer's Binding Response against the expected value, so a
		// decoded TLV must round-trip without shifting a field between the two slots.
		for _, flags := range []uint8{FlagEMSKPresent, FlagMSKPresent, FlagEMSKPresent | FlagMSKPresent} {
			tlv, err := NewCryptoBinding(Version, flags, SubTypeBindingResponse, response, emsk, msk)
			require.NoError(t, err)
			tlvs, err := Parse(mustMarshal(t, tlv))
			require.NoError(t, err)
			cb, err := tlvs[0].CryptoBinding()
			require.NoError(t, err)
			again, err := cb.Value()
			require.NoError(t, err)
			assert.Equal(t, tlv.Value, again, "flags %d", flags)
		}
	})

	t.Run("flags pick which MAC is read back", func(t *testing.T) {
		tests := []struct {
			name     string
			flags    uint8
			wantEMSK []byte
			wantMSK  []byte
		}{
			{"emsk only", FlagEMSKPresent, emsk, nil},
			{"msk only", FlagMSKPresent, nil, msk},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tlv, err := NewCryptoBinding(Version, tc.flags, SubTypeBindingResponse, response, emsk, msk)
				require.NoError(t, err)
				tlvs, err := Parse(mustMarshal(t, tlv))
				require.NoError(t, err)
				cb, err := tlvs[0].CryptoBinding()
				require.NoError(t, err)
				assert.Equal(t, tc.wantEMSK, cb.EMSKCompoundMAC)
				assert.Equal(t, tc.wantMSK, cb.MSKCompoundMAC)
			})
		}
	})

	t.Run("the absent MAC keeps its slot and is zero-filled", func(t *testing.T) {
		// §6.3 computes the MAC over the TLV with both MAC fields zeroed, so a slot the
		// Flags do not announce has to stay zero on the wire, in its own position.
		emskOnly, err := NewCryptoBinding(Version, FlagEMSKPresent, SubTypeBindingRequest, request, emsk, msk)
		require.NoError(t, err)
		assert.Equal(t, bytes.Repeat([]byte{0}, compoundMACLen), emskOnly.Value[4+nonceLen+compoundMACLen:])
		assert.Equal(t, emsk, emskOnly.Value[4+nonceLen:4+nonceLen+compoundMACLen])

		mskOnly, err := NewCryptoBinding(Version, FlagMSKPresent, SubTypeBindingRequest, request, emsk, msk)
		require.NoError(t, err)
		assert.Equal(t, bytes.Repeat([]byte{0}, compoundMACLen), mskOnly.Value[4+nonceLen:4+nonceLen+compoundMACLen])
		assert.Equal(t, msk, mskOnly.Value[4+nonceLen+compoundMACLen:])

		tlvs, err := Parse(mustMarshal(t, mskOnly))
		require.NoError(t, err)
		cb, err := tlvs[0].CryptoBinding()
		require.NoError(t, err)
		assert.Nil(t, cb.EMSKCompoundMAC)
		assert.Equal(t, msk, cb.MSKCompoundMAC)
	})

	t.Run("encode rejects what §4.2.13 does not define", func(t *testing.T) {
		tests := []struct {
			name    string
			version uint8
			flags   uint8
			sub     uint8
			nonce   []byte
			emsk    []byte
			msk     []byte
			msg     string
		}{
			{"short nonce", Version, FlagEMSKPresent, SubTypeBindingRequest, request[:nonceLen-1], emsk, nil, "fixes it at 32"},
			{"zero flags", Version, 0, SubTypeBindingRequest, request, emsk, nil, "Flags 0"},
			{"unknown flags", Version, 4, SubTypeBindingRequest, request, emsk, nil, "Flags 4"},
			{"unknown sub-type", Version, FlagEMSKPresent, 2, request, emsk, nil, "Sub-Type 2"},
			{"wrong received version", 0, FlagEMSKPresent, SubTypeBindingRequest, request, emsk, nil, "Received-Ver is 0"},
			{"response nonce bit clear", Version, FlagEMSKPresent, SubTypeBindingResponse, request, emsk, nil, "Response nonce must end"},
			{"request nonce bit set", Version, FlagEMSKPresent, SubTypeBindingRequest, response, emsk, nil, "Request nonce must end"},
			{"emsk too short", Version, FlagEMSKPresent, SubTypeBindingRequest, request, emsk[:19], nil, "EMSK Compound MAC is 19"},
			{"msk too short", Version, FlagMSKPresent, SubTypeBindingRequest, request, nil, msk[:19], "MSK Compound MAC is 19"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				_, err := NewCryptoBinding(tc.version, tc.flags, tc.sub, tc.nonce, tc.emsk, tc.msk)
				require.ErrorIs(t, err, ErrInvalidTLV)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})

	t.Run("decode rejects a tampered value", func(t *testing.T) {
		good, err := NewCryptoBinding(Version, FlagEMSKPresent|FlagMSKPresent, SubTypeBindingRequest, request, emsk, msk)
		require.NoError(t, err)

		tests := []struct {
			name string
			poke func(v []byte) []byte
			msg  string
		}{
			{"wrong type", nil, "is not a Crypto-Binding"},
			{"short", func(v []byte) []byte { return v[:len(v)-1] }, "Length is 75"},
			{"reserved octet", func(v []byte) []byte { v[0] = 1; return v }, "Reserved octet"},
			{"version", func(v []byte) []byte { v[1] = 2; return v }, "Version is 2"},
			{"received version", func(v []byte) []byte { v[2] = 0; return v }, "Received-Ver is 0"},
			{"flags", func(v []byte) []byte { v[3] = 0x40 | SubTypeBindingRequest; return v }, "Flags 4"},
			{"sub-type", func(v []byte) []byte { v[3] = FlagEMSKPresent<<4 | 9; return v }, "Sub-Type 9"},
			{"request nonce lsb", func(v []byte) []byte { v[4+nonceLen-1] |= 1; return v }, "Request nonce must end"},
			{"response nonce lsb", func(v []byte) []byte {
				v[3] = FlagEMSKPresent<<4 | SubTypeBindingResponse
				v[4+nonceLen-1] &= 0xfe
				return v
			}, "Response nonce must end"},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				tlv := good
				if tc.poke != nil {
					tlv.Value = tc.poke(append([]byte(nil), good.Value...))
				} else {
					tlv.Type = TypeResult
				}
				_, err := tlv.CryptoBinding()
				require.ErrorIs(t, err, ErrInvalidTLV)
				assert.Contains(t, err.Error(), tc.msg)
			})
		}
	})
}

func TestResponseNonce(t *testing.T) {
	request := make([]byte, nonceLen)
	for i := range request {
		request[i] = byte(i * 7)
	}
	request[nonceLen-1] &= 0xfe

	response, err := ResponseNonce(request)
	require.NoError(t, err)
	require.Len(t, response, nonceLen)
	assert.Equal(t, request[nonceLen-1]|1, response[nonceLen-1], "only the least significant bit changes")
	for i := 0; i < nonceLen-1; i++ {
		assert.Equal(t, request[i], response[i])
	}

	set := nonceWithLSB(request, 1)
	_, err = ResponseNonce(set)
	require.ErrorIs(t, err, ErrInvalidTLV)
	assert.Contains(t, err.Error(), "Binding Request nonce must end")

	_, err = ResponseNonce(request[:nonceLen-1])
	require.ErrorIs(t, err, ErrInvalidTLV)
	assert.Contains(t, err.Error(), "fixes it at 32")
}

func TestTLVTypesMatchRFC9930(t *testing.T) {
	// The enum is the codec's whole claim about what a peer may send; one drifted
	// constant would rename a TLV in a reject log.
	allocated := []Type{
		TypeAuthorityID, TypeIdentityType, TypeResult, TypeNAK, TypeError, TypeChannelBinding,
		TypeVendorSpecific, TypeRequestAction, TypeEAPPayload, TypeIntermediateResult, TypePAC,
		TypeCryptoBinding, TypeBasicPasswordAuthReq, TypeBasicPasswordAuthResp, TypePKCS7,
		TypePKCS10, TypeTrustedServerRoot, TypeCSRAttributes, TypeIdentityHint,
	}
	require.Len(t, allocated, 19, "RFC 9930 §4.2 allocates nineteen TLV types")

	seen := map[Type]bool{}
	for _, typ := range allocated {
		require.False(t, seen[typ], "type %d is declared twice", typ)
		seen[typ] = true
	}
	for n := Type(1); n <= Type(19); n++ {
		assert.True(t, seen[n], "TLV type %d is missing from the enum", n)
	}

	t.Run("type 0 is unassigned", func(t *testing.T) {
		assert.Equal(t, Type(0), TypeUnassigned)
	})

	t.Run("the crypto binding is a 76-octet value", func(t *testing.T) {
		assert.Equal(t, 4+nonceLen+2*compoundMACLen, cryptoBindingLen)
	})
}

func TestOuterTLVSequenceKeepsItsOrder(t *testing.T) {
	// §3.6 exchanges the outer TLVs as a sequence; the codec must not reorder them.
	data, err := Marshal(NewIdentityType(IdentityUser), eapPayloadTLV(t), NewResult(StatusSuccess))
	require.NoError(t, err)
	tlvs, err := Parse(data)
	require.NoError(t, err)
	require.Len(t, tlvs, 3)
	assert.Equal(t, []Type{TypeIdentityType, TypeEAPPayload, TypeResult},
		[]Type{tlvs[0].Type, tlvs[1].Type, tlvs[2].Type})
}
