package adminapi

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// The backup envelope wraps the versioned SystemBackup payload with a manifest
// and, optionally, authenticated encryption. Bare SystemBackup payloads remain
// accepted on restore for backups produced before the envelope existed.
const (
	backupEnvelopeFormat  = "mwx-isp-backup"
	backupEnvelopeVersion = 1

	backupKDFName       = "pbkdf2-sha256"
	backupKDFIterations = 600000
	backupSaltSize      = 16
	backupKeySize       = 32

	// backupKDFMinIterations is the lowest recorded iteration count accepted on
	// restore. Restores accept a range instead of exact equality so raising the
	// write cost never strands archives written by an earlier release, while a
	// crafted archive cannot ask the server to derive a weaker key than this
	// release writes.
	backupKDFMinIterations = backupKDFIterations
	// backupKDFMaxIterations caps the CPU a single archive can demand before it
	// is authenticated. Raising backupKDFIterations must stay below this bound.
	backupKDFMaxIterations = 4 * backupKDFIterations

	// backupMinPassphraseLength applies to newly written encrypted archives.
	// Restoring is exempt: an archive written before this policy must stay
	// readable with whatever passphrase encrypted it.
	backupMinPassphraseLength = 12
)

var (
	// errBackupEncryptedNeedsPassphrase reports that the archive is encrypted but
	// the server has no TOUGHRADIUS_BACKUP_PASSPHRASE configured.
	errBackupEncryptedNeedsPassphrase = errors.New("backup is encrypted and no backup passphrase is configured")
	// errBackupDecryption reports an authentication failure while opening an
	// encrypted archive. It intentionally does not distinguish a wrong
	// passphrase from a corrupted archive.
	errBackupDecryption = errors.New("backup decryption failed")
	// errBackupPassphraseTooShort reports that the configured passphrase cannot
	// protect a new archive. Existing archives with a short passphrase remain
	// restorable; only writing is refused.
	errBackupPassphraseTooShort = fmt.Errorf("backup passphrase must be at least %d characters", backupMinPassphraseLength)
)

// backupManifest describes the payload contained in a backup envelope.
//
// RecordCounts is computed from the serialized payload and re-checked on
// restore so a truncated or edited archive is rejected before any write.
// PayloadSHA256 binds the manifest to the canonical payload re-encoding; for
// encrypted archives it covers the decrypted plaintext.
type backupManifest struct {
	// FormatVersion is the envelope container version.
	FormatVersion int `json:"format_version"`
	// SchemaVersion is the SystemBackup schema version of the payload.
	SchemaVersion string `json:"schema_version"`
	// CreatedAt is the payload creation timestamp, mirrored from the payload.
	CreatedAt time.Time `json:"created_at"`
	// Encrypted reports whether the payload is stored as ciphertext.
	Encrypted bool `json:"encrypted"`
	// RecordCounts maps each backup table to the number of serialized records.
	RecordCounts map[string]int `json:"record_counts"`
	// PayloadSHA256 is the hex-encoded SHA-256 of the canonical payload re-encoding.
	PayloadSHA256 string `json:"payload_sha256"`
	// KDF is the key derivation function used for encrypted archives.
	KDF string `json:"kdf,omitempty"`
	// KDFIterations is the PBKDF2 iteration count for encrypted archives.
	KDFIterations int `json:"kdf_iterations,omitempty"`
	// Salt is the base64-encoded KDF salt for encrypted archives.
	Salt string `json:"salt,omitempty"`
}

