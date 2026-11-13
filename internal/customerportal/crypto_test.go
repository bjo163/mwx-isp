package customerportal

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOTPHashIsBoundToChallengeAndSecret(t *testing.T) {
	hash := HashOTP("server-secret", "challenge-a", "012345")
	require.True(t, VerifyOTP("server-secret", "challenge-a", "012345", hash))
	require.False(t, VerifyOTP("server-secret", "challenge-b", "012345", hash))
	require.False(t, VerifyOTP("wrong-secret", "challenge-a", "012345", hash))
	require.False(t, VerifyOTP("server-secret", "challenge-a", "012346", hash))
	require.False(t, VerifyOTP("server-secret", "challenge-a", "12345", hash))
}

func TestOpaqueTokenHasEnoughEntropyAndIsNotStoredAsDigest(t *testing.T) {
	token, err := GenerateOpaqueToken()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(token), 40)
	require.NotEqual(t, token, HashToken("secret", token))
	require.Regexp(t, regexp.MustCompile("^[A-Za-z0-9_-]+$"), token)
}

func TestGeneratedOTPIsSixDigits(t *testing.T) {
	for i := 0; i < 30; i++ {
		otp, err := GenerateOTP()
		require.NoError(t, err)
		require.Regexp(t, regexp.MustCompile("^[0-9]{6}$"), otp)
	}
}
