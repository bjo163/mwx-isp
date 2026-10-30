package teapmsg

import (
	"bytes"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionFitsTheThreeBitField(t *testing.T) {
	assert.Equal(t, uint8(1), Version, "RFC 9930 §4.1 documents version 1")
	assert.Zero(t, Version&^versionMask, "the encoder refuses a version the Ver field cannot hold")
}

func TestParseDecodesTheFields(t *testing.T) {
	outer := []byte{0x00, 0x01, 0x00, 0x00}
	tlsData := []byte{0x16, 0x03, 0x03, 0x00, 0x05}

	cases := []struct {
		name    string
		payload []byte
		want    Packet
	}{
		{
			name:    "acknowledgment",
			payload: []byte{Version},
			want:    Packet{Version: Version},
		},
		{
			name:    "reserved bit is ignored",
			payload: []byte{Version | flagReserved},
			want:    Packet{Version: Version},
		},
		{
			name:    "version above one is carried, not judged",
			payload: []byte{0x07},
			want:    Packet{Version: 7},
		},
		{
			name:    "length field only",
			payload: []byte{FlagLengthIncluded | Version, 0x00, 0x00, 0x00, 0x05},
			want:    Packet{Flags: FlagLengthIncluded, Version: Version, HasLength: true, MessageLength: 5},
		},
		{
			name:    "outer tlvs only",
			payload: append([]byte{FlagOuterTLVs | Version, 0x00, 0x00, 0x00, 0x04}, outer...),
			want:    Packet{Flags: FlagOuterTLVs, Version: Version, HasOuterTLVs: true, OuterTLVs: outer},
		},
		{
			name:    "tls data and outer tlvs",
			payload: append(append([]byte{FlagLengthIncluded | FlagOuterTLVs | Version, 0x00, 0x00, 0x00, 0x05, 0x00, 0x00, 0x00, 0x04}, tlsData...), outer...),
			want: Packet{
				Flags:         FlagLengthIncluded | FlagOuterTLVs,
				Version:       Version,
				HasLength:     true,
				MessageLength: 5,
				HasOuterTLVs:  true,
				TLSData:       tlsData,
				OuterTLVs:     outer,
			},
		},
		{
			name:    "empty outer tlv field",
			payload: []byte{FlagOuterTLVs | Version, 0x00, 0x00, 0x00, 0x00},
			want:    Packet{Flags: FlagOuterTLVs, Version: Version, HasOuterTLVs: true},
		},
		{
			name:    "tls data only",
			payload: append([]byte{Version}, tlsData...),
			want:    Packet{Version: Version, TLSData: tlsData},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.payload)
			require.NoError(t, err)
			assert.Equal(t, tc.want.Flags, got.Flags)
			assert.Equal(t, tc.want.Version, got.Version)
			assert.Equal(t, tc.want.HasLength, got.HasLength)
			assert.Equal(t, tc.want.MessageLength, got.MessageLength)
			assert.Equal(t, tc.want.HasOuterTLVs, got.HasOuterTLVs)
			assert.True(t, bytes.Equal(tc.want.TLSData, got.TLSData), "TLS data parses back")
			assert.True(t, bytes.Equal(tc.want.OuterTLVs, got.OuterTLVs), "Outer TLVs parse back")
		})
	}
}

