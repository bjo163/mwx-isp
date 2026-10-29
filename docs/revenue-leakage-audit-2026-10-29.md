# Revenue-leakage audit over the invoice ledger that already exists (ECO-009)

Research note and bounded prototype for `ECO-009` (TR-F028, Wave 2, tag `RISET`).
Delivered 2026-10-29 on top of `a4047ff`. No product code imports
`internal/revenue`; nothing in it writes.

## Question

Can an exception report be derived from the invoice, payment, subscription,
RADIUS-user and online-session rows that MWX-ISP already stores — with every
finding traceable to a billing rule the product already enforces — without
claiming a single rupiah of lost revenue that the rows do not support?

The item asks for the "duplicate / missed grace / paid partial" exceptions from
synthetic data and a sanitized sample. Two constraints shape the answer:

- The criterion is `Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang
  hilang tanpa bukti.` A finding that cannot name its rule is not a finding, and a
  money figure that cannot be read out of stored rows is not a figure.
- TR-N001 forbids growing this into a charging, ERP or finance-reporting system,
  so the audit reads the ledger it is given and proposes nothing.

## Evidence available

The billing service is the only place these state transitions exist, so the rules
are quoted from it rather than invented. `Rules()` returns the catalog, and every
rule names the function and file that enforces it:

| Rule | Function | Guard |
| --- | --- | --- |
| `billing.generate-monthly` | `generateMonthlyInvoices` | `status=active AND 1<=billing_day<=28 AND now.Day()>=billing_day AND start_date<=billing date AND no invoice for (subscription, period_start)` |
| `billing.one-invoice-per-period` | `generateMonthlyInvoices` | unique index on `(tenant_id, subscription_id, period_start)` |
| `billing.process-overdue` | `ProcessOverdueInvoices` | `balance>0 AND status IN (issued,partial,overdue) AND dayAfter(now, due_date)` sets overdue |
| `billing.suspend-overdue` | `SuspendOverdueSubscriptions` | `status=overdue AND dayAfter(now, due_date+grace_days)` suspends with `billing_overdue` |
| `billing.radius-user-disable` | `SuspendOverdueSubscriptions` | `radius_user_id>0` sets `radius_user.status=disabled` in the same transaction |
| `billing.suspend-no-disconnect` | `SuspendOverdueSubscriptions` | documented: it does not send Disconnect-Requests; callers must disconnect matching sessions |
| `billing.auto-reactivate` | `RecordPayment` | a clearing payment reactivates a `billing_overdue` suspension only when no other invoice has `balance>0 AND status IN (issued,partial,overdue) AND due_date < today` |
| `billing.ledger` | `RecordPayment` | `paid_amount` grows and `balance` shrinks by the same amount |
| `billing.status-from-balance` | `RecordPayment` | `status=paid` only when the balance reaches zero |
| `billing.void-is-closed` | `RecordPayment` | `status=void` rejects a payment |
| `radius.online-means-delivery` | `GetNetRadiusOnlineFromRequest` | a `radius_online` row is built from an Accounting-Start packet |

All sources are `internal/billing/service.go` except the last, which is
`internal/radiusd/radius.go`. `TestEveryRuleIsTraceableToRealSource` opens those
two files and asserts that `func <name>(` really exists in the file the rule
quotes, so the catalog cannot drift into a rule nobody wrote.

## Prototype

`internal/revenue` is three files:

| File | Scope |
| --- | --- |
| `leakage.go` | rule catalog, finding kinds, `Audit(Input, now) Report`, and the four checks (`auditLedger`, `auditInvoiceClock`, `auditSubscriptions`, `auditAccessState`). |
| `load.go` | `Load(db)` and `AuditDatabase(db, now)`: read the same slice from a supplied database. Six `Find` calls, no writes, no `ORDER BY` because the audit sorts its own inputs. |
| `render.go` | `RenderMarkdown(Report)`: totals, per-kind counts, per-tenant totals, findings with their rule and evidence, rules cited, and the gaps. |

