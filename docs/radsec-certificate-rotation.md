# RadSec and EAP certificate rotation (TR-F003 / ENG-004)

RadSec (RADIUS over TLS) used to load its key pair and client CA bundle once at
startup, so replacing a certificate meant restarting the daemon and dropping
every live TLS connection. The TLS material is now reloaded in place: the server
checks the configured files on each TLS handshake and swaps in new material when
one of them changed. The same rotation can be rolled back the same way, and a
badly written file leaves the last known-good material serving.

EAP-TLS / PEAP / EAP-TTLS never had that limitation: their providers resolve the
managed certificate from the database on every handshake, so selecting or
re-importing a certificate takes effect on the next handshake.

## Configured inputs

| Setting | Default path | Contents |
| --- | --- | --- |
| `radiusd.radsec_cert` | `private/radsec.tls.crt` | Server leaf certificate; may include intermediates |
| `radiusd.radsec_key` | `private/radsec.tls.key` | Private key matching the leaf |
| `radiusd.radsec_ca_cert` | `private/ca.crt` | CA bundle used to verify client certificates; a missing file means no client CA is configured |

The paths resolve through `GetRadsecCertPath` / `GetRadsecKeyPath` /
`GetRadsecCaCertPath`, so relative values are joined onto `system.workdir`.

## Reload semantics

- **Startup is fail-closed.** The listener is not created when the key pair
  cannot be loaded, the leaf is already expired, or a present CA bundle contains
  no PEM certificates.
- **Unchanged files cost three `stat` calls.** A handshake reuses the current
  snapshot without parsing any PEM.
- **A detected change is parsed and validated before it is used:** the key must
  match the leaf, the leaf must not be expired, and the CA bundle must contain at
  least one certificate. Only a fully valid snapshot replaces the current one.
- **A failed reload keeps the last known-good snapshot** and logs the reason;
  the next handshake retries. A truncated write therefore degrades to "still
  serving the old certificate", never to "no certificate" or "empty trust store".
- **Change detection uses modification time and size.** Install new material by
  writing a temporary file and renaming it over the configured path; editing a
  file in place can be seen half-written and then retried on the next handshake.
- **Private key custody:** a key readable by group or other is logged as a
  warning on Unix. Windows ACLs are not inspected; restrict the file through the
  deployment directory policy.

Every successful load logs the identity that will be presented:

```text
radius: radsec tls material loaded: subject="CN=radsec.example" serial=1234 not_after=2027-10-28T00:00:00Z previous_serial=1233
```

## Rotating the server certificate

1. Back up the current `radsec.tls.crt` and `radsec.tls.key`.
2. Prepare the new leaf. Clients must trust its issuer: keep the same CA when
   possible. If the CA itself changes, distribute the new CA to clients before
   this step.
3. Install the pair with atomic renames so the certificate and key always match:

   ```sh
   install -m 0600 new.crt "$WORKDIR/private/radsec.tls.crt.new"
   install -m 0600 new.key "$WORKDIR/private/radsec.tls.key.new"
   mv "$WORKDIR/private/radsec.tls.crt.new" "$WORKDIR/private/radsec.tls.crt"
   mv "$WORKDIR/private/radsec.tls.key.new" "$WORKDIR/private/radsec.tls.key"
   ```

4. No restart is needed. The next TLS handshake presents the new leaf; the log
   line above records the new serial and the previous one.

## Rotating the client CA bundle

The server verifies client certificates against `radsec_ca_cert`. Replace it in
two steps so no client loses access:

1. Write the **old and new CA** PEM blocks into the bundle. After the next
   handshake, client certificates from either CA are accepted.
2. Migrate clients to the new CA, then drop the old block from the bundle. The
   removal also takes effect on the next handshake.

## Rollback

Restore the backed-up files with the same atomic rename. The next handshake
reloads them, so a rollback needs no restart. If a rollback write is interrupted,
the server keeps the last known-good certificate and logs the reload error;
verify the log before declaring the rollback finished.

## EAP-TLS / PEAP / EAP-TTLS

These methods take their material from the managed certificate store
(`sys_cert`), not from the RadSec file paths:

- Select a different certificate with `radius.EapTlsServerCert` /
  `radius.EapTlsClientCa` on the system config page.
- Re-importing a certificate under the selected name replaces the material.
- Either change is picked up on the next handshake; there is no cached TLS
  configuration to invalidate.
- Rollback is switching the setting back to the previous certificate name.

## Observing a handshake

```sh
openssl s_client -connect 127.0.0.1:2083 \
  -CAfile new-ca.crt -cert client.crt -key client.key \
  -tls1_2 -brief </dev/null
```

A successful rotation shows `Verification: OK` and the peer's new subject/serial.
Cross-check the serial against the `radsec tls material loaded` log line. A
`certificate verify failed` result means the client does not trust the new leaf
(or its sender does not trust the new client CA) yet — finish the overlap step
before removing the old trust anchor.

## Evidence

| Test | Claim |
| --- | --- |
| `TestRadsecTLSFiles_RotatesWithoutRestart` | A running listener presents serial `1002` after the files are replaced and `1001` again after a rollback |
| `TestRadsecTLSFiles_KeepsLastGoodOnInvalidReload` | A truncated certificate and a CA bundle with no PEM keep the last known-good snapshot serving |
| `TestRadsecTLSFiles_RejectsExpiredCertificate` | An expired leaf is refused at startup |
| `TestRadsecTLSFiles_KeepsLastGoodOnExpiredReload` | Rotating to an already expired leaf is refused; the valid certificate keeps serving |
| `TestRadsecTLSFiles_ClientCAAOverlap` | An old+new CA bundle accepts client certificates from both CAs, including while the bundle is rewritten under concurrent handshakes |
| `TestSettingsTLSConfigProvider_RotatesBetweenHandshakes` | EAP material switches serial `101` → `202` → `101` between provider calls, with no caching |

Verification commands:

```sh
go test ./internal/radiusd/ -run 'TestRadsecTLSFiles' -count=1
go test ./internal/radiusd/plugins/eap/handlers/ -run TestSettingsTLSConfigProvider_RotatesBetweenHandshakes -count=1
```

## Known limits

- Client certificates are still optional in the RadSec TLS policy
  (`tls.VerifyClientCertIfGiven`), so a client that presents none is not
  rejected. Untrusted-certificate rejection is a separate regression
  (ENG-009) and is not changed here.
- No OCSP or CRL checking: a revoked certificate keeps working until it is
  replaced or its CA is removed from the bundle.
- Change detection is modification time plus size, so an in-place rewrite that
  preserves both is not detected. Always install through a rename.
- The handshake evidence comes from an in-process TLS listener on loopback.
  A pilot against a real NAS/RadSec client, with its own trust store, remains an
  external step.
