package billing

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bjo163/mwx-isp/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	// ErrInvalidPayment indicates that a payment is non-positive or exceeds the remaining invoice balance.
	ErrInvalidPayment = errors.New("payment must be positive and no greater than invoice balance")
	// ErrInvoiceClosed indicates that an invoice is void, paid, or otherwise has no remaining balance.
	ErrInvoiceClosed = errors.New("invoice cannot accept payment in its current state")
	// ErrDuplicatePayment indicates that a payment with the same idempotency key
	// already exists for the tenant. No money state was changed.
	ErrDuplicatePayment = errors.New("payment with this idempotency key already recorded")
	// ErrIdempotencyConflict indicates that an existing payment uses the same
	// idempotency key for different invoice, amount, method, or reference.
	ErrIdempotencyConflict = errors.New("idempotency key reused with different payment details")
	// ErrPaymentNotReversible indicates that a payment is not in the received
	// state, for example because it was already reversed.
	ErrPaymentNotReversible = errors.New("payment cannot be reversed in its current state")
	// ErrInvoiceVoided indicates that a payment's invoice is void. A void invoice
	// carries no collectible balance, so reversing a payment against it would
	// create an unreconciled credit instead of restoring a debt.
	ErrInvoiceVoided = errors.New("invoice is void and cannot take a payment correction")
	// ErrInvalidCorrectionReason indicates that the operator-supplied reason is
	// missing or longer than the audit trail accepts.
	ErrInvalidCorrectionReason = errors.New("correction reason must be between 8 and 200 characters")
)

// Bounds for the operator-supplied correction reason.
const (
	correctionReasonMin = 8
	correctionReasonMax = 200
)

// GenerateMonthlyInvoices creates an invoice for every active subscription
// whose billing day has arrived in the current month and whose start date is not
// later than that day. The period begins on the billing day and ends the day
// before the next billing month. dueDays is limited to 0..90 and determines the
// invoice due date. A unique subscription/period constraint makes repeated
// calls idempotent. The function returns the number of inserted invoices and
// the first database error; callers should run it from a single scheduler.
func GenerateMonthlyInvoices(db *gorm.DB, now time.Time, dueDays int) (int, error) {
	return generateMonthlyInvoices(db, now, dueDays, nil)
}

// GenerateMonthlyInvoicesForSubscriptions creates current-period invoices for
// only the supplied subscription IDs. It uses the same sequence allocation,
// item creation, and billing-event behavior as GenerateMonthlyInvoices. An
// empty ID list is rejected so callers cannot accidentally invoice every
// active subscription.
func GenerateMonthlyInvoicesForSubscriptions(db *gorm.DB, now time.Time, dueDays int, subscriptionIDs []int64) (int, error) {
	if len(subscriptionIDs) == 0 {
		return 0, fmt.Errorf("at least one subscription ID is required")
	}
	return generateMonthlyInvoices(db, now, dueDays, subscriptionIDs)
}

func generateMonthlyInvoices(db *gorm.DB, now time.Time, dueDays int, subscriptionIDs []int64) (int, error) {
	if dueDays < 0 || dueDays > 90 {
		return 0, fmt.Errorf("due days must be between 0 and 90")
	}
	var subscriptions []domain.Subscription
	query := db.Where("status = ?", domain.SubscriptionActive)
	if len(subscriptionIDs) > 0 {
		query = query.Where("id IN ?", subscriptionIDs)
	}
	if err := query.Find(&subscriptions).Error; err != nil {
		return 0, err
	}
	created := 0
	for _, sub := range subscriptions {
		if sub.BillingDay < 1 || sub.BillingDay > 28 || now.Day() < sub.BillingDay {
			continue
		}
		invoiceDate := time.Date(now.Year(), now.Month(), sub.BillingDay, 0, 0, 0, 0, now.Location())
		if dateAfter(sub.StartDate, invoiceDate) {
			continue
		}
		periodStart := invoiceDate
		periodEnd := invoiceDate.AddDate(0, 1, -1)
		var count int64
		if err := db.Model(&domain.Invoice{}).Where("subscription_id = ? AND period_start = ?", sub.ID, periodStart).Count(&count).Error; err != nil {
			return created, err
		}
		if count > 0 {
			continue
		}
		var customer domain.Customer
		var pkg domain.InternetPackage
		if err := db.First(&customer, sub.CustomerID).Error; err != nil {
			return created, err
		}
		if err := db.First(&pkg, sub.PackageID).Error; err != nil {
			return created, err
		}
		invoice := domain.Invoice{
			CustomerID: sub.CustomerID, SubscriptionID: sub.ID,
			InvoiceDate: invoiceDate, DueDate: invoiceDate.AddDate(0, 0, dueDays),
			PeriodStart: periodStart, PeriodEnd: periodEnd,
			Subtotal: pkg.Price, Total: pkg.Price, Balance: pkg.Price,
			Status: domain.InvoiceIssued,
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			serial, err := nextMonthlySerial(tx, "invoice", invoiceDate.Format("200601"))
			if err != nil {
				return err
			}
			invoice.InvoiceNo = fmt.Sprintf("INV-%s-%06d", invoiceDate.Format("200601"), serial)
			if err := tx.Create(&invoice).Error; err != nil {
				return err
			}
			item := domain.InvoiceItem{InvoiceID: invoice.ID, Description: pkg.Name, Quantity: 1, UnitPrice: pkg.Price, Total: pkg.Price}
			if err := tx.Create(&item).Error; err != nil {
				return err
			}
			return tx.Create(&domain.BillingEvent{CustomerID: customer.ID, SubscriptionID: sub.ID, InvoiceID: invoice.ID, Type: "invoice_generated", Description: invoice.InvoiceNo, CreatedAt: now}).Error
		})
		if err != nil {
			if db.Migrator().HasTable(&domain.Invoice{}) && isUniqueViolation(err) {
				continue
			}
			return created, err
		}
		created++
	}
	return created, nil
}

