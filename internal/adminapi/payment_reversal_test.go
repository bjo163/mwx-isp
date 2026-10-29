package adminapi

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type reversalHTTPFixture struct {
	handler       http.Handler
	db            *gorm.DB
	invoice       domain.Invoice
	adminToken    string
	operatorToken string
}

// newReversalHTTPFixture serves the real router so the route-level role guard is
// exercised, not just the service function.
func newReversalHTTPFixture(t *testing.T) reversalHTTPFixture {
	t.Helper()
	handler, db, appCtx := newRoleMatrixServer(t)
	require.NoError(t, db.AutoMigrate(&domain.SysOprLog{}))

	admin := seedMatrixOperator(t, db, 9401, LevelAdmin, "billing-admin")
	operator := seedMatrixOperator(t, db, 9402, LevelOperator, "billing-operator")
	customer := domain.Customer{Name: "Correction Customer", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	now := time.Now()
	invoice := domain.Invoice{
		InvoiceNo:      "INV-CORR-1",
		CustomerID:     customer.ID,
		SubscriptionID: 1,
		InvoiceDate:    now,
		DueDate:        now.AddDate(0, 0, 7),
		PeriodStart:    now,
		PeriodEnd:      now.AddDate(0, 1, 0),
		Subtotal:       100000,
		Total:          100000,
		Balance:        100000,
		Status:         domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)

	return reversalHTTPFixture{
		handler:       handler,
		db:            db,
		invoice:       invoice,
		adminToken:    matrixToken(t, appCtx, admin),
		operatorToken: matrixToken(t, appCtx, operator),
	}
}

func (f reversalHTTPFixture) recordPayment(t *testing.T) domain.Payment {
	t.Helper()
	status, body := matrixRequest(t, f.handler, http.MethodPost, "/api/v1/isp/payments", f.adminToken, map[string]any{
		"invoice_id": fmt.Sprintf("%d", f.invoice.ID),
		"amount":     100000,
		"method":     "cash",
	})
	require.Equalf(t, http.StatusCreated, status, "%s", body)
	var payment domain.Payment
	decodeMatrixData(t, body, &payment)
	return payment
}

func TestPaymentReversalRequiresAdminAndRecordsTrail(t *testing.T) {
	f := newReversalHTTPFixture(t)
	payment := f.recordPayment(t)

	var invoice domain.Invoice
	require.NoError(t, f.db.First(&invoice, f.invoice.ID).Error)
	require.Equal(t, domain.InvoicePaid, invoice.Status)

	// A regular operator cannot correct money movements.
	status, body := matrixRequest(t, f.handler, http.MethodPost,
		fmt.Sprintf("/api/v1/isp/payments/%d/reverse", payment.ID), f.operatorToken,
		map[string]string{"reason": "operator typed the wrong invoice"})
	require.Equalf(t, http.StatusForbidden, status, "%s", body)
	require.Equal(t, domain.InvoicePaid, loadReversalInvoice(t, f.db, f.invoice.ID).Status)
	require.Zero(t, loadReversalEventCount(t, f.db, "payment_reversed"))

	// A correction without a substantive reason is refused by an admin too.
	status, body = matrixRequest(t, f.handler, http.MethodPost,
		fmt.Sprintf("/api/v1/isp/payments/%d/reverse", payment.ID), f.adminToken,
		map[string]string{"reason": "oops"})
	require.Equalf(t, http.StatusBadRequest, status, "%s", body)
	require.Contains(t, string(body), "INVALID_CORRECTION_REASON")
	require.Equal(t, domain.InvoicePaid, loadReversalInvoice(t, f.db, f.invoice.ID).Status)

	// The admin correction reverses the ledger and leaves a dual trail.
	status, body = matrixRequest(t, f.handler, http.MethodPost,
		fmt.Sprintf("/api/v1/isp/payments/%d/reverse", payment.ID), f.adminToken,
		map[string]string{"reason": "operator typed the wrong invoice"})
	require.Equalf(t, http.StatusOK, status, "%s", body)
	var result struct {
		Payment domain.Payment `json:"payment"`
		Invoice domain.Invoice `json:"invoice"`
	}
	decodeMatrixData(t, body, &result)
	require.Equal(t, domain.PaymentReversed, result.Payment.Status)
	require.Equal(t, domain.InvoiceIssued, result.Invoice.Status)
	require.EqualValues(t, 100000, result.Invoice.Balance)
	require.EqualValues(t, 0, result.Invoice.PaidAmount)

	require.Equal(t, domain.InvoiceIssued, loadReversalInvoice(t, f.db, f.invoice.ID).Status)
	require.EqualValues(t, 1, loadReversalEventCount(t, f.db, "payment_reversed"))

	var trail domain.BillingEvent
	require.NoError(t, f.db.Where("type = ?", "payment_reversed").First(&trail).Error)
	require.Contains(t, trail.Description, payment.PaymentNo)
	require.Contains(t, trail.Description, "billing-admin")
	require.Contains(t, trail.Description, "operator typed the wrong invoice")

	var log domain.SysOprLog
	require.NoError(t, f.db.First(&log).Error)
	require.Equal(t, "billing-admin", log.OprName)
	require.NotEmpty(t, log.OprIp)
	require.Equal(t, "payment.reverse", log.OptAction)
	require.Contains(t, log.OptDesc, f.invoice.InvoiceNo)

	// Repeating the correction is refused and does not duplicate the trail.
	status, body = matrixRequest(t, f.handler, http.MethodPost,
		fmt.Sprintf("/api/v1/isp/payments/%d/reverse", payment.ID), f.adminToken,
		map[string]string{"reason": "operator typed the wrong invoice"})
	require.Equalf(t, http.StatusConflict, status, "%s", body)
	require.Contains(t, string(body), "PAYMENT_NOT_REVERSIBLE")
	require.EqualValues(t, 1, loadReversalEventCount(t, f.db, "payment_reversed"))
}

func TestPaymentReversalStaysInsideTheTenant(t *testing.T) {
	f := newReversalHTTPFixture(t)
	require.NoError(t, f.db.Create(&domain.Tenant{
		ID: 2, Name: "Other ISP", Slug: "other", Kind: "isp", Status: "active",
	}).Error)
	now := time.Now()
	foreignInvoice := domain.Invoice{
		TenantID: 2, InvoiceNo: "INV-OTHER-1", CustomerID: 9001, SubscriptionID: 9001,
		InvoiceDate: now, DueDate: now.AddDate(0, 0, 7), PeriodStart: now, PeriodEnd: now.AddDate(0, 1, 0),
		Subtotal: 50000, Total: 50000, Balance: 0, PaidAmount: 50000, Status: domain.InvoicePaid,
	}
	require.NoError(t, f.db.Create(&foreignInvoice).Error)
	foreignPayment := domain.Payment{
		TenantID: 2, PaymentNo: "PAY-OTHER-000001", InvoiceID: foreignInvoice.ID, CustomerID: 9001,
		Amount: 50000, Method: "cash", PaidAt: now, Status: domain.PaymentReceived,
	}
	require.NoError(t, f.db.Create(&foreignPayment).Error)

	status, body := matrixRequest(t, f.handler, http.MethodPost,
		fmt.Sprintf("/api/v1/isp/payments/%d/reverse", foreignPayment.ID), f.adminToken,
		map[string]string{"reason": "wrong tenant attempt"})
	require.Equalf(t, http.StatusNotFound, status, "%s", body)

	var stored domain.Payment
	require.NoError(t, f.db.First(&stored, foreignPayment.ID).Error)
	require.Equal(t, domain.PaymentReceived, stored.Status, "a cross-tenant correction must not touch the row")
	require.Zero(t, loadReversalEventCount(t, f.db, "payment_reversed"))
}

func loadReversalInvoice(t *testing.T, db *gorm.DB, id int64) domain.Invoice {
	t.Helper()
	var invoice domain.Invoice
	require.NoError(t, db.First(&invoice, id).Error)
	return invoice
}

func loadReversalEventCount(t *testing.T, db *gorm.DB, eventType string) int64 {
	t.Helper()
	var count int64
	require.NoError(t, db.Model(&domain.BillingEvent{}).Where("type = ?", eventType).Count(&count).Error)
	return count
}
