package adminapi

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

// TestBackupKDFPolicyStaysConsistent keeps the written cost inside the window
// restores accept, so raising the default cannot write archives the same
// release would refuse to open.
func TestBackupKDFPolicyStaysConsistent(t *testing.T) {
	if backupKDFMinIterations > backupKDFIterations {
		t.Fatalf("minimum accepted iterations %d exceeds the written count %d", backupKDFMinIterations, backupKDFIterations)
	}
	if backupKDFIterations > backupKDFMaxIterations {
		t.Fatalf("written iterations %d exceed the accepted maximum %d", backupKDFIterations, backupKDFMaxIterations)
	}
}

// BenchmarkBackupKeyDerivation measures the PBKDF2-SHA256 cost at the iteration
// counts that bracket the envelope policy. It is the evidence behind the
// 600,000 default and the accepted restore range; rerun it on the target
// deployment hardware before changing either number.
//
//	go test ./internal/adminapi -run '^$' -bench BenchmarkBackupKeyDerivation -benchtime 3x
func BenchmarkBackupKeyDerivation(b *testing.B) {
	salt := make([]byte, backupSaltSize)
	for _, iterations := range []int{100000, 310000, backupKDFIterations, 1200000, backupKDFMaxIterations} {
		b.Run(fmt.Sprintf("iterations_%d", iterations), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				if _, err := pbkdf2.Key(sha256.New, backupTestPassphrase, salt, iterations, backupKeySize); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkBackupEnvelopeRoundTrip measures the full write-and-open cost of an
// encrypted archive, so the operator-visible backup latency stays tied to the
// measured key derivation cost instead of an estimate.
func BenchmarkBackupEnvelopeRoundTrip(b *testing.B) {
	backup := SystemBackup{Version: backupVersion, CreatedAt: time.Now().UTC()}
	body, _, err := buildBackupEnvelope(&backup, backupTestPassphrase)
	if err != nil {
		b.Fatalf("build envelope: %v", err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, _, err := openBackupPayload(body, backupTestPassphrase); err != nil {
			b.Fatalf("open envelope: %v", err)
		}
	}
}
