package adminapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/payment/midtrans"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

func midtransClient() (*midtrans.Client, midtrans.Config, error) {
	cfg, err := midtrans.ConfigFromEnv()
	if err != nil {
		return nil, cfg, err
	}
	client, err := midtrans.New(cfg, nil)
	return client, cfg, err
}

// getInvoicePaymentChannel creates a checkout from the server-side invoice balance.
// The browser cannot choose or override the amount.
func getInvoicePaymentChannel(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid invoice ID", nil)
	}
	idempotencyKey := strings.TrimSpace(c.Request().Header.Get("Idempotency-Key"))
	if !validIdempotencyKey(idempotencyKey) {
		return fail(c, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "A valid Idempotency-Key header is required", nil)
	}
	client, providerCfg, err := midtransClient()
	if err != nil {
		if errors.Is(err, midtrans.ErrNotConfigured) {
			return fail(c, http.StatusServiceUnavailable, "PAYMENT_PROVIDER_UNCONFIGURED",
				"Online payments are not configured. Ask an administrator to configure Midtrans.", nil)
		}
		return fail(c, http.StatusServiceUnavailable, "PAYMENT_PROVIDER_INVALID", "The online payment provider configuration is invalid", nil)
	}
	configuredCurrency := strings.TrimSpace(GetAppContext(c).GetSettingsStringValue("isp", "Currency"))
	if configuredCurrency != "" && !strings.EqualFold(configuredCurrency, "IDR") {
		return fail(c, http.StatusConflict, "PAYMENT_CURRENCY_UNSUPPORTED", "Midtrans checkout is currently enabled only for IDR invoices", nil)
	}

	db := GetDB(c)
	var invoice domain.Invoice
	if err := db.First(&invoice, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "INVOICE_NOT_FOUND", "Invoice not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load invoice", nil)
	}
	if invoice.Balance <= 0 || invoice.Status == domain.InvoiceVoid || invoice.Status == domain.InvoicePaid {
		return fail(c, http.StatusConflict, "INVOICE_NOT_PAYABLE", "This invoice has no payable balance", nil)
	}

	var previous domain.PaymentAttempt
	err = db.Where("provider = ? AND idempotency_key = ?", "midtrans", idempotencyKey).First(&previous).Error
	if err == nil {
		if previous.InvoiceID != invoice.ID || previous.Amount != invoice.Balance {
			return fail(c, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "This Idempotency-Key is already bound to different invoice details", nil)
		}
		if previous.Status == "pending" && previous.RedirectURL != "" {
			return ok(c, map[string]any{
				"provider": "midtrans", "order_id": previous.OrderID, "amount": previous.Amount,
				"currency": previous.Currency, "status": previous.Status,
				"redirect_url": previous.RedirectURL, "client_key": providerCfg.ClientKey,
				"replayed": true,
			})
		}
		if previous.Status == "creating" {
			return fail(c, http.StatusConflict, "CHECKOUT_IN_PROGRESS", "Checkout creation is already in progress; retry shortly", nil)
		}
		return fail(c, http.StatusConflict, "CHECKOUT_NOT_REUSABLE", "This checkout attempt cannot be reused; start a new payment attempt", nil)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to check previous checkout", nil)
	}

	var customer domain.Customer
	if err := db.First(&customer, invoice.CustomerID).Error; err != nil {
		return fail(c, http.StatusConflict, "CUSTOMER_NOT_AVAILABLE", "The invoice customer record is unavailable", nil)
	}
	now := time.Now()
	orderID := fmt.Sprintf("MWX-%s-%s-%s", strconv.FormatInt(invoice.TenantID, 36), strconv.FormatInt(invoice.ID, 36), strconv.FormatInt(now.UnixNano(), 36))
	attempt := domain.PaymentAttempt{
		InvoiceID: invoice.ID, CustomerID: invoice.CustomerID,
		Provider: "midtrans", OrderID: orderID, IdempotencyKey: &idempotencyKey,
		Amount: invoice.Balance, Currency: "IDR", Status: "creating",
		CreatedAt: now, UpdatedAt: now,
	}
	if err := db.Create(&attempt).Error; err != nil {
		return fail(c, http.StatusConflict, "CHECKOUT_CREATE_CONFLICT", "A checkout attempt with this key already exists; retry with the same key", nil)
	}

	checkout, err := client.CreateCheckout(c.Request().Context(), orderID, invoice.Balance, midtrans.Customer{
		Name: customer.Name, Email: customer.Email, Phone: customer.Phone,
	})
	if err != nil {
		_ = db.Model(&attempt).Updates(map[string]any{
			"status": "failed", "failure_reason": "provider_checkout_failed", "updated_at": time.Now(),
		}).Error
		return fail(c, http.StatusBadGateway, "PAYMENT_PROVIDER_ERROR", "The payment provider could not create a checkout", nil)
	}
	if err := db.Model(&attempt).Updates(map[string]any{
		"status": "pending", "snap_token": checkout.Token,
		"redirect_url": checkout.RedirectURL, "updated_at": time.Now(),
	}).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "CHECKOUT_PERSIST_FAILED", "Checkout was created but could not be saved; contact support before retrying", nil)
	}

	return c.JSON(http.StatusCreated, Response{Data: map[string]any{
		"provider": "midtrans", "order_id": orderID, "amount": invoice.Balance,
		"currency": "IDR", "status": "pending", "redirect_url": checkout.RedirectURL,
		"client_key": providerCfg.ClientKey, "replayed": false,
	}})
}

