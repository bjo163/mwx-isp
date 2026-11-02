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
| 1 | [EAP-PWD (RFC 5931) M12.2–M12.5](https://github.com/bjo163/mwx-isp/issues/129) | M12.2–M12.5 | `LOKAL` |
| 2 | [RADSEC production certificate missing](https://github.com/bjo163/mwx-isp/issues/130) | — | `ASET` |

Lihat [open issues](https://github.com/bjo163/mwx-isp/issues) untuk daftar lengkap.

## Status rilis

- Rilis terakhir **v0.23.0** — `main` = `bbce8f5`, tag `v0.23.0` memicu `release.yml`: GitHub Release 9 binary + `checksums.txt`, image `0.23.0` + `v0.23.0` + `latest` pada digest `sha256:c10e90ab…` (`linux/amd64|arm64|arm/v7`). Produksi e-isp.id sudah berjalan di image itu.
- Jalur rilis: tidak ada bot bump-otomatis. `VERSION` + `CHANGELOG.md` dinaikkan manual, tag `v*` memicu `release.yml`.
- **Merge ke `main` tidak men-deploy apa pun.** Repo tidak punya webhook (`gh api repos/bjo163/mwx-isp/hooks` = `[]`) dan queue deploy Coolify kosong setelah merge hari ini. Produksi berubah hanya jika `release.yml` menerbitkan image `:latest` lalu deploy Coolify dipicu manual. `docker-compose.coolify.yml` menarik image dengan `pull_policy: always`.
- RADSEC `2083` belum melayani di produksi: `open /var/toughradius/private/radsec.tls.crt: no such file or directory`.

## Aturan repo

- Hanya branch `dev` dan `main` yang aktif (ruleset).
- Push hanya ke `dev`. `main` dilindungi (classic branch protection): required checks = **nama job CI** `Test`, `Frontend Build`, `Build (linux, amd64)` (app `github-actions`) dengan `strict: false`, wajib PR (0 approval), linear history, conversation resolution selesai, no force push, no delete, `enforce_admins: false`.
- CI hijau di `dev` → `auto-merge.yml` membuka (atau memakai kembali) PR `dev → main` dan mempromosikannya dengan **`gh pr merge --squash`, tanpa `--admin` dan tanpa langkah sinkronisasi**. Yang dulu membuatnya macet, semuanya dibereskan 2026-11-02: required check bernama `CI` adalah konteks yang tidak pernah diposting job mana pun (job Actions memakai **nama job**, dan `Build (...)` lahir dari matrix), sehingga promosi selalu tertolak bagi token Actions yang memang bukan admin; dan langkah "bawa `main` ke `dev`" tidak bisa dipakai karena push yang dilakukan `GITHUB_TOKEN` **tidak memicu CI**, jadi rantai `workflow_run` putus. Karena itu `strict` dimatikan. Gerbangnya tetap sama: job ini hanya jalan untuk run CI hijau pada SHA yang dipromosikan, dan SHA itu wajib head dari PR.
- `main` wajib linear history, jadi merge commit ditolak dan `--rebase` menolak PR yang memuat merge commit — promosi dengan squash juga berarti `dev` tidak pernah benar-benar "membawa" `main`, jadi `dev` selalu tampak tertinggal dan itu memang disengaja.
- File workflow yang dieksekusi adalah salinan di `main`, jadi perbaikan `auto-merge.yml` baru berlaku pada ronde berikutnya; run hasil `workflow_run` menampilkan `headSha` = ujung `main`, bukan SHA `dev` — jangan memfilter run dari kolom itu.
- Tag `v*` di `main` → `release.yml` otomatis rilis.

---

*Last updated: 2026-11-02*