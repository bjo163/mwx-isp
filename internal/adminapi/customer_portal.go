package adminapi

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/customerportal"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/notify"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	customerOTPWindow = 15 * time.Minute
	customerOTPTTL = 5 * time.Minute
	customerOTPMaxAttempts = 5
	customerOTPMaxRequests = 3
	customerPortalSessionTTL = 12 * time.Hour
)

var (
	errCustomerOTPInvalid = errors.New("customer verification failed")
	customerPhoneChars = regexp.MustCompile("[^0-9]")
)

type customerOTPRequest struct {
	TenantSlug string `json:"tenant_slug"`
	CustomerNo string `json:"customer_no"`
	Phone string `json:"phone"`
}

type customerOTPVerifyRequest struct {
	TenantSlug string `json:"tenant_slug"`
	ChallengeID string `json:"challenge_id"`
	Code string `json:"code"`
}

func customerPortalSecret(c echo.Context) string {
	return GetAppContext(c).Config().Web.Secret
}

func normalizedCustomerPhone(raw string) string {
	digits := customerPhoneChars.ReplaceAllString(strings.TrimSpace(raw), "")
	if strings.HasPrefix(digits, "0") {
		digits = "62" + strings.TrimLeft(digits, "0")
	} else if strings.HasPrefix(digits, "8") {
		digits = "62" + digits
	}
	if len(digits) < 8 || len(digits) > 15 || strings.HasPrefix(digits, "0") {
		return ""
	}
	return digits
}

func customerPortalTenant(c echo.Context, slug string) (domain.Tenant, error) {
	var tenant domain.Tenant
	err := GetAppContext(c).DB().Where("slug = ? AND status = ?", strings.ToLower(strings.TrimSpace(slug)), "active").First(&tenant).Error
	return tenant, err
}

// requestCustomerOTP intentionally returns the same response shape whether or
// not an account matches. This prevents unauthenticated customer enumeration.
func requestCustomerOTP(c echo.Context) error {
	var in customerOTPRequest
	if err := c.Bind(&in); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_REQUEST", "Invalid verification request", nil)
	}
	challengeID, err := customerportal.GenerateOpaqueToken()
	if err != nil {
		return fail(c, http.StatusInternalServerError, "OTP_UNAVAILABLE", "Customer verification is temporarily unavailable", nil)
	}
	generic := map[string]any{
		"challenge_id": challengeID,
		"message": "Jika data cocok dengan pelanggan aktif, kode verifikasi akan dikirim ke WhatsApp terdaftar.",
		"expires_in_seconds": int(customerOTPTTL.Seconds()),
	}
	phone := normalizedCustomerPhone(in.Phone)
	if strings.TrimSpace(in.TenantSlug) == "" || strings.TrimSpace(in.CustomerNo) == "" || phone == "" {
		return ok(c, generic)
	}
	tenant, err := customerPortalTenant(c, in.TenantSlug)
	if err != nil {
		return ok(c, generic)
	}
	setRequestTenant(c, tenant.ID)
	db := GetDB(c)
	var customer domain.Customer
	if err := db.Where("customer_no = ? AND status = ?", strings.TrimSpace(in.CustomerNo), domain.CustomerActive).First(&customer).Error; err != nil {
		return ok(c, generic)
	}
	registeredPhone := normalizedCustomerPhone(customer.Phone)
	if registeredPhone == "" || registeredPhone != phone {
		return ok(c, generic)
	}

	secret := customerPortalSecret(c)
	phoneHash := customerportal.HashIdentifier(secret, "customer-phone-v1", phone)
	ipHash := customerportal.HashIdentifier(secret, "customer-ip-v1", c.RealIP())
	var recent int64
	if err := db.Model(&domain.CustomerAuthChallenge{}).
		Where("(phone_hash = ? OR ip_hash = ?) AND created_at >= ?", phoneHash, ipHash, time.Now().Add(-customerOTPWindow)).
		Count(&recent).Error; err != nil {
		return ok(c, generic)
	}
	if recent >= customerOTPMaxRequests {
		return ok(c, generic)
	}

	otp, err := customerportal.GenerateOTP()
	if err != nil {
		return ok(c, generic)
	}
	now := time.Now()
	challenge := domain.CustomerAuthChallenge{
		CustomerID: customer.ID, ChallengeID: challengeID,
		PhoneHash: phoneHash, IPHash: ipHash,
		CodeHash: customerportal.HashOTP(secret, challengeID, otp),
		ExpiresAt: now.Add(customerOTPTTL), CreatedAt: now,
	}
	if err := db.Create(&challenge).Error; err != nil {
		return ok(c, generic)
	}

	provider, ok := GetAppContext(c).(interface {
		WhatsAppManager() (*notify.WhatsAppManager, error)
	})
	if !ok {
		_ = db.Model(&challenge).Update("consumed_at", now).Error
		return ok(c, generic)
	}
	manager, err := provider.WhatsAppManager()
	if err != nil || manager == nil {
		_ = db.Model(&challenge).Update("consumed_at", now).Error
		return ok(c, generic)
	}
	message := fmt.Sprintf("Kode verifikasi MWX-ISP Anda: %s. Berlaku 5 menit. Jangan bagikan kode ini kepada siapa pun.", otp)
	if err := manager.Send(c.Request().Context(), phone, message); err != nil {
		_ = db.Model(&challenge).Update("consumed_at", now).Error
	}
	return ok(c, generic)
}

