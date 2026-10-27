package app

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/config"
	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/sessionqueue"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRunBillingForTenantScopesInvoiceGenerationAndDisconnect(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.InternetPackage{}, &domain.Subscription{}, &domain.Invoice{},
		&domain.InvoiceItem{}, &domain.BillingEvent{}, &domain.RadiusUser{}, &domain.RadiusOnline{},
		&domain.DocumentSequence{}, &domain.RadiusSessionAction{}, &domain.RadiusSessionActionAudit{},
	))

	now := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	for _, tenantID := range []int64{10, 20} {
		ctx := tenancy.WithTenantID(context.Background(), tenantID)
		tenantDB := db.WithContext(ctx)
		customer := domain.Customer{Name: "Same Name", Status: domain.CustomerActive}
		require.NoError(t, tenantDB.Create(&customer).Error)
		pkg := domain.InternetPackage{Code: "HOME", Name: "Home", Price: 150000, RadiusProfileID: 1, BillingCycle: "monthly", Status: "active"}
		require.NoError(t, tenantDB.Create(&pkg).Error)
		user := domain.RadiusUser{Username: "shared-user", Password: "secret", Status: "enabled"}
		require.NoError(t, tenantDB.Create(&user).Error)
		sub := domain.Subscription{SubscriptionNo: "SUB-001", CustomerID: customer.ID, PackageID: pkg.ID,
			RadiusUserID: user.ID, Status: domain.SubscriptionActive, StartDate: now.AddDate(0, -1, 0), BillingDay: 1}
		require.NoError(t, tenantDB.Create(&sub).Error)
		session := domain.RadiusOnline{Username: user.Username, AcctSessionId: fmt.Sprintf("session-%d", tenantID)}
		require.NoError(t, tenantDB.Create(&session).Error)
	}

	var disconnectedTenantIDs []int64
	a := &Application{gormDB: db}
	a.SetSessionActionSender(func(ctx context.Context, session domain.RadiusOnline) sessionqueue.Outcome {
		tenantID, ok := tenancy.TenantID(ctx)
		require.True(t, ok)
		disconnectedTenantIDs = append(disconnectedTenantIDs, tenantID)
		return sessionqueue.Outcome{Success: true, ResponseCode: "Disconnect-ACK", Attempts: 1}
	})
	// Mark tenant 10's subscription overdue so this run suspends and disconnects
	// only tenant 10's duplicate username session.
	tenantA := db.WithContext(tenancy.WithTenantID(context.Background(), 10))
	var sub domain.Subscription
	require.NoError(t, tenantA.First(&sub).Error)
	_, err = billing.GenerateMonthlyInvoices(tenantA, now, 10)
	require.NoError(t, err)
	var invoice domain.Invoice
	require.NoError(t, tenantA.First(&invoice).Error)
	require.NoError(t, tenantA.Model(&invoice).Update("due_date", now.AddDate(0, 0, -5)).Error)

	require.NoError(t, a.runBillingForTenant(10, now, 10, true))
	require.Equal(t, []int64{10}, disconnectedTenantIDs)

	var tenantAInvoices, tenantBInvoices []domain.Invoice
	require.NoError(t, db.WithContext(tenancy.WithTenantID(context.Background(), 10)).Find(&tenantAInvoices).Error)
	require.NoError(t, db.WithContext(tenancy.WithTenantID(context.Background(), 20)).Find(&tenantBInvoices).Error)
	require.Len(t, tenantAInvoices, 1)
	require.Empty(t, tenantBInvoices, "tenant A billing must not generate invoices for tenant B")

	var actions []domain.RadiusSessionAction
	require.NoError(t, db.Find(&actions).Error)
	require.Len(t, actions, 1, "only the suspended tenant's session is queued")
	assert.Equal(t, int64(10), actions[0].TenantID)
	assert.Equal(t, sessionqueue.StatusSucceeded, actions[0].Status)

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Find(&audits).Error)
	require.Len(t, audits, 1)
	assert.Equal(t, int64(10), audits[0].TenantID)
	assert.Equal(t, sessionqueue.SystemActor.Name, audits[0].OperatorName)
}

