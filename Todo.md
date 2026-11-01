# MWX-ISP — Todo

Satu-satunya antrean kerja proyek ini. Yang di bawah bukan changelog: baris selesai hanya menyimpan di mana buktinya, supaya tidak perlu dicari ulang.

**Aturan**

1. Ambil item paling atas dari antrean aktif. Item tanpa tanda `LOKAL` tidak bisa dieksekusi dari repo ini.
2. Selesai = centang di sini, lalu tutup issue-nya. GitHub Issues cuma cermin, bukan sumber status.
3. Item `ASET` / `RISET` tidak boleh ditandai selesai tanpa bukti eksekusi nyata (perangkat, akun, angka).
4. Alur kerja: commit ke `dev` → PR `dev` → `main` → CI hijau → merge → tag `vX.Y.Z` (`main` tidak protected, tapi jangan push langsung ke `main`).

Penanda: `LOKAL` = cukup repo · `ASET` = butuh perangkat/host/akun nyata · `RISET` = kajian, bukan kode · `BLOCKED` = menunggu bukti eksternal.

---

## Status baseline (2026-11-01)

- Rilis terakhir **v0.22.1** (`main`, tag ada, GitHub Release lengkap: 9 binary + `checksums.txt` + image multi-arch termasuk `linux/arm/v7`).
- Repo baru saja dirampingkan (493 file: `docs/`, `docs-site/`, `scripts/`, `tools/`, `.agents/`, 10 workflow). Yang masih dibutuhkan produksi dan sengaja dipulangkan: `Dockerfile`, `docker-compose.yml`, `docker-compose.coolify.yml`, `.dockerignore`, `.golangci.yml`, `config/`, dua RFC yang dibaca tes TEAP.
- Jalur rilis berubah: tidak ada lagi bot bump-otomatis. `VERSION` + `CHANGELOG.md` dinaikkan manual, tag `v*` memicu `.github/workflows/release.yml`.
- **v0.23.0 menunggu persetujuan owner** — belum di-push, belum di-tag.

## Antrean aktif

| # | Item | Kode | Tipe | Kondisi |
| --- | --- | --- | --- | --- |
| 1 | Accounting-Stop yang diproses lebih dulu dari task Start-nya hilang tanpa jejak | TR-F010 | `LOKAL` | `internal/radiusd/radius_acct.go:103` mengirim Accounting-Response **sebelum** `submitAcctTask` (`:133`), dan worker pool tidak menjamin urutan per `Acct-Session-Id`; `start_handler.go:73` menulis `radius_online` sebelum `radius_accounting` (`:97`), jadi Stop di celah itu mengembalikan `no records found` dan trafik sesi tidak terbukukan. Flake CI `TestISPBillingLifecycleThroughRadius` sudah diredam dengan sync point di tes (`test/integration/isp_lifecycle_test.go`); perbaikan produknya butuh keputusan desain (satu transaksi di Start, atau Stop handler menunggu/meng-upsert, atau ordering per sesi) dan belum diambil |
| 2 | EAP-PWD (RFC 5931) | M12.2–M12.5 | `LOKAL` | demand-gated, kode belum ada. Teks spesifikasinya tidak ikut dipulangkan saat perampingan — ambil ulang dari IETF sebelum mulai |

## Sisa backlog audit

Butuh aset nyata (14 · `ASET`): `ENG-002` supplicant EAP-TLS 1.2/1.3 · `ENG-013` pilot CoA/Disconnect NAS nyata · `ENG-041` instalasi fresh VPS bersih · `ENG-042` simulasi kegagalan registry saat upgrade · `ENG-043` drill restore seluruh volume VPS · `ENG-045` RTO/RPO operator · `ENG-046` akses localhost/SSH tunnel tanpa DNS · `ENG-047` audit health & expiry sertifikat deployment · `ENG-048` UAT WhatsApp pairing ke nomor uji · `ENG-049` UAT SNMPv3 authPriv perangkat nyata · `ENG-050` matriks NAS/firmware RADIUS+CoA · `ENG-014` prioritas vendor VSA dari NAS riil · `ENG-020` restore lintas tenant setelah perubahan skema · `ENG-044` studi WAL PITR PostgreSQL

Riset ekonomi (10 · `RISET`): `ECO-001` willingness-to-pay · `ECO-002` waktu tugas operator sebelum/sesudah · `ECO-003` model TCO deployment · `ECO-004` lisensi managed support vs self-hosted · `ECO-005` desain paket dukungan & SLA · `ECO-007` kendali biaya telemetri WhatsApp · `ECO-008` alokasi biaya per NAS/subscriber · `ECO-010` risiko churn dari sinyal operasional · `ECO-011` strategi distribusi community/open-source · `ECO-012` partner interoperability kit

Riset lanjutan, research-only (10 · `RISET`): `FUT-003` ringkasan insiden berbukti · `FUT-004` deteksi anomali latency · `FUT-005` graph hipotesis penyebab · `FUT-008` preview dampak CoA · `FUT-012` eksperimen adversarial EAP fragment scheduler · `FUT-013` snapshot NOC offline read-only · `FUT-014` field diagnostics mini-mode · `FUT-019` edge collector polling read-only · `FUT-021` knowledge base kompatibilitas firmware · `FUT-024` prediksi kapasitas dari SNMP nyata

## Selesai — bukti di mana

| Milestone | Kode / bukti |
| --- | --- |
| M11 TEAP `M11.1`–`M11.10` | `internal/radiusd/plugins/eap/{teapmsg,teaptlv,teapkeys,teapinner,teapbind}`, `handlers/teap_phase2.go`; acceptance `test/integration/teap_acceptance_test.go`; spec di `docs/rfcs/rfc9930-teap-version-1.txt` + `rfc9427-tls-based-eap-types-tls13.txt`. Sisa hanya interoperabilitas peer nyata (`ENG-002`) |
| M5 vendor VSA (14 paket) | `internal/radiusd/vendors/{alcatel,aruba,cisco,f5,h3c,hillstone,huawei,ikuai,juniper,microsoft,mikrotik,pfsense,radback,zte}` + enhancer per vendor, registrasi `plugins/init.go` |
| M6 observability | `internal/webserver/metrics.go` + `metrics_test.go`, rute di `webserver/server.go`, inkremen di `radiusd/acct_plugin_runner.go`; token `TOUGHRADIUS_METRICS_TOKEN`, default nonaktif |
| M14/M14.5 LDAP robustness | `internal/ldapauth/ldapauth.go` (`dialFunc` ctx, `armTimeout`), `radiusd/ldap_backend.go` (`VerifyCleartext` memasang tenggat). Pooling **sengaja ditolak**: Bind melekatkan koneksi ke identitas subscriber; tabungan ~0,5 ms dial loopback tidak sebanding risiko bocor identitas. Profil `ldaps://` lintas jaringan belum diukur |
| M19 tenant brand / M20 invoice branding | `domain.InvoiceDocumentSnapshot`, `internal/invoicedoc/render.go`, `/tenant/invoice`, `InvoiceStudioPage.tsx`; acceptance `test/integration/{tenant_brand_experience_test.go,invoice_branding_test.go}`, `web/e2e/tenant-brand-experience.spec.ts`; merge #116 |
| Billing: invoice yatim tidak menghentikan penegakan | `internal/billing/service.go` (`SuspendOverdueSubscriptions` lewati `ErrRecordNotFound`, `RecordPayment` tetap membukukan uang) + series `billing_invoice_without_subscription_total` |
| Build 32-bit terjaring sebelum merge | `pfsense_enhancer.go` / `cmd/radtest` (`toUint32FromInt` bandingkan via `uint64`), `pkg/validutil/validutil_test.go`; penyebabnya matrix `Build` cuma amd64/arm64 sedangkan image menambahkan `arm/v7` |
| FUP fixture tidak lagi kalah di dua jam pertama tiap bulan | `test/integration/fup_tenant_test.go` (`closedStart`/`closedStop` clamp ke `monthStart`), `internal/app/jobs_test.go` |
| EN/ID untuk halaman custom + branding | `web/src/i18n/{en-US,id-ID}.ts`, `TestSaveSettings_UnknownAndInvalidValuesRejected` |

## Residual rilis

- Tag `v0.18.0`–`v0.19.8` dan `v0.22.0` ada tanpa GitHub Release (`v0.22.0` gagal di build `linux/arm/v7`; perbaikannya keluar di `v0.22.1`). Backfill sekarang berarti menjalankan `release.yml` dengan `workflow_dispatch` dan **butuh persetujuan eksplisit**.
- `workflow_dispatch` di `release.yml` menghasilkan **draft** release + image bertag `draft-vX.Y.Z`; itu jalur kering untuk membuktikan pipeline tanpa menyangkutkannya ke `:latest`.
- Yang tidak akan pernah diklaim dari repo ini: instalasi/upgrade VPS bersih, kegagalan registry/port nyata, timer, RTO produksi.
