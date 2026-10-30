package teapinner

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teapkeys"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/teaptlv"
)

// rfc9930Path is the specification this package's bounds come from, read from the copy
// the repository carries rather than from memory.
const rfc9930Path = "../../../../../docs/rfcs/rfc9930-teap-version-1.txt"

// bothFlags is what a peer announces when it bound both Compound MACs, the usual case
// for a method that derived an EMSK.
const bothFlags = teaptlv.FlagEMSKPresent | teaptlv.FlagMSKPresent

// machineThenUser is the sequence the roadmap calls certificate plus password: the
// EMSK-deriving method runs first, so it detects an on-path attacker before any
// password is sent (§6.2.1's example is exactly this order).
func machineThenUser(t *testing.T) Plan {
	t.Helper()
	plan, err := NewPlan(DefaultPolicy(), EAPTLS, BasicPassword)
	require.NoError(t, err)
	return plan
}

// run drives one method to a recorded outcome.
func run(t *testing.T, s *Session, server, peer teaptlv.Status, flags uint8) Method {
	t.Helper()
	m, ok := s.Next()
	require.True(t, ok, "the plan must still have a method to run")
	require.NoError(t, s.Record(server, peer, flags))
	return m
}

func TestNewPlanRejectsWhatTheSpecificationForbids(t *testing.T) {
	for _, tc := range []struct {
		name   string
		method Method
		want   error
	}{
		{
			name:   "a tunnel inside the tunnel",
			method: Method{Name: "PEAP", DerivesMSK: true, DerivesEMSK: true, CryptoBinds: true, Tunneled: true},
			want:   ErrTunneledMethod,
		},
		{
			name:   "a method that resumes its own session",
			method: Method{Name: "EAP-TLS (resumed)", DerivesEMSK: true, CryptoBinds: true, Resumable: true},
			want:   ErrInnerResumption,
		},
		{
			// §3.6.5 names MD5-Challenge, OTP and GTC as the methods that "are therefore
			// not useful as Phase 2 methods within TEAP".
			name:   "an EAP method that binds no keys",
			method: Method{Name: "EAP-MD5", DerivesMSK: true},
			want:   ErrNoCryptoBinding,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPlan(Policy{}, tc.method)
			assert.ErrorIs(t, err, tc.want)
		})
	}

	// A password exchange is not an inner EAP method, so §3.6.5's crypto-binding
	// requirement does not reject it on its own; §6.2.1's conjunction rule does, and only
	// when the policy enforces it.
	_, err := NewPlan(Policy{}, BasicPassword)
	assert.NoError(t, err, "without the §6.2.1 flag a bare password plan is admitted")
	_, err = NewPlan(Policy{ForbidUnboundPassword: true}, BasicPassword)
	assert.ErrorIs(t, err, ErrUnboundPassword)
}

func TestNewPlanEnforcesTheOrderingChecksThePolicySelects(t *testing.T) {
	// §6.2.3 lists verifying that the first method derives an EMSK as one of the checks
	// an implementation can enforce, so it is a choice and not an absolute.
	_, err := NewPlan(Policy{RequireEMSKFirst: true}, EAPFASTMSCHAPv2, EAPTLS)
	assert.ErrorIs(t, err, ErrEMSKFirst)

	// The same plan is legal with the flag off, which is the documented deviation for a
	// peer that cannot do otherwise.
	_, err = NewPlan(Policy{}, EAPFASTMSCHAPv2, EAPTLS)
	require.NoError(t, err)

	// Only the first method is examined: a later one without an EMSK is the password
	// exchange in the recommended sequence.
	plan, err := NewPlan(DefaultPolicy(), EAPTLS, EAPFASTMSCHAPv2, BasicPassword)
	require.NoError(t, err)
	assert.Len(t, plan.Methods(), 3)

	// §3.6.6 allows a Phase 2 with no Inner Method at all, so an empty plan is a plan.
	empty, err := NewPlan(DefaultPolicy())
	require.NoError(t, err)
	assert.Empty(t, empty.Methods())
	assert.Equal(t, defaultPasswordRounds, empty.PasswordRounds())

	// The round-trip limit §3.6.3 makes the implementation set is the plan's, so a
	// configured value replaces the default and a non-positive one asks for it.
	configured, err := NewPlan(Policy{MaxPasswordRounds: 5})
	require.NoError(t, err)
	assert.Equal(t, 5, configured.PasswordRounds())
	zeroed, err := NewPlan(Policy{MaxPasswordRounds: -1})
	require.NoError(t, err)
	assert.Equal(t, defaultPasswordRounds, zeroed.PasswordRounds())
}

