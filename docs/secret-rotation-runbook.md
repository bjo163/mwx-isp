# SNMP credential rotation when `web.secret` changes (TR-F031 / ENG-024)

`web.secret` does two jobs: it signs admin JWTs, and its key material seals the
SNMP credentials stored by the network monitor. Changing it without a migration
makes every stored SNMP credential unreadable: the probe reports "SNMP
credentials unavailable" and the target stops producing interface metrics.
`cmd/rotate-secrets` re-encrypts those values from the old secret to the new one.

## What is affected

- `net_monitor_target.snmp_community_encrypted` (SNMPv2c community),
  `snmp_auth_encrypted` and `snmp_privacy_encrypted` (SNMPv3 authPriv
  passwords). They are AES-256-GCM ciphertext produced by
  `internal/securestore`, versioned with byte `0x01` and authenticated with the
  additional data `mwx-isp-secret-v1`; the key is derived from `web.secret`.
- Admin sessions: every issued token is signed with the same secret, so a
  rotation invalidates all of them and every operator must sign in again.
- Not affected: the backup passphrase (`TOUGHRADIUS_BACKUP_PASSPHRASE`) is a
  separate key, and RADIUS, portal and voucher passwords are stored elsewhere.

Application JSON backups and PostgreSQL dumps contain the ciphertext byte for
byte. Restoring one after a rotation writes back the old ciphertext; see
[Backups and cutover](#backups-and-cutover).

## Tool

```sh
go run ./cmd/rotate-secrets -c mwx-isp.yml              # dry run (default)
go run ./cmd/rotate-secrets -c mwx-isp.yml -apply      # persist
```

The tool reads both secrets from the environment, so they never appear in the
process list or shell history:

- `MWX_ISP_OLD_WEB_SECRET` — the secret currently sealing the stored values.
- `MWX_ISP_NEW_WEB_SECRET` — the secret that will be configured next.

It runs one transaction over every tenant. All credentials are decrypted with
the old secret and re-encrypted with the new one; if any value fails to
authenticate, the transaction rolls back and no row changes. A dry run performs
the same updates and then rolls them back, so it proves the data can be
re-encrypted without changing persisted state. Output contains counts only —
never a community, password or secret.

## Preconditions

1. A fresh VPS backup (`scripts/backup-db.sh`) and a copy of the current
   configuration, including the current `web.secret`.
2. A maintenance window. Stop the service before applying: the running process
   keeps the old secret in memory, so it could not read the rotated values.
3. The new secret is high-entropy and stored where the deployment keeps its
   other secrets (config file or `TOUGHRADIUS_WEB_SECRET`).

## Procedure

1. Count the credentials that should rotate:

   ```sql
   SELECT count(*) FROM net_monitor_target
   WHERE snmp_community_encrypted IS NOT NULL
      OR snmp_auth_encrypted IS NOT NULL
      OR snmp_privacy_encrypted IS NOT NULL;
   ```

2. Stop the MWX-ISP service.

3. Dry run with the old and new secret set in the environment. Expect
   `ok apply=false targets=N fields=M`, where `targets` matches step 1 and
   `fields` counts the non-empty credential columns. Nothing is written.

4. Apply: re-run the same command with `-apply`. Expect `ok apply=true ...`
   with the same counts.

5. Put the new secret in the configuration (`web.secret` or
   `TOUGHRADIUS_WEB_SECRET`), start the service, and confirm `/ready` is green.

6. Verify: sign in (old sessions are expected to be rejected), open Network &
   Alerts, confirm the target still reports its SNMP credentials as configured,
   and check that the next poll is reachable with interface metrics. Search the
   application log for the community string — it must not appear; the log
   export redaction treats `community` as a secret key as a second line of
   defence (`docs/log-redaction.md`).

7. If encrypted application backups are part of the retention window, export a
   fresh one so the recovery copy matches the new secret.

## Rollback

- **Dry run or aborted run:** nothing was written. Fix the mismatched secret or
  inventory and retry.
- **Applied but the cutover fails:** run the tool in reverse while the service is
  stopped, then restart with the old secret:

  ```sh
  MWX_ISP_OLD_WEB_SECRET=<new> MWX_ISP_NEW_WEB_SECRET=<old> \
    go run ./cmd/rotate-secrets -c mwx-isp.yml -apply
  ```

- **Database damaged or mixed state:** restore the safety backup taken in the
  preconditions and keep the old `web.secret`.

## Backups and cutover

Ciphertext in a snapshot is bound to the secret that was active when the
snapshot was taken. After restoring a pre-rotation JSON export or database dump:

- restore the previous `web.secret` long enough to run the tool again, or
- re-enter the SNMP credentials for each target in the UI, which seals them
  with the current secret.

The reverse ordering matters: rotating the secret and then restoring an old
snapshot returns the installation to old ciphertext with a new secret, and the
monitor cannot decrypt it until one of the steps above is done.

## Limits

- No zero-downtime rotation: the running process holds one key, so the service
  is stopped between the apply step and the restart.
- The tool rotates only SNMP monitor credentials. Any future securestore field
  must be added to `RotateSNMPSecrets` before it is used.
- The tool does not validate the new secret's strength or update the
  configuration file; that remains the operator's step.
- A shared secret across replicas means every replica must be restarted with the
  new value.

## Verification

| Check | Command |
| --- | --- |
| Dry run rolls back, apply re-encrypts all three columns, failure is atomic, secrets never appear in errors | `go test ./internal/networkmonitor/ -run TestRotateSNMPSecrets -count=1` |
| CLI dry run/apply on a SQLite fixture, with no plaintext or secret in the output | `go test ./cmd/rotate-secrets/ -count=1` |
