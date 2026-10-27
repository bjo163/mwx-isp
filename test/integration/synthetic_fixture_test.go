//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/demoseed"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// syntheticFixture is the reusable two-tenant dataset requested by audit item
// H15: one tenant with a full 250-voucher batch, a second tenant with the same
// voucher codes, and invoices covering paid, partially paid and unpaid states.
// It needs no real NAS, RADIUS traffic or WhatsApp pairing.
type syntheticFixture struct {
	TenantA, TenantB     int64
	BatchA, BatchB       int64
	SharedInvoiceNo      string
	SharedVoucherCode    string
	TenantBOnlyVoucher   string
	CustomerA, CustomerB int64
}

func seedSyntheticTwoTenantFixture(t *testing.T) syntheticFixture {
	t.Helper()
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	now := time.Now()
	fixture := syntheticFixture{
		SharedInvoiceNo:    "INV-FIX-" + suffix,
		SharedVoucherCode:  "FIX-" + suffix + "-0001",
		TenantBOnlyVoucher: "FIX-B-ONLY-" + suffix,
	}

	type tenantSeed struct {
		name     string
		slug     string
		short    string // keeps size:32 identifiers inside the column limit
		vouchers int
		tenantID *int64
		batchID  *int64
		customer *int64
	}
	seeds := []tenantSeed{
		{name: "Fixture tenant A", slug: "it-fix-a-" + suffix, short: "A", vouchers: 250, tenantID: &fixture.TenantA, batchID: &fixture.BatchA, customer: &fixture.CustomerA},
		{name: "Fixture tenant B", slug: "it-fix-b-" + suffix, short: "B", vouchers: 5, tenantID: &fixture.TenantB, batchID: &fixture.BatchB, customer: &fixture.CustomerB},
	}
	for _, seed := range seeds {
		tenant := domain.Tenant{Name: seed.name, Slug: seed.slug, Kind: "isp", Status: "active"}
		require.NoError(t, db.Create(&tenant).Error)
		*seed.tenantID = tenant.ID
		tdb := db.WithContext(tenancy.WithTenantID(context.Background(), tenant.ID))

		customer := domain.Customer{
			CustomerNo: "FIX-" + suffix + "-" + seed.short,
			Name:       seed.name + " customer",
			Status:     domain.CustomerActive,
			Notes:      "it-synthetic-fixture",
		}
		require.NoError(t, tdb.Create(&customer).Error)
		*seed.customer = customer.ID
		pkg := domain.InternetPackage{
			Code: "FIX-PKG-" + suffix + "-" + seed.short, Name: "Fixture package " + seed.slug,
			Price: 150_000, BillingCycle: "monthly", Status: "active",
		}
		require.NoError(t, tdb.Create(&pkg).Error)
		subscription := domain.Subscription{
			SubscriptionNo: "FIX-SUB-" + suffix + "-" + seed.short, CustomerID: customer.ID, PackageID: pkg.ID,
			Status: domain.SubscriptionActive, StartDate: now.AddDate(0, -2, 0), BillingDay: 1, GraceDays: 3,
		}
		require.NoError(t, tdb.Create(&subscription).Error)

		batch := domain.HotspotBatch{
			BatchNo: "FIX-BATCH-" + suffix + "-" + seed.short, Name: "Fixture batch " + seed.slug,
			PackageID: pkg.ID, Quantity: seed.vouchers, Price: 5_000,
			ValiditySeconds: 86_400, CodeLength: 8, CreatedBy: "it-synthetic-fixture",
		}
		require.NoError(t, tdb.Create(&batch).Error)
		*seed.batchID = batch.ID
		vouchers := make([]domain.HotspotVoucher, seed.vouchers)
		for i := range vouchers {
			vouchers[i] = domain.HotspotVoucher{
				BatchID: batch.ID, PackageID: pkg.ID,
				Code:     fmt.Sprintf("FIX-%s-%04d", suffix, i+1),
				Password: fmt.Sprintf("P%04d", i+1),
				Price:    5_000, ValiditySeconds: 86_400, Status: "active",
			}
		}
		if seed.vouchers == 5 {
			vouchers = append(vouchers, domain.HotspotVoucher{
				BatchID: batch.ID, PackageID: pkg.ID,
				Code: fixture.TenantBOnlyVoucher, Password: "P-BONLY",
				Price: 5_000, ValiditySeconds: 86_400, Status: "active",
			})
		}
		require.NoError(t, tdb.Create(&vouchers).Error)

		invoiceStatuses := []struct {
			status string
			paid   int64
			period time.Time
		}{
			{status: "paid", paid: 150_000, period: now.AddDate(0, -3, 0)},
			{status: "partial", paid: 50_000, period: now.AddDate(0, -2, 0)},
			{status: "issued", paid: 0, period: now.AddDate(0, -1, 0)},
		}
		if seed.vouchers == 5 {
			invoiceStatuses = invoiceStatuses[:1]
		}
		for i, spec := range invoiceStatuses {
			invoiceNo := fmt.Sprintf("INV-FIX-%s-%d", suffix, i+1)
			if i == 0 {
				invoiceNo = fixture.SharedInvoiceNo
			}
			invoice := domain.Invoice{
				InvoiceNo: invoiceNo, CustomerID: customer.ID, SubscriptionID: subscription.ID,
				InvoiceDate: spec.period, DueDate: spec.period.AddDate(0, 0, 14),
				PeriodStart: spec.period, PeriodEnd: spec.period.AddDate(0, 1, -1),
				Subtotal: 150_000, Total: 150_000, PaidAmount: spec.paid, Balance: 150_000 - spec.paid,
				Status: spec.status,
			}
			require.NoError(t, tdb.Create(&invoice).Error)
			if spec.paid > 0 {
				item := invoiceItemFor(invoice)
				require.NoError(t, tdb.Create(&item).Error)
				payment := domain.Payment{
					PaymentNo:  fmt.Sprintf("FIX-PAY-%s-%d", suffix, i+1),
					CustomerID: customer.ID, InvoiceID: invoice.ID, Amount: spec.paid,
					Method: "bank_transfer", PaidAt: now, Status: "received",
				}
				require.NoError(t, tdb.Create(&payment).Error)
			}
		}
	}
	return fixture
}

