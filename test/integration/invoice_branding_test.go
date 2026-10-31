//go:build integration

package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/bjo163/mwx-isp/internal/billing"
	"github.com/bjo163/mwx-isp/internal/domain"
	"github.com/bjo163/mwx-isp/internal/tenancy"
	"github.com/bjo163/mwx-isp/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invoiceDocumentView struct {
	Kind            string `json:"kind"`
	Number          string `json:"number"`
	Status          string `json:"status"`
	Snapshot        bool   `json:"snapshot"`
	TemplateVersion int    `json:"template_version"`
	Issuer          struct {
		Name string `json:"name"`
	} `json:"issuer"`
	PaymentInstructions struct {
		BankName          string `json:"bank_name"`
		BankAccountNumber string `json:"bank_account_number"`
	} `json:"payment_instructions"`
	Total      int64      `json:"total"`
	Paid       int64      `json:"paid_amount"`
	Balance    int64      `json:"balance"`
	SnapshotAt *time.Time `json:"captured_at"`
}

// seedTenantBillingFixture creates a customer, package and subscription inside
// one tenant, so invoice generation runs through the production tenant scope.
func seedTenantBillingFixture(t *testing.T, tenantID int64, label string, price int64, start time.Time) int64 {
	t.Helper()
	db := h.appCtx.DB()
	suffix := uniqueSuffix()
	tenantDB := db.WithContext(tenancy.WithTenantID(t.Context(), tenantID))
	customer := domain.Customer{
		ID: common.UUIDint64(), CustomerNo: fmt.Sprintf("DOC-%s-%s", label, suffix),
		Name: label + " customer", Status: domain.CustomerActive,
	}
	require.NoError(t, tenantDB.Create(&customer).Error)
	pkg := domain.InternetPackage{
		ID: common.UUIDint64(), Code: fmt.Sprintf("DOC-%s-%s", label, suffix),
		Name: label + " package", Price: price, BillingCycle: "monthly", Status: "active",
	}
	require.NoError(t, tenantDB.Create(&pkg).Error)
	subscription := domain.Subscription{
		ID: common.UUIDint64(), CustomerID: customer.ID, PackageID: pkg.ID,
		Status: domain.SubscriptionActive, StartDate: start, BillingDay: 1, GraceDays: 3,
	}
	require.NoError(t, tenantDB.Create(&subscription).Error)
	return subscription.ID
}

func putDocumentTemplate(t *testing.T, actor tenantActor, issuer, accent, bank, account string) {
	t.Helper()
	payload, err := json.Marshal(map[string]any{
		"issuer_name": issuer, "accent_color": accent, "paper_size": "A4",
		"show_tax_id": true, "bank_name": bank,
		"bank_account_name": issuer, "bank_account_number": account,
		"payment_note": "Cantumkan nomor invoice",
	})
	require.NoError(t, err)
	status, body := actor.client.put(t, "/api/v1/tenant-document-template", payload)
	require.Equalf(t, http.StatusOK, status, "template save for %s: %s", actor.tenant.Slug, body)
	var template domain.TenantDocumentTemplate
	unwrapData(t, body, &template)
	require.Equal(t, issuer, template.IssuerName)
}

