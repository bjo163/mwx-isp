package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/config"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/securestore"
)

const (
	testOldSecret = "old-web-secret-0123456789abcdef" //nolint:gosec // G101: fixture value, not a deployed credential.
	testNewSecret = "new-web-secret-0123456789abcdef" //nolint:gosec // G101: fixture value, not a deployed credential.
	testCommunity = "community-s3cret-value"
)

func rotationEnv(key string) string {
	switch key {
	case oldSecretEnvVar:
		return testOldSecret
	case newSecretEnvVar:
		return testNewSecret
	}
	return ""
}

func writeRotationConfig(t *testing.T, dir string) string {
	t.Helper()
	configPath := filepath.Join(dir, "mwx-isp.yml")
	body := fmt.Sprintf("system:\n  workdir: %q\ndatabase:\n  type: sqlite\n  name: %q\n",
		filepath.ToSlash(dir), "rotation.db")
	require.NoError(t, os.WriteFile(configPath, []byte(body), 0o600))
	return configPath
}

func openRotationDB(t *testing.T, configPath string) *gorm.DB {
	t.Helper()
	db, err := openDatabase(config.LoadConfig(configPath))
	require.NoError(t, err)
	return db
}

func closeRotationDB(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
}

func seedRotationTarget(t *testing.T, configPath string) {
	t.Helper()
	db := openRotationDB(t, configPath)
	defer closeRotationDB(t, db)
	require.NoError(t, db.AutoMigrate(&domain.NetMonitorTarget{}))

	sealed, err := securestore.Seal(testOldSecret, []byte(testCommunity))
	require.NoError(t, err)
	require.NoError(t, db.Create(&domain.NetMonitorTarget{
		Name: "edge-router", Kind: "router", Address: "192.0.2.10", ProbeType: "snmp",
		SNMPVersion: "v2c", SNMPCommunityEncrypted: sealed,
		IntervalSeconds: 60, TimeoutMilliseconds: 1000, FailureThreshold: 2,
	}).Error)
}

func storedCommunity(t *testing.T, configPath, secret string) error {
	t.Helper()
	db := openRotationDB(t, configPath)
	defer closeRotationDB(t, db)
	var target domain.NetMonitorTarget
	require.NoError(t, db.First(&target).Error)
	plaintext, err := securestore.Open(secret, target.SNMPCommunityEncrypted)
	if err != nil {
		return err
	}
	require.Equal(t, testCommunity, string(plaintext))
	return nil
}

func TestRunDryRunReportsCountsAndWritesNothing(t *testing.T) {
	configPath := writeRotationConfig(t, t.TempDir())
	seedRotationTarget(t, configPath)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-c", configPath}, rotationEnv, &stdout, &stderr)

	require.Equal(t, 0, code, stderr.String())
	assert.Contains(t, stdout.String(), "apply=false targets=1 fields=1")
	for _, leaked := range []string{testCommunity, testOldSecret, testNewSecret} {
		assert.NotContains(t, stdout.String(), leaked)
		assert.NotContains(t, stderr.String(), leaked)
	}

	require.NoError(t, storedCommunity(t, configPath, testOldSecret),
		"the dry run must leave the stored ciphertext readable with the old secret")
	assert.Error(t, storedCommunity(t, configPath, testNewSecret))
}

func TestRunApplyReencryptsCredentials(t *testing.T) {
	configPath := writeRotationConfig(t, t.TempDir())
	seedRotationTarget(t, configPath)

	var stdout, stderr bytes.Buffer
	code := run([]string{"-c", configPath, "-apply"}, rotationEnv, &stdout, &stderr)

	require.Equal(t, 0, code, stderr.String())
	assert.Contains(t, stdout.String(), "apply=true targets=1 fields=1")
	assert.NotContains(t, stdout.String(), testCommunity)

	require.NoError(t, storedCommunity(t, configPath, testNewSecret))
	assert.Error(t, storedCommunity(t, configPath, testOldSecret))
}

func TestRunRequiresBothSecrets(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, func(string) string { return "" }, &stdout, &stderr)

	assert.Equal(t, 2, code)
	assert.Contains(t, stderr.String(), oldSecretEnvVar)
	assert.Contains(t, stderr.String(), newSecretEnvVar)
}