`Input` is a plain projection of stored rows (`Subscription`, `InternetPackage`,
`Invoice`, `Payment`, `RadiusUser`, `RadiusOnline`), so the identical rule set
runs over a synthetic fixture and over a sanitized export. `Audit` partitions
rows by `TenantID` first and applies every rule inside one partition, which is
what makes a cross-tenant claim structurally impossible rather than merely
unlikely.

## Finding catalog

Eleven kinds. "Figure" states whether an IDR amount is derivable from stored
rows; seven of the eleven deliberately say no, and the report counts them
separately (`Totals.Unquantified`).

| Kind | Rule cited | Figure |
| --- | --- | --- |
| `unbilled_period` | `billing.generate-monthly` | yes: package price x missing periods |
| `invalid_billing_day` | `billing.generate-monthly` | no: a blocked guard is not a proven receivable |
| `duplicate_period` | `billing.one-invoice-per-period` | yes: the duplicate row's total |
| `stale_overdue_status` | `billing.process-overdue` | no: the balance is counted once, by the grace check |
| `grace_exceeded_still_active` | `billing.suspend-overdue`, or `billing.process-overdue` when the status never advanced | yes: the invoice balance |
| `status_balance_conflict` | `billing.status-from-balance` | yes: the magnitude of the balance |
| `ledger_mismatch` | `billing.ledger` | yes: the disagreement with the payment rows |
| `payment_on_void_invoice` | `billing.void-is-closed` | yes: money on an invoice that collects nothing |
| `enabled_radius_user_after_suspend` | `billing.radius-user-disable` | no |
| `online_after_billing_suspend` | `billing.suspend-no-disconnect` | no |
| `stuck_reactivation` | `billing.auto-reactivate` | no |

`grace_exceeded_still_active` deliberately picks the rule that actually failed:
when an invoice never reached `overdue`, the suspend guard never saw it, so the
process-overdue rule is named instead. One balance is then reported once, whatever
the status path, which `TestGraceElapsedWhileTheSubscriptionStaysActive` pins.

## Example

`leakageShowcase()` in the test file builds one tenant that breaks eight of the
eleven rules at once. The audit's own renderer produces:

```
Findings: 8 (5 not quantified) · Exposure: IDR 400000 · Tenants: 1
```

with each finding rendered as one traceable line:

```
tenant=1 grace_exceeded_still_active INV-202610-000001 [billing.process-overdue] exposure=IDR 100000
  evidence=subscription=SUB-0001 status=active grace_days=3 due_date=2026-10-19 now=2026-10-29;
           invoice status=issued balance=100000

tenant=1 unbilled_period SUB-0003 [billing.generate-monthly] exposure=IDR 200000
  evidence=no invoice for period_start 2026-09-05; no invoice for period_start 2026-10-05;
           0 invoice row(s) present for this subscription; package BRONZE price=IDR 100000 cycle=monthly

tenant=1 online_after_billing_suspend SUB-0004 [billing.suspend-no-disconnect] exposure=not quantified
  evidence=1 radius_online row(s) for sub-0004; session SES-0004 started 2026-10-29T07:00:00Z
           reports 7200 second(s) of service; basis: radius.online-means-delivery
           @ GetNetRadiusOnlineFromRequest (internal/radiusd/radius.go)
```

Three of those lines carry a figure and five refuse to, which is the shape the
criterion asks for: the report can total IDR 400000 without pretending that the
suspended-but-still-connected subscriber cost anything measurable.

## Decisions

1. **A rule or nothing.** Every `Finding` carries a `Rule` with function and
   source file, and `TestEveryFindingCitesARuleEvidenceAndLimit` fails any finding
   with an empty rule, empty evidence, or an empty note. The note is mandatory
   because each kind has a limit that is easy to overread.
2. **No figure without a row behind it.** `Quantified` is a separate flag from
   `Exposure`, so an unquantified finding cannot smuggle a number into the total,
   and `Totals.Exposure` sums quantified findings only. The total is labelled
   "money at stake", not "money lost".