func invoiceItemFor(invoice domain.Invoice) domain.InvoiceItem {
	return domain.InvoiceItem{
		InvoiceID: invoice.ID, Description: "Fixture monthly subscription",
		Quantity: 1, UnitPrice: invoice.Subtotal, Total: invoice.Subtotal,
	}
}

func TestSyntheticTwoTenantFixtureCoversVouchersAndPartialInvoices(t *testing.T) {
	fixture := seedSyntheticTwoTenantFixture(t)
	db := h.appCtx.DB()
	ctxA := tenancy.WithTenantID(context.Background(), fixture.TenantA)
	ctxB := tenancy.WithTenantID(context.Background(), fixture.TenantB)
	dbA := db.WithContext(ctxA)
	dbB := db.WithContext(ctxB)

	requireVoucher := func(tdb *gorm.DB, batchID int64, want int64) {
		t.Helper()
		var count int64
		require.NoError(t, tdb.Model(&domain.HotspotVoucher{}).Where("batch_id = ?", batchID).Count(&count).Error)
		assert.EqualValues(t, want, count)
	}
	requireVoucher(dbA, fixture.BatchA, 250)
	requireVoucher(dbB, fixture.BatchB, 6) // five shared codes plus one tenant-B-only code

	var sharedA, sharedB domain.HotspotVoucher
	require.NoError(t, dbA.Where("code = ?", fixture.SharedVoucherCode).First(&sharedA).Error)
	require.NoError(t, dbB.Where("code = ?", fixture.SharedVoucherCode).First(&sharedB).Error)
	assert.Equal(t, fixture.BatchA, sharedA.BatchID, "tenant A must resolve its own voucher row")
	assert.Equal(t, fixture.BatchB, sharedB.BatchID, "tenant B must resolve its own voucher row")
	assert.NotEqual(t, sharedA.ID, sharedB.ID)

	var leaked int64
	require.NoError(t, dbA.Model(&domain.HotspotVoucher{}).Where("code = ?", fixture.TenantBOnlyVoucher).Count(&leaked).Error)
	assert.Zero(t, leaked, "tenant A must never see the tenant-B-only voucher")

	var invoicesA []domain.Invoice
	require.NoError(t, dbA.Order("invoice_no ASC").Find(&invoicesA).Error)
	require.Len(t, invoicesA, 3, "tenant A fixture must include partial invoice states")
	statuses := map[string]int64{}
	for _, invoice := range invoicesA {
		statuses[invoice.Status] = invoice.Balance
	}
	assert.EqualValues(t, 0, statuses["paid"])
	assert.EqualValues(t, 100_000, statuses["partial"])
	assert.EqualValues(t, 150_000, statuses["issued"])

	var sharedInvoiceA, sharedInvoiceB domain.Invoice
	require.NoError(t, dbA.Where("invoice_no = ?", fixture.SharedInvoiceNo).First(&sharedInvoiceA).Error)
	require.NoError(t, dbB.Where("invoice_no = ?", fixture.SharedInvoiceNo).First(&sharedInvoiceB).Error)
	assert.Equal(t, fixture.CustomerA, sharedInvoiceA.CustomerID)
	assert.Equal(t, fixture.CustomerB, sharedInvoiceB.CustomerID)
	assert.NotEqual(t, sharedInvoiceA.ID, sharedInvoiceB.ID)

	// Demo cleanup scoped to tenant A must preserve unmarked fixture rows in both
	// tenants, proving production data is never swept away.
	require.NoError(t, demoseed.Clean(dbA))
	requireVoucher(dbA, fixture.BatchA, 250)
	requireVoucher(dbB, fixture.BatchB, 6)
	var fixturesA, fixturesB int64
	require.NoError(t, dbA.Model(&domain.Customer{}).Where("notes = ?", "it-synthetic-fixture").Count(&fixturesA).Error)
	require.NoError(t, dbB.Model(&domain.Customer{}).Where("notes = ?", "it-synthetic-fixture").Count(&fixturesB).Error)
	assert.EqualValues(t, 1, fixturesA)
	assert.EqualValues(t, 1, fixturesB)
}
