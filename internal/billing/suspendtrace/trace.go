// Package suspendtrace reconstructs why a subscription is suspended from the
// rows MWX-ISP already stores.
//
// It is a bounded research prototype behind FUT-009. The billing job decides to
// suspend in `billing.SuspendOverdueSubscriptions`, but the decision's inputs —
// the grace period in force, the job clock, the disconnect result — are not all
// persisted. This package explains what it can prove from `isp_subscription`,
// `isp_invoice`, `isp_billing_event`, and `radius_user`, and names what it
// cannot prove instead of inventing it.
package suspendtrace

import (
	"errors"
	"fmt"
	"time"

	"github.com/bjo163/mwx-isp/internal/domain"
	"gorm.io/gorm"
)

var (
	// ErrSubscriptionNotFound is returned when the id does not resolve inside
	// the caller's tenant scope.
	ErrSubscriptionNotFound = errors.New("subscription not found")
)

// StepStatus is the outcome of one explanation step.
type StepStatus string

const (
	// Matched means the rule input was found and it supports the suspension.
	Matched StepStatus = "matched"
	// NotMatched means the input was found and it does not support the
	// suspension under current data.
	NotMatched StepStatus = "not_matched"
	// NotApplicable means the rule did not need this step.
	NotApplicable StepStatus = "not_applicable"
	// Unknown means the row needed for the step is missing or was never
	// written.
	Unknown StepStatus = "unknown"
)

// radiusUserDisabled mirrors the status billing.SuspendOverdueSubscriptions
// writes on the linked user; there is no shared domain constant for it.
const radiusUserDisabled = "disabled"

// Step is one rule input with its evidence and outcome.
type Step struct {
	Rule     string
	Detail   string
	Evidence string
	Status   StepStatus
}

// Trace is the reconstructed explanation for one subscription.
type Trace struct {
	SubscriptionID   int64
	SubscriptionNo   string
	Status           string
	SuspensionReason string
	// Conclusion is one sentence describing the current state.
	Conclusion string
	// Steps are the rule inputs in evaluation order.
	Steps []Step
	// Gaps name what cannot be reconstructed from stored rows.
	Gaps []string
}

// Explain reads the subscription and the rows that record its billing history.
//
// db must already be scoped to the tenant (for example the `GetDB(c)` handle);
// this function never widens the query. now is the evaluation clock for the
// grace-period rule, not the clock the suspension was decided on.
func Explain(db *gorm.DB, subscriptionID int64, now time.Time) (Trace, error) {
	var sub domain.Subscription
	if err := db.First(&sub, subscriptionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Trace{}, ErrSubscriptionNotFound
		}
		return Trace{}, fmt.Errorf("load subscription: %w", err)
	}

	trace := Trace{
		SubscriptionID:   sub.ID,
		SubscriptionNo:   sub.SubscriptionNo,
		Status:           sub.Status,
		SuspensionReason: sub.SuspensionReason,
	}

	trace.Steps = append(trace.Steps, Step{
		Rule:     "subscription state",
		Detail:   fmt.Sprintf("status=%s, grace_days=%d", sub.Status, sub.GraceDays),
		Evidence: fmt.Sprintf("isp_subscription#%d", sub.ID),
		Status:   subscriptionStateStatus(sub),
	})

	trace.Steps = append(trace.Steps, evaluateBillingRule(db, sub, now, &trace))
	trace.Steps = append(trace.Steps, evaluateRadiusUser(db, sub))
	trace.Steps = append(trace.Steps, evaluateEventTrail(db, sub))

	switch {
	case sub.Status != domain.SubscriptionSuspended:
		trace.Conclusion = "subscription is not suspended; the steps show what the billing rule would decide now"
	case sub.SuspensionReason == domain.SuspensionBillingOverdue:
		trace.Conclusion = "subscription is suspended for an unpaid overdue invoice"
	default:
		trace.Conclusion = "subscription is suspended, but the stored reason does not identify the billing rule"
		trace.Gaps = append(trace.Gaps,
			"a suspension outside the billing rule records neither the operator nor a reason beyond suspension_reason")
	}
	return trace, nil
}

func subscriptionStateStatus(sub domain.Subscription) StepStatus {
	if sub.Status == domain.SubscriptionSuspended {
		return Matched
	}
	return NotApplicable
}

