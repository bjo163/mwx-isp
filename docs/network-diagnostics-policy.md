# Network diagnostics target policy (TR-F031, ENG-027)

`POST /api/v1/network/diagnostics/ping` is an administrator reachability
probe: it opens a TCP connection to one target and reports latency and loss.
Without limits it would be a general port scanner running inside the MWX-ISP
host, able to reach loopback services, cloud metadata addresses, and any
network the host can route to. This policy defines what it may probe, and the
same address rules govern the scheduled **monitor targets** that poll a device
on an interval.

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
| This network / broadcast | `0.10.0.1`, `255.255.255.255` | Reserved, never a device |
| Loopback | `127.0.0.1`, `::1`, `::ffff:127.0.0.1` | Would probe services on the MWX-ISP host |
| Link-local | `169.254.169.254`, `fe80::1`, `fe80::1%eth0` | Cloud metadata and APIPA; the classic SSRF target |
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

A monitor target polls on a schedule, so its address is a standing instruction
rather than a one-off request, and it follows the same rules:

- **Input** (`POST`/`PUT /api/v1/network/monitor-targets`, `INVALID_TARGET`)
  applies the table above: an explicit IP literal only — a hostname would be
  resolved at probe time (DNS rebinding) — and no unspecified, "this network",
  broadcast, loopback, link-local, or multicast address.
- **Loopback is refused here as well.** Registration decides *which device* is
  polled; it does not decide whether the probe may reach the daemon's own
  listeners. A local service is reachable through the host's LAN address, which
  is part of the registered inventory. (An earlier revision allowed loopback for
  this case; ENG-027 closed it.)
- **The address is re-checked at egress**, immediately before each probe. Rows
  written before this rule existed are therefore refused rather than dialed: the
  poller logs a warning, the manual check returns `CHECK_FAILED`, and a refusal
  does not record a "down" sample or open an incident.
- **The HTTP probe never follows a redirect.** The `Location` header is an
  address no operator validated, so a monitored device could otherwise pivot the
  probe to any host the daemon can reach. The first response decides the result,
  and a 3xx counts as reachable, which is what a redirected check used to report.

## Not a scanner

Nothing in the monitor accepts a range or a list: one target is one address, one
probe type (`icmp`, `tcp`, `snmp`, or `http`), and one port. The quick check
endpoint checks a single registered target on demand, an admin can register at
most 500 targets, and the poller runs at most eight probes concurrently on the
configured 30–3600 second interval. There is no port sweep, topology discovery,
or arbitrary payload; the HTTP probe issues one `GET` and reads no body.

## Verification

```bash
go test ./internal/adminapi/ -run 'TestParseProbeTarget|TestAuthorizeProbeTarget|TestMonitorTargetRejectsLinkLocal|TestCreateMonitorTargetEnforcesAddressPolicy|TestLivePing' -count=1
go test ./internal/networkmonitor/ -run 'TestCheckTargetAddress|TestMonitorCheckTargetRefusesStoredLoopback|TestHTTPProbeDoesNotFollowRedirects' -count=1
```

- `TestParseProbeTargetRejectsMalformedAndUnsafeInputs` covers hostnames
  (rebinding), malformed ports, loopback, link-local/metadata, unspecified, and
  multicast.
- `TestAuthorizeProbeTargetRequiresRegistrationForPrivateRanges` covers public
  targets, unregistered private/CGNAT addresses, and registration through an
  IPAM pool, a NAS record, and a monitor target.
- `TestLivePingRejectsUnsafeTargets` drives the HTTP handler and asserts the
  API error codes; `TestLivePingIsRateLimited` pins the limiter.
- `TestMonitorTargetRejectsLinkLocalAddress` pins the registration rule, and
  `TestCreateMonitorTargetEnforcesAddressPolicy` drives the create handler with
  loopback, metadata, a hostname, the unspecified address, and a private address.
- `TestCheckTargetAddress` is the policy table (allowed private/public/ULA,
  refused loopback, link-local, multicast, "this network", broadcast, hostnames,
  URLs, and `ip:port`); `TestMonitorCheckTargetRefusesStoredLoopback` proves a
  legacy row is refused at egress without fabricating an outage; and
  `TestHTTPProbeDoesNotFollowRedirects` pins the redirect policy.

## Limits

- Public unicast addresses can still be probed, and a registered device's ports
  can be probed. That is the intended diagnostic surface, not an internet-wide
  scanner.
- The check is a TCP connect, not ICMP; the probe cannot send payloads.
- Revoking an IPAM pool or monitor target removes it from the authorised set on
  the next request; there is no cached allowlist to expire.
- The monitor's SNMP and TCP probes resolve no names either: with hostnames
  refused, there is no DNS step to rebind.
- Rejecting loopback is a policy decision, not a technical limit: an operator who
  genuinely needs to monitor a service on the MWX-ISP host can point the target
  at the host's LAN address.