// TestBillingQueuesFailedDisconnectForRetry guards #52: when the NAS is
// unreachable, billing suspension must leave a durable pending action instead of
// only logging a warning, repeated billing runs must not duplicate it, and the
// retry scheduler must converge it later.
func TestBillingQueuesFailedDisconnectForRetry(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.InternetPackage{}, &domain.Subscription{}, &domain.Invoice{},
		&domain.InvoiceItem{}, &domain.BillingEvent{}, &domain.RadiusUser{}, &domain.RadiusOnline{},
		&domain.DocumentSequence{}, &domain.Tenant{}, &domain.RadiusSessionAction{}, &domain.RadiusSessionActionAudit{},
	))

	now := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Create(&domain.Tenant{ID: 10, Name: "Tenant 10", Slug: "tenant-10", Kind: "isp", Status: "active"}).Error)
	tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), 10))
	customer := domain.Customer{Name: "Overdue", Status: domain.CustomerActive}
	require.NoError(t, tenantDB.Create(&customer).Error)
	pkg := domain.InternetPackage{Code: "HOME", Name: "Home", Price: 150000, RadiusProfileID: 1, BillingCycle: "monthly", Status: "active"}
	require.NoError(t, tenantDB.Create(&pkg).Error)
	user := domain.RadiusUser{Username: "overdue-user", Password: "secret", Status: "enabled"}
	require.NoError(t, tenantDB.Create(&user).Error)
	sub := domain.Subscription{SubscriptionNo: "SUB-001", CustomerID: customer.ID, PackageID: pkg.ID,
		RadiusUserID: user.ID, Status: domain.SubscriptionActive, StartDate: now.AddDate(0, -1, 0), BillingDay: 1}
	require.NoError(t, tenantDB.Create(&sub).Error)
	session := domain.RadiusOnline{Username: user.Username, AcctSessionId: "session-overdue"}
	require.NoError(t, tenantDB.Create(&session).Error)
	_, err = billing.GenerateMonthlyInvoices(tenantDB, now, 10)
	require.NoError(t, err)
	var invoice domain.Invoice
	require.NoError(t, tenantDB.First(&invoice).Error)
	require.NoError(t, tenantDB.Model(&invoice).Update("due_date", now.AddDate(0, 0, -5)).Error)

	failCalls := 0
	a := &Application{gormDB: db}
	a.SetSessionActionSender(func(context.Context, domain.RadiusOnline) sessionqueue.Outcome {
		failCalls++
		return sessionqueue.Outcome{ResponseCode: "Disconnect-NAK", Error: "NAS unreachable", TimedOut: true}
	})
	require.NoError(t, a.runBillingForTenant(10, now, 10, true))
	require.Equal(t, 1, failCalls)

	var action domain.RadiusSessionAction
	require.NoError(t, tenantDB.First(&action).Error)
	assert.Equal(t, sessionqueue.StatusPending, action.Status)
	assert.Equal(t, 1, action.Attempts)
	assert.True(t, action.LastTimedOut)
	assert.Equal(t, now.Add(time.Minute), action.NextRetryAt)

	// A repeated billing run must not create a second action; the due retry is
	// executed in place without resetting the attempt budget.
	require.NoError(t, a.runBillingForTenant(10, now.Add(time.Hour), 10, true))
	var actionCount int64
	require.NoError(t, tenantDB.Model(&domain.RadiusSessionAction{}).Count(&actionCount).Error)
	assert.Equal(t, int64(1), actionCount)
	require.NoError(t, tenantDB.First(&action).Error)
	assert.Equal(t, 2, action.Attempts, "the retry increments the budget instead of resetting it")
	assert.Equal(t, now.Add(time.Hour).Add(sessionqueue.Backoff(2)), action.NextRetryAt)

	// Once the NAS recovers and the backoff elapses, the retry task converges it.
	a.SetSessionActionSender(func(context.Context, domain.RadiusOnline) sessionqueue.Outcome {
		return sessionqueue.Outcome{Success: true, ResponseCode: "Disconnect-ACK", Attempts: 1}
	})
	require.NoError(t, tenantDB.Model(&domain.RadiusSessionAction{}).Where("id = ?", action.ID).
		Update("next_retry_at", now.Add(-time.Minute)).Error)
	a.SchedSessionActionRetryTask()

	require.NoError(t, tenantDB.First(&action).Error)
	assert.Equal(t, sessionqueue.StatusSucceeded, action.Status)
	assert.Equal(t, 3, action.Attempts)

	var audits []domain.RadiusSessionActionAudit
	require.NoError(t, db.Order("id").Find(&audits).Error)
	require.Len(t, audits, 3)
	assert.False(t, audits[0].Success)
	assert.False(t, audits[1].Success)
	assert.True(t, audits[2].Success)
}

