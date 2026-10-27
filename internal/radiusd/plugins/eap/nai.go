package eap

import "strings"

// IsAnonymousNAI reports whether username is an anonymous Network Access
// Identifier as described by RFC 7542 §2.4: an omitted user part ("@realm"),
// the literal user part "anonymous" ("anonymous@realm"), or the bare username
// "anonymous" that current practice uses in place of an omitted user part.
//
// An anonymous NAI names no subscriber; it exists so the request can be routed
// to the realm while the real identity stays confidential. A method that proves
// a separate identity -- EAP-TLS binds the certificate to the RADIUS user
// (RFC 5216 §5.2) -- must therefore not compare that identity against the outer
// identifier.
func IsAnonymousNAI(username string) bool {
	s := strings.TrimSpace(username)
	if s == "" {
		return false
	}

	at := strings.LastIndex(s, "@")
	if at < 0 {
		return strings.EqualFold(s, "anonymous")
	}

	local, realm := s[:at], s[at+1:]
	if realm == "" {
		return false
	}
	return local == "" || strings.EqualFold(local, "anonymous")
}
