// Package redact removes secrets and customer identifiers from text that
// leaves the process.
//
// It backs the operator log export (internal/adminapi) and the local support
// bundle (internal/supportbundle), so both surfaces apply one rule set instead
// of drifting apart. The rules are intentionally conservative: a value is
// removed or pseudonymized whenever its key looks sensitive, and free text is
// swept for credential assignments, URL credentials, e-mail addresses and
// IPv4 addresses even when the surrounding key is unknown.
package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Marker replaces secret values.
const Marker = "[REDACTED]"

// Class is the data class a field belongs to.
type Class int

const (
	// Plain covers operational values that stay readable, such as usernames,
	// namespaces, levels and row identifiers.
	Plain Class = iota
	// Secret covers credentials and other values that must never be exported.
	Secret
	// CustomerPII covers values that identify a subscriber (name, phone,
	// e-mail, address, NIK). They are exported as a stable pseudonym.
	CustomerPII
	// Address covers IP addresses. Only the network prefix is exported.
	Address
)

var (
	secretKeyFragments = []string{
		"password", "passwd", "passphrase", "secret", "token", "authorization",
		"apikey", "credential", "community", "privatekey",
	}
	piiKeyFragments = []string{
		"phone", "msisdn", "email", "address", "fullname", "customername", "contact",
	}

	// The key part allows surrounding letters, digits, '_', '-' and '.' so that
	// `db_password: ...` is recognized, not only a bare `password: ...`. An
	// optional Bearer/Basic scheme is consumed with the credential so the token
	// itself does not survive next to the key.
	secretAssignmentPattern = regexp.MustCompile(`(?i)([A-Za-z0-9_.\-]*(?:password|passwd|passphrase|secret|token|authorization|api[_-]?key|credential|private[_-]?key|community)[A-Za-z0-9_.\-]*)(\s*[=:]\s*)(?:bearer\s+|basic\s+)?("[^"]*"|'[^']*'|[^\s,;}]+)`)
	urlCredentialPattern    = regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.\-]*://)([^/\s:@]+):([^/\s@]+)@`)
	ipv4Pattern             = regexp.MustCompile(`\b(\d{1,3}\.\d{1,3}\.\d{1,3})\.\d{1,3}\b`)
	emailPattern            = regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)
)

// ClassifyKey returns the data class of a structured field key. Matching is
// case-insensitive and ignores '_', '-' and '.' separators.
func ClassifyKey(key string) Class {
	normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "", ".", "").Replace(key))
	if containsAnyFragment(normalized, secretKeyFragments) {
		return Secret
	}
	if normalized == "ip" || strings.HasSuffix(normalized, "ip") || strings.HasSuffix(normalized, "ipaddress") || strings.Contains(normalized, "ipaddr") {
		return Address
	}
	if normalized == "nik" || containsAnyFragment(normalized, piiKeyFragments) {
		return CustomerPII
	}
	return Plain
}

// Value redacts a structured field according to its key class. Plain string
// values are still swept as free text; other types pass through unchanged.
func Value(key string, value any) any {
	switch ClassifyKey(key) {
	case Secret:
		return Marker
	case CustomerPII:
		return Pseudonymize(toString(value))
	case Address:
		return MaskAddress(toString(value))
	default:
		if text, ok := value.(string); ok {
			return Text(text)
		}
		return value
	}
}

// Text sweeps free text for credential assignments, URL credentials, e-mail
// addresses and IPv4 addresses.
func Text(text string) string {
	text = secretAssignmentPattern.ReplaceAllString(text, "$1$2"+Marker)
	text = urlCredentialPattern.ReplaceAllString(text, "$1$2:"+Marker+"@")
	text = emailPattern.ReplaceAllStringFunc(text, Pseudonymize)
	return ipv4Pattern.ReplaceAllString(text, "$1.x")
}

// Pseudonymize returns a stable, non-reversible identifier so two records about
// the same customer can still be correlated after export.
func Pseudonymize(value string) string {
	if strings.TrimSpace(value) == "" {
		return value
	}
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

// MaskAddress keeps the IPv4 /24 or IPv6 /48 prefix. The host part identifies a
// subscriber; the prefix is enough to correlate.
func MaskAddress(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return value
	}
	if v4 := ip.To4(); v4 != nil {
		return fmt.Sprintf("%d.%d.%d.x", v4[0], v4[1], v4[2])
	}
	groups := strings.Split(ip.String(), ":")
	if len(groups) < 3 {
		return value
	}
	return strings.Join(groups[:3], ":") + "::x"
}

func containsAnyFragment(value string, fragments []string) bool {
	for _, fragment := range fragments {
		if strings.Contains(value, fragment) {
			return true
		}
	}
	return false
}

func toString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprint(value)
}
