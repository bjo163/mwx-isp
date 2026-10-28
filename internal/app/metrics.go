package app

import "sync/atomic"

// Scheduler counters exported through GetSystemMetrics. The series labels are
// fixed at compile time: no tenant, user or subscription identifier is part of a
// metric name, so the cardinality of a deployment's metrics does not grow with
// the number of tenants it serves.
var (
	fupTenantAttempts atomic.Int64
	fupTenantFailures atomic.Int64
)

// GetSystemMetrics returns a snapshot of the in-process counters keyed by their
// metric names. The key set is the whole label set: callers that expose these
// values must not add a tenant or subscriber dimension to them.
func GetSystemMetrics() map[string]int64 {
	return map[string]int64{
		"fup_scheduler_tenant_attempts_total": fupTenantAttempts.Load(),
		"fup_scheduler_tenant_failures_total": fupTenantFailures.Load(),
	}
}
