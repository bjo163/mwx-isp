# Installer, update, and release verification — 2026-10-27

This report covers audit card H14 (TR-F020 / TR-F022). It records what was
verified locally against the scripts and published artifacts, which defects were
fixed, and which rows of the install/update matrix remain environment-bound and
were **not executed**. No production or customer VPS was touched.

## Environment

- Development host: Windows 10/11 with Docker Desktop, Git Bash; Go 1.27.0,
  Python 3.14.7.
- Source baseline: `dev` after merging `origin/dev` (release `v0.13.0`, tag
  commit `225e032`); H01/H05/H06 commits included.
- Published artifact inspected: GitHub release `v0.13.0` and
  `ghcr.io/bjo163/mwx-isp:0.13.0`.
- Not used: production VPS, provider DNS, Let's Encrypt, customer volumes.

## Matrix

| Scenario | Code path inspected | Verification status | Recovery behavior |
| --- | --- | --- | --- |
| Fresh install on a clean host | `scripts/vps-install.sh`: host preflight, Docker install, checkout, generated `.env`, image pull, compose up, DB/app health, Caddy HTTP/HTTPS probe, systemd timers | Static review; **not executed on a clean Ubuntu/Debian VPS** | Fails before writing `.env` when host prerequisites are unsupported; otherwise `fail` exits 1 with the failing step and logs command. Success is printed only after health and endpoint probes. |
| Reinstall after data-preserving uninstall | `scripts/vps-uninstall.sh` saves `.env` to `${APP_DIR}.env.uninstalled` and keeps an empty install directory; installer restores it | Defect reproduced locally, **fixed**, and exercised in a container simulation with a real `git clone` and a fake Docker CLI | The installer clones first and restores `.env` afterwards (`0600 root:root`); both secrets survived and the install-state marker was written for a retry. |
| Deleted/unusable caller cwd | Installer, repair, and uninstaller start with `cd /`; uninstaller keeps the empty install directory | Static review | Shell keeps a valid cwd; reinstall can reuse the empty directory. |
| Registry/image offline | Fresh path runs `docker compose pull || fail` before starting the stack; update path pulls before stopping the app | Fresh-path failure exercised in the reinstall simulation with a simulated registry outage; no real outage executed | Fresh install exits 1 before any container starts (the simulation stopped at this point). Update calls `rollback` (source reset, old PostgreSQL/app images tagged `:rollback`, restore instructions). |
| Port conflict on 80/443/1816/1812/1813/2083 | Compose up and Caddy start failures carry explicit port-conflict and `docker compose logs` guidance | Static review; no real port conflict executed | Non-zero exit; rerun after freeing the port resumes the installation (install-state marker). |
| Update interrupted mid-run | `scripts/vps-update.sh` is idempotent per git revision/content image ID; pre-update backup; rollback tags; install-state marker covers first install | **New health guard exercised in a container simulation** (fake git/Docker): healthy → exit 0, unhealthy → exit 1 with recovery text | Rerun resumes from the current step. If the source/images already match but the app is unhealthy, the update exits 1 with inspection, container-retry, and backup-restore commands instead of reporting success. |
| Image identity/digest | `docker-compose.yml` uses floating `latest`/patch tags; backup metadata records `app_image_id` (content ID), tag, and release ref; restore requires the recorded image | `v0.13.0` image digest and labels inspected | Digest pinning is **not** used; updates compare image content IDs and the backup set records the matching image ID. Pin a digest through `MWX_ISP_IMAGE` if an operator needs immutable deployment. |
| App/frontend/binary version consistency | Release workflow injects the tag into binaries; Docker build previously did not inject version/commit | Released Windows binary verified; **container metadata defect reproduced and fixed** | From the next published image, `mwx-isp -v` and OCI version/revision labels match the release tag and checked-out commit. |
| Backup and rollback | Pre-update `backup-db.sh`, `restore-db.sh`, image `:rollback` tags, forward-only schema migration | Backup/restore drill already executed for H06; update rollback reviewed statically | Automatic rollback covers source and both images; database schema is not reversed automatically, so an incompatible schema requires restoring the recorded backup set explicitly. |
| Exit codes | `set -Eeuo pipefail`; `fail` uses 1, usage errors use 2; lock contention in the updater returns 0 | Installer pull failure returned 1; update guard returned 0 (healthy) and 1 (unhealthy) in the simulations | Update returns 0 when another install/update/restore holds the lock (timer-friendly no-op); operator-initiated failures return non-zero with recovery text. |

## Defects found and fixed

### 1. Reinstall could never clone after a data-preserving uninstall

The uninstaller preserves `.env` as `${APP_DIR}.env.uninstalled` and leaves an
empty install directory. The installer restored `.env` **before** cloning, so
`git clone` refused the now non-empty destination:

```text
fatal: destination path '.../dst' already exists and is not an empty directory.
```

Reproduced locally with an empty repository and a directory containing only
`.env` (`git clone` exit 128). This broke the documented reinstall path in the
operations guide.

Fix: `scripts/vps-install.sh` now restores the preserved configuration through
`restore_preserved_env()` **after** the checkout exists. The existing-checkout
branch restores it before Compose validation, preserving the previous behavior.

### 2. "Already up to date" hid an unhealthy app

An update interrupted during the health wait can leave the new source and image
in place while the app is unhealthy. On the next run the updater compared git
revision and image IDs, printed "already up to date", and exited 0 without
checking the container. The timer therefore reported success for a broken
deployment.

Fix: `scripts/vps-update.sh` now waits for `healthy` (or `running` for an image
without a healthcheck) before declaring the installation current. Otherwise it
exits 1 and prints inspection, app-container retry, and backup-restore commands.

