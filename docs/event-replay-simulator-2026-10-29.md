# Event replay simulator, a bounded study (FUT-022)

Status: research note and prototype, 2026-10-29 on `dev`.
Scope: `TR-F028` billing ledger and `radius_accounting` session history;
builds on the suspension invariant from `ENG-032`.
Non-goal: a new admin endpoint, a stored replay snapshot, or a re-execution of
the original jobs. Nothing in this study changes billing, RADIUS, or the
scheduler.

## Question

When an invoice balance or a live session looks wrong, an operator can read the
current row but cannot see how it got there. The billing trail is a list of
events, and RADIUS history is a list of accounting rows; neither is a state
machine an operator can step through by hand. Can the rows MWX-ISP already
stores be replayed deterministically into the state they imply, so a
disagreement with the stored row becomes a concrete reproduction step?

## Evidence available

| Evidence | Rows | What it carries |
| --- | --- | --- |
| Billing trail | `isp_billing_event` | Event type, invoice/payment reference in `description`, `created_at`, time-ordered `id` |
| Payment ledger | `isp_payment` | Payment number, amount, `received`/`reversed` status, `paid_at` |
| Invoice row | `isp_invoice` | `total`, `paid_amount`, `balance`, `status`, `due_date` |
| Session ledger | `radius_accounting` | One row per Accounting-Start, counters folded in by Accounting-Stop |
| Live state | `radius_online` | Sessions the server still considers online |
| Required work | `radius_session_action` | Queued/attempted Disconnect-Requests with status and attempts |
| Operator trail | `radius_session_action_audit` | CoA/Disconnect attempts with success and trigger time |

Two gaps are structural and shape the whole design:

- Event rows carry no amounts. The amount of a `payment_received` event is the
  amount of the payment row its description names.
- Interim-Update packets are not stored as rows (`internal/radiusd/plugins/accounting/handlers/update_handler.go`).
  `radius_accounting` therefore holds the start row and the final counters, not
  the traffic history.

## Prototype

`internal/replay` is read-only and imports no application package.

`ReplayInvoice` (`internal/replay/replay.go`) sorts the events by `created_at`
then `id` and folds:

- `payment_received` → look up the payment number in the description, add its
  amount;
- `payment_reversed` → subtract the amount, but only when that payment was
  received and not already reversed;
- every other event type is counted and ignored for money.

The resulting paid amount yields `balance = total - paid` and a status derived
with the same rule `billing.ReversePayment` uses: `paid` at a zero balance,
`partial` between, otherwise `issued` or `overdue` depending on the start of
today. `issued` and `overdue` are compared as one class because their difference
is scheduler timing, not ledger state.

`ReplaySession` sorts accounting rows by NAS start time and re-derives the
lifetime, then checks the live row, the action queue, and the audit trail
against it.

## Divergence catalogue

Every kind below has a distinct physical cause, so a report can be acted on
rather than merely read.

| Kind | Cause it reproduces |
| --- | --- |
| `paid_amount_mismatch`, `balance_mismatch`, `status_mismatch` | Invoice row edited or partially migrated outside `billing` |
| `missing_payment_event` | Payment written without its trail row (or trail pruned) |
| `unknown_payment_event` | Event naming a payment number that no longer exists |
| `reversal_without_payment` | Double reversal, or a reversed event for a payment still marked `received` |
| `duplicate_payment_event` | The same payment received twice in the trail |
| `negative_paid_amount` | A payment row with a non-positive amount |
| `event_order` | Backdated insert, clock step, or manual import (detected by an `id` decrease in timestamp order) |
| `stop_before_start`, `session_time_mismatch` | NAS clock step or a malformed Accounting-Stop |
| `counter_regression` | Reused `Acct-Session-Id` whose later counters are lower |
| `overlapping_session_lifetime` | Two lifetimes for one identifier overlap, or a row starts while the previous is open |
| `orphan_open_session` | The Accounting-Stop never reached the server and the online row is gone |
| `closed_accounting_with_online_row` | Accounting-Stop recorded but the online row survived |
| `online_without_accounting` | Online row with no Accounting-Start row at all |
| `stale_open_session` | Open row older than 30 days |
| `disconnect_not_delivered` | Queued Disconnect-Request still `pending`/`running`/`failed` while the session is online |
| `disconnect_did_not_end_session` | Disconnect reported success (queue or audit) while the session is online |

## Example

A tampered invoice (stored `paid_amount = 150000`, `balance = 0`, `paid`, while
the trail holds one 50 000 payment) replays as:

```
INV-REPLAY-1: paid_amount_mismatch: stored paid amount is 150000, the trail replays 50000
INV-REPLAY-1: balance_mismatch: stored balance is 0, total minus replayed paid is 100000
INV-REPLAY-1: status_mismatch: stored status is paid (paid), the replayed balance implies partial (partial)
```