func TestMethodPrintsAsItself(t *testing.T) {
	// The name is the only thing a log line or an error can say about a method, so a
	// Method has to format as it rather than as the six capability fields it carries.
	for _, m := range []Method{EAPTLS, EAPFASTMSCHAPv2, BasicPassword} {
		assert.Equal(t, m.Name, m.String())
		assert.Equal(t, m.Name, fmt.Sprint(m))
	}
}

func TestMethodsCopyIsNotThePlan(t *testing.T) {
	plan := machineThenUser(t)
	methods := plan.Methods()
	require.Len(t, methods, 2)
	methods[0].Name = "tampered"
	assert.Equal(t, "EAP-TLS", plan.Methods()[0].Name, "a caller must not be able to rewrite the plan it was handed")
}

func TestSessionRunsTheSequenceSerially(t *testing.T) {
	s := NewSession(machineThenUser(t))

	first, ok := s.Next()
	require.True(t, ok)
	assert.Equal(t, EAPTLS.Name, first.Name)

	// §3.6.2 permits no second method in parallel, so the handshake cannot ask for one
	// before the running method has an outcome.
	_, ok = s.Next()
	assert.False(t, ok, "the outstanding method must be recorded before the next one starts")
	running, ok := s.Running()
	require.True(t, ok)
	assert.Equal(t, EAPTLS.Name, running.Name)

	require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags))
	assert.False(t, s.Complete(), "the password exchange has not run yet")

	second := run(t, s, teaptlv.StatusSuccess, teaptlv.StatusSuccess, teaptlv.FlagMSKPresent)
	assert.Equal(t, BasicPassword.Name, second.Name)
	assert.True(t, s.Complete())
	assert.False(t, s.pending)

	_, ok = s.Next()
	assert.False(t, ok, "a complete plan has nothing left to run")
	_, ok = s.Running()
	assert.False(t, ok)

	authenticated := s.Authenticated()
	require.Len(t, authenticated, 2)
	assert.Equal(t, []string{EAPTLS.Name, BasicPassword.Name},
		[]string{authenticated[0].Name, authenticated[1].Name})
}

func TestSessionTreatsEitherFailureAsTheMethodFailing(t *testing.T) {
	// §3.6.2: "If both peer and server indicate success, then the authentication is
	// considered successful. If either indicates failure, then the authentication is
	// considered failed."
	plan, err := NewPlan(Policy{}, EAPTLS, EAPFASTMSCHAPv2)
	require.NoError(t, err)

	for _, tc := range []struct {
		name   string
		server teaptlv.Status
		peer   teaptlv.Status
		ok     bool
	}{
		{"both succeed", teaptlv.StatusSuccess, teaptlv.StatusSuccess, true},
		{"the peer rejects a success the server saw", teaptlv.StatusSuccess, teaptlv.StatusFailure, false},
		{"the server rejects a success the peer saw", teaptlv.StatusFailure, teaptlv.StatusSuccess, false},
		{"both fail", teaptlv.StatusFailure, teaptlv.StatusFailure, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSession(plan)
			_, ok := s.Next()
			require.True(t, ok)
			// A failed method sends no Crypto-Binding TLV, so Flags 0 is the only legal
			// value here whatever the statuses say.
			flags := uint8(0)
			if tc.ok {
				flags = bothFlags
			}
			require.NoError(t, s.Record(tc.server, tc.peer, flags))
			if tc.ok {
				assert.Len(t, s.Authenticated(), 1)
			} else {
				assert.Empty(t, s.Authenticated(), "a method one side failed is not authenticated")
			}

			// The sequence continues: the failure of one Inner Method says nothing about
			// the next one.
			second, ok := s.Next()
			require.True(t, ok)
			assert.Equal(t, EAPFASTMSCHAPv2.Name, second.Name)
		})
	}
}