3. **Tenants partitioned, not filtered.** The audit takes the whole slice and
   splits by `TenantID` itself, so a caller that passes a full sample export
   cannot mix tenants by accident; `TestFindingsNeverCrossATenant` builds two
   tenants with identical IDs, invoice numbers and usernames and asserts that only
   the tenant with the missing period is reported.
4. **Row order cannot change the verdict.** Each slice is sorted before indexing,
   so `Load` needs no `ORDER BY` and `TestReportIsIndependentOfRowOrder` compares
   a forward and a reversed input with `reflect.DeepEqual`.
5. **The scheduler's guards, not a new billing opinion.** `unbilled_period`
   re-applies `generateMonthlyInvoices`'s own conditions, including "the current
   month only counts once today has reached the billing day" and "a billing day
   outside 1..28 is its own kind", so the audit cannot invent periods the product
   was never going to bill.
6. **Read-only, and said so.** No GORM writes exist in the package and
   `TestLoadWritesNothing` counts rows before and after an audit to prove it.

## Verification

```
go test ./internal/revenue/ -count=1
ok  	github.com/bjo163/mwx-isp/internal/revenue	1.891s

golangci-lint run ./internal/revenue/...
0 issues.

go build ./...
(no output; grep for internal/revenue outside the package returns nothing,
 so no product code imports the prototype)
```

28 tests pass. The ones that carry the criterion:

- `TestCleanLedgerProducesNoFindings` — ten months of paid invoices, one live
  session, no finding.
- `TestEveryFindingCitesARuleEvidenceAndLimit` — no finding may ship without a
  rule, stored values, a note, and a figure only when `Quantified` is set.
- `TestEveryRuleIsTraceableToRealSource` — the function each rule names exists in
  the file the rule quotes.
- `TestGraceStillRunningIsNotLeakage`, `TestCurrentMonthIsOnlyExpectedOnceTheBillingDayPassed`,
  `TestManualSuspensionIsOutsideTheBillingRules`, `TestSuspensionWithAnOutstandingInvoiceIsNotStuck`,
  `TestReversedPaymentIsNotCountedAsMoneyReceived` — the negative cases.
- `TestFindingsNeverCrossATenant` — two tenants with identical subscription IDs,
  invoice numbers and usernames produce a finding only for the tenant whose own
  rows are short.
- `TestReportIsIndependentOfRowOrder`, `TestFindingsAreOrderedByTenantKindSubject`
  — the same rows always produce the same bytes.
- `TestLoadReadsEveryTableTheAuditNeeds`, `TestLoadWritesNothing` — a database
  handle produces the same per-kind counts as the fixture and is left untouched.

Files: `internal/revenue/leakage.go`, `internal/revenue/load.go`,
`internal/revenue/render.go` (905 lines) with `leakage_test.go` and
`load_test.go` (752 lines).

## Residual limits

- **Absence has no cause.** A missing invoice row looks the same whether the
  subscription was skipped or the scheduled job never ran; the report says so in
  its gaps section instead of guessing.
- **Today's prices.** `unbilled_period` values a period at the package price
  stored now, not the price in force that month, and a manually created invoice
  with a different `period_start` satisfies no guard here while still collecting
  money.
- **Sessions are current, not historical.** A `radius_online` row that never
  received its Accounting-Stop is indistinguishable from a live session, so
  `online_after_billing_suspend` proves delivery only as of the read.
- **`void` is not produced today.** No product path sets an invoice to `void`, so
  `payment_on_void_invoice` only fires on legacy or repaired rows; it is kept
  because the ledger's own guard rejects payments on void invoices.
- **Money state, not intent.** `internal/replay` owns the event trail; this audit
  reads the resulting rows and cannot tell who changed them or when.
- **No pilot data yet.** Every number above comes from synthetic fixtures. The
  item's second input — a sanitized sample database — is reachable through `Load`
  but has not been run, because no sample export exists locally; that is the only
  part of the acceptance criterion still unproven.
