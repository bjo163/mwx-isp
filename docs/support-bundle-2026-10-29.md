# Redacted Support Bundle (ENG-060)

Status: delivered 2026-10-29 on `dev` as a local proof of concept.
Scope: `TR-F022` audit discipline — an incident-handoff package that an operator
can build on the host without a new network path.
Non-goal: uploading to a vendor, a hosted crash-reporting service, or a UI flow
inside the admin console.

## Problem

When an operator needs help with an incident they currently copy the raw
configuration file and the log by hand. Both contain credentials
(`database.passwd`, `web.secret`, `backup.passphrase`), customer identifiers
(phone numbers, e-mail addresses, names), and subscriber addresses. The log
export endpoint `/admin/loki/query` already redacts its output
([log-redaction.md](log-redaction.md)), but nothing produced a file that could be
handed to a second pair of eyes, and copying files by hand silently bypasses the
redaction pass.

The bundle tool therefore has two jobs: apply the same redaction rules to every
byte that leaves the host, and make "review before export" a step that cannot be
skipped.

## Decision

1. **Staging and export are separate commands.** `Stage` writes a staging
   directory that contains no archive. `Approve` is the only path that produces
   a `.tar.gz`, and it requires a reviewer identity. A single command that both
   redacts and ships would make the review advisory; here it is structural.
2. **Every payload file is redacted.** The configuration is parsed as YAML,
   walked key by key, and re-encoded; the log is read line by line, with
   structured (`JSON`) lines redacted per field and unstructured lines swept as
   text.
3. **The rules live in one package.** `internal/redact` is now the single
   implementation of the rules; `internal/adminapi/system_logs.go` delegates to
   it, so the log export and the bundle cannot drift apart.
4. **Approval re-verifies the bytes.** `Approve` re-reads every staged file,
   compares its size and sha256 against the manifest, recomputes the bundle
   checksum, and refuses when anything changed after staging. A reviewer who
   approves after an edit gets `ErrChecksumMismatch`, not an archive.
5. **No sockets, no database, no implicit output path.** The tool opens only the
   files named on the command line and writes only under `-out`. "Export" means
   the operator moves the archive; nothing is uploaded.
6. **The log is bounded.** Only the last `DefaultLogTailBytes` (2 MiB) are
   staged, starting at a line boundary, so a bundle covers the incident window
   instead of the whole log history.

## Bundle layout

Staged directory `support-bundle-<UTC timestamp>`:

| File | Content |
| --- | --- |
| `config.redacted.yaml` | Configuration with secrets, customer identifiers, and addresses replaced; a two-line header states the rules. |
| `logs/app.log` | Redacted log tail, same line order as the source. |
| `redaction-report.json` | Per-source input/output byte counts, counts per data class, and the dotted key paths that were redacted. |
| `meta.json` | `format`, `version`, MWX-ISP version, Go version, OS/arch, staging timestamp. |
| `manifest.json` | Per-file `path`/`bytes`/`sha256`, the bundle checksum, and the review state (`pending_review`). |
| `review.json` | Written by `Approve` only: `status`, `reviewer`, `reviewed_at`, `bundle_checksum`. |

The bundle checksum is `sha256` over the sorted `path\x00sha256` lines, so it
covers the set of payload files and not just each file in isolation. An archive
is `support-bundle-<timestamp>.tar.gz` with a `.sha256` sidecar in the
`sha256sum` format (`<digest>  <name>`).

## Redaction rules

`internal/redact` classifies a field by its key and falls back to a text sweep:

| Class | Keys (normalized: `_`, `-`, `.` removed) | Export form |
| --- | --- | --- |
| Secret | fragments `password`, `passwd`, `passphrase`, `secret`, `token`, `authorization`, `apikey`, `credential`, `community`, `privatekey` | `[REDACTED]` |
| Address | `ip`, keys ending in `ip` or `ipaddress`, keys containing `ipaddr` | IPv4 `/24` (`203.0.113.x`), IPv6 `/48` (`2001:db8:1234::x`) |
| Customer PII | `nik`, fragments `phone`, `msisdn`, `email`, `address`, `fullname`, `customername`, `contact` | `sha256:<16 hex>` stable pseudonym |
| Plain | everything else | value unchanged, unless the string itself matched a text rule |

The text sweep removes credential assignments
(`db_password: …`, `password=…`, `authorization: Bearer …`), URL userinfo
(`postgres://user:pw@host` → `postgres://user:[REDACTED]@host`), e-mail
addresses (pseudonymized), and IPv4 literals (`10.20.30.40` → `10.20.30.x`).
Pseudonyms and masks are deterministic, so two entries about the same customer
or the same /24 remain correlatable after redaction.

## Review gate

```
go run ./cmd/support-bundle -c mwx-isp.yml -log data/logs/app.log -out ./support-out
# read support-out/support-bundle-<ts>/redaction-report.json and the payload
# delete the directory if anything sensitive survived
go run ./cmd/support-bundle -approve ./support-out/support-bundle-<ts> -reviewer <name>
```

Staging prints the bundle checksum and the redacted counts, and ends with
"No archive was created and nothing was sent off-host." Approving an already
approved bundle fails with `ErrAlreadyApproved`, and a second archive for the
same directory is never produced.

## Demo

