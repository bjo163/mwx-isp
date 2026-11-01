# LDAP connection handling — load evidence 2026-10-31

Feature: TR-F025 (LDAP/AD credential backend) · roadmap item M14.5.

Scope: measure what an LDAP-backed authentication actually costs, decide
whether M14.5 should buy connection pooling or request-context propagation,
implement the decision, and record the numbers so a future round can revisit it
without re-deriving them. Anchor files:
[`internal/ldapauth/ldapauth.go`](../internal/ldapauth/ldapauth.go) and the
measurement tool [`tools/ldapload/main.go`](../internal/ldapload/main.go).

## Why a measurement was required

M14.3 delivered the verifier with one connection per authentication attempt and
explicitly deferred "connection reuse and richer observability". Shipping reuse
or rejecting it on the strength of a guess would have been the wrong call for a
path that sits in front of every login of every subscriber, so the decision was
replaced with numbers from a real directory.

## Harness

Tool: `tools/ldapload` (`go run ./tools/ldapload`). It measures four things
against a live directory and prints them as a table plus a throughput ladder
plus a cancellation probe.

Directory: the CI test container from
[`docker-compose.test.yml`](../docker-compose.test.yml) —
`bitnamilegacy/openldap:2.6`, `ldap://127.0.0.1:1389`, root
`dc=example,dc=org`, users `alice`/`bob`.

Environment: go1.27.0, AMD Ryzen 5 7500F, Windows/amd64, 12 logical CPUs,
loopback. 200 measured requests after 20 warm-up per stage, operation timeout
3 s, concurrency ladder 1 / 10 / 50.

Commands:

```bash
docker compose -f docker-compose.test.yml up -d openldap

/tmp/ldapload -url ldap://127.0.0.1:1389 -bind-mode template \
  -bind-dn-template 'cn=%s,ou=users,dc=example,dc=org' \
  -user alice -password alice-password -requests 200 -warmup 20

/tmp/ldapload -url ldap://127.0.0.1:1389 -bind-mode search \
  -base-dn dc=example,dc=org -user-filter '(cn=%s)' \
  -search-bind-dn cn=admin,dc=example,dc=org -search-bind-password adminpassword \
  -user alice -password alice-password -requests 200 -warmup 20
```

The stages are:

| Stage | What it times |
| --- | --- |
| `dial (no operation)` | TCP setup only — no LDAP PDU is sent |
| `bind on warm connection` | one user Bind on a single long-lived connection; the floor a pool could reach |
| `verify (fresh connection)` | the production `ldapauth.Verifier.Verify` path, i.e. dial + operation + close |
| throughput ladder | `Verify` per second at 1 / 10 / 50 concurrent authentications |
| cancellation probe | `Verify` against a listener that accepts and then stays silent, with a caller deadline far shorter than the operation timeout |

## Measured cost

### Template bind mode (DN from a fixed template, one round trip)

| Stage | p50 | p95 | p99 | max | errors |
| --- | --- | --- | --- | --- | --- |
| dial (no operation) | 525.1 µs | 1.2195 ms | 2.5533 ms | 4.013 ms | 0 |
| bind on warm connection | 679.1 µs | 1.3018 ms | 1.6652 ms | 1.868 ms | 0 |
| verify (fresh connection) | 2.8869 ms | 5.8793 ms | 7.9733 ms | 9.0802 ms | 0 |

Mean verify 3.33 ms, of which 2.57 ms (77 %) is connection setup.

| Concurrency | auth/s | p95 | errors |
| --- | --- | --- | --- |
| 1 | 408.8 | 3.80 ms | 0 |
| 10 | 645.3 | 32.37 ms | 0 |
| 50 | 507.3 | 149.16 ms | 0 |

### Search bind mode (service Bind + Search + user Bind, three round trips)