func TestParseRejectsMalformedFraming(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
		want    error
	}{
		{name: "no flags octet", payload: nil, want: ErrEmptyPayload},
		{name: "truncated message length", payload: []byte{FlagLengthIncluded | Version, 0x00}, want: ErrShortLength},
		{name: "truncated outer tlv length", payload: []byte{FlagOuterTLVs | Version, 0x00, 0x00}, want: ErrShortOuterTLVs},
		{
			name:    "outer tlv length beyond the packet",
			payload: []byte{FlagOuterTLVs | Version, 0x00, 0x00, 0x00, 0x05, 0x00},
			want:    ErrShortOuterTLVs,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.payload)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestPacketPredicates(t *testing.T) {
	ack, err := Parse([]byte{Version})
	require.NoError(t, err)
	assert.True(t, ack.IsACK())
	assert.False(t, ack.More())
	assert.False(t, ack.Start())

	start, err := Parse([]byte{FlagStart | Version})
	require.NoError(t, err)
	assert.True(t, start.Start())
	assert.False(t, start.IsACK(), "a Start asks the peer for a version and a flight")

	more, err := Parse([]byte{FlagMoreFragments | Version, 0x01})
	require.NoError(t, err)
	assert.True(t, more.More())
	assert.False(t, more.IsACK())

	for _, flags := range []uint8{FlagLengthIncluded, FlagMoreFragments, FlagStart, FlagOuterTLVs} {
		p, err := Parse([]byte{flags | Version, 0x00, 0x00, 0x00, 0x00})
		require.NoError(t, err)
		assert.False(t, p.IsACK(), "flag %#x must not read as a bare acknowledgment", flags)
	}

	withData, err := Parse([]byte{Version, 0x01})
	require.NoError(t, err)
	assert.False(t, withData.IsACK(), "TLS data is a flight, not an acknowledgment")
}

// expectedFlags normalizes the flags a Packet declares through its fields: the
// encoder derives L and O from HasLength and HasOuterTLVs and never sets R.
func expectedFlags(in Packet) uint8 {
	flags := in.Flags & flagsMask &^ flagReserved
	if in.HasLength {
		flags |= FlagLengthIncluded
	} else {
		flags &^= FlagLengthIncluded
	}
	if in.HasOuterTLVs {
		flags |= FlagOuterTLVs
	} else {
		flags &^= FlagOuterTLVs
	}
	return flags
}

func TestFitsMessageStopsAtTheFourOctetBound(t *testing.T) {
	assert.True(t, fitsMessage(math.MaxUint32))
	assert.False(t, fitsMessage(math.MaxUint32+1))
}

func TestEncodeRoundTrips(t *testing.T) {
	cases := []struct {
		name string
		in   Packet
	}{
		{name: "acknowledgment", in: Packet{Version: Version}},
		{name: "start with outer tlvs", in: Packet{
			Flags: FlagStart, Version: Version, HasOuterTLVs: true, OuterTLVs: []byte{0x00, 0x01},
		}},
		{name: "length and tls data", in: Packet{
			Version: Version, HasLength: true, MessageLength: 3, Flags: FlagMoreFragments, TLSData: []byte{1, 2, 3},
		}},
		{name: "everything", in: Packet{
			Flags:         FlagLengthIncluded | FlagOuterTLVs,
			Version:       Version,
			HasLength:     true,
			MessageLength: 2,
			HasOuterTLVs:  true,
			TLSData:       []byte{9, 9},
			OuterTLVs:     []byte{0xAA},
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.in.Encode()
			require.NoError(t, err)
			got, err := Parse(encoded)
			require.NoError(t, err)

			assert.Equal(t, expectedFlags(tc.in), got.Flags)
			assert.Equal(t, tc.in.Version, got.Version)
			assert.Equal(t, tc.in.HasLength, got.HasLength)
			assert.Equal(t, tc.in.MessageLength, got.MessageLength)
			assert.Equal(t, tc.in.HasOuterTLVs, got.HasOuterTLVs)
			assert.True(t, bytes.Equal(tc.in.TLSData, got.TLSData), "TLS data survives the round trip")
			assert.True(t, bytes.Equal(tc.in.OuterTLVs, got.OuterTLVs), "Outer TLVs survive the round trip")
		})
	}
}

func TestEncodeDerivesTheFlagFromItsLength(t *testing.T) {
	// The flags a caller left set must not survive when the matching field is
	// absent, or the receiver would read octets that are not there.
	p := &Packet{Flags: FlagLengthIncluded | FlagOuterTLVs, Version: Version}
	encoded, err := p.Encode()
	require.NoError(t, err)
	assert.Equal(t, []byte{Version}, encoded)
}

func TestEncodeRefusesAVersionOutsideTheField(t *testing.T) {
	p := &Packet{Version: 8}
	_, err := p.Encode()
	assert.ErrorIs(t, err, ErrInvalidVersion)
}

func TestFragmentJoinsBackIntoTheMessage(t *testing.T) {
	tlsData := []byte("a TLS flight that does not fit one packet")
	outer := []byte{0x00, 0x01, 0x00, 0x00}

	packets, err := Fragment(Version, tlsData, outer, 10)
	require.NoError(t, err)
	require.Len(t, packets, (len(tlsData)+9)/10)
	assert.True(t, packets[0].HasLength, "the first fragment declares the message length")
	assert.Equal(t, len(tlsData), int(packets[0].MessageLength))
	assert.True(t, packets[0].HasOuterTLVs, "the Outer TLVs travel with the first fragment")
	assert.Equal(t, outer, packets[0].OuterTLVs)
	for i, p := range packets {
		assert.Equal(t, Version, p.Version)
		assert.Equal(t, i < len(packets)-1, p.More(), "every fragment but the last sets M")
		assert.LessOrEqual(t, len(p.TLSData), 10)
		if i > 0 {
			assert.False(t, p.HasLength, "only the first fragment declares the length")
			assert.Zero(t, len(p.OuterTLVs), "only the first fragment carries Outer TLVs")
		}
	}

	r := NewReassembler(0)
	for i, p := range packets {
		complete, err := r.Accept(p)
		require.NoError(t, err)
		assert.Equal(t, i == len(packets)-1, complete)
	}
	assert.Equal(t, tlsData, r.Buffer())
	assert.Equal(t, outer, r.OuterTLVs())
	declared, ok := r.Declared()
	assert.True(t, ok)
	assert.Equal(t, len(tlsData), int(declared))
}

func TestFragmentOfAnEmptyMessageCarriesOnlyTheOuterTLVs(t *testing.T) {
	outer := []byte{0x00, 0x01}
	packets, err := Fragment(Version, nil, outer, 0)
	require.NoError(t, err)
	require.Len(t, packets, 1)
	assert.False(t, packets[0].HasLength, "nothing is fragmented, so there is no length to declare")
	assert.False(t, packets[0].More())
	assert.True(t, packets[0].HasOuterTLVs)
	assert.Equal(t, outer, packets[0].OuterTLVs)
	assert.Empty(t, packets[0].TLSData)

	bare, err := Fragment(Version, nil, nil, 0)
	require.NoError(t, err)
	require.Len(t, bare, 1)
	assert.False(t, bare[0].HasOuterTLVs)
	assert.True(t, bare[0].IsACK(), "with no TLS data and no Outer TLVs it is a bare acknowledgment")
}

func TestFragmentRefusesAVersionOutsideTheField(t *testing.T) {
	_, err := Fragment(8, []byte{1}, nil, 0)
	assert.ErrorIs(t, err, ErrInvalidVersion)
}

func TestReassemblerAcceptsTheEmptyMessage(t *testing.T) {
	r := NewReassembler(0)
	complete, err := r.Accept(&Packet{Version: Version})
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Empty(t, r.Buffer())
}

func TestReassemblerRejectsBadSequences(t *testing.T) {
	cases := []struct {
		name  string
		setup func() *Reassembler
		frag  *Packet
		want  error
	}{
		{
			name:  "declared length above the maximum",
			setup: func() *Reassembler { return NewReassembler(4) },
			frag:  &Packet{Version: Version, HasLength: true, MessageLength: 5, TLSData: []byte{1}},
			want:  ErrMaxLengthExceeded,
		},
		{
			name:  "outer tlvs above the maximum",
			setup: func() *Reassembler { return NewReassembler(4) },
			frag:  &Packet{Version: Version, HasOuterTLVs: true, OuterTLVs: []byte{1, 2, 3, 4, 5}},
			want:  ErrMaxLengthExceeded,
		},
		{
			name: "accumulated data above the maximum",
			setup: func() *Reassembler {
				r := NewReassembler(4)
				_, err := r.Accept(&Packet{Version: Version, Flags: FlagMoreFragments, TLSData: []byte{1, 2, 3}})
				require.NoError(t, err)
				return r
			},
			frag: &Packet{Version: Version, TLSData: []byte{4, 5}},
			want: ErrMaxLengthExceeded,
		},
		{
			name: "data above the declared length",
			setup: func() *Reassembler {
				r := NewReassembler(0)
				_, err := r.Accept(&Packet{Version: Version, HasLength: true, MessageLength: 2, Flags: FlagMoreFragments, TLSData: []byte{1}})
				require.NoError(t, err)
				return r
			},
			frag: &Packet{Version: Version, TLSData: []byte{2, 3}},
			want: ErrDeclaredLengthExceeded,
		},
		{
			name: "final length mismatch",
			setup: func() *Reassembler {
				r := NewReassembler(0)
				_, err := r.Accept(&Packet{Version: Version, HasLength: true, MessageLength: 5, Flags: FlagMoreFragments, TLSData: []byte{1}})
				require.NoError(t, err)
				return r
			},
			frag: &Packet{Version: Version, TLSData: []byte{2}},
			want: ErrFinalLengthMismatch,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.setup().Accept(tc.frag)
			assert.ErrorIs(t, err, tc.want)
		})
	}
}

