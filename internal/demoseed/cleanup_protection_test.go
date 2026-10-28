package demoseed

import (
	"context"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newDemoSeedTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(domain.Tables...))
	return db
}

func seedDemoData(t *testing.T, db *gorm.DB) *demoSeeder {
	t.Helper()
	seeder := &demoSeeder{db: db, now: time.Date(2026, time.October, 4, 10, 0, 0, 0, time.UTC), historyDays: 1}
	require.NoError(t, seeder.run())
	return seeder
}

// TestCleanPreservesRealRecordsThatReferenceDemoInventory pins the "data riil
// tidak tersentuh" half of the acceptance: operator records that point at sample
// inventory keep both themselves and the inventory rows they depend on.
func TestCleanPreservesRealRecordsThatReferenceDemoInventory(t *testing.T) {
	db := newDemoSeedTestDB(t)
	seeder := seedDemoData(t, db)

	var demoProfile domain.RadiusProfile
	require.NoError(t, db.Where("remark = ? AND name = ?", demoMarker, "demo-basic").First(&demoProfile).Error)
	var demoNode domain.NetNode
	require.NoError(t, db.Where("remark = ? AND name = ?", demoMarker, "demo-edge").First(&demoNode).Error)
	var demoPackage domain.InternetPackage
	require.NoError(t, db.Where("code = ?", "DEMO-PKG-STARTER").First(&demoPackage).Error)
	var demoCustomer domain.Customer
	require.NoError(t, db.Where("notes = ?", demoMarker).Order("id ASC").First(&demoCustomer).Error)

	realCustomer := domain.Customer{CustomerNo: "MWX-100001", Name: "Real Holdings", Status: domain.CustomerActive, Notes: "operator-owned"}
	require.NoError(t, db.Create(&realCustomer).Error)
	realUser := domain.RadiusUser{
		NodeId: demoNode.ID, ProfileId: demoProfile.ID, Username: "real-subscriber",
		Password: "real-secret", Status: "enabled", Remark: "operator-owned",
	}
	require.NoError(t, db.Create(&realUser).Error)
	realSubscription := domain.Subscription{
		SubscriptionNo: "SUB-REAL-0001", CustomerID: realCustomer.ID, PackageID: demoPackage.ID,
		RadiusUserID: realUser.ID, Status: domain.SubscriptionActive, StartDate: seeder.now,
	}
	require.NoError(t, db.Create(&realSubscription).Error)
	realInvoice := domain.Invoice{
		InvoiceNo: "INV-REAL-0001", CustomerID: realCustomer.ID, SubscriptionID: realSubscription.ID,
		InvoiceDate: seeder.now, DueDate: seeder.now.AddDate(0, 0, 14),
		PeriodStart: seeder.now, PeriodEnd: seeder.now.AddDate(0, 1, 0),
		Subtotal: 250_000, Total: 250_000, Balance: 250_000, Status: "issued", Notes: "operator-owned",
	}
	require.NoError(t, db.Create(&realInvoice).Error)

	// The sample customer gets a real subscription and a real invoice of its own,
	// the way an operator would keep billing someone they turned into a customer.
	adoptedSubscription := domain.Subscription{
		SubscriptionNo: "SUB-REAL-0002", CustomerID: demoCustomer.ID, PackageID: demoPackage.ID,
		Status: domain.SubscriptionActive, StartDate: seeder.now,
	}
	require.NoError(t, db.Create(&adoptedSubscription).Error)
	adoptedInvoice := domain.Invoice{
		InvoiceNo: "INV-REAL-0002", CustomerID: demoCustomer.ID, SubscriptionID: adoptedSubscription.ID,
		InvoiceDate: seeder.now, DueDate: seeder.now.AddDate(0, 0, 14),
		PeriodStart: seeder.now, PeriodEnd: seeder.now.AddDate(0, 1, 0),
		Subtotal: 250_000, Total: 250_000, Balance: 250_000, Status: "issued", Notes: "operator-owned",
	}
	require.NoError(t, db.Create(&adoptedInvoice).Error)

	require.NoError(t, seeder.clean())

	var keptCustomer domain.Customer
	require.NoError(t, db.First(&keptCustomer, realCustomer.ID).Error)
	var keptUser domain.RadiusUser
	require.NoError(t, db.First(&keptUser, realUser.ID).Error)
	require.Equal(t, "real-secret", keptUser.Password, "real credentials must survive sample cleanup")
	var keptSubscription domain.Subscription
	require.NoError(t, db.First(&keptSubscription, realSubscription.ID).Error)
	var keptAdopted domain.Subscription
	require.NoError(t, db.First(&keptAdopted, adoptedSubscription.ID).Error)
	var keptPackage domain.InternetPackage
	require.NoError(t, db.First(&keptPackage, demoPackage.ID).Error, "sample package used by a real subscription must be retained")
	var keptProfile domain.RadiusProfile
	require.NoError(t, db.First(&keptProfile, demoProfile.ID).Error, "sample profile used by a real user must be retained")
	var keptNode domain.NetNode
	require.NoError(t, db.First(&keptNode, demoNode.ID).Error, "sample node used by a retained profile must be retained")
	var keptDemoCustomer domain.Customer
	require.NoError(t, db.First(&keptDemoCustomer, demoCustomer.ID).Error, "sample customer carrying a real invoice must be retained")
	var keptInvoice domain.Invoice
	require.NoError(t, db.First(&keptInvoice, realInvoice.ID).Error)
	var keptAdoptedInvoice domain.Invoice
	require.NoError(t, db.First(&keptAdoptedInvoice, adoptedInvoice.ID).Error)

	var remainingDemoUsers int64
	require.NoError(t, db.Model(&domain.RadiusUser{}).Where("remark = ?", demoMarker).Count(&remainingDemoUsers).Error)
	require.Zero(t, remainingDemoUsers)
	var remainingDemoPackages int64
	require.NoError(t, db.Model(&domain.InternetPackage{}).Where("code LIKE ?", "DEMO-PKG-%").Count(&remainingDemoPackages).Error)
	require.EqualValues(t, 1, remainingDemoPackages, "only the package real subscriptions use may remain")
}

