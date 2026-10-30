package teapkeys

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// rfc9427Path is the vendored text of the specification that registers these labels.
// The tests read it, so a label that stops appearing in the document stops passing
// here. The separators are forward slashes because Go accepts them on every platform.
const rfc9427Path = "../../../../../docs/rfcs/rfc9427-tls-based-eap-types-tls13.txt"

// call is one export request exactly as the package made it.
type call struct {
	label   string
	context []byte
	length  int
}

// recorder is a fake TLS exporter that keeps every request and answers it with key
// material derived from that request. Deriving from the input matters: a constant
// pattern would let a chain that folds the wrong secret forward look correct.
type recorder struct {
	calls []call
	// failCall is the one-based index of the call that hands back one octet fewer than
	// asked for; zero fails none. It stands in for the RFC 9427 §2.1 mistake of
	// exporting the wrong length.
	failCall int
	// err, when set, is returned by every call, standing in for a session that refuses
	// to export at all.
	err error
}

func (r *recorder) ExportKey(label string, context []byte, length int) ([]byte, error) {
	r.calls = append(r.calls, call{label: label, context: bytes.Clone(context), length: length})
	if r.err != nil {
		return nil, r.err
	}
	if r.failCall == len(r.calls) {
		return make([]byte, length-1), nil
	}
	return expand(label, context, length), nil
}

// expand is the recorder's stand-in for the TLS 1.3 exporter: it hashes the label,
// the context and the length, so each distinct request yields distinct octets and the
// length changes the answer, as RFC 9427 §2.1 says a TLS 1.3 export does.
func expand(label string, context []byte, length int) []byte {
	out := make([]byte, 0, length+sha256.Size)
	for counter := byte(0); len(out) < length; counter++ {
		h := sha256.New()
		h.Write([]byte(label))
		h.Write(context)
		h.Write([]byte{byte(length), counter}) //nolint:gosec // G115: the lengths here are 20 to 64, so one octet holds them exactly.
		out = append(out, h.Sum(nil)...)
	}
	return out[:length]
}

// request returns the n'th recorded export, one-based, so a test names the derivation
// it means instead of counting in prose.
func (r *recorder) request(t *testing.T, n int) call {
	t.Helper()
	require.LessOrEqual(t, n, len(r.calls), "expected at least %d export calls, got %s", n, r.labels())
	return r.calls[n-1]
}

// labels renders the recorded request order for a failure message.
func (r *recorder) labels() []string {
	parts := make([]string, 0, len(r.calls))
	for _, c := range r.calls {
		parts = append(parts, c.label)
	}
	return parts
}

// secret returns n recognizable octets for a method key handed to the chain.
func secret(n int) []byte {
	out := make([]byte, n)
	for i := range out {
		out[i] = byte(i)
	}
	return out
}

func TestTypeIsTheEAPTypeRFC9930Quotes(t *testing.T) {
	assert.Equal(t, uint8(0x37), Type, "RFC 9930 §7.5 assigns TEAP method type 55")

	raw := collapseSpace(readSpec(t, rfc9427Path))
	// The seed is exported with the Type as its context, so the constant and the clause
	// that uses it must be read together.
	assert.Contains(t, raw, `TLS-Exporter("EXPORTER: teap session key seed", Type, 40)`)
}

// TestExporterLabelsAreInTheSpecification is this package's anti-fabrication gate:
// every label and length is quoted from the vendored RFC 9427 §2.2, because a plausible
// label no peer derives fails interoperability silently.
func TestExporterLabelsAreInTheSpecification(t *testing.T) {
	raw := collapseSpace(readSpec(t, rfc9427Path))
	for _, quote := range []string{
		`IMSK[j] = TLS-Exporter("TEAPbindkey@ietf.org", secret, 32)`,
		`session_key_seed = TLS-Exporter("EXPORTER: teap session key seed", Type, 40)`,
		`IMCK[j] = TLS-Exporter("EXPORTER: Inner Methods Compound Keys", S-IMCK[j-1] || IMSK[j], 60)`,
		`MSK = TLS-Exporter( "EXPORTER: Session Key Generating Function", S-IMCK[n], 64)`,
		`EMSK = TLS-Exporter( "EXPORTER: Extended Session Key Generating Function", S-IMCK[n], 64)`,
	} {
		assert.Contains(t, raw, quote, "the derivation this package implements is not in RFC 9427 §2.2")
	}
}

func TestSessionKeySeedRequest(t *testing.T) {
	var r recorder
	seed, err := SessionKeySeed(&r)
	require.NoError(t, err)

	require.Len(t, r.calls, 1)
	assert.Equal(t, labelSessionKeySeed, r.calls[0].label)
	assert.Equal(t, []byte{0x37}, r.calls[0].context, "RFC 9427 §2.2 uses the Type as the context")
	assert.Equal(t, sIMCKLen, r.calls[0].length)
	assert.Equal(t, expand(labelSessionKeySeed, []byte{0x37}, sIMCKLen), seed)
}