func verifyCustomerOTP(c echo.Context) error {
	var in customerOTPVerifyRequest
	if err := c.Bind(&in); err != nil {
		return fail(c, http.StatusUnauthorized, "INVALID_CUSTOMER_SESSION", "Kode verifikasi tidak valid atau kedaluwarsa", nil)
	}
	if len(in.Code) != 6 || strings.TrimSpace(in.ChallengeID) == "" {
		return fail(c, http.StatusUnauthorized, "INVALID_CUSTOMER_SESSION", "Kode verifikasi tidak valid atau kedaluwarsa", nil)
	}
	tenant, err := customerPortalTenant(c, in.TenantSlug)
	if err != nil {
		return fail(c, http.StatusUnauthorized, "INVALID_CUSTOMER_SESSION", "Kode verifikasi tidak valid atau kedaluwarsa", nil)
	}
	setRequestTenant(c, tenant.ID)
	db := GetDB(c)
	secret := customerPortalSecret(c)
	token, err := customerportal.GenerateOpaqueToken()
	if err != nil {
		return fail(c, http.StatusInternalServerError, "SESSION_UNAVAILABLE", "Customer session is temporarily unavailable", nil)
	}
	now := time.Now()
	session := domain.CustomerPortalSession{
		TokenHash: customerportal.HashToken(secret, token),
		ExpiresAt: now.Add(customerPortalSessionTTL), CreatedAt: now,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var challenge domain.CustomerAuthChallenge
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("challenge_id = ?", strings.TrimSpace(in.ChallengeID)).First(&challenge).Error; err != nil {
			return errCustomerOTPInvalid
		}
		if challenge.ConsumedAt != nil || !challenge.ExpiresAt.After(now) || challenge.Attempts >= customerOTPMaxAttempts {
			return errCustomerOTPInvalid
		}
		if !customerportal.VerifyOTP(secret, challenge.ChallengeID, in.Code, challenge.CodeHash) {
			if err := tx.Model(&challenge).Update("attempts", challenge.Attempts+1).Error; err != nil {
				return err
			}
			return errCustomerOTPInvalid
		}
		var customer domain.Customer
		if err := tx.Where("id = ? AND status = ?", challenge.CustomerID, domain.CustomerActive).First(&customer).Error; err != nil {
			return errCustomerOTPInvalid
		}
		consumed := now
		if err := tx.Model(&challenge).Update("consumed_at", consumed).Error; err != nil {
			return err
		}
		session.TenantID = challenge.TenantID
		session.CustomerID = customer.ID
		return tx.Create(&session).Error
	})
	if err != nil {
		if errors.Is(err, errCustomerOTPInvalid) || errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusUnauthorized, "INVALID_CUSTOMER_SESSION", "Kode verifikasi tidak valid atau kedaluwarsa", nil)
		}
		return fail(c, http.StatusInternalServerError, "SESSION_CREATE_FAILED", "Unable to create customer session", nil)
	}
	return c.JSON(http.StatusOK, Response{Data: map[string]any{
		"access_token": token,
		"token_type": "Bearer",
		"expires_at": session.ExpiresAt,
	}})
}

// customerPortalAuth validates the customer bearer token independently from
// the staff JWT middleware and installs the token's tenant/customer context.
func customerPortalAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			header := strings.TrimSpace(c.Request().Header.Get(echo.HeaderAuthorization))
			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
				return fail(c, http.StatusUnauthorized, "CUSTOMER_AUTH_REQUIRED", "Customer sign-in is required", nil)
			}
			appCtx := GetAppContext(c)
			tokenHash := customerportal.HashToken(appCtx.Config().Web.Secret, strings.TrimSpace(parts[1]))
			var session domain.CustomerPortalSession
			err := appCtx.DB().Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).First(&session).Error
			if err != nil {
				return fail(c, http.StatusUnauthorized, "INVALID_CUSTOMER_SESSION", "Customer session is invalid or expired", nil)
			}
			setRequestTenant(c, session.TenantID)
			c.Set("customer_id", session.CustomerID)
			c.Set("customer_session_id", session.ID)
			return next(c)
		}
	}
}

