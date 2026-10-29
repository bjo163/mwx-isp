package redact

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyKey(t *testing.T) {
	cases := map[string]Class{
		"password":          Secret,
		"db_password":       Secret,
		"database.passwd":   Secret,
		"backup.passphrase": Secret,
		"web.secret":        Secret,
		"api_key":           Secret,
		"authorization":     Secret,
		"snmp_community":    Secret,
		"tls.private_key":   Secret,
		"phone":             CustomerPII,
		"customer_name":     CustomerPII,
		"email":             CustomerPII,
		"address":           CustomerPII,
		"msisdn":            CustomerPII,
		"nik":               CustomerPII,
		"ip":                Address,
		"remote_ip":         Address,
		"framed_ipaddr":     Address,
		"nas_ip_address":    Address,
		"username":          Plain,
		"namespace":         Plain,
		"level":             Plain,
		"tenant_id":         Plain,
		"idempotency_key":   Plain,
	}
	for key, want := range cases {
		assert.Equal(t, want, ClassifyKey(key), key)
	}
}

func TestTextRemovesCredentialForms(t *testing.T) {
	cases := map[string]string{ //nolint:gosec // G101: deterministic redaction fixtures, not real credentials
		`password=hunter2 retry`:                     `password=[REDACTED] retry`,
		`db_password: s3cr3t, then continue`:         `db_password: [REDACTED], then continue`,
		`authorization: Bearer eyJhbGciOiJIUzI1NiJ9`: `authorization: [REDACTED]`,
		`api_key='abc def'`:                          `api_key=[REDACTED]`,
		`postgres://mwxisp:pw@db.internal:5432/isp`:  `postgres://mwxisp:[REDACTED]@db.internal:5432/isp`,
	}
	for input, want := range cases {
		assert.Equal(t, want, Text(input), input)
	}
}

func TestTextPseudonymizesAndMasksFreeText(t *testing.T) {
	out := Text("customer alice@example.com connected from 203.0.113.77")
	assert.NotContains(t, out, "alice@example.com")
	assert.NotContains(t, out, "203.0.113.77")
	assert.Contains(t, out, Pseudonymize("alice@example.com"))
	assert.Contains(t, out, "203.0.113.x")
}

func TestValueByClass(t *testing.T) {
	assert.Equal(t, Marker, Value("password", "hunter2"))
	assert.Equal(t, Pseudonymize("+6281234567890"), Value("phone", "+6281234567890"))
	assert.Equal(t, "203.0.113.x", Value("remote_ip", "203.0.113.77"))
	assert.Equal(t, "alice", Value("username", "alice"))
	assert.Equal(t, 42, Value("tenant_id", 42), "non-strings on plain keys pass through")
	assert.Equal(t, "see password=[REDACTED]", Value("message", "see password=hunter2"))
}

func TestPseudonymizeAndMask(t *testing.T) {
	assert.Equal(t, "", Pseudonymize(""))
	assert.Equal(t, Pseudonymize("+6281234567890"), Pseudonymize("+6281234567890"),
		"the same input must pseudonymize to the same value for correlation")
	assert.NotEqual(t, Pseudonymize("+6281234567890"), Pseudonymize("+6281234567891"))
	assert.Contains(t, Pseudonymize("+6281234567890"), "sha256:")

	assert.Equal(t, "203.0.113.x", MaskAddress("203.0.113.77"))
	assert.Equal(t, "2001:db8:1234::x", MaskAddress("2001:db8:1234:5678::1"))
	assert.Equal(t, "not-an-address", MaskAddress("not-an-address"))
	assert.Equal(t, "", MaskAddress(""))
}