// TestCleanKeepsAdoptedSubscriptionAndDropsSeedLink verifies that reparenting a
// sample subscription to a real customer is treated as adoption: cleanup leaves
// the subscription with its new owner and only removes the demo marker that made
// the seeder consider it sample data.
func TestCleanKeepsAdoptedSubscriptionAndDropsSeedLink(t *testing.T) {
	db := newDemoSeedTestDB(t)
	seeder := seedDemoData(t, db)

	var adopted domain.Subscription
	require.NoError(t, db.Joins("JOIN isp_customer ON isp_customer.id = isp_subscription.customer_id").
		Where("isp_customer.notes = ?", demoMarker).Order("isp_subscription.id ASC").First(&adopted).Error)
	var owner domain.Customer
	require.NoError(t, db.First(&owner, adopted.CustomerID).Error)
	realCustomer := domain.Customer{CustomerNo: "MWX-100002", Name: "Adopting ISP", Status: domain.CustomerActive, Notes: "operator-owned"}
	require.NoError(t, db.Create(&realCustomer).Error)
	require.NoError(t, db.Model(&domain.Subscription{}).Where("id = ?", adopted.ID).
		Updates(map[string]interface{}{"customer_id": realCustomer.ID, "subscription_no": "SUB-REAL-ADOPTED"}).Error)

	require.NoError(t, seeder.clean())

	var kept domain.Subscription
	require.NoError(t, db.First(&kept, adopted.ID).Error, "a subscription reparented to a real customer must not be deleted")
	require.Equal(t, realCustomer.ID, kept.CustomerID)
	var seedLinks int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("type = ? AND subscription_id = ?", "demo_seed", adopted.ID).Count(&seedLinks).Error)
	require.Zero(t, seedLinks, "the adopted subscription must drop its sample marker")

	// The sample data can be regenerated without touching the adopted record.
	require.NoError(t, seeder.run())
	var reseededOwner domain.Customer
	require.NoError(t, db.Where("name = ? AND notes = ?", owner.Name, demoMarker).First(&reseededOwner).Error)
	var regenerated int64
	require.NoError(t, db.Model(&domain.Subscription{}).Where("customer_id = ?", reseededOwner.ID).Count(&regenerated).Error)
	require.EqualValues(t, 1, regenerated, "seeding again must recreate the sample subscription")
	var stillAdopted domain.Subscription
	require.NoError(t, db.First(&stillAdopted, adopted.ID).Error)
	require.Equal(t, realCustomer.ID, stillAdopted.CustomerID)
}

