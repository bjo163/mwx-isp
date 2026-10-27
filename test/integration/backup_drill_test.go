//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/bjo163/mwx-isp/internal/adminapi"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
)

// postRestore uploads a backup archive to the real restore endpoint, optionally
// appending a query string (for ?dry_run=true) and a dry_run form field.
func postRestore(t *testing.T, c *apiClient, payload []byte, query string, dryRunField bool) (int, []byte) {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if dryRunField {
		require.NoError(t, writer.WriteField("dry_run", "1"))
	}
	part, err := writer.CreateFormFile("upload", "backup.json")
	require.NoError(t, err)
	_, err = part.Write(payload)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest(http.MethodPost, c.base+"/api/v1/system/restore"+query, &buf)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.http.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	data, err := readAll(resp.Body)
	require.NoError(t, err)
	return resp.StatusCode, data
}

func readAll(r interface{ Read([]byte) (int, error) }) ([]byte, error) {
	var out bytes.Buffer
	_, err := out.ReadFrom(r)
	return out.Bytes(), err
}

func countWhere(t *testing.T, db *gorm.DB, model any, query string, args ...any) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(model).Where(query, args...).Count(&count).Error)
	return count
}

// TestBackupRestoreDrillEncryptedPostgres is the recovery drill for AUDIT-06 on
// an isolated PostgreSQL database. It proves an encrypted, manifest-carrying
// backup restores tenant ownership, RADIUS/NAS, voucher credentials, billing
// relationships, notification settings and certificate keys; that tampered,
// wrong-key and dry-run requests leave the database untouched; and that active
// online sessions are intentionally not carried across a restore.
func TestBackupRestoreDrillEncryptedPostgres(t *testing.T) {
	c := newAPIClient(t)
	suffix := uniqueSuffix()
	db := h.appCtx.DB()
	drillStarted := time.Now()
	var pgVersion string
	require.NoError(t, db.Raw("SELECT version()").Scan(&pgVersion).Error)

	originalPassphrase := h.appCtx.Config().Backup.Passphrase
	h.appCtx.Config().Backup.Passphrase = "drill-passphrase-" + suffix
	defer func() { h.appCtx.Config().Backup.Passphrase = originalPassphrase }()

	tenant := domain.Tenant{Name: "Drill Tenant " + suffix, Slug: "drill-" + suffix, Kind: "isp", Status: "active"}
	require.NoError(t, db.Create(&tenant).Error)
	tenantDB := db.WithContext(tenancy.WithTenantID(t.Context(), tenant.ID))

	nas := domain.NetNas{
		ID: common.UUIDint64(), Name: "drill-nas-" + suffix, Ipaddr: "198.51.100.77",
		Secret: "nas-secret-" + suffix, CoaPort: 3799, Status: common.ENABLED,
	}
	require.NoError(t, tenantDB.Create(&nas).Error)

	profile := domain.RadiusProfile{ID: common.UUIDint64(), Name: "drill-profile-" + suffix, Status: common.ENABLED}
	require.NoError(t, tenantDB.Create(&profile).Error)
	user := domain.RadiusUser{
		ID: common.UUIDint64(), ProfileId: profile.ID, Username: "drill-user-" + suffix,
		Password: "drill-user-password-" + suffix, Status: common.ENABLED,
		ExpireTime: time.Now().AddDate(1, 0, 0),
	}
	require.NoError(t, tenantDB.Create(&user).Error)

	customer := domain.Customer{ID: common.UUIDint64(), CustomerNo: "C-DRILL-" + suffix, Name: "Drill Customer", Status: domain.CustomerActive}
	require.NoError(t, tenantDB.Create(&customer).Error)
	pkg := domain.InternetPackage{
		ID: common.UUIDint64(), Code: "PKG-DRILL-" + suffix, Name: "Drill Package",
		Price: 150000, RadiusProfileID: profile.ID, BillingCycle: "monthly", Status: "active",
	}
	require.NoError(t, tenantDB.Create(&pkg).Error)
	subscription := domain.Subscription{
		SubscriptionNo: "SUB-DRILL-" + suffix, CustomerID: customer.ID, PackageID: pkg.ID,
		RadiusUserID: user.ID, Status: domain.SubscriptionActive, StartDate: time.Now(),
	}
	require.NoError(t, tenantDB.Create(&subscription).Error)

	batch := domain.HotspotBatch{BatchNo: "BATCH-DRILL-" + suffix, PackageID: pkg.ID, Quantity: 1, CreatedAt: time.Now()}
	require.NoError(t, tenantDB.Create(&batch).Error)
	voucher := domain.HotspotVoucher{
		BatchID: batch.ID, PackageID: pkg.ID, RadiusUserID: user.ID, Code: "DRILL-" + suffix,
		Password: "drill-vp-" + suffix, Status: "active", CreatedAt: time.Now(),
	}
	require.NoError(t, tenantDB.Create(&voucher).Error)

	settings := domain.NotificationSettings{
		ID: common.UUIDint64(), TenantID: tenant.ID, WhatsAppEnabled: true,
		RecipientsJSON: `["ops@drill.test"]`, EventsJSON: `["incident"]`,
	}
	require.NoError(t, tenantDB.Create(&settings).Error)

	cert := domain.SysCert{
		ID: common.UUIDint64(), Name: "drill-cert-" + suffix, CertType: "server",
		Cert:       "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----",
		PrivateKey: "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----",
	}
	require.NoError(t, db.Create(&cert).Error)

	online := domain.RadiusOnline{
		ID: common.UUIDint64(), TenantID: tenant.ID, Username: user.Username,
		NasAddr: nas.Ipaddr, AcctSessionId: "drill-session-" + suffix, AcctStartTime: time.Now(),
	}
	require.NoError(t, tenantDB.Create(&online).Error)

	// ---- 1) Encrypted export carries a manifest and no record contents.
	exportStarted := time.Now()
	status, backupBytes := c.get(t, "/api/v1/system/backup")
	exportDuration := time.Since(exportStarted)
	require.Equalf(t, http.StatusOK, status, "backup body: %s", string(backupBytes))
	for _, secret := range []string{user.Username, voucher.Password, cert.PrivateKey, "BEGIN PRIVATE KEY"} {
		require.NotContains(t, string(backupBytes), secret, "encrypted backups must not leak record contents")
	}

	var envelope struct {
		Format   string `json:"format"`
		Manifest struct {
			SchemaVersion string         `json:"schema_version"`
			Encrypted     bool           `json:"encrypted"`
			RecordCounts  map[string]int `json:"record_counts"`
			PayloadSHA256 string         `json:"payload_sha256"`
			KDF           string         `json:"kdf"`
		} `json:"manifest"`
		Payload    json.RawMessage `json:"payload"`
		Ciphertext string          `json:"ciphertext"`
	}
	require.NoErrorf(t, json.Unmarshal(backupBytes, &envelope), "backup not JSON: %s", string(backupBytes))
	require.Equal(t, "mwx-isp-backup", envelope.Format)
	require.True(t, envelope.Manifest.Encrypted)
	require.Empty(t, envelope.Payload)
	require.NotEmpty(t, envelope.Ciphertext)
	require.NotEmpty(t, envelope.Manifest.PayloadSHA256)
	require.Equal(t, "pbkdf2-sha256", envelope.Manifest.KDF)
	require.GreaterOrEqual(t, envelope.Manifest.RecordCounts["radius_user"], 1)
	require.GreaterOrEqual(t, envelope.Manifest.RecordCounts["net_nas"], 1)
	require.GreaterOrEqual(t, envelope.Manifest.RecordCounts["isp_hotspot_voucher"], 1)
	require.GreaterOrEqual(t, envelope.Manifest.RecordCounts["notification_settings"], 1)
	require.GreaterOrEqual(t, envelope.Manifest.RecordCounts["certs"], 1)
	require.NotContains(t, envelope.Manifest.RecordCounts, "radius_online",
		"transient online sessions must not be part of a backup")
	t.Logf("H06_DRILL export go=%s postgres=%q db=%s schema=%s encrypted=%v bytes=%d payload_sha256=%s tenants=%d users=%d vouchers=%d certs=%d export_ms=%d",
		runtime.Version(), pgVersion, h.dbName, envelope.Manifest.SchemaVersion,
		envelope.Manifest.Encrypted, len(backupBytes), envelope.Manifest.PayloadSHA256,
		envelope.Manifest.RecordCounts["tenants"], envelope.Manifest.RecordCounts["radius_user"],
		envelope.Manifest.RecordCounts["isp_hotspot_voucher"], envelope.Manifest.RecordCounts["certs"],
		exportDuration.Milliseconds())

	// ---- 2) Simulate data loss for the drill tenant.
	for _, row := range []struct {
		model any
		id    int64
	}{
		{&domain.HotspotVoucher{}, voucher.ID}, {&domain.HotspotBatch{}, batch.ID},
		{&domain.Subscription{}, subscription.ID}, {&domain.Customer{}, customer.ID},
		{&domain.InternetPackage{}, pkg.ID}, {&domain.NotificationSettings{}, settings.ID},
		{&domain.RadiusOnline{}, online.ID},
	} {
		require.NoError(t, tenantDB.Delete(row.model, row.id).Error)
	}
	for _, row := range []struct {
		model any
		id    int64
	}{
		{&domain.RadiusUser{}, user.ID}, {&domain.RadiusProfile{}, profile.ID},
		{&domain.NetNas{}, nas.ID}, {&domain.SysCert{}, cert.ID},
	} {
		require.NoError(t, db.Delete(row.model, row.id).Error)
	}
	require.NoError(t, db.Delete(&domain.Tenant{}, tenant.ID).Error)
	require.Zero(t, countWhere(t, db, &domain.Tenant{}, "id = ?", tenant.ID))

	// ---- 3) Dry-run reports the payload without mutating the database.
	status, dryBody := postRestore(t, c, backupBytes, "?dry_run=true", false)
	require.Equalf(t, http.StatusOK, status, "dry-run body: %s", string(dryBody))
	var dryRun struct {
		DryRun       bool                         `json:"dry_run"`
		Encrypted    bool                         `json:"encrypted"`
		WouldRestore adminapi.SystemRestoreResult `json:"would_restore"`
		Manifest     struct {
			RecordCounts map[string]int `json:"record_counts"`
		} `json:"manifest"`
	}
	unwrapData(t, dryBody, &dryRun)
	require.True(t, dryRun.DryRun)
	require.True(t, dryRun.Encrypted)
	require.GreaterOrEqual(t, dryRun.WouldRestore.Users, 1)
	require.Equal(t, envelope.Manifest.RecordCounts, dryRun.Manifest.RecordCounts)
	require.Zero(t, countWhere(t, db, &domain.Tenant{}, "id = ?", tenant.ID), "dry-run must not insert rows")
	var lostUser domain.RadiusUser
	require.ErrorIs(t, db.Where("username = ?", user.Username).First(&lostUser).Error, gorm.ErrRecordNotFound)

	// ---- 4) A wrong key is rejected without writes.
	h.appCtx.Config().Backup.Passphrase = "wrong-passphrase-" + suffix
	status, wrongKeyBody := postRestore(t, c, backupBytes, "", false)
	require.Equalf(t, http.StatusBadRequest, status, "wrong-key body: %s", string(wrongKeyBody))
	require.Contains(t, string(wrongKeyBody), "BACKUP_DECRYPT_FAILED")
	require.Zero(t, countWhere(t, db, &domain.Tenant{}, "id = ?", tenant.ID))
	h.appCtx.Config().Backup.Passphrase = "drill-passphrase-" + suffix

	// ---- 5) Restore with the matching key and verify relationships + secrets.
	restoreStarted := time.Now()
	status, restoreBody := postRestore(t, c, backupBytes, "", false)
	restoreDuration := time.Since(restoreStarted)
	require.Equalf(t, http.StatusOK, status, "restore body: %s", string(restoreBody))
	refreshSharedAdminToken(t)

	var restoreResult adminapi.SystemRestoreResult
	unwrapData(t, restoreBody, &restoreResult)

	var restoredTenant domain.Tenant
	require.NoError(t, db.First(&restoredTenant, tenant.ID).Error)
	require.Equal(t, tenant.Slug, restoredTenant.Slug)

	var restoredUser domain.RadiusUser
	require.NoError(t, tenantDB.Where("id = ?", user.ID).First(&restoredUser).Error)
	require.Equal(t, user.Password, restoredUser.Password)
	require.Equal(t, profile.ID, restoredUser.ProfileId)

	var restoredNas domain.NetNas
	require.NoError(t, tenantDB.First(&restoredNas, nas.ID).Error)
	require.Equal(t, nas.Secret, restoredNas.Secret)

	var restoredVoucher domain.HotspotVoucher
	require.NoError(t, tenantDB.First(&restoredVoucher, voucher.ID).Error)
	require.Equal(t, voucher.Password, restoredVoucher.Password)
	require.Equal(t, user.ID, restoredVoucher.RadiusUserID)
	require.Equal(t, batch.ID, restoredVoucher.BatchID)

	var restoredSubscription domain.Subscription
	require.NoError(t, tenantDB.First(&restoredSubscription, subscription.ID).Error)
	require.Equal(t, customer.ID, restoredSubscription.CustomerID)
	require.Equal(t, pkg.ID, restoredSubscription.PackageID)
	require.Equal(t, user.ID, restoredSubscription.RadiusUserID)

	var restoredSettings domain.NotificationSettings
	require.NoError(t, tenantDB.First(&restoredSettings, settings.ID).Error)
	require.True(t, restoredSettings.WhatsAppEnabled)
	require.Equal(t, settings.RecipientsJSON, restoredSettings.RecipientsJSON)

	var restoredCert domain.SysCert
	require.NoError(t, db.First(&restoredCert, cert.ID).Error)
	require.Equal(t, cert.PrivateKey, restoredCert.PrivateKey)

	require.Zero(t, countWhere(t, tenantDB, &domain.RadiusOnline{}, "acct_session_id = ?", online.AcctSessionId),
		"online sessions are transient and must be regenerated by the RADIUS service, not restored")
	t.Logf("H06_DRILL restore applied users=%d nas=%d certs=%d customers=%d subscriptions=%d vouchers=%d settings=%d restore_ms=%d total_ms=%d expected_loss=%q",
		restoreResult.Users, restoreResult.Nas, restoreResult.Certs, restoreResult.Customers,
		restoreResult.Subscriptions, restoreResult.Vouchers, restoreResult.NotificationSettings,
		restoreDuration.Milliseconds(), time.Since(drillStarted).Milliseconds(),
		"radius_online transient rows and in-memory runtime state; destination-only rows are preserved")
}