func TestRecordDemandsTheBindingASuccessOwes(t *testing.T) {
	emskMethod, err := NewPlan(Policy{}, EAPTLS)
	require.NoError(t, err)
	mskMethod, err := NewPlan(Policy{}, EAPFASTMSCHAPv2)
	require.NoError(t, err)
	password, err := NewPlan(Policy{}, EAPTLS, BasicPassword)
	require.NoError(t, err)

	t.Run("a success without any Crypto-Binding TLV", func(t *testing.T) {
		s := NewSession(emskMethod)
		_, ok := s.Next()
		require.True(t, ok)
		assert.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, 0), ErrNoBinding)
	})

	t.Run("an EMSK-deriving method bound only by the MSK MAC", func(t *testing.T) {
		// §6.2.3's second check: "If an EMSK has been derived and no EMSK Compound MAC is
		// seen, a fatal error can be returned."
		s := NewSession(emskMethod)
		_, ok := s.Next()
		require.True(t, ok)
		err := s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, teaptlv.FlagMSKPresent)
		assert.ErrorIs(t, err, ErrNoEMSKBinding)
		assert.Empty(t, s.Authenticated(), "a method the binding does not cover is not authenticated")
	})

	t.Run("an MSK-only method needs no EMSK MAC", func(t *testing.T) {
		// EAP-FAST-MSCHAPv2 derives no EMSK, so §6.2.3's rule cannot ask for a MAC over a
		// key that does not exist.
		s := NewSession(mskMethod)
		_, ok := s.Next()
		require.True(t, ok)
		require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, teaptlv.FlagMSKPresent))
		assert.Len(t, s.Authenticated(), 1)
	})

	t.Run("the password exchange still ends in a binding", func(t *testing.T) {
		s := NewSession(password)
		_, ok := s.Next()
		require.True(t, ok)
		require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags))
		_, ok = s.Next()
		require.True(t, ok)
		// §3.6.3: the Intermediate-Result TLV that finishes the password round "MUST be
		// accompanied by a Crypto-Binding TLV" when it says success.
		assert.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, 0), ErrNoBinding)
		require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, teaptlv.FlagMSKPresent))
	})

	t.Run("a failure that carries a binding", func(t *testing.T) {
		// §6.2.4: "If the Inner Method has failed, the server does not send a
		// Crypto-Binding TLV." A peer that answers a failure with one is not in the state
		// machine this session is in.
		s := NewSession(emskMethod)
		_, ok := s.Next()
		require.True(t, ok)
		assert.ErrorIs(t, s.Record(teaptlv.StatusFailure, teaptlv.StatusFailure, teaptlv.FlagMSKPresent),
			ErrUnexpectedBinding)
	})

	t.Run("nothing is running", func(t *testing.T) {
		s := NewSession(emskMethod)
		err := s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags)
		assert.ErrorIs(t, err, ErrNotRunning)
		_, ok := s.Next()
		require.True(t, ok)
		require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags))
		assert.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags), ErrNotRunning,
			"the same outcome cannot be recorded twice")
	})
}

func TestKeysForAppliesThePasswordRule(t *testing.T) {
	// §6.2.1 fixes what a password round contributes: no MSK, no EMSK, and an all-zero
	// IMSK as a consequence - which is teapkeys' reading of empty keys, so the two
	// packages compose without this one deriving anything.
	keys, err := KeysFor(BasicPassword, []byte("not a key"), []byte("nor this"))
	require.NoError(t, err)
	assert.Empty(t, keys.MSK)
	assert.Empty(t, keys.EMSK)

	seed := make([]byte, 40)
	chain, err := teapkeys.NewChain(seed)
	require.NoError(t, err)
	require.NoError(t, chain.Advance(exporter{}, mustKeys(t, EAPTLS, make([]byte, 64), make([]byte, 64))))
	beforeMSK, beforeEMSK := chain.CompoundMACKeys()
	require.NoError(t, chain.Advance(exporter{}, keys))
	afterMSK, afterEMSK := chain.CompoundMACKeys()

	assert.NotEqual(t, beforeMSK, afterMSK, "the password round still advances the MSK derivation")
	assert.Equal(t, beforeEMSK, afterEMSK, "with no EMSK the EMSK derivation carries forward (§6.2.5)")
}

func TestKeysForRefusesKeysAMethodDidNotProduce(t *testing.T) {
	// The configuration says EAP-TLS derives both keys; a handshake that reaches the
	// chain with one of them missing has a bug, and folding zeros in would bind a session
	// that never had those keys.
	_, err := KeysFor(EAPTLS, nil, make([]byte, 64))
	assert.ErrorIs(t, err, ErrMissingMethodKeys)
	_, err = KeysFor(EAPTLS, make([]byte, 64), nil)
	assert.ErrorIs(t, err, ErrMissingMethodKeys)

	_, err = KeysFor(Method{Name: "EAP-GTC"}, nil, nil)
	assert.ErrorIs(t, err, ErrNoCryptoBinding, "a non-binding method never belongs in the chain")

	keys, err := KeysFor(EAPFASTMSCHAPv2, make([]byte, 64), nil)
	require.NoError(t, err)
	assert.Len(t, keys.MSK, 64)
	assert.Empty(t, keys.EMSK, "a method that derives no EMSK contributes none")
}

