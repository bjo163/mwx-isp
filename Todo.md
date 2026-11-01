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

- Rilis terakhir **v0.22.1** (PR #121 merge, tag `v0.22.1`); GitHub Release-nya terbit lengkap beserta image multi-arch termasuk `linux/arm/v7`. `main` memuat seluruh `dev`. Perubahan docs-only setelahnya tidak memicu rilis baru, sesuai kebijakan rilis.
- M19 (brand tenant, website, katalog paket) dan M20 (invoice & receipt branding) **sudah di `main`** beserta acceptance PostgreSQL/browser.
- M11 TEAP `M11.1`–`M11.10` **sudah di `main`** bersama `test/integration/teap_acceptance_test.go`; yang tersisa hanya interoperabilitas peer nyata (`ENG-002`).
- M5 vendor VSA expansion (`M5.1`–`M5.5`) **selesai dan sudah di `main`**: seluruh 14 paket dictionary vendor punya consumer, dan tidak ada enhancer vendor yang tersisa.
- M6 observability **selesai dan sudah di `main`**: counter dan gauge yang sudah dikumpulkan kini terbaca lewat `GET /metrics` (format teks Prometheus, bertoken, default nonaktif).
- M14 LDAP **selesai dan sudah di `main`** dengan tertutupnya M14.5: tenggat permintaan kini membatasi setiap round trip ke directory; connection pooling dievaluasi terhadap angka nyata dan sengaja tidak diambil.
- PR promosi **#120** (`dev` → `main`) sudah merge (31 check hijau pada SHA yang direview) dan memicu tag `v0.22.0` (tanpa rilis karena build arm/v7 gagal); **#121** adalah PR promosi berikutnya yang membawa perbaikan build 32-bit dan **berhasil menerbitkan rilis `v0.22.1`**.

---

## 1. Selesai — M19 close-out

| # | Item | Kode | Tipe | Sumber | Bukti |
| --- | --- | --- | --- | --- | --- |
| 1 | Acceptance dua tenant: PostgreSQL + browser E2E untuk branding/website/katalog | M19.5.1 | `LOKAL` | issue #107 | `test/integration/tenant_brand_experience_test.go`, `web/e2e/tenant-brand-experience.spec.ts`; merge #116 |
| 2 | Handbook EN/ID, API, domain, roadmap, dan bukti rilis untuk tenant brand experience | M19.5.2 | `LOKAL` | issue #108 | `docs-site/src/en|id/tenant-brand-experience.md`, `docs/tenant-brand-experience.md`; merge #116 |

## 2. Selesai — M20 invoice & receipt branding (TR-F035)

| # | Item | Kode | Tipe | Sumber | Bukti |
| --- | --- | --- | --- | --- | --- |
| 3 | Snapshot identitas legal issuer saat invoice diterbitkan | M20.1 | `LOKAL` | issue #109 | `domain.InvoiceDocumentSnapshot`, `billing/document.go`; merge #116 |
| 4 | Template invoice dan receipt per tenant | M20.2 | `LOKAL` | issue #110 | `domain.TenantDocumentTemplate`, `/tenant/invoice`, `InvoiceStudioPage.tsx`; merge #116 |
| 5 | Preview PDF/print dengan kesetiaan historis | M20.3 | `LOKAL` | issue #111 | `internal/invoicedoc/render.go`, snapshot-only rendering; merge #116 |
| 6 | Instruksi pembayaran dan QR dari sumber terverifikasi | M20.4 | `LOKAL` | issue #112 | template payment fields + tenant-scoped asset validation; merge #116 |
| 7 | Uji riwayat PostgreSQL, isolasi tenant, dan invarian billing | M20.5 | `LOKAL` | issue #113 | `test/integration/invoice_branding_test.go`, `adminapi/invoice_document_test.go`; merge #116 |

Batas keras M20 terpenuhi: total invoice, penomoran dokumen, status pembayaran, aturan reversal, dan semantik ledger tidak berubah; dokumen hanya membaca baris billing.

## 3. Selesai — M5 vendor VSA expansion (TR-F005)

| # | Item | Kode | Tipe | Bukti |
| --- | --- | --- | --- | --- |
| 8 | Enhancer Access-Accept untuk tujuh vendor sisa + pemulihan dictionary yang dibutuhkan | M5.5 | `LOKAL` | `enhancers/{alcatel,juniper,radback,microsoft,f5,hillstone,pfsense}_enhancer.go` + tes per vendor; `vendors/{f5,hillstone,pfsense}` hasil radius-dict-gen; dua accessor reply di `vendors/radback`; registrasi di `plugins/init.go`; `scripts/check-vendor-dict-usage.sh` melaporkan 14 paket punya consumer |

Mapping yang dipakai: `IpAddr` → AAT-PPP-Address, `GetAddrPool` → AAT-IP-Pool-Definition / Juniper-Ip-Pool-Name / IP-Address-Pool-Name / Hillstone-VPN-DHCP-Pool, `Username` → Juniper-Local-User-Name, `Vlanid2` → Juniper-VoIP-Vlan, `GetDomain` → Subscriber-Profile-Name / MS-Service-Class / F5-LTM-User-Partition / Hillstone-User-Role-Bame, `GetUpRate`/`GetDownRate` → pfSense-Bandwidth-Max-Up/Down. Sisa M5 hanya vendor baru yang demand-driven, bukan item terjadwal.

## 4. Selesai — M6 observability (TR-F015)

| # | Item | Kode | Tipe | Bukti |
| --- | --- | --- | --- | --- |
| 9 | Ekspos counter RADIUS + gauge sistem lewat `GET /metrics` (Prometheus text) dan aktifkan counter `radus_online`/`radus_offline` | M6.1 | `LOKAL` | `internal/webserver/metrics.go` + `metrics_test.go`; registrasi rute di `internal/webserver/server.go`; inkremen di `internal/radiusd/acct_plugin_runner.go` + `TestHandleAccountingStartAndStopCounters`; env `TOUGHRADIUS_METRICS_TOKEN` (`MWX_ISP_METRICS_TOKEN` di compose) di `.env.example`, `.env.vps.example`, `env.coolify`, `docker-compose*.yml`; handbook EN/ID `ops-guide` + `docs/public-endpoint-hardening.md` |

Sisanya sudah ada dari M3/M18 (agregasi dashboard, bucketing timezone, ringkasan IPv6), jadi M6 tidak menambah metrik baru — hanya jalur baca yang sebelumnya tidak ada.

## 5. Selesai — M14.5 LDAP connection robustness (TR-F025) + tiga putaran gerbang PR #120 dan pemulihan rilis v0.22.0

| # | Item | Kode | Tipe | Bukti |
| --- | --- | --- | --- | --- |
| 10 | Bukti beban koneksi LDAP lalu keputusan desain: propagasi request context, pooling ditolak dengan pemicu buka-kembali tercatat | M14.5 / ENG-015 | `LOKAL` | `tools/ldapload` (`main.go` + `main_test.go`); `docs/ldap-connection-load-evidence-2026-10-31.md`; `internal/ldapauth/ldapauth.go` (`dialFunc` menerima ctx, `armTimeout`, cek ctx sebelum tiap round trip) + `TestVerify_ContextHonoredAfterDial`; `VerifyCleartext` di `internal/radiusd/ldap_backend.go` memasang tenggat `ldap.Timeout` saat pemanggil tidak bawa deadline + `TestLDAPBackend_VerifyCleartextBoundsSilentDirectory`; semantik `ldap.Timeout` di `internal/app/config_schemas.json` + `web/src/i18n/en-US.ts`; handbook EN/ID `auth-ldap` |
| 11 | Invoice yatim (tanpa baris subscription) dilewati oleh penegakan tagihan, dicatat di log + metrik, dan pembayaran tetap dibukukan | TR-F028 / ENG-065 | `LOKAL` | `internal/billing/service.go` (`SuspendOverdueSubscriptions` melewati `gorm.ErrRecordNotFound`; `RecordPayment` tetap membukukan uang dan hanya melewatkan reaktivasi; `OrphanInvoices` + warn log) + `TestSuspendOverdueContinuesPastADanglingInvoice` / `TestRecordPaymentBooksAnInvoiceWithoutSubscription`; series `billing_invoice_without_subscription_total` di `internal/app/metrics.go` + handbook EN/ID `ops-guide` |
| 12 | Fixture FUP (integrasi dan unit) dijepit ke awal periode kalender supaya tidak lagi kalah di dua jam pertama tiap bulan | TR-F025 gerbang CI | `LOKAL` | `test/integration/fup_tenant_test.go` (`closedStart` / `closedStop` di-clamp ke `monthStart`); `internal/app/jobs_test.go` (`seedFUPAccounting` clamp `AcctStartTime`/`AcctStopTime`) — terkonfirmasi merah lokal pada 01 Nov 00:40 sebelum clamp, hijau sesudahnya |
| 13 | Fixture nomor dokumen dibersihkan setelah tes agar invoice tanpa subscription tidak menetap di dataset bersama | gerbang CI putaran kedua | `LOKAL` | `test/integration/invoice_numbering_test.go` (`t.Cleanup` hapus invoice, customer, dan sequence) |
| 14 | Counter job latar benar-benar terbit di `GET /metrics`, bukan hanya tersedia di in-process snapshot | TR-F015 (M6) gerbang review putaran ketiga | `LOKAL` | `internal/webserver/metrics.go` (`prometheusMetricsText` menggabungkan `app.GetSystemMetrics()` ke blok counter) + assertion tiga nama series di `internal/webserver/metrics_test.go` — sebelumnya `GetSystemMetrics()` tidak punya caller produksi sama sekali, jadi paragraf handbook dan komentar `OrphanInvoices` mengklaim jalur baca yang belum ada |
| 15 | Tes lifecycle ISP menunggu baris accounting Start sebelum mengirim Stop | TR-F010 gerbang CI | `LOKAL` | `test/integration/isp_lifecycle_test.go` (`waitForAccounting` sebelum `acctStop`) — sama seperti yang sudah dilakukan `test/integration/ipv6_test.go`; kegagalan CI `TestISPBillingLifecycleThroughRadius` (23:48:01.621 ERROR `no records found with acct_session_id`, 1 dari 2 run pada head yang sama) muncul dari Stop yang diproses duluan |
| 18 | Build 32-bit diperbaiki supaya rilis multi-arch bisa terbit, dan CI memeriksa `linux/arm` sebelum merge | TR-F022 / gerbang rilis v0.22.0 | `LOKAL` | `internal/radiusd/plugins/auth/enhancers/pfsense_enhancer.go` dan `cmd/radtest/radtest.go` (`toUint32FromInt`) membandingkan lewat `uint64`, plus `pkg/validutil/validutil_test.go` (`int64(628123456789)`) — ketiganya `math.MaxUint32 vs int` yang hanya overflow saat `int` 32 bit; langkah "Check 32-bit build" di job `Test` `.github/workflows/ci.yml` menjalankan `GOOS=linux GOARCH=arm go vet -tags=integration,eap_accept ./...` (59 s di CI, run 41307890748). `go vet`, bukan `go build`, karena file tes ikut dikompilasi — dan itu satu-satunya cara baris `validutil_test.go` di atas ikut terjaring. Bukti merah: log `Publish GHCR image` run 41258101735, `linux/amd64->arm/v7`, `pfsense_enhancer.go:58:57: math.MaxUint32 (untyped int constant 4294967295) overflows int` |

Satu nomor baris dipakai ulang: tabel §6 sebelumnya memakai `# 11` untuk EAP-PWD; nomor itu sekarang `# 17` di bawah.

Dua temuan itu muncul dari job `Integration (PostgreSQL/OpenLDAP)` yang merah tepat di jam 00:01 (+08) saat PR #120 dibuka, dan keduanya direproduksi lokal dengan `TZ=Asia/Makassar` pada suite yang sama. Yang pertama bukan urusan tanggal: satu invoice yang tidak punya baris subscription membuat seluruh loop berhenti karena fungsi itu mengembalikan error pertama, sehingga suspensi sesudahnya tidak pernah jalan (di produksi jalur ini dipanggil per tenant lewat `runBillingForTenant`; yang memakai DB tanpa scope tenant hanya `test/integration/isp_lifecycle_test.go`). Invoice semacam itu bisa benar-benar ada di produksi — restore backup men-upsert invoice tanpa memvalidasi referensinya (`system_backup.go` melakukan `checkTenantReference` pada ticket, tidak pada invoice) — jadi jalurnya ditutup di produk, bukan cuma di tes: invoice yatim dilewati, diwarn, dan dihitung di satu series berlabel tetap; `RecordPayment` tetap membukukan uangnya dan hanya melewatkan reaktivasi. Yang kedua murni batas kalender: `fupMonthlyTraffic` hanya menjumlah sesi yang mulai di dalam bulan berjalan, jadi sesi "dua jam lalu" keluar dari periode pada dua jam pertama tiap bulan. Perilaku produk FUP benar — periode baru memang kuota baru — yang salah adalah fixture tesnya. Putaran review kedua menambahkan `t.Cleanup` pada fixture nomor dokumen (invoice `subscription_id = 0` hasil tes itu menetap di dataset bersama dan jadi pemicu temuan pertama di run berikutnya) dan memperbaiki komentar akar masalah yang semula salah menyebut reseed demo — reseed justru menghapus subscription hanya ketika `remainingInvoices == 0` (`internal/demoseed/seeder.go`).

Angka yang jadi dasar keputusan (OpenLDAP container di `docker-compose.test.yml`, loopback, go1.27.0): biaya koneksi = 77 % (template) / 84 % (search) dari latency satu verifikasi, dan tenggat pemanggil 200 ms dulu diabaikan — `Verify` baru kembali setelah `ldap.Timeout` 3 s penuh. Setelah propagasi context: 200,5 ms / 200,1 ms. Pooling ditolak karena Bind membuat koneksi melekat pada identitas subscriber; tabungan yang dibayar (~0,5 ms dial loopback) tidak sebanding dengan risiko bocor identitas. Profil pada directory produksi sungguhan (`ldaps://` lintas jaringan) tetap belum dikerjakan dan tidak diklaim.

Putaran review ketiga menutup satu temuan yang membuat dua putaran sebelumnya setengah benar: `app.GetSystemMetrics()` tidak punya caller produksi, jadi `fup_scheduler_*` dan `billing_invoice_without_subscription_total` tidak pernah muncul di `GET /metrics` — klaim handbook dan komentar kode lebih dulu memeriksa kenyataan itu (`internal/webserver/metrics.go` kini menggabungkan snapshot tersebut, dan mutation check membuktikan assertion-nya jatuh kalau wiring dilepas). Di putaran yang sama CI membuktikan satu cacat lama yang tidak berhubungan dengan PR ini: `TestISPBillingLifecycleThroughRadius` kalah 1 dari 2 run karena Accounting-Response dikirim sebelum task accounting jalan, sehingga Stop bisa diproses lebih dulu dari Start-nya dan hilang. Tesnya sekarang menunggu baris accounting (sama seperti pola yang sudah dipakai `test/integration/ipv6_test.go`); sisanya — order task per sesi di jalur produksi — tercatat sebagai #16 dan sengaja tidak dikerjakan di PR ini.

Setelah #120 merge dengan 31 check hijau, rilis tetap gagal — dan untuk alasan yang tidak bisa dilihat CI mana pun: matrix `Build` hanya amd64/arm64, sedangkan `Publish GHCR image` menambah `linux/arm/v7`, tempat `int` cuma 32 bit, sehingga `up > math.MaxUint32` menolak dikompilasi (#18). `pfsense_enhancer.go` masuk jalur rilis; `toUint32FromInt` di `cmd/radtest` dan satu baris tabel `validutil_test.go` adalah kelas cacat yang sama dan ikut dirapikan karena `GOARCH=arm go vet -tags=integration,eap_accept ./...` sekarang bersih untuk seluruh repo. Yang ditutup bukan cuma tambalannya: job `Test` kini mem-vet tree untuk `linux/arm`, dan karena vet ikut mengompilasi file tes, jalur itu juga menangkap literal besar di fixture — platform yang benar-benar dikirim rilis diperiksa sebelum merge, bukan di jam yang sama dengan tag.

## 6. Menunggu keputusan atau permintaan

| # | Item | Kode | Tipe | Kondisi |
| --- | --- | --- | --- | --- |
| 16 | Accounting-Stop yang diproses lebih dulu dari task Start-nya hilang tanpa jejak | TR-F010 | `LOKAL` | Ditemukan lewat CI (flake di `TestISPBillingLifecycleThroughRadius`), bukan dari PR ini. `internal/radiusd/radius_acct.go:103` mengirim Accounting-Response **sebelum** `submitAcctTask` dijalankan (`:133`), dan worker pool tidak menjamin urutan per `Acct-Session-Id`; `StartHandler` juga menulis `radius_online` (`internal/radiusd/plugins/accounting/handlers/start_handler.go:73`) sebelum `radius_accounting` (`:97`), jadi Stop yang datang di celah itu mengembalikan `no records found` dan trafik sesi tidak terbukukan. Putaran ini hanya menutup sync point tesnya (#15); perbaikan produk butuh keputusan desain (satu transaksi di Start, atau Stop handler yang menunggu/meng-upsert, atau ordering per sesi) dan sengaja tidak diambil di PR ini |
| 17 | EAP-PWD (RFC 5931) | M12.2–M12.5 | `LOKAL` | demand-gated; spesifikasi sudah tersedia (`docs/rfcs/rfc5931-eap-pwd.txt`), kode belum ada |

## 7. Backlog audit — 34 item sisa

### 7.1 Bisa dieksekusi lokal — selesai semua

| ID | Item | TR-F | Bukti | Sisa |
| --- | --- | --- | --- | --- |
| ENG-055 | State loading, stale-data, retry pada OperationsPage | TR-F031 | commit `6fb38ca`, e2e `operations-feedback.spec.ts` | verifikasi SNMP nyata tetap `ENG-049` |
| ENG-064 | Pemeriksaan dead-letter dan alert session-action queue | TR-F010 | commit `2f4879d` + `bd609cb`, alert per tenant | pilot NAS nyata tetap `ENG-013` |
| ENG-058 | Label confidence dan mapping pada NOC impact UI | TR-F031 | commit `6fb38ca`, e2e NOC impact | ikut `ENG-055` |
| ENG-005 | Evaluasi inner EAP-TTLS CHAP dan MS-CHAP | TR-F004 | commit `dba0660` + `2a84f56`, unit + `test/integration/ttls_test.go` | interoperabilitas supplicant nyata tetap `ENG-002` |

Implementasi dan tes lokalnya selesai; klaim interoperabilitas/perangkat nyata tetap tidak dibuat dan tetap berada di daftar `ASET`.

### 7.2 Butuh aset nyata (14 · `ASET`; `ENG-015` keluar dari daftar ini setelah profil beban dieksekusi terhadap kontainer OpenLDAP CI di §5)

`ENG-002` supplicant EAP-TLS 1.2/1.3 · `ENG-013` pilot CoA/Disconnect NAS nyata · `ENG-041` instalasi fresh VPS bersih · `ENG-042` simulasi kegagalan registry saat upgrade · `ENG-043` drill restore seluruh volume VPS · `ENG-045` RTO/RPO operator · `ENG-046` akses localhost/SSH tunnel tanpa DNS · `ENG-047` audit health & expiry sertifikat deployment · `ENG-048` UAT WhatsApp pairing ke nomor uji · `ENG-049` UAT SNMPv3 authPriv perangkat nyata · `ENG-050` matriks NAS/firmware RADIUS+CoA · `ENG-014` prioritas vendor VSA dari NAS riil · `ENG-020` restore lintas tenant setelah perubahan skema · `ENG-044` studi WAL PITR PostgreSQL

### 7.3 Riset ekonomi (10 · `RISET`)

`ECO-001` willingness-to-pay · `ECO-002` waktu tugas operator sebelum/sesudah · `ECO-003` model TCO deployment · `ECO-004` lisensi managed support vs self-hosted · `ECO-005` desain paket dukungan & SLA · `ECO-007` kendali biaya telemetri WhatsApp · `ECO-008` alokasi biaya per NAS/subscriber · `ECO-010` risiko churn dari sinyal operasional · `ECO-011` strategi distribusi community/open-source · `ECO-012` partner interoperability kit

### 7.4 Riset lanjutan (10 · `RISET`, research-only)

`FUT-003` ringkasan insiden berbukti · `FUT-004` deteksi anomali latency · `FUT-005` graph hipotesis penyebab · `FUT-008` preview dampak CoA · `FUT-012` eksperimen adversarial EAP fragment scheduler · `FUT-013` snapshot NOC offline read-only · `FUT-014` field diagnostics mini-mode · `FUT-019` edge collector polling read-only · `FUT-021` knowledge base kompatibilitas firmware · `FUT-024` prediksi kapasitas dari SNMP nyata

Semua `FUT` dan `ECO` berstatus research-only: tidak boleh naik ke kode produksi tanpa pembaruan `docs/feature-checklist.md` dan persetujuan scope.

## 8. Residual rilis

- Tag `v0.18.0`–`v0.19.8` ada tanpa GitHub Release (sudah tersaingi `v0.20.0`). Backfill hanya lewat `workflow_dispatch` dengan `release_tag`, dan **butuh persetujuan eksplisit**.
- Tag `v0.22.0` tetap ada tanpa GitHub Release: build `linux/arm/v7` gagal sehingga aset tidak pernah terbit. Perbaikan sudah masuk di `v0.22.1` yang terbit lengkap, jadi `v0.22.0` tidak di-backfill kecuali diminta.
- Pemeriksaan operasional eksternal yang belum pernah dieksekusi dan tidak boleh diklaim: instalasi/upgrade VPS bersih, kegagalan registry/port nyata, timer, dan RTO produksi.

## 9. Status penutupan (2026-11-01)

Semua pekerjaan lokal yang tercatat di `Todo.md` sudah selesai. Sisa `ASET` (14 item) dan `RISET` (20 item) tetap membutuhkan aset nyata atau kajian — tidak bisa diselesaikan dari repo ini. Audit rilis terverifikasi: `v0.22.1` terbit lengkap (multi-arch, assets, GHCR), `v0.22.0` tag ada tanpa aset karena gagal build arm/v7 (semua dipindai dan diperbaiki). Catatan rilis otomatis di `CHANGELOG.md`.