| Stage | p50 | p95 | p99 | max | errors |
| --- | --- | --- | --- | --- | --- |
| dial (no operation) | 523.9 µs | 1.0474 ms | 1.5617 ms | 1.6167 ms | 0 |
| bind on warm connection | 600.1 µs | 767.5 µs | 1.262 ms | 1.3575 ms | 0 |
| verify (fresh connection) | 3.3601 ms | 6.3995 ms | 13.2911 ms | 24.6238 ms | 0 |

Mean verify 3.89 ms, of which 3.27 ms (84 %) is connection setup.

| Concurrency | auth/s | p95 | errors |
| --- | --- | --- | --- |
| 1 | 236.3 | 6.35 ms | 0 |
| 10 | 675.4 | 23.53 ms | 0 |
| 50 | 715.0 | 106.90 ms | 0 |

### Cancellation probe — the finding that decided the milestone

Before this change, with a 200 ms caller deadline and a 3 s operation timeout
against a directory that accepts the TCP connection and then never answers:

```
context deadline 200.00ms, operation timeout 3.00s, Verify returned after 3.00s
error: ldapauth: directory unavailable: bind: LDAP Result Code 200
       "Network Error": ldap: connection timed out
```

`Verify` ignored the caller entirely and blocked for the full configured
operation timeout, in both bind modes. That is the defect M14.5 exists to
remove: the RADIUS worker pool is 1024 goroutines, so a silent — not dead, just
slow — directory pins every worker that touches it for the whole timeout. A
directory that answers in 4 s with a 5 s configured timeout looks healthy in the
metrics while having consumed every authentication slot the server has.

## Decision

Implement request-context propagation. Do **not** implement connection pooling.

### Why not pooling

Setup is the dominant cost (77 % / 84 % of a verify locally), so on paper a pool
is the obvious win. It is still the wrong trade here, for one structural reason:
a Bind is not a transport operation, it is an identity operation. After
`Bind(userDN, password)` the connection *is* that subscriber — its anonymous vs.
authenticated state, and in search mode its service-account state, all change
what a subsequent request on the same socket is allowed to see. Returning a
post-Bind connection to a shared pool, or reusing one under another user, turns
a credential bug into a directory-disclosure bug. And the two ways to avoid that
both cost most of the saving:

- bind-and-discard (pool only pre-anonymous connections): each authentication
  still pays its own Bind, and after an anonymous-pool Bind the connection
  cannot be returned unauthenticated without a fresh anonymous Bind, so the pool
  degenerates into "keep the socket, redo the handshake"; the measured residual
  saving is the `dial` row alone — 525 µs of a 3.33 ms verify locally.
- per-user connections: LDAP credentials are per-subscriber, so a pool sized for
  a real ISP holds one TLS session per concurrent user, which is precisely the
  memory and file-descriptor pressure the pool was meant to relieve.

Against that, the absolute numbers are small: 525 µs of setup per authentication
on loopback, and both modes already serve 236–409 auth/s on a single worker at 1
concurrency, well above what an ISP of this size authenticates. The throughput
ladder also shows the ceiling is the directory's own concurrency, not the
per-authentication dial: template mode plateaus and degrades past 10 concurrent
authentications, which a pool would not fix.

So pooling stays deferred. Re-open it only if a real deployment shows the dial
share dominating — the concrete trigger is a directory reachable only over a
high-latency or TLS-heavy link where `dial` is no longer a fraction of a
millisecond, i.e. where `verify − bind-on-warm` exceeds ~10 ms. Re-measure with
`tools/ldapload` before building anything.

### What was implemented

Request-context propagation in `internal/ldapauth`, so the caller's deadline
bounds the directory instead of the other way round. Three checkpoints, in
increasing order of reach:

1. **Dial races the context.** `dialFunc` now takes the caller's
   `context.Context`; `dialReal` runs `ldap.DialURL` in a goroutine and selects
   against `ctx.Done()`. If the caller gives up first, the late socket is closed
   by a background drain rather than leaked, and `ctx.Err()` is returned.
2. **Socket timeout capped to the remaining deadline.** After dialing, the
   per-operation timeout is set to `min(cfg.Timeout, time.Until(deadline))`, so
   the library's own read deadline fires inside the caller's budget. If the
   deadline has already lapsed, `Verify` fails before issuing an operation.
