package adminapi

import (
	"encoding/json"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTenantDocumentTemplateVersionsAndValidates(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.TenantDocumentTemplate{}))

	body := `{"issuer_name":"PT Alpha Network","header_text":"Invoice resmi","footer_text":"Terima kasih",
		"accent_color":"#16A34A","paper_size":"A4","show_tax_id":true,
		"bank_name":"BCA","bank_account_name":"PT Alpha Network","bank_account_number":"123-456-7890",
		"payment_note":"Transfer sesuai nomor invoice"}`
	c, recorder := newTenantRequestContext(t, e, db, appCtx, http.MethodPut, "/api/v1/tenant-document-template", body, domain.DefaultTenantID, nil)
	require.NoError(t, updateTenantDocumentTemplateHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())

	var saved domain.TenantDocumentTemplate
	require.NoError(t, db.Where("tenant_id = ?", domain.DefaultTenantID).First(&saved).Error)
	assert.Equal(t, 1, saved.Version)
	assert.Equal(t, "PT Alpha Network", saved.IssuerName)
	assert.Equal(t, "123-456-7890", saved.BankAccountNumber)
	assert.True(t, saved.ShowTaxID)

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPut, "/api/v1/tenant-document-template",
		`{"accent_color":"#2563EB"}`, domain.DefaultTenantID, nil)
	require.NoError(t, updateTenantDocumentTemplateHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	require.NoError(t, db.Where("tenant_id = ?", domain.DefaultTenantID).First(&saved).Error)
	assert.Equal(t, 2, saved.Version, "each edit must bump the template version")
	assert.Equal(t, "#2563EB", saved.AccentColor)

	for name, payload := range map[string]string{
		"invalid color":       `{"accent_color":"green"}`,
		"invalid paper":       `{"paper_size":"A5"}`,
		"partial bank fields": `{"bank_name":""}`,
		"external QR":         `{"qr_image_url":"https://example.com/qr.png"}`,
	} {
		c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPut, "/api/v1/tenant-document-template", payload, domain.DefaultTenantID, nil)
		require.NoError(t, updateTenantDocumentTemplateHandler(c), name)
		assert.Equalf(t, http.StatusBadRequest, recorder.Code, "%s: %s", name, recorder.Body.String())
	}
	var unchanged domain.TenantDocumentTemplate
	require.NoError(t, db.Where("tenant_id = ?", domain.DefaultTenantID).First(&unchanged).Error)
	assert.Equal(t, 2, unchanged.Version, "a rejected edit must not create a version")

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodPost, "/api/v1/tenant-document-template/reset", "", domain.DefaultTenantID, nil)
	require.NoError(t, resetTenantDocumentTemplateHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	require.NoError(t, db.Where("tenant_id = ?", domain.DefaultTenantID).First(&saved).Error)
	assert.Equal(t, 3, saved.Version, "reset is a new version, not a silent rewrite")
	assert.Empty(t, saved.BankAccountNumber)

	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Name: "Tenant Two", Slug: "doc-tenant-two", Kind: "isp", Status: "active"}).Error)
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/tenant-document-template", "", 2, nil)
	require.NoError(t, getTenantDocumentTemplateHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	var other domain.TenantDocumentTemplate
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &struct {
		Data *domain.TenantDocumentTemplate `json:"data"`
	}{Data: &other}))
	assert.Empty(t, other.IssuerName, "a second tenant must not read the first tenant's template")
}

