# Network diagnostics target policy (TR-F031, ENG-027)

`POST /api/v1/network/diagnostics/ping` is an administrator reachability
probe: it opens a TCP connection to one target and reports latency and loss.
Without limits it would be a general port scanner running inside the MWX-ISP
host, able to reach loopback services, cloud metadata addresses, and any
network the host can route to. This policy defines what it may probe.

## Accepted input

The request body is `{"host": "<ip>[:port]", "count": 1..10}`.

- The host must be an explicit IPv4 or IPv6 literal. **Hostnames are refused**
  (`TARGET_NOT_IP`): no name is resolved, so a name cannot resolve to one
  address during validation and a different one at dial time (DNS rebinding),
  and the endpoint is not a name resolver. Malformed values (`:0`, `:65536`,
  `:http`, empty host) are refused before any lookup.
- The port defaults to `80` and must be `1..65535`.
- A request probes a single target at most ten times. There is no target list,
  range, or sweep input.

## Refused destinations

These are refused for every role, whatever the inventory says
(`TARGET_NOT_ALLOWED`):

| Class | Example | Why |
| --- | --- | --- |
| Unspecified | `0.0.0.0`, `::` | Not a destination |
| Loopback | `127.0.0.1`, `::1` | Would probe services on the MWX-ISP host |
| Link-local | `169.254.169.254`, `fe80::1` | Cloud metadata and APIPA; the classic SSRF target |
| Multicast / interface-local | `224.0.0.1`, `ff02::1` | Not probeable by connect |

## Private and public destinations

- **Public unicast** (for example `8.8.8.8`) is allowed: an upstream
  connectivity check is a normal operator action.
- **Private, CGNAT (`100.64.0.0/10`), and IPv6 unique-local** destinations are
  allowed only when they belong to the tenant's registered inventory
  (`TARGET_NOT_REGISTERED` otherwise):
  - inside a configured IPAM pool CIDR,
  - equal to a NAS `ipaddr`, or
  - equal to a registered monitor target address.

  The endpoint therefore cannot sweep an unrelated network: every private
  address it can reach is one the tenant has already declared. Register a
  monitor target if a device outside the inventory has to be probed.

The inventory queries run through the tenant-scoped database handle, so one
tenant's pools, NAS records, and monitor targets do not authorise another
tenant's probe.

## Rate limit

The route is admin-only and additionally limited to one request every two
seconds with a burst of five, keyed by tenant and resolved client IP (the same
resolved IP the login limiter uses). Combined with the single-target input and
the inventory rule above, it cannot be driven as a scanner.

## Monitor target registration

The same rules apply when a monitor target is created or updated: the address
must be one explicit unicast IP, link-local addresses are refused, and
unspecified or multicast addresses were already refused. Loopback remains
valid here because registration is itself the explicit operator decision, and
a local service check is a legitimate use (unlike an ad-hoc probe of an
arbitrary host).

## Verification

```bash
go test ./internal/adminapi/ -run 'TestParseProbeTarget|TestAuthorizeProbeTarget|TestMonitorTargetRejectsLinkLocal|TestLivePing' -count=1
```

- `TestParseProbeTargetRejectsMalformedAndUnsafeInputs` covers hostnames
  (rebinding), malformed ports, loopback, link-local/metadata, unspecified, and
  multicast.
- `TestAuthorizeProbeTargetRequiresRegistrationForPrivateRanges` covers public
  targets, unregistered private/CGNAT addresses, and registration through an
  IPAM pool, a NAS record, and a monitor target.
- `TestLivePingRejectsUnsafeTargets` drives the HTTP handler and asserts the
  API error codes; `TestLivePingIsRateLimited` pins the limiter.
- `TestMonitorTargetRejectsLinkLocalAddress` pins the registration rule.

## Limits

- Public unicast addresses can still be probed, and a registered device's ports
  can be probed. That is the intended diagnostic surface, not an internet-wide
  scanner.
- The check is a TCP connect, not ICMP; the probe cannot send payloads.
- Revoking an IPAM pool or monitor target removes it from the authorised set on
  the next request; there is no cached allowlist to expire.
