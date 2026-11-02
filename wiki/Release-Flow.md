# Release Flow

## Versioning

- `VERSION` file berisi versi (contoh: `0.23.0`)
- `CHANGELOG.md` berisi changelog per rilis
- Tag `v*` di `main` memicu `release.yml`

## Proses Rilis

1. Edit `VERSION` dan `CHANGELOG.md` di `dev`
2. Commit → push → CI hijau → auto-merge ke `main`
3. Tag `vX.Y.Z` di `main`
4. `release.yml` berjalan:
   - Build frontend (artifact)
   - Cross-compile 9 binary (amd64, arm64, arm/v7, riscv64, loong64, dll)
   - `sha256sum * > checksums.txt`
   - Build multi-arch image (linux/amd64, linux/arm64, linux/arm/v7)
   - Push ke GHCR (`ghcr.io/bjo163/mwx-isp`)
   - Create GitHub Release

## Image Tags

| Tag | Keterangan |
|-----|-----------|
| `0.23.0` | Major version |
| `v0.23.0` | With v prefix |
| `latest` | Latest stable |

## Cacat Dikenal

- `release.yml` dengan `workflow_dispatch` menghasilkan draft release + image `draft-vX.Y.Z`
- Tag `v0.18.0`–`v0.19.8` dan `v0.22.0` ada tanpa GitHub Release (backfill butuh persetujuan eksplisit)

---

*Last updated: 2026-11-02*
