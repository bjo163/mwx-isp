# Session action queue (suspend convergence)

This document describes how a suspended subscriber's live session is converged
with the NAS, and how failed RFC 5176 Disconnect-Requests are retried instead of
being logged and forgotten. It is the reference for issue #52 (AUDIT-08,
historical H05/H04) and the H05 close-out, and must be updated when suspension,
disconnect, or retry behavior changes.

## Why a queue

The billing scheduler (`internal/app/jobs.go`, `runBillingForTenant`) commits the
subscription suspension and disables the RADIUS user in one transaction. The
RADIUS user being disabled only stops *new* authentications; an already-running
session stays online until the NAS is told to terminate it. A Disconnect-Request
can fail because the NAS is offline, a UDP packet is lost, or the process
restarts between the database commit and the wire exchange.

Logging that failure leaves the subscriber suspended in the database but online
at the NAS — an access-control and billing mismatch. The durable queue in
`internal/sessionqueue` closes that gap: the required action is persisted before
it is attempted and retried until it succeeds, the NAS answers, or the bounded
budget is exhausted.

## State model

| Status | Meaning |
| --- | --- |
| `pending` | Recorded and waiting for `next_retry_at`. Also used after a failed automatic attempt that still has budget. |
| `running` | Claimed by one worker; a `claim_token` fences the final update. |
| `succeeded` | The NAS returned a Disconnect-ACK. |
| `failed` | The automatic budget (`max_attempts`, default 8) is exhausted. Manual retry still allowed. |
| `canceled` | The action is no longer needed: the session disappeared, or the subscriber was re-enabled before the action ran. |

Rows live in `radius_session_action`. The idempotency key is
`disconnect:<reason>:<session row id>`, unique per tenant
(`tenant_id`, `idempotency_key`). Repeated billing runs, process restarts, and
concurrent schedulers therefore cannot create a second action for the same
suspended session. A new session row (new `acct_session_id`) produces a new key,
which is how a re-suspension after reactivation is queued.

## Enqueue path

There are two producers, distinguished by the action reason:

| Reason | Producer |
| --- | --- |
| `billing_suspension` | `internal/app/jobs.go` (`runBillingForTenant`) after the automatic billing pass suspends overdue subscriptions. |
| `manual_suspension` | `subscriptionAction` in `internal/adminapi/isp.go` when an operator suspends or terminates a subscription. |

Both paths commit the subscription/user state first, then queue a `pending`
action per online session and call `ProcessDue` once for the tenant:

1. The subscription is suspended (or terminated) and the RADIUS user is disabled
   in one transaction. Manual suspension stores `suspension_reason = manual`;
   billing enforcement stores `billing_overdue`.
2. For every online session of that user the producer calls
   `sessionqueue.EnqueueDisconnect`, which inserts a `pending` row with
   `next_retry_at = now`. No secrets are stored — only the session identity
   (username, `acct_session_id`, NAS address) and the action reason.
3. The same request/run immediately calls `ProcessDue` for the tenant, so a
   reachable NAS is converged without a second scheduler tick.

Enqueueing happens even when the process has no sender configured (for example
in a deployment that has not wired the CoA client). The rows stay visible and
actionable instead of silently dropping the requirement. A NAS failure no longer
turns an already-committed manual suspension into an API error: the operator
gets a successful response and the pending action appears in the session queue.

## Scheduler, backoff and recovery

`SchedSessionActionRetryTask` runs every 5 minutes (`@every 5m`) and also once
asynchronously at startup, which is the recovery pass for actions left pending
by a previous process. Each pass:

1. Recovers `running` rows older than 10 minutes (crashed worker) back to
   `pending`.
2. Claims due `pending` rows for one tenant at a time, oldest first.
3. Re-checks the desired state before sending: if the session is gone, or the
   RADIUS user is no longer disabled, the action is `canceled` without touching
   the NAS.
4. Sends one Disconnect-Request with a 20-second bound and finalizes the row.

Failed attempts are rescheduled with exponential backoff: 1 minute, doubling to
a 6-hour cap (`sessionqueue.Backoff`). After `max_attempts` (default 8) the row
becomes `failed` and stops retrying automatically. The attempt counter is
incremented at claim time, so crash recovery cannot loop forever.

## Manual retry and audit

`GET /api/v1/session-actions` lists the tenant's queue, filterable by
`status` (comma separated), `username`, and `reason`. The online sessions page
shows `pending`/`failed` actions from both producers in a panel above the
session list, including the action reason.

`POST /api/v1/session-actions/:id/retry` (admin only) force-claims a `pending`
or `failed` action, resets the automatic attempt budget, increments
`manual_retries`, and runs the exchange synchronously. A NAK or timeout is a
completed exchange: HTTP 200 with `success=false` and the structured cause. An
unknown action returns 404 (including actions owned by another tenant), a
running action returns 409 `ACTION_RUNNING`, and a succeeded/canceled action
returns 409 `NOT_RETRYABLE`.

Every completed attempt writes a `radius_session_action_audit` row with the
operator identity (`system:scheduler` for automatic attempts), response code,
Error-Cause, wire attempts, RTT, timeout flag, and error text. Canceled actions
never reached the NAS and therefore produce no attempt audit; the cancel reason
is stored on the action row.

## Tenancy

Every query in the queue is scoped to one tenant. The unique key includes
`tenant_id`, so two tenants can hold the same logical action. The list and retry
endpoints resolve the tenant from the authenticated request and hide other
tenants' rows. Platform-wide backup does include the queue tables, but the queue
is operational state: after a restore, the next billing run re-enqueues any
subscriber that is still suspended with a live session, so convergence does not
depend on the restored queue contents.

## Verification

- Unit: `go test ./internal/sessionqueue/`
- Unit: `go test ./internal/app/ -run 'Billing|SessionAction'`
- Unit: `go test ./internal/adminapi/ -run 'SessionAction|SuspendQueuesDurableDisconnect'`
  (manual suspend with fake-NAS ACK and timeout, duplicate-free queueing)
- Unit: `go test ./internal/billing/ -run 'RecordPayment|SuspendOverdue'`
  (manual suspension preserved, multiple overdue invoices block reactivation)
- PostgreSQL + fake NAS: `go test -tags=integration ./test/integration/ -run TestPostgresSessionActionQueue`
  (ACK + audit, NAK, timeout then NAS recovery, concurrent retry claim,
  cancellation after reactivation, cross-tenant isolation)
- Frontend: `cd web && npm run type-check`

## Operational notes

- A permanently unreachable NAS ends in `failed`; retry manually after fixing
  the NAS. A truly new session gets a new action automatically.
- `succeeded` rows are kept for history; they are not garbage-collected yet.
  The same retention policy as other session audit tables applies.
- The queue never stores NAS secrets or subscriber passwords. The sender
  resolves the NAS secret at attempt time.
