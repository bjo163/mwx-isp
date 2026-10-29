# Billing-suspend explanation trace, a bounded study (FUT-009)

Status: research note and prototype, 2026-10-29 on `dev`.
Scope: `TR-F028` billing enforcement; builds on the suspension invariant from
`ENG-032`.
Non-goal: a new UI, a stored explanation record, or a change to the decision
itself. Nothing in this study alters `billing.SuspendOverdueSubscriptions`.

## Question

An operator looking at a suspended subscription can see *that* it was
suspended, and the stored `suspension_reason` says `billing_overdue`. The
question an explanation should answer is narrower: which invoice, which due
date, which grace period, and which evidence prove that the billing rule ran —
and which of those facts the database never stored.

## What the rule actually decides

`billing.SuspendOverdueSubscriptions` (`internal/billing/service.go`) reads:

```go
db.Where("balance > 0 AND status = ?", domain.InvoiceOverdue).Find(&invoices)
...
if sub.Status != domain.SubscriptionActive ||
   !dateAfter(now, invoice.DueDate.AddDate(0, 0, sub.GraceDays)) {
    continue
}
```

and writes, in one transaction: `subscription.status = suspended`,
`subscription.suspension_reason = billing_overdue`, `radius_user.status =
disabled`, and a `subscription_suspended` billing event whose `invoice_id` and
`created_at` are the only record of which invoice triggered it and when.

So the persisted inputs are: the invoice number, balance, due date and status;
the subscription's current grace period; the linked user's status; the event
(`invoice_overdue` before `subscription_suspended`). The persisted decision
outputs are: the subscription state, the reason string, and the user status.

## Prototype

`internal/billing/suspendtrace` (analysis only; nothing imports it from product
code) returns a `Trace` with an ordered `Steps` list and a `Gaps` list:

| Step | Evidence | What it proves |
| --- | --- | --- |
| subscription state | `isp_subscription#id` | status, reason, current grace days |
| overdue invoice | `isp_invoice#id` | earliest unpaid overdue invoice, balance, due date, and the due+grace threshold evaluated at the caller's clock |
| radius user | `radius_user#id` | whether the linked user is disabled, which a billing suspension expects |
| billing events | `isp_billing_event` | `invoice_overdue` and `subscription_suspended` timestamps and invoice ids |

Example for the fixture (subscription suspended, invoice `INV-TRACE-1` due
2026-10-18, grace 3 days):

```
conclusion: subscription is suspended for an unpaid overdue invoice
steps:
  subscription state  matched      status=suspended, grace_days=3
  overdue invoice     matched      invoice INV-TRACE-1 balance=150000 due=2026-10-18
                                  grace_days=3 threshold=2026-10-21
  radius user         matched      username=trace-user status=disabled
  billing events      matched      invoice_overdue@2026-10-18T10:00:00Z(invoice#1);
                                  subscription_suspended@2026-10-21T10:00:00Z(invoice#1)
gaps:
  the grace period in force at decision time is not stored; the trace uses the
  subscription's current grace_days
```

## Findings

1. **The explanation is reconstructible for the common case.** The invoice, the
   threshold date, the user status, and both timestamps are all present, so an
   operator can be shown the exact row that caused the suspension.
2. **Two decision inputs are lost.** Grace days are read live from the
   subscription, so changing `grace_days` after the fact silently rewrites the
   explanation. The job clock is not stored either; the nearest evidence is the
   event's `created_at`.
3. **A paid invoice makes the suspension unexplainable from invoices alone.**
   Once the invoice balance reaches zero, no row satisfies the rule, yet the
   subscription may still be suspended if reactivation was not applied. The
   trace reports `not_matched` and names the gap instead of showing the paid
   invoice as if it had caused the suspension. This is the case an operator is
   most likely to ask about.
4. **Manual suspensions have no trail.** `suspension_reason` is the only record;
   there is no operator, timestamp, or comment tied to the subscription row
   itself. The trace refuses to attribute such a suspension to the billing rule.
5. **A re-enabled user diverges from the recorded decision.** If someone sets
   the linked `radius_user` back to `active` without reactivating the
   subscription, the trace flags the mismatch rather than assuming the
   disconnect happened.

## Decision

1. **Do not persist an explanation today.** The rows above already reconstruct
   the common case, and a stored explanation would be a second source of truth
   that can drift from the actual decision.
2. **If an explanation is ever exposed, expose it read-only from these rows**
   (this prototype is that function) rather than writing a new table.
3. **If the billing rule changes, record the new input it reads.** Points 2 and
   3 above are the argument: any input that is not persisted makes the
   explanation a guess. A future change that adds, for example, a minimum
   balance or an invoice-age rule must store the evaluated threshold next to
   the suspension event.

## Limits of the prototype

- It explains the *current* evaluation, not the decision as it was made; the
  caller's clock and the subscription's current grace days are substituted.
- The billing rule is duplicated in prose and in the step code rather than
  shared, because extracting it would change product code; a divergence between
  the two is possible.
- It covers only the billing rule. FUP, manual suspension, and operator-driven
  disable paths have their own inputs and are not reconstructed.
- The event query is capped at ten rows; a subscription with a long history
  shows the oldest ten, not the most relevant ones.
- Tenant scoping is the caller's: pass the `GetDB(c)` handle. The function adds
  no tenant predicate of its own.

## Verification

```
go test -count=1 ./internal/billing/suspendtrace/
ok  github.com/bjo163/mwx-isp/internal/billing/suspendtrace  1.519s

golangci-lint run ./internal/billing/suspendtrace/...
0 issues.
```

Coverage: a full billing suspension (invoice, threshold, user, events, grace
gap), a suspension whose invoice was later paid, a manual suspension that must
not be attributed to the rule, an active subscription with no history, a
re-enabled radius user, and an unknown subscription id.
