# Branching Strategy

## Branches

| Branch | Tujuan | Perlindungan |
|--------|--------|-------------|
| `dev` | Development, push only | Ruleset: creation/update/deletion |
| `main` | Production | Protected: CI required, linear history, no force push |

## Alur Kerja

```
commit ke dev → push dev → CI hijau → auto-merge ke main
                                                ↓
                                         tag vX.Y.Z → auto release
```

## Ruleset

- Hanya `dev` dan `main` yang bisa dibuat/diupdate/dihapus
- Branch lain (feature, fix, dll) ditolak oleh ruleset
- `main` dilindungi: CI wajib hijau, linear history required, no force push

## Auto-Merge

- Workflow `auto-merge.yml` trigger setelah CI di `dev` selesai hijau
- Otomatis buka PR `dev` → `main` dan merge
- Tidak perlu merge manual

## Auto-Release

- Tag `v*` di `main` memicu `release.yml`
- Hasil: 9 binary + checksums.txt + image multi-arch + GitHub Release

---

*Last updated: 2026-11-02*
