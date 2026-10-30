package eap

import "errors"

// EAP-related errors
var (
	ErrInvalidEAPMessage    = errors.New("invalid EAP message")
	ErrStateNotFound        = errors.New("EAP state not found")
	ErrPasswordMismatch     = errors.New("password mismatch")
	ErrUnsupportedEAPType   = errors.New("unsupported EAP type")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrOTPNotConfigured     = errors.New("EAP-OTP validation is not configured")
	// ErrTLSHandshakeNotImplemented is returned by the EAP-TLS handler skeleton
	// until the TLS handshake / fragmentation logic is implemented (milestone
	// M1.2). It guarantees the skeleton never grants access.
	ErrTLSHandshakeNotImplemented = errors.New("EAP-TLS handshake is not implemented yet")
	// ErrTLSNotConfigured is returned by the EAP-TLS handler when no server
	// certificate / client CA material has been provided, so the TLS handshake
	// cannot be driven. It guarantees the handler never authenticates a client
	// without configured trust anchors.
	ErrTLSNotConfigured = errors.New("EAP-TLS is not configured (missing server certificate or client CA)")
	// ErrTLSHandshakeFailed is returned when the TLS handshake fails, including
	// when the peer certificate does not chain to a configured CA (RFC 5216
	// §2.2 / §5.3).
	ErrTLSHandshakeFailed = errors.New("EAP-TLS handshake failed")
	// ErrTLSIdentityMismatch is returned when the authenticated certificate
	// identity does not match the RADIUS User-Name (RFC 5216 §5.2 identity
	// mapping).
	ErrTLSIdentityMismatch = errors.New("EAP-TLS certificate identity does not match the requested user")
	// ErrTLSNoIdentity is returned when no usable identity can be derived from
	// the verified peer certificate.
	ErrTLSNoIdentity = errors.New("EAP-TLS peer certificate carries no usable identity")
	// ErrTLSUnexpectedFragment is returned when a peer response violates the
	// EAP-TLS fragmentation exchange (e.g. a non-ACK arrives while the server is
	// still sending fragments, RFC 5216 §2.1.5).
	ErrTLSUnexpectedFragment = errors.New("EAP-TLS unexpected fragment in handshake exchange")
	// ErrPEAPInnerNotImplemented is returned after PEAP's outer TLS tunnel has
	// been established but before the inner EAP method is implemented (milestone
	// M8.3). It guarantees M8.2 can never grant access.
	ErrPEAPInnerNotImplemented = errors.New("PEAP inner EAP method is not implemented yet")
	// ErrPEAPInnerProtocol is returned when the inner EAP-MSCHAPv2 exchange
	// carried inside the established PEAP tunnel violates the expected protocol
	// (unexpected EAP code/type/opcode for the current inner sub-state). It
	// guarantees a malformed inner exchange rejects rather than grants.
	ErrPEAPInnerProtocol = errors.New("PEAP inner EAP-MSCHAPv2 protocol violation")
	// ErrTTLSInnerNotImplemented is returned when an established EAP-TTLS tunnel
	// carries a well-formed inner method that MWX-ISP does not yet support.
	// M9.3 implements inner PAP (a User-Password AVP); a tunnel that omits the
	// User-Password AVP (e.g. an inner CHAP / MS-CHAP / MS-CHAP-V2 exchange, RFC
	// 5281 §11.2.2-§11.2.4, scheduled for M9.4) rejects with this error rather
	// than granting access.
	ErrTTLSInnerNotImplemented = errors.New("EAP-TTLS inner authentication method is not implemented yet")
	// ErrTTLSInnerProtocol is returned when the AVP sequence carried inside an
	// established EAP-TTLS tunnel is malformed: a truncated AVP header, an AVP
	// Length below its header size, or an AVP Length that overruns the decrypted
	// payload (RFC 5281 §10). It guarantees a malformed inner flight rejects
	// rather than grants.
	ErrTTLSInnerProtocol = errors.New("EAP-TTLS inner AVP protocol violation")
	// ErrTEAPVersionUnsupported is returned when a peer answers the TEAP Start
	// with a version this server does not implement. RFC 9930 §3.1 step 5 lets the
	// server terminate the conversation with an EAP Failure in that case.
	ErrTEAPVersionUnsupported = errors.New("EAP-TEAP version negotiation failed")
	// ErrTEAPProtocol is returned when a TEAP packet violates the outer framing
	// rules of RFC 9930 §4.1 beyond what the codec can reject — a Start bit in a
	// peer message, a version that changes mid-conversation, or Outer TLVs
	// outside the initial message.
	ErrTEAPProtocol = errors.New("EAP-TEAP outer-layer protocol violation")
	// ErrTEAPUnexpectedFragment is returned when a peer response arrives while the
	// server is still sending TEAP fragments and is not the expected
	// acknowledgment (RFC 9930 §4.1).
	ErrTEAPUnexpectedFragment = errors.New("EAP-TEAP unexpected fragment in handshake exchange")
	// ErrTEAPRequiresTLS13 is returned when the TEAP tunnel negotiates a TLS
	// version below 1.3. RFC 9930 §6.2 defines a TLS 1.2 key schedule over the
	// RFC 5246 TLS-PRF as well, and teapkeys implements only the RFC 9427 §2.2
	// TLS 1.3 form, so a 1.2 tunnel cannot be keyed by this server.
	ErrTEAPRequiresTLS13 = errors.New("EAP-TEAP requires a TLS 1.3 tunnel")
	// ErrTEAPPhase2NotImplemented is returned once the Phase 1 tunnel is up but
	// the handler was built without the Phase 2 options, so no inner exchange can
	// run. RFC 9930 §3.6 makes the Result TLV the only way to finish a
	// conversation, so such a handler rejects rather than granting access on the
	// tunnel alone.
	ErrTEAPPhase2NotImplemented = errors.New("EAP-TEAP Phase 2 is not enabled on this handler")
	// ErrTEAPPhase2NotConfigured is returned when a tunnel completes but the
	// handler holds no Inner Method material (an inner TLS configuration and a
	// password prompt are what Phase 2 needs). It fails closed like
	// ErrTEAPPhase2NotImplemented.
	ErrTEAPPhase2NotConfigured = errors.New("EAP-TEAP Phase 2 is not configured (missing inner TLS material)")
	// ErrTEAPInnerProtocol is returned when the TLVs or the inner EAP exchange
	// carried inside the tunnel violate RFC 9930 §3.6 or §4.2.
	ErrTEAPInnerProtocol = errors.New("EAP-TEAP inner Phase 2 protocol violation")
	// ErrTEAPCryptoBinding is returned when the peer's Crypto-Binding TLV does
	// not verify, names the wrong Sub-Type, or omits a Compound MAC the session
	// requires (RFC 9930 §4.2.13, §6.2.4, §6.3).
	ErrTEAPCryptoBinding = errors.New("EAP-TEAP Crypto-Binding verification failed")
	// ErrTEAPPeerFailure is returned when the peer's Result or Intermediate-Result
	// TLV reports failure. RFC 9930 §3.6.2: "If either indicates failure, then the
	// authentication is considered failed."
	ErrTEAPPeerFailure = errors.New("EAP-TEAP peer reported a failed Phase 2 result")
	// ErrTEAPPassword is returned when the Basic-Password-Auth exchange of §3.6.3
	// cannot be completed: the policy, the round trip or the credential check
	// rejected it.
	ErrTEAPPassword = errors.New("EAP-TEAP Basic-Password-Auth failed")
)
