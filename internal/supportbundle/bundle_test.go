package supportbundle

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//nolint:gosec // G101: deterministic redaction fixture, not a real credential
const testConfigYAML = `system:
  appid: mwx-isp
  location: Jakarta
  workdir: /var/lib/mwx
database:
  host: db.internal
  name: isp
  user: mwxisp
  passwd: hunter2-db
web:
  host: 127.0.0.1
  port: 8080
  secret: web-signing-secret
radiusd:
  radsec_key: /etc/mwx/radsec.key
backup:
  passphrase: backup-passphrase
notify:
  admin_email: noc@example.com
  db_dsn: postgres://mwxisp:hunter2-db@db.internal:5432/isp
`

const testLogPlain = "tail line with token=abc123 from 198.51.100.9"

var testLogJSON = `{"level":"info","msg":"login attempt for alice@example.com from 203.0.113.77 password=hunter2","username":"alice","password":"hunter2","authorization":"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig","customer_name":"Budi Santoso","phone":"+6281234567890","email":"alice@example.com","remote_ip":"203.0.113.77","tenant_id":1}`

var testNow = time.Date(2026, time.October, 29, 9, 0, 0, 0, time.UTC)

func writeFixture(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func stageFixture(t *testing.T) (*Staged, string, string) {
	t.Helper()
	work := t.TempDir()
	configPath := writeFixture(t, work, "mwx-isp.yml", testConfigYAML)
	logPath := writeFixture(t, work, "app.log", testLogJSON+"\n"+testLogPlain+"\n")
	outDir := filepath.Join(work, "out")
	staged, err := Stage(Options{
		OutputDir:  outDir,
		ConfigPath: configPath,
		LogPath:    logPath,
		AppVersion: "v0.15.0-test",
		Now:        testNow,
	})
	require.NoError(t, err)
	return staged, outDir, logPath
}

func TestStageRedactsSecretsAndCustomerData(t *testing.T) {
	staged, outDir, _ := stageFixture(t)

	assert.Equal(t, filepath.Join(outDir, "support-bundle-20261029T090000Z"), staged.Dir)

	config := readFile(t, filepath.Join(staged.Dir, configFile))
	for _, leak := range []string{"hunter2-db", "web-signing-secret", "backup-passphrase", "noc@example.com"} {
		assert.NotContains(t, config, leak, "%q must not survive config staging", leak)
	}
	assert.Contains(t, config, redact.Marker)
	assert.Contains(t, config, redact.Pseudonymize("noc@example.com"))
	assert.Contains(t, config, "postgres://mwxisp:"+redact.Marker+"@db.internal:5432/isp")
	// A certificate path is operational context, not key material.
	assert.Contains(t, config, "/etc/mwx/radsec.key")

	logs := readFile(t, filepath.Join(staged.Dir, logFile))
	for _, leak := range []string{
		"hunter2", "eyJhbGciOiJIUzI1NiJ9", "Budi Santoso", "+6281234567890",
		"alice@example.com", "203.0.113.77", "198.51.100.9", "abc123",
	} {
		assert.NotContains(t, logs, leak, "%q must not survive log staging", leak)
	}
	assert.Contains(t, logs, `"username":"alice"`, "the operational identifier stays searchable")
	assert.Contains(t, logs, "203.0.113.x")
	assert.Contains(t, logs, `"tenant_id":1`)

	report := readReport(t, staged.Dir)
	assert.Equal(t, 5, report.Totals.Secrets, "three config secrets plus two in the log line")
	assert.Equal(t, 4, report.Totals.PII, "admin_email plus three log fields")
	assert.Equal(t, 1, report.Totals.Addresses)
	assert.GreaterOrEqual(t, report.Totals.FreeText, 1, "the DSN carries an inline password")
	assert.Len(t, report.Sources, 2)
	configSource := report.Sources[0]
	assert.Equal(t, "config", configSource.Source)
	assert.Equal(t, 3, configSource.Counts.Secrets)
	assert.Equal(t, 1, configSource.Counts.PII)
	assert.Contains(t, configSource.Keys, "database.passwd")
	assert.Contains(t, configSource.Keys, "web.secret")
	assert.Contains(t, configSource.Keys, "backup.passphrase")
	assert.Equal(t, "logs", report.Sources[1].Source)
	assert.GreaterOrEqual(t, report.Sources[1].Counts.Secrets, 2)
	assert.GreaterOrEqual(t, report.Sources[1].Counts.PII, 3)
	assert.GreaterOrEqual(t, report.Sources[1].Counts.Addresses, 1)

	manifest := readManifestFile(t, staged.Dir)
	assert.Equal(t, Format, manifest.Format)
	assert.Equal(t, Version, manifest.Version)
	assert.Equal(t, "pending_review", manifest.Review.Status)
	assert.Len(t, manifest.Files, 4, "config, logs, redaction report, meta")
	assert.Len(t, manifest.Checksum, 64)
	for _, entry := range manifest.Files {
		content := readFile(t, filepath.Join(staged.Dir, filepath.FromSlash(entry.Path)))
		sum := sha256.Sum256([]byte(content))
		assert.Equal(t, hex.EncodeToString(sum[:]), entry.SHA256, entry.Path)
		assert.EqualValues(t, len(content), entry.Bytes, entry.Path)
	}

	entries, err := os.ReadDir(staged.Dir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.NotEqual(t, reviewFile, entry.Name(), "staging must not approve itself")
		assert.False(t, strings.HasSuffix(entry.Name(), ".tar.gz"), "staging must not export an archive")
	}
}

func TestStageIsDeterministicForTheSameInputs(t *testing.T) {
	first, _, _ := stageFixture(t)
	second, _, _ := stageFixture(t)
	assert.Equal(t, first.Manifest.Checksum, second.Manifest.Checksum)
}

func TestStageRejectsBadInputsWithoutLeavingABundle(t *testing.T) {
	work := t.TempDir()
	outDir := filepath.Join(work, "out")

	_, err := Stage(Options{ConfigPath: filepath.Join(work, "x.yml")})
	require.ErrorIs(t, err, ErrOutputDirRequired)

	_, err = Stage(Options{OutputDir: outDir})
	require.ErrorIs(t, err, ErrNothingToStage)

	broken := writeFixture(t, work, "broken.yml", "system: [unclosed\n")
	_, err = Stage(Options{OutputDir: outDir, ConfigPath: broken})
	require.Error(t, err)
	entries, readErr := os.ReadDir(outDir)
	if readErr == nil {
		assert.Empty(t, entries, "a parse failure must not leave a staged directory")
	}

	require.NoError(t, os.MkdirAll(outDir, 0o750))
	_, err = Stage(Options{OutputDir: outDir, LogPath: filepath.Join(work, "app.log"), LogTailBytes: -1})
	require.Error(t, err)
}

func TestStageBoundsTheLogTail(t *testing.T) {
	work := t.TempDir()
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, `{"level":"info","msg":"line `+strings.Repeat("x", 40)+`","username":"alice"}`)
	}
	logPath := writeFixture(t, work, "app.log", strings.Join(lines, "\n")+"\n")
	staged, err := Stage(Options{
		OutputDir:    filepath.Join(work, "out"),
		LogPath:      logPath,
		LogTailBytes: 300,
		Now:          testNow,
	})
	require.NoError(t, err)

	logs := readFile(t, filepath.Join(staged.Dir, logFile))
	kept := strings.Count(strings.TrimSuffix(logs, "\n"), "\n") + 1
	assert.Less(t, kept, 50, "the tail bound must drop older lines")
	assert.NotEmpty(t, kept)
	assert.NotContains(t, logs, "line xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n\"")
	for _, line := range strings.Split(strings.TrimSuffix(logs, "\n"), "\n") {
		assert.True(t, strings.HasPrefix(line, `{"level":"info"`), "a partial line must not survive: %q", line)
	}
}