func handlePaymentWebhook(c echo.Context) error {
	_, providerCfg, err := midtransClient()
	if err != nil {
		if errors.Is(err, midtrans.ErrNotConfigured) {
			return fail(c, http.StatusServiceUnavailable, "PAYMENT_PROVIDER_UNCONFIGURED", "Payment callbacks are not configured", nil)
		}
		return fail(c, http.StatusServiceUnavailable, "PAYMENT_PROVIDER_INVALID", "Payment provider configuration is invalid", nil)
	}
	var notification midtrans.Notification
	decoder := json.NewDecoder(io.LimitReader(c.Request().Body, 64*1024))
	if err := decoder.Decode(&notification); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_NOTIFICATION", "Invalid payment notification payload", nil)
	}
	if !midtrans.VerifyNotification(providerCfg.ServerKey, notification) {
		return fail(c, http.StatusUnauthorized, "INVALID_PAYMENT_SIGNATURE", "Payment notification signature is invalid", nil)
	}
	amount, err := midtrans.ParseGrossAmount(notification.GrossAmount)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_PAYMENT_AMOUNT", "Payment notification amount is invalid", nil)
	}
	if notification.Currency != "" && !strings.EqualFold(notification.Currency, "IDR") {
		return fail(c, http.StatusBadRequest, "INVALID_PAYMENT_CURRENCY", "Only IDR payment notifications are accepted", nil)
	}

	// Resolve the immutable order ID before applying a tenant-scoped ledger mutation.
	unscoped := GetAppContext(c).DB()
	var attempt domain.PaymentAttempt
	if err := unscoped.Where("provider = ? AND order_id = ?", "midtrans", notification.OrderID).First(&attempt).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "PAYMENT_ATTEMPT_NOT_FOUND", "Unknown payment order", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load payment attempt", nil)
	}
	if amount != attempt.Amount || attempt.Currency != "IDR" {
		return fail(c, http.StatusBadRequest, "PAYMENT_DETAILS_MISMATCH", "Payment notification does not match the stored invoice amount or currency", nil)
	}
	db := unscoped.WithContext(tenancy.WithTenantID(c.Request().Context(), attempt.TenantID))
	now := time.Now()
	status := strings.ToLower(strings.TrimSpace(notification.TransactionStatus))
	updates := map[string]any{
		"raw_status": status, "provider_transaction_id": notification.TransactionID,
		"updated_at": now,
	}

	// Once settlement has been booked, older pending/deny callbacks must not roll it back.
	if attempt.PaymentID > 0 && (attempt.Status == "paid" || attempt.Status == "refunded" || attempt.Status == "chargeback") {
		switch status {
		case "settlement", "capture", "pending", "deny", "cancel", "expire":
			return ok(c, map[string]any{"received": true, "status": attempt.Status, "duplicate_or_stale": true})
		}
	}

	switch status {
	case "settlement", "capture":
		if status == "capture" && !strings.EqualFold(notification.FraudStatus, "accept") {
			updates["status"] = "pending"
			if err := db.Model(&attempt).Updates(updates).Error; err != nil {
				return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to update payment attempt", nil)
			}
			return ok(c, map[string]any{"received": true, "status": "pending"})
		}
		if attempt.PaymentID == 0 {
			payment := domain.Payment{
				InvoiceID: attempt.InvoiceID, Amount: attempt.Amount, Method: "midtrans",
				Reference: notification.TransactionID,
				Notes: "Midtrans Snap transaction " + notification.TransactionID,
				IdempotencyKey: &attempt.OrderID,
			}
			err := billing.RecordPayment(db, &payment, now, GetAppContext(c).GetSettingsBoolValue("isp", "AutoReactivate"))
			switch {
			case err == nil:
				attempt.PaymentID = payment.ID
			case errors.Is(err, billing.ErrDuplicatePayment):
				var existing domain.Payment
				if loadErr := db.Where("idempotency_key = ?", attempt.OrderID).First(&existing).Error; loadErr != nil {
					return fail(c, http.StatusInternalServerError, "PAYMENT_LEDGER_ERROR", "Payment was already recorded but its ledger entry could not be loaded", nil)
				}
				attempt.PaymentID = existing.ID
			case errors.Is(err, billing.ErrInvalidPayment), errors.Is(err, billing.ErrInvoiceClosed):
				updates["status"] = "reconciliation_required"
				updates["failure_reason"] = "invoice_changed_before_provider_settlement"
				if saveErr := db.Model(&attempt).Updates(updates).Error; saveErr != nil {
					return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to record payment reconciliation state", nil)
				}
				return ok(c, map[string]any{"received": true, "status": "reconciliation_required"})
			default:
				return fail(c, http.StatusInternalServerError, "PAYMENT_LEDGER_ERROR", "Unable to record settled payment", nil)
			}
		}
		updates["status"] = "paid"
		updates["payment_id"] = attempt.PaymentID

	case "pending":
		updates["status"] = "pending"
	case "deny", "cancel", "expire":
		updates["status"] = status
	case "refund", "chargeback":
		if attempt.PaymentID <= 0 {
			updates["status"] = "reconciliation_required"
			updates["failure_reason"] = "provider_reversal_without_local_payment"
			break
		}
		reason := fmt.Sprintf("Midtrans %s transaction %s", status, notification.TransactionID)
		_, _, reverseErr := billing.ReversePayment(db, attempt.PaymentID, billing.PaymentCorrection{Operator: "midtrans-webhook", Reason: reason}, now)
		if reverseErr != nil && !errors.Is(reverseErr, billing.ErrPaymentNotReversible) {
			return fail(c, http.StatusInternalServerError, "PAYMENT_REVERSAL_ERROR", "Unable to reconcile the provider reversal", nil)
		}
		if status == "refund" {
			updates["status"] = "refunded"
		} else {
			updates["status"] = "chargeback"
		}
	case "partial_refund", "partial_chargeback":
		updates["status"] = "reconciliation_required"
		updates["failure_reason"] = "partial_provider_reversal_requires_manual_reconciliation"
	default:
		updates["status"] = "reconciliation_required"
		updates["failure_reason"] = "unrecognized_provider_transaction_status"
	}

	if err := db.Model(&attempt).Updates(updates).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to persist provider notification", nil)
	}
	return ok(c, map[string]any{"received": true, "status": updates["status"]})
}