func TestCleanOnlyTouchesTheCallingTenantInventory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, tenancy.RegisterCallbacks(db))
	require.NoError(t, db.AutoMigrate(domain.Tables...))

	for _, tenantID := range []int64{10, 20} {
		tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
		node := domain.NetNode{Name: "demo-core", Remark: demoMarker}
		require.NoError(t, tenantDB.Create(&node).Error)
		spare := domain.NetNode{Name: "demo-spare", Remark: demoMarker}
		require.NoError(t, tenantDB.Create(&spare).Error)
		profile := domain.RadiusProfile{NodeId: node.ID, Name: "demo-basic", Status: "enabled", Remark: demoMarker}
		require.NoError(t, tenantDB.Create(&profile).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusProfile{
			NodeId: spare.ID, Name: "demo-spare-profile", Status: "enabled", Remark: demoMarker,
		}).Error)
		require.NoError(t, tenantDB.Create(&domain.NetNas{
			NodeId: node.ID, Name: "demo-bras-1", Identifier: "demo-bras-1", Remark: demoMarker, Status: "disabled",
		}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusUser{
			NodeId: node.ID, ProfileId: profile.ID, Username: "demo-alice", Password: "demo-password",
			Remark: demoMarker, Status: "disabled",
		}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusAccounting{
			Username: "demo-alice", AcctSessionId: "demo-session", NasId: "demo-bras-1", NasClass: demoMarker,
		}).Error)
		require.NoError(t, tenantDB.Create(&domain.RadiusUser{
			NodeId: node.ID, ProfileId: profile.ID, Username: "real-subscriber", Password: "real-secret",
			Remark: "operator-owned", Status: "enabled",
		}).Error)
	}

	ctxA := tenancy.WithTenantID(context.Background(), 10)
	require.NoError(t, Clean(db.WithContext(ctxA)))

	// Tenant A: the unreferenced sample rows are gone, while the sample node and
	// profile a real user still points at are retained.
	assertTenantDemoCounts(t, db, 10, map[string]int64{
		"net_node": 1, "net_nas": 0, "radius_profile": 1, "radius_user": 0, "radius_accounting": 0,
	})
	tenantA := db.WithContext(ctxA)
	var survivor domain.NetNode
	require.NoError(t, tenantA.Where("remark = ?", demoMarker).First(&survivor).Error)
	require.Equal(t, "demo-core", survivor.Name)
	var survivorProfile domain.RadiusProfile
	require.NoError(t, tenantA.Where("remark = ?", demoMarker).First(&survivorProfile).Error)
	require.Equal(t, "demo-basic", survivorProfile.Name)

	// Tenant B is untouched in every table.
	assertTenantDemoCounts(t, db, 20, map[string]int64{
		"net_node": 2, "net_nas": 1, "radius_profile": 2, "radius_user": 1, "radius_accounting": 1,
	})

	for _, tenantID := range []int64{10, 20} {
		tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
		var realUser domain.RadiusUser
		require.NoError(t, tenantDB.Where("username = ?", "real-subscriber").First(&realUser).Error)
		require.Equal(t, "real-secret", realUser.Password)
	}
}

func assertTenantDemoCounts(t *testing.T, db *gorm.DB, tenantID int64, want map[string]int64) {
	t.Helper()
	tenantDB := db.WithContext(tenancy.WithTenantID(context.Background(), tenantID))
	models := map[string]struct {
		model interface{}
		where string
		args  []interface{}
	}{
		"net_node":          {&domain.NetNode{}, "remark = ?", []interface{}{demoMarker}},
		"net_nas":           {&domain.NetNas{}, "remark = ?", []interface{}{demoMarker}},
		"radius_profile":    {&domain.RadiusProfile{}, "remark = ?", []interface{}{demoMarker}},
		"radius_user":       {&domain.RadiusUser{}, "remark = ?", []interface{}{demoMarker}},
		"radius_accounting": {&domain.RadiusAccounting{}, "nas_class = ?", []interface{}{demoMarker}},
	}
	for key, expected := range want {
		check := models[key]
		var count int64
		require.NoError(t, tenantDB.Model(check.model).Where(check.where, check.args...).Count(&count).Error)
		require.Equalf(t, expected, count, "unexpected sample count for %s in tenant %d", key, tenantID)
	}
}