// ProcessOverdueInvoices marks unpaid invoices overdue once the current date
// passes the invoice due date. Each status change and its audit event are saved
// in one transaction. It returns the first database error encountered.
func ProcessOverdueInvoices(db *gorm.DB, now time.Time) error {
	var invoices []domain.Invoice
	if err := db.Where("balance > 0 AND status IN ?", []string{domain.InvoiceIssued, domain.InvoicePartial, domain.InvoiceOverdue}).Find(&invoices).Error; err != nil {
		return err
	}
	for _, invoice := range invoices {
		if !dateAfter(now, invoice.DueDate) {
			continue
		}
		if invoice.Status != domain.InvoiceOverdue {
			if err := db.Transaction(func(tx *gorm.DB) error {
				if err := tx.Model(&invoice).Where("status <> ?", domain.InvoiceOverdue).Update("status", domain.InvoiceOverdue).Error; err != nil {
					return err
				}
				return tx.Create(&domain.BillingEvent{CustomerID: invoice.CustomerID, SubscriptionID: invoice.SubscriptionID, InvoiceID: invoice.ID, Type: "invoice_overdue", Description: invoice.InvoiceNo, CreatedAt: now}).Error
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

// SuspendOverdueSubscriptions suspends active subscriptions when an overdue
// invoice remains unpaid after that subscription's grace period. The
// subscription status, billing suspension reason, linked RadiusUser disable,
// and billing event are updated in one transaction. It does not send
// Disconnect-Requests; callers must disconnect matching online sessions after
// this function succeeds. It returns the first database error encountered.
func SuspendOverdueSubscriptions(db *gorm.DB, now time.Time) error {
	var invoices []domain.Invoice
	if err := db.Where("balance > 0 AND status = ?", domain.InvoiceOverdue).Find(&invoices).Error; err != nil {
		return err
	}
	for _, invoice := range invoices {
		var sub domain.Subscription
		if err := db.First(&sub, invoice.SubscriptionID).Error; err != nil {
			return err
		}
		if sub.Status != domain.SubscriptionActive || !dateAfter(now, invoice.DueDate.AddDate(0, 0, sub.GraceDays)) {
			continue
		}
		err := db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&sub).Where("status = ?", domain.SubscriptionActive).Updates(map[string]interface{}{"status": domain.SubscriptionSuspended, "suspension_reason": domain.SuspensionBillingOverdue, "updated_at": now}).Error; err != nil {
				return err
			}
			if sub.RadiusUserID > 0 {
				if err := tx.Model(&domain.RadiusUser{}).Where("id = ?", sub.RadiusUserID).Update("status", "disabled").Error; err != nil {
					return err
				}
			}
			return tx.Create(&domain.BillingEvent{CustomerID: sub.CustomerID, SubscriptionID: sub.ID, InvoiceID: invoice.ID, Type: "subscription_suspended", Description: domain.SuspensionBillingOverdue, CreatedAt: now}).Error
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// RecordPayment atomically records a positive payment no greater than the
// invoice balance and updates invoice balance and status. The invoice row is
// locked for update where supported by the database to prevent concurrent
// overpayment. If autoReactivate is true, the final payment reactivates a
// subscription suspended for billing only when it has no other overdue invoice;
// manual suspensions are never cleared here. payment receives its generated
// number, customer ID, timestamp, and received status on success. The function
// returns ErrInvalidPayment, ErrInvoiceClosed, or a wrapped database error.
//
// When payment.IdempotencyKey is set, a retry that reuses the key for the same
// invoice, amount, method, and reference records nothing and returns
// ErrDuplicatePayment before the invoice state is examined, so replaying a
// payment that closed the invoice still reports a duplicate instead of a closed
// invoice. Reusing the key with different details returns ErrIdempotencyConflict.
// Payments without a key are always recorded as new payments, which keeps two
// legitimately identical cash payments separate.
func RecordPayment(db *gorm.DB, payment *domain.Payment, now time.Time, autoReactivate bool) error {
	if payment.Amount <= 0 {
		return ErrInvalidPayment
	}
	return db.Transaction(func(tx *gorm.DB) error {
		var invoice domain.Invoice
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, payment.InvoiceID).Error; err != nil {
			return err
		}
		if payment.IdempotencyKey != nil {
			var existing domain.Payment
			err := tx.Where("tenant_id = ? AND idempotency_key = ?", invoice.TenantID, *payment.IdempotencyKey).First(&existing).Error
			if err == nil {
				if existing.InvoiceID != payment.InvoiceID || existing.Amount != payment.Amount ||
					existing.Method != payment.Method || existing.Reference != payment.Reference {
					return ErrIdempotencyConflict
				}
				return ErrDuplicatePayment
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
		}
		if invoice.Status == domain.InvoiceVoid || invoice.Status == domain.InvoicePaid || invoice.Balance <= 0 {
			return ErrInvoiceClosed
		}
		if payment.Amount > invoice.Balance {
			return ErrInvalidPayment
		}
		payment.CustomerID = invoice.CustomerID
		payment.PaidAt = now
		payment.Status = domain.PaymentReceived
		serial, err := nextMonthlySerial(tx, "payment", now.Format("200601"))
		if err != nil {
			return err
		}
		payment.PaymentNo = fmt.Sprintf("PAY-%s-%06d", now.Format("200601"), serial)
		if err := tx.Create(payment).Error; err != nil {
			// A concurrent request with the same key won the unique index race;
			// the caller can reload it by key.
			if payment.IdempotencyKey != nil && isUniqueViolation(err) {
				return ErrDuplicatePayment
			}
			return err
		}
		invoice.PaidAmount += payment.Amount
		invoice.Balance -= payment.Amount
		if invoice.Balance == 0 {
			invoice.Status = domain.InvoicePaid
		} else {
			invoice.Status = domain.InvoicePartial
		}
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}
		if err := tx.Create(&domain.BillingEvent{CustomerID: invoice.CustomerID, SubscriptionID: invoice.SubscriptionID, InvoiceID: invoice.ID, Type: "payment_received", Description: payment.PaymentNo, CreatedAt: now}).Error; err != nil {
			return err
		}
		if invoice.Status == domain.InvoicePaid && autoReactivate && invoice.SubscriptionID > 0 {
			var sub domain.Subscription
			if err := tx.First(&sub, invoice.SubscriptionID).Error; err != nil {
				return err
			}
			if sub.Status == domain.SubscriptionSuspended && sub.SuspensionReason == domain.SuspensionBillingOverdue {
				var remaining int64
				startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				if err := tx.Model(&domain.Invoice{}).Where("subscription_id = ? AND id <> ? AND balance > 0 AND status IN ? AND due_date < ?", sub.ID, invoice.ID, []string{domain.InvoiceIssued, domain.InvoicePartial, domain.InvoiceOverdue}, startOfToday).Count(&remaining).Error; err != nil {
					return err
				}
				if remaining > 0 {
					return nil
				}
				sub.Status = domain.SubscriptionActive
				sub.SuspensionReason = ""
				sub.UpdatedAt = now
				if err := tx.Save(&sub).Error; err != nil {
					return err
				}
				if sub.RadiusUserID > 0 {
					if err := tx.Model(&domain.RadiusUser{}).Where("id = ?", sub.RadiusUserID).Update("status", "enabled").Error; err != nil {
						return err
					}
				}
				if err := tx.Create(&domain.BillingEvent{CustomerID: sub.CustomerID, SubscriptionID: sub.ID, InvoiceID: invoice.ID, Type: "subscription_reactivated", Description: "billing_overdue payment settled", CreatedAt: now}).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func nextMonthlySerial(tx *gorm.DB, kind, period string) (int64, error) {
	return NextDocumentSerial(tx, kind, period)
}

// PaymentCorrection identifies the operator who reverses a payment and records
// why. Operator and OperatorIP are written to the operator audit log; Reason is
// written to both the billing event and that log.
type PaymentCorrection struct {
	// Operator is the authenticated operator username.
	Operator string
	// OperatorIP is the request source address.
	OperatorIP string
	// Reason is the operator justification, trimmed before storage.
	Reason string
}

// ReversePayment records an operator correction for a received manual payment.
//
// The correction is a local ledger reversal: it never calls a payment gateway,
// ERP, or any other external system. The payment keeps its number, amount,
// method and timestamp and moves to the reversed state, while the invoice paid
// amount and balance are adjusted by exactly the payment amount. The resulting
// invoice status is derived from the corrected balance: a remaining paid amount
// keeps the invoice partial, and a fully reverted invoice returns to issued, or
// to overdue when its due date has already passed. A void invoice is rejected
// because it no longer carries a collectible balance, and reversing the same
// payment twice returns ErrPaymentNotReversible.
//
// The payment update, the invoice update, one payment_reversed billing event
// and one sys_opr_log row are written in a single transaction, so a failed
// correction leaves no partial audit trail.
func ReversePayment(db *gorm.DB, paymentID int64, correction PaymentCorrection, now time.Time) (*domain.Payment, *domain.Invoice, error) {
	reason := strings.TrimSpace(correction.Reason)
	if n := utf8.RuneCountInString(reason); n < correctionReasonMin || n > correctionReasonMax {
		return nil, nil, ErrInvalidCorrectionReason
	}
	operator := strings.TrimSpace(correction.Operator)
	if operator == "" {
		return nil, nil, fmt.Errorf("payment correction requires an authenticated operator")
	}

	var payment domain.Payment
	var invoice domain.Invoice
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&payment, paymentID).Error; err != nil {
			return err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&invoice, payment.InvoiceID).Error; err != nil {
			return err
		}
		if invoice.Status == domain.InvoiceVoid {
			return ErrInvoiceVoided
		}
		if payment.Status != domain.PaymentReceived {
			return ErrPaymentNotReversible
		}
		if payment.Amount <= 0 || payment.Amount > invoice.PaidAmount || payment.Amount > invoice.Total {
			return fmt.Errorf("payment %s does not match the invoice ledger", payment.PaymentNo)
		}

		payment.Status = domain.PaymentReversed
		payment.UpdatedAt = now
		if err := tx.Save(&payment).Error; err != nil {
			return err
		}

		invoice.PaidAmount -= payment.Amount
		invoice.Balance += payment.Amount
		if invoice.PaidAmount <= 0 {
			invoice.PaidAmount = 0
			startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			if invoice.DueDate.Before(startOfToday) {
				invoice.Status = domain.InvoiceOverdue
			} else {
				invoice.Status = domain.InvoiceIssued
			}
		} else {
			invoice.Status = domain.InvoicePartial
		}
		invoice.UpdatedAt = now
		if err := tx.Save(&invoice).Error; err != nil {
			return err
		}

		if err := tx.Create(&domain.BillingEvent{
			CustomerID:     invoice.CustomerID,
			SubscriptionID: invoice.SubscriptionID,
			InvoiceID:      invoice.ID,
			Type:           "payment_reversed",
			Description:    fmt.Sprintf("%s reversed by %s: %s", payment.PaymentNo, operator, reason),
			CreatedAt:      now,
		}).Error; err != nil {
			return err
		}
		return tx.Create(&domain.SysOprLog{
			OprName:   operator,
			OprIp:     strings.TrimSpace(correction.OperatorIP),
			OptAction: "payment.reverse",
			OptDesc:   fmt.Sprintf("%s on invoice %s: %s", payment.PaymentNo, invoice.InvoiceNo, reason),
			OptTime:   now,
		}).Error
	})
	if err != nil {
		return nil, nil, err
	}
	return &payment, &invoice, nil
}

// NextDocumentSerial atomically allocates the next serial for a document kind
// and period within the tenant scope attached to tx. Call it inside the same
// transaction that persists the numbered record.
func NextDocumentSerial(tx *gorm.DB, kind, period string) (int64, error) {
	sequence := domain.DocumentSequence{Kind: kind, Period: period, Value: 1}
	err := tx.Clauses(
		clause.OnConflict{
			Columns: []clause.Column{{Name: "tenant_id"}, {Name: "kind"}, {Name: "period"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"value": gorm.Expr("isp_document_sequence.value + 1"),
			}),
		},
		clause.Returning{Columns: []clause.Column{{Name: "value"}}},
	).Create(&sequence).Error
	return sequence.Value, err
}

func dateAfter(a, b time.Time) bool {
	y1, m1, d1 := a.Date()
	y2, m2, d2 := b.Date()
	return time.Date(y1, m1, d1, 0, 0, 0, 0, a.Location()).After(time.Date(y2, m2, d2, 0, 0, 0, 0, b.Location()))
}

func isUniqueViolation(err error) bool {
	return err != nil && (errors.Is(err, gorm.ErrDuplicatedKey) || containsUniqueConstraint(err.Error()))
}

func containsUniqueConstraint(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "unique constraint") || strings.Contains(message, "duplicate key")
}