// backupEnvelope is the top-level backup file container.
//
// Exactly one of Payload (plaintext) or Ciphertext (base64 nonce||AES-GCM) is
// populated. The manifest is never encrypted, but for encrypted archives it is
// bound to the ciphertext through the AEAD additional data, so editing any
// manifest field also breaks authentication.
type backupEnvelope struct {
	Format     string          `json:"format"`
	Manifest   backupManifest  `json:"manifest"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Ciphertext string          `json:"ciphertext,omitempty"`
}

// backupRecordCounts returns the number of records per backup table. The keys
// are stable identifiers used by the manifest and are independent of record
// order.
func backupRecordCounts(b *SystemBackup) map[string]int {
	counts := map[string]int{
		"tenants":               len(b.Tenants),
		"tenant_memberships":    len(b.Memberships),
		"configs":               len(b.Configs),
		"operators":             len(b.Operators),
		"certs":                 len(b.Certs),
		"accounting":            len(b.Accounting),
		"session_actions":       len(b.SessionActions),
		"hotspot_batches":       len(b.HotspotBatches),
		"hotspot_vouchers":      len(b.Vouchers),
		"monitor_targets":       len(b.MonitorTargets),
		"monitor_samples":       len(b.MonitorSamples),
		"monitor_incidents":     len(b.MonitorIncidents),
		"notification_settings": len(b.NotificationSettings),
		"notification_outbox":   len(b.NotificationOutbox),
		"operator_logs":         len(b.OprLogs),
	}
	for table, rows := range tenantOwnedBackupRows(b) {
		counts[table] = reflect.ValueOf(rows).Len()
	}
	return counts
}

// buildBackupEnvelope serializes b into a manifest-carrying envelope. When
// passphrase is non-empty the payload is encrypted with AES-256-GCM and a key
// derived through PBKDF2-SHA256; the passphrase itself is never stored.
func buildBackupEnvelope(b *SystemBackup, passphrase string) ([]byte, bool, error) {
	passphrase = strings.TrimSpace(passphrase)
	if passphrase != "" && utf8.RuneCountInString(passphrase) < backupMinPassphraseLength {
		return nil, false, errBackupPassphraseTooShort
	}
	payload, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("encode backup payload: %w", err)
	}
	digest, err := backupPayloadDigest(b)
	if err != nil {
		return nil, false, err
	}
	manifest := backupManifest{
		FormatVersion: backupEnvelopeVersion,
		SchemaVersion: b.Version,
		CreatedAt:     b.CreatedAt,
		RecordCounts:  backupRecordCounts(b),
		PayloadSHA256: digest,
	}
	envelope := backupEnvelope{Format: backupEnvelopeFormat, Manifest: manifest}
	encrypted := passphrase != ""
	if encrypted {
		salt := make([]byte, backupSaltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, false, fmt.Errorf("generate backup salt: %w", err)
		}
		manifest.Encrypted = true
		manifest.KDF = backupKDFName
		manifest.KDFIterations = backupKDFIterations
		manifest.Salt = base64.StdEncoding.EncodeToString(salt)

		ciphertext, err := encryptBackupPayload(passphrase, salt, payload, manifest)
		if err != nil {
			return nil, false, err
		}
		envelope.Manifest = manifest
		envelope.Ciphertext = ciphertext
	} else {
		envelope.Payload = payload
	}

	body, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return nil, false, fmt.Errorf("encode backup envelope: %w", err)
	}
	return body, encrypted, nil
}

// encryptBackupPayload seals payload with AES-256-GCM. The manifest (excluding
// the ciphertext itself) is authenticated as additional data, and its recorded
// KDFIterations drives the derivation cost so a version bump only has to change
// the count written for new archives, not the decryption logic.
func encryptBackupPayload(passphrase string, salt, payload []byte, manifest backupManifest) (string, error) {
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, manifest.KDFIterations, backupKeySize)
	if err != nil {
		return "", fmt.Errorf("derive backup key: %w", err)
	}
	aead, err := newBackupAEAD(key)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate backup nonce: %w", err)
	}
	sealed := aead.Seal(nil, nonce, payload, backupManifestAAD(manifest))
	return base64.StdEncoding.EncodeToString(append(nonce, sealed...)), nil
}

// decryptBackupPayload authenticates and opens a ciphertext produced by
// encryptBackupPayload. Any manifest edit also fails authentication because the
// manifest is bound as additional data.
func decryptBackupPayload(passphrase, ciphertext string, manifest backupManifest) ([]byte, error) {
	if manifest.KDF != backupKDFName {
		return nil, fmt.Errorf("%w: unsupported key derivation %q", errBackupDecryption, manifest.KDF)
	}
	if err := backupKDFIterationCount(manifest.KDFIterations); err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(manifest.Salt)
	if err != nil || len(salt) < backupSaltSize {
		return nil, fmt.Errorf("%w: invalid KDF salt", errBackupDecryption)
	}
	raw, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid ciphertext encoding", errBackupDecryption)
	}
	key, err := pbkdf2.Key(sha256.New, passphrase, salt, manifest.KDFIterations, backupKeySize)
	if err != nil {
		return nil, fmt.Errorf("%w: derive backup key", errBackupDecryption)
	}
	aead, err := newBackupAEAD(key)
	if err != nil {
		return nil, err
	}
	if len(raw) < aead.NonceSize()+aead.Overhead() {
		return nil, fmt.Errorf("%w: ciphertext is truncated", errBackupDecryption)
	}
	nonce, sealed := raw[:aead.NonceSize()], raw[aead.NonceSize():]
	payload, err := aead.Open(nil, nonce, sealed, backupManifestAAD(manifest))
	if err != nil {
		return nil, errBackupDecryption
	}
	return payload, nil
}

// backupKDFIterationCount validates a manifest-recorded iteration count against
// the range this release can open without changing the envelope format.
func backupKDFIterationCount(iterations int) error {
	if iterations < backupKDFMinIterations || iterations > backupKDFMaxIterations {
		return fmt.Errorf("%w: KDF iteration count %d is outside the supported range [%d, %d]",
			errBackupDecryption, iterations, backupKDFMinIterations, backupKDFMaxIterations)
	}
	return nil
}

func newBackupAEAD(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create backup cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create backup authenticator: %w", err)
	}
	return aead, nil
}

// backupManifestAAD binds the stable manifest fields to the ciphertext. It uses
// only strings and integers so both sides construct identical bytes.
func backupManifestAAD(manifest backupManifest) []byte {
	return []byte(fmt.Sprintf("%s\x00%d\x00%s\x00%s\x00%d",
		backupEnvelopeFormat, manifest.FormatVersion, manifest.SchemaVersion,
		manifest.PayloadSHA256, manifest.KDFIterations))
}

// openBackupPayload extracts the raw SystemBackup payload from either a
// manifest-carrying envelope or a legacy bare payload. It validates the
// container version and decrypts ciphertext; a nil manifest means the archive
// predates the envelope and only structural validation applies. The manifest
// itself is verified against the decoded payload by verifyBackupManifest.
func openBackupPayload(raw []byte, passphrase string) ([]byte, *backupManifest, error) {
	var probe struct {
		Format     string          `json:"format"`
		Manifest   json.RawMessage `json:"manifest"`
		Payload    json.RawMessage `json:"payload"`
		Ciphertext string          `json:"ciphertext"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, nil, fmt.Errorf("invalid backup file format: %w", err)
	}
	if probe.Format == "" {
		return raw, nil, nil
	}
	if probe.Format != backupEnvelopeFormat {
		return nil, nil, fmt.Errorf("unsupported backup container %q", probe.Format)
	}
	if len(probe.Manifest) == 0 {
		return nil, nil, errors.New("backup manifest is missing")
	}
	var manifest backupManifest
	if err := json.Unmarshal(probe.Manifest, &manifest); err != nil {
		return nil, nil, fmt.Errorf("invalid backup manifest: %w", err)
	}
	if manifest.FormatVersion != backupEnvelopeVersion {
		return nil, nil, fmt.Errorf("unsupported backup container version %d", manifest.FormatVersion)
	}

	var payload []byte
	switch {
	case probe.Ciphertext != "":
		if !manifest.Encrypted {
			return nil, &manifest, errors.New("backup manifest does not declare the encrypted payload")
		}
		if strings.TrimSpace(passphrase) == "" {
			return nil, &manifest, errBackupEncryptedNeedsPassphrase
		}
		plaintext, err := decryptBackupPayload(passphrase, probe.Ciphertext, manifest)
		if err != nil {
			return nil, &manifest, err
		}
		payload = plaintext
	case len(probe.Payload) > 0 && string(probe.Payload) != "null":
		if manifest.Encrypted {
			return nil, &manifest, errors.New("backup manifest declares encryption but the payload is plaintext")
		}
		payload = probe.Payload
	default:
		return nil, &manifest, errors.New("backup contains neither payload nor ciphertext")
	}

	return payload, &manifest, nil
}

