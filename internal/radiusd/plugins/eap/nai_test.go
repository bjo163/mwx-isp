package eap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAnonymousNAI(t *testing.T) {
	tests := []struct {
		name     string
		username string
		want     bool
	}{
		{"privacy profile from RFC 7542 section 2.4", "anonymous@example.com", true},
		{"omitted user part", "@example.com", true},
		{"bare anonymous", "anonymous", true},
		{"case-insensitive user part", "Anonymous@example.com", true},
		{"surrounding whitespace", " anonymous@example.com ", true},
		{"named subscriber", "alice@example.com", false},
		{"named subscriber without realm", "alice", false},
		{"a name that merely starts with anonymous", "anonymousguest@example.com", false},
		{"empty user part without realm", "@", false},
		{"empty username", "", false},
		{"realm-looking user part", "realm.example.com@anonymous", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsAnonymousNAI(tt.username))
		})
	}
}
