package adminapi

import (
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/invoicedoc"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const tenantAssetURLPrefix = "/api/v1/tenant-branding/assets/"

var (
	bankAccountNumberPattern = regexp.MustCompile(`^[0-9][0-9 .-]{3,33}$`)
	tenantAssetNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,120}$`)
)

type tenantDocumentTemplateInput struct {
	IssuerName        *string `json:"issuer_name"`
	LogoURL           *string `json:"logo_url"`
	HeaderText        *string `json:"header_text"`
	FooterText        *string `json:"footer_text"`
	AccentColor       *string `json:"accent_color"`
	PaperSize         *string `json:"paper_size"`
	ShowTaxID         *bool   `json:"show_tax_id"`
	BankName          *string `json:"bank_name"`
	BankAccountName   *string `json:"bank_account_name"`
	BankAccountNumber *string `json:"bank_account_number"`
	PaymentNote       *string `json:"payment_note"`
	QRImageURL        *string `json:"qr_image_url"`
}

func tenantDocumentTenantID(c echo.Context) (int64, error) {
	id, ok := c.Get("tenant_id").(int64)
	if !ok || id <= 0 {
		return 0, fmt.Errorf("tenant context is required")
	}
	return id, nil
}

func defaultTenantDocumentTemplate(tenantID int64) domain.TenantDocumentTemplate {
	template := domain.TenantDocumentTemplate{
		TenantID: tenantID, Version: 1, PaperSize: "A4", ShowTaxID: true, AccentColor: "#16A34A",
	}
	return template
}

func loadTenantDocumentTemplate(db *gorm.DB, tenantID int64) (domain.TenantDocumentTemplate, bool, error) {
	var template domain.TenantDocumentTemplate
	err := db.Where("tenant_id = ?", tenantID).First(&template).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultTenantDocumentTemplate(tenantID), false, nil
	}
	if err != nil {
		return domain.TenantDocumentTemplate{}, false, err
	}
	return template, true, nil
}

func getTenantDocumentTemplateHandler(c echo.Context) error {
	tenantID, err := tenantDocumentTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	template, _, err := loadTenantDocumentTemplate(GetDB(c), tenantID)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load document template", err.Error())
	}
	return ok(c, template)
}

func validateTenantDocumentAsset(c echo.Context, tenantID int64, raw string, allowExternal bool) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", nil
	}
	if strings.HasPrefix(value, "/") {
		rest, found := strings.CutPrefix(value, tenantAssetURLPrefix)
		if !found {
			return "", errors.New("asset must be an uploaded tenant branding asset")
		}
		parts := strings.Split(rest, "/")
		if len(parts) != 2 {
			return "", errors.New("asset path is not a tenant branding asset")
		}
		assetTenant, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || assetTenant != tenantID {
			return "", errors.New("asset belongs to another tenant")
		}
		name := parts[1]
		if !tenantAssetNamePattern.MatchString(name) || filepath.Base(name) != name {
			return "", errors.New("asset name is invalid")
		}
		path := filepath.Join(GetAppContext(c).Config().GetDataDir(), "tenant-branding", strconv.FormatInt(tenantID, 10), name)
		// #nosec G304 -- tenantID is the authenticated numeric tenant and name matched the asset pattern plus filepath.Base, so the join cannot leave the tenant's branding directory.
		file, err := os.Open(path)
		if err != nil {
			return "", errors.New("asset file does not exist")
		}
		defer func() { _ = file.Close() }()
		info, err := file.Stat()
		if err != nil || info.Size() > 5<<20 {
			return "", errors.New("asset file is too large")
		}
		config, format, err := image.DecodeConfig(file)
		if err != nil || (format != "png" && format != "jpeg") {
			return "", errors.New("asset must be a PNG or JPEG image")
		}
		if config.Width <= 0 || config.Height <= 0 || config.Width > 4096 || config.Height > 4096 {
			return "", errors.New("asset dimensions are not supported")
		}
		return value, nil
	}
	if allowExternal && validTenantBrandingAssetURL(value) {
		return value, nil
	}
	return "", errors.New("asset must be an uploaded tenant branding asset")
}

func validateTenantDocumentTemplate(c echo.Context, tenantID int64, input tenantDocumentTemplateInput) error {
	limits := []struct {
		name  string
		value *string
		max   int
	}{
		{"issuer_name", input.IssuerName, 150}, {"logo_url", input.LogoURL, 500},
		{"header_text", input.HeaderText, 300}, {"footer_text", input.FooterText, 300},
		{"bank_name", input.BankName, 80}, {"bank_account_name", input.BankAccountName, 120},
		{"bank_account_number", input.BankAccountNumber, 40}, {"payment_note", input.PaymentNote, 300},
		{"qr_image_url", input.QRImageURL, 500},
	}
	for _, field := range limits {
		if field.value != nil && len(strings.TrimSpace(*field.value)) > field.max {
			return fmt.Errorf("%s must not exceed %d characters", field.name, field.max)
		}
	}
	if input.AccentColor != nil && strings.TrimSpace(*input.AccentColor) != "" &&
		!brandingAccentPattern.MatchString(strings.TrimSpace(*input.AccentColor)) {
		return errors.New("accent_color must be a six-digit hexadecimal value")
	}
	if input.PaperSize != nil {
		paper := strings.ToUpper(strings.TrimSpace(*input.PaperSize))
		if paper != "A4" && paper != "LETTER" {
			return errors.New("paper_size must be A4 or Letter")
		}
	}
	if input.LogoURL != nil {
		if _, err := validateTenantDocumentAsset(c, tenantID, *input.LogoURL, true); err != nil {
			return fmt.Errorf("logo_url: %w", err)
		}
	}
	if input.QRImageURL != nil {
		if _, err := validateTenantDocumentAsset(c, tenantID, *input.QRImageURL, false); err != nil {
			return fmt.Errorf("qr_image_url: %w", err)
		}
	}
	if input.BankAccountNumber != nil && strings.TrimSpace(*input.BankAccountNumber) != "" &&
		!bankAccountNumberPattern.MatchString(strings.TrimSpace(*input.BankAccountNumber)) {
		return errors.New("bank_account_number must contain only digits, spaces, dots, or dashes")
	}
	return nil
}

func applyTenantDocumentTemplate(template *domain.TenantDocumentTemplate, input tenantDocumentTemplateInput) {
	set := func(target *string, value *string) {
		if value != nil {
			*target = strings.TrimSpace(*value)
		}
	}
	set(&template.IssuerName, input.IssuerName)
	set(&template.LogoURL, input.LogoURL)
	set(&template.HeaderText, input.HeaderText)
	set(&template.FooterText, input.FooterText)
	set(&template.AccentColor, input.AccentColor)
	set(&template.PaymentNote, input.PaymentNote)
	set(&template.QRImageURL, input.QRImageURL)
	set(&template.BankName, input.BankName)
	set(&template.BankAccountName, input.BankAccountName)
	set(&template.BankAccountNumber, input.BankAccountNumber)
	if input.PaperSize != nil {
		template.PaperSize = strings.ToUpper(strings.TrimSpace(*input.PaperSize))
	}
	if input.ShowTaxID != nil {
		template.ShowTaxID = *input.ShowTaxID
	}
	if template.AccentColor != "" {
		template.AccentColor = strings.ToUpper(template.AccentColor)
	}
	if template.PaperSize == "" {
		template.PaperSize = "A4"
	}
}

func validatePaymentInstructions(template domain.TenantDocumentTemplate) error {
	if template.BankAccountNumber == "" && template.BankName == "" && template.BankAccountName == "" {
		return nil
	}
	if template.BankName == "" || template.BankAccountName == "" || template.BankAccountNumber == "" {
		return errors.New("bank_name, bank_account_name, and bank_account_number must be provided together")
	}
	return nil
}

func updateTenantDocumentTemplateHandler(c echo.Context) error {
	tenantID, err := tenantDocumentTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	var input tenantDocumentTemplateInput
	if err := c.Bind(&input); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_REQUEST", "Unable to parse document template", nil)
	}
	if err := validateTenantDocumentTemplate(c, tenantID, input); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_DOCUMENT_TEMPLATE", err.Error(), nil)
	}
	db := GetDB(c)
	template, exists, err := loadTenantDocumentTemplate(db, tenantID)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load document template", err.Error())
	}
	if !exists {
		template = defaultTenantDocumentTemplate(tenantID)
	}
	applyTenantDocumentTemplate(&template, input)
	if err := validatePaymentInstructions(template); err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_DOCUMENT_TEMPLATE", err.Error(), nil)
	}
	if !exists {
		template.Version = 1
	} else {
		template.Version++
	}
	template.UpdatedAt = time.Now().UTC()
	if template.ID == 0 {
		template.CreatedAt = template.UpdatedAt
		if err := db.Create(&template).Error; err != nil {
			return fail(c, http.StatusConflict, "DOCUMENT_TEMPLATE_SAVE_FAILED", "Failed to create document template", err.Error())
		}
	} else if err := db.Save(&template).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DOCUMENT_TEMPLATE_SAVE_FAILED", "Failed to update document template", err.Error())
	}
	return ok(c, template)
}

func resetTenantDocumentTemplateHandler(c echo.Context) error {
	tenantID, err := tenantDocumentTenantID(c)
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_TENANT", err.Error(), nil)
	}
	db := GetDB(c)
	template, exists, err := loadTenantDocumentTemplate(db, tenantID)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load document template", err.Error())
	}
	version := template.Version
	defaults := defaultTenantDocumentTemplate(tenantID)
	if exists {
		defaults.ID = template.ID
		defaults.Version = version + 1
		defaults.CreatedAt = template.CreatedAt
	}
	defaults.UpdatedAt = time.Now().UTC()
	if defaults.CreatedAt.IsZero() {
		defaults.CreatedAt = defaults.UpdatedAt
	}
	if defaults.ID == 0 {
		if err := db.Create(&defaults).Error; err != nil {
			return fail(c, http.StatusConflict, "DOCUMENT_TEMPLATE_SAVE_FAILED", "Failed to reset document template", err.Error())
		}
	} else if err := db.Save(&defaults).Error; err != nil {
		return fail(c, http.StatusInternalServerError, "DOCUMENT_TEMPLATE_SAVE_FAILED", "Failed to reset document template", err.Error())
	}
	return ok(c, defaults)
}

type invoiceDocumentIssuer struct {
	Name      string `json:"name"`
	TaxID     string `json:"tax_id"`
	Address   string `json:"address"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	ShowTaxID bool   `json:"show_tax_id"`
}

