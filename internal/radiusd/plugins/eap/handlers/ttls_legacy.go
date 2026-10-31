package handlers

import (
	"crypto/md5"
	"crypto/subtle"
	"fmt"

	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap"
	"github.com/bjo163/mwx-isp/internal/radiusd/plugins/eap/tlsengine"
	"layeh.com/radius/rfc2759"
)

// EAP-TTLS inner CHAP (RFC 5281 §11.2.2, algorithm RFC 1994) and MS-CHAP
// (RFC 5281 §11.2.3, algorithm RFC 2433) constants.
const (
	// ttlsAVPCodeCHAPPassword is the CHAP-Password AVP code (RADIUS attribute 3,
	// RFC 2865 §5.3): one Ident octet followed by the 16-octet CHAP response.
	ttlsAVPCodeCHAPPassword = 3
	// ttlsAVPCodeCHAPChallenge is the CHAP-Challenge AVP code (RADIUS attribute
	// 60, RFC 2865 §5.40): the 16-octet challenge both peers derive.
	ttlsAVPCodeCHAPChallenge = 60

	// ttlsMSCHAPResponseCode is the MS-CHAP-Response AVP code (RFC 2548 §2.1):
	// Ident, Flags, an LM-Response and an NT-Response.
	ttlsMSCHAPResponseCode = 1

	// ttlsCHAPChallengeLen is the challenge material CHAP consumes: a 16-octet
	// CHAP-Challenge followed by a 1-octet CHAP Identifier (RFC 5281 §11.2.2).
	ttlsCHAPChallengeLen = 17
	// ttlsMSCHAPChallengeLen is the challenge material MS-CHAP consumes: an
	// 8-octet MS-CHAP-Challenge followed by a 1-octet Ident (RFC 5281 §11.2.3).
	ttlsMSCHAPChallengeLen = 9
	// ttlsMSCHAPResponseLen is the fixed MS-CHAP-Response value length:
	// Ident(1) + Flags(1) + LM-Response(24) + NT-Response(24) (RFC 2548 §2.3.1).
	ttlsMSCHAPResponseLen = 50
	// ttlsMSCHAPFlagUseNTResponse is bit 0 of the MS-CHAP-Response Flags field:
	// when set, the NT-Response field is valid (RFC 2433 §2.2).
	ttlsMSCHAPFlagUseNTResponse = 0x01
)

// chapResponse computes the 16-octet CHAP response defined by RFC 1994 §4.1:
// MD5(Identifier || secret || challenge). The secret is the cleartext password,
// which the TLS tunnel protects on the wire.
func chapResponse(identifier uint8, secret, challenge []byte) []byte {
	sum := md5.New() //nolint:gosec // G401: CHAP is defined over MD5 (RFC 1994); the method is legacy and opt-in.
	sum.Write([]byte{identifier})
	sum.Write(secret)
	sum.Write(challenge)
	return sum.Sum(nil)
}

