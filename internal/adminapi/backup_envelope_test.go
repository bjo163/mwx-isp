package adminapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/domain"
)

const backupTestPassphrase = "correct horse battery staple"

// decodeBackupEnvelope splits a backup response into its manifest envelope and
// the decoded SystemBackup payload.
func decodeBackupEnvelope(t *testing.T, body []byte) (backupEnvelope, SystemBackup) {
	t.Helper()
	var env backupEnvelope
	require.NoErrorf(t, json.Unmarshal(body, &env), "backup response is not an envelope: %s", string(body))
	require.Equal(t, backupEnvelopeFormat, env.Format)
	var payload SystemBackup
	require.NoError(t, json.Unmarshal(env.Payload, &payload))
	return env, payload
}

func TestBackupEnvelopeManifestMatchesPayload(t *testing.T) {
	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	profile := createTestProfile(srcDB, "envelope-profile")
	createTestUser(srcDB, "envelope_user", profile.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))

	env, backup := decodeBackupEnvelope(t, rec.Body.Bytes())
	assert.Equal(t, backupEnvelopeVersion, env.Manifest.FormatVersion)
	assert.Equal(t, backupVersion, env.Manifest.SchemaVersion)
	assert.Equal(t, backup.Version, env.Manifest.SchemaVersion)
	assert.True(t, env.Manifest.CreatedAt.Equal(backup.CreatedAt))
	digest, err := backupPayloadDigest(&backup)
	require.NoError(t, err)
	assert.Equal(t, digest, env.Manifest.PayloadSHA256)
	assert.False(t, env.Manifest.Encrypted)
	assert.Empty(t, env.Ciphertext)
	assert.Equal(t, 1, env.Manifest.RecordCounts["radius_user"])
	assert.Equal(t, 1, env.Manifest.RecordCounts["radius_profile"])
	assert.Equal(t, 0, env.Manifest.RecordCounts["hotspot_vouchers"])

	dstDB := setupBackupTestDB(t)
	dstAppCtx := setupTestApp(t, dstDB)
	rreq, rrec := restoreRequest(t, rec.Body.Bytes())
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), dstDB, rreq, rrec, dstAppCtx)))
	assert.Equal(t, http.StatusOK, rrec.Code, rrec.Body.String())
	var restored domain.RadiusUser
	require.NoError(t, dstDB.Where("username = ?", "envelope_user").First(&restored).Error)
	assert.Equal(t, "testpass123", restored.Password)
}

func TestEncryptedBackupRequiresMatchingPassphrase(t *testing.T) {
	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	srcAppCtx.Config().Backup.Passphrase = backupTestPassphrase
	profile := createTestProfile(srcDB, "encrypted-profile")
	createTestUser(srcDB, "encrypted_user", profile.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))
	body := rec.Body.Bytes()

	var env backupEnvelope
	require.NoError(t, json.Unmarshal(body, &env))
	require.True(t, env.Manifest.Encrypted)
	require.NotEmpty(t, env.Ciphertext)
	require.Empty(t, env.Payload)
	require.Equal(t, backupKDFName, env.Manifest.KDF)
	require.Equal(t, backupKDFIterations, env.Manifest.KDFIterations)
	require.NotEmpty(t, env.Manifest.Salt)
	assert.NotContains(t, string(body), "encrypted_user", "encrypted backups must not leak record contents")
	assert.NotContains(t, string(body), "testpass123", "encrypted backups must not leak credentials")
	assert.NotContains(t, string(body), backupTestPassphrase, "the passphrase must never be stored in the archive")
	assert.Contains(t, rec.Header().Get("Content-Disposition"), ".encrypted.json")

	// A server without a passphrase must refuse the encrypted archive.
	missingDB := setupBackupTestDB(t)
	missingAppCtx := setupTestApp(t, missingDB)
	rreq, rrec := restoreRequest(t, body)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), missingDB, rreq, rrec, missingAppCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "BACKUP_ENCRYPTED")
	assert.Zero(t, countRows(t, missingDB, &domain.RadiusUser{}))

	// A wrong passphrase must fail authentication without writing anything.
	wrongDB := setupBackupTestDB(t)
	wrongAppCtx := setupTestApp(t, wrongDB)
	wrongAppCtx.Config().Backup.Passphrase = "not the right passphrase"
	rreq, rrec = restoreRequest(t, body)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), wrongDB, rreq, rrec, wrongAppCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "BACKUP_DECRYPT_FAILED")
	assert.Zero(t, countRows(t, wrongDB, &domain.RadiusUser{}))

	// The matching passphrase restores the payload.
	dstDB := setupBackupTestDB(t)
	dstAppCtx := setupTestApp(t, dstDB)
	dstAppCtx.Config().Backup.Passphrase = backupTestPassphrase
	rreq, rrec = restoreRequest(t, body)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), dstDB, rreq, rrec, dstAppCtx)))
	require.Equal(t, http.StatusOK, rrec.Code, rrec.Body.String())
	var restored domain.RadiusUser
	require.NoError(t, dstDB.Where("username = ?", "encrypted_user").First(&restored).Error)
	assert.Equal(t, "testpass123", restored.Password)
}