type invoiceDocumentPaymentInstructions struct {
	BankName          string `json:"bank_name"`
	BankAccountName   string `json:"bank_account_name"`
	BankAccountNumber string `json:"bank_account_number"`
	Note              string `json:"note"`
}

type invoiceDocumentCustomer struct {
	Name  string `json:"name"`
	No    string `json:"no"`
	Phone string `json:"phone"`
}

type invoiceDocumentResponse struct {
	Kind                string                            `json:"kind"`
	Number              string                            `json:"number"`
	Status              string                            `json:"status"`
	Currency            string                            `json:"currency"`
	IssuedAt            time.Time                         `json:"issued_at"`
	DueAt               time.Time                         `json:"due_at"`
	PeriodStart         time.Time                         `json:"period_start"`
	PeriodEnd           time.Time                         `json:"period_end"`
	Issuer              invoiceDocumentIssuer             `json:"issuer"`
	BrandName           string                            `json:"brand_name"`
	AccentColor         string                            `json:"accent_color"`
	PaperSize           string                            `json:"paper_size"`
	HeaderText          string                            `json:"header_text"`
	FooterText          string                            `json:"footer_text"`
	LogoURL             string                            `json:"logo_url"`
	PaymentInstructions invoiceDocumentPaymentInstructions `json:"payment_instructions"`
	QRImageURL          string                            `json:"qr_image_url"`
	Customer            invoiceDocumentCustomer           `json:"customer"`
	Items               []domain.InvoiceItem              `json:"items"`
	Payments            []domain.Payment                  `json:"payments"`
	Subtotal            int64                             `json:"subtotal"`
	Total               int64                             `json:"total"`
	PaidAmount          int64                             `json:"paid_amount"`
	Balance             int64                             `json:"balance"`
	Snapshot            bool                              `json:"snapshot"`
	TemplateVersion     int                               `json:"template_version"`
	CapturedAt          *time.Time                        `json:"captured_at,omitempty"`
}

