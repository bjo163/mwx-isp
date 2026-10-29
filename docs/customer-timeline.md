# Customer 360 activity timeline

This document defines the read-only customer timeline introduced for issue #54
(AUDIT F02). It merges existing tenant-scoped records into one chronological
view so an operator can follow an incident from invoice to suspension,
disconnect, payment, or support ticket. The endpoint stores no new events: every
item maps to a row that already exists.

## Endpoint

```
GET /api/v1/isp/customers/:id/timeline
```

Query parameters:

| Parameter | Meaning |
| --- | --- |
| `page` | 1-based page number; defaults to 1. |
| `pageSize` / `perPage` | Page size; defaults to 20, capped at 100. |
| `from` | Lower bound. Defaults to `to - 90 days`. |
| `to` | Upper bound. Defaults to now. A date-only value covers the whole day. |
| `sources` | Comma-separated subset of `billing,invoice,payment,session,ticket`. |

`from`, `to`, and `sources` are validated. An unknown source, an unparsable
date, or `from > to` returns HTTP 400 with an explicit error code. The customer
lookup is tenant-scoped: a customer that belongs to another tenant returns HTTP
404, exactly like a missing record.

## Event contract

| Field | Meaning |
| --- | --- |
| `event_id` | Stable identifier derived from the source row, e.g. `payment:42:received`. |
| `source` | `billing`, `invoice`, `payment`, `session`, or `ticket`. |
| `type` | Normalized event type, e.g. `session_disconnected`. |
| `occurred_at` | Persisted timestamp of the source row, always serialized in UTC. |
| `tenant_id`, `customer_id`, `subscription_id` | Ownership identifiers. |
| `summary` | Operator-readable description built from the row. |
| `source_path` | React Admin path of the underlying document. |
| `severity` | `info`, `success`, or `warning`, derived from the stored status. |

## Sources and timestamps

| Source | Row | `occurred_at` | Types |
| --- | --- | --- | --- |
| Billing | `BillingEvent` | `created_at` | Stored event type, e.g. `subscription_suspended`. |
| Invoice | `Invoice` | `invoice_date` | `invoice_issued`. Backdated invoices therefore keep their issuance date instead of the row insertion time. |
| Payment | `Payment` | `paid_at` | `payment_received`. A payment whose status is `reversed` keeps the same type and `event_id` but is summarized as reversed with `warning` severity; the correction itself appears as the `payment_reversed` billing event at its own timestamp. |
| Session | `RadiusOnline` | `acct_start_time` | `session_connected`. |
| Session | `RadiusAccounting` | `acct_stop_time` | `session_disconnected`; rows without a stop time are skipped. |
| Ticket | `TroubleTicket` | `created_at` | `ticket_created`. |
| Ticket | `TroubleTicket` | `resolved_at` | `ticket_resolved`; replaces the update event when present. |
| Ticket | `TroubleTicket` | `updated_at` | `ticket_updated`; only for unresolved tickets whose `updated_at` is after `created_at`. |

The status shown in an invoice summary is the current row status (for example
`overdue`), so it is a snapshot, not a historical transition. The timeline never
invents an event, never derives an offline state from a missing live session,
and never serializes RADIUS credentials or private attributes.

## Ordering and pagination

Events are ordered by `occurred_at DESC, source ASC, event_id ASC`. The tie-break
is applied both by each source query (`... DESC, id ASC`) and by the in-memory
merge, so pages are deterministic and no event repeats or disappears when a page
boundary falls inside a group of equal timestamps. Each source is fetched up to
`page * pageSize` rows, which is sufficient to fill the requested page after the
merge. The response carries `pagination.page`, `pagination.page_size`, and
`pagination.total` inside the payload because the shared `apiRequest` helper
unwraps the `data` envelope and drops `meta`.

## UI

`web/src/components/CustomerTimeline.tsx` renders the timeline inside the
customer detail page (`CUSTOMER 360 TIMELINE` panel). It exposes source toggles,
date inputs, previous/next paging, severity chips, and a link to each source
document.

## Guardrails and evidence

- Tenant isolation is enforced in every query, including the username-to-
  subscription join used for sessions. Unit tests cover a foreign tenant that
  owns a session with the same username.
- Stable ordering and cross-page deduplication are covered by
  `TestCustomerTimelinePaginationIsDeterministic`.
- Date/source filtering and the 90-day default window are covered by
  `TestCustomerTimelineDateRangeAndSourceFilters`.
- PostgreSQL persistence, persisted ticket timestamps, and response shape are
  covered by `TestPostgresCustomerTimelineAggregatesPersistedRows`.