func TestSwapMSCHAPv2MSKSwapsTheTwoHalves(t *testing.T) {
	// §3.6.4: "the first and the second 16 octets of the EAP-MSCHAPv2 Master Session Key
	// (MSK) are swapped when it is used as the Inner Method Session Keys (IMSKs) for
	// TEAP."
	msk := make([]byte, 32)
	for i := range msk {
		msk[i] = byte(i)
	}
	got, err := SwapMSCHAPv2MSK(msk)
	require.NoError(t, err)
	assert.Equal(t, append([]byte(nil), msk[16:]...), got[:16], "the second half comes first")
	assert.Equal(t, append([]byte(nil), msk[:16]...), got[16:], "the first half comes second")

	again, err := SwapMSCHAPv2MSK(got)
	require.NoError(t, err)
	assert.Equal(t, msk, again, "the swap is its own inverse")
	assert.NotEqual(t, msk, got, "and it does change the key")

	// Anything that is not two halves of the width the clause names cannot be swapped,
	// including a truncated MSK an implementation might otherwise have padded silently.
	for _, size := range []int{0, 15, 16, 17, 31, 33, 64} {
		_, err := SwapMSCHAPv2MSK(make([]byte, size))
		assert.ErrorIs(t, err, ErrMSKLength, "%d octets is not two 16-octet halves", size)
	}
}

func TestPasswordRoundsDemandTheFirstPrompt(t *testing.T) {
	r := NewPasswordRounds(0)
	_, err := r.Request("")
	assert.ErrorIs(t, err, ErrNoPrompt, "the first request has to tell the user what to type")

	tlv, err := r.Request("Password: ")
	require.NoError(t, err)
	prompt, err := tlv.Prompt()
	require.NoError(t, err)
	assert.Equal(t, "Password: ", prompt)
	assert.Equal(t, 1, r.Rounds())
}

func TestPasswordRoundsHoldTheUsername(t *testing.T) {
	// §3.6.3 makes the peer copy the first username into every later response and the
	// server reject an identity that changes. The limit on round trips is what makes the
	// check worth having: without it, a peer could keep re-asking until it guessed.
	r := NewPasswordRounds(2)
	_, err := r.Request("Password: ")
	require.NoError(t, err)
	_, err = r.Respond([]byte("machine.example.com"), []byte("hunter2"))
	require.NoError(t, err)
	assert.Equal(t, []byte("machine.example.com"), r.Username())

	// A subsequent round is asked for a password only, so an empty prompt is legal here.
	_, err = r.Request("")
	require.NoError(t, err)

	want, err := teaptlv.NewBasicPasswordAuthResp([]byte("machine.example.com"), []byte("totp-1234"))
	require.NoError(t, err)
	second, err := r.Respond([]byte("machine.example.com"), []byte("totp-1234"))
	require.NoError(t, err)
	assert.Equal(t, want, second, "the response is the codec's own encoding of the echoed identity")

	_, err = r.Respond([]byte("someone.else"), []byte("hunter2"))
	assert.ErrorIs(t, err, ErrIdentityChanged, "the identity may not change mid-authentication")
	assert.Equal(t, []byte("machine.example.com"), r.Username(),
		"a rejected round must not take the session's identity with it")

	// teaptlv owns the zero-length rules of §4.2.15, so they fail here too.
	_, err = r.Respond(nil, []byte("hunter2"))
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV)
	_, err = r.Respond([]byte("machine.example.com"), nil)
	assert.ErrorIs(t, err, teaptlv.ErrInvalidTLV)
}

func TestPasswordRoundsStopAtTheLimit(t *testing.T) {
	// §3.6.3: "Implementations MUST limit the number of round trips for password
	// authentication." The limit is the configuration's, so both the default and an
	// explicit one have to hold.
	for _, limit := range []int{1, 2, 3} {
		r := NewPasswordRounds(limit)
		for i := range limit {
			_, err := r.Request("Password: ")
			require.NoError(t, err, "round %d of %d must be allowed", i+1, limit)
			_, err = r.Respond([]byte("user"), []byte("secret"))
			require.NoError(t, err)
		}
		_, err := r.Request("Password: ")
		assert.ErrorIs(t, err, ErrTooManyRounds, "%d rounds configured", limit)
		assert.Equal(t, limit, r.Rounds())
	}

	// A limit of zero or less is the caller asking for the default, not for no rounds.
	r := NewPasswordRounds(0)
	_, err := r.Request("Password: ")
	require.NoError(t, err)
	assert.Equal(t, defaultPasswordRounds, NewPasswordRounds(-1).limit)
}