func loadInvoiceSnapshot(db *gorm.DB, invoice *domain.Invoice) (domain.InvoiceDocumentSnapshot, bool, error) {
	var snapshot domain.InvoiceDocumentSnapshot
	err := db.Where("tenant_id = ? AND invoice_id = ?", invoice.TenantID, invoice.ID).First(&snapshot).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		live, buildErr := billing.BuildInvoiceDocumentSnapshot(db, invoice, time.Now())
		return live, false, buildErr
	}
	if err != nil {
		return domain.InvoiceDocumentSnapshot{}, false, err
	}
	return snapshot, true, nil
}

func loadInvoiceForDocument(c echo.Context, id int64) (domain.Invoice, []domain.InvoiceItem, []domain.Payment, domain.Customer, error) {
	db := GetDB(c)
	var invoice domain.Invoice
	if err := db.First(&invoice, id).Error; err != nil {
		return invoice, nil, nil, domain.Customer{}, err
	}
	var items []domain.InvoiceItem
	if err := db.Where("invoice_id = ?", invoice.ID).Order("id ASC").Find(&items).Error; err != nil {
		return invoice, nil, nil, domain.Customer{}, err
	}
	var payments []domain.Payment
	if err := db.Where("invoice_id = ?", invoice.ID).Order("paid_at ASC, id ASC").Find(&payments).Error; err != nil {
		return invoice, nil, nil, domain.Customer{}, err
	}
	var customer domain.Customer
	if err := db.First(&customer, invoice.CustomerID).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return invoice, nil, nil, domain.Customer{}, err
	}
	return invoice, items, payments, customer, nil
}

