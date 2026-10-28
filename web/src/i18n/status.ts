// Shared status vocabulary rendered by StatusChip. Every value that can reach
// a chip must be listed here; the per-locale dictionaries are typed as
// StatusMessages so `tsc` rejects a locale that misses a translation. Values
// that are not listed still render, but fall back to the uppercased enum.

export const statusValues = [
  // Billing lifecycle (internal/domain/isp.go).
  'pending', 'active', 'inactive', 'suspended', 'terminated', 'billing_overdue',
  'draft', 'issued', 'partial', 'paid', 'overdue', 'void', 'received',
  // Incident lifecycle and priorities (internal/domain/isp_operations.go).
  'open', 'scheduled', 'assigned', 'in_progress', 'resolved', 'closed',
  'low', 'normal', 'medium', 'high', 'urgent', 'critical',
  // Connectivity states shared by sessions, monitors and subscribers.
  'up', 'down', 'online', 'offline', 'unknown', 'failed',
] as const;

export type StatusValue = (typeof statusValues)[number];

export type StatusMessages = Record<StatusValue, string>;