// evaluateBillingRule checks the invoices the suspension rule reads: unpaid,
// overdue, and past due date plus the subscription's grace period.
func evaluateBillingRule(db *gorm.DB, sub domain.Subscription, now time.Time, trace *Trace) Step {
	var invoices []domain.Invoice
	err := db.Where("subscription_id = ? AND balance > 0 AND status = ?", sub.ID, domain.InvoiceOverdue).
		Order("due_date ASC, id ASC").Find(&invoices).Error
	if err != nil {
		return Step{Rule: "overdue invoice", Detail: err.Error(), Status: Unknown}
	}
	if len(invoices) == 0 {
		step := Step{
			Rule:   "overdue invoice",
			Detail: "no unpaid overdue invoice exists for this subscription now",
			Status: NotMatched,
		}
		if sub.Status == domain.SubscriptionSuspended && sub.SuspensionReason == domain.SuspensionBillingOverdue {
			trace.Gaps = append(trace.Gaps,
				"the subscription is suspended for billing but no overdue invoice matches now; the suspension predates a payment or a status change that is not recorded on the subscription")
		}
		return step
	}

	// The rule suspends as soon as one unpaid overdue invoice is past its
	// grace threshold, so the earliest invoice is the one that explains it.
	invoice := invoices[0]
	threshold := invoice.DueDate.AddDate(0, 0, sub.GraceDays)
	step := Step{
		Rule: "overdue invoice",
		Detail: fmt.Sprintf("invoice %s balance=%d due=%s grace_days=%d threshold=%s (%d unpaid overdue invoice(s))",
			invoice.InvoiceNo, invoice.Balance, invoice.DueDate.Format(time.DateOnly),
			sub.GraceDays, threshold.Format(time.DateOnly), len(invoices)),
		Evidence: fmt.Sprintf("isp_invoice#%d", invoice.ID),
		Status:   Matched,
	}
	if !now.After(threshold) {
		step.Status = NotMatched
		step.Detail += "; current clock has not passed the threshold"
	}
	trace.Gaps = append(trace.Gaps,
		"the grace period in force at decision time is not stored; the trace uses the subscription's current grace_days")
	return step
}

func evaluateRadiusUser(db *gorm.DB, sub domain.Subscription) Step {
	if sub.RadiusUserID <= 0 {
		return Step{Rule: "radius user", Detail: "subscription has no linked RADIUS user", Status: NotApplicable}
	}
	var user domain.RadiusUser
	if err := db.First(&user, sub.RadiusUserID).Error; err != nil {
		return Step{
			Rule:     "radius user",
			Detail:   fmt.Sprintf("linked user #%d is not readable", sub.RadiusUserID),
			Evidence: fmt.Sprintf("radius_user#%d", sub.RadiusUserID),
			Status:   Unknown,
		}
	}
	step := Step{
		Rule:     "radius user",
		Detail:   fmt.Sprintf("username=%s status=%s", user.Username, user.Status),
		Evidence: fmt.Sprintf("radius_user#%d", user.ID),
	}
	if sub.Status == domain.SubscriptionSuspended && user.Status != radiusUserDisabled {
		step.Status = NotMatched
		step.Detail += "; a billing suspension expects the linked user to be disabled"
	} else {
		step.Status = Matched
	}
	return step
}

func evaluateEventTrail(db *gorm.DB, sub domain.Subscription) Step {
	var events []domain.BillingEvent
	err := db.Where("subscription_id = ? AND type IN ?", sub.ID,
		[]string{"invoice_overdue", "subscription_suspended"}).
		Order("created_at ASC, id ASC").Limit(10).Find(&events).Error
	if err != nil {
		return Step{Rule: "billing events", Detail: err.Error(), Status: Unknown}
	}
	if len(events) == 0 {
		return Step{
			Rule:   "billing events",
			Detail: "no overdue or suspension event was recorded for this subscription",
			Status: Unknown,
		}
	}
	detail := ""
	for i, event := range events {
		if i > 0 {
			detail += "; "
		}
		detail += fmt.Sprintf("%s@%s(invoice#%d)", event.Type, event.CreatedAt.UTC().Format(time.RFC3339), event.InvoiceID)
	}
	return Step{
		Rule:     "billing events",
		Detail:   detail,
		Evidence: fmt.Sprintf("%d event(s)", len(events)),
		Status:   Matched,
	}
}