func currencyFor(c echo.Context) string {
	currency := GetAppContext(c).GetSettingsStringValue("isp", "Currency")
	if strings.TrimSpace(currency) == "" {
		currency = "IDR"
	}
	return currency
}

func invoiceDocumentFrom(c echo.Context, invoice domain.Invoice, items []domain.InvoiceItem, payments []domain.Payment, customer domain.Customer, snapshot domain.InvoiceDocumentSnapshot, captured bool) invoiceDocumentResponse {
	response := invoiceDocumentResponse{
		Kind: "invoice", Number: invoice.InvoiceNo, Status: invoice.Status, Currency: currencyFor(c),
		IssuedAt: invoice.InvoiceDate, DueAt: invoice.DueDate,
		PeriodStart: invoice.PeriodStart, PeriodEnd: invoice.PeriodEnd,
		Issuer: invoiceDocumentIssuer{
			Name: snapshot.IssuerName, TaxID: snapshot.IssuerTaxID, Address: snapshot.IssuerAddress,
			Email: snapshot.IssuerEmail, Phone: snapshot.IssuerPhone, ShowTaxID: snapshot.ShowTaxID,
		},
		BrandName:   snapshot.BrandName,
		AccentColor: snapshot.AccentColor, PaperSize: snapshot.PaperSize,
		HeaderText: snapshot.HeaderText, FooterText: snapshot.FooterText, LogoURL: snapshot.LogoURL,
		PaymentInstructions: invoiceDocumentPaymentInstructions{
			BankName: snapshot.BankName, BankAccountName: snapshot.BankAccountName,
			BankAccountNumber: snapshot.BankAccountNumber, Note: snapshot.PaymentNote,
		},
		QRImageURL: snapshot.QRImageURL,
		Customer: invoiceDocumentCustomer{Name: customer.Name, No: customer.CustomerNo, Phone: customer.Phone},
		Items:     items, Payments: payments,
		Subtotal: invoice.Subtotal, Total: invoice.Total, PaidAmount: invoice.PaidAmount, Balance: invoice.Balance,
		Snapshot: captured, TemplateVersion: snapshot.TemplateVersion,
	}
	if captured && !snapshot.CapturedAt.IsZero() {
		capturedAt := snapshot.CapturedAt
		response.CapturedAt = &capturedAt
	}
	return response
}

func getInvoiceDocumentHandler(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid invoice ID", nil)
	}
	invoice, items, payments, customer, err := loadInvoiceForDocument(c, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fail(c, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil)
	}
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load invoice document", err.Error())
	}
	snapshot, captured, err := loadInvoiceSnapshot(GetDB(c), &invoice)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to resolve invoice document", err.Error())
	}
	return ok(c, invoiceDocumentFrom(c, invoice, items, payments, customer, snapshot, captured))
}

