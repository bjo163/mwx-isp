# Payment Concurrency and Replay (H04)

Money state is written only by `billing.RecordPayment`
(`internal/billing/service.go`). The admin handler is the sole HTTP entry point;
`/portal/invoices/:id/simulate-pay` is disabled (`410`) and
`/portal/payments/webhook` is unconfigured (`503`).

## Invariants

`RecordPayment` runs in one transaction and:

1. Locks the invoice row with `SELECT ... FOR UPDATE` before reading its balance.
2. Rejects void/paid invoices and any invoice with `balance <= 0` (`ErrInvoiceClosed`).
3. Rejects `amount > balance` (`ErrInvalidPayment`), so a balance can never go negative.
4. Allocates the next `PAY-...` serial atomically via `NextDocumentSerial` inside the same transaction.
5. Creates exactly one `Payment` and one `payment_received` `BillingEvent`, then updates `PaidAmount`, `Balance` and the partial/paid status in the same commit.
6. Optionally reactivates a billing-suspended subscription only after the final payment and only when no other overdue invoice remains.

All amounts are integer IDR; no floating point is introduced.

## Scenario coverage

`test/integration/payment_concurrency_test.go`
(`TestPostgresConcurrentPaymentsDoNotOverbookInvoice`) runs against PostgreSQL
because SQLite does not provide real row locks:

| Step | Expected result |
| --- | --- |
| Two concurrent `POST /isp/payments` of 60,000 against a 100,000 invoice | Exactly one `201`; the other `400` (amount exceeds remaining balance) or `409` (closed) |
| Invoice after both requests | `Balance = 40,000`, `PaidAmount = 60,000`, `Status = partial` |
| Row counts | Exactly 1 `Payment` and 1 `payment_received` `BillingEvent` |
| Remaining 40,000 | `201`, `Balance = 0`, `Status = paid` |
| Another 40,000 afterwards | `409 INVOICE_CLOSED`, no extra rows |
| Final counts | 2 payments, 2 events |

The test asserts counts in the database, so a duplicate booking from either
concurrent request fails the test even if both HTTP responses look plausible.

## Boundaries

- No external payment provider is integrated yet; the webhook is intentionally
  unavailable, so there is no provider event-replay surface to exercise.
- `Payment.Reference` is free text and is not a uniqueness key. Two admin-entered
  payments with the same reference are treated as two distinct payments. Adding
  provider integration must introduce a uniqueness key on
  `(tenant_id, provider, provider_event_id)` and reject duplicate events before
  calling `RecordPayment`.
- The concurrency guarantee relies on the database row lock; SQLite unit tests
  cannot prove it.

## Verification

```bash
docker compose -f docker-compose.test.yml up -d
TEST_DATABASE_HOST=127.0.0.1 TEST_DATABASE_PORT=15432 \
TEST_DATABASE_USER=mwxisp TEST_DATABASE_PASSWORD=mwxisp TEST_DATABASE_NAME=postgres \
go test -tags=integration ./test/integration -run TestPostgresConcurrentPaymentsDoNotOverbookInvoice -count=1
docker compose -f docker-compose.test.yml down -v
```
