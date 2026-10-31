package app

import (
	"sync/atomic"

	"github.com/bjo163/mwx-isp/internal/billing"
)

// Scheduler counters exported through GetSystemMetrics. The series labels are
// fixed at compile time: no tenant, user or subscription identifier is part of a
// metric name, so the cardinality of a deployment's metrics does not grow with
// the number of tenants it serves.
var (
	fupTenantAttempts atomic.Int64
	fupTenantFailures atomic.Int64
)

// GetSystemMetrics returns a snapshot of the in-process counters keyed by their
// metric names, both from this package and from the services it schedules. The
// key set is the whole label set: callers that expose these values must not add
// a tenant or subscriber dimension to them.
func GetSystemMetrics() map[string]int64 {
	return map[string]int64{
		"fup_scheduler_tenant_attempts_total": fupTenantAttempts.Load(),
		"fup_scheduler_tenant_failures_total": fupTenantFailures.Load(),
		// An overdue invoice with no subscription row behind it is skipped by the
		// billing enforcement sweep rather than stopping it; the count is the
		// signal that the debt is still sitting there with nothing to suspend.
		"billing_invoice_without_subscription_total": billing.OrphanInvoices(),
	}
}
