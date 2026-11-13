// Package customerportal provides cryptographic primitives for passwordless
// customer self-care sessions. OTP codes and bearer tokens are never persisted
// in plaintext.
package customerportal

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

func GenerateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate customer OTP: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func GenerateOpaqueToken() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate customer portal token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func HashOTP(secret, challengeID, otp string) string {
	return hash(secret, "customer-otp-v1", challengeID+"\x00"+otp)
}

func HashToken(secret, token string) string {
	return hash(secret, "customer-session-v1", token)
}

func HashIdentifier(secret, namespace, value string) string {
	return hash(secret, namespace, strings.TrimSpace(value))
}

func VerifyOTP(secret, challengeID, otp, expectedHash string) bool {
	if len(expectedHash) != 64 || len(otp) != 6 {
		return false
	}
	actual, err := hex.DecodeString(HashOTP(secret, challengeID, otp))
	if err != nil {
		return false
	}
	expected, err := hex.DecodeString(expectedHash)
	if err != nil || len(expected) != len(actual) {
		return false
	}
	return subtle.ConstantTimeCompare(actual, expected) == 1
}

func hash(secret, namespace, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(namespace + "\x00" + value))
	return hex.EncodeToString(mac.Sum(nil))
}
