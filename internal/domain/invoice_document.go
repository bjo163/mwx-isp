package domain

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// TenantDocumentTemplate is the constrained presentation of a tenant's invoice
// and receipt documents. It carries display and payment-instruction text only;
// totals, status, numbering, and the ledger stay on their operational records.
type TenantDocumentTemplate struct {
	ID                int64     `json:"id,string" gorm:"primaryKey"`
	TenantID          int64     `json:"-" gorm:"not null;default:1;uniqueIndex:udx_isp_document_template_tenant,priority:1"`
	Version           int       `json:"version" gorm:"not null;default:1"`
	IssuerName        string    `json:"issuer_name" gorm:"size:150"`
	LogoURL           string    `json:"logo_url" gorm:"size:500"`
	HeaderText        string    `json:"header_text" gorm:"size:300"`
	FooterText        string    `json:"footer_text" gorm:"size:300"`
	AccentColor       string    `json:"accent_color" gorm:"size:7"`
	PaperSize         string    `json:"paper_size" gorm:"size:8;default:A4"`
	ShowTaxID         bool      `json:"show_tax_id" gorm:"not null;default:true"`
	BankName          string    `json:"bank_name" gorm:"size:80"`
	BankAccountName   string    `json:"bank_account_name" gorm:"size:120"`
	BankAccountNumber string    `json:"bank_account_number" gorm:"size:40"`
	PaymentNote       string    `json:"payment_note" gorm:"size:300"`
	QRImageURL        string    `json:"qr_image_url" gorm:"size:500"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TableName returns the tenant document template table.
func (TenantDocumentTemplate) TableName() string { return "isp_document_template" }

// InvoiceDocumentSnapshot freezes the issuer identity, template and payment
// instructions used when an invoice is issued. Later profile or template edits
// must not change an already issued document.
type InvoiceDocumentSnapshot struct {
	ID                int64     `json:"id,string" gorm:"primaryKey"`
	TenantID          int64     `json:"-" gorm:"not null;default:1;uniqueIndex:udx_isp_invoice_document,priority:1"`
	InvoiceID         int64     `json:"invoice_id,string" gorm:"not null;uniqueIndex:udx_isp_invoice_document,priority:2"`
	IssuerName        string    `json:"issuer_name" gorm:"size:150"`
	IssuerTaxID       string    `json:"issuer_tax_id" gorm:"size:80"`
	IssuerAddress     string    `json:"issuer_address" gorm:"size:500"`
	IssuerEmail       string    `json:"issuer_email" gorm:"size:150"`
	IssuerPhone       string    `json:"issuer_phone" gorm:"size:32"`
	BrandName         string    `json:"brand_name" gorm:"size:150"`
	LogoURL           string    `json:"logo_url" gorm:"size:500"`
	HeaderText        string    `json:"header_text" gorm:"size:300"`
	FooterText        string    `json:"footer_text" gorm:"size:300"`
	AccentColor       string    `json:"accent_color" gorm:"size:7"`
	PaperSize         string    `json:"paper_size" gorm:"size:8"`
	ShowTaxID         bool      `json:"show_tax_id"`
	BankName          string    `json:"bank_name" gorm:"size:80"`
	BankAccountName   string    `json:"bank_account_name" gorm:"size:120"`
	BankAccountNumber string    `json:"bank_account_number" gorm:"size:40"`
	PaymentNote       string    `json:"payment_note" gorm:"size:300"`
	QRImageURL        string    `json:"qr_image_url" gorm:"size:500"`
	TemplateVersion   int       `json:"template_version" gorm:"not null;default:1"`
	CapturedAt        time.Time `json:"captured_at" gorm:"not null"`
}

// TableName returns the immutable invoice document snapshot table.
func (InvoiceDocumentSnapshot) TableName() string { return "isp_invoice_document" }

// BeforeUpdate rejects any mutation so an issued document keeps its legal and
// visual identity even if the tenant profile or template changes later.
func (s *InvoiceDocumentSnapshot) BeforeUpdate(*gorm.DB) error {
	return errors.New("invoice document snapshots are immutable")
}