// backupPayloadDigest returns the canonical digest stored in the manifest.
// Hashing a canonical re-encoding (rather than the transmitted bytes) keeps the
// digest stable when an archive is re-indented or re-wrapped by tooling, while
// still detecting any change to record contents.
func backupPayloadDigest(b *SystemBackup) (string, error) {
	canonical, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("encode canonical backup payload: %w", err)
	}
	return sha256Hex(canonical), nil
}

// verifyBackupManifest re-checks the manifest against the decoded payload so a
// valid hash over a mismatched manifest cannot smuggle unexpected record sets
// into a restore.
func verifyBackupManifest(manifest *backupManifest, b *SystemBackup) error {
	if manifest.SchemaVersion != b.Version {
		return fmt.Errorf("manifest schema version %q does not match payload version %q", manifest.SchemaVersion, b.Version)
	}
	if !manifest.CreatedAt.Equal(b.CreatedAt) {
		return errors.New("manifest creation time does not match the payload")
	}
	digest, err := backupPayloadDigest(b)
	if err != nil {
		return err
	}
	if digest != manifest.PayloadSHA256 {
		return errors.New("backup payload integrity check failed")
	}
	counts := backupRecordCounts(b)
	for table, want := range manifest.RecordCounts {
		got, known := counts[table]
		if !known {
			return fmt.Errorf("manifest lists unknown table %q", table)
		}
		if got != want {
			return fmt.Errorf("manifest records %d %s rows but the payload contains %d", want, table, got)
		}
	}
	for table, got := range counts {
		if _, ok := manifest.RecordCounts[table]; !ok {
			return fmt.Errorf("manifest is missing the %s record count", table)
		}
		if manifest.RecordCounts[table] != got {
			return fmt.Errorf("manifest records %d %s rows but the payload contains %d", manifest.RecordCounts[table], table, got)
		}
	}
	return nil
}

