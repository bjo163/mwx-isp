package domain

import "time"

// TenantWebsite stores one tenant's editable draft and currently published site.
// Content is structured JSON rendered as inert text by the frontend.
type TenantWebsite struct {
	ID int64 `json:"id,string" gorm:"primaryKey"`
	TenantID int64 `json:"tenant_id,string" gorm:"not null;uniqueIndex"`
	DraftContent string `json:"draft_content" gorm:"type:text"`
	PublishedContent string `json:"published_content" gorm:"type:text"`
	PublishedVersion int `json:"published_version" gorm:"not null;default:0"`
	PublishedAt *time.Time `json:"published_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (TenantWebsite) TableName() string { return "tenant_website" }

// TenantWebsiteVersion is an immutable snapshot of a published site revision.
type TenantWebsiteVersion struct {
	ID int64 `json:"id,string" gorm:"primaryKey"`
	TenantID int64 `json:"tenant_id,string" gorm:"not null;index;uniqueIndex:udx_tenant_website_version,priority:1"`
	Version int `json:"version" gorm:"not null;uniqueIndex:udx_tenant_website_version,priority:2"`
	ContentJSON string `json:"content_json" gorm:"type:text;not null"`
	CreatedAt time.Time `json:"created_at"`
}

func (TenantWebsiteVersion) TableName() string { return "tenant_website_version" }