// exporter stands in for the TLS 1.3 exporter so the key-rule test can drive a real
// teapkeys.Chain without a tunnel.
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

func mustKeys(t *testing.T, m Method, msk, emsk []byte) teapkeys.InnerMethodKeys {
	t.Helper()
	keys, err := KeysFor(m, msk, emsk)
	require.NoError(t, err)
	return keys
}

// collapseSpace joins the line-wrapped prose of an RFC so a quoted sentence matches
// however the text file breaks its lines.
func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestTheBoundsMatchTheVendoredClauses is the anti-fabrication gate: every rule this
// package enforces is quoted from the text the repository carries, so a clause that
// moves or disappears stops the build here rather than turning into a policy nobody
// can trace.
func TestTheBoundsMatchTheVendoredClauses(t *testing.T) {
	raw, err := os.ReadFile(rfc9930Path) //nolint:gosec // G304: the path is the constant above, under docs/rfcs.
	require.NoError(t, err, "read %s", rfc9930Path)
	spec := collapseSpace(string(raw))

	for _, clause := range []string{
		// §3.6.2 - serial execution, and the per-method exchange.
		"Inner Methods are executed serially in a sequence.",
		"This version of TEAP does not support initiating multiple Inner Methods simultaneously in parallel.",
		"When the result indicates success, it MUST be accompanied by a Crypto-Binding TLV.",
		"If either indicates failure, then the authentication is considered failed.",
		"the server may attempt to authenticate the peer with a different method (EAP or password).",
		// §3.6.3 - the password exchange.
		"The first Basic-Password-Auth-Req TLV received in a session MUST include a prompt",
		"Implementations MUST limit the number of round trips for password authentication.",
		"It is reasonable to use one or two round trips.",
		"Servers MUST track the username across multiple password rounds and reject authentication if the identity changes",
		// §3.6.4 - the MSCHAPv2 IMSK.
		"the first and the second 16 octets of the EAP-MSCHAPv2 Master Session Key (MSK) are swapped",
		// §3.6.5 - the limitations on inner methods. Each phrase stops where the
		// vendored text breaks a hyphenated word across lines, since that is what the
		// whitespace collapse above cannot restore.
		"Tunneled EAP methods such as PEAP [PEAP], EAP-TTLS [RFC5281], and EAP-FAST [RFC4851] MUST NOT be used for inner EAP authentication.",
		"Implementations MUST NOT permit resumption for the inner EAP methods such as EAP-TLS.",
		"TEAP implementations MUST NOT permit the use of inner EAP methods",
		"binding of the Inner Method to the TLS session.",
		"Implementations SHOULD limit the permitted inner EAP methods to a small set such as EAP-TLS and the EAP-FAST-MSCHAPv2 variant of",
		// §3.6.6 - a Phase 2 that runs no Inner Method at all.
		"A TEAP server may initiate the Crypto-Binding TLV and Result TLV exchange without initiating any Inner Methods in TEAP Phase 2.",
		// §6.2.1 and §6.2.3 - the key and ordering consequences.
		"such as when basic password authentication is used or when no Inner Method has been run, then both MSK and IMSK[j] are set to all zeroes",
		"Verifying that the first Inner Method used is one that derives EMSK.",
		"Verifying that if any Inner Method derives EMSK, the received Crypto-Binding TLV for that method contains an EMSK Compound MAC.",
		// §6.2.4 - no binding for a failed method.
		"If the Inner Method has failed, the server does not send a Crypto-Binding TLV.",
	} {
		assert.Contains(t, spec, collapseSpace(clause), "RFC 9930 no longer states %q", clause)
	}

	// The two numbers the clauses fix.
	assert.Equal(t, 2, defaultPasswordRounds, "§3.6.3 calls one or two round trips reasonable")
	assert.Equal(t, 32, 2*mskHalfLen, "§3.6.4 swaps two 16-octet halves of a 32-octet MSK")
}