func TestInvoiceDocumentSnapshotFreezesIssuerIdentity(t *testing.T) {
	db, _, _ := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(&domain.TenantBranding{}, &domain.TenantDocumentTemplate{}, &domain.InvoiceDocumentSnapshot{},
		&domain.Customer{}, &domain.Invoice{}, &domain.InvoiceItem{}, &domain.Payment{}))
	require.NoError(t, db.Model(&domain.Tenant{}).Where("id = ?", domain.DefaultTenantID).Updates(map[string]any{
		"company_name": "PT Alpha Network", "tax_id": "01.234.567.8-901.000",
		"billing_address": "Jl. Merdeka 1", "contact_email": "billing@alpha.test",
	}).Error)
	require.NoError(t, db.Create(&domain.TenantBranding{
		TenantID: domain.DefaultTenantID, BrandName: "Alpha Fiber", CompanyName: "PT Alpha Network",
		AccentColor: "#16A34A",
	}).Error)
	require.NoError(t, db.Create(&domain.TenantDocumentTemplate{
		TenantID: domain.DefaultTenantID, Version: 4, IssuerName: "PT Alpha Network",
		BankName: "BCA", BankAccountName: "PT Alpha Network", BankAccountNumber: "1234567890",
	}).Error)

	issuedAt := time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-202610-000001", CustomerID: 1, SubscriptionID: 1,
		InvoiceDate: issuedAt, DueDate: issuedAt.AddDate(0, 0, 10),
		PeriodStart: issuedAt, PeriodEnd: issuedAt.AddDate(0, 1, -1),
		Subtotal: 250000, Total: 250000, Balance: 250000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)
	require.NoError(t, billing.CaptureInvoiceDocument(db, &invoice, issuedAt))

	var snapshot domain.InvoiceDocumentSnapshot
	require.NoError(t, db.Where("invoice_id = ?", invoice.ID).First(&snapshot).Error)
	assert.Equal(t, "PT Alpha Network", snapshot.IssuerName)
	assert.Equal(t, "01.234.567.8-901.000", snapshot.IssuerTaxID)
	assert.Equal(t, "Alpha Fiber", snapshot.BrandName)
	assert.Equal(t, 4, snapshot.TemplateVersion)
	assert.Equal(t, "1234567890", snapshot.BankAccountNumber)

	require.NoError(t, db.Model(&domain.TenantBranding{}).Where("tenant_id = ?", domain.DefaultTenantID).Update("brand_name", "Changed Brand").Error)
	require.NoError(t, db.Model(&domain.TenantDocumentTemplate{}).Where("tenant_id = ?", domain.DefaultTenantID).Updates(map[string]any{"bank_account_number": "999", "version": 5}).Error)
	require.NoError(t, db.Model(&domain.Tenant{}).Where("id = ?", domain.DefaultTenantID).Update("company_name", "PT Changed").Error)

	var reread domain.InvoiceDocumentSnapshot
	require.NoError(t, db.Where("invoice_id = ?", invoice.ID).First(&reread).Error)
	assert.Equal(t, "PT Alpha Network", reread.IssuerName, "a profile edit must not rewrite an issued document")
	assert.Equal(t, "Alpha Fiber", reread.BrandName)
	assert.Equal(t, "1234567890", reread.BankAccountNumber)
	assert.Equal(t, 4, reread.TemplateVersion)

	reread.IssuerName = "Forged Issuer"
	assert.ErrorContains(t, db.Save(&reread).Error, "immutable")
	require.NoError(t, db.Where("invoice_id = ?", invoice.ID).First(&reread).Error)
	assert.Equal(t, "PT Alpha Network", reread.IssuerName)
}

