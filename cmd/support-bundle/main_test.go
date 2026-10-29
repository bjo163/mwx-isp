package main

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fixtureConfig = `database:
  host: db.internal
  passwd: hunter2-db
web:
  secret: web-signing-secret
`

const fixtureLog = `{"level":"info","username":"alice","password":"hunter2","customer_name":"Budi Santoso","remote_ip":"203.0.113.77"}
`

func runCLI(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func writeInputs(t *testing.T) (string, string, string) {
	t.Helper()
	work := t.TempDir()
	configPath := filepath.Join(work, "mwx-isp.yml")
	logPath := filepath.Join(work, "app.log")
	require.NoError(t, os.WriteFile(configPath, []byte(fixtureConfig), 0o600))
	require.NoError(t, os.WriteFile(logPath, []byte(fixtureLog), 0o600))
	return work, configPath, logPath
}

func stagedDirFrom(t *testing.T, stdout string) string {
	t.Helper()
	match := regexp.MustCompile(`Staged support bundle: (.+)`).FindStringSubmatch(stdout)
	require.Len(t, match, 2, stdout)
	return strings.TrimSpace(match[1])
}

func TestRunStagesThenApproves(t *testing.T) {
	work, configPath, logPath := writeInputs(t)
	outDir := filepath.Join(work, "out")

	code, stdout, stderr := runCLI(t, "-c", configPath, "-log", logPath, "-out", outDir, "-version", "v0.15.0-test")
	require.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "No archive was created")
	assert.Contains(t, stdout, "Review before export")
	assertNotContains(t, stdout, "hunter2")

	dir := stagedDirFrom(t, stdout)
	require.FileExists(t, filepath.Join(dir, "manifest.json"))
	assertNoArchives(t, outDir)

	code, stdout, stderr = runCLI(t, "-approve", dir, "-reviewer", "alice@example.com")
	require.Equal(t, 0, code, stderr)
	assert.Contains(t, stdout, "Approved by alice@example.com")
	assert.Contains(t, stdout, "SHA256:")
	archive := strings.TrimSpace(strings.TrimPrefix(strings.Split(stdout, "\n")[1], "Archive:"))
	require.FileExists(t, archive)
	require.FileExists(t, archive+".sha256")

	code, _, stderr = runCLI(t, "-approve", dir, "-reviewer", "bob@example.com")
	require.Equal(t, 1, code)
	assert.Contains(t, stderr, "already approved")
}

func TestRunRejectsUsageErrors(t *testing.T) {
	work, configPath, logPath := writeInputs(t)
	outDir := filepath.Join(work, "out")

	cases := []struct {
		name string
		args []string
		want string
	}{
		{name: "no input", args: []string{"-out", outDir}, want: "provide -c"},
		{name: "no output", args: []string{"-c", configPath}, want: "-out is required"},
		{name: "approve without reviewer", args: []string{"-approve", outDir}, want: "reviewer"},
		{name: "mixed modes", args: []string{"-approve", outDir, "-c", configPath}, want: "cannot be combined"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, stderr := runCLI(t, tc.args...)
			assert.Equal(t, 2, code)
			assert.Contains(t, strings.ToLower(stderr), strings.ToLower(tc.want))
		})
	}

	code, _, stderr := runCLI(t, "-c", filepath.Join(work, "missing.yml"), "-log", logPath, "-out", outDir)
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "read config")
}

func TestRunRefusesToApproveTamperedBundle(t *testing.T) {
	work, configPath, logPath := writeInputs(t)
	outDir := filepath.Join(work, "out")

	code, stdout, stderr := runCLI(t, "-c", configPath, "-log", logPath, "-out", outDir)
	require.Equal(t, 0, code, stderr)
	dir := stagedDirFrom(t, stdout)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "logs", "app.log"), []byte("edited after review\n"), 0o600))
	code, _, stderr = runCLI(t, "-approve", dir, "-reviewer", "alice@example.com")
	assert.Equal(t, 1, code)
	assert.Contains(t, stderr, "does not match its manifest")
	assertNoArchives(t, outDir)
}

func assertNoArchives(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasSuffix(entry.Name(), ".tar.gz"), entry.Name())
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	assert.NotContains(t, haystack, needle)
}