func getCustomerSelfCare(c echo.Context) error {
	customerID, ok := c.Get("customer_id").(int64)
	if !ok || customerID <= 0 {
		return fail(c, http.StatusUnauthorized, "CUSTOMER_AUTH_REQUIRED", "Customer sign-in is required", nil)
	}
	db := GetDB(c)
	var customer domain.Customer
	if err := db.First(&customer, customerID).Error; err != nil {
		return fail(c, http.StatusUnauthorized, "CUSTOMER_NOT_AVAILABLE", "Customer account is unavailable", nil)
	}
	var subscriptions []domain.Subscription
	if err := db.Where("customer_id = ?", customerID).Order("id DESC").Limit(10).Find(&subscriptions).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load subscriptions", nil)
	}
	type subscriptionView struct {
		SubscriptionNo string `json:"subscription_no"`
		Status string `json:"status"`
		StartDate time.Time `json:"start_date"`
		BillingDay int `json:"billing_day"`
		PackageName string `json:"package_name"`
		PackagePrice int64 `json:"package_price"`
	}
	subscriptionViews := make([]subscriptionView, 0, len(subscriptions))
	for _, subscription := range subscriptions {
		view := subscriptionView{
			SubscriptionNo: subscription.SubscriptionNo,
			Status: subscription.Status,
			StartDate: subscription.StartDate,
			BillingDay: subscription.BillingDay,
		}
		var pkg domain.InternetPackage
		if err := db.Select("name", "price").First(&pkg, subscription.PackageID).Error; err == nil {
			view.PackageName = pkg.Name
			view.PackagePrice = pkg.Price
		}
		subscriptionViews = append(subscriptionViews, view)
	}

	var invoices []domain.Invoice
	if err := db.Where("customer_id = ?", customerID).Order("invoice_date DESC").Limit(24).Find(&invoices).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load invoices", nil)
	}
	type paymentView struct {
		PaymentNo string `json:"payment_no"`
		InvoiceID int64 `json:"invoice_id,string"`
		Amount int64 `json:"amount"`
		Method string `json:"method"`
		PaidAt time.Time `json:"paid_at"`
		Status string `json:"status"`
	}
	var storedPayments []domain.Payment
	if err := db.Where("customer_id = ?", customerID).Order("paid_at DESC").Limit(24).Find(&storedPayments).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load payments", nil)
	}
	payments := make([]paymentView, 0, len(storedPayments))
	for _, payment := range storedPayments {
		payments = append(payments, paymentView{
			PaymentNo: payment.PaymentNo, InvoiceID: payment.InvoiceID,
			Amount: payment.Amount, Method: payment.Method,
			PaidAt: payment.PaidAt, Status: payment.Status,
		})
	}

	type ticketView struct {
		TicketNo string `json:"ticket_no"`
		Subject string `json:"subject"`
		Category string `json:"category"`
		Priority string `json:"priority"`
		Status string `json:"status"`
		CreatedAt time.Time `json:"created_at"`
	}
	var storedTickets []domain.TroubleTicket
	if err := db.Where("customer_id = ?", customerID).Order("created_at DESC").Limit(20).Find(&storedTickets).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to load support tickets", nil)
	}
	tickets := make([]ticketView, 0, len(storedTickets))
	for _, ticket := range storedTickets {
		tickets = append(tickets, ticketView{
			TicketNo: ticket.TicketNo, Subject: ticket.Subject,
			Category: ticket.Category, Priority: ticket.Priority,
			Status: ticket.Status, CreatedAt: ticket.CreatedAt,
		})
	}

	return ok(c, map[string]any{
		"customer": map[string]any{
			"customer_no": customer.CustomerNo,
			"name": customer.Name,
			"phone": customer.Phone,
			"email": customer.Email,
			"address": customer.Address,
			"status": customer.Status,
		},
		"subscriptions": subscriptionViews,
		"invoices": invoices,
		"payments": payments,
		"tickets": tickets,
	})
}

func logoutCustomerPortal(c echo.Context) error {
	sessionID, ok := c.Get("customer_session_id").(int64)
	if !ok || sessionID <= 0 {
		return fail(c, http.StatusUnauthorized, "CUSTOMER_AUTH_REQUIRED", "Customer sign-in is required", nil)
	}
	now := time.Now()
	if err := GetDB(c).Model(&domain.CustomerPortalSession{}).Where("id = ? AND revoked_at IS NULL", sessionID).Update("revoked_at", now).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Unable to end customer session", nil)
	}
	return ok(c, map[string]bool{"logged_out": true})
}