3. **Context checks at every round trip.** Before StartTLS, before the service
   Bind, between the service Bind and the Search, and between the Search and the
   user Bind — so a deadline that lapses mid-flow aborts rather than continuing
   against a directory that has stopped answering the request.

Context errors are deliberately *not* re-wrapped as `ErrUnavailable`:
`context.Canceled` and `context.DeadlineExceeded` are propagated so a caller can
tell "this request was abandoned" from "the directory is down". The RADIUS
adapter in `internal/radiusd/ldap_backend.go` still maps both to
`radus_reject_ldap_error`, which keeps an abandoned authentication out of the
wrong-password counters.

**One catch the code required.** `internal/ldapauth` can only honor a deadline
the caller actually sets, and the auth pipeline builds its stage context from
`context.Background()` — no deadline at all. Propagation alone would therefore
have changed nothing on the live RADIUS path. So `VerifyCleartext` now arms a
`ldap.Timeout` deadline itself when the caller brings none, and a caller with a
tighter budget (an EAP handshake, a future per-request auth timeout) overrides
it. The observable effect is that `ldap.Timeout` became a bound on the *whole*
directory exchange instead of a bound each of its round trips may consume in
full: search mode against a silent peer previously cost dial + service Bind +
Search + user Bind, each up to the timeout, and now costs the timeout once.
`TestLDAPBackend_VerifyCleartextBoundsSilentDirectory` pins that with a listener
that accepts and goes silent.

### Re-measured after the change

Same harness, same container, same flags:

| Bind mode | deadline | before | after |
| --- | --- | --- | --- |
| template | 200 ms caller, 3 s operation | Verify returned after **3.00 s** | **200.49 ms** |
| search | 200 ms caller, 3 s operation | Verify returned after **3.00 s** | **200.09 ms** |

The caller now stops inside its own deadline in both modes. A silent directory
consumes a worker for the length of the authentication's budget instead of the
length of `ldap.Timeout` per round trip.

On the live path the budget is armed by the backend rather than by the pipeline —
see "One catch the code required" above — so the production worst case is one
`ldap.Timeout` for the whole exchange, which for search mode is what used to take
up to four of them.

## Limits of this evidence

- **TLS cost is not measured.** The test image runs `slapd -h "ldapi:///
  ldap://:1389/"` with no listener and no certificate, so `dial` here is a plain
  loopback TCP handshake. Against a real `ldaps://` domain controller, setup
  includes TCP + TLS handshake + (for search mode) two further round trips, and
  every number in the "measured cost" tables is a floor, not an estimate. The
  pooling conclusion is stated as conditional on exactly that gap.
- **One machine, loopback, containerized slapd with an in-memory-ish backend.**
  No WAN latency, no replication, no lock contention from a real directory with
  a real index. The throughput plateau is this slapd's plateau, not a
  production DC's.
- **Cancellation is measured with the request's worst case** — a peer that
  accepts and goes silent — which is the case that actually pins workers. A
  peer that refuses the connection was already handled correctly by M14.3.
- The percentages are setup-share of *latency*, not of CPU. They justify not
  pooling at this scale; they do not prove pooling is never worth it.

## Verification

- Unit tests: `go test ./internal/ldapauth/` — the new
  `TestVerify_ContextHonoredAfterDial` covers all four behaviors (timeout capped
  to the remaining deadline, already-expired deadline does not dial, deadline
  lapsing between the service Bind and the Search aborts, a dial that outlives
  the caller is abandoned).
- Integration: `TestLDAPIntegrationAcceptance` in
  [`test/integration/ldap_test.go`](../test/integration/ldap_test.go) still
  passes unchanged against the same OpenLDAP container, which is the point —
  context propagation had to add a guarantee without changing any authentication
  outcome.
- Tool tests: `go test ./tools/ldapload/` for the percentile, summary and flag
  parsing that the tables above depend on.