func TestRestoreRejectsTamperedBackup(t *testing.T) {
	// Plaintext payload tampering is caught by the manifest hash.
	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	profile := createTestProfile(srcDB, "tamper-profile")
	createTestUser(srcDB, "tamper_user", profile.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))

	var env backupEnvelope
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &env))
	env.Payload = bytes.Replace(env.Payload, []byte("tamper_user"), []byte("evil_user"), 1)
	tampered, err := json.Marshal(env)
	require.NoError(t, err)

	dstDB := setupBackupTestDB(t)
	dstAppCtx := setupTestApp(t, dstDB)
	sentinel := createTestUser(dstDB, "sentinel_user", profile.ID)
	rreq, rrec := restoreRequest(t, tampered)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), dstDB, rreq, rrec, dstAppCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "INVALID_BACKUP")
	assert.Equal(t, int64(1), countRows(t, dstDB, &domain.RadiusUser{}), "failed restore must not write rows")
	var stillThere domain.RadiusUser
	require.NoError(t, dstDB.First(&stillThere, sentinel.ID).Error, "failed restore must not change existing rows")

	// Encrypted ciphertext tampering fails AEAD authentication.
	encDB := setupBackupTestDB(t)
	encAppCtx := setupTestApp(t, encDB)
	encAppCtx.Config().Backup.Passphrase = backupTestPassphrase
	ereq := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	erec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), encDB, ereq, erec, encAppCtx)))

	var encEnv backupEnvelope
	require.NoError(t, json.Unmarshal(erec.Body.Bytes(), &encEnv))
	raw, err := base64.StdEncoding.DecodeString(encEnv.Ciphertext)
	require.NoError(t, err)
	raw[len(raw)-1] ^= 0xff
	encEnv.Ciphertext = base64.StdEncoding.EncodeToString(raw)
	tamperedCipher, err := json.Marshal(encEnv)
	require.NoError(t, err)

	targetDB := setupBackupTestDB(t)
	targetAppCtx := setupTestApp(t, targetDB)
	targetAppCtx.Config().Backup.Passphrase = backupTestPassphrase
	rreq, rrec = restoreRequest(t, tamperedCipher)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), targetDB, rreq, rrec, targetAppCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "BACKUP_DECRYPT_FAILED")
	assert.Zero(t, countRows(t, targetDB, &domain.RadiusUser{}))
}

func TestRestoreRejectsManifestCountMismatch(t *testing.T) {
	backup := SystemBackup{
		Version:   backupVersion,
		CreatedAt: time.Now().UTC().Truncate(time.Second),
		Tenants:   []domain.Tenant{{ID: 1, Name: "Default", Slug: "default"}},
		Users:     []domain.RadiusUser{{ID: 2, TenantID: 1, Username: "real_user", Password: "pw", Status: "enabled"}},
	}
	body, _, err := buildBackupEnvelope(&backup, "")
	require.NoError(t, err)

	var env backupEnvelope
	require.NoError(t, json.Unmarshal(body, &env))
	env.Manifest.RecordCounts["radius_user"] = 9
	mutated, err := json.Marshal(env)
	require.NoError(t, err)

	db := setupBackupTestDB(t)
	appCtx := setupTestApp(t, db)
	rreq, rrec := restoreRequest(t, mutated)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), db, rreq, rrec, appCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "INVALID_BACKUP")
	assert.Zero(t, countRows(t, db, &domain.RadiusUser{}))
}