Run against a fixture `mwx-isp.yml` carrying four credentials, one operator
e-mail, and two private IPv4 addresses, plus a three-line JSON log:

```
Staged support bundle: Z:\tmp\sb-demo\out\support-bundle-20261029T003100Z
Bundle checksum: 61a9ad7d2678a1cd1ca9371ee28f6a1ae26349f0151cb1b84742c21d407c5cde
Redacted: 4 secrets, 3 customer identifiers, 2 addresses, 5 free-text values
...
Approved by operator@isp.example.id
Archive: Z:\tmp\sb-demo\out\support-bundle-20261029T003100Z.tar.gz
SHA256: 5da92df9b6ee18fb6f9f44c1187e550ce1e1d59e0a53ea3c2be6df72aa4c4ec1
```

The staged configuration keeps the operator-facing shape and loses the
sensitive values:

```yaml
backup:
    passphrase: '[REDACTED]'
database:
    host: 10.20.30.x
    passwd: '[REDACTED]'
notify:
    admin_email: sha256:672e056d4a1cbd0f
radiusd:
    radsec_key: /etc/mwx/radsec/server.key
system:
    location: Jakarta
web:
    secret: '[REDACTED]'
    host: 0.0.0.x
```

The log keeps its structure because that is what makes it useful
(`username`, `job`, `namespace`, `level`, timestamps survive) while the
credential, the e-mail address, the phone number, the customer name, and the
host part of every address do not:

```json
{"customer_name":"sha256:d37d197c75da1b43","job":"radiusd","level":"info","msg":"access request for sha256:0bfe3b7a0aea2ec2 from 203.0.113.x","namespace":"isp","password":"[REDACTED]","phone":"sha256:62397bbd6a8c9ae5","remote_ip":"203.0.113.x","tenant_id":1,"ts":1793269200,"username":"alice"}
{"job":"billing","level":"warn","msg":"db retry password=[REDACTED] dsn=postgres://mwxisp:[REDACTED]@10.20.30.x:5432/isp","namespace":"isp","ts":1793269260}
{"job":"radiusd","level":"error","msg":"coa failed for 10.20.30.x token=[REDACTED],"namespace":"isp","nas_ip":"10.20.30.x","ts":1793269320}
```

The report accounts for the difference:

```json
"totals": { "secrets": 4, "pii": 3, "addresses": 2, "free_text": 5 }
```

with `config` at 3 secrets / 1 pseudonym and `logs` at 1 secret / 2 pseudonyms /
2 addresses, and the config's `redacted_keys` naming
`backup.passphrase`, `database.passwd`, `notify.admin_email`, `web.secret`.

## Evidence

```
go build ./...

go test -count=1 ./internal/redact/ ./internal/supportbundle/ ./cmd/support-bundle/
ok  github.com/bjo163/mwx-isp/internal/redact          0.816s
ok  github.com/bjo163/mwx-isp/internal/supportbundle   1.040s
ok  github.com/bjo163/mwx-isp/cmd/support-bundle       1.008s

go test -count=1 ./internal/adminapi/ -run 'TestRedactSystemLog|TestMaskSystemLog|TestQuerySystemLogs'
ok  github.com/bjo163/mwx-isp/internal/adminapi        3.495s

golangci-lint run ./internal/redact/... ./internal/supportbundle/... ./cmd/support-bundle/...
0 issues.
```

Coverage:

- `internal/redact/redact_test.go` — key classification (including
  `framed_ipaddr` → Address and `idempotency_key` → Plain), every credential
  form in the text sweep, free-text pseudonymization and masking, and the
  stability of pseudonyms and masks.
- `internal/supportbundle/bundle_test.go` — no raw secret, e-mail, phone,
  customer name, or host part in the staged files; report totals per class and
  per source; manifest `pending_review` with matching per-file digests; staging
  is byte-identical for identical inputs (including the staging timestamp);
  a bad input leaves no staged directory; the log tail is bounded and starts on
  a line boundary; approval requires a reviewer and matching checksums; the
  archive holds exactly the six expected entries with no leaks, carries a
  sidecar digest, and cannot be approved twice; a foreign directory is refused.
- `cmd/support-bundle/main_test.go` — stage-then-approve through the CLI, usage
  errors exit `2`, missing input exits `1`, a tampered staged file is refused
  with no archive produced.

## Residual limits

- Phone numbers and postal addresses are recognised by key name only; a phone
  number embedded in free text is not masked. The same is true of IPv6
  addresses in free text. Add a pattern to `internal/redact` and a fixture here
  before relying on either case.
- The configuration is re-encoded through `yaml.v3`: comments, key order, and
  original formatting are not preserved. Values are preserved.
- Only the log tail is staged. Evidence older than the tail window has to be
  gathered separately.
- The `.sha256` sidecar is an integrity check, not a signature. It detects
  accidental corruption and post-staging edits on the same host; it does not
  authenticate the bundle to the recipient.
- The tool is a standalone command. It is not wired into the admin API or the
  web console, and there is no scheduled bundle collection.
- The output directory is operator-chosen; the tool does not enforce that it is
  outside the repository or on an encrypted volume.
- A configuration key whose value is a hostname (`database.host: db.internal`)
  is not an address-class key, so the hostname is retained. Only `ip`/`ipaddr`
  keys and IPv4 literals in text are masked.