A failed billing disconnect that never reached the NAS replays as:

```
sess-1: disconnect_not_delivered: the disconnect action (billing_suspension) is failed after 3 attempt(s) while the session is still online
sess-1: disconnect_did_not_end_session: the last disconnect audit at 2026-10-29T09:00:00Z reported success but the session is still online
```

The second pair is the reproduction that matters operationally: the audit says
the NAS acknowledged the Disconnect-Request, the queue says the retry never
succeeded, and the online row says the subscriber still has access. One of the
three is wrong, and the replay names all three rather than the latest one.

## Decisions

1. **Replay from rows, not from packets.** The packet-level harness already
   exists (`internal/radiusd/accounting_order_test.go` dispatches ordered
   Start/Interim/Stop packets through the real handlers), but production does
   not store packets. The replay therefore reproduces contradictions in stored
   history, not traffic.
2. **A divergence is evidence, not a verdict.** Every finding states both stored
   and replayed values. The tool never "repairs" anything and never writes.
3. **Keep it an unenforced prototype.** It is not wired to the admin API
   because there is no production corpus to calibrate false positives, a full
   scan is unpaginated, and an operator-facing surface would need permissions
   and its own audit trail. The package doc says nothing imports it.
4. **Do not persist a replay snapshot.** The stored rows are the source of
   truth; a saved replay would be one more cache that can drift from them, the
   same reasoning as the FUT-009 explanation trace.
5. **Fold in the same order the scheduler wrote.** Events are ordered by
   `created_at` then `id`; payment rows are ordered by `paid_at` then `id`. Any
   other order would make a double reversal look legal.

## Verification

```
go test ./internal/replay/ -count=1
ok  	github.com/bjo163/mwx-isp/internal/replay	1.483s

golangci-lint run ./internal/replay/...
0 issues.

go test ./internal/tenancy/ -count=1      # the raw SQL ratchet is unaffected
ok  	github.com/bjo163/mwx-isp/internal/tenancy	1.485s
```

Unit coverage (`internal/replay/replay_test.go`):

- `TestReplayInvoiceRebuildsLedgerFromTrail` — generated, received, and reversed
  events fold to the stored partial invoice with no divergences.
- `TestReplayInvoiceDetectsTamperedLedger` — paid amount, balance, and status
  classes are all reported.
- `TestReplayInvoiceDetectsMissingReversalEvent` — a reversed payment without a
  reversal event is reported even though the stored invoice looks consistent.
- `TestReplayInvoiceDetectsReversalWithoutPayment` — a second reversal of the
  same payment is rejected by the fold.
- `TestReplayInvoiceDetectsDuplicateReceipt` — a duplicated `payment_received`
  event is reported and does not double the paid amount.
- `TestReplayInvoiceDetectsEventOrderAndUnknownPayment` — an `id` inversion in
  timestamp order and an event naming a missing payment.
- `TestReplayInvoiceDetectsNegativeLedger` — a non-positive payment amount
  produces a negative replayed balance.
- `TestReplaySessionRebuildsClosedLifetime`,
  `TestReplaySessionDetectsImpossibleLifetime`,
  `TestReplaySessionDetectsOrphanAndStaleOpenSession`,
  `TestReplaySessionDetectsUndeliveredDisconnect`,
  `TestReplaySessionDetectsOnlineWithoutAccounting`,
  `TestReplaySessionDetectsCounterRegressionAndOverlap`.
- `TestReplayBillingReadsStoredLedger` and `TestReplayRadiusReadsStoredSession`
  run the database loaders against SQLite, including a tampered invoice and an
  online row that outlived its stopped accounting row.
- `TestReplayEmptyDatabaseReportsNothing` pins that an empty tenant yields no
  findings and still reports its evidence gaps.

## Residual limits

- No production corpus has been run through the replay, so the false-positive
  rate of each kind is unmeasured; the SQLite fixtures are hand-built.
- The loaders scan every invoice with a trail and every accounting row. Bounding
  by tenant and time window, and paginating, is work-to-do before any endpoint
  exposes them.
- Void invoices are skipped: voiding has no event and no maintained balance.
- Payment amounts are read from payment rows, so an edited amount is invisible
  to the replay; only the direction of movement is event-sourced.
- Interim-Update history is absent, so `counter_regression` can only see
  differences between rows, not inside one session's traffic.
- NAS clock skew is not corrected; timestamps are compared only for internal
  contradictions.
- Action and audit rows are retention-pruned, so a missing row is not proof
  that no action was attempted.
- Billing suspension and RADIUS disconnect are not cross-referenced: linking a
  subscription to its RadiusUser and Acct-Session-Id would turn "suspended but
  still online" into a single finding, but that join is not built.
- Only SQLite has been exercised; PostgreSQL behaviour is inherited from the
  same GORM queries but unverified in this study.
