# MWX-ISP — Todo

Satu-satunya sumber kebenaran untuk proyek ini adalah **GitHub Issues** dan **GitHub Wiki**.
File ini hanya berisi pointer ke sana, bukan task tracker.

## Wiki

- [Home](https://github.com/bjo163/mwx-isp/wiki/Home) — overview, quick links, active tasks
- [Architecture](https://github.com/bjo163/mwx-isp/wiki/Architecture) — desain sistem
- [Deployment](https://github.com/bjo163/mwx-isp/wiki/Deployment) — konfigurasi Docker/Coolify
- [Release Flow](https://github.com/bjo163/mwx-isp/wiki/Release-Flow) — proses rilis
- [Branching Strategy](https://github.com/bjo163/mwx-isp/wiki/Branching-Strategy) — alur git

## Antrean aktif (mirror dari GitHub Issues)

| # | Issue | Kode | Tipe |
| --- | --- | --- | --- |
| 1 | [Accounting-Stop processed before Start missing trace](https://github.com/bjo163/mwx-isp/issues/128) | TR-F010 | `LOKAL` |
| 2 | [EAP-PWD (RFC 5931) M12.2–M12.5](https://github.com/bjo163/mwx-isp/issues/129) | M12.2–M12.5 | `LOKAL` |
| 3 | [RADSEC production certificate missing](https://github.com/bjo163/mwx-isp/issues/130) | — | `ASET` |

Lihat [open issues](https://github.com/bjo163/mwx-isp/issues) untuk daftar lengkap.

## Status rilis

- Rilis terakhir **v0.23.0** — `main` = `bbce8f5`, tag `v0.23.0` memicu `release.yml`: GitHub Release 9 binary + `checksums.txt`, image `0.23.0` + `v0.23.0` + `latest` pada digest `sha256:c10e90ab…` (`linux/amd64|arm64|arm/v7`). Produksi e-isp.id sudah berjalan di image itu.
- Jalur rilis: tidak ada bot bump-otomatis. `VERSION` + `CHANGELOG.md` dinaikkan manual, tag `v*` memicu `release.yml`.
- Deploy otomatis dari `main` (Coolify GitHub App). `docker-compose.coolify.yml` menarik `:latest` dengan `pull_policy: always`.
- RADSEC `2083` belum melayani di produksi: `open /var/toughradius/private/radsec.tls.crt: no such file or directory`.

## Aturan repo

- Hanya branch `dev` dan `main` yang aktif (ruleset).
- Push hanya ke `dev`. `main` dilindungi: CI wajib hijau, linear history, no force push.
- CI hijau di `dev` → `auto-merge.yml` buka & merge PR ke `main` otomatis.
- Tag `v*` di `main` → `release.yml` otomatis rilis.

---

*Last updated: 2026-11-02*