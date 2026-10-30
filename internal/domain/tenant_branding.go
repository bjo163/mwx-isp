package domain

import "time"

// TenantBranding stores customizable branding configuration for a tenant,
// including logo, colors, theme, and website appearance settings.
type TenantBranding struct {
	ID        int64     `json:"id,string" gorm:"primaryKey"`
	TenantID  int64     `json:"tenant_id,string" gorm:"not null;uniqueIndex"`
	Tenant    Tenant    `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`

	LogoURL      string `json:"logo_url" gorm:"size:500"`
	FaviconURL   string `json:"favicon_url" gorm:"size:500"`
	PrimaryColor string `json:"primary_color" gorm:"size:7;default:#1976d2"`
	SecondaryColor string `json:"secondary_color" gorm:"size:7;default:#dc004e"`
	Theme        string `json:"theme" gorm:"size:16;default:light"`

	CompanyName  string `json:"company_name" gorm:"size:150"`
	Tagline      string `json:"tagline" gorm:"size:300"`
	WebsiteURL   string `json:"website_url" gorm:"size:500"`
	SupportEmail string `json:"support_email" gorm:"size:150"`
	SupportPhone string `json:"support_phone" gorm:"size:32"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the storage table for tenant branding configuration.
func (TenantBranding) TableName() string { return "tenant_branding" }
