package domain

import "time"

// TenantBranding stores a tenant's public-facing brand and support presentation.
// Legal issuer identity remains on Tenant and is intentionally separate.
type TenantBranding struct {
	ID       int64  `json:"id,string" gorm:"primaryKey"`
	TenantID int64  `json:"tenant_id,string" gorm:"not null;uniqueIndex"`
	Tenant   Tenant `json:"tenant,omitempty" gorm:"foreignKey:TenantID"`

	BrandName        string `json:"brand_name" gorm:"size:150"`
	ShortName        string `json:"short_name" gorm:"size:24"`
	CompanyName      string `json:"company_name" gorm:"size:150"` // Legacy public display name.
	Tagline          string `json:"tagline" gorm:"size:300"`
	LogoURL          string `json:"logo_url" gorm:"size:500"`
	CompactLogoURL   string `json:"compact_logo_url" gorm:"size:500"`
	LightLogoURL     string `json:"light_logo_url" gorm:"size:500"`
	DarkLogoURL      string `json:"dark_logo_url" gorm:"size:500"`
	FaviconURL       string `json:"favicon_url" gorm:"size:500"`
	SocialImageURL   string `json:"social_image_url" gorm:"size:500"`
	PrimaryColor     string `json:"primary_color" gorm:"size:7;default:#1976d2"`
	SecondaryColor   string `json:"secondary_color" gorm:"size:7;default:#dc004e"`
	AccentColor      string `json:"accent_color" gorm:"size:7"`
	BackgroundColor  string `json:"background_color" gorm:"size:7"`
	SurfaceColor     string `json:"surface_color" gorm:"size:7"`
	TextColor        string `json:"text_color" gorm:"size:7"`
	LinkColor        string `json:"link_color" gorm:"size:7"`
	Theme            string `json:"theme" gorm:"size:16;default:light"`
	Typography       string `json:"typography" gorm:"size:32;default:system"`
	WebsiteURL       string `json:"website_url" gorm:"size:500"`
	SupportEmail     string `json:"support_email" gorm:"size:150"`
	SupportPhone     string `json:"support_phone" gorm:"size:32"`
	SupportWhatsApp  string `json:"support_whatsapp" gorm:"size:32"`
	ContactAddress   string `json:"contact_address" gorm:"size:500"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the storage table for tenant branding configuration.
func (TenantBranding) TableName() string { return "tenant_branding" }
