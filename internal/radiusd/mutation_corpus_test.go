package radiusd

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"layeh.com/radius"
)

// mutationSeedLimit keeps the generated corpus to small frames. The 4096-byte
// boundary frame is already covered by the fixed ENG-011 corpus; mutating it
// would multiply the golden file size without adding a new failure mode.
const mutationSeedLimit = 512

// Mutation kinds. Each kind has one physical cause, so a failure names the
// class of defect that reappeared.
const (
	// mutFlipBit flips the low bit of the code, identifier or last byte.
	mutFlipBit = "flip_bit"
	// mutTruncate cuts the frame short without changing its Length field.
	mutTruncate = "truncate"
	// mutDeclaredLength rewrites the Length field independently of the body.
	mutDeclaredLength = "declared_length"
	// mutAttributeLength rewrites the first attribute's Length byte.
	mutAttributeLength = "attribute_length"
	// mutDuplicateAttributes repeats the attribute list inside one frame.
	mutDuplicateAttributes = "duplicate_attributes"
	// mutDropByteKeepLength removes the last byte and leaves Length alone.
	mutDropByteKeepLength = "drop_byte_keep_length"
	// mutDropByteFixLength removes the last byte and shrinks Length to match.
	mutDropByteFixLength = "drop_byte_fix_length"
)

// mutationCase is one generated mutant and the parse outcome the current parser
// produces for it.
type mutationCase struct {
	Name       string `json:"name"`
	Seed       string `json:"seed"`
	Kind       string `json:"kind"`
	Frame      string `json:"frame"`
	Outcome    string `json:"outcome"`
	Code       int    `json:"code,omitempty"`
	Attributes int    `json:"attributes,omitempty"`
}

// Outcome names recorded in the golden file.
const (
	outcomePacket         = "packet"
	outcomeMalformedFrame = "malformed_frame"
	outcomeShortRead      = "short_read"
	outcomeBadAttributes  = "bad_attributes"
)

var updateRadiusMutationCorpus = flag.Bool("update-radius-mutation-corpus", false,
	"rewrite testdata/radius_mutation_corpus.json from the current mutator and parser")

type mutation struct {
	name  string
	kind  string
	frame []byte
}

// mutationsFor derives the deterministic mutant set of one valid seed frame.
// The mutations are positional rather than random, so the same seed always
// yields the same corpus and a reviewed diff shows exactly which bytes moved.
func mutationsFor(seedName string, seed []byte) []mutation {
	var mutants []mutation
	add := func(kind, suffix string, frame []byte) {
		if len(frame) == 0 {
			return
		}
		mutants = append(mutants, mutation{
			name:  seedName + "/" + suffix,
			kind:  kind,
			frame: append([]byte(nil), frame...),
		})
	}

	flip := func(position int) {
		if position < 0 || position >= len(seed) {
			return
		}
		frame := append([]byte(nil), seed...)
		frame[position] ^= 0x01
		add(mutFlipBit, "flip byte "+itoa(position), frame)
	}
	if len(seed) > 0 {
		flip(0)
		flip(1)
		flip(len(seed) - 1)
	}

	for _, size := range []int{len(seed) / 2, len(seed) - 1, len(seed) - 4} {
		if size > 0 && size < len(seed) {
			add(mutTruncate, "truncate to "+itoa(size), seed[:size])
		}
	}

	for _, declared := range []int{0, 19, 20, 4097, len(seed) - 1, len(seed) + 1} {
		if declared < 0 || declared > 0xFFFF || declared == len(seed) {
			continue
		}
		frame := append([]byte(nil), seed...)
		binary.BigEndian.PutUint16(frame[2:4], uint16(declared))
		add(mutDeclaredLength, "declare "+itoa(declared), frame)
	}

	if len(seed) > 21 {
		for _, length := range []byte{0, 1, 255} {
			frame := append([]byte(nil), seed...)
			frame[21] = length
			add(mutAttributeLength, "first attribute length "+itoa(int(length)), frame)
		}
	}

	if len(seed) > 20 {
		frame := append([]byte(nil), seed...)
		frame = append(frame, seed[20:]...)
		if len(frame) <= maxRadiusPacketLength {
			binary.BigEndian.PutUint16(frame[2:4], uint16(len(frame))) //nolint:gosec // G115: bounded by maxRadiusPacketLength
			add(mutDuplicateAttributes, "duplicate attribute list", frame)
		}

		add(mutDropByteKeepLength, "drop last byte keep length", seed[:len(seed)-1])

		shrunk := append([]byte(nil), seed[:len(seed)-1]...)
		if len(shrunk) >= 20 {
			binary.BigEndian.PutUint16(shrunk[2:4], uint16(len(shrunk))) //nolint:gosec // G115: seed length is bounded by mutationSeedLimit
			add(mutDropByteFixLength, "drop last byte fix length", shrunk)
		}
	}

	return mutants
}

