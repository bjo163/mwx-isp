package adminapi

import (
	"bytes"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/require"
)

func signedMidtransNotification(serverKey string, n map[string]any) []byte {
	raw, _ := json.Marshal(n)
	var parsed struct {
		OrderID string `json:"order_id"`
		StatusCode string `json:"status_code"`
		GrossAmount string `json:"gross_amount"`
	}
	_ = json.Unmarshal(raw, &parsed)
	sum := sha512.Sum512([]byte(parsed.OrderID + parsed.StatusCode + parsed.GrossAmount + serverKey))
	n["signature_key"] = hex.EncodeToString(sum[:])
	raw, _ = json.Marshal(n)
	return raw
}

func TestMidtransWebhookVerifiesAndBooksSettlementOnce(t *testing.T) {
	t.Setenv("TOUGHRADIUS_PAYMENT_PROVIDER", "midtrans")
	t.Setenv("TOUGHRADIUS_MIDTRANS_SERVER_KEY", "test-server-key")
	t.Setenv("TOUGHRADIUS_MIDTRANS_CLIENT_KEY", "test-client-key")
	t.Setenv("TOUGHRADIUS_MIDTRANS_ENVIRONMENT", "sandbox")

	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.Customer{}, &domain.Invoice{}, &domain.InvoiceItem{}, &domain.Payment{},
		&domain.PaymentAttempt{}, &domain.BillingEvent{}, &domain.DocumentSequence{},
		&domain.Subscription{}, &domain.RadiusUser{}, &domain.SysOprLog{},
	))
	customer := domain.Customer{Name: "Customer A", CustomerNo: "MWX-TEST-1"}
	require.NoError(t, db.Create(&customer).Error)
	now := time.Now()
	invoice := domain.Invoice{
		InvoiceNo: "INV-TEST-1", CustomerID: customer.ID,
		InvoiceDate: now, DueDate: now.Add(24*time.Hour),
		PeriodStart: now, PeriodEnd: now.Add(30*24*time.Hour),
		Subtotal: 150000, Total: 150000, Balance: 150000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)
	orderID := "MWX-1-1-1700000000000000000"
	attempt := domain.PaymentAttempt{
		InvoiceID: invoice.ID, CustomerID: customer.ID, Provider: "midtrans",
		OrderID: orderID, Amount: 150000, Currency: "IDR", Status: "pending",
		CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, db.Create(&attempt).Error)

	body := signedMidtransNotification("test-server-key", map[string]any{
		"order_id": orderID, "status_code": "200", "gross_amount": "150000.00",
		"transaction_status": "settlement", "transaction_id": "TXN-1", "currency": "IDR",
	})
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/payments/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		ctx := CreateTestContext(e, db, req, rec, appCtx)
		require.NoError(t, handlePaymentWebhook(ctx))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	var count int64
	require.NoError(t, db.Model(&domain.Payment{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
	require.NoError(t, db.First(&invoice, invoice.ID).Error)
	require.EqualValues(t, 0, invoice.Balance)
	require.NoError(t, db.First(&attempt, attempt.ID).Error)
	require.Equal(t, "paid", attempt.Status)
}

func TestMidtransWebhookRejectsForgedSignature(t *testing.T) {
	t.Setenv("TOUGHRADIUS_PAYMENT_PROVIDER", "midtrans")
	t.Setenv("TOUGHRADIUS_MIDTRANS_SERVER_KEY", "test-server-key")
	t.Setenv("TOUGHRADIUS_MIDTRANS_CLIENT_KEY", "test-client-key")

	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.PaymentAttempt{}, &domain.Invoice{}, &domain.Payment{}))
	body := []byte(`{"order_id":"not-known","status_code":"200","gross_amount":"1000.00","signature_key":"forged","transaction_status":"settlement"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/portal/payments/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	ctx := CreateTestContext(e, db, req, rec, appCtx)
	require.NoError(t, handlePaymentWebhook(ctx))
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