func TestReassemblerRejectsASecondLengthHeader(t *testing.T) {
	r := NewReassembler(0)
	_, err := r.Accept(&Packet{Version: Version, HasLength: true, MessageLength: 4, Flags: FlagMoreFragments, TLSData: []byte{1, 2}})
	require.NoError(t, err)

	_, err = r.Accept(&Packet{Version: Version, HasLength: true, MessageLength: 4, TLSData: []byte{3, 4}})
	assert.ErrorIs(t, err, ErrLengthOnContinuation)
}

func TestReassemblerRejectsOuterTLVsOnAContinuation(t *testing.T) {
	r := NewReassembler(0)
	_, err := r.Accept(&Packet{Version: Version, Flags: FlagMoreFragments, TLSData: []byte{1}})
	require.NoError(t, err)

	_, err = r.Accept(&Packet{Version: Version, HasOuterTLVs: true, OuterTLVs: []byte{1}})
	assert.ErrorIs(t, err, ErrOuterTLVsOnContinuation)
}

func TestLoadReassemblerContinuesAcrossRounds(t *testing.T) {
	r := LoadReassembler([]byte{1, 2}, 4, true, 0)
	declared, ok := r.Declared()
	assert.True(t, ok)
	assert.Equal(t, uint32(4), declared)
	assert.Equal(t, []byte{1, 2}, r.Buffer())

	complete, err := r.Accept(&Packet{Version: Version, TLSData: []byte{3, 4}})
	require.NoError(t, err)
	assert.True(t, complete)
	assert.Equal(t, []byte{1, 2, 3, 4}, r.Buffer())
	assert.Empty(t, r.OuterTLVs())

	// A loaded reassembler has already seen its first fragment, so a fresh L
	// header is a continuation violation rather than a restart.
	fresh := LoadReassembler(nil, 0, false, 0)
	_, err = fresh.Accept(&Packet{Version: Version, Flags: FlagMoreFragments, TLSData: []byte{1}})
	require.NoError(t, err)
	_, err = fresh.Accept(&Packet{Version: Version, HasLength: true, MessageLength: 2, TLSData: []byte{2}})
	assert.ErrorIs(t, err, ErrLengthOnContinuation)
}

func TestFragmentReassemblesByteForByte(t *testing.T) {
	payload := bytes.Repeat([]byte{0xA5}, 64)
	packets, err := Fragment(Version, payload, nil, 16)
	require.NoError(t, err)

	r := NewReassembler(0)
	for _, p := range packets {
		encoded, err := p.Encode()
		require.NoError(t, err)
		parsed, err := Parse(encoded)
		require.NoError(t, err)
		_, err = r.Accept(parsed)
		require.NoError(t, err)
	}
	assert.Equal(t, payload, r.Buffer())
}

func TestErrorMessagesNameThePackage(t *testing.T) {
	for _, err := range []error{
		ErrEmptyPayload, ErrShortLength, ErrShortOuterTLVs, ErrLengthOnContinuation,
		ErrOuterTLVsOnContinuation, ErrInvalidVersion, ErrDeclaredLengthExceeded,
		ErrMaxLengthExceeded, ErrFinalLengthMismatch,
	} {
		assert.True(t, bytes.HasPrefix([]byte(err.Error()), []byte("teap: ")), err.Error())
	}
}