func TestInvoiceAndReceiptPDFUseIssuedSnapshot(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.TenantBranding{}, &domain.TenantDocumentTemplate{}, &domain.InvoiceDocumentSnapshot{},
		&domain.Customer{}, &domain.Invoice{}, &domain.InvoiceItem{}, &domain.Payment{},
	))
	customer := domain.Customer{TenantID: domain.DefaultTenantID, CustomerNo: "CUST-1", Name: "Budi", Phone: "0812000000", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	invoice := domain.Invoice{
		TenantID: domain.DefaultTenantID, InvoiceNo: "INV-202610-000002", CustomerID: customer.ID, SubscriptionID: 1,
		InvoiceDate: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		DueDate:     time.Date(2026, time.October, 11, 0, 0, 0, 0, time.UTC),
		PeriodStart: time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC),
		Subtotal:    250000, Total: 250000, Balance: 250000, Status: domain.InvoiceIssued,
	}
	require.NoError(t, db.Create(&invoice).Error)
	require.NoError(t, db.Create(&domain.InvoiceItem{TenantID: domain.DefaultTenantID, InvoiceID: invoice.ID, Description: "Home 100 Mbps", Quantity: 1, UnitPrice: 250000, Total: 250000}).Error)
	require.NoError(t, billing.CaptureInvoiceDocument(db, &invoice, invoice.InvoiceDate))
	payment := domain.Payment{
		TenantID: domain.DefaultTenantID, PaymentNo: "PAY-202610-000001", CustomerID: customer.ID, InvoiceID: invoice.ID,
		Amount: 250000, Method: "bank_transfer", PaidAt: invoice.InvoiceDate.Add(24 * time.Hour), Status: domain.PaymentReceived,
	}
	require.NoError(t, db.Create(&payment).Error)

	invoiceID := strconv.FormatInt(invoice.ID, 10)
	c, recorder := newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/isp/invoices/"+invoiceID+"/document", "", domain.DefaultTenantID, map[string]string{"id": invoiceID})
	require.NoError(t, getInvoiceDocumentHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	var document struct {
		Data invoiceDocumentResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	assert.True(t, document.Data.Snapshot)
	assert.Equal(t, invoice.InvoiceNo, document.Data.Number)
	assert.EqualValues(t, 250000, document.Data.Total)

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/isp/invoices/"+invoiceID+"/document.pdf", "", domain.DefaultTenantID, map[string]string{"id": invoiceID})
	require.NoError(t, getInvoiceDocumentPDFHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	assert.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	assert.Contains(t, recorder.Header().Get("Content-Disposition"), invoice.InvoiceNo)
	require.True(t, len(recorder.Body.Bytes()) > 400)
	assert.Equal(t, "%PDF", recorder.Body.String()[:4])

	paymentID := strconv.FormatInt(payment.ID, 10)
	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/isp/payments/"+paymentID+"/receipt", "", domain.DefaultTenantID, map[string]string{"id": paymentID})
	require.NoError(t, getPaymentReceiptHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &document))
	assert.Equal(t, "receipt", document.Data.Kind)
	assert.Equal(t, payment.PaymentNo, document.Data.Number)

	c, recorder = newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/isp/payments/"+paymentID+"/receipt.pdf", "", domain.DefaultTenantID, map[string]string{"id": paymentID})
	require.NoError(t, getPaymentReceiptPDFHandler(c))
	require.Equalf(t, http.StatusOK, recorder.Code, "%s", recorder.Body.String())
	assert.Equal(t, "application/pdf", recorder.Header().Get("Content-Type"))
	assert.Equal(t, "%PDF", recorder.Body.String()[:4])
}

func TestInvoiceDocumentIsTenantScoped(t *testing.T) {
	db, e, appCtx := CreateTestAppContext(t)
	require.NoError(t, db.AutoMigrate(
		&domain.TenantBranding{}, &domain.TenantDocumentTemplate{}, &domain.InvoiceDocumentSnapshot{},
		&domain.Customer{}, &domain.Invoice{}, &domain.InvoiceItem{}, &domain.Payment{},
	))
	require.NoError(t, db.Create(&domain.Tenant{ID: 2, Name: "Tenant Two", Slug: "doc-tenant-two-2", Kind: "isp", Status: "active"}).Error)
	customer := domain.Customer{TenantID: domain.DefaultTenantID, CustomerNo: "CUST-9", Name: "Sari", Status: domain.CustomerActive}
	require.NoError(t, db.Create(&customer).Error)
	invoice := domain.Invoice{TenantID: domain.DefaultTenantID, InvoiceNo: "INV-202610-000009", CustomerID: customer.ID, SubscriptionID: 1,
		InvoiceDate: time.Now(), DueDate: time.Now().AddDate(0, 0, 10), PeriodStart: time.Now(), PeriodEnd: time.Now(),
		Subtotal: 100000, Total: 100000, Balance: 100000, Status: domain.InvoiceIssued}
	require.NoError(t, db.Create(&invoice).Error)

	id := strconv.FormatInt(invoice.ID, 10)
	c, recorder := newTenantRequestContext(t, e, db, appCtx, http.MethodGet, "/api/v1/isp/invoices/"+id+"/document", "", 2, map[string]string{"id": id})
	require.NoError(t, getInvoiceDocumentHandler(c))
	assert.Equal(t, http.StatusNotFound, recorder.Code, "another tenant must not read the document")
}
