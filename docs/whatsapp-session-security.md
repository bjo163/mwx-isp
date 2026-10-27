# WhatsApp session protection (TR-F030, ENG-025)

The optional WhatsApp transport (`internal/notify`) is a linked WhatsApp Web
device. Unlike an API token, a paired session is a full set of client
credentials, so the deployment database that holds it is as sensitive as the
operator's phone.

## What is stored where

`NewWhatsAppManager` attaches whatsmeow's SQL store to the application's own
database, using the same DSN and credentials as the rest of MWX-ISP. Pairing
creates rows in 16 `whatsmeow_*` tables. The material that matters:

| Table | Content |
| --- | --- |
| `whatsmeow_device` | Noise key pair, identity key pair, signed pre-key and signature, adv secret key, signed device identity, push name |
| `whatsmeow_identity_keys` | Contact identity keys, keyed by the linked JID |
| `whatsmeow_pre_keys`, `whatsmeow_sessions`, `whatsmeow_sender_keys` | Signal session state for every correspondent |
| `whatsmeow_app_state_*`, `whatsmeow_contacts`, `whatsmeow_chat_settings`, `whatsmeow_privacy_tokens`, `whatsmeow_nct_salt`, `whatsmeow_lid_map` | App-state sync keys, address book, mappings |
| `whatsmeow_message_secrets`, `whatsmeow_event_buffer`, `whatsmeow_retry_buffer` | Message secrets and buffered (still encrypted) transport payloads |

`whatsmeow_device` is the root: every other table references its JID with
`ON DELETE CASCADE`, so removing the device row removes the derived key
material. That cascade only runs while SQLite enforces foreign keys; the manager
enables `PRAGMA foreign_keys=ON` before whatsmeow's own `Upgrade` runs, and
`Upgrade` refuses to start if enforcement is off. PostgreSQL enforces the
constraint unconditionally.

Consequences worth stating plainly:

- **A database dump or copy is a linked device.** `scripts/backup-db.sh`
  (`pg_dump -Fc`) includes the `whatsmeow_*` tables. Anyone who can restore or
  read that dump holds the session keys.
- **The JSON system backup does not contain them.** `SystemBackup` in
  `internal/adminapi/backup_envelope.go` has no `whatsmeow_*` records and
  restore does not touch those tables, so restoring a JSON snapshot neither
  revives nor destroys a paired session.
- File permissions: the installer sets `APP_DIR` to `0750` and `.env` to
  `0600 root:root` (`scripts/vps-install.sh`); SQLite data lives under
  `workdir/data`, created `0700` (`internal/app/database.go`); host backups use
  `umask 077` and a `0700` root directory (`scripts/backup-db.sh`).

## Logging posture

No session material can reach a log from this package:

- Both whatsmeow loggers are `waLog.Noop` — the SQL container and the client.
- `internal/notify` imports no logger at all; it returns errors rather than
  logging, and the errors describe the failure ("delete WhatsApp session after
  failed logout"), never key bytes or the QR payload.
- `Status()` deliberately omits all key fields; it exposes connection state,
  the linked account JID, and — while pairing — the short-lived QR image.
  `GET /api/v1/system/notifications/whatsapp` returns it to an administrator;
  pairing and disconnect require a platform administrator
  (`internal/adminapi/notifications.go`).

`TestWhatsAppTransportOnlyUsesSilentLogger` scans the package source and fails
if a logger other than `waLog.Noop` is introduced.

## Revoking a session

The supported path is the administration UI's **Disconnect** action
(`POST /api/v1/system/notifications/whatsapp/disconnect`, platform
administrator), backed by `WhatsAppManager.Disconnect`:

1. It cancels any in-flight pairing, clears the in-memory client and QR, and
   marks the manager `not_linked`.
2. With a live, logged-in client it calls `Logout`, which asks WhatsApp to
   remove the companion device.
3. Whether or not the remote call succeeds, it deletes the local device row.
   Derived identity/session rows go with it through the cascades above.
4. If the remote logout failed *and* the local delete also failed, the endpoint
   now returns an error instead of reporting success — the previous code
   swallowed that error, which could leave key material behind while telling
   the operator the device was unlinked.

After revoking, verify:

- `GET /api/v1/system/notifications/whatsapp` returns `state: "not_linked"`,
  `logged_in: false`, no `account`, and no `qr_image`.
- In the database: `SELECT count(*) FROM whatsmeow_device;` is `0`, and the
  same holds for `whatsmeow_identity_keys` and `whatsmeow_sessions`.
- **On the operator's phone:** WhatsApp → Linked devices. If the host was
  offline when revoking, `Logout` cannot reach WhatsApp, so only local keys are
  gone and the device is still listed. Remove it there; that is the step that
  actually invalidates the credential on WhatsApp's side.
- Delete or rotate database dumps taken while the device was linked; they still
  contain usable session keys. Retention in `scripts/backup-db.sh` (default 30
  days) is the only automated cleanup.

## Test coverage

The revoke path is exercised in a test environment without a phone:

- `TestDisconnectRevokesPersistedWhatsAppSession` and
  `TestUnlinkRevokesPersistedWhatsAppSession` seed a fabricated paired device
  (device keys, one contact identity, one session), then assert that
  `whatsmeow_device`, `whatsmeow_identity_keys` and `whatsmeow_sessions` are
  empty and the status is `not_linked`.
- `TestWhatsAppManagerEnforcesForeignKeysOnSQLite` pins the cascade precondition
  (whatsmeow refuses to start when the pragma is missing).
- `TestWhatsAppStatusDoesNotExposeSessionMaterial` asserts the status JSON has
  no key or secret fields.

```bash
go test ./internal/notify/ -run 'TestWhatsApp|TestDisconnect|TestUnlink' -count=1
```

## Limits

- The remote logout branch needs a real linked account; CI only covers the
  local-delete branch (`IsLoggedIn` is false for an unconnected client) and the
  cascade semantics. Confirm the phone-side removal in the pairing UAT
  (ENG-048).
- Revoking does not scrub historical dumps, replicas, or copies made before the
  revoke. Those must be deleted under the retention policy.
- The transport is one-way and operator-allowlisted; it never processes inbound
  messages, but the session still authorises the full linked-device protocol.
