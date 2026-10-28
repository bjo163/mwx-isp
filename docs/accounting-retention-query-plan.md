# Accounting retention and query-plan study — 2026-10-29

Feature: TR-F011 (accounting audit query) · backlog item ENG-040.

TR-F011 requires accounting records to stay queryable for auditing while
retention cleanup stays configurable. This study answers two questions with
measurements instead of intuition:

1. Do the statements behind the accounting list, its count, and the daily
   retention delete need a new index or table partitioning?
2. What does the 90-day retention window cost in rows, bytes and buffers at
   realistic session volumes?

The harness is `tools/acctbench`, a reproducible PostgreSQL benchmark. It
seeds synthetic accounting rows, runs the admin statements, captures
`EXPLAIN (ANALYZE, BUFFERS)`, then repeats everything with candidate indexes so
the decision is anchored to plans and latency.

## Workload

`radius_accounting` holds one row per RADIUS session: the row is created at
Accounting-Start and updated at Accounting-Stop (`internal/radiusd/plugins/
accounting/handlers/start_handler.go`, `stop_handler.go`). Interim updates
refresh `radius_online`, not the accounting row, so the table grows with
sessions — not with interim packets.

The seeded statements mirror production code:

| Harness query | Production statement |
| --- | --- |
| `list_default` | `ListAccounting` default page: tenant predicate, `ORDER BY acct_start_time DESC`, `LIMIT 10` |
| `list_page_201` | same list at `OFFSET 2000` (page 201 at 10 rows per page) |
| `list_date_window` | same list with the UI 7-day `acct_start_time` range |
| `list_username_like` | `username LIKE '%…%'` substring search plus tenant predicate |
| `count_date_window` | `query.Count(&total)` behind the list, with the 7-day range |
| `retention_count` | `SchedClearExpireData` predicate: `acct_stop_time > epoch AND acct_stop_time < cutoff` per tenant |
| `retention_delete_plan` | the same predicate as a `DELETE` (planned, not executed) |

Existing indexes come from `domain.RadiusAccounting`: single-column btree
indexes on `tenant_id`, `username`, `acct_session_id`, `acct_start_time` and
`acct_stop_time`.

Synthetic fixture: uniform session start times across 90 days, 8 tenants with
equal share, 625 usernames per tenant, 40 NAS, one session in 200 still open
(`acct_stop_time` zero) so the retention epoch guard keeps real work to exclude.

## Environment

| Item | Value |
| --- | --- |
| Host | Windows 11, AMD Ryzen 5 7500F (6C/12T), 31.6 GB RAM |
| Go | 1.27.0 windows/amd64 |
| PostgreSQL | 16.9 (embedded binaries, UTF8, default `shared_buffers`, `max_parallel_workers_per_gather=2`) |
| Table size | 1,000,000 rows → 364 MB; 5,000,000 rows → 1,810 MB (table + indexes) |

Latency is the p50/p95 of 20 samples of 10 executions after three warm-up
batches, each phase on a fresh connection pool. The fresh pool matters: pgx
caches one prepared plan per SQL text per connection, so a reused pool keeps
executing the pre-index plan and hides the effect of a candidate index
(observed during the first runs of this study).

The `EXPLAIN` wall times in the plans below are higher than the measured p50
because a one-off parallel query pays worker start-up on Windows; the plans are
used for shape and buffer counts, the p50/p95 table for latency.

## Measurements

Baseline, existing indexes only:

| Query | 1M rows p50 / p95 | 5M rows p50 / p95 | 5M plan and buffers |
| --- | --- | --- | --- |
| `list_default` | 0.05 / 0.09 ms | 0.05 / 0.11 ms | Index Scan Backward on `acct_start_time`, 8 buffers |
| `list_page_201` | 1.37 / 1.44 ms | 1.46 / 1.64 ms | same index, 2,010 rows entered, 634 buffers |
| `list_date_window` | 0.05 / 0.11 ms | 0.11 / 0.17 ms | index range on `acct_start_time`, 8 buffers |
| `list_username_like` | 5.23 / 5.40 ms | 5.71 / 6.21 ms | backward scan + `username LIKE` filter, ~36k rows removed per worker, 4,685 buffers |
| `count_date_window` | 6.19 / 6.38 ms | 42.37 / 50.71 ms | parallel index scan, 16,201 matching rows per tenant, 14,059 buffers |
| `retention_count` | 0.11 / 0.15 ms | 0.11 / 0.17 ms | index range on `acct_stop_time`, 3 buffers |
| `retention_delete_plan` | — | — | `Delete` driven by `idx_radius_accounting_acct_stop_time` |

Reading the table:

- **Listing is not the cost.** The default page reads eight buffers and stays
  at 0.05 ms from 1M to 5M rows because `acct_start_time` is indexed and each
  tenant owns a contiguous slice of the index.