// TestSchedClearExpireData verifies the daily cleanup removes only genuinely
// stale rows: radius_online sessions that have missed several interim updates,
// and terminated radius_accounting rows whose AcctStopTime predates the
// retention window. Live sessions (recent online rows) and active accounting
// rows (zero AcctStopTime) must always survive.
func TestSchedClearExpireData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.SysOprLog{}, &domain.RadiusOnline{}, &domain.RadiusAccounting{}))

	now := time.Now()
	for _, tenantID := range []int64{10, 20} {
		tenant := domain.Tenant{ID: tenantID, Name: fmt.Sprintf("Tenant %d", tenantID), Slug: fmt.Sprintf("tenant-%d", tenantID), Kind: "isp", Status: "active"}
		require.NoError(t, db.Create(&tenant).Error)
		tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
		// Each tenant may reuse session identifiers. Retention must delete stale
		// data in both organizations while preserving recent/active records.
		require.NoError(t, tenantDB.Create(&domain.RadiusOnline{Username: "dangling", AcctSessionId: "same-session", LastUpdate: now.Add(-20 * time.Minute)}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusOnline{Username: "live-recent", AcctSessionId: "live-recent", LastUpdate: now.Add(-60 * time.Second)}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusOnline{Username: "live-quiet", AcctSessionId: "live-quiet", LastUpdate: now.Add(-10 * time.Minute)}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{Username: "old-stopped", AcctStopTime: now.AddDate(0, 0, -40)}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{Username: "recent-stopped", AcctStopTime: now.AddDate(0, 0, -5)}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{Username: "active", AcctStartTime: now.AddDate(0, 0, -40)}).Error)
		require.NoError(t, tenantDB.Create(&domain.SysOprLog{OprName: fmt.Sprintf("old-%d", tenantID), OptTime: now.AddDate(-2, 0, 0)}).Error)
		require.NoError(t, tenantDB.Create(&domain.SysOprLog{OprName: fmt.Sprintf("recent-%d", tenantID), OptTime: now.AddDate(0, 0, -10)}).Error)
	}

	cm := &ConfigManager{
		configs: map[string]string{"radius.AccountingHistoryDays": "30"},
		schemas: make(map[string]*ConfigSchema),
	}
	a := &Application{gormDB: db, configManager: cm}

	a.SchedClearExpireData()

	require.ElementsMatch(t, []string{"live-recent", "live-quiet", "live-recent", "live-quiet"}, onlineUsernames(t, db),
		"stale sessions should be removed in every tenant; live sessions must remain")
	require.ElementsMatch(t, []string{"recent-stopped", "active", "recent-stopped", "active"}, accountingUsernames(t, db),
		"old accounting rows should be pruned in every tenant; recent and active rows must remain")
	var auditLogs []domain.SysOprLog
	require.NoError(t, db.Find(&auditLogs).Error)
	require.Len(t, auditLogs, 2)
	assert.ElementsMatch(t, []string{"recent-10", "recent-20"}, []string{auditLogs[0].OprName, auditLogs[1].OprName},
		"old operator audit rows should be pruned independently in every tenant")
}

