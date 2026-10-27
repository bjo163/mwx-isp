package adminapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePaymentIdempotencyKeyReplaysWithoutDoubleBooking(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Invoice{}, &domain.Payment{}, &domain.BillingEvent{}, &domain.DocumentSequence{},
	))
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-IDEM-API",
		Total: 100000, Balance: 100000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)

	submit := func(key string, amount int64) (*httptest.ResponseRecorder, error) {
		body := fmt.Sprintf(`{"invoice_id":"%d","amount":%d,"method":"cash"}`, invoice.ID, amount)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/isp/payments", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if key != "" {
			req.Header.Set("Idempotency-Key", key)
		}
		rec := httptest.NewRecorder()
		c := CreateTestContext(e, db, req, rec, appCtx)
		c.Set("tenant_id", domain.DefaultTenantID)
		return rec, createPayment(c)
	}
	paymentOf := func(t *testing.T, rec *httptest.ResponseRecorder) domain.Payment {
		t.Helper()
		var envelope struct {
			Data domain.Payment `json:"data"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
		return envelope.Data
	}

	first, err := submit("pay-retry-1", 40000)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, first.Code, first.Body.String())

	replay, err := submit("pay-retry-1", 40000)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, replay.Code, replay.Body.String())
	assert.Equal(t, "true", replay.Header().Get("Idempotent-Replay"))
	assert.Equal(t, paymentOf(t, first).PaymentNo, paymentOf(t, replay).PaymentNo)

	conflict, err := submit("pay-retry-1", 50000)
	require.NoError(t, err)
	require.Equal(t, http.StatusConflict, conflict.Code, conflict.Body.String())
	assert.Contains(t, conflict.Body.String(), "IDEMPOTENCY_CONFLICT")

	invalid, err := submit("bad key", 40000)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, invalid.Code, invalid.Body.String())
	assert.Contains(t, invalid.Body.String(), "INVALID_IDEMPOTENCY_KEY")

	second, err := submit("pay-retry-2", 40000)
	require.NoError(t, err)
	require.Equal(t, http.StatusCreated, second.Code, second.Body.String())

	var updated domain.Invoice
	require.NoError(t, db.First(&updated, invoice.ID).Error)
	assert.Equal(t, int64(80000), updated.PaidAmount)
	assert.Equal(t, int64(20000), updated.Balance)

	var payments, events int64
	require.NoError(t, db.Model(&domain.Payment{}).Where("invoice_id = ?", invoice.ID).Count(&payments).Error)
	assert.EqualValues(t, 2, payments)
	require.NoError(t, db.Model(&domain.BillingEvent{}).Where("invoice_id = ? AND type = ?", invoice.ID, "payment_received").Count(&events).Error)
	assert.EqualValues(t, 2, events)
}