// classifyMutation parses one mutant and names the outcome class the parser
// produces.
func classifyMutation(frame []byte) (outcome string, code, attributes int) {
	packet, err := parseTcpPacket(bytes.NewReader(frame), radFuzzSecret)
	switch {
	case err == nil && packet != nil:
		return outcomePacket, int(packet.Code), len(packet.Attributes)
	case errors.Is(err, errMalformedFrame):
		return outcomeMalformedFrame, 0, 0
	case errors.Is(err, io.ErrUnexpectedEOF), errors.Is(err, io.EOF):
		return outcomeShortRead, 0, 0
	default:
		return outcomeBadAttributes, 0, 0
	}
}

// generateMutationCorpus walks every valid seed frame of the fixed corpus and
// records each mutant with its parse outcome.
func generateMutationCorpus() []mutationCase {
	var cases []mutationCase
	for _, seed := range radiusFrameCorpus() {
		if seed.want != expectPacket || len(seed.frame) > mutationSeedLimit {
			continue
		}
		for _, mutant := range mutationsFor(seed.name, seed.frame) {
			outcome, code, attributes := classifyMutation(mutant.frame)
			cases = append(cases, mutationCase{
				Name:       mutant.name,
				Seed:       seed.name,
				Kind:       mutant.kind,
				Frame:      hex.EncodeToString(mutant.frame),
				Outcome:    outcome,
				Code:       code,
				Attributes: attributes,
			})
		}
	}
	return cases
}

func itoa(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for value > 0 {
		i--
		digits[i] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[i:])
}

// TestRadiusMutationCorpusMatchesGolden keeps the generated mutants and the
// outcomes the parser produced for them under version control. A parser change
// that alters any outcome fails here; an intended change regenerates the golden
// with -update-radius-mutation-corpus and shows the diff for review.
func TestRadiusMutationCorpusMatchesGolden(t *testing.T) {
	generated := generateMutationCorpus()
	require.NotEmpty(t, generated)

	path := filepath.Join("testdata", "radius_mutation_corpus.json")
	if *updateRadiusMutationCorpus {
		raw, err := json.MarshalIndent(generated, "", "  ")
		require.NoError(t, err)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
		require.NoError(t, os.WriteFile(path, append(raw, '\n'), 0o644)) //nolint:gosec // G306: reviewed test fixture
	}

	raw, err := os.ReadFile(path) //nolint:gosec // G304: fixed testdata path
	require.NoError(t, err, "run go test ./internal/radiusd -update-radius-mutation-corpus to create the golden")
	var golden []mutationCase
	require.NoError(t, json.Unmarshal(raw, &golden))
	require.Equal(t, generated, golden, "the golden corpus is stale; regenerate and review the diff")

	for _, tc := range golden {
		t.Run(tc.Name, func(t *testing.T) {
			frame, err := hex.DecodeString(tc.Frame)
			require.NoError(t, err)
			reader := bytes.NewReader(frame)
			packet, parseErr := parseTcpPacket(reader, radFuzzSecret)

			switch tc.Outcome {
			case outcomePacket:
				require.NoError(t, parseErr)
				require.NotNil(t, packet)
				assert.Equal(t, radius.Code(tc.Code), packet.Code)
				assert.Len(t, packet.Attributes, tc.Attributes)
			case outcomeMalformedFrame:
				assert.ErrorIs(t, parseErr, errMalformedFrame)
			case outcomeShortRead:
				assert.True(t, errors.Is(parseErr, io.ErrUnexpectedEOF) || errors.Is(parseErr, io.EOF),
					"expected a short read, got %v", parseErr)
			case outcomeBadAttributes:
				require.Error(t, parseErr)
				assert.False(t, errors.Is(parseErr, errMalformedFrame), "attribute mutant reported as a framing error")
				assert.False(t, errors.Is(parseErr, io.ErrUnexpectedEOF), "attribute mutant reported as a short read")
				assert.False(t, errors.Is(parseErr, io.EOF), "attribute mutant reported as a short read")
			default:
				t.Fatalf("unknown outcome %q", tc.Outcome)
			}

			checkRadiusFrameContract(t, frame, reader, packet, parseErr)
		})
	}
}

// TestRadiusMutationGeneratorIsDeterministic pins that two runs produce the
// same mutants in the same order, so the golden diff never contains random
// churn.
func TestRadiusMutationGeneratorIsDeterministic(t *testing.T) {
	first := generateMutationCorpus()
	second := generateMutationCorpus()
	assert.Equal(t, first, second)
}

// TestRadiusMutationGeneratorCoversEveryKind guards the mutator itself: a kind
// that silently stops producing cases would shrink the regression surface
// without failing any byte-level assertion.
func TestRadiusMutationGeneratorCoversEveryKind(t *testing.T) {
	kinds := make(map[string]int)
	seeds := make(map[string]bool)
	for _, tc := range generateMutationCorpus() {
		kinds[tc.Kind]++
		seeds[tc.Seed] = true
	}

	for _, kind := range []string{
		mutFlipBit, mutTruncate, mutDeclaredLength, mutAttributeLength,
		mutDuplicateAttributes, mutDropByteKeepLength, mutDropByteFixLength,
	} {
		assert.NotZero(t, kinds[kind], "mutation kind %s produced no cases", kind)
	}
	assert.GreaterOrEqual(t, len(seeds), 4, "expected several valid seeds, got %d", len(seeds))
}
