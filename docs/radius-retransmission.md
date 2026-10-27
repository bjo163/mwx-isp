# RADIUS Retransmission and Replay (ENG-010)

A NAS that does not receive a response retransmits the same packet: same
RADIUS `Identifier`, same Request Authenticator, same attributes. This document
maps what MWX-ISP does with that second copy on each transport, which state can
change twice, and which tenant boundary applies.

## Duplicate-detection inventory

| Path | Mechanism | Key | Effect of a retransmission |
| --- | --- | --- | --- |
| UDP Access-Request | No completed-response cache; `authRate` duplicate window (`internal/radiusd/auth_rate_limiter.go:79`) | `tenantID + username` | Inside the window the copy is rejected (`there is a authentication still in process`). After the window it re-runs the pipeline; see "Auth side effects" below. |
| UDP Accounting-Request | Handler idempotency, not packet cache (`internal/radiusd/radius_acct.go:103`) | `(tenant_id, acct_session_id)` | Start: one online row, one accounting row. Interim: same row is refreshed. Stop: same rows are finalized. |
| RadSec (TLS, both codes) | In-flight map per connection (`internal/radiusd/radsec_server.go:282-374`) | `(remote address, Identifier)` | A copy arriving while the first is still executing is dropped. A copy arriving after completion is processed again and relies on the handler idempotency above. |
| EAP state machine | Per-state serialization (`internal/radiusd/plugins/eap/coordinator.go:93-105`) | `tenantID + RADIUS State` | The duplicate waits for the in-flight handshake step, then runs against the resulting state; it cannot advance the TLS engine concurrently. |
| NAS lookup cache | `nasCache` (`internal/radiusd/radius.go:147-167`) | source IP, or `tenant:<id>:<ip>` for control paths | No duplicate effect; an ambiguous IP shared by two tenants fails closed with `ErrAmbiguousNASIP`. |
| User lookup cache | `userCache` (`internal/radiusd/radius.go:184-202`) | `tenantID + macauth + identity` | No duplicate effect; tenant is part of the key. |

There is deliberately no RFC 5080 §2.2.2 response cache on the UDP path. A cache
would have to store signed responses per NAS for the duplicate window, and the
only state-changing side effects of a repeated Access-Request are already
idempotent or tenant-scoped (see below). Adding it would not change row counts,
only response latency for the retransmission.

## Auth side effects of a repeated Access-Request

| Side effect | Duplicate behaviour | Source |
| --- | --- | --- |
| `radius_user.last_online` | Overwritten with a later timestamp; no new row. | `auth_stages.go:376` |
| MAC/VLAN bind | Written only when the value differs; repeating the same packet writes nothing. | `radius.go:490-502` |
| Hotspot voucher activation | Conditional update `WHERE first_login_at IS NULL`; only one caller wins the expiry window. | `radius.go:269-322` |
| Rate-limit timestamp | One entry per `tenantID + username`; the copy is refused inside the window, not queued. | `auth_rate_limiter.go:79-99` |
| Accept/reject metrics | Counted per processed request, so a retransmission that is answered counts again. Metrics are operational counters, not billing state. | `auth_stages.go:385` |
| EAP handshake | Serialized per `tenantID + State`; an EAP copy cannot concurrently advance the shared TLS engine. | `coordinator.go:93-105` |

## Accounting side effects of a repeated Accounting-Request

`StartHandler` inserts the online row with `ON CONFLICT (tenant_id,
acct_session_id) DO NOTHING`; `created=false` skips the accounting insert
(`start_handler.go:86-92`, `repository/gorm/session_repository.go:35-53`). If
the accounting insert fails after the online row was created, the handler
compensates by deleting the online row so the retransmission recreates both rows
instead of leaving a session without accounting.

`UpdateHandler` patches the existing online row (`session_repository.go:55`) and
never appends a billing record. One observable duplicate effect remains: each
interim update writes one `radius_traffic_sample`; a replayed interim with
unchanged counters inserts a sample with a zero rate. Samples are monitoring
data, not invoice input, and a zero sample does not change billing totals.

`StopHandler` finalizes the same accounting row by `acct_session_id` and deletes
the online row (`stop_handler.go:51-93`). Replaying the stop updates the same
row with the same counters and finds nothing left to delete; tenant B's session
with the same `Acct-Session-Id` is a different row because the unique key and
every query carry `tenant_id`.

## Cross-tenant replay

- The tenant is derived from the NAS record selected by packet source IP before
  any user lookup (`auth_stages.go:78-93`, `radius_acct.go:57-74`). A username
  that exists in two tenants never resolves across the boundary.
- EAP `State` values are namespaced as `tenant:<id>:<state>`
  (`coordinator.go:14-36`), so a captured State replayed through another
  tenant's NAS is not found.
- A source IP configured by two tenants is ambiguous and refused
  (`ErrAmbiguousNASIP`), never resolved by NAS-Identifier.
- RadSec's in-flight key includes the connection's remote address, so it cannot
  suppress a request on another tenant's connection.

## Evidence

`internal/radiusd/retransmission_test.go`:

- `TestRetransmittedAccountingPacketsHaveNoDoubleEffect` builds byte-identical
  Accounting-Start/Interim/Stop packets, replays each one, and asserts one
  online row, one accounting row, final counters equal to the stop values, and
  that a second tenant using the same `Acct-Session-Id` and username keeps its
  own session.
- `TestRetransmittedAccessRequestRateLimitIsTenantScoped` asserts the duplicate
  window blocks the retransmission inside one tenant but not the same username
  in another tenant.

Existing coverage reused by this audit: `handlers_test.go`
(duplicate start/retry), `session_repository_test.go` (conflict insert),
`coordinator_concurrency_test.go` and `coordinator_test.go` (EAP retransmit and
tenant state), `radius_test.go` (ambiguous NAS, voucher expiry scoping).

## Verification

```bash
go test ./internal/radiusd/ -run 'TestRetransmitted' -count=1
go test ./internal/radiusd/... -count=1
```

## Limits

- The RadSec completed-response path and the UDP response cache are not
  implemented; a retransmission that arrives after the first request finished is
  handled as a new request and answered by the idempotent handlers above.
- Duplicate interim updates add zero-rate traffic samples; if those samples ever
  feed billing or SLA calculations, the interim handler must deduplicate by
  `(session, counters, timestamp)` first.
- The duplicate window is in-process. Two application instances behind the same
  NAS IP would each keep their own window; the database idempotency keys remain
  the cross-instance guarantee for accounting state.