### 3. Container version metadata did not match the release

The `v0.13.0` image had OCI labels for version and revision, but the binary
inside reported build defaults, and the revision label pointed at the promotion
merge commit rather than the tag commit:

```text
$ docker run --rm --entrypoint /usr/local/bin/mwx-isp ghcr.io/bjo163/mwx-isp:0.13.0 -v
MWX-ISP develop
Build Time: unknown
Git Commit: unknown

$ docker image inspect ghcr.io/bjo163/mwx-isp:0.13.0 \
    --format '{{index .Config.Labels "org.opencontainers.image.revision"}}'
733ec045c2400133730885b095a6f6acec7c9d4a   # promotion merge commit
```

The release binary itself was correct (`MWX-ISP v0.13.0`, commit `225e032`), so
only the container was inconsistent.

Fix:

- `Dockerfile` accepts `MWX_ISP_VERSION`, `MWX_ISP_COMMIT`, and
  `MWX_ISP_BUILD_TIME`, injects them through `-ldflags`, and uses the same
  values in OCI labels.
- `docker-publish.yml` resolves the checked-out tag commit and build time, then
  passes them as build args and labels. `revision` no longer uses the caller's
  `github.sha`.
- `release-publish.yml` runs the Linux release binary and fails the release if
  its `-v` output does not report the tag version and short commit.

The same quoting used by the Dockerfile was exercised locally in a minimal Go
container and printed the injected values, rather than the `develop` defaults.

## Local checks executed

| Check | Result |
| --- | --- |
| `bash -n scripts/*.sh` | All shell scripts parse successfully. |
| `python -m unittest discover -s scripts -p 'test_*.py' -v` | 10 tests pass (release SemVer/changelog and release-to-dev sync). |
| `actionlint v1.7.12 -shellcheck=` on all 12 workflow files | Exit 0. |
| Docker `-ldflags` injection simulation on `golang:1.27.3-bookworm` | Printed `MWX-ISP v0.13.0-h14test`, commit `225e032`, build time `2026-10-27T01:52:30Z`. |
| `gh release view v0.13.0` | Release contains binaries for Linux amd64/arm64/armv7/riscv64/loong64, macOS amd64/arm64, Windows amd64/arm64, plus `checksums.txt`. |
| Published Windows binary `mwx-isp_windows_amd64.exe -v` | `MWX-ISP v0.13.0`, build time `2026-10-27_01:52:30`, commit `225e032`. |
| `ghcr.io/bjo163/mwx-isp:0.13.0` image | Digest `sha256:0762929a94094925261cbf852ef18832482dae626f3b388de80e8900d8e7f9fc`; label `version=0.13.0`; revision bug and binary defaults reproduced as described above. |
| Installed-workflow run history | Auto-release for the `v0.13.0` promotion completed successfully; container and binary assets are present. |

## Changes in this audit

- `scripts/vps-install.sh` — restore the uninstaller `.env` only after cloning.
- `scripts/vps-update.sh` — verify app health before reporting "already up to
  date"; give precise recovery commands otherwise.
- `Dockerfile` — inject version, commit, and build time; mirror them in labels.
- `.github/workflows/docker-publish.yml` — pass build metadata and use the
  checked-out commit for the revision label.
- `.github/workflows/release-publish.yml` — fail a release whose Linux binary
  does not report the tag version and commit.
- `docs-site/src/en/quickstart.md`, `docs-site/src/id/quickstart.md`,
  `docs-site/src/en/ops-guide.md`, `docs-site/src/id/ops-guide.md`,
  `docs-site/src/en/agent-guide.md` — document version verification, the fixed
  reinstall flow, the backup schema `9.3`, and the two release-publish triggers.
- No schema change, migration, or deploy-script interface change.

## Verify the fixes

```bash
# Static gates
for f in scripts/*.sh; do bash -n "$f"; done
(cd scripts && python -m unittest discover -s . -p 'test_*.py' -v)
actionlint -shellcheck= .github/workflows/*.yml

# After the next release is published
docker image inspect ghcr.io/bjo163/mwx-isp:<version> \
  --format 'version={{index .Config.Labels "org.opencontainers.image.version"}} revision={{index .Config.Labels "org.opencontainers.image.revision"}}'
docker run --rm --entrypoint /usr/local/bin/mwx-isp ghcr.io/bjo163/mwx-isp:<version> -v
```

The `-v` output should report `MWX-ISP v<version>` and the tag commit; the
labels should report `<version>` and the same commit.

## Not executed — explicitly not claimed

- Fresh installation or upgrade on a real clean Ubuntu/Debian VPS with root,
  systemd, provider DNS, Let's Encrypt, and real image pulls.
- Upgrade from each historical version; only the `v0.13.0` artifacts and the
  current scripts were inspected.
- A real registry outage, a real host port conflict, or systemd timer execution.
- Host/browser compatibility matrix.
- Full VPS backup-set restore and production RTO (see the H06 drill for the
  isolated PostgreSQL portion).
- Image digest pinning and automatic database-schema rollback. Floating tags
  (`latest`, PostgreSQL patch tag, `caddy:2-alpine`) remain intentional; the
  backup set records the image content ID used at backup time.
- An update interrupted after the PostgreSQL image has already advanced may
  overwrite the `mwx-isp-postgres:rollback` tag on the next attempt. The
  pre-update backup set remains the authoritative recovery point.

## Related evidence

- [Backup and recovery drill](backup-restore-drill-2026-10-27.md)
- [Backup and recovery guide](backup-and-recovery.md)
- [Roadmap M18](roadmap.md)