func TestExportRefusesWrongLength(t *testing.T) {
	// RFC 9427 §2.1: TLS 1.3 exporter output depends on the length, so an answer with
	// the wrong count is a broken derivation, not a key to be truncated.
	var r recorder
	r.failCall = 1
	_, err := SessionKeySeed(&r)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrKeyLength)
	assert.Contains(t, err.Error(), "asked for 40 octets and got 39")
}

func TestExportPropagatesHandshakeFailure(t *testing.T) {
	sentinel := errors.New("tls: handshake not completed")
	var r recorder
	r.err = sentinel
	_, err := SessionKeySeed(&r)
	require.Error(t, err)
	assert.ErrorIs(t, err, sentinel)
	assert.Contains(t, err.Error(), `export "EXPORTER: teap session key seed"`)

	var nilExporter Exporter
	_, err = SessionKeySeed(nilExporter)
	assert.ErrorIs(t, err, ErrNoExporter, "there is no session to export from before phase 1 ends")
}

func TestAdvanceRunsBothChainsInOrder(t *testing.T) {
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)

	msk, emsk := secret(64), secret(64)[32:]
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: msk, EMSK: emsk}))

	assert.Equal(t,
		[]string{labelIMSK, labelInnerMethods, labelIMSK, labelInnerMethods},
		r.labels(), "IMSK then IMCK on the MSK path first, then the EMSK path (RFC 9930 §6.2.1)")

	assert.Equal(t, msk, r.request(t, 1).context, "the method MSK is the first IMSK secret")
	assert.Equal(t, emsk, r.request(t, 3).context, "the method EMSK is the second IMSK secret")
	for _, n := range []int{1, 3} {
		assert.Equal(t, imskLen, r.request(t, n).length)
	}
	for _, n := range []int{2, 4} {
		assert.Equal(t, imckLen, r.request(t, n).length)
	}
}

func TestAdvanceThreadsSeedIntoFirstCompoundKey(t *testing.T) {
	seed := secret(sIMCKLen)
	var r recorder
	chain, err := NewChain(seed)
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))

	imck := r.request(t, 2)
	assert.Equal(t, labelInnerMethods, imck.label)
	assert.Equal(t, imckLen, imck.length)
	require.Len(t, imck.context, sIMCKLen+imskLen)
	assert.Equal(t, seed, imck.context[:sIMCKLen], "S-IMCK[j-1] comes first (RFC 9427 §2.2)")
	assert.Equal(t, expand(labelIMSK, secret(32), imskLen), imck.context[sIMCKLen:], "then IMSK[j]")
}

func TestAdvanceSplitsCompoundKey(t *testing.T) {
	seed := secret(sIMCKLen)
	imsk := expand(labelIMSK, secret(32), imskLen)
	var r recorder
	chain, err := NewChain(seed)
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))

	imck := expand(labelInnerMethods, append(append([]byte(nil), seed...), imsk...), imckLen)
	mskCMK, emskCMK := chain.CompoundMACKeys()
	assert.Equal(t, imck[sIMCKLen:], mskCMK, "CMK[j] is the last 20 octets of IMCK[j]")
	assert.Len(t, mskCMK, cmkLen)
	assert.Nil(t, emskCMK, "a method without an EMSK never advances the EMSK path")
}

func TestAdvanceSubstitutesZeroIMSK(t *testing.T) {
	// RFC 9930 §6.2.1: a method that yields neither MSK nor EMSK - the
	// Basic-Password-Auth TLVs - contributes 32 zero octets rather than nothing.
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{}))

	assert.Equal(t, []string{labelInnerMethods}, r.labels(), "no secret means no IMSK export")
	imck := r.request(t, 1)
	assert.Equal(t, make([]byte, imskLen), imck.context[sIMCKLen:], "the all-zero IMSK")
}

func TestAdvanceCarriesEMSKChainForward(t *testing.T) {
	// RFC 9930 §6.2.5: when a method has no EMSK, implementations keep the previous
	// IMCK_EMSK rather than derive one from a secret that does not exist.
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(64), EMSK: secret(64)[32:]}))
	_, emskCMKBefore := chain.CompoundMACKeys()
	require.NotNil(t, emskCMKBefore)

	before := len(r.calls)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))
	assert.Equal(t, []string{labelIMSK, labelInnerMethods}, r.labels()[before:],
		"a method without an EMSK touches only the MSK path")

	_, emskCMKAfter := chain.CompoundMACKeys()
	assert.Equal(t, emskCMKBefore, emskCMKAfter, "IMCK_EMSK[j] = IMCK_EMSK[j-1]")
}

func TestAdvanceErrorNamesThePath(t *testing.T) {
	// Both paths use the same two labels, so a failure has to say which chain broke.
	for _, tc := range []struct {
		name     string
		failCall int
		want     string
	}{
		{"imsk from msk", 1, "MSK path"},
		{"imck msk", 2, "MSK path"},
		{"imsk from emsk", 3, "EMSK path"},
		{"imck emsk", 4, "EMSK path"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r recorder
			r.failCall = tc.failCall
			chain, err := NewChain(secret(sIMCKLen))
			require.NoError(t, err)
			err = chain.Advance(&r, InnerMethodKeys{MSK: secret(64), EMSK: secret(64)[32:]})
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrKeyLength)
			assert.Contains(t, err.Error(), "("+tc.want+")")
		})
	}
}

