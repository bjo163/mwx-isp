package domain

import "time"

// TenantPackagePresentation contains marketing metadata for one operational
// InternetPackage. It is not a source of price, billing, RADIUS, speed or FUP.
type TenantPackagePresentation struct {
	ID int64 `json:"id,string" gorm:"primaryKey"`
	TenantID int64 `json:"tenant_id,string" gorm:"not null;index;uniqueIndex:udx_tenant_package_presentation,priority:1"`
	PackageID int64 `json:"package_id,string" gorm:"not null;index;uniqueIndex:udx_tenant_package_presentation,priority:2"`
	DisplayName string `json:"display_name" gorm:"size:150"`
	ShortDescription string `json:"short_description" gorm:"size:500"`
	Category string `json:"category" gorm:"size:80"`
	Icon string `json:"icon" gorm:"size:48"`
	ImageURL string `json:"image_url" gorm:"size:500"`
	Badge string `json:"badge" gorm:"size:40"`
	Featured bool `json:"featured" gorm:"not null;default:false"`
	HighlightColor string `json:"highlight_color" gorm:"size:7"`
	Highlights string `json:"highlights" gorm:"type:text"`
	CTALabel string `json:"cta_label" gorm:"size:64"`
	Terms string `json:"terms" gorm:"size:1000"`
	DisplayOrder int `json:"display_order" gorm:"not null;default:0"`
	PublicVisible bool `json:"public_visible" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TableName returns the tenant-scoped package presentation table.
func (TenantPackagePresentation) TableName() string { return "isp_package_presentation" }
