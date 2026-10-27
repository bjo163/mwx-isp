# Isolated backup/restore drill — 2026-10-27 (H06)

This report records an executed restore rehearsal for the application snapshot
path (`GET /api/v1/system/backup` → `POST /api/v1/system/restore`) on a
throwaway PostgreSQL instance. It is evidence for H06 and complements the
procedure in [backup-and-recovery.md](backup-and-recovery.md). No production
database, production volume, or real subscriber data was used.

## Environment

| Item | Value |
| --- | --- |
| Date (UTC) | 2026-10-27T03:14Z (test run started 03:14, log captured 03:16) |
| Source revision | `dev` at `660848d` plus the H06 drill-logging change; latest release tag `v0.13.0` |
| Go toolchain | go1.27.0 windows/amd64 |
| Database | PostgreSQL 16.15 (Alpine), `postgres:16-alpine` container |
| Isolation | `docker compose -p mwx-isp-h06 -f docker-compose.test.yml up -d postgres`, loopback port 15432, tmpfs data directory |
| Test database | `toughradius_it_23360_1793070931050778300`, created and dropped by the integration harness |
| Orchestration | `go test -tags=integration ./test/integration/ -run 'TestBackupRestoreDrillEncryptedPostgres|TestBackupRestoreRejectsCrossTenantAndRollsBackPartialFailure'` |

The harness seeds the default tenant plus a second synthetic tenant
(`drill-<suffix>`) with its own NAS, profile, RADIUS user, customer, package,
subscription, voucher batch/voucher, notification settings, certificate and an
online session. Two tenants therefore exist in the exported snapshot.

## Result summary

| Metric | Result |
| --- | --- |
| Backup envelope | encrypted, schema `9.3`, `pbkdf2-sha256` |
| Artifact size | 150,584 bytes |
| Payload SHA-256 | `add8032ed4a91903cf72fa37d20e9478d057a67a695c9668bf6a52f9346373b7` |
| Manifest counts | tenants 2, users 7, vouchers 1, certs 1 |
| Export duration | 1,727 ms |
| Restore duration | 789 ms |
| Total drill test | 5,819 ms (seed, delete, dry-run, wrong-key, restore, assertions) |
| Restore applied | users 7, NAS 4, certs 1, customers 7, subscriptions 7, vouchers 1, notification settings 1 |

Both drill tests passed:

- `TestBackupRestoreDrillEncryptedPostgres` — encrypted export excludes record
  contents; deleting the drill tenant then restoring rebuilds tenant ownership,
  RADIUS/NAS secrets, voucher credentials, billing relationships, notification
  settings and the certificate private key; the online session is intentionally
  not restored.
- `TestBackupRestoreRejectsCrossTenantAndRollsBackPartialFailure` — a
  cross-tenant reference is rejected before writing, and a mid-transaction
  unique-constraint failure leaves the database unchanged.

The run also verified the safe-failure paths on the same isolated database:

- Dry-run restore returned the manifest counts and inserted no rows.
- A wrong passphrase returned `BACKUP_DECRYPT_FAILED` with no writes.
- Canceled/rollback cases left the pre-restore state intact.

## Expected data loss

After restoring the application snapshot on a compatible schema:

- **Not restored by design:** `radius_online` active sessions and in-memory
  runtime/cache state. Subscribers reconnect and the rows are rebuilt. This was
  asserted in the drill.
- **Not part of the JSON snapshot:** database schema, application data volume
  (certificate files outside the `certs` table, logs, uploads) and
  Docker/Caddy configuration. A complete recovery needs the VPS backup set from
  `scripts/backup-db.sh`.
- **Destination-only records are preserved:** restore is an upsert by ID; it
  does not delete rows that exist only in the target database.

## WAL / point-in-time recovery

WAL archiving and PITR are **not configured** in the current deployment. Both
paths are logical: `pg_dump -Fc` for the VPS set and the application JSON
snapshot for the API path. The achievable RPO is the backup interval (daily for
the VPS schedule), not per-transaction. If WAL archiving is enabled later, this
drill must be repeated to measure PITR recovery time and data loss.

## Rollback matrix

| Scenario | Compatibility | Procedure |
| --- | --- | --- |
| Data-only rollback within schema `9.x` | Compatible: same major is accepted by `validateBackup`; exact `9.3` requires tenant records. | Re-upload the previous JSON snapshot; restore is transactional and rolls back completely on validation/constraint failure. |
| Binary rollback after a newer binary migrated the schema | **Forward-only.** `AutoMigrate` has no down migrations; the older binary is not guaranteed to work on the newer schema. | Restore the matching VPS backup set. `restore-db.sh` enforces the recorded `app_image_id`/tag, checks `SHA256SUMS`, takes a safety backup, then replaces the database and data volume before starting that image. |
| Rollback after a failed VPS restore verification | Compatible: the safety backup is taken before the replacement. | Run `restore-db.sh` with the safety backup directory and keep the previous image tag until verification passes. |

The application JSON restore is therefore a safe data rollback only while the
schema major remains compatible. Once a migration has shipped, recovering the
old binary and old schema requires the VPS backup set, which is why the image
identity is part of its metadata.

## Limits of this drill

- Isolated logical restore only; the full `scripts/backup-db.sh` /
  `scripts/restore-db.sh` VPS exercise (root, Compose installation, image
  pull, data volume, health gate) was **not** run here and remains an
  environment-bound operational rehearsal.
- No WAL/PITR, no off-host copy, no Caddy/Compose restore.
- No production RTO is claimed. The measured 789 ms restore is handler time for
  150 KB of synthetic data on a local container; it is not a production sizing
  result.

## Reproduce

```bash
docker compose -p mwx-isp-h06 -f docker-compose.test.yml up -d postgres
TEST_DATABASE_HOST=127.0.0.1 TEST_DATABASE_PORT=15432 \
TEST_DATABASE_USER=mwxisp TEST_DATABASE_PASSWORD=mwxisp TEST_DATABASE_NAME=postgres \
go test -tags=integration -count=1 -v ./test/integration/ \
  -run 'TestBackupRestoreDrillEncryptedPostgres|TestBackupRestoreRejectsCrossTenantAndRollsBackPartialFailure'
docker compose -p mwx-isp-h06 -f docker-compose.test.yml down -v
```

The `H06_DRILL` log lines in the verbose output carry the environment,
checksum, counts and timings reproduced above.