// backupRestoreResult reports how many records a payload would restore. It is
// shared by dry-run responses and mirrors the per-table counters used during a
// real restore.
func backupRestoreResult(b *SystemBackup) SystemRestoreResult {
	return SystemRestoreResult{
		Tenants: len(b.Tenants), Nodes: len(b.Nodes), Nas: len(b.Nas),
		Profiles: len(b.Profiles), Users: len(b.Users), Configs: len(b.Configs),
		Operators: len(b.Operators), Certs: len(b.Certs), Accounting: len(b.Accounting),
		SessionActions: len(b.SessionActions), Customers: len(b.Customers),
		Packages: len(b.Packages), Subscriptions: len(b.Subscriptions),
		Invoices: len(b.Invoices), InvoiceItems: len(b.InvoiceItems),
		Payments: len(b.Payments), BillingEvents: len(b.BillingEvents),
		Sequences: len(b.Sequences), HotspotBatches: len(b.HotspotBatches),
		Vouchers: len(b.Vouchers), IPAMPools: len(b.IPAMPools),
		TroubleTickets: len(b.TroubleTickets), ODPs: len(b.ODPs),
		MonitorTargets: len(b.MonitorTargets), MonitorSamples: len(b.MonitorSamples),
		MonitorIncidents:     len(b.MonitorIncidents),
		NotificationSettings: len(b.NotificationSettings), NotificationOutbox: len(b.NotificationOutbox),
		OprLogs: len(b.OprLogs),
	}
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