// handleInnerCHAP validates an inner CHAP exchange (RFC 5281 §11.2.2). Both
// peers derive the 17-octet challenge material from the TLS session under the
// "ttls challenge" label; the handler rejects unless the peer's CHAP-Challenge
// AVP and CHAP Identifier equal the derived values, then verifies the response
// against the cleartext password. CHAP is single-round: on success the MS-MPPE
// keys are derived from the TLS session and the handler grants.
//
// An external directory verifier cannot be used: a bind proves only that the
// directory accepts the password, and CHAP never reveals it, so CHAP stays
// unavailable for directory-backed users instead of silently succeeding.
func (h *TTLSHandler) handleInnerCHAP(ctx *eap.EAPContext, engine *tlsengine.Engine, avps []ttlsAVP) ([]byte, bool, error) {
	chal, err := engine.ExportKey(ttlsChallengeLabel, nil, ttlsCHAPChallengeLen)
	if err != nil {
		return nil, false, fmt.Errorf("failed to derive EAP-TTLS implicit challenge: %w", err)
	}
	challenge := chal[:16]
	expectedIdent := chal[16]

	presented, ok := findTTLSAVP(avps, ttlsAVPCodeCHAPChallenge)
	if !ok {
		return nil, false, fmt.Errorf("%w: missing CHAP-Challenge AVP", eap.ErrTTLSInnerProtocol)
	}
	if subtle.ConstantTimeCompare(presented, challenge) != 1 {
		return nil, false, fmt.Errorf("%w: CHAP-Challenge AVP does not match the derived challenge", eap.ErrTTLSInnerProtocol)
	}

	response, ok := findTTLSAVP(avps, ttlsAVPCodeCHAPPassword)
	if !ok {
		return nil, false, fmt.Errorf("%w: missing CHAP-Password AVP", eap.ErrTTLSInnerProtocol)
	}
	if len(response) != 1+md5.Size {
		return nil, false, fmt.Errorf("%w: CHAP-Password is %d octets, expected %d",
			eap.ErrTTLSInnerProtocol, len(response), 1+md5.Size)
	}
	if response[0] != expectedIdent {
		return nil, false, fmt.Errorf("%w: CHAP-Password Identifier %d does not match the derived Identifier %d",
			eap.ErrTTLSInnerProtocol, response[0], expectedIdent)
	}

	if ctx.Verifier != nil && ctx.Verifier.Active() {
		return nil, false, fmt.Errorf("%w: inner CHAP needs the cleartext password, which a directory bind does not provide", eap.ErrTTLSInnerNotImplemented)
	}

	password, err := ctx.PwdProvider.GetPassword(ctx.User, ctx.IsMacAuth)
	if err != nil {
		return nil, false, err
	}

	if subtle.ConstantTimeCompare(chapResponse(response[0], []byte(password), challenge), response[1:]) != 1 {
		return nil, false, eap.ErrPasswordMismatch
	}

	if err := h.deriveMPPEKeys(ctx.Response, engine); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

// handleInnerMSCHAP validates an inner MS-CHAP exchange (RFC 5281 §11.2.3,
// algorithm RFC 2433). Both peers derive the 9-octet challenge material from
// the TLS session under the "ttls challenge" label; the handler rejects unless
// the peer's MS-CHAP-Challenge AVP and Ident equal the derived values, then
// verifies the NT-Response with the NT hash of the cleartext password. It is
// single-round like CHAP.
//
// Only the NT-Response is accepted: this server has no LM hash, so a client
// that sends an LM-only response (Flags bit 0 clear) is rejected explicitly
// rather than being granted or failed without explanation. Directory-backed
// users are refused for the same reason as inner CHAP — a bind cannot produce
// the NT hash.
func (h *TTLSHandler) handleInnerMSCHAP(ctx *eap.EAPContext, engine *tlsengine.Engine, avps []ttlsAVP) ([]byte, bool, error) {
	chal, err := engine.ExportKey(ttlsChallengeLabel, nil, ttlsMSCHAPChallengeLen)
	if err != nil {
		return nil, false, fmt.Errorf("failed to derive EAP-TTLS implicit challenge: %w", err)
	}
	challenge := chal[:8]
	expectedIdent := chal[8]

	presented, ok := findTTLSVendorAVP(avps, ttlsVendorMicrosoft, ttlsMSCHAPChallengeCode)
	if !ok {
		return nil, false, fmt.Errorf("%w: missing MS-CHAP-Challenge AVP", eap.ErrTTLSInnerProtocol)
	}
	if subtle.ConstantTimeCompare(presented, challenge) != 1 {
		return nil, false, fmt.Errorf("%w: MS-CHAP-Challenge AVP does not match the derived challenge", eap.ErrTTLSInnerProtocol)
	}

	response, ok := findTTLSVendorAVP(avps, ttlsVendorMicrosoft, ttlsMSCHAPResponseCode)
	if !ok {
		return nil, false, fmt.Errorf("%w: missing MS-CHAP-Response AVP", eap.ErrTTLSInnerProtocol)
	}
	if len(response) != ttlsMSCHAPResponseLen {
		return nil, false, fmt.Errorf("%w: MS-CHAP-Response is %d octets, expected %d",
			eap.ErrTTLSInnerProtocol, len(response), ttlsMSCHAPResponseLen)
	}
	ident, flags := response[0], response[1]
	ntResponse := response[26:50]
	if ident != expectedIdent {
		return nil, false, fmt.Errorf("%w: MS-CHAP-Response Ident %d does not match the derived Ident %d",
			eap.ErrTTLSInnerProtocol, ident, expectedIdent)
	}
	if flags&ttlsMSCHAPFlagUseNTResponse == 0 || allZero(ntResponse) {
		return nil, false, fmt.Errorf("%w: inner MS-CHAP LM-only response is not supported", eap.ErrTTLSInnerNotImplemented)
	}

	if ctx.Verifier != nil && ctx.Verifier.Active() {
		return nil, false, fmt.Errorf("%w: inner MS-CHAP needs the NT hash of the password, which a directory bind does not provide", eap.ErrTTLSInnerNotImplemented)
	}

	password, err := ctx.PwdProvider.GetPassword(ctx.User, ctx.IsMacAuth)
	if err != nil {
		return nil, false, err
	}

	ucs2Password, err := rfc2759.ToUTF16([]byte(password))
	if err != nil {
		return nil, false, fmt.Errorf("failed to encode the password for MS-CHAP: %w", err)
	}
	expected := rfc2759.ChallengeResponse(challenge, rfc2759.NTPasswordHash(ucs2Password))
	if subtle.ConstantTimeCompare(expected, ntResponse) != 1 {
		return nil, false, eap.ErrPasswordMismatch
	}

	if err := h.deriveMPPEKeys(ctx.Response, engine); err != nil {
		return nil, false, err
	}
	return nil, true, nil
}

func allZero(data []byte) bool {
	for _, octet := range data {
		if octet != 0 {
			return false
		}
	}
	return true
}
