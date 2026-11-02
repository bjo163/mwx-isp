#WYWH-ISP Home

MWX-ISP adalah server RADIUS open-source untuk ISP, enterprise, dan carrier.

## Quick Links

- [Arsitektur](Architecture) - Desain sistem dan komponen
- [Deployment](Deployment) - Panduan instalasi dan konfigurasi
- [Release Flow](Release-Flow) - Proses rilis dan versioning
- [Branching Strategy](Branching-Strategy) - Alur kerja git (dev → main)

## Active Tasks

- [#128](../../issues/128) - TR-F010: Accounting-Stop processed before Start missing trace
- [#129](../../issues/129) - EAP-PWD (RFC 5931) M12.2–M12.5
- [#130](../../issues/130) - RADSEC production certificate missing

## Current Status

- **Latest Release:** v0.23.0 ([9 binaries + checksums.txt](../../releases/tag/v0.23.0))
- **Container Image:** `ghcr.io/bjo163/mwx-isp:latest` (linux/amd64|arm64|arm/v7)
- **Production:** e-isp.id (auto-deploy via Coolify GitHub App)

## Repository Rules

- Hanya branch `dev` dan `main` yang aktif
- Push hanya ke `dev`
- CI hijau di `dev` → auto PR → auto merge ke `maig` auto merge ke `main`
- Tag `v*` di `main` → auto release (9 binary + image multi-arch)

---

*