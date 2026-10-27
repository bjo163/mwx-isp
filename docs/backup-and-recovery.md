# MWX-ISP backup and recovery

MWX-ISP has two different backup paths. The application JSON export is a
portable snapshot of selected application records. The VPS backup set captures
the PostgreSQL database, application data volume, checksums, and matching app
image metadata. Neither should be treated as an off-host copy until an operator
copies it to a separately protected system.

## Application JSON export

The platform-admin endpoint `GET /api/v1/system/backup` downloads the versioned
snapshot. Schema `9.3` includes tenant and operator data, RADIUS profiles, users
and finalized accounting, ISP customers/packages/subscriptions/billing, hotspot
batches and vouchers, IPAM pools, trouble tickets/work orders, ODPs,
network-monitor configuration/history, notification settings/outbox, and
document sequences. Tenant ownership is stored in the backup's `tenant_ids`
map because the normal API intentionally hides `TenantID`.

### Envelope and manifest

Exports are wrapped in a `mwx-isp-backup` envelope (format version `1`):

- `manifest`: `schema_version`, `created_at`, `record_counts` for every
  exported table, a SHA-256 digest of the canonical payload, and the encryption
  descriptor when applicable.
- `payload`: the `SystemBackup` document when encryption is disabled.
- `ciphertext`: Base64 of `nonce || AES-256-GCM ciphertext` when encryption is
  enabled; `payload` is then omitted so record contents never appear in
  plaintext.

The digest is computed over the canonical compact JSON encoding of the backup
document, not over the transmitted bytes, so a re-indented or re-wrapped file
still verifies while modified record content does not. On restore the server
requires the manifest schema to match, recomputes the digest, and compares
every table's record count in both directions. A missing or extra table count
is rejected.

Bare pre-envelope backups (a plain `SystemBackup` document without `format`)
remain accepted for compatibility; they carry no manifest or encryption and
should only be used for migration, not routine recovery.

### Encryption

Set `TOUGHRADIUS_BACKUP_PASSPHRASE` (or `backup.passphrase` in the config file)
on the server. Exports then use AES-256-GCM with a key derived from the
passphrase via PBKDF2-SHA256 (600,000 iterations, random 16-byte salt); the
manifest and its digest are authenticated as additional data. The endpoint
returns a `mwx-isp-backup-<timestamp>.encrypted.json` download and the same
ciphertext in the response body.

The passphrase is never serialized by the API, written to logs, or included in
the backup. Store it in the deployment secret store or an offline escrow, never
beside the backup files:

- Restores require the same passphrase. Without it, the server rejects the file
  with `BACKUP_ENCRYPTED`; a wrong key fails with `BACKUP_DECRYPT_FAILED`.
- Do not change or lose the escrowed passphrase while encrypted backups are
  inside the retention window. To rotate it, decrypt with the old value, set the
  new value, and export again.
- If the passphrase is lost, encrypted backups cannot be recovered; retain a
  separately protected recovery copy of the key.

### Restore behavior

Restore uses an upsert by record ID inside one database transaction. It applies
captured tenant ownership, validates references, and preserves records that
exist only in the destination. It does not delete destination-only rows,
restore database schema, replace the application data volume, or restore
Docker/Caddy configuration. A validation or unique-constraint failure rolls the
whole transaction back, leaving the database unchanged. Operators can only be
restored by a super operator, and platform-admin rows are preserved.

Active RADIUS online-session rows and in-memory runtime/cache state are
intentionally not exported: they are transient and are rebuilt as devices
reconnect. Finalized accounting history is exported.

### Dry run

Before a destructive restore, POST the same file with `?dry_run=true` (or a
`dry_run=true` form field). The server parses the file, decrypts and verifies
the manifest, applies tenant ownership, runs all validation and authorization
checks, and returns `dry_run`, `encrypted`, `manifest`, and `would_restore`
record counts without opening a transaction or writing any row. A dry run that
reports success does not itself change the live system.