// TestBackupRestoreRejectsCrossTenantAndRollsBackPartialFailure proves that a
// cross-tenant reference and a mid-transaction database failure both leave the
// existing database unchanged.
func TestBackupRestoreRejectsCrossTenantAndRollsBackPartialFailure(t *testing.T) {
	c := newAPIClient(t)
	suffix := uniqueSuffix()
	db := h.appCtx.DB()

	// ---- Cross-tenant reference: a subscription in tenant B points at a
	// package owned by tenant A. Validation must reject it before any write.
	otherTenantID := common.UUIDint64()
	customerID, packageID, subscriptionID := common.UUIDint64(), common.UUIDint64(), common.UUIDint64()
	cross := adminapi.SystemBackup{
		Version:   "9.3",
		CreatedAt: time.Now().UTC(),
		Tenants: []domain.Tenant{
			{ID: domain.DefaultTenantID, Name: "Default ISP", Slug: "default"},
			{ID: otherTenantID, Name: "Cross Tenant " + suffix, Slug: "cross-" + suffix},
		},
		Customers: []domain.Customer{{ID: customerID, TenantID: otherTenantID, CustomerNo: "C-CROSS-" + suffix, Name: "Cross customer"}},
		Packages:  []domain.InternetPackage{{ID: packageID, TenantID: domain.DefaultTenantID, Code: "PKG-CROSS-" + suffix, Name: "Cross package"}},
		Subscriptions: []domain.Subscription{{
			ID: subscriptionID, TenantID: otherTenantID, CustomerID: customerID,
			PackageID: packageID, Status: domain.SubscriptionPending, StartDate: time.Now(),
		}},
		TenantIDs: map[string]map[string]int64{
			"isp_customer":     {strconv.FormatInt(customerID, 10): otherTenantID},
			"isp_package":      {strconv.FormatInt(packageID, 10): domain.DefaultTenantID},
			"isp_subscription": {strconv.FormatInt(subscriptionID, 10): otherTenantID},
		},
	}
	payload, err := json.Marshal(&cross)
	require.NoError(t, err)

	status, body := postRestore(t, c, payload, "", false)
	require.Equalf(t, http.StatusBadRequest, status, "cross-tenant body: %s", string(body))
	require.Contains(t, string(body), "another tenant")
	require.Zero(t, countWhere(t, db, &domain.Tenant{}, "id = ?", otherTenantID))
	require.Zero(t, countWhere(t, db, &domain.Subscription{}, "id = ?", subscriptionID))

	// ---- Partial failure: the transaction inserts the new tenant, then hits a
	// unique-constraint violation on notification_settings and must roll back.
	var existingSettings domain.NotificationSettings
	err = db.Where("tenant_id = ?", domain.DefaultTenantID).First(&existingSettings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		existingSettings = domain.NotificationSettings{ID: common.UUIDint64(), TenantID: domain.DefaultTenantID, UpdatedAt: time.Now()}
		require.NoError(t, db.Create(&existingSettings).Error)
	} else {
		require.NoError(t, err)
	}

	rollbackTenantID := common.UUIDint64()
	conflictingSettingsID := common.UUIDint64()
	rollback := adminapi.SystemBackup{
		Version:   "9.3",
		CreatedAt: time.Now().UTC(),
		Tenants: []domain.Tenant{
			{ID: domain.DefaultTenantID, Name: "Default ISP", Slug: "default"},
			{ID: rollbackTenantID, Name: "Rollback Tenant " + suffix, Slug: "rollback-" + suffix},
		},
		NotificationSettings: []domain.NotificationSettings{{
			ID: conflictingSettingsID, TenantID: domain.DefaultTenantID, UpdatedAt: time.Now(),
		}},
		TenantIDs: map[string]map[string]int64{
			"notification_settings": {strconv.FormatInt(conflictingSettingsID, 10): domain.DefaultTenantID},
		},
	}
	payload, err = json.Marshal(&rollback)
	require.NoError(t, err)

	status, body = postRestore(t, c, payload, "", false)
	require.Equalf(t, http.StatusInternalServerError, status, "rollback body: %s", string(body))
	require.Contains(t, string(body), "RESTORE_ERROR")
	require.Zero(t, countWhere(t, db, &domain.Tenant{}, "id = ?", rollbackTenantID),
		"a failed restore must roll back rows written earlier in the same transaction")
	require.Zero(t, countWhere(t, db, &domain.NotificationSettings{}, "id = ?", conflictingSettingsID))
}