func loadDocumentImage(c echo.Context, tenantID int64, raw string) *invoicedoc.Image {
	if strings.TrimSpace(raw) == "" || !strings.HasPrefix(raw, tenantAssetURLPrefix) {
		return nil
	}
	rest, _ := strings.CutPrefix(raw, tenantAssetURLPrefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 2 {
		return nil
	}
	tenant, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || tenant != tenantID || !tenantAssetNamePattern.MatchString(parts[1]) || filepath.Base(parts[1]) != parts[1] {
		return nil
	}
	path := filepath.Join(GetAppContext(c).Config().GetDataDir(), "tenant-branding", strconv.FormatInt(tenantID, 10), parts[1])
	// #nosec G304 -- tenant matches the authenticated tenant and parts[1] matched the asset pattern plus filepath.Base, so the join cannot leave the tenant's branding directory.
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 || len(data) > 5<<20 {
		return nil
	}
	imageType := ""
	switch {
	case len(data) > 8 && string(data[1:4]) == "PNG":
		imageType = "PNG"
	case len(data) > 3 && data[0] == 0xFF && data[1] == 0xD8:
		imageType = "JPG"
	default:
		return nil
	}
	return &invoicedoc.Image{Data: data, Type: imageType}
}

func buildInvoicePDFDocument(c echo.Context, invoice domain.Invoice, items []domain.InvoiceItem, payments []domain.Payment, customer domain.Customer, snapshot domain.InvoiceDocumentSnapshot) invoicedoc.Document {
	doc := invoicedoc.Document{
		Kind: invoicedoc.KindInvoice, Number: invoice.InvoiceNo, Status: invoice.Status, Currency: currencyFor(c),
		IssuedAt: invoice.InvoiceDate, DueAt: invoice.DueDate,
		PeriodFrom: invoice.PeriodStart, PeriodTo: invoice.PeriodEnd,
		IssuerName: snapshot.IssuerName, IssuerTaxID: snapshot.IssuerTaxID, IssuerAddress: snapshot.IssuerAddress,
		IssuerEmail: snapshot.IssuerEmail, IssuerPhone: snapshot.IssuerPhone,
		BrandName: snapshot.BrandName, AccentColor: snapshot.AccentColor, PaperSize: snapshot.PaperSize,
		HeaderText: snapshot.HeaderText, FooterText: snapshot.FooterText, ShowTaxID: snapshot.ShowTaxID,
		BankName: snapshot.BankName, BankAccountName: snapshot.BankAccountName,
		BankAccountNumber: snapshot.BankAccountNumber, PaymentNote: snapshot.PaymentNote,
		CustomerName: customer.Name, CustomerNo: customer.CustomerNo, CustomerPhone: customer.Phone,
		Subtotal: invoice.Subtotal, Total: invoice.Total, PaidAmount: invoice.PaidAmount, Balance: invoice.Balance,
		Snapshot: true, SnapshotVersion: snapshot.TemplateVersion,
		Logo: loadDocumentImage(c, invoice.TenantID, snapshot.LogoURL),
		QR:   loadDocumentImage(c, invoice.TenantID, snapshot.QRImageURL),
	}
	for _, item := range items {
		doc.Items = append(doc.Items, invoicedoc.Item{Description: item.Description, Quantity: item.Quantity, UnitPrice: item.UnitPrice, Total: item.Total})
	}
	for _, payment := range payments {
		doc.Payments = append(doc.Payments, invoicedoc.PaymentLine{
			Number: payment.PaymentNo, Method: payment.Method, Reference: payment.Reference,
			Amount: payment.Amount, PaidAt: payment.PaidAt,
		})
	}
	return doc
}

func writePDF(c echo.Context, filename string, doc invoicedoc.Document) error {
	data, err := invoicedoc.RenderPDF(doc)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DOCUMENT_RENDER_FAILED", "Failed to render document", err.Error())
	}
	c.Response().Header().Set("Content-Type", "application/pdf")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.Blob(http.StatusOK, "application/pdf", data)
}

