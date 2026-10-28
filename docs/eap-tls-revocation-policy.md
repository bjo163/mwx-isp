# EAP-TLS certificate revocation policy — decision record

Feature: TR-F004 (EAP authentication) · backlog item ENG-003.

Status: **decision recorded, not deployed.** The verifier exists only as a
proof of concept in the test tree. Nothing in this document claims that
certificate revocation is enforced in production.

## Context

EAP-TLS authenticates the peer with a client certificate. The engine is built
with `ClientAuth = RequireAndVerifyClientCert` and a `ClientCAs` pool
(`internal/radiusd/plugins/eap/tlsengine/engine.go:145`), so crypto/tls checks
the chain, the key usages, the validity window and the EKU — and nothing else.
**Go's TLS stack does not fetch or evaluate CRLs or OCSP responses.** The
RadSec listener has the same property (`VerifyClientCertIfGiven` at
`internal/radiusd/radsec_tls.go:90` and `:210`).

Consequence: a client certificate that the CA has revoked stays usable until
its `NotAfter` date, on both EAP-TLS and RadSec. The only revocation control in
place today is the operator removing the CA from the managed certificate store,
which invalidates every certificate issued by it.

## Threat model

| Case | Covered today | Covered after the proposed policy |
| --- | --- | --- |
| Lost or stolen device with a valid client certificate | no — valid until expiry | yes, after the CA publishes a CRL and the server reloads it |
| Offboarding: employee keeps a client certificate | no | yes, same condition |
| Leaked certificate used after the subscriber's account is disabled | partially — RADIUS user status still rejects | unchanged (status check already applies) |
| Rogue certificate issued by a compromised CA | no | no — a compromised CA can also issue a clean CRL |
| Attacker presents a self-signed or foreign-CA certificate | yes — chain verification rejects | unchanged |
| Expanded trust: intermediate CA path | chain verification only | yes if the intermediate's issuer publishes a CRL |

Revocation is therefore a **containment** control for a leaked credential, not a
defence against a compromised CA. It only works while the CA keeps publishing
CRLs and the server keeps ingesting them.

## Scope

| Identity | Verified by | Revocation coverage after the policy |
| --- | --- | --- |
| EAP-TLS client certificate | `RequireAndVerifyClientCert` | yes — every certificate in the presented chain with a CRL for its issuer |
| PEAP / EAP-TTLS outer tunnel | server-only (`ServerOnly`), client presents no certificate | not applicable |
| RadSec peer certificate | `VerifyClientCertIfGiven` | out of scope of this decision — same hook can be applied, but the RadSec config is separate and certificate-less peers are allowed by policy |
| Server certificate presented to the peer | the peer's own trust store | out of server scope |
| Intermediate CA certificates | chain verification | yes — the same per-certificate loop applies |

## Data source: CRL first, OCSP deferred

| Option | Freshness | Operational cost | Failure surface |
| --- | --- | --- | --- |
| CRL file in the managed certificate store | as good as the publication interval + reload | CA must publish a CRL; operator must upload/refresh it | stale file, wrong issuer, signature failure |
| OCSP responder | near real time | server must reach the responder during authentication; `golang.org/x/crypto/ocsp` dependency; responder availability | timeout, responder down, unsigned/mismatched response |

Decision: **CRL**. EAP authentication happens on the RADIUS hot path, and an
OCSP round trip inside a handshake that already spans several RADIUS
round-trips adds a network dependency the operator cannot bound. OCSP is
revisited only if a CA cannot publish CRLs.

## Timeout and staleness

- A CRL is valid from `ThisUpdate` to `NextUpdate` (RFC 5280 §5.1.2.4/§5.1.2.5).
- A CRL that is not yet valid, has no `NextUpdate`, or is past `NextUpdate`
  **fails closed**: the server cannot answer "is this certificate revoked?", so
  it must not guess.
- No grace period is configured. The alternative (a grace window) trades
  availability for security, and the operator already controls the publication
  interval and the reload.
