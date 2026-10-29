# Backup KDF and envelope parameters — evaluation 2026-10-29

Feature: TR-F020 (backup and restore) · backlog item ENG-026.

Scope: measure the PBKDF2 cost that protects encrypted JSON backups, decide the
envelope version/compatibility contract, check that older archives stay
restorable across a cost change, and record the passphrase policy. Anchor file:
[`internal/adminapi/backup_envelope.go`](../internal/adminapi/backup_envelope.go)
at `a5989dc`.

## Envelope contract (unchanged)

| Field | Value |
| --- | --- |
| Container format | `mwx-isp-backup` |
| Container version | 1 |
| KDF | `pbkdf2-sha256`, HMAC-SHA256, 32-byte key |
| Salt | 16 random bytes per archive |
| Cipher | AES-256-GCM, nonce prepended to ciphertext |
| Additional data | format, container version, schema version, payload digest, iteration count |
| Written iteration count | 600,000 |
| Accepted iteration count on restore | 600,000 – 2,400,000 (read from the manifest) |

No wire/format change was needed for this evaluation. Container version stays 1;
the manifest already records `kdf`, `kdf_iterations`, and `salt`, so the cost is
a property of each archive rather than of the current binary.

## Measured cost

Command (go1.27.0, AMD Ryzen 5 7500F, Windows/amd64, 12 logical CPUs):

```bash
go test ./internal/adminapi -run '^$' -bench 'BenchmarkBackup' -benchtime 3x -count 1
```

| Iterations | Derivation time | Offline guesses/second/core |
| --- | --- | --- |
| 100,000 | 13.3 ms | ~75 |
| 310,000 | 41.7 ms | ~24 |
| **600,000 (written)** | **79.5 ms** | **~12.6** |
| 1,200,000 | 161.6 ms | ~6.2 |
| 2,400,000 (accepted cap) | 355.3 ms | ~2.8 |

Full open of a 600,000-iteration archive (parse, derive, authenticate) measured
87.1 ms — the derivation accounts for essentially all of it. Backups and
restores are rare, operator-triggered actions, so 80–360 ms is not an
operator-visible cost; the number that matters is the offline guessing rate,
which scales linearly with the iteration count.

Decision: keep 600,000 as the written cost (the OWASP recommendation for
PBKDF2-HMAC-SHA256 at the time the envelope shipped) and accept a 4× window on
restore. The window is what lets a later release raise the cost for new archives
without making every existing archive undecryptable, and it bounds the work a
crafted archive can demand from a restore.

## Defect found and fixed: exact-equality iteration check

Before this change the decrypt path required the recorded count to equal the
compiled constant:

```go
if manifest.KDFIterations != backupKDFIterations {
    return nil, fmt.Errorf("%w: unsupported KDF iteration count", errBackupDecryption)
}
```

Raising `backupKDFIterations` would therefore reject every archive written by an
earlier release — a silent, unrecoverable compatibility break with no format
change to signal it. Restore now validates the recorded count against
`[backupKDFMinIterations, backupKDFMaxIterations]` and derives with the recorded
value; encryption uses `manifest.KDFIterations` instead of the constant. The
minimum equals the written cost, so a crafted archive cannot ask for a weaker
derivation than this release writes; the maximum caps the CPU spent before
authentication.

## Passphrase policy

Before: any non-empty `TOUGHRADIUS_BACKUP_PASSPHRASE` was accepted, including a
single character. The KDF cost only buys time proportional to the passphrase
entropy above it.

After: writing a new encrypted archive requires at least 12 characters and fails
with `BACKUP_PASSPHRASE_TOO_SHORT` otherwise. Restoring is exempt, so an archive
written earlier with a shorter passphrase stays readable with that passphrase.
Twelve characters is a floor, not a target: a long random value from the secret
store remains the recommendation, and the passphrase is still never serialized,
logged, or embedded in the archive.

## Evidence

Tests (all in `internal/adminapi`):

- `TestEncryptedBackupRestoresAcrossSupportedIterationCounts` — fixtures written
  at 600,000, 600,001, and 2,400,000 iterations all restore to a fresh database.
- `TestEncryptedBackupRejectsUnsupportedIterationCounts` — counts 0, 599,999,
  2,400,001, and 2^30 fail with `BACKUP_DECRYPT_FAILED` and write zero rows; the
  bound is checked before key derivation, so the 2^30 case returns immediately.
- `TestEncryptedBackupRefusesShortPassphrase` — a short passphrase refuses new
  exports, and a legacy archive encrypted with it still restores.
- `TestBackupKDFPolicyStaysConsistent` — keeps the written default inside the
  accepted window so a future bump cannot strand archives within one release.
- Existing tamper, wrong-passphrase, manifest-count, dry-run, and unknown
  container tests remain green.

Commands:

```bash
go test ./internal/adminapi -run 'TestEncryptedBackup|TestBackupEnvelope|TestRestore' -count=1
go test ./internal/adminapi -run '^$' -bench 'BenchmarkBackup' -benchtime 3x -count 1
```

## Residual limits

- Benchmarked on one x86-64 desktop, not on the target VPS: a small shared vCPU
  may be several times slower. Re-run the benchmark on deployment hardware
  before raising the written cost; that check is blocked on a claimed host.
- Archives keep the cost they were written with. Nothing re-encrypts an old
  archive at the current cost, so the recorded count (not the constant) must
  remain authoritative.
- If a future release raises `backupKDFIterations` beyond
  `backupKDFMaxIterations`, both constants must move together; the consistency
  test fails rather than silently writing archives that this release cannot
  open.
- This is not a general cryptographic review. The AES-GCM framing, salt size,
  and nonce generation are unchanged and were not re-evaluated.
