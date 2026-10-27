//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type paymentAttempt struct {
	status int
	body   []byte
}

// postPaymentRaw issues one payment without touching *testing.T so it is safe
// to call from a goroutine. The row lock in billing.RecordPayment is what makes
// the two requests serialize; the assertions run after both complete.
func postPaymentRaw(client *apiClient, invoiceID, amount int64, reference string) (paymentAttempt, error) {
	payload, err := json.Marshal(map[string]interface{}{
		"invoice_id": fmt.Sprint(invoiceID),
		"amount":     amount,
		"method":     "manual",
		"reference":  reference,
	})
	if err != nil {
		return paymentAttempt{}, err
	}
	req, err := http.NewRequest(http.MethodPost, client.base+"/api/v1/isp/payments", bytes.NewReader(payload))
	if err != nil {
		return paymentAttempt{}, err
	}
	req.Header.Set("Authorization", "Bearer "+client.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.http.Do(req)
	if err != nil {
		return paymentAttempt{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return paymentAttempt{}, err
	}
	return paymentAttempt{status: resp.StatusCode, body: body}, nil
}

// TestPostgresConcurrentPaymentsDoNotOverbookInvoice proves that two operators
// submitting a payment at the same time cannot both book against the same
// remaining balance, that a rejected attempt leaves no payment or billing event
// behind, and that a fully settled invoice rejects further payments.
func TestPostgresConcurrentPaymentsDoNotOverbookInvoice(t *testing.T) {
	client := newAPIClient(t)
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	now := time.Now().UTC().Truncate(time.Second)

	customer := domain.Customer{
		ID: common.UUIDint64(), CustomerNo: "C-PAY-" + suffix, Name: "Payment " + suffix, Status: domain.CustomerActive,
	}
	require.NoError(t, db.Create(&customer).Error)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-PAY-" + suffix, CustomerID: customer.ID,
		InvoiceDate: now, DueDate: now.Add(72 * time.Hour), PeriodStart: now, PeriodEnd: now.Add(30 * 24 * time.Hour),
		Subtotal: 100000, Total: 100000, Balance: 100000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)

	attempts := make([]paymentAttempt, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			attempts[i], errs[i] = postPaymentRaw(client, invoice.ID, 60000, fmt.Sprintf("concurrent-%s-%d", suffix, i))
		}(i)
	}
	wg.Wait()
	for _, err := range errs {
		require.NoError(t, err)
	}

	accepted, rejected := 0, 0
	for _, attempt := range attempts {
		switch attempt.status {
		case http.StatusCreated:
			accepted++
		case http.StatusBadRequest, http.StatusConflict:
			rejected++
		default:
			require.Failf(t, "unexpected payment status", "status %d body %s", attempt.status, attempt.body)
		}
	}
	assert.Equal(t, 1, accepted, "exactly one of two concurrent payments must be accepted")
	assert.Equal(t, 1, rejected, "the other concurrent payment must be rejected")

	var stored domain.Invoice
	require.NoError(t, db.First(&stored, invoice.ID).Error)
	assert.Equal(t, int64(40000), stored.Balance)
	assert.Equal(t, int64(60000), stored.PaidAmount)
	assert.Equal(t, domain.InvoicePartial, stored.Status)
	assert.GreaterOrEqual(t, stored.Balance, int64(0))

	var paymentCount int64
	require.NoError(t, db.Model(&domain.Payment{}).Where("invoice_id = ?", invoice.ID).Count(&paymentCount).Error)
	assert.Equal(t, int64(1), paymentCount, "the rejected attempt must not persist a payment")
	var eventCount int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("invoice_id = ? AND type = ?", invoice.ID, "payment_received").Count(&eventCount).Error)
	assert.Equal(t, int64(1), eventCount, "the rejected attempt must not duplicate the billing event")

	// Settling the remaining balance succeeds, and the invoice is then closed to
	// further payments without creating additional income.
	second, err := postPaymentRaw(client, invoice.ID, 40000, "settle-"+suffix)
	require.NoError(t, err)
	require.Equalf(t, http.StatusCreated, second.status, "%s", second.body)
	require.NoError(t, db.First(&stored, invoice.ID).Error)
	assert.Zero(t, stored.Balance)
	assert.Equal(t, int64(100000), stored.PaidAmount)
	assert.Equal(t, domain.InvoicePaid, stored.Status)

	third, err := postPaymentRaw(client, invoice.ID, 40000, "duplicate-"+suffix)
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, third.status)
	assert.Contains(t, string(third.body), "INVOICE_CLOSED")

	require.NoError(t, db.Model(&domain.Payment{}).Where("invoice_id = ?", invoice.ID).Count(&paymentCount).Error)
	assert.Equal(t, int64(2), paymentCount, "settlement plus the accepted concurrent payment only")
	require.NoError(t, db.Model(&domain.BillingEvent{}).
		Where("invoice_id = ? AND type = ?", invoice.ID, "payment_received").Count(&eventCount).Error)
	assert.Equal(t, int64(2), eventCount)
}
