# Payment Correction Policy (ENG-034)

Status: delivered 2026-10-29 on `dev`.
Scope: `TR-F028` manual invoice/payment ledger, admin/operator web console.
Non-goal: payment-gateway refunds, ERP credit notes, customer-facing
self-service corrections (see `TR-N001` in `feature-checklist.md`).

## Problem

Before this change an operator who recorded a payment against the wrong invoice,
or the wrong amount, had no supported correction path. `billing.RecordPayment`
is append-only (`internal/billing/service.go`): it locks the invoice, refuses
`amount <= 0` and `amount > balance`, and writes a `payment_received` event.
Nothing reversed a payment, so a typo could only be "fixed" by inserting
financial rows by hand — outside the transaction and audit trail that protect
the rest of the ledger.

External money movement is not available to paper over the mistake either:
`/portal/invoices/:id/payment-channel` returns `503`,
`/portal/invoices/:id/simulate-pay` returns `410`, and
`/portal/payments/webhook` returns `503` (fail-closed, pinned by
`TestUnconfiguredPaymentEndpointsFailClosed`). A correction therefore has to be
a *local ledger reversal*, not a gateway refund.

## Decision

1. **Corrections are ledger reversals, never deletions.** A reversed payment
   keeps its `payment_no`, amount, method, and `paid_at`; only its `status`
   changes to `reversed`. The invoice is restored by exactly the reversed
   amount. History is readable both ways: which payment was taken out, and what
   the invoice balance was before.
2. **The correction is admin-only.** The route is registered on the
   `requireAdmin()` group in `internal/adminapi/isp.go`, so `operator`-level
   sessions receive `403 PERMISSION_DENIED`. This matches the existing rule that
   only admins may record payments through the console.
3. **The reason is mandatory and substantive.** After trimming, the reason must
   be 8–200 runes (`correctionReasonMin` / `correctionReasonMax`); otherwise the
   request fails with `400 INVALID_CORRECTION_REASON` before any row is read for
   update.
4. **One correction per payment.** Repeating the call returns
   `409 PAYMENT_NOT_REVERSIBLE` and writes no second event. This makes the
   endpoint safe to retry from a flaky client.
5. **No external side effects.** `billing.ReversePayment` touches only the
   tenant-local `payments`, `invoices`, `billing_events`, and `sys_opr_logs`
   tables. It never calls a gateway, ERP, or notification provider.

## Status matrix

`billing.ReversePayment` derives the invoice state from the restored
`PaidAmount`, never from a client-supplied value:

| Payment being reversed | Invoice before | Invoice after | Balance after |
| --- | --- | --- | --- |
| Only payment, full amount, due date in the future | `paid` | `issued` | `Total` |
| Only payment, full amount, due date already passed | `paid` | `overdue` | `Total` |
| Last of several payments | `paid` | `partial` | previous balance + amount |
| The only partial payment | `partial` | `issued` | `Total` |
| Any payment on a `void` invoice | `void` | **rejected** `409 INVOICE_VOID` | unchanged |
| Payment not in `received` state (e.g. `pending`) | any | **rejected** `409 PAYMENT_NOT_REVERSIBLE` | unchanged |
| Payment amount inconsistent with the invoice ledger | any | **rejected** `500 PAYMENT_CORRECTION_FAILED` | unchanged |

The overdue check compares against the start of "today" in the operator's
timezone, the same convention `GenerateMonthlyInvoices` uses, so a reversal on
the due date itself does not immediately mark the invoice overdue.

The ledger guard (`payment.Amount <= 0`, `> invoice.PaidAmount`, or
`> invoice.Total`) exists because a hand-edited or partially migrated database
could otherwise be "corrected" into a negative `PaidAmount` or a balance above
the invoice total. Rejecting is safer than inventing a reconciled state.

## Transaction and audit trail

Everything below happens in one `db.Transaction`; a failure at any step rolls
back the status changes:

1. `SELECT ... FOR UPDATE` on the payment row.
2. `SELECT ... FOR UPDATE` on the invoice row, which serialises concurrent
   corrections and payments for the same invoice.
3. Validate invoice/payment state and the ledger guard.
4. `payment.status = reversed`.
5. Restore `invoice.paid_amount` / `invoice.balance`, derive the status.
6. Write a `billing_events` row: type `payment_reversed`, description
   `"<PAY-…> reversed by <operator>: <reason>"`.
