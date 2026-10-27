//go:build integration

package integration

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// TestPostgresInvoiceNumberingConcurrentAndRollback proves that concurrent
// invoice allocations never share a number and that a transaction aborted after
// allocating releases the number, so a retry after a crash reuses it instead of
// leaving an unexplained gap.
func TestPostgresInvoiceNumberingConcurrentAndRollback(t *testing.T) {
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	kind := "invoice-" + suffix[len(suffix)-8:]
	const period = "202610"

	customer := domain.Customer{
		ID: common.UUIDint64(), CustomerNo: "C-NUM-" + suffix, Name: "Numbering " + suffix, Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&customer).Error)

	const workers = 8
	serials := make([]int64, workers)
	errs := make([]error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = db.Transaction(func(tx *gorm.DB) error {
				serial, err := billing.NextDocumentSerial(tx, kind, period)
				if err != nil {
					return err
				}
				serials[i] = serial
				now := time.Now().UTC()
				return tx.Create(&domain.Invoice{
					ID: common.UUIDint64(), TenantID: domain.DefaultTenantID,
					InvoiceNo: fmt.Sprintf("INV-%s-%06d", suffix, serial), CustomerID: customer.ID,
					InvoiceDate: now, DueDate: now, PeriodStart: now, PeriodEnd: now.AddDate(0, 1, 0),
					Subtotal: 1, Total: 1, Balance: 1, Status: domain.InvoiceIssued,
				}).Error
			})
		}(i)
	}
	wg.Wait()
	for i, err := range errs {
		require.NoErrorf(t, err, "worker %d", i)
	}

	seen := make(map[int64]struct{}, workers)
	for _, serial := range serials {
		_, duplicate := seen[serial]
		assert.Falsef(t, duplicate, "serial %d was allocated twice", serial)
		seen[serial] = struct{}{}
	}
	assert.Len(t, seen, workers, "every concurrent allocation must be unique")

	var stored domain.DocumentSequence
	require.NoError(t, db.Where("kind = ? AND period = ?", kind, period).First(&stored).Error)
	require.EqualValues(t, workers, stored.Value)

	// A transaction that allocated but never committed models a crash between
	// allocation and commit; it must not consume the number.
	crash := errors.New("simulated crash after allocation")
	require.ErrorIs(t, db.Transaction(func(tx *gorm.DB) error {
		if _, err := billing.NextDocumentSerial(tx, kind, period); err != nil {
			return err
		}
		return crash
	}), crash)
	require.NoError(t, db.Where("kind = ? AND period = ?", kind, period).First(&stored).Error)
	require.EqualValues(t, workers, stored.Value, "an aborted allocation must not consume a number")

	serial, err := billing.NextDocumentSerial(db, kind, period)
	require.NoError(t, err)
	assert.EqualValues(t, workers+1, serial, "the retry after the crash reuses the released number")

	var invoiceCount int64
	require.NoError(t, db.Model(&domain.Invoice{}).Where("customer_id = ?", customer.ID).Count(&invoiceCount).Error)
	assert.EqualValues(t, workers, invoiceCount)
}
