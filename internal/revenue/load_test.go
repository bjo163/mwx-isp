package revenue

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var revenueDBSeq atomic.Int64

// revenueTestDB opens a throwaway in-memory database holding the tables the
// audit reads, standing in for a sanitized sample export.
func revenueTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:revenue-%d?mode=memory&cache=shared", revenueDBSeq.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(
		&domain.Subscription{}, &domain.InternetPackage{}, &domain.Invoice{}, &domain.Payment{},
		&domain.RadiusUser{}, &domain.RadiusOnline{},
	))
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func TestLoadReadsEveryTableTheAuditNeeds(t *testing.T) {
	db := revenueTestDB(t)
	input := leakageShowcase()
	for _, row := range input.Subscriptions {
		require.NoError(t, db.Create(&row).Error)
	}
	for _, row := range input.Packages {
		require.NoError(t, db.Create(&row).Error)
	}
	for _, row := range input.Invoices {
		require.NoError(t, db.Create(&row).Error)
	}
	for _, row := range input.Payments {
		require.NoError(t, db.Create(&row).Error)
	}
	for _, row := range input.Users {
		require.NoError(t, db.Create(&row).Error)
	}
	for _, row := range input.Online {
		require.NoError(t, db.Create(&row).Error)
	}

	loaded, err := Load(db)
	require.NoError(t, err)
	assert.Len(t, loaded.Subscriptions, len(input.Subscriptions))
	assert.Len(t, loaded.Packages, len(input.Packages))
	assert.Len(t, loaded.Invoices, len(input.Invoices))
	assert.Len(t, loaded.Payments, len(input.Payments))
	assert.Len(t, loaded.Users, len(input.Users))
	assert.Len(t, loaded.Online, len(input.Online))

	fromRows := Audit(input, auditNow)
	fromDatabase, err := AuditDatabase(db, auditNow)
	require.NoError(t, err)
	assert.Equal(t, fromRows.Totals.ByKind, fromDatabase.Totals.ByKind,
		"the same rows give the same findings whether they came from a fixture or a database:\nfixture:\n%s\ndatabase:\n%s",
		renderFindings(fromRows), renderFindings(fromDatabase))
	assert.Equal(t, fromRows.Totals.Exposure, fromDatabase.Totals.Exposure)
}

func TestLoadWritesNothing(t *testing.T) {
	db := revenueTestDB(t)
	require.NoError(t, db.Create(&domain.Subscription{
		ID: 20, TenantID: fixtureTenant, SubscriptionNo: "SUB-0020", CustomerID: 9,
		PackageID: 2, Status: domain.SubscriptionActive,
		StartDate: monthStart(time.October), BillingDay: 5, GraceDays: 3,
	}).Error)

	var before int64
	require.NoError(t, db.Model(&domain.Subscription{}).Count(&before).Error)
	loaded, err := Load(db)
	require.NoError(t, err)
	report := Audit(loaded, auditNow)
	assert.NotEmpty(t, report.Findings, "a subscription with no invoice at all is the case this audit exists for")
	finding := assertKind(t, report, KindUnbilledPeriod)
	assert.Contains(t, strings.Join(finding.Evidence, " "), "package row missing")
	assert.False(t, finding.Quantified, "without the package row there is no price to claim")

	var after int64
	require.NoError(t, db.Model(&domain.Subscription{}).Count(&after).Error)
	assert.Equal(t, before, after, "the audit must not touch the ledger")
}