func TestApproveRequiresReviewerAndMatchingChecksums(t *testing.T) {
	staged, outDir, _ := stageFixture(t)

	_, err := Approve(staged.Dir, "   ")
	require.ErrorIs(t, err, ErrReviewerRequired)
	assertNoArchive(t, outDir)

	// A staged file edited after staging is exactly what the review gate exists
	// to catch.
	tampered := filepath.Join(staged.Dir, configFile)
	require.NoError(t, os.WriteFile(tampered, []byte("system:\n  appid: changed\n"), 0o600))
	_, err = Approve(staged.Dir, "alice@example.com")
	require.ErrorIs(t, err, ErrChecksumMismatch)
	assertNoArchive(t, outDir)
}

func TestApproveProducesAVerifiedArchive(t *testing.T) {
	staged, outDir, _ := stageFixture(t)

	bundle, err := Approve(staged.Dir, "alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(staged.Dir+".tar.gz"), bundle.ArchivePath)
	require.FileExists(t, bundle.ArchivePath)
	require.FileExists(t, bundle.ChecksumPath)

	archive, err := os.ReadFile(bundle.ArchivePath)
	require.NoError(t, err)
	sum := sha256.Sum256(archive)
	assert.Equal(t, hex.EncodeToString(sum[:]), bundle.SHA256)
	sidecar := readFile(t, bundle.ChecksumPath)
	assert.Equal(t, bundle.SHA256+"  "+filepath.Base(bundle.ArchivePath)+"\n", sidecar)

	names, contents := readArchive(t, bundle.ArchivePath)
	assert.ElementsMatch(t, []string{
		manifestFile, reportFile, reviewFile, configFile, logFile, metaFile,
	}, names)
	for name, content := range contents {
		for _, leak := range []string{"hunter2", "hunter2-db", "web-signing-secret", "backup-passphrase", "Budi Santoso", "203.0.113.77"} {
			assert.NotContains(t, content, leak, "%s in %s must be redacted", leak, name)
		}
	}
	assert.Contains(t, contents[reviewFile], `"status": "approved"`)
	assert.Contains(t, contents[reviewFile], `"reviewer": "alice@example.com"`)
	assert.Contains(t, contents[manifestFile], `"status": "approved"`)

	_, err = Approve(staged.Dir, "bob@example.com")
	require.ErrorIs(t, err, ErrAlreadyApproved)

	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	var archives int
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".tar.gz") {
			archives++
		}
	}
	assert.Equal(t, 1, archives)
}

func TestApproveRejectsAForeignDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := Approve(dir, "alice")
	require.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "manifest"))
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path) //nolint:gosec // G304: fixed paths inside the test's temp directory.
	require.NoError(t, err)
	return string(raw)
}

func readManifestFile(t *testing.T, dir string) Manifest {
	t.Helper()
	var manifest Manifest
	require.NoError(t, json.Unmarshal([]byte(readFile(t, filepath.Join(dir, manifestFile))), &manifest))
	return manifest
}

func readReport(t *testing.T, dir string) RedactionReport {
	t.Helper()
	var report RedactionReport
	require.NoError(t, json.Unmarshal([]byte(readFile(t, filepath.Join(dir, reportFile))), &report))
	return report
}

func assertNoArchive(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tar.gz"), "no archive may exist before approval")
	}
}

func readArchive(t *testing.T, path string) ([]string, map[string]string) {
	t.Helper()
	file, err := os.Open(path) //nolint:gosec // G304: archive written by the test into its own temp directory.
	require.NoError(t, err)
	defer func() { _ = file.Close() }()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer func() { _ = reader.Close() }()
	tarReader := tar.NewReader(reader)
	names := make([]string, 0, 6)
	contents := make(map[string]string, 6)
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		body, err := io.ReadAll(tarReader)
		require.NoError(t, err)
		names = append(names, header.Name)
		contents[header.Name] = string(body)
	}
	return names, contents
}