func getInvoiceDocumentPDFHandler(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid invoice ID", nil)
	}
	invoice, items, payments, customer, err := loadInvoiceForDocument(c, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return fail(c, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil)
	}
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load invoice document", err.Error())
	}
	snapshot, _, err := loadInvoiceSnapshot(GetDB(c), &invoice)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to resolve invoice document", err.Error())
	}
	return writePDF(c, invoice.InvoiceNo+".pdf", buildInvoicePDFDocument(c, invoice, items, payments, customer, snapshot))
}

func getPaymentReceiptHandler(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid payment ID", nil)
	}
	db := GetDB(c)
	var payment domain.Payment
	if err := db.First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "NOT_FOUND", "Payment not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load payment", err.Error())
	}
	var invoice domain.Invoice
	if err := db.First(&invoice, payment.InvoiceID).Error; err != nil {
		return fail(c, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil)
	}
	var customer domain.Customer
	db.First(&customer, payment.CustomerID)
	snapshot, captured, err := loadInvoiceSnapshot(db, &invoice)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to resolve receipt identity", err.Error())
	}
	var items []domain.InvoiceItem
	db.Where("invoice_id = ?", invoice.ID).Order("id ASC").Find(&items)
	var payments []domain.Payment
	db.Where("invoice_id = ? AND status = ?", invoice.ID, domain.PaymentReceived).Order("paid_at ASC, id ASC").Find(&payments)
	response := invoiceDocumentFrom(c, invoice, items, payments, customer, snapshot, captured)
	response.Kind = "receipt"
	response.Number = payment.PaymentNo
	response.Status = payment.Status
	response.IssuedAt = payment.PaidAt
	response.DueAt = time.Time{}
	return ok(c, response)
}

func getPaymentReceiptPDFHandler(c echo.Context) error {
	id, err := parseIDParam(c, "id")
	if err != nil {
		return fail(c, http.StatusBadRequest, "INVALID_ID", "Invalid payment ID", nil)
	}
	db := GetDB(c)
	var payment domain.Payment
	if err := db.First(&payment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fail(c, http.StatusNotFound, "NOT_FOUND", "Payment not found", nil)
		}
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to load payment", err.Error())
	}
	var invoice domain.Invoice
	if err := db.First(&invoice, payment.InvoiceID).Error; err != nil {
		return fail(c, http.StatusNotFound, "NOT_FOUND", "Invoice not found", nil)
	}
	var customer domain.Customer
	db.First(&customer, payment.CustomerID)
	snapshot, _, err := loadInvoiceSnapshot(db, &invoice)
	if err != nil {
		return fail(c, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to resolve receipt identity", err.Error())
	}
	var items []domain.InvoiceItem
	db.Where("invoice_id = ?", invoice.ID).Order("id ASC").Find(&items)
	doc := buildInvoicePDFDocument(c, invoice, items, nil, customer, snapshot)
	doc.Kind = invoicedoc.KindReceipt
	doc.Number = payment.PaymentNo
	doc.Status = payment.Status
	doc.IssuedAt = payment.PaidAt
	doc.DueAt = time.Time{}
	doc.PeriodFrom = time.Time{}
	doc.PeriodTo = time.Time{}
	doc.PaidAmount = payment.Amount
	doc.Balance = receiptBalance(db, invoice, payment)
	doc.Payments = []invoicedoc.PaymentLine{{
		Number: payment.PaymentNo, Method: payment.Method, Reference: payment.Reference,
		Amount: payment.Amount, PaidAt: payment.PaidAt,
	}}
	return writePDF(c, payment.PaymentNo+".pdf", doc)
}

// receiptBalance derives the invoice balance immediately after this payment
// from immutable payment rows. It never recalculates or stores invoice totals.
func receiptBalance(db *gorm.DB, invoice domain.Invoice, payment domain.Payment) int64 {
	var paid int64
	if err := db.Model(&domain.Payment{}).
		Where("invoice_id = ? AND status = ? AND (paid_at < ? OR (paid_at = ? AND id <= ?))",
			invoice.ID, domain.PaymentReceived, payment.PaidAt, payment.PaidAt, payment.ID).
		Select("COALESCE(SUM(amount), 0)").Scan(&paid).Error; err != nil {
		return invoice.Balance
	}
	balance := invoice.Total - paid
	if balance < 0 {
		return 0
	}
	return balance
}