7. Write a `sys_opr_logs` row: `opt_action = payment.reverse`, description
   `"<PAY-…> on invoice <INV-…>: <reason>"`, operator username and client IP.

Step 6 feeds the customer timeline (`docs/customer-timeline.md`); step 7 is the
operator audit log used by the security review. Both carry the same reason, so
a discrepancy between the two trails is detectable.

## HTTP contract

`POST /api/v1/isp/payments/:id/reverse` — admin or super only.

```json
{ "reason": "operator typed the wrong invoice" }
```

Success `200`:

```json
{
  "payment": { "id": 42, "status": "reversed", … },
  "invoice": { "id": 17, "status": "issued", "paid_amount": 0, "balance": 150000, … }
}
```

Failure codes: `INVALID_ID` (400), `INVALID_REQUEST` (400, unparsable body),
`INVALID_CORRECTION_REASON` (400), `NOT_FOUND` (404, including a payment that
belongs to another tenant), `INVOICE_VOID` (409), `PAYMENT_NOT_REVERSIBLE`
(409), `PERMISSION_DENIED` (403), `UNAUTHORIZED` (401).

A cross-tenant payment id resolves to `404`, not `403`: `GetDB(c)` applies the
tenant scope before the handler sees the row, so existence itself is not
disclosed across tenants.

## UI

`PaymentShowActions` (`web/src/resources/isp.tsx`) renders a **Reverse payment**
button only when the viewer is an admin and the payment status is `received`.
The dialog requires the same 8–200 character reason and disables the submit
button until it is satisfied. After success it refreshes the record, which now
shows the `reversed` status chip (translated as "Dikoreksi" in `id-ID`) and no
longer offers the reversal button.

## Evidence

```
go test ./internal/billing/ -run 'TestReversePayment' -count=1
ok  	github.com/bjo163/mwx-isp/internal/billing	1.781s

go test ./internal/adminapi/ -run 'TestPaymentReversal' -count=1
ok  	github.com/bjo163/mwx-isp/internal/adminapi	3.496s

cd web && npm run type-check   # tsc --noEmit, no output
cd web && npm run build        # vite build succeeded
```

Unit coverage (`internal/billing/reversal_test.go`):

- `TestReversePaymentRestoresInvoiceAcrossPaymentStates` — paid→issued,
  paid→overdue, paid→partial, partial→issued.
- `TestReversePaymentRejectsVoidInvoice` — void row of the matrix; payment and
  invoice unchanged, zero events, zero operator logs.
- `TestReversePaymentIsSingleShot` — second call returns
  `ErrPaymentNotReversible`, trail count stays at 1.
- `TestReversePaymentRequiresOperatorReason` — empty, 4-rune, 201-rune, and
  missing-operator corrections rejected before any write.
- `TestReversePaymentRejectsUnreceivedPayment` — a `pending` payment is not
  money in the ledger and cannot be corrected.
- `TestReversePaymentRejectsLedgerMismatch` — a payment that overstates the
  invoice is refused instead of producing a negative balance.

HTTP coverage (`internal/adminapi/payment_reversal_test.go`), against the real
router:

- `TestPaymentReversalRequiresAdminAndRecordsTrail` — operator `403`, short
  reason `400`, admin `200` with the expected invoice state, exactly one
  `payment_reversed` event and one `sys_opr_logs` row carrying the reason,
  repeat call `409` with the trail still at one row.
- `TestPaymentReversalStaysInsideTheTenant` — a tenant-2 payment reversed by a
  tenant-1 admin returns `404`; the foreign row and the event table are
  untouched.

## Residual limits

- The reversal path has no UI-driven end-to-end test; the browser flow is
  checked by `tsc`/`vite build` and by the HTTP tests above, not by Playwright.
- Concurrent `RecordPayment` and `ReversePayment` on the same invoice are
  serialised by `FOR UPDATE` on SQLite/PostgreSQL, but no dedicated race test
  exercises both at once; the row locks are the same ones `payment-concurrency.md`
  already relies on.
- Notifications already sent for the original payment (WhatsApp receipt) are
  not recalled. Operators are expected to send a manual correction message.
- Refunds, if an operator actually returned cash to the customer, are recorded
  outside MWX-ISP and are not expressed as a negative payment.
