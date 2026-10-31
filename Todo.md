# MWX-ISP — Todo

Antrean kerja tunggal untuk proyek ini. Semua pekerjaan dibaca, diurutkan, dan dicentang dari file ini. Dokumen lain tetap ada sebagai detail, bukan sebagai daftar kerja yang bersaing:

| Sumber | Perannya sekarang |
| --- | --- |
| `Todo.md` (file ini) | Satu-satunya antrean kerja + urutan prioritas |
| [`docs/roadmap.md`](docs/roadmap.md) | Definisi milestone `M1`–`M20`, peta `TR-F`, guardrail |
| [`docs/feature-checklist.md`](docs/feature-checklist.md) | Batas scope fitur dan non-goal `TR-N001`–`TR-N006` |
| [`docs/MWX-ISP-audit-backlog-2026-10-27.md`](docs/MWX-ISP-audit-backlog-2026-10-27.md) | Bukti dan detail 100 item `ENG`/`ECO`/`FUT` |
| GitHub Issues | Cermin per item; **bukan** sumber status lagi |

**Aturan kerja**

1. Ambil item dari atas daftar aktif. Item tanpa tanda `LOKAL` belum bisa dieksekusi dari repo ini.
2. Satu item selesai = centang di `Todo.md`, sinkronkan `docs/roadmap.md`, lalu tutup issue-nya. Tidak ada status ganda.
3. Item baru harus punya `TR-F` yang sudah ada di `docs/feature-checklist.md`. Kalau tidak ada, perbarui checklist dulu.
4. Pekerjaan tetap lewat branch → PR → CI → review. Tidak ada push langsung ke `main`.
5. Jangan pernah menandai item eksternal (VPS bersih, NAS nyata, supplicant nyata, nomor WhatsApp nyata, RTO produksi) sebagai selesai tanpa bukti eksekusi.

Penanda tipe: `LOKAL` = bisa dikerjakan di repo · `ASET` = butuh perangkat/host/akun nyata · `RISET` = kajian, bukan kode · `BLOCKED` = menunggu bukti eksternal.

---

## Status baseline

- `main` = `dev` = `94fa0dd` — rilis **v0.20.0**, CI hijau, aset dan image GHCR sudah diverifikasi.
- M19 fitur (brand tenant, website, katalog paket) **sudah rilis**; tersisa acceptance dan dokumentasi.
- M11 TEAP `M11.1`–`M11.10` **sudah di `main`** bersama `test/integration/teap_acceptance_test.go`; yang tersisa hanya interoperabilitas peer nyata (`ENG-002`).
- Tidak ada PR terbuka.

---

## 1. Aktif sekarang — tutup M19

| # | Item | Kode | Tipe | Sumber |
| --- | --- | --- | --- | --- |
| 1 | Acceptance dua tenant: PostgreSQL + browser E2E untuk branding/website/katalog | M19.5.1 | `LOKAL` | issue #107 |
| 2 | Handbook EN/ID, API, domain, roadmap, dan bukti rilis untuk tenant brand experience | M19.5.2 | `LOKAL` | issue #108 |

Catatan #108: `docs/tenant-brand-experience.md` sudah ada, tetapi `docs-site/src/en` dan `docs-site/src/id` **belum** menyebut Brand Studio / Website Studio / Package Studio sama sekali.

## 2. Berikutnya — M20 invoice & receipt branding (TR-F035)

Belum ada implementasi sama sekali: tidak ada issuer snapshot, template invoice/receipt, maupun jalur PDF/print di `internal/`.

| # | Item | Kode | Tipe | Sumber |
| --- | --- | --- | --- | --- |
| 3 | Snapshot identitas legal issuer saat invoice diterbitkan | M20.1 | `LOKAL` | issue #109 |
| 4 | Template invoice dan receipt per tenant | M20.2 | `LOKAL` | issue #110 |
| 5 | Preview PDF/print dengan kesetiaan historis | M20.3 | `LOKAL` | issue #111 |
| 6 | Instruksi pembayaran dan QR dari sumber terverifikasi | M20.4 | `LOKAL` | issue #112 |
| 7 | Uji riwayat PostgreSQL, isolasi tenant, dan invarian billing | M20.5 | `LOKAL` | issue #113 |

Batas keras M20: jangan mengubah total invoice, penomoran dokumen, status pembayaran, aturan reversal, atau semantik ledger.

## 3. Menunggu keputusan atau permintaan