// TestSchedClearExpireData_DisabledRetention verifies that AccountingHistoryDays=0
// disables accounting cleanup (matching the config schema's "0=disabled"), so even
// very old terminated rows and active rows are kept — while the independent
// radius_online cleanup still removes dangling sessions.
func TestSchedClearExpireData_DisabledRetention(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(&domain.Tenant{}, &domain.SysOprLog{}, &domain.RadiusOnline{}, &domain.RadiusAccounting{}))

	now := time.Now()
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		Username: "ancient", AcctStopTime: now.AddDate(0, 0, -1000),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusAccounting{
		Username: "active", AcctStartTime: now.AddDate(0, 0, -1000),
	}).Error)
	require.NoError(t, db.Create(&domain.RadiusOnline{
		Username: "dangling", AcctSessionId: "sess-dangling", LastUpdate: now.Add(-20 * time.Minute),
	}).Error)

	cm := &ConfigManager{
		configs: map[string]string{"radius.AccountingHistoryDays": "0"},
		schemas: make(map[string]*ConfigSchema),
	}
	a := &Application{gormDB: db, configManager: cm}

	a.SchedClearExpireData()

	require.ElementsMatch(t, []string{"ancient", "active"}, accountingUsernames(t, db),
		"AccountingHistoryDays=0 must disable accounting cleanup entirely")

	var onlineCount int64
	require.NoError(t, db.Model(&domain.RadiusOnline{}).Count(&onlineCount).Error)
	require.Equal(t, int64(0), onlineCount, "online cleanup runs regardless of AccountingHistoryDays")
}

func onlineUsernames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var rows []domain.RadiusOnline
	require.NoError(t, db.Find(&rows).Error)
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Username)
	}
	return names
}

func accountingUsernames(t *testing.T, db *gorm.DB) []string {
	t.Helper()
	var rows []domain.RadiusAccounting
	require.NoError(t, db.Find(&rows).Error)
	names := make([]string, 0, len(rows))
	for _, r := range rows {
		names = append(names, r.Username)
	}
	return names
}

// TestInitJobRegistersCleanup is the regression guard for M6.4: SchedClearExpireData
// was defined but never registered with cron, so expired online/accounting data was
// never auto-purged. initJob schedules monitor, tenant-aware retention, billing,
// subscriber FUP and session action retry jobs (5 cron entries total).
func TestInitJobRegistersCleanup(t *testing.T) {
	a := &Application{appConfig: &config.AppConfig{}}
	a.initJob()
	defer a.sched.Stop()

	require.Len(t, a.sched.Entries(), 5,
		"expected monitor, tenant-aware retention, billing, FUP and session action retry cron entries")
}

type fupFixture struct {
	sub     domain.Subscription
	user    domain.RadiusUser
	pkg     domain.InternetPackage
	profile domain.RadiusProfile
}

func newFUPDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:fup-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(
		&domain.Tenant{}, &domain.Customer{}, &domain.InternetPackage{}, &domain.RadiusProfile{},
		&domain.RadiusUser{}, &domain.Subscription{}, &domain.RadiusOnline{}, &domain.RadiusAccounting{},
		&domain.BillingEvent{},
	))
	return db
}

func seedFUPFixture(t *testing.T, db *gorm.DB, tenantID int64) fupFixture {
	t.Helper()
	require.NoError(t, db.Create(&domain.Tenant{
		ID: tenantID, Name: fmt.Sprintf("Tenant %d", tenantID), Slug: fmt.Sprintf("tenant-%d", tenantID),
		Kind: "isp", Status: "active",
	}).Error)
	tdb := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
	profile := domain.RadiusProfile{Name: "normal", UpRate: 2048, DownRate: 4096, Status: "enabled"}
	require.NoError(t, tdb.Create(&profile).Error)
	pkg := domain.InternetPackage{
		Code: "HOME", Name: "Home", Price: 100000, RadiusProfileID: profile.ID, BillingCycle: "monthly",
		Status: "active", FupLimitGB: 1, FupRateUp: 256, FupRateDown: 512,
	}
	require.NoError(t, tdb.Create(&pkg).Error)
	user := domain.RadiusUser{Username: "shared-user", Password: "secret", Status: "enabled", UpRate: 2048, DownRate: 4096}
	require.NoError(t, tdb.Create(&user).Error)
	customer := domain.Customer{CustomerNo: "MWX-000001", Name: "Customer", Status: domain.CustomerActive}
	require.NoError(t, tdb.Create(&customer).Error)
	sub := domain.Subscription{
		SubscriptionNo: "SUB-000001", CustomerID: customer.ID, PackageID: pkg.ID, RadiusUserID: user.ID,
		Status: domain.SubscriptionActive, StartDate: time.Now().AddDate(0, -1, 0), BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, tdb.Create(&sub).Error)
	return fupFixture{sub: sub, user: user, pkg: pkg, profile: profile}
}