- Reload is pull-based at construction time (the provider builds a fresh TLS
  config per handshake), so a new CRL takes effect on the next handshake,
  matching how certificate material already rotates (`docs/radsec-certificate-rotation.md`).

## Fail mode decision

| Situation | Behaviour | Rationale |
| --- | --- | --- |
| CRL for the issuer is present and the certificate is listed | **reject** the handshake | the credential is revoked |
| CRL present, stale or not yet valid | **reject** (fail closed) | unknown revocation state must not be treated as "not revoked" |
| CRL present, signature does not verify against the issuer | **reject** | an unauthenticated CRL is attacker-controlled input |
| No CRL for the issuer | **accept** | documented coverage limit: the operator must publish a CRL for every CA trusted for client authentication |
| No revocation data configured at all | **accept** (unchanged today) | rolling this out must not silently lock out a working deployment |

The "no CRL" row is the load-bearing limit: the hook alone is not a revocation
guarantee. The operator checklist is: for each CA in the EAP client-CA bundle,
publish a CRL, upload it, and alert when `NextUpdate` approaches.

## Proof of concept

`internal/radiusd/plugins/eap/tlsengine/engine.go` gained one optional field,
`Config.VerifyConnection`, passed to `tls.Config.VerifyConnection` and run after
the built-in chain verification (`nil` keeps today's behaviour). The proposed
policy lives in `revocation_poc_test.go`; it walks the presented chain, matches
CRLs by issuer, verifies the CRL signature against the CA, enforces the
`ThisUpdate`/`NextUpdate` window and compares serial numbers.

Two scenarios are exercised with a real EAP-TLS handshake through the engine:

| Test | Scenario | Result |
| --- | --- | --- |
| `TestRevocationPOC_RevokedCertificate/current_engine_accepts_a_revoked_certificate` | revoked client certificate, no revocation data | handshake completes — this is the gap |
| `TestRevocationPOC_RevokedCertificate/proposed_verifier_rejects_the_revoked_certificate` | same certificate, CRL lists its serial | server rejects with `client certificate is revoked` |
| `TestRevocationPOC_StaleListFailsClosed` | CRL past `NextUpdate`, certificate not listed | server rejects with `revocation list expired` |
| `TestRevocationPOC_IssuerWithoutCRLIsAccepted` | no CRL for the issuer | handshake completes — coverage limit |

```bash
go test ./internal/radiusd/plugins/eap/tlsengine/ -count=1
```

Passed on 2026-10-29 (`dev`), including the pre-existing engine, identity,
cipher and application-data tests.

## What is deliberately not done

- **No production wiring.** `NewSettingsTLSConfigProvider` does not set
  `VerifyConnection`, so production behaviour is unchanged. Enabling it needs:
  a revocation material source in the managed certificate store (a CRL alongside
  the CA), a config schema entry, a refresh policy, and operator UI — each a
  TR-F004 surface change that requires its own checklist decision.
- **No OCSP.** See above.
- **No RadSec coverage.** `radsec_tls.go` would need the same hook plus a
  decision on certificate-less peers.

## Residual risks

- The window between a compromise and the next CRL publication is not bounded by
  this design; it is bounded by the CA's publication interval.
- A CRL is per-issuer and can be large; parsing it per handshake would be
  wasteful, so a cached parse per reload is expected before production wiring.
- The POC verifies a single CRL set against a single CA; cross-signed
  intermediates and delegated CRL signers (RFC 5280 §4.2.1.6
  `cRLSign`/indirect CRLs) are not exercised.
- EAP-TLS handshakes are frequent (every session), so the per-handshake cost of
  the check must stay in the microsecond range; the POC does not benchmark it.

## References

- RFC 5216 §2.2, §5.3 — EAP-TLS peer certificate authentication.
- RFC 5280 §5, §5.1.2.4–5, §6.3 — CRL format, validity window, processing.
- RFC 6960 — OCSP (evaluated, deferred).
- RFC 6614 §2.4.2 — RadSec peer certificate validation.
- RFC 9190 — EAP-TLS 1.3 (the reason a TLS 1.3 client may complete its side
  before the server's rejection alert arrives).
