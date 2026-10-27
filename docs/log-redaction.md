# Log export redaction (TR-F022)

`/admin/loki/query` (`internal/adminapi/system_logs.go`) reads the application
log file and returns entries to a platform administrator. Operators still filter
on the raw line, but only the redacted form leaves the process, so the same rule
applies whether the file is an on-host log or a future support bundle
(`ENG-060`).

## Data classes

| Class | Keys | Export form |
| --- | --- | --- |
| Secret | `password`, `passwd`, `secret`, `token`, `authorization`, `api_key`, `credential`, `community`, `private_key` (matched as fragments of the normalized key) | `[REDACTED]` |
| IP address | `ip`, `remote_ip`, `nas_ip`, `client_ip`, `source_ip`, `destination_ip`, `ipaddr`, `*ip` | IPv4 `/24` (`203.0.113.x`), IPv6 `/48` (`2001:db8:1234::x`) |
| Customer PII | `phone`, `msisdn`, `email`, `address`, `full_name`, `customer_name`, `contact`, `nik` | `sha256:<16 hex>` pseudonym |
| Operational | `username`, `namespace`, `job`, `level`, tenant/row IDs | unchanged |

Free text (`msg`, `message`, any other string field) is also swept for
`password=`/`token:`-style assignments, e-mail addresses and IPv4 addresses, so
a value that leaks into the message body is still redacted.

## Correlation

Pseudonyms and masks are deterministic, so two entries about the same customer or
the same /24 can still be tied together, and the redacted output keeps the
timestamp, level, namespace, job and username that the audit trail is keyed on.
`TestRedactSystemLogEntryPseudonymsAreStable` pins this.

## Verification

- `go test ./internal/adminapi/ -run 'TestRedactSystemLog|TestMaskSystemLog|TestQuerySystemLogs'`
  — a fixture line carrying a password, bearer token, customer name, phone,
  e-mail and host IP is exported with none of those raw values, while the
  username survives and the address keeps its prefix.
- The endpoint-level test exercises `querySystemLogs`, not just the helper, so a
  future refactor that bypasses the redaction pass fails the unit lane.

## Known limits

- Phone numbers and postal addresses are only recognised by their key name; a
  phone number embedded in free text is not masked. Add a pattern here (and a
  fixture) before relying on that case.
- IPv6 addresses in free text are not masked; structured `*ip` fields are.
- The rules cover this export path. Log writers should still avoid logging
  credentials in the first place; this is a last line of defence, not a licence
  to log sensitive values.