func seedFUPOnline(t *testing.T, db *gorm.DB, tenantID int64, sessionID string, bytes int64) {
	t.Helper()
	tdb := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
	require.NoError(t, tdb.Create(&domain.RadiusOnline{
		Username: "shared-user", AcctSessionId: sessionID, AcctInputTotal: bytes / 2, AcctOutputTotal: bytes / 2,
		LastUpdate: time.Now(), AcctStartTime: time.Now().Add(-time.Hour),
	}).Error)
}

func seedFUPAccounting(t *testing.T, db *gorm.DB, tenantID int64, sessionID string, bytes int64, stopTime time.Time) {
	t.Helper()
	tdb := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
	require.NoError(t, tdb.Create(&domain.RadiusAccounting{
		Username: "shared-user", AcctSessionId: sessionID, AcctInputTotal: bytes / 2, AcctOutputTotal: bytes / 2,
		AcctStartTime: time.Now().Add(-2 * time.Hour), AcctStopTime: stopTime,
	}).Error)
}

func TestSchedFUPCheckScopesUsageEventsAndDisconnectsPerTenant(t *testing.T) {
	db := newFUPDB(t)
	fixtures := map[int64]fupFixture{
		10: seedFUPFixture(t, db, 10),
		20: seedFUPFixture(t, db, 20),
		30: seedFUPFixture(t, db, 30),
	}

	seedFUPOnline(t, db, 10, "session-10", 2<<30)
	seedFUPOnline(t, db, 20, "session-20", 512<<20)
	// An open accounting row mirrors a live session and must not be added again.
	seedFUPAccounting(t, db, 20, "accounting-20-open", 10<<30, time.Time{})
	// Closed accounting history must count even when the online row is gone.
	seedFUPOnline(t, db, 30, "session-30", 1<<20)
	seedFUPAccounting(t, db, 30, "accounting-30-closed", 2<<30, time.Now().Add(-time.Minute))

	a := &Application{gormDB: db}
	var disconnectedTenantIDs []int64
	a.SetSessionDisconnectHandler(func(ctx context.Context, session domain.RadiusOnline) error {
		tenantID, ok := tenancy.TenantID(ctx)
		require.True(t, ok, "FUP disconnect must carry a tenant context")
		disconnectedTenantIDs = append(disconnectedTenantIDs, tenantID)
		return nil
	})

	a.SchedFUPCheckTask()

	for tenantID, wantTriggered := range map[int64]bool{10: true, 20: false, 30: true} {
		var sub domain.Subscription
		require.NoError(t, db.WithContext(tenancy.WithTenantID(context.Background(), tenantID)).First(&sub, fixtures[tenantID].sub.ID).Error)
		require.Equal(t, wantTriggered, sub.FupTriggered, "tenant %d FUP state", tenantID)

		var user domain.RadiusUser
		require.NoError(t, db.WithContext(tenancy.WithTenantID(context.Background(), tenantID)).First(&user, fixtures[tenantID].user.ID).Error)
		if wantTriggered {
			require.Equal(t, 256, user.UpRate, "tenant %d upload rate", tenantID)
			require.Equal(t, 512, user.DownRate, "tenant %d download rate", tenantID)
		} else {
			require.Equal(t, 2048, user.UpRate, "tenant %d upload rate", tenantID)
			require.Equal(t, 4096, user.DownRate, "tenant %d download rate", tenantID)
		}
	}

	var events []domain.BillingEvent
	require.NoError(t, db.Find(&events).Error)
	require.Len(t, events, 2)
	var eventTenantIDs []int64
	for _, event := range events {
		require.Equal(t, "fup_throttled", event.Type)
		eventTenantIDs = append(eventTenantIDs, event.TenantID)
	}
	assert.ElementsMatch(t, []int64{10, 30}, eventTenantIDs, "FUP audit events must belong to the affected tenants")
	assert.ElementsMatch(t, []int64{10, 30}, disconnectedTenantIDs, "only throttled tenants may be disconnected")
}