### Sensitive material

The JSON contains sensitive material, including RADIUS and voucher passwords,
operator password hashes, certificate private keys, and encrypted monitor
credentials. Monitor credentials are sealed with `web.secret`, so a snapshot
restored after that secret changes needs the rotation steps in
[secret-rotation-runbook.md](secret-rotation-runbook.md). Treat the download and
stored copy as secrets. Use encrypted
exports for anything that leaves the host. The endpoint also attempts to write a
local copy under the configured backup directory; use the VPS backup set for
operational recovery and verify the resulting artifacts.

## VPS database and application-data backup

Run `scripts/backup-db.sh` as root on the installed VPS. By default, it writes
to `/var/backups/mwx-isp/<UTC timestamp>/`:

- `database.dump`: PostgreSQL custom-format dump.
- `application-data.tar.gz`: the app data volume.
- `metadata.txt`: source revision and matching application image identity.
- `SHA256SUMS`: checksums for the backup artifacts.

The script expects the app and database containers to be running, uses a lock
to avoid overlapping backup/update operations, and applies local retention.
Copy completed backup directories off-host using a protected channel and keep
the matching image available for the recovery window.

## Restore

`scripts/restore-db.sh <backup-directory>` validates checksums and matching app
image metadata, creates a safety backup of the current installation, then
replaces the live PostgreSQL database and app data volume. It requires an
explicit `RESTORE` confirmation unless `MWX_ISP_RESTORE_CONFIRM=RESTORE` is set.

For an application JSON snapshot, restore it through the Admin API while the
service is running, preferably after a VPS safety backup:

1. Validate the artifact first: run the dry-run request and compare the
   reported manifest counts with the expected export.
2. Restore the file with `POST /api/v1/system/restore` and confirm the response
   counts match the manifest.
3. Re-authenticate: a restore can invalidate cached session/token state.

Restore scripts are destructive to the current database and app data after the
safety backup. Run them only with the exact backup directory selected and with
the VPS installation stopped from other update/restore work.

## Verification and rollback

After any restore, verify before returning traffic:

- `/ready` and container health stay green, and logs show no restore errors.
- Sign in and check tenant, subscriber, package, voucher, and ticket counts
  against the manifest.
- Spot-check a RADIUS authentication and a voucher redemption, and confirm
  certificate-based EAP still presents the restored certificate.
- Confirm billing/subscription links and notification settings point at the
  expected tenants.

If verification fails, roll back by restoring the safety backup created by
`restore-db.sh` (or the pre-restore JSON snapshot) and start the matching app
image. Keep the previous image tag and backup directory until the new state has
been verified and the retention window closes.

## Retention, RPO, and RTO

Suggested minimum retention is 7 daily, 4 weekly, and 12 monthly backup sets,
with at least one copy off-host and encrypted where it leaves the trusted
network. Rotation must not delete the only copy covering the current retention
window, and the matching container image must stay available for that window.

The achievable RPO is the backup interval: a daily VPS schedule loses at most
one day of finalized data, while the JSON export's RPO depends on how often it
is taken. The application export does not include the database schema or the
application data volume, so a complete recovery needs both paths.

No production RTO is claimed yet. CI exercises the JSON export/restore path,
including encrypted export, tamper rejection, dry-run zero mutation, transaction
rollback on failure, and a PostgreSQL round-trip with relationships and secrets
intact. An isolated PostgreSQL drill executed on 2026-10-27 recorded the
artifact checksum, manifest counts, export/restore duration, expected data loss
and the rollback matrix in
[backup-restore-drill-2026-10-27.md](backup-restore-drill-2026-10-27.md). That
drill does not certify the full VPS disaster-recovery procedure: WAL/PITR is
not configured, and the production restore script, off-host copy, image
provenance and data-volume restore still need a scheduled isolated VPS exercise
before a recovery-time objective can be claimed.
