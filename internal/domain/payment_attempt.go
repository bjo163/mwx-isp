package domain

import "time"

// PaymentAttempt is a provider-side checkout attempt. It is separate from the
// accounting Payment row: creating a checkout never records money as received.
type PaymentAttempt struct {
	ID                    int64     `json:"id,string" gorm:"primaryKey"`
	TenantID              int64     `json:"-" gorm:"not null;default:1;uniqueIndex:udx_payment_attempt_tenant_order,priority:1;uniqueIndex:udx_payment_attempt_tenant_idem,priority:1;index"`
	InvoiceID             int64     `json:"invoice_id,string" gorm:"not null;index"`
	CustomerID            int64     `json:"customer_id,string" gorm:"not null;index"`
	Provider              string    `json:"provider" gorm:"size:24;not null;index"`
	OrderID               string    `json:"order_id" gorm:"size:50;not null;uniqueIndex:udx_payment_attempt_tenant_order,priority:2"`
	IdempotencyKey        *string   `json:"-" gorm:"size:64;uniqueIndex:udx_payment_attempt_tenant_idem,priority:2"`
	ProviderTransactionID string    `json:"provider_transaction_id,omitempty" gorm:"size:100;index"`
	Amount                int64     `json:"amount" gorm:"not null"`
	Currency              string    `json:"currency" gorm:"size:8;not null;default:IDR"`
	Status                string    `json:"status" gorm:"size:32;not null;index"`
	RawStatus             string    `json:"raw_status,omitempty" gorm:"size:32"`
	SnapToken             string    `json:"-" gorm:"size:128"`
	RedirectURL           string    `json:"redirect_url,omitempty" gorm:"size:512"`
	PaymentID             int64     `json:"payment_id,string,omitempty" gorm:"index"`
	FailureReason         string    `json:"failure_reason,omitempty" gorm:"size:255"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TableName returns the provider checkout-attempt table.
func (PaymentAttempt) TableName() string { return "isp_payment_attempt" }
