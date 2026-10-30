package notify

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"go.mau.fi/whatsmeow/proto/waAdv"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const testSessionPhone = "6281234500001"

func newSessionTestManager(t *testing.T) *WhatsAppManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "session.db")), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	// Mirror internal/app/getSqliteDatabase: one shared connection keeps the
	// manager's PRAGMA foreign_keys=ON in force for every statement.
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	manager, err := NewWhatsAppManager(db)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, manager.Close())
		require.NoError(t, sqlDB.Close())
	})
	return manager
}

// linkSession seeds the rows a completed QR pairing would leave behind: the
// device (with its noise, identity and signed pre-keys) plus one contact
// identity and session.
func linkSession(t *testing.T, manager *WhatsAppManager) types.JID {
	t.Helper()
	ctx := context.Background()
	device := manager.container.NewDevice()
	jid := types.NewJID(testSessionPhone, types.DefaultUserServer)
	device.ID = &jid
	device.Account = &waAdv.ADVSignedDeviceIdentity{
		Details:             []byte("session-test-details"),
		AccountSignatureKey: make([]byte, 32),
		AccountSignature:    make([]byte, 64),
		DeviceSignature:     make([]byte, 64),
	}
	require.NoError(t, manager.container.PutDevice(ctx, device))
	session := sqlstore.NewSQLStore(manager.container, jid)
	require.NoError(t, session.PutIdentity(ctx, "551199998888", [32]byte{7}))
	require.NoError(t, session.PutSession(ctx, "551199998888", []byte("serialized-session")))
	require.EqualValues(t, 1, countSessionRows(t, manager, "whatsmeow_device"))
	return jid
}

func attachManagerClient(t *testing.T, manager *WhatsAppManager) {
	t.Helper()
	device, err := manager.container.GetFirstDevice(context.Background())
	require.NoError(t, err)
	require.NotNil(t, device.ID)
	manager.mu.Lock()
	manager.client = manager.newClient(device)
	manager.device = device.ID
	manager.state = "connecting"
	manager.mu.Unlock()
}

func countSessionRows(t *testing.T, manager *WhatsAppManager, table string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, manager.db.Table(table).Count(&count).Error)
	return count
}

func requireSessionErased(t *testing.T, manager *WhatsAppManager, jid types.JID) {
	t.Helper()
	for _, table := range []string{"whatsmeow_device", "whatsmeow_identity_keys", "whatsmeow_sessions"} {
		require.Zero(t, countSessionRows(t, manager, table), "%s still holds rows for %s", table, jid)
	}
	status := manager.Status()
	require.Equal(t, "not_linked", status.State)
	require.False(t, status.Connected)
	require.False(t, status.LoggedIn)
	require.Empty(t, status.Account)
}

func TestWhatsAppManagerEnforcesForeignKeysOnSQLite(t *testing.T) {
	manager := newSessionTestManager(t)
	var enabled int
	require.NoError(t, manager.db.Raw("PRAGMA foreign_keys").Scan(&enabled).Error)
	require.Equal(t, 1, enabled, "deleting the device only cascades while foreign keys are enforced")
}

func TestDisconnectRevokesPersistedWhatsAppSession(t *testing.T) {
	manager := newSessionTestManager(t)
	jid := linkSession(t, manager)
	attachManagerClient(t, manager)

	require.NoError(t, manager.Disconnect(context.Background()))

	requireSessionErased(t, manager, jid)
}

func TestUnlinkRevokesPersistedWhatsAppSession(t *testing.T) {
	manager := newSessionTestManager(t)
	jid := linkSession(t, manager)
	attachManagerClient(t, manager)

	require.NoError(t, manager.Unlink(context.Background()))

	requireSessionErased(t, manager, jid)
}

func TestWhatsAppStatusDoesNotExposeSessionMaterial(t *testing.T) {
	manager := newSessionTestManager(t)
	linkSession(t, manager)
	attachManagerClient(t, manager)

	payload, err := json.Marshal(manager.Status())
	require.NoError(t, err)

	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(payload, &fields))
	for key := range fields {
		require.Contains(t,
			[]string{"connected", "logged_in", "state", "account", "qr_image", "qr_expires"},
			key, "unexpected status field %q", key)
	}
	for _, secret := range []string{"serialized-session", "session-test-details", "noise", "adv_secret", "identity_key"} {
		require.NotContains(t, string(payload), secret)
	}
}

// TestWhatsAppTransportOnlyUsesSilentLogger pins the property that the
// transport cannot write session material to a log: every whatsmeow logger it
// builds is waLog.Noop and the package has no other logging sink.
func TestWhatsAppTransportOnlyUsesSilentLogger(t *testing.T) {
	entries, err := os.ReadDir(".")
	require.NoError(t, err)
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		source, err := os.ReadFile(name) //nolint:gosec // G304: name comes from this package's own directory listing.
		require.NoError(t, err)
		text := string(source)
		scanned++
		for _, sink := range []string{"zap.L()", "slog.", "log.Printf", "log.Print(", "log.Println", "waLog.New", "waLog.Stdout"} {
			require.NotContains(t, text, sink, "%s must not send WhatsApp traffic to a log", name)
		}
		for _, line := range strings.Split(text, "\n") {
			index := strings.Index(line, "waLog.")
			if index < 0 {
				continue
			}
			require.Contains(t, line[index:], "waLog.Noop", "only waLog.Noop is allowed in %s", name)
		}
	}
	require.NotZero(t, scanned)
}