func TestRestoreDryRunPerformsZeroMutations(t *testing.T) {
	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	profile := createTestProfile(srcDB, "dry-run-profile")
	createTestUser(srcDB, "dry_run_user", profile.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))
	payload := rec.Body.Bytes()

	dstDB := setupBackupTestDB(t)
	dstAppCtx := setupTestApp(t, dstDB)
	sentinel := createTestUser(dstDB, "dry_run_sentinel", profile.ID)

	type dryRunResponse struct {
		Data struct {
			DryRun       bool                `json:"dry_run"`
			Encrypted    bool                `json:"encrypted"`
			WouldRestore SystemRestoreResult `json:"would_restore"`
			Manifest     *backupManifest     `json:"manifest"`
		} `json:"data"`
	}

	// Query-parameter form.
	rreq, rrec := restoreRequest(t, payload)
	rreq.URL.RawQuery = "dry_run=true"
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), dstDB, rreq, rrec, dstAppCtx)))
	require.Equal(t, http.StatusOK, rrec.Code, rrec.Body.String())
	var resp dryRunResponse
	require.NoError(t, json.Unmarshal(rrec.Body.Bytes(), &resp))
	assert.True(t, resp.Data.DryRun)
	assert.False(t, resp.Data.Encrypted)
	assert.Equal(t, 1, resp.Data.WouldRestore.Users)
	require.NotNil(t, resp.Data.Manifest)
	assert.Equal(t, 1, resp.Data.Manifest.RecordCounts["radius_user"])
	assert.Equal(t, int64(1), countRows(t, dstDB, &domain.RadiusUser{}), "dry-run must not insert rows")

	// Multipart form-field form.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	require.NoError(t, writer.WriteField("dry_run", "1"))
	part, err := writer.CreateFormFile("upload", "backup.json")
	require.NoError(t, err)
	_, err = part.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	freq := httptest.NewRequest(http.MethodPost, "/api/v1/system/restore", body)
	freq.Header.Set("Content-Type", writer.FormDataContentType())
	frec := httptest.NewRecorder()
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), dstDB, freq, frec, dstAppCtx)))
	require.Equal(t, http.StatusOK, frec.Code, frec.Body.String())
	require.NoError(t, json.Unmarshal(frec.Body.Bytes(), &resp))
	assert.True(t, resp.Data.DryRun)
	assert.Equal(t, int64(1), countRows(t, dstDB, &domain.RadiusUser{}))
	var unchanged domain.RadiusUser
	require.NoError(t, dstDB.First(&unchanged, sentinel.ID).Error)
	assert.Equal(t, "dry_run_sentinel", unchanged.Username)
}

func TestRestoreDryRunStillEnforcesOperatorAuthorization(t *testing.T) {
	db := setupBackupTestDB(t)
	appCtx := setupTestApp(t, db)

	payload := []byte(`{"version":"9.0","operators":[{"id":"42","username":"newadmin","level":"super","status":"enabled"}]}`)
	rreq, rrec := restoreRequest(t, payload)
	rreq.URL.RawQuery = "dry_run=true"
	c := CreateTestContext(setupTestEcho(), db, rreq, rrec, appCtx)
	c.Set("current_operator", &domain.SysOpr{ID: 7, Username: "admin", Level: LevelAdmin, Status: "enabled"})

	require.NoError(t, restoreSystem(c))
	assert.Equal(t, http.StatusForbidden, rrec.Code)
	assert.Zero(t, countRows(t, db, &domain.SysOpr{}))
}

func TestRestoreRejectsUnknownBackupContainer(t *testing.T) {
	db := setupBackupTestDB(t)
	appCtx := setupTestApp(t, db)

	rreq, rrec := restoreRequest(t, []byte(`{"format":"some-other-tool","payload":{}}`))
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), db, rreq, rrec, appCtx)))
	assert.Equal(t, http.StatusBadRequest, rrec.Code)
	assert.Contains(t, rrec.Body.String(), "INVALID_BACKUP")
}

func countRows(t *testing.T, db *gorm.DB, model any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(model).Count(&count).Error)
	return count
}

// newBackupFixture exports a small but fully-formed backup from a disposable
// database so encrypted fixtures carry the same tenant metadata a real archive
// has.
func newBackupFixture(t *testing.T) SystemBackup {
	t.Helper()
	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	profile := createTestProfile(srcDB, "compat-profile")
	createTestUser(srcDB, "compat_user", profile.ID)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))
	_, backup := decodeBackupEnvelope(t, rec.Body.Bytes())
	return backup
}

