package teapreadiness

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripCommentsRemovesCommentsAndKeepsLiterals(t *testing.T) {
	cases := []struct {
		name    string
		src     string
		removed []string
		kept    []string
	}{
		{
			name:    "line comment",
			src:     "const TypeTEAP = 55 // TEAP is type 55\nconst TypeTLS = 13\n",
			removed: []string{"is type 55"},
			kept:    []string{"const TypeTEAP = 55", "const TypeTLS = 13"},
		},
		{
			name:    "block comment across lines",
			src:     "/* TEAP needs\n   a Crypto-Binding TLV */\nconst x = 1\n",
			removed: []string{"Crypto-Binding"},
			kept:    []string{"const x = 1"},
		},
		{
			name:    "slashes inside a string are not comments",
			src:     "const s = \"https://example.test/teap\"\n",
			removed: nil,
			kept:    []string{"https://example.test/teap"},
		},
		{
			name:    "raw string keeps its contents",
			src:     "const raw = `// not a comment: teap`\n",
			removed: nil,
			kept:    []string{"// not a comment: teap"},
		},
		{
			name:    "doc comment disappears",
			src:     "// NewTEAPHandler builds a handler.\nfunc NewTEAPHandler() {}\n",
			removed: []string{"builds a handler"},
			kept:    []string{"func NewTEAPHandler()"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripComments(tc.src)
			for _, want := range tc.removed {
				assert.NotContains(t, got, want)
			}
			for _, want := range tc.kept {
				assert.Contains(t, got, want)
			}
			assert.Equal(t, strings.Count(tc.src, "\n"), strings.Count(got, "\n"),
				"line numbers must survive stripping")
		})
	}
}

func TestScanReadsOnlyProductCode(t *testing.T) {
	root := t.TempDir()
	write := func(rel, content string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o750))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o600))
	}
	write("internal/radiusd/plugins/eap/interfaces.go", "// TypeTEAP is unused here.\nconst TypeTLS = 13\nconst TypeTLV = 33\n")
	write("internal/radiusd/plugins/eap/interfaces_test.go", "const TypeTEAP = 55\n")
	write("internal/radiusd/plugins/eap/testdata/fixture.go", "const TypeTEAP = 55\n")
	write("internal/radiusd/plugins/eap/handlers/peap_inner.go", "const tlvTypeResult = 3\n")
	write("internal/radiusd/auth_stages.go", "// nothing\n")
	write("internal/app/config_manager.go", "Enum: []string{\"eap-tls\"}\n")
	write("docs/teap-notes.md", "const TypeTEAP = 55\n")

	tree, err := Scan(root)
	require.NoError(t, err)

	got := tree.paths()
	assert.Equal(t, []string{
		"internal/app/config_manager.go",
		"internal/radiusd/auth_stages.go",
		"internal/radiusd/plugins/eap/handlers/peap_inner.go",
		"internal/radiusd/plugins/eap/interfaces.go",
	}, got)

	// A test file, a testdata fixture and a markdown note all spell the
	// identifier; none of them may make the mechanism look implemented.
	assert.Equal(t, StatusMissing, find(t, Analyze(tree), "teap-eap-type").Status)
	assert.Equal(t, StatusReady, find(t, Analyze(tree), "result-tlv").Status)
}

func TestScanFailsWhenADirectoryIsMissing(t *testing.T) {
	_, err := Scan(t.TempDir())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scan directory")
}

func TestScanReadsTheRealProductTree(t *testing.T) {
	tree, err := Scan(repoRoot(t))
	require.NoError(t, err)
	for _, rel := range []string{
		"internal/radiusd/plugins/eap/interfaces.go",
		"internal/radiusd/plugins/eap/handlers/tls_config.go",
		"internal/radiusd/auth_stages.go",
		"internal/app/config_manager.go",
	} {
		require.Contains(t, tree.Files, rel)
		assert.NotEmpty(t, tree.Files[rel], rel)
	}
	for path := range tree.Files {
		assert.NotContains(t, path, "internal/teapreadiness", "the prototype must not scan itself")
		assert.NotContains(t, path, "_test.go", path)
	}
}
