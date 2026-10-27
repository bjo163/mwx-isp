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
4. Allocates the next `PAY-...` serial atomically via `NextDocumentSerial` inside the same transaction (crash/retry policy: [`document-numbering.md`](document-numbering.md)).
5. Creates exactly one `Payment` and one `payment_received` `BillingEvent`, then updates `PaidAmount`, `Balance` and the partial/paid status in the same commit.
6. Optionally reactivates a billing-suspended subscription only after the final payment and only when no other overdue invoice remains.

All amounts are integer IDR; no floating point is introduced.

## Integer money invariants

Money is stored and compared as `int64` (`domain.InternetPackage.Price`,
`Invoice.Subtotal/Total/Balance/PaidAmount`, `Payment.Amount`), so totals never
drift through rounding. The boundaries are enforced in three places:

| Boundary | Behaviour |
| --- | --- |
| Package price input | Negative prices are rejected by validation; a JSON number outside `int64` (for example `9223372036854775808`) fails decoding with `400`. `0` is accepted for free packages, `9223372036854775807` is stored exactly. |
| Invoice creation | `Subtotal = Total = Balance = unit price` with a single `Quantity 1` item, so no multiplication can overflow. |
| `RecordPayment` | Rejects `amount <= 0` and `amount > balance` (`ErrInvalidPayment`). Because `balance <= total` at every step, `PaidAmount + Balance <= Total` and `Balance` never goes negative. |

Unit coverage lives in `internal/billing/service_test.go`
(`TestInvoiceTotalsStayExactAtInt64Boundary`,
`TestRecordPaymentRejectsNonPositiveAmount`) and
`internal/adminapi/isp_invoice_math_test.go`
(`TestSavePackagePriceBoundaries`). The API is the only input surface: there is
no manual invoice endpoint, so an invoice total can only come from a stored
package price.

## Manual payment idempotency

`POST /api/v1/isp/payments` accepts an optional `Idempotency-Key` header with
the same syntax rule as public registration: 1–64 printable ASCII characters.
The key is stored on the payment row and is unique per tenant
(`udx_isp_payment_tenant_idem`).

| Request | Result |
| --- | --- |
| First request with key `K` | `201 Created`, payment recorded |
| Identical retry with key `K` (same invoice, amount, method, reference) | `200 OK` with `Idempotent-Replay: true`; the original payment is returned and paid amount/balance are unchanged |
| Retry with key `K` and a different invoice, amount, method, or reference | `409 IDEMPOTENCY_CONFLICT`; nothing is recorded |
| Retry with key `K` after the invoice became fully paid | `200 OK` replay, not `409 INVOICE_CLOSED` |
| Two requests with different keys | two separate payments |
| Two identical requests without a key | two separate payments |

The last row is deliberate: two equal cash instalments on one invoice are
legitimate, so identical request content without an explicit key is never
treated as a retry. Retry protection is opt-in and driven by the key.

The lookup runs after the invoice row lock but before the closed/balance checks,
so a replay still responds successfully once the invoice is closed. The unique
index is the final guard: a concurrent duplicate that loses the insert race is
translated into the same duplicate path, and the handler reloads the winning
payment by key.

The admin payment dialog generates one key per dialog session and reuses it for
submits until the payment is accepted, so a lost response can be retried without
risk. `cmd/demo-seed` and internal callers that pass no key keep the old
behaviour.

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
- `Payment.Reference` is free text and is not a uniqueness key or a retry key;
  manual retries are deduplicated by `Idempotency-Key` only. Two admin-entered
  payments with the same reference and no key are treated as two distinct
  payments. Adding
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