// buildEncryptedFixtureAtIterations returns an encrypted envelope exactly as a
// release writing the given KDF cost would have produced, so restore
// compatibility can be exercised against counts other than the current default.
func buildEncryptedFixtureAtIterations(t *testing.T, backup *SystemBackup, passphrase string, iterations int) []byte {
	t.Helper()
	payload, err := json.MarshalIndent(backup, "", "  ")
	require.NoError(t, err)
	digest, err := backupPayloadDigest(backup)
	require.NoError(t, err)
	salt := []byte("0123456789abcdef")
	manifest := backupManifest{
		FormatVersion: backupEnvelopeVersion,
		SchemaVersion: backup.Version,
		CreatedAt:     backup.CreatedAt,
		Encrypted:     true,
		RecordCounts:  backupRecordCounts(backup),
		PayloadSHA256: digest,
		KDF:           backupKDFName,
		KDFIterations: iterations,
		Salt:          base64.StdEncoding.EncodeToString(salt),
	}
	ciphertext, err := encryptBackupPayload(passphrase, salt, payload, manifest)
	require.NoError(t, err)
	body, err := json.Marshal(backupEnvelope{
		Format:     backupEnvelopeFormat,
		Manifest:   manifest,
		Ciphertext: ciphertext,
	})
	require.NoError(t, err)
	return body
}

// restoreEncryptedFixture restores body into a fresh database with passphrase
// configured and returns the recorder.
func restoreEncryptedFixture(t *testing.T, body []byte, passphrase string) (*httptest.ResponseRecorder, *gorm.DB) {
	t.Helper()
	db := setupBackupTestDB(t)
	appCtx := setupTestApp(t, db)
	appCtx.Config().Backup.Passphrase = passphrase
	rreq, rrec := restoreRequest(t, body)
	require.NoError(t, restoreSystem(CreateTestContext(setupTestEcho(), db, rreq, rrec, appCtx)))
	return rrec, db
}

// TestEncryptedBackupRestoresAcrossSupportedIterationCounts pins the envelope
// compatibility contract: the recorded KDF cost is read from the manifest, so a
// later release may raise the write cost without making earlier archives
// undecryptable and without changing the container format.
func TestEncryptedBackupRestoresAcrossSupportedIterationCounts(t *testing.T) {
	backup := newBackupFixture(t)
	for _, iterations := range []int{backupKDFMinIterations, backupKDFIterations + 1, backupKDFMaxIterations} {
		t.Run(fmt.Sprintf("iterations_%d", iterations), func(t *testing.T) {
			body := buildEncryptedFixtureAtIterations(t, &backup, backupTestPassphrase, iterations)

			rrec, db := restoreEncryptedFixture(t, body, backupTestPassphrase)
			require.Equal(t, http.StatusOK, rrec.Code, rrec.Body.String())
			var restored domain.RadiusUser
			require.NoError(t, db.Where("username = ?", "compat_user").First(&restored).Error)
		})
	}
}

// TestEncryptedBackupRejectsUnsupportedIterationCounts keeps the accepted range
// narrow: a crafted archive cannot downgrade the derivation cost or demand
// unbounded CPU before authentication.
func TestEncryptedBackupRejectsUnsupportedIterationCounts(t *testing.T) {
	backup := newBackupFixture(t)
	for _, iterations := range []int{0, backupKDFMinIterations - 1, backupKDFMaxIterations + 1, 1 << 30} {
		t.Run(fmt.Sprintf("iterations_%d", iterations), func(t *testing.T) {
			body := buildEncryptedFixtureAtIterations(t, &backup, backupTestPassphrase, iterations)

			rrec, db := restoreEncryptedFixture(t, body, backupTestPassphrase)
			assert.Equal(t, http.StatusBadRequest, rrec.Code)
			assert.Contains(t, rrec.Body.String(), "BACKUP_DECRYPT_FAILED")
			assert.Zero(t, countRows(t, db, &domain.RadiusUser{}))
		})
	}
}

// TestEncryptedBackupRefusesShortPassphrase pins the write-side policy while
// ensuring a legacy archive encrypted with a shorter passphrase still restores.
func TestEncryptedBackupRefusesShortPassphrase(t *testing.T) {
	backup := newBackupFixture(t)

	srcDB := setupBackupTestDB(t)
	srcAppCtx := setupTestApp(t, srcDB)
	srcAppCtx.Config().Backup.Passphrase = "short-pass"
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/backup", nil)
	rec := httptest.NewRecorder()
	require.NoError(t, backupSystem(CreateTestContext(setupTestEcho(), srcDB, req, rec, srcAppCtx)))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "BACKUP_PASSPHRASE_TOO_SHORT")

	legacy := buildEncryptedFixtureAtIterations(t, &backup, "short-pass", backupKDFIterations)
	rrec, db := restoreEncryptedFixture(t, legacy, "short-pass")
	require.Equal(t, http.StatusOK, rrec.Code, rrec.Body.String())
	var restored domain.RadiusUser
	require.NoError(t, db.Where("username = ?", "compat_user").First(&restored).Error)
}