func TestNewChainChecksAndCopiesTheSeed(t *testing.T) {
	for _, length := range []int{0, sIMCKLen - 1, sIMCKLen + 1} {
		_, err := NewChain(make([]byte, length))
		assert.ErrorIs(t, err, ErrSeedLength, "a %d-octet seed is not RFC 9427 §2.2's 40", length)
	}

	seed := secret(sIMCKLen)
	want := bytes.Clone(seed)
	chain, err := NewChain(seed)
	require.NoError(t, err)
	// The caller is free to reuse the buffer it handed over.
	clear(seed)

	var r recorder
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))
	assert.Equal(t, want, r.request(t, 2).context[:sIMCKLen], "the chain keeps its own copy of the seed")
}

func TestCompoundMACKeysAreCopies(t *testing.T) {
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32), EMSK: secret(64)[32:]}))

	mskCMK, _ := chain.CompoundMACKeys()
	require.Len(t, mskCMK, cmkLen)
	clear(mskCMK)
	after, _ := chain.CompoundMACKeys()
	assert.NotEqual(t, mskCMK, after, "handing out the chain's own buffer would let a caller corrupt CMK[n]")
}

func TestSessionKeysBeforeAnyInnerMethod(t *testing.T) {
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	_, _, err = chain.SessionKeys(&r, false)
	assert.ErrorIs(t, err, ErrNoInnerMethod, "S-IMCK[0] alone is not a session key")
	assert.Empty(t, r.calls)
}

func TestSessionKeysWithoutEMSKChain(t *testing.T) {
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))
	_, _, err = chain.SessionKeys(&r, true)
	assert.ErrorIs(t, err, ErrNoEMSK,
		"no EMSK means no EMSK Compound MAC could ever have been verified (RFC 9930 §6.2.2)")
}

func TestSessionKeysSelectsThePathThePeerBound(t *testing.T) {
	var r recorder
	chain, err := NewChain(secret(sIMCKLen))
	require.NoError(t, err)
	// One method with both keys, so the two derivations differ from here on.
	require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(64), EMSK: secret(64)[32:]}))

	msk, emsk, err := chain.SessionKeys(&r, false)
	require.NoError(t, err)
	require.Len(t, r.calls, 6)
	assert.Equal(t, labelMSK, r.request(t, 5).label)
	assert.Equal(t, labelEMSK, r.request(t, 6).label)
	for _, n := range []int{5, 6} {
		assert.Equal(t, resultKeyLen, r.request(t, n).length,
			"each key is its own 64-octet export, not a slice of a larger one (RFC 9427 §2.1)")
	}
	assert.Equal(t, expand(labelMSK, chainSIMCK(chain), resultKeyLen), msk)
	assert.Equal(t, expand(labelEMSK, chainSIMCK(chain), resultKeyLen), emsk)

	_, _, err = chain.SessionKeys(&r, true)
	require.NoError(t, err)
	mskPathSeed := r.request(t, 5).context
	assert.NotEqual(t, mskPathSeed, r.request(t, 7).context,
		"useEMSK switches to S-IMCK_EMSK[n] (RFC 9930 §6.2.2)")
}

func TestSessionKeysErrorNamesTheKey(t *testing.T) {
	// One MSK-only method makes two export calls, so the outer MSK is the third and the
	// outer EMSK the fourth.
	for _, tc := range []struct {
		name     string
		failCall int
		want     string
	}{
		{"outer msk", 3, "outer MSK"},
		{"outer emsk", 4, "outer EMSK"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var r recorder
			chain, err := NewChain(secret(sIMCKLen))
			require.NoError(t, err)
			require.NoError(t, chain.Advance(&r, InnerMethodKeys{MSK: secret(32)}))
			r.failCall = tc.failCall
			_, _, err = chain.SessionKeys(&r, false)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "("+tc.want+")")
		})
	}
}

// chainSIMCK reads the MSK path's running S-IMCK, which the tests otherwise only see
// through the exporter requests.
func chainSIMCK(c *Chain) []byte { return bytes.Clone(c.msk.sIMCK) }

// readSpec loads a vendored specification for the citation checks.
func readSpec(t *testing.T, path string) string {
	t.Helper()
	//nolint:gosec // G304: the path is a repository-relative constant, not user input.
	raw, err := os.ReadFile(filepath.FromSlash(path))
	require.NoError(t, err, "RFC 9427 must stay in docs/rfcs for these labels to be citable")
	return string(raw)
}

// collapseSpace squeezes whitespace so a quotation survives the RFC's line wrapping.
func collapseSpace(in string) string {
	return strings.Join(strings.Fields(in), " ")
}
