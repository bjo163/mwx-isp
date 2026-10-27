package adminapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	webresp "github.com/bjo163/mwx-isp/pkg/web"
)

// systemLogSensitiveFixture is a zap-style line carrying every data class the
// export path must handle: a credential, a bearer token, a customer name, a
// phone number, an e-mail address and a host IP. The username stays because it
// is the operational identifier the audit trail is keyed on.
const systemLogSensitiveFixture = `{"level":"info","ts":1757396220,"msg":"login attempt for alice@example.com from 203.0.113.77 password=hunter2","namespace":"isp","job":"toughradius","username":"alice","password":"hunter2","authorization":"Bearer eyJhbGciOiJIUzI1NiJ9.payload.sig","customer_name":"Budi Santoso","phone":"+6281234567890","email":"alice@example.com","remote_ip":"203.0.113.77"}` + "\n"

func TestRedactSystemLogEntryRemovesSensitiveDataClasses(t *testing.T) {
	entry, ok := parseSystemLogLine(strings.TrimSpace(systemLogSensitiveFixture))
	require.True(t, ok)

	redacted := redactSystemLogEntry(entry)

	exported := redacted.Line + " " + redacted.Message
	for _, leak := range []string{
		"hunter2",
		"eyJhbGciOiJIUzI1NiJ9",
		"Budi Santoso",
		"+6281234567890",
		"alice@example.com",
		"203.0.113.77",
	} {
		assert.NotContains(t, exported, leak, "%q must not survive the export path", leak)
	}

	assert.Equal(t, "alice", redacted.Fields["username"], "the operational identifier stays searchable")
	assert.Equal(t, systemLogRedactedMarker, redacted.Fields["password"])
	assert.Equal(t, systemLogRedactedMarker, redacted.Fields["authorization"])
	assert.Equal(t, "203.0.113.x", redacted.Fields["remote_ip"])
}

func TestRedactSystemLogEntryPseudonymsAreStable(t *testing.T) {
	first, ok := parseSystemLogLine(strings.TrimSpace(systemLogSensitiveFixture))
	require.True(t, ok)
	second, ok := parseSystemLogLine(`{"level":"info","ts":1757396221,"customer_name":"Budi Santoso","phone":"+6281234567890"}`)
	require.True(t, ok)

	firstRedacted := redactSystemLogEntry(first)
	secondRedacted := redactSystemLogEntry(second)

	assert.Equal(t, firstRedacted.Fields["customer_name"], secondRedacted.Fields["customer_name"],
		"the same customer must pseudonymize to the same value for correlation")
	assert.Equal(t, firstRedacted.Fields["phone"], secondRedacted.Fields["phone"])
	assert.True(t, strings.HasPrefix(firstRedacted.Fields["customer_name"].(string), "sha256:"))
}

func TestMaskSystemLogAddressKeepsOnlyTheNetworkPrefix(t *testing.T) {
	tests := map[string]string{
		"203.0.113.77":          "203.0.113.x",
		"2001:db8:1234:5678::1": "2001:db8:1234::x",
		"not-an-address":        "not-an-address",
		"":                      "",
	}
	for input, want := range tests {
		assert.Equal(t, want, maskSystemLogAddress(input), input)
	}
}

func TestQuerySystemLogsExportsOnlyRedactedEntries(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	logPath := filepath.Join(t.TempDir(), "toughradius.log")
	require.NoError(t, os.WriteFile(logPath, []byte(systemLogSensitiveFixture), 0600))
	appCtx.Config().Logger.Filename = logPath

	req := httptest.NewRequest(http.MethodGet, "/admin/loki/query", nil)
	q := req.URL.Query()
	q.Set("starttime", "2025-09-08 13:37:00")
	q.Set("endtime", "2025-09-11 13:37:00")
	q.Set("limit", "10")
	req.URL.RawQuery = q.Encode()
	rec := httptest.NewRecorder()

	require.NoError(t, querySystemLogs(CreateTestContext(e, db, req, rec, appCtx)))
	require.Equal(t, http.StatusOK, rec.Code)

	var resp webresp.WebRestResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	body := rec.Body.String()
	for _, leak := range []string{"hunter2", "Budi Santoso", "+6281234567890", "203.0.113.77"} {
		assert.NotContains(t, body, leak, "%q must not reach the operator response", leak)
	}
	assert.Contains(t, body, "203.0.113.x", "the address class must stay correlatable at prefix level")
}