// TestSentinelsAreAllReachable keeps every exported error honest: each one is the
// verdict of a rule the specification states, so a constant nobody returns any more is
// a rule that stopped being enforced.
func TestSentinelsAreAllReachable(t *testing.T) {
	_, err := NewPlan(Policy{}, Method{Name: "PEAP", CryptoBinds: true, Tunneled: true})
	require.ErrorIs(t, err, ErrTunneledMethod)
	_, err = NewPlan(Policy{}, Method{Name: "EAP-TLS", CryptoBinds: true, Resumable: true})
	require.ErrorIs(t, err, ErrInnerResumption)
	_, err = NewPlan(Policy{}, Method{Name: "EAP-MD5"})
	require.ErrorIs(t, err, ErrNoCryptoBinding)
	_, err = NewPlan(Policy{ForbidUnboundPassword: true}, BasicPassword)
	require.ErrorIs(t, err, ErrUnboundPassword)
	_, err = NewPlan(Policy{RequireEMSKFirst: true}, EAPFASTMSCHAPv2)
	require.ErrorIs(t, err, ErrEMSKFirst)

	s := NewSession(mustPlan(t, Policy{}, EAPTLS))
	_, ok := s.Next()
	require.True(t, ok)
	require.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, 0), ErrNoBinding)
	require.NoError(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags))
	require.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, bothFlags), ErrNotRunning)

	s = NewSession(mustPlan(t, Policy{}, EAPTLS))
	_, ok = s.Next()
	require.True(t, ok)
	require.ErrorIs(t, s.Record(teaptlv.StatusSuccess, teaptlv.StatusSuccess, teaptlv.FlagMSKPresent), ErrNoEMSKBinding)

	s = NewSession(mustPlan(t, Policy{}, EAPTLS))
	_, ok = s.Next()
	require.True(t, ok)
	require.ErrorIs(t, s.Record(teaptlv.StatusFailure, teaptlv.StatusFailure, teaptlv.FlagMSKPresent), ErrUnexpectedBinding)

	_, err = KeysFor(EAPTLS, nil, nil)
	require.ErrorIs(t, err, ErrMissingMethodKeys)
	_, err = SwapMSCHAPv2MSK(nil)
	require.ErrorIs(t, err, ErrMSKLength)

	r := NewPasswordRounds(1)
	_, err = r.Request("")
	require.ErrorIs(t, err, ErrNoPrompt)
	_, err = r.Request("Password: ")
	require.NoError(t, err)
	_, err = r.Respond([]byte("a"), []byte("b"))
	require.NoError(t, err)
	_, err = r.Request("again: ")
	require.ErrorIs(t, err, ErrTooManyRounds)

	r = NewPasswordRounds(2)
	_, err = r.Request("Password: ")
	require.NoError(t, err)
	_, err = r.Respond([]byte("a"), []byte("b"))
	require.NoError(t, err)
	_, err = r.Request("")
	require.NoError(t, err)
	_, err = r.Respond([]byte("b"), []byte("c"))
	require.ErrorIs(t, err, ErrIdentityChanged)

	for _, sentinel := range []error{
		ErrTunneledMethod, ErrInnerResumption, ErrNoCryptoBinding, ErrUnboundPassword, ErrEMSKFirst,
		ErrNoBinding, ErrNoEMSKBinding, ErrUnexpectedBinding, ErrNoPrompt, ErrIdentityChanged,
		ErrTooManyRounds, ErrNotRunning, ErrMSKLength, ErrMissingMethodKeys,
	} {
		assert.True(t, strings.HasPrefix(sentinel.Error(), "teapinner: "), "%q breaks the error prefix convention", sentinel)
	}
}

func mustPlan(t *testing.T, policy Policy, methods ...Method) Plan {
	t.Helper()
	plan, err := NewPlan(policy, methods...)
	require.NoError(t, err)
	return plan
}

// TestPlanMethodsSurviveConstruction checks that a caller cannot mutate the sequence a
// validated Plan describes, which is what makes the §3.6.5 verdicts durable.
func TestPlanMethodsSurviveConstruction(t *testing.T) {
	methods := []Method{EAPTLS, BasicPassword}
	plan := mustPlan(t, DefaultPolicy(), methods...)
	methods[0] = Method{Name: "EAP-FAST", CryptoBinds: true, Tunneled: true}
	assert.Equal(t, EAPTLS, plan.Methods()[0], "the Plan owns its copy of the sequence")

	plan2, err := NewPlan(DefaultPolicy(), plan.Methods()...)
	require.NoError(t, err)
	assert.Equal(t, plan.Methods(), plan2.Methods())
}