// TestPostgresInvoiceDocumentsFollowIssueTimeSnapshot proves the M20 boundary on
// PostgreSQL: each tenant's template only decorates its own documents, the
// issuer identity and payment instructions freeze when the invoice is issued,
// later template edits never rewrite an issued document, and the billing
// totals, numbering, status and ledger stay exactly as the invoice service left
// them.
func TestPostgresInvoiceDocumentsFollowIssueTimeSnapshot(t *testing.T) {
	db := h.appCtx.DB()
	platform := newAPIClient(t)
	alpha := provisionTenantActor(t, platform, "docs-alpha")
	beta := provisionTenantActor(t, platform, "docs-beta")

	issuedAt := time.Date(2026, time.October, 5, 0, 0, 0, 0, time.UTC)
	alphaSubscription := seedTenantBillingFixture(t, alpha.tenant.ID, "alpha", 250000, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC))
	betaSubscription := seedTenantBillingFixture(t, beta.tenant.ID, "beta", 150000, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC))

	putDocumentTemplate(t, alpha, "PT Alpha Network", "#16A34A", "Bank Mandiri", "1230009876543")
	putDocumentTemplate(t, beta, "PT Beta Net", "#2563EB", "Bank BCA", "7770001112223")

	alphaDB := db.WithContext(tenancy.WithTenantID(t.Context(), alpha.tenant.ID))
	betaDB := db.WithContext(tenancy.WithTenantID(t.Context(), beta.tenant.ID))
	created, err := billing.GenerateMonthlyInvoicesForSubscriptions(alphaDB, issuedAt, 10, []int64{alphaSubscription})
	require.NoError(t, err)
	require.Equal(t, 1, created, "alpha subscription must produce one invoice")
	created, err = billing.GenerateMonthlyInvoicesForSubscriptions(betaDB, issuedAt, 10, []int64{betaSubscription})
	require.NoError(t, err)
	require.Equal(t, 1, created, "beta subscription must produce one invoice")

	var alphaInvoice domain.Invoice
	require.NoError(t, alphaDB.Where("subscription_id = ?", alphaSubscription).First(&alphaInvoice).Error)
	var betaInvoice domain.Invoice
	require.NoError(t, betaDB.Where("subscription_id = ?", betaSubscription).First(&betaInvoice).Error)
	require.EqualValues(t, 250000, alphaInvoice.Total, "billing totals must come from the package price")
	require.EqualValues(t, 250000, alphaInvoice.Balance)
	require.Equal(t, domain.InvoiceIssued, alphaInvoice.Status)

	// Issuing captured exactly one snapshot, carrying the tenant template and
	// the invoice's own financial record.
	var snapshot domain.InvoiceDocumentSnapshot
	require.NoError(t, alphaDB.Where("invoice_id = ?", alphaInvoice.ID).First(&snapshot).Error)
	assert.Equal(t, "PT Alpha Network", snapshot.IssuerName)
	assert.Equal(t, "Bank Mandiri", snapshot.BankName)
	assert.EqualValues(t, 1, snapshot.TemplateVersion)
	var count int64
	require.NoError(t, alphaDB.Model(&domain.InvoiceDocumentSnapshot{}).Where("invoice_id = ?", alphaInvoice.ID).Count(&count).Error)
	assert.EqualValues(t, 1, count, "an invoice must capture one document snapshot")

	documentPath := fmt.Sprintf("/api/v1/isp/invoices/%d/document", alphaInvoice.ID)
	status, body := alpha.client.get(t, documentPath)
	require.Equalf(t, http.StatusOK, status, "alpha document read: %s", body)
	var document struct {
		Data invoiceDocumentView `json:"data"`
	}
	require.NoError(t, json.Unmarshal(body, &document))
	assert.True(t, document.Data.Snapshot, "an issued invoice must be served from its snapshot")
	assert.Equal(t, "invoice", document.Data.Kind)
	assert.Equal(t, alphaInvoice.InvoiceNo, document.Data.Number)
	assert.Equal(t, "PT Alpha Network", document.Data.Issuer.Name)
	assert.Equal(t, "Bank Mandiri", document.Data.PaymentInstructions.BankName)
	assert.EqualValues(t, 1, document.Data.TemplateVersion)
	assert.EqualValues(t, alphaInvoice.Total, document.Data.Total)
	assert.NotNil(t, document.Data.SnapshotAt)

	status, pdf := alpha.client.get(t, documentPath+".pdf")
	require.Equalf(t, http.StatusOK, status, "alpha invoice PDF: %s", pdf)
	assert.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")), "invoice endpoint must emit a PDF")

	// A later template save bumps its version but must not touch the document a
	// customer already received.
	var eventsBefore int64
	require.NoError(t, alphaDB.Model(&domain.BillingEvent{}).Count(&eventsBefore).Error)
	putDocumentTemplate(t, alpha, "PT Alpha Renamed", "#111827", "Bank BRI", "5550004443332")
	status, body = alpha.client.get(t, documentPath)
	require.Equalf(t, http.StatusOK, status, "alpha document read after template change: %s", body)
	require.NoError(t, json.Unmarshal(body, &document))
	assert.Equal(t, "PT Alpha Network", document.Data.Issuer.Name, "an issued document keeps its issuer")
	assert.Equal(t, "Bank Mandiri", document.Data.PaymentInstructions.BankName, "payment instructions are frozen at issue")
	assert.EqualValues(t, 1, document.Data.TemplateVersion, "the snapshot keeps the template version used at issue")

	// The snapshot itself refuses mutation, so history cannot be rewritten even
	// by a direct database update.
	require.ErrorContains(t, alphaDB.Model(&snapshot).Update("issuer_name", "Rewritten").Error,
		"immutable", "an issued snapshot must reject updates")

	// Template edits are presentation-only: billing rows are untouched.
	var eventsAfter int64
	require.NoError(t, alphaDB.Model(&domain.BillingEvent{}).Count(&eventsAfter).Error)
	assert.EqualValues(t, eventsBefore, eventsAfter, "template edits must not write billing events")
	var reread domain.Invoice
	require.NoError(t, alphaDB.First(&reread, alphaInvoice.ID).Error)
	assert.Equal(t, alphaInvoice.InvoiceNo, reread.InvoiceNo, "template edits must not renumber an invoice")
	assert.EqualValues(t, alphaInvoice.Total, reread.Total)
	assert.EqualValues(t, alphaInvoice.Balance, reread.Balance)
	assert.Equal(t, alphaInvoice.Status, reread.Status)

	// The neighbour tenant can neither read alpha's document nor inherit its
	// template.
	status, body = beta.client.get(t, documentPath)
	assert.Equal(t, http.StatusNotFound, status, "another tenant must not read the document: %s", body)
	assert.NotContains(t, string(body), "PT Alpha Network")
	status, body = beta.client.get(t, fmt.Sprintf("/api/v1/isp/invoices/%d/document.pdf", alphaInvoice.ID))
	assert.Equal(t, http.StatusNotFound, status, "another tenant must not download the PDF: %s", body)

	status, body = beta.client.get(t, "/api/v1/tenant-document-template")
	require.Equalf(t, http.StatusOK, status, "beta template read: %s", body)
	var betaTemplate domain.TenantDocumentTemplate
	unwrapData(t, body, &betaTemplate)
	assert.Equal(t, "PT Beta Net", betaTemplate.IssuerName)
	assert.NotEqual(t, "PT Alpha Renamed", betaTemplate.IssuerName)

	betaPath := fmt.Sprintf("/api/v1/isp/invoices/%d/document", betaInvoice.ID)
	status, body = alpha.client.get(t, betaPath)
	assert.Equal(t, http.StatusNotFound, status, "alpha must not read beta's document: %s", body)

	// A payment receipt is likewise tenant-scoped, reports the balance as of that
	// payment, and renders as a PDF from the same frozen invoice snapshot.
	paymentPayload, err := json.Marshal(map[string]any{
		"invoice_id": fmt.Sprint(alphaInvoice.ID), "amount": 250000,
		"method": "bank_transfer", "reference": "TRF-DOC-1",
	})
	require.NoError(t, err)
	status, body = alpha.client.post(t, "/api/v1/isp/payments", paymentPayload)
	require.Equalf(t, http.StatusCreated, status, "alpha payment recording: %s", body)
	var payment domain.Payment
	unwrapData(t, body, &payment)
	require.NotZero(t, payment.ID)

	receiptPath := fmt.Sprintf("/api/v1/isp/payments/%d/receipt", payment.ID)
	status, body = alpha.client.get(t, receiptPath)
	require.Equalf(t, http.StatusOK, status, "alpha receipt read: %s", body)
	require.NoError(t, json.Unmarshal(body, &document))
	assert.Equal(t, "receipt", document.Data.Kind)
	assert.Equal(t, payment.PaymentNo, document.Data.Number)
	assert.EqualValues(t, 0, document.Data.Balance, "the receipt shows the balance after that payment")
	assert.EqualValues(t, 250000, document.Data.Paid)

	status, pdf = alpha.client.get(t, receiptPath+".pdf")
	require.Equalf(t, http.StatusOK, status, "alpha receipt PDF: %s", pdf)
	assert.True(t, bytes.HasPrefix(pdf, []byte("%PDF-")), "receipt endpoint must emit a PDF")

	status, body = beta.client.get(t, receiptPath)
	assert.Equal(t, http.StatusNotFound, status, "another tenant must not read the receipt: %s", body)

	// The invoice itself keeps its post-payment state; documents never mutate it.
	require.NoError(t, alphaDB.First(&reread, alphaInvoice.ID).Error)
	assert.Equal(t, domain.InvoicePaid, reread.Status)
	assert.EqualValues(t, 0, reread.Balance)
	assert.EqualValues(t, 250000, reread.PaidAmount)
}
