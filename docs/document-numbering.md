# Document numbering policy

Invoice, payment, voucher, and ticket numbers all come from
`billing.NextDocumentSerial` (`internal/billing/service.go`). The helper upserts
one `isp_document_sequence` row per `(tenant_id, kind, period)` with
`INSERT ... ON CONFLICT (tenant_id, kind, period) DO UPDATE value = value + 1
RETURNING value`, so concurrent callers serialize on that row and each gets a
distinct number.

| Kind | Period | Number shape |
| --- | --- | --- |
| `invoice` | `YYYYMM` | `INV-YYYYMM-NNNNNN` |
| `payment` | `YYYYMM` | `PAY-YYYYMM-NNNNNN` |
| `voucher` | `YYMMDD` | `BATCH-YYMMDD-NNNNNN` |
| `ticket` | `YYMMDD` | `<PREFIX>-YYMMDD-NNNNNN` |

## Rules

1. Allocation runs inside the same transaction as the document insert. A
   rollback therefore rolls the sequence increment back with the document.
2. Numbers are unique per tenant, kind, and period. Two tenants may both hold
   `INV-202610-000001`; every unique key and lookup includes `tenant_id`.
3. Numbers are never reused after a commit, even if the document is later voided
   or deleted. An operator-visible gap therefore means a committed document took
   that number, not that an allocation was lost.

## Crash and retry behaviour

- **Crash before commit** (process dies, connection drops, transaction aborts):
  the increment is rolled back, so the retry receives the same number. This
  leaves neither a duplicate nor a gap.
- **Crash after commit, retry of the same logical work**:
  `GenerateMonthlyInvoices` hits the subscription/period unique index and skips
  the existing invoice, so no second number is allocated. A retried manual
  payment is covered by the `Idempotency-Key` contract in
  [`payment-concurrency.md`](payment-concurrency.md).
- Two retries racing each other are serialized by the sequence row; the loser
  of the invoice insert rolls back its increment.

## Verification

```bash
# Rollback releases the number and the monthly retry reuses it (SQLite unit).
go test ./internal/billing/ -run 'NextDocumentSerial|RetryReusesRolledBack' -count=1

# PostgreSQL: 8 concurrent allocations are unique, an aborted allocation
# releases its number, and the retry after the crash reuses it.
TEST_DATABASE_HOST=127.0.0.1 TEST_DATABASE_PORT=15432 \
TEST_DATABASE_USER=mwxisp TEST_DATABASE_PASSWORD=mwxisp TEST_DATABASE_NAME=postgres \
go test -tags=integration ./test/integration -run TestPostgresInvoiceNumberingConcurrentAndRollback -count=1
```

The SQLite unit tests prove the transactional rollback path; only the
PostgreSQL run proves the row-lock behaviour under real concurrency, so the
integration test must pass before a release.
