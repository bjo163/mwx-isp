package billing

import (
	"errors"
	"strings"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"gorm.io/gorm"
)

// CaptureInvoiceDocument freezes the issuer identity, document template and
// payment instructions of an invoice at issuance. It must run inside the same
// transaction that creates the invoice so a failed issue leaves no snapshot.
// Installations whose schema predates this table keep issuing invoices; the
// snapshot is added by the normal startup migration.
func CaptureInvoiceDocument(tx *gorm.DB, invoice *domain.Invoice, now time.Time) error {
	if tx == nil || invoice == nil || invoice.ID == 0 {
		return errors.New("an issued invoice is required")
	}
	if !tx.Migrator().HasTable(&domain.InvoiceDocumentSnapshot{}) {
		return nil
	}
	var existing int64
	if err := tx.Model(&domain.InvoiceDocumentSnapshot{}).Where("tenant_id = ? AND invoice_id = ?", invoice.TenantID, invoice.ID).Count(&existing).Error; err != nil {
		return err
	}
	if existing > 0 {
		return nil
	}
	snapshot, err := BuildInvoiceDocumentSnapshot(tx, invoice, now)
	if err != nil {
		return err
	}
	return tx.Create(&snapshot).Error
}

// BuildInvoiceDocumentSnapshot resolves the immutable document identity for an
// invoice from the tenant legal identity, tenant branding and tenant document
// template. Missing rows fall back to safe defaults instead of failing an
// issue, so a tenant that never opened the editor still gets a valid document.
func BuildInvoiceDocumentSnapshot(tx *gorm.DB, invoice *domain.Invoice, now time.Time) (domain.InvoiceDocumentSnapshot, error) {
	if tx == nil || invoice == nil {
		return domain.InvoiceDocumentSnapshot{}, errors.New("invoice is required")
	}
	tenantID := invoice.TenantID
	if tenantID <= 0 {
		tenantID = domain.DefaultTenantID
	}

	var tenant domain.Tenant
	if err := tx.Where("id = ?", tenantID).First(&tenant).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.InvoiceDocumentSnapshot{}, err
	}

	var branding domain.TenantBranding
	if err := tx.Where("tenant_id = ?", tenantID).First(&branding).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.InvoiceDocumentSnapshot{}, err
	}

	var template domain.TenantDocumentTemplate
	if err := tx.Where("tenant_id = ?", tenantID).First(&template).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.InvoiceDocumentSnapshot{}, err
	}

	firstNonEmpty := func(values ...string) string {
		for _, value := range values {
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		}
		return ""
	}
	version := template.Version
	if version <= 0 {
		version = 1
	}
	paperSize := strings.TrimSpace(template.PaperSize)
	if paperSize == "" {
		paperSize = "A4"
	}
	accent := firstNonEmpty(template.AccentColor, branding.AccentColor, branding.PrimaryColor)
	if accent == "" {
		accent = "#16A34A"
	}

	return domain.InvoiceDocumentSnapshot{
		TenantID:          tenantID,
		InvoiceID:         invoice.ID,
		IssuerName:        firstNonEmpty(template.IssuerName, branding.CompanyName, tenant.CompanyName, tenant.Name),
		IssuerTaxID:       strings.TrimSpace(tenant.TaxID),
		IssuerAddress:     firstNonEmpty(tenant.BillingAddress, branding.ContactAddress),
		IssuerEmail:       firstNonEmpty(tenant.ContactEmail, branding.SupportEmail),
		IssuerPhone:       firstNonEmpty(tenant.ContactPhone, branding.SupportPhone),
		BrandName:         firstNonEmpty(branding.BrandName, branding.CompanyName, tenant.CompanyName, tenant.Name),
		LogoURL:           firstNonEmpty(template.LogoURL, branding.LogoURL, branding.CompactLogoURL),
		HeaderText:        strings.TrimSpace(template.HeaderText),
		FooterText:        strings.TrimSpace(template.FooterText),
		AccentColor:       strings.ToUpper(accent),
		PaperSize:         paperSize,
		ShowTaxID:         template.ID == 0 || template.ShowTaxID,
		BankName:          strings.TrimSpace(template.BankName),
		BankAccountName:   strings.TrimSpace(template.BankAccountName),
		BankAccountNumber: strings.TrimSpace(template.BankAccountNumber),
		PaymentNote:       strings.TrimSpace(template.PaymentNote),
		QRImageURL:        strings.TrimSpace(template.QRImageURL),
		TemplateVersion:   version,
		CapturedAt:        now.UTC(),
	}, nil
}
