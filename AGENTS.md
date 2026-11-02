# MWX-ISP — AGENTS.md

Satu-satunya sumber kebenaran untuk proyek ini adalah **GitHub Issues** dan **GitHub Wiki**.
File-file di repo ini (kecuali `README.md`, `VERSION`, `CHANGELOG.md`) adalah **hanya referensi**.

## Wiki (sumber kebenaran)

- [Home](https://github.com/bjo163/mwx-isp/wiki/Home) — overview, quick links, active tasks
- [Architecture](https://github.com/bjo163/mwx-isp/wiki/Architecture) — desain sistem
- [Deployment](https://github.com/bjo163/mwx-isp/wiki/Deployment) — konfigurasi Docker/Coolify
- [Release Flow](https://github.com/bjo163/mwx-isp/wiki/Release-Flow) — proses rilis
- [Branching Strategy](https://github.com/bjo163/mwx-isp/wiki/Branching-Strategy) — alur git

## Issues (task tracker)

Setiap task aktif ada di GitHub Issues. Lihat [open issues](https://github.com/bjo163/mwx-isp/issues).
Todo.md di repo ini hanya berisi **pointer ke Issues**, bukan task tracker.

## Aturan

1. **Push hanya ke `dev`** — branch `main` dilindungi (CI wajib, linear history, no force push)
2. **CI hijau di `dev` → auto-merge ke `main`** — workflow `auto-merge.yml`
3. **Tag `v*` di `main` → auto-release** — `release.yml` menghasilkan 9 binary + checksums.txt + image multi-arch
4. **Tidak ada branch selain `dev` dan `main`** — ruleset menolak creation/update/deletion branch lain

## Komit

- Commit ke `dev` → push → CI hijau → auto-merge ke `main`
- Jangan push langsung ke `main`
- Jangan buat branch baru (ditolak ruleset)

---

*Last updated: 2026-11-02*