| # | Item | Kode | Tipe | Kondisi |
| --- | --- | --- | --- | --- |
| 8 | EAP-PWD (RFC 5931) | M12.2–M12.5 | `LOKAL` | demand-gated; spesifikasi sudah tersedia (`docs/rfcs/rfc5931-eap-pwd.txt`), kode belum ada |
| 9 | Vendor VSA lanjutan (alcatel/juniper/radback/microsoft/f5/hillstone/pfSense) | M5 sisa | `LOKAL` | demand-driven; pattern parser/enhancer sudah ada |
| 10 | LDAP pooling/reconnect | M14.5 / ENG-015 | `BLOCKED` | menunggu bukti beban koneksi |
| 11 | Observability dan operasi | M6 | `LOKAL` | P3, belum dijadwalkan |

## 4. Backlog audit — 39 item sisa

### 4.1 Bisa dieksekusi lokal (4)

| ID | Item | TR-F | Catatan |
| --- | --- | --- | --- |
| ENG-055 | State loading, stale-data, retry pada OperationsPage | TR-F031 | dep `ENG-049` (SNMP nyata) untuk verifikasi akhir |
| ENG-064 | Pemeriksaan dead-letter dan alert session-action queue | TR-F010 | dep `ENG-013` (NAS nyata) |
| ENG-058 | Label confidence dan mapping pada NOC impact UI | TR-F031 | dep `ENG-055` |
| ENG-005 | Evaluasi inner EAP-TTLS CHAP dan MS-CHAP | TR-F004 | dep `ENG-002` (supplicant nyata) |

Keempatnya masih bergantung pada bukti eksternal di ujung, jadi hasil lokalnya adalah implementasi + tes, bukan klaim interoperabilitas.

### 4.2 Butuh aset nyata (15 · `ASET`)

`ENG-002` supplicant EAP-TLS 1.2/1.3 · `ENG-013` pilot CoA/Disconnect NAS nyata · `ENG-041` instalasi fresh VPS bersih · `ENG-042` simulasi kegagalan registry saat upgrade · `ENG-043` drill restore seluruh volume VPS · `ENG-045` RTO/RPO operator · `ENG-046` akses localhost/SSH tunnel tanpa DNS · `ENG-047` audit health & expiry sertifikat deployment · `ENG-048` UAT WhatsApp pairing ke nomor uji · `ENG-049` UAT SNMPv3 authPriv perangkat nyata · `ENG-050` matriks NAS/firmware RADIUS+CoA · `ENG-014` prioritas vendor VSA dari NAS riil · `ENG-020` restore lintas tenant setelah perubahan skema · `ENG-044` studi WAL PITR PostgreSQL

### 4.3 Riset ekonomi (10 · `RISET`)

`ECO-001` willingness-to-pay · `ECO-002` waktu tugas operator sebelum/sesudah · `ECO-003` model TCO deployment · `ECO-004` lisensi managed support vs self-hosted · `ECO-005` desain paket dukungan & SLA · `ECO-007` kendali biaya telemetri WhatsApp · `ECO-008` alokasi biaya per NAS/subscriber · `ECO-010` risiko churn dari sinyal operasional · `ECO-011` strategi distribusi community/open-source · `ECO-012` partner interoperability kit

### 4.4 Riset lanjutan (10 · `RISET`, research-only)

`FUT-003` ringkasan insiden berbukti · `FUT-004` deteksi anomali latency · `FUT-005` graph hipotesis penyebab · `FUT-008` preview dampak CoA · `FUT-012` eksperimen adversarial EAP fragment scheduler · `FUT-013` snapshot NOC offline read-only · `FUT-014` field diagnostics mini-mode · `FUT-019` edge collector polling read-only · `FUT-021` knowledge base kompatibilitas firmware · `FUT-024` prediksi kapasitas dari SNMP nyata

Semua `FUT` dan `ECO` berstatus research-only: tidak boleh naik ke kode produksi tanpa pembaruan `docs/feature-checklist.md` dan persetujuan scope.

## 5. Residual rilis

- Tag `v0.18.0`–`v0.19.8` ada tanpa GitHub Release (sudah tersaingi `v0.20.0`). Backfill hanya lewat `workflow_dispatch` dengan `release_tag`, dan **butuh persetujuan eksplisit**.
- Pemeriksaan operasional eksternal yang belum pernah dieksekusi dan tidak boleh diklaim: instalasi/upgrade VPS bersih, kegagalan registry/port nyata, timer, dan RTO produksi.
