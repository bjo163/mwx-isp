package domain

import "time"

// TenantDomain records a custom hostname claimed by one tenant. A hostname
// resolves to a tenant only after its DNS TXT ownership challenge is verified.
type TenantDomain struct {
	ID               int64      `json:"id,string" gorm:"primaryKey"`
	TenantID         int64      `json:"tenant_id,string" gorm:"not null;index;uniqueIndex:udx_tenant_domain_tenant_host,priority:1"`
	Hostname         string     `json:"hostname" gorm:"size:253;not null;uniqueIndex;uniqueIndex:udx_tenant_domain_tenant_host,priority:2"`
	VerificationToken string    `json:"verification_token" gorm:"size:64;not null"`
	VerifiedAt       *time.Time `json:"verified_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName returns the tenant custom-domain mapping table.
func (TenantDomain) TableName() string { return "tenant_domain" }