- **Deep paging degrades linearly, not catastrophically.** Page 201 enters
  2,010 index entries and filters 14,063 rows for the tenant predicate; 1.5 ms
  at 5M rows is still well below a UI interaction budget.
- **The 7-day count is the heaviest admin statement.** It must visit every row
  in the window for the tenant: 42 ms at 5M rows where a tenant holds 625k
  rows. This is the first statement that would need attention at larger scale.
- **Substring search does not improve or degrade with table size** because the
  planner walks the `acct_start_time` index backwards and stops after finding
  10 matches (~5.7 ms at 5M rows). Cold-cache execution was measured at ~100 ms
  in the same fixture.
- **Retention is cheap.** The cutoff predicate is served by
  `idx_radius_accounting_acct_stop_time` with three buffers; the `DELETE` uses
  the same index, and the per-tenant loop keeps each statement small.
- **Go-side allocation is negligible** (0.0 MB per statement at 10 rows per
  page). The memory that matters is shared buffers per query, shown above.

## Candidate 1 — composite index `(tenant_id, acct_start_time DESC)`

| Query | 1M speedup | 5M speedup |
| --- | --- | --- |
| `list_default` | 1.00x | 0.51x (0.05 → 0.10 ms, timer noise) |
| `list_page_201` | 1.01x | 1.08x |
| `list_date_window` | 1.01x | 1.06x |
| `list_username_like` | 1.00x | 1.05x |
| `count_date_window` | 1.00x | 1.03x |
| `retention_count` | 0.95x | 1.01x |

The planner never chooses the composite index: the single-column
`acct_start_time` index already delivers ordered pages in eight buffers, and
adding `tenant_id` as a leading column makes the index larger without changing
the plan. The index costs 30 MB at 1M rows (+8.3%) and 150 MB at 5M rows
(+8.3%) for no measured gain.

**Rejected.**

## Candidate 2 — `pg_trgm` GIN index on `username`

| Query | 1M speedup | 5M speedup | Index size |
| --- | --- | --- | --- |
| `list_username_like` | 0.27x (5.23 → 19.33 ms) | 1.01x (5.71 → 5.63 ms) | +52 MB at 1M, +250 MB at 5M |

The GIN index is chosen for the substring predicate, but each trigram of
`%user-1-000123%` has postings for a large share of the table, so the index
scan touches 8,692 buffers — as expensive as scanning the tenant's rows, and at
1M rows measurably worse. Other queries are unaffected.

**Rejected.**

## Decision

No schema change.

1. **No new index.** Existing single-column indexes serve the list, count and
   retention predicates; both candidates were measured and rejected above.
2. **No partitioning.** At 5M rows the table is 1.8 GB and the default list
   page is 0.05 ms. Declarative partitioning would complicate migrations and
   tenant-scoped cleanup for no measured gain at the volumes an operator
   deployment reaches inside the 90-day window.
3. **No archive pipeline.** Retention keeps deleting rows older than
   `radius.AccountingHistoryDays` (default 90, `0` disables). The delete is
   index-supported and the window is already configurable, so an operator who
   needs longer legal retention raises the setting; nightly database backups,
   not a second table, are what protect the deleted history. A separate archive
   store stays out of scope until an operator asks for retention beyond what a
   backup-restore can serve.

Revisit triggers, in order of likelihood:

- `count_date_window` p95 above 100 ms at the operator's real volume, or the
  table growing past ~50M rows (≈550k sessions/day at 90 days) — then re-run
  the harness and evaluate a covering index or partitioning with data.
- Substring search p95 above 250 ms on cold cache — the fix to evaluate first
  is narrowing the search semantics (prefix or exact username) rather than a
  trigram index, which this study shows does not pay for itself.
- Retention delete duration no longer negligible in scheduler logs.

## Reproduce

```bash
# Scratch PostgreSQL only: the tool seeds, indexes and truncates the table.
go run ./tools/acctbench \
  -dsn "host=127.0.0.1 port=5432 user=mwx password=... dbname=mwxbench sslmode=disable" \
  -rows 1000000 -tenants 8 -iters 20
```

`-rows 5000000` reproduces the second scale. The DSN database name must contain
`bench`, `test` or `scratch` unless `-force` is passed. `-keep` leaves the
seeded rows and candidate indexes in place for manual `EXPLAIN` work.

On this host PostgreSQL was not installed and Docker was not running, so the
measurements used `embedded-postgres` 1.34.0 (PostgreSQL 16.9) from a scratch
module outside the repository; no server or container was left running.

## Limits

- Synthetic data: uniform session starts and equal tenant shares. A skewed
  deployment (one large tenant) behaves like the single-tenant case, where the
  tenant predicate matches every row and the plans above still apply.
- Measurement is warm-cache and single-client; it does not model concurrent
  operator traffic or a cold page cache after a restart.
- No production trace or real operator volume was used, so the numbers are a
  planning baseline, not a capacity guarantee.
- The harness measures PostgreSQL only; SQLite is used by tests, not by
  production.