func TestSchedFUPCheckResetIsPeriodAwareAndIdempotent(t *testing.T) {
	db := newFUPDB(t)
	fixture := seedFUPFixture(t, db, 10)
	seedFUPOnline(t, db, 10, "session-10", 1<<20)
	tdb := db.WithContext(tenancy.WithTenantID(context.Background(), 10))
	require.NoError(t, tdb.Model(&domain.Subscription{}).Where("id = ?", fixture.sub.ID).Update("fup_triggered", true).Error)
	require.NoError(t, tdb.Model(&domain.RadiusUser{}).Where("id = ?", fixture.user.ID).Updates(map[string]interface{}{"up_rate": 256, "down_rate": 512}).Error)
	require.NoError(t, tdb.Create(&domain.BillingEvent{
		CustomerID: fixture.sub.CustomerID, SubscriptionID: fixture.sub.ID, Type: "fup_throttled",
		CreatedAt: time.Now().AddDate(0, -1, 0),
	}).Error)

	a := &Application{gormDB: db}
	var disconnectCount int
	a.SetSessionDisconnectHandler(func(ctx context.Context, session domain.RadiusOnline) error {
		tenantID, ok := tenancy.TenantID(ctx)
		require.True(t, ok)
		require.Equal(t, int64(10), tenantID)
		disconnectCount++
		return nil
	})

	a.SchedFUPCheckTask()

	var sub domain.Subscription
	require.NoError(t, tdb.First(&sub, fixture.sub.ID).Error)
	require.False(t, sub.FupTriggered, "a throttle from the previous period must reset on the next run")
	var user domain.RadiusUser
	require.NoError(t, tdb.First(&user, fixture.user.ID).Error)
	require.Equal(t, fixture.profile.UpRate, user.UpRate)
	require.Equal(t, fixture.profile.DownRate, user.DownRate)
	require.Equal(t, 1, disconnectCount)

	var events []domain.BillingEvent
	require.NoError(t, tdb.Order("id ASC").Find(&events).Error)
	require.Len(t, events, 2)
	require.Equal(t, "fup_restored", events[1].Type)
	require.Equal(t, int64(10), events[1].TenantID)

	a.SchedFUPCheckTask()
	require.NoError(t, tdb.Find(&events).Error)
	require.Len(t, events, 2, "repeated runs must not duplicate the restore event")
	require.Equal(t, 1, disconnectCount, "repeated runs must not disconnect again")

	require.NoError(t, tdb.Model(&domain.Subscription{}).Where("id = ?", fixture.sub.ID).Update("fup_triggered", true).Error)
	require.NoError(t, tdb.Create(&domain.BillingEvent{
		CustomerID: fixture.sub.CustomerID, SubscriptionID: fixture.sub.ID, Type: "fup_throttled",
		CreatedAt: time.Now(),
	}).Error)
	a.SchedFUPCheckTask()
	require.NoError(t, tdb.First(&sub, fixture.sub.ID).Error)
	require.True(t, sub.FupTriggered, "a current-period throttle must not be reset by a temporary traffic drop")
	require.NoError(t, tdb.Find(&events).Error)
	require.Len(t, events, 3)
	require.Equal(t, 1, disconnectCount)
}
