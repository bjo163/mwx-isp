# MWX-ISP — Evidence-first Repository Audit, 27 Oktober 2026

## Baseline terverifikasi
- Repository: https://github.com/bjo163/mwx-isp; default `main`; HEAD `1b791f80eed0ead3d679dd890fed56a342d4c104` (release v0.14.0).
- `main...dev`: **identical** menurut GitHub compare API (0 ahead/behind). Hanya branch `main` dan `dev` yang terdeteksi.
- Struktur: Go server modular monolith (internal/radiusd, adminapi, billing, tenancy, networkmonitor, notify, app), React Admin + Vite TypeScript frontend, PostgreSQL/SQLite, Docker/VPS scripts, GitHub Actions.
- GitHub Issues #45–#54 (AUDIT-01..AUDIT-10) **closed**, termasuk tenancy, patched dependency, voucher lifecycle, backup, branch synchronization, NOC mapping, dan customer timeline; jangan membuat issue dengan deliverable identik.
- PR #58 merged pada 2026-10-27, H01/H05/H06/H14 close-out. v0.14.0 release published 2026-10-27.
- `docs/roadmap.md` menyatakan M18 delivered; M10.5 anonymous outer NAI belum selesai; M14.5 LDAP pooling blocked menunggu load evidence; M5 future vendor work demand-driven.
- `docs/feature-checklist.md` mengklasifikasikan **payment gateway/customer self-service/hosted captive portal** sebagai non-goals; portal/voucher/ticket yang sudah ada bersifat keep/harden/hide, bukan perluasan produk.
- Uji PostgreSQL terisolasi H06 sudah dilaporkan; belum terbukti drill keseluruhan VPS/data volumes maupun RTO produksi. Uji real SNMP/NAS dan clean VPS masih batas eksplisit.

## Temuan / risiko yang dibedakan statusnya
1. **VERIFIED**: kebutuhan M10.5 privacy identity; test M10.1–M10.4 tidak berarti identitas luar sudah aman. Teknisnya perlu pipeline order review.
2. **VERIFIED**: browser landing/portal mengiklankan entry yang di API memerlukan operator JWT; tidak boleh diiklankan sebagai self-service.
3. **VERIFIED**: HTTP public payment endpoints fail closed menurut checklist. Tidak boleh mengusulkan menghidupkannya tanpa legal/commercial scope approval.
4. **VERIFIED**: dokumen H06 mencakup isolated PG restore dan secara eksplisit mengecualikan VPS volumes/prod RTO.
5. **VERIFIED**: operator WA realtime pairing dan SNMP/real NAS pilot belum terbukti operasional pada target pengguna.
6. **INFERRED/PROPOSED**: berbagai threat-model, overflow, fuzz, browser UX, dan business research dalam 100-issue inventory **belum otomatis berarti ada bug**; issue menginstruksikan inspeksi/pengujian dahulu.

## Reallocation domain versus berkas Ultra Megaprompt V5
V5 menawarkan 32 Engineering / 30 Frontier / 8 Economy / 10 Charity / 20 Religion-Ontology untuk MoonWitness-oriented ecosystem. Namun V5 juga secara eksplisit meminta adaptasi bila domain tidak berlaku. Untuk **repo ISP** tepatnya: **64 Engineering (64%), 24 Frontier experiment (24%), 12 Economy research (12%), 0 Charity, 0 Faith/Ontology**. Ini portofolio tepat **100** dan tidak memasukkan fikih, donasi, kitab suci, atau corpus ke produk RADIUS yang tidak memilikinya. Jika ingin membangun domain lain, tempatkan pada repo/domain yang relevan dan jangan membebani TR-F produksi. Aspek gaya *funky, grunge, neo-brutalist, Gen-Z* masuk FUT-017/FUT-018 sebagai eksperimen design-token, bukan service terpisah.

## Pembatasan saat eksekusi
- Research-only bagi eksperimen FUT dan ekonomi ECO, kecuali `docs/feature-checklist.md` diperbarui dan scope disetujui sebelum code change.
- Program otomatis tidak boleh membuat branch baru, force push, atau mengubah versi/tag, issue lama, maupun produksi tanpa izin terpisah.
- `AGENT.md` adalah instruksi agent kanonis. Semua PR baru harus mengacu TR-F dan gate Go tests/lint + web build sesuai scope. Penelitian ini tidak menjalankan `go test` lokal dan tidak menyatakan hasil itu green.
- Semua rekomendasi baru memiliki ID portofolio yang **belum** merupakan GitHub issue number.


---

# Exact 100-Issue Master Matrix — Adaptive ISP Portfolio

**NOTE:** these are draft IDs, not newly created GitHub issues. 64 Engineering + 24 Frontier + 12 Economy; Charity 0, Faith 0 due repo scope.

| Rank | ID | Title | Domain | TR-F | Priority | Wave | Score* | Dependency | Evidence |
|---:|---|---|---|---|---|---:|---:|---|---|
| 1 | `ENG-013` | Pilot RFC 5176 CoA/Disconnect pada NAS nyata | Engineering Excellence | `TR-F010` | P1 | 1 | 87 | — | PROPOSED_VERIFICATION |
| 2 | `ENG-017` | Audit penurunan hak anggota tenant saat token lama aktif | Engineering Excellence | `TR-F033` | P1 | 0 | 87 | — | PROPOSED_VERIFICATION |
| 3 | `ENG-032` | Invariant suspend manual tidak tertimpa bayar otomatis | Engineering Excellence | `TR-F028` | P1 | 1 | 87 | — | PROPOSED_VERIFICATION |
| 4 | `ENG-041` | Uji instalasi fresh pada VPS Ubuntu/Debian bersih | Engineering Excellence | `TR-F020` | P1 | 0 | 87 | — | VERIFIED_GAP |
| 5 | `ENG-002` | Matriks interoperabilitas EAP-TLS 1.2/1.3 dengan supplicant nyata | Engineering Excellence | `TR-F004` | P1 | 1 | 85 | ENG-001 | PROPOSED_VERIFICATION |
| 6 | `ENG-019` | Ratcheting audit query SQL raw dan joins tenancy | Engineering Excellence | `TR-F033` | P1 | 1 | 85 | — | PROPOSED_VERIFICATION |
| 7 | `ENG-028` | Penyamaran PII di logs, traces dan support evidence | Engineering Excellence | `TR-F022` | P1 | 1 | 85 | — | PROPOSED_VERIFICATION |
| 8 | `ENG-043` | Drill pemulihan seluruh volume VPS dari backup | Engineering Excellence | `TR-F020` | P1 | 1 | 85 | ENG-041 | VERIFIED_GAP |
| 9 | `ENG-055` | State loading, stale-data dan retry pada OperationsPage | Engineering Excellence | `TR-F031` | P1 | 1 | 85 | ENG-049 | PROPOSED_VERIFICATION |
| 10 | `ENG-001` | Audit & rancang outer anonymous NAI pada EAP-TLS 1.3 | Engineering Excellence | `TR-F004` | P1 | 0 | 83 | — | VERIFIED_GAP |
| 11 | `ENG-011` | Harness fuzz paket RADIUS cacat dan oversized | Engineering Excellence | `TR-F001` | P1 | 1 | 83 | ENG-010 | PROPOSED_VERIFICATION |
| 12 | `ENG-018` | Konsistensi tenant untuk NAS ambigu dan username sama | Engineering Excellence | `TR-F033` | P1 | 0 | 83 | ENG-017 | PROPOSED_VERIFICATION |
| 13 | `ENG-021` | Rekonsiliasi landing/portal publik yang faktanya JWT-only | Engineering Excellence | `TR-F018` | P1 | 0 | 83 | — | VERIFIED_GAP |
| 14 | `ENG-031` | Invoice numbering tahan retry setelah crash PostgreSQL | Engineering Excellence | `TR-F028` | P1 | 1 | 83 | — | PROPOSED_VERIFICATION |
| 15 | `ENG-033` | Idempotency pembayaran manual dengan request ulang | Engineering Excellence | `TR-F028` | P1 | 1 | 83 | — | PROPOSED_VERIFICATION |
| 16 | `ENG-052` | E2E matriks tombol per role operator/admin/super | Engineering Excellence | `TR-F016` | P1 | 1 | 83 | ENG-017 | PROPOSED_VERIFICATION |
| 17 | `ENG-009` | Regression RadSec mTLS sertifikat tidak tepercaya | Engineering Excellence | `TR-F003` | P1 | 1 | 81 | ENG-004 | PROPOSED_VERIFICATION |
| 18 | `ENG-010` | Audit duplikasi RADIUS retransmission dan replay | Engineering Excellence | `TR-F001` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 19 | `ENG-012` | Test accounting Start/Interim/Stop out-of-order | Engineering Excellence | `TR-F002` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 20 | `ENG-020` | Latihan restore lintas tenant setelah perubahan skema | Engineering Excellence | `TR-F033` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 21 | `ENG-022` | Threat-model autentikasi browser dan sesi admin | Engineering Excellence | `TR-F016` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 22 | `ENG-023` | Validasi rate limiting saat reverse proxy multi-hop | Engineering Excellence | `TR-F018` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 23 | `ENG-024` | Runbook rotasi secret SNMP dan encrypted community | Engineering Excellence | `TR-F031` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 24 | `ENG-025` | Audit perlindungan kredensial koneksi WhatsApp | Engineering Excellence | `TR-F030` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 25 | `ENG-027` | Hardening diagnostik jaringan terhadap SSRF | Engineering Excellence | `TR-F031` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 26 | `ENG-035` | Konsistensi status Subscription dan RadiusUser saat transisi | Engineering Excellence | `TR-F027` | P1 | 1 | 81 | ENG-032 | PROPOSED_VERIFICATION |
| 27 | `ENG-037` | Validasi integer IDR dan batas invoice items | Engineering Excellence | `TR-F028` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 28 | `ENG-038` | Voucher terminal-state concurrency regressions | Engineering Excellence | `TR-F007` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 29 | `ENG-042` | Simulasi kegagalan registry ketika upgrade aplikasi | Engineering Excellence | `TR-F020` | P1 | 1 | 81 | ENG-041 | PROPOSED_VERIFICATION |
| 30 | `ENG-056` | Informasi jujur untuk action portal/payment yang disabled | Engineering Excellence | `TR-F029` | P1 | 1 | 81 | ENG-021 | PROPOSED_VERIFICATION |
| 31 | `ENG-061` | Stabilitas test E2E CI branch main dan dev | Engineering Excellence | `TR-F022` | P1 | 1 | 81 | — | PROPOSED_VERIFICATION |
| 32 | `ENG-064` | Pemeriksaan session-action queue dead-letter dan alert | Engineering Excellence | `TR-F010` | P1 | 1 | 81 | ENG-013 | PROPOSED_VERIFICATION |
| 33 | `ENG-049` | UAT SNMPv3 authPriv dengan perangkat sungguhan | Engineering Excellence | `TR-F031` | P2 | 2 | 75 | — | VERIFIED_GAP |
| 34 | `ENG-053` | Navigasi keyboard pada data-grid dan dialog kritis | Engineering Excellence | `TR-F013` | P2 | 1 | 75 | — | PROPOSED_VERIFICATION |
| 35 | `ECO-001` | Wawancara willingness-to-pay pada operator ISP kecil | Sustainable Economy | `TR-F027` | P2 | 2 | 73 | — | PROPOSED_VERIFICATION |
| 36 | `ENG-045` | Tetapkan RTO/RPO operator berdasarkan rehearsal | Engineering Excellence | `TR-F020` | P2 | 2 | 73 | ENG-043 | PROPOSED_VERIFICATION |
| 37 | `ENG-050` | Matrix NAS/firmware teruji untuk RADIUS dan CoA | Engineering Excellence | `TR-F010` | P2 | 2 | 73 | ENG-013 | PROPOSED_VERIFICATION |
| 38 | `ENG-058` | Label confidence dan mapping pada NOC impact UI | Engineering Excellence | `TR-F031` | P2 | 2 | 73 | ENG-055 | PROPOSED_VERIFICATION |
| 39 | `ENG-004` | Kesiapan rotasi sertifikat RadSec dan EAP | Engineering Excellence | `TR-F003` | P2 | 1 | 71 | — | PROPOSED_VERIFICATION |
| 40 | `ENG-016` | Pemeriksaan berkala perbedaan fork layeh dan upstream | Engineering Excellence | `TR-F021` | P2 | 2 | 71 | — | PROPOSED_VERIFICATION |
| 41 | `ENG-030` | Inventory SBOM backend/frontend serta lisensi | Engineering Excellence | `TR-F022` | P2 | 1 | 71 | — | PROPOSED_VERIFICATION |
| 42 | `ENG-048` | UAT WhatsApp pairing ke nomor uji opt-in | Engineering Excellence | `TR-F030` | P2 | 2 | 71 | — | VERIFIED_GAP |
| 43 | `ECO-002` | Ukur waktu tugas operator sebelum/sesudah MWX-ISP | Sustainable Economy | `TR-F029` | P2 | 2 | 69 | ECO-001 | PROPOSED_VERIFICATION |
| 44 | `ECO-003` | Model total cost of ownership deployment ISP | Sustainable Economy | `TR-F020` | P2 | 2 | 69 | ENG-045 | PROPOSED_VERIFICATION |
| 45 | `ENG-008` | Pastikan EAP-OTP tak dapat diaktifkan tanpa validator | Engineering Excellence | `TR-F004` | P2 | 1 | 69 | — | PROPOSED_VERIFICATION |
| 46 | `ENG-015` | Profil biaya LDAP bind di bawah concurrent load | Engineering Excellence | `TR-F025` | P2 | 1 | 69 | — | PROPOSED_VERIFICATION |
| 47 | `ENG-029` | PNG branding parser dan metadata validation | Engineering Excellence | `TR-F029` | P2 | 1 | 69 | — | PROPOSED_VERIFICATION |
| 48 | `ENG-036` | Jejak scheduler FUP terpisah per tenant sesudah perbaikan | Engineering Excellence | `TR-F033` | P2 | 2 | 69 | ENG-019 | PROPOSED_VERIFICATION |
| 49 | `ENG-039` | Perlindungan marker demo data saat clean-up | Engineering Excellence | `TR-F032` | P2 | 1 | 69 | — | PROPOSED_VERIFICATION |
| 50 | `ENG-040` | Studi archival accounting retention dan query plan | Engineering Excellence | `TR-F011` | P2 | 2 | 69 | — | PROPOSED_VERIFICATION |
| 51 | `ENG-046` | Uji akses localhost SSH tunnel pada instalasi tanpa DNS | Engineering Excellence | `TR-F020` | P2 | 1 | 69 | ENG-041 | PROPOSED_VERIFICATION |
| 52 | `ENG-051` | Test fungsional website editor yang masih kurang coverage | Engineering Excellence | `TR-F029` | P2 | 1 | 69 | — | VERIFIED_GAP |
| 53 | `ENG-054` | Rekonsiliasi ID/EN seluruh status tagihan dan insiden | Engineering Excellence | `TR-F023` | P2 | 2 | 69 | — | PROPOSED_VERIFICATION |
| 54 | `ENG-057` | Audit grafik dashboard konsistensi timezone dan nominal | Engineering Excellence | `TR-F015` | P2 | 2 | 69 | — | PROPOSED_VERIFICATION |
| 55 | `ENG-059` | Audit import RADIUS user dengan preview dry-run saja | Engineering Excellence | `TR-F007` | P2 | 2 | 69 | — | PROPOSED_VERIFICATION |
| 56 | `ENG-062` | Sistem upgrade dependency dengan ratchet keamanan | Engineering Excellence | `TR-F022` | P2 | 2 | 69 | ENG-030 | PROPOSED_VERIFICATION |
| 57 | `ENG-063` | Contract testing respons API, paginasi dan error | Engineering Excellence | `TR-F012` | P2 | 1 | 69 | — | PROPOSED_VERIFICATION |
| 58 | `ECO-004` | Evaluasi lisensi managed support vs self-hosted | Sustainable Economy | `TR-F020` | P2 | 3 | 67 | ECO-001 | PROPOSED_VERIFICATION |
| 59 | `ECO-005` | Desain paket dukungan & SLA yang realistis | Sustainable Economy | `TR-F020` | P2 | 3 | 67 | ENG-045 | PROPOSED_VERIFICATION |
| 60 | `ECO-006` | Validasi jasa onboarding dan migrasi pelanggan | Sustainable Economy | `TR-F027` | P2 | 3 | 67 | ENG-059 | PROPOSED_VERIFICATION |
| 61 | `ECO-007` | Pengendalian biaya telemetri dan alert whatsmeow | Sustainable Economy | `TR-F030` | P2 | 2 | 67 | ENG-048 | PROPOSED_VERIFICATION |
| 62 | `ECO-008` | Cost allocation unit per NAS dan subscriber aktif | Sustainable Economy | `TR-F015` | P2 | 3 | 67 | ECO-003 | PROPOSED_VERIFICATION |
| 63 | `ECO-009` | Audit peluang revenue leakage dari invoice yang sudah ada | Sustainable Economy | `TR-F028` | P2 | 2 | 67 | ENG-031 | PROPOSED_VERIFICATION |
| 64 | `ECO-011` | Kajian strategi distribusi community/open-source | Sustainable Economy | `TR-F023` | P2 | 3 | 67 | ECO-001 | PROPOSED_VERIFICATION |
| 65 | `ECO-012` | Partner interoperability kit berbasis fixture NAS | Sustainable Economy | `TR-F005` | P2 | 4 | 67 | ENG-050 | PROPOSED_VERIFICATION |
| 66 | `ENG-003` | Pemetaan kebijakan pencabutan sertifikat EAP-TLS | Engineering Excellence | `TR-F004` | P2 | 2 | 67 | — | PROPOSED_VERIFICATION |
| 67 | `ENG-005` | Evaluasi dukungan inner EAP-TTLS CHAP dan MS-CHAP | Engineering Excellence | `TR-F004` | P2 | 2 | 67 | ENG-002 | PROPOSED_VERIFICATION |
| 68 | `ENG-014` | Prioritas vendor VSA berbasis model NAS riil | Engineering Excellence | `TR-F005` | P2 | 2 | 67 | ENG-013 | PROPOSED_VERIFICATION |
| 69 | `ENG-026` | Evaluasi parameter KDF dan envelope backup | Engineering Excellence | `TR-F020` | P2 | 2 | 67 | — | PROPOSED_VERIFICATION |
| 70 | `ENG-034` | Pembetulan transaksi payment operator yang salah input | Engineering Excellence | `TR-F028` | P2 | 2 | 67 | ENG-033 | PROPOSED_VERIFICATION |
| 71 | `ENG-044` | Studi PostgreSQL WAL PITR untuk RPO terukur | Engineering Excellence | `TR-F017` | P2 | 2 | 67 | ENG-043 | PROPOSED_VERIFICATION |
| 72 | `ENG-047` | Audit health dan peringatan expiry sertifikat deployment | Engineering Excellence | `TR-F020` | P2 | 2 | 67 | ENG-041 | PROPOSED_VERIFICATION |
| 73 | `ENG-060` | Studi support bundle tersensor untuk insiden operator | Engineering Excellence | `TR-F022` | P2 | 2 | 67 | ENG-028 | PROPOSED_VERIFICATION |
| 74 | `FUT-013` | Offline read-only NOC snapshot dengan TTL dan enkripsi | Frontier Innovation | `TR-F031` | P3 | 3 | 52 | ENG-055 | PROPOSED_VERIFICATION |
| 75 | `FUT-017` | Experimental neo-brutalist manga skin via existing tokens | Frontier Innovation | `TR-F029` | P3 | 3 | 52 | ENG-053 | PROPOSED_VERIFICATION |
| 76 | `ECO-010` | Riset risiko churn berbasis sinyal operasional agregat | Sustainable Economy | `TR-F031` | P3 | 4 | 50 | ECO-002 | PROPOSED_VERIFICATION |
| 77 | `ENG-006` | Riset kompatibilitas TEAP terbatas untuk milestone M11 | Engineering Excellence | `TR-F004` | P3 | 3 | 50 | — | PROPOSED_VERIFICATION |
| 78 | `ENG-007` | Riset kebutuhan EAP-PWD milestone M12 | Engineering Excellence | `TR-F004` | P3 | 3 | 50 | — | PROPOSED_VERIFICATION |
| 79 | `FUT-001` | Proof-of-concept intent search untuk perintah operator read-only | Frontier Innovation | `TR-F013` | P3 | 3 | 50 | ENG-052 | PROPOSED_VERIFICATION |
| 80 | `FUT-002` | AI incident triage berbasis log tersensor | Frontier Innovation | `TR-F031` | P3 | 3 | 50 | ENG-028 | PROPOSED_VERIFICATION |
| 81 | `FUT-003` | Ringkasan insiden berbukti tanpa mengklaim pelanggan offline | Frontier Innovation | `TR-F031` | P3 | 3 | 50 | ENG-058 | PROPOSED_VERIFICATION |
| 82 | `FUT-004` | Deteksi pola anomali latency dari metrik historis bounded | Frontier Innovation | `TR-F031` | P3 | 3 | 50 | ENG-049 | PROPOSED_VERIFICATION |
| 83 | `FUT-005` | Graph hipotesis penyebab insiden dengan uncertainty | Frontier Innovation | `TR-F031` | P3 | 3 | 50 | ENG-058 | PROPOSED_VERIFICATION |
| 84 | `FUT-006` | Visual graph statis NAS-ODP dari relasi yang telah diisi operator | Frontier Innovation | `TR-F031` | P3 | 3 | 50 | ENG-018 | PROPOSED_VERIFICATION |
| 85 | `FUT-007` | Simulasi perubahan profil bandwidth tanpa menulis NAS | Frontier Innovation | `TR-F008` | P3 | 4 | 50 | ENG-016 | PROPOSED_VERIFICATION |
| 86 | `FUT-008` | Preview dampak CoA terhadap sesi yang aktif sebelum aksi | Frontier Innovation | `TR-F010` | P3 | 3 | 50 | ENG-013 | PROPOSED_VERIFICATION |
| 87 | `FUT-009` | Explanation trace billing-suspend dari rule engine existing | Frontier Innovation | `TR-F028` | P3 | 3 | 50 | ENG-032 | PROPOSED_VERIFICATION |
| 88 | `FUT-010` | Static tenant-scope policy checker dari query AST sederhana | Frontier Innovation | `TR-F033` | P3 | 3 | 50 | ENG-019 | PROPOSED_VERIFICATION |
| 89 | `FUT-011` | Pembangkitan regression tests untuk paket RADIUS via mutator | Frontier Innovation | `TR-F001` | P3 | 3 | 50 | ENG-011 | PROPOSED_VERIFICATION |
| 90 | `FUT-012` | Eksperimen adversarial EAP TLS fragment scheduler | Frontier Innovation | `TR-F004` | P3 | 3 | 50 | ENG-002 | PROPOSED_VERIFICATION |
| 91 | `FUT-014` | Field diagnostics mini-mode untuk teknisi yang offline | Frontier Innovation | `TR-F031` | P3 | 4 | 50 | FUT-013 | PROPOSED_VERIFICATION |
| 92 | `FUT-015` | Adaptive contextual workspace dari tugas operator | Frontier Innovation | `TR-F013` | P3 | 4 | 50 | ENG-053 | PROPOSED_VERIFICATION |
| 93 | `FUT-016` | Voice-assisted navigation untuk aksesibilitas internal | Frontier Innovation | `TR-F013` | P3 | 4 | 50 | ENG-053 | PROPOSED_VERIFICATION |
| 94 | `FUT-018` | Riset visual incident card grunge yang bisa dicetak | Frontier Innovation | `TR-F029` | P3 | 4 | 50 | FUT-017 | PROPOSED_VERIFICATION |
| 95 | `FUT-019` | Edge collector polling read-only pada jaringan terbatas | Frontier Innovation | `TR-F031` | P3 | 4 | 50 | ENG-049 | PROPOSED_VERIFICATION |
| 96 | `FUT-020` | Per-tenant statistik operasional privat berbasis agregat | Frontier Innovation | `TR-F033` | P3 | 4 | 50 | ENG-017 | PROPOSED_VERIFICATION |
| 97 | `FUT-021` | Knowledge base kompatibilitas firmware berbasis bukti uji | Frontier Innovation | `TR-F005` | P3 | 3 | 50 | ENG-050 | PROPOSED_VERIFICATION |
| 98 | `FUT-022` | Event replay simulator untuk reproduksi billing dan NAS | Frontier Innovation | `TR-F028` | P3 | 3 | 50 | ENG-032 | PROPOSED_VERIFICATION |
| 99 | `FUT-023` | Graph semantik relasi protokol RADIUS dan feature ID | Frontier Innovation | `TR-F021` | P3 | 3 | 50 | ENG-016 | PROPOSED_VERIFICATION |
| 100 | `FUT-024` | Prediksi kapasitas network yang dibatasi data SNMP nyata | Frontier Innovation | `TR-F031` | P3 | 4 | 50 | ENG-049 | PROPOSED_VERIFICATION |

*Scores are **provisional planning heuristics**, not measured success or empirical severity. Critical/security findings need on-site verification before prioritization.*


---

# 25-Report Index — factual vs proposed, not fictional research

## Report 01 Executive ecosystem intelligence
MWX-ISP mengelola ISP ops—RADIUS/EAP, NAS, sessions, billing, tenant, bounded monitoring, operator UI. Keunggulan adalah modular monolith dan test coverage; risiko adalah real-device acceptance dan scope leakage.

## Report 02 Architecture & implementation state
Go/Echo/GORM + React Admin/Vite + PostgreSQL/SQLite + Docker/Caddy, satu monolith; code paths dicatat pada per-issue. Current main/dev identical, v0.14.0.

## Report 03 Engineering excellence
Prioritas utama bukan mengulang AUDIT-01..10, melainkan EAP anonymized NAI, tenant guard tests, public UX clarity, clean VPS rehearsal, real NAS/SNMP.

## Report 04 Frontier research
24 konsep hanya eksperimen bounded/research; no remote shell, automated network writes, magic AI agent, atau production changes sebelum approved scope.

## Report 05 Economic opportunity
12 research tasks untuk willingness-to-pay, total cost, support, unit economics dan integrator adoption; tidak menyamakan opportunity dengan actual revenue.

## Report 06 Charity and humanitarian
Applicability = NOT APPLICABLE untuk RADIUS admin monolith; tidak menambah donasi, beneficiary atau zakat ke ISP software. If user creates a separate charity repo, audit there independently.

## Report 07 Faith, religion, ontology
Applicability = NOT APPLICABLE. RFC/protocol relation graph FUT-023 adalah knowledge model teknis, BUKAN klaim theological ontology.

## Report 08 Unified ISP product vision
Reliable multi-tenant ISP operations, auditable network access and billing, usable operator experiences; emerging capabilities must pass security and relevance gates.

## Report 09 Gap analysis
Known: M10.5, real device pilot, clean host VPS, incomplete customer-facing entry. Proposed: 100 test/research improvements with explicit uncertain evidence.

## Report 10 Candidate discovery
Identified 100 scoped distinct candidates; NOT claimed that 250 candidates or 15 papers/15 repos were researched. Such coverage remains outside current pass.

## Report 11 Exact 100-issue matrix
See 02-issue-master-100.md and 06-issues-machine-readable.json.

## Report 12 Engineering issue bodies
See 03-engineering-64.md; all include 22 sections, evidence statuses, test criteria, microtasks and mitigation.

## Report 13 Frontier issue bodies
See 04-frontier-24.md; all prototypes research-only with go/no-go gate.

## Report 14 Economic issue bodies
See 05-economy-12.md; research-only, no payment infrastructure.

## Report 15 Charity issue bodies
Zero, by adaptive applicability rule in original V5; cannot fabricate charitable ISP backlog.

## Report 16 Faith/ontology issue bodies
Zero, by adaptive applicability rule in original V5; network RFC graph distinct from sacred texts.

## Report 17 Dedup and reconciliation
#45–#54 closed; M18 delivered. All 100 titles unique and issue IDs unique; semantically similar regression tasks require agent review before creating on GitHub.

## Report 18 Dependency and correlation
See 07-correlation-matrix.csv. All explicit blockers checked for referential integrity and acyclic graph. Related-to is not falsely promoted to BLOCKS.

## Report 19 Architecture contract map
Access: TR-F016/033. RADIUS: TR-F001-011. ISP/billing: TR-F027/028. Web/branding: TR-F013/029. Network: TR-F031. Operator notifications: TR-F030. Docs/release: TR-F020-024.

## Report 20 Milestone and execution blueprint
See 08-execution-roadmap.md. Waves 0–4 are dependency-aware, not assumed dates.

## Report 21 Rankings
Rank from the SAME 100 matrix by planning score (illustrative). Priorities require explicit pilot evidence; no new issue IDs invented.

## Report 22 Cost complexity sustainability
No actual cloud/API revenue or utilization measured. COO/TCO research ECO-003/007/008 establishes assumptions and three scenarios.

## Report 23 Risk & governance register
Important risks: tenant data isolation, RADIUS protocol correctness, SNMP secrets, WA unofficial provider reliability, backup recovery, misleading public UX and premature AI/autonomy.

## Report 24 First execution batch
Wave0: ENG-001, ENG-017, ENG-018, ENG-021, ENG-041. Next run real-device SNMP/NAS, UI auth clarity and monitoring freshness regressions as prerequisites allow.

## Report 25 Portfolio integrity certificate
Counts 64/24/12 (total100) PASS; unique IDs PASS; dependency DAG PASS; no new GitHub issues PASS. Full source archaeology, 250 candidates, production tests, device UAT, market interviews and legal checks NOT COMPLETE; no perfect consistency claim.


---

# Dependency-aware execution roadmap

Every wave is an ordering recommendation, **not** automatic permission to implement. No invented completion dates.

## Wave 0: Critical verification / exact scope gaps
- Number of issues: **5**
- IDs: `ENG-001`, `ENG-017`, `ENG-018`, `ENG-021`, `ENG-041`
- Gate: measurable, cited outcome; no unexpected downstream writes.

## Wave 1: Safety reliability and regression foundations
- Number of issues: **37**
- IDs: `ENG-002`, `ENG-004`, `ENG-008`, `ENG-009`, `ENG-010`, `ENG-011`, `ENG-012`, `ENG-013`, `ENG-015`, `ENG-019`, `ENG-020`, `ENG-022`, `ENG-023`, `ENG-024`, `ENG-025`, `ENG-027`, `ENG-028`, `ENG-029`, `ENG-030`, `ENG-031`, `ENG-032`, `ENG-033`, `ENG-035`, `ENG-037`, `ENG-038`, `ENG-039`, `ENG-042`, `ENG-043`, `ENG-046`, `ENG-051`, `ENG-052`, `ENG-053`, `ENG-055`, `ENG-056`, `ENG-061`, `ENG-063`, `ENG-064`
- Gate: measurable, cited outcome; no unexpected downstream writes.

## Wave 2: Measurement, research and hardening
- Number of issues: **25**
- IDs: `ENG-003`, `ENG-005`, `ENG-014`, `ENG-016`, `ENG-026`, `ENG-034`, `ENG-036`, `ENG-040`, `ENG-044`, `ENG-045`, `ENG-047`, `ENG-048`, `ENG-049`, `ENG-050`, `ENG-054`, `ENG-057`, `ENG-058`, `ENG-059`, `ENG-060`, `ENG-062`, `ECO-001`, `ECO-002`, `ECO-003`, `ECO-007`, `ECO-009`
- Gate: measurable, cited outcome; no unexpected downstream writes.

## Wave 3: Controlled experiments + commercial discovery
- Number of issues: **23**
- IDs: `ENG-006`, `ENG-007`, `FUT-001`, `FUT-002`, `FUT-003`, `FUT-004`, `FUT-005`, `FUT-006`, `FUT-008`, `FUT-009`, `FUT-010`, `FUT-011`, `FUT-012`, `FUT-013`, `FUT-017`, `FUT-021`, `FUT-022`, `FUT-023`, `ECO-004`, `ECO-005`, `ECO-006`, `ECO-008`, `ECO-011`
- Gate: measurable, cited outcome; no unexpected downstream writes.

## Wave 4: Longer-horizon prototype decisions
- Number of issues: **10**
- IDs: `FUT-007`, `FUT-014`, `FUT-015`, `FUT-016`, `FUT-018`, `FUT-019`, `FUT-020`, `FUT-024`, `ECO-010`, `ECO-012`
- Gate: measurable, cited outcome; no unexpected downstream writes.

## Critical path
`ENG-001 → ENG-002`; `ENG-017 → ENG-018`; `ENG-041 → ENG-043 → ENG-045 → ECO-005`. Other independent tasks may run parallel.

## Governance hard stops
- Never automatically merge/deploy from planning artifact.
- Never create new branch (main/dev policy).
- Research-only FUT/ECO until approved TR-F scope revision; no customer-facing unapproved features.
- Finance operations remain within existing manual ISP invoice/payment scope.


---

# 03 Engineering 64

# ENG-001 — Audit & rancang outer anonymous NAI pada EAP-TLS 1.3

## 01 Identity
- ID: `ENG-001`; domain: **Engineering Excellence**; primary subdomain: **feature**; type: `feature`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **0**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:feature`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Susun urutan resolve NAS tenant lalu validasi sertifikat sebelum mapping identitas internal, tanpa bergantung pada outer username.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit & rancang outer anonymous nai pada eap-tls 1.3**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Paket uji outer anonymous@realm dan identitas internal berbeda lolos; mismatch sertifikat ditolak; TLS1.2 regresi hijau.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **contract-first change**. Work only within `Susun urutan resolve NAS tenant lalu validasi sertifikat sebelum mapping identitas internal, tanpa bergantung pada outer username.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/roadmap.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Susun urutan resolve NAS tenant lalu validasi sertifikat sebelum mapping identitas internal, tanpa bergantung pada outer username.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Paket uji outer anonymous@realm dan identitas internal berbeda lolos; mismatch sertifikat ditolak; TLS1.2 regresi hijau.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Paket uji outer anonymous@realm dan identitas internal berbeda lolos; mismatch sertifikat ditolak; TLS1.2 regresi hijau.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-002.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Paket uji outer anonymous@realm dan identitas internal berbeda lolos; mismatch sertifikat ditolak; TLS1.2 regresi hijau. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Susun urutan resolve NAS tenant lalu validasi sertifikat sebelum mapping identitas internal, tanpa bergantung pada outer username.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Paket uji outer anonymous@realm dan identitas internal berbeda lolos; mismatch sertifikat ditolak; TLS1.2 regresi hijau.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-002 — Matriks interoperabilitas EAP-TLS 1.2/1.3 dengan supplicant nyata

## 01 Identity
- ID: `ENG-002`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **85/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji klien nyata terhadap negosiasi, protected success, MPPE keys dan close_notify tanpa mengulang kode M10.1–10.4.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **matriks interoperabilitas eap-tls 1.2/1.3 dengan supplicant nyata**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Artefak konfigurasi klien, trace tersanitasi, minimal satu sukses dan satu penolakan terdokumentasi untuk kedua versi.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`test/integration/eap_tls_test.go`](https://github.com/bjo163/mwx-isp/blob/main/test/integration/eap_tls_test.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji klien nyata terhadap negosiasi, protected success, MPPE keys dan close_notify tanpa mengulang kode M10.1–10.4.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `test/integration/eap_tls_test.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji klien nyata terhadap negosiasi, protected success, MPPE keys dan close_notify tanpa mengulang kode M10.1–10.4.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Artefak konfigurasi klien, trace tersanitasi, minimal satu sukses dan satu penolakan terdokumentasi untuk kedua versi.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Artefak konfigurasi klien, trace tersanitasi, minimal satu sukses dan satu penolakan terdokumentasi untuk kedua versi.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-001.
- **ENABLES:** ENG-005, FUT-012.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Artefak konfigurasi klien, trace tersanitasi, minimal satu sukses dan satu penolakan terdokumentasi untuk kedua versi. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Uji klien nyata terhadap negosiasi, protected success, MPPE keys dan close_notify tanpa mengulang kode M10.1–10.4.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Artefak konfigurasi klien, trace tersanitasi, minimal satu sukses dan satu penolakan terdokumentasi untuk kedua versi.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-003 — Pemetaan kebijakan pencabutan sertifikat EAP-TLS

## 01 Identity
- ID: `ENG-003`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Buat threat model dan POC CRL/OCSP fail mode terhadap CA yang dikonfigurasi; jangan mengklaim sudah diterapkan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pemetaan kebijakan pencabutan sertifikat eap-tls**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Keputusan ADR menjelaskan cakupan revocation, timeout dan fail-open/fail-closed disertai uji dua skenario.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/plugins/eap/handlers/tls_config.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/plugins/eap/handlers/tls_config.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Buat threat model dan POC CRL/OCSP fail mode terhadap CA yang dikonfigurasi; jangan mengklaim sudah diterapkan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/plugins/eap/handlers/tls_config.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat threat model dan POC CRL/OCSP fail mode terhadap CA yang dikonfigurasi; jangan mengklaim sudah diterapkan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Keputusan ADR menjelaskan cakupan revocation, timeout dan fail-open/fail-closed disertai uji dua skenario.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Keputusan ADR menjelaskan cakupan revocation, timeout dan fail-open/fail-closed disertai uji dua skenario.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Keputusan ADR menjelaskan cakupan revocation, timeout dan fail-open/fail-closed disertai uji dua skenario. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Buat threat model dan POC CRL/OCSP fail mode terhadap CA yang dikonfigurasi; jangan mengklaim sudah diterapkan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Keputusan ADR menjelaskan cakupan revocation, timeout dan fail-open/fail-closed disertai uji dua skenario.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-004 — Kesiapan rotasi sertifikat RadSec dan EAP

## 01 Identity
- ID: `ENG-004`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **71/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit reload certificate chain, private key custody, expiry dan overlap rotation tanpa restart tidak aman.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F003` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **kesiapan rotasi sertifikat radsec dan eap**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Hasil simulasi cert lama/baru serta dokumentasi rollback dan observasi handshake tercatat.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/radsec_server.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/radsec_server.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F003`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Audit reload certificate chain, private key custody, expiry dan overlap rotation tanpa restart tidak aman.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F003` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/radsec_server.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit reload certificate chain, private key custody, expiry dan overlap rotation tanpa restart tidak aman.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Hasil simulasi cert lama/baru serta dokumentasi rollback dan observasi handshake tercatat.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Hasil simulasi cert lama/baru serta dokumentasi rollback dan observasi handshake tercatat.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-009.
- **REUSES:** `TR-F003` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Hasil simulasi cert lama/baru serta dokumentasi rollback dan observasi handshake tercatat. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F003`. Scope only: **Audit reload certificate chain, private key custody, expiry dan overlap rotation tanpa restart tidak aman.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Hasil simulasi cert lama/baru serta dokumentasi rollback dan observasi handshake tercatat.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-005 — Evaluasi dukungan inner EAP-TTLS CHAP dan MS-CHAP

## 01 Identity
- ID: `ENG-005`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Pisahkan inner PAP/MS-CHAP-V2 yang delivered dari CHAP/MS-CHAP lain yang belum dijanjikan selesai; ukur demand client.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **evaluasi dukungan inner eap-ttls chap dan ms-chap**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Tabel kompatibilitas, risiko downgrade, dan keputusan go/no-go per inner method dipublikasikan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Pisahkan inner PAP/MS-CHAP-V2 yang delivered dari CHAP/MS-CHAP lain yang belum dijanjikan selesai; ukur demand client.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/feature-checklist.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Pisahkan inner PAP/MS-CHAP-V2 yang delivered dari CHAP/MS-CHAP lain yang belum dijanjikan selesai; ukur demand client.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Tabel kompatibilitas, risiko downgrade, dan keputusan go/no-go per inner method dipublikasikan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Tabel kompatibilitas, risiko downgrade, dan keputusan go/no-go per inner method dipublikasikan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-002.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Tabel kompatibilitas, risiko downgrade, dan keputusan go/no-go per inner method dipublikasikan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Pisahkan inner PAP/MS-CHAP-V2 yang delivered dari CHAP/MS-CHAP lain yang belum dijanjikan selesai; ukur demand client.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Tabel kompatibilitas, risiko downgrade, dan keputusan go/no-go per inner method dipublikasikan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-006 — Riset kompatibilitas TEAP terbatas untuk milestone M11

## 01 Identity
- ID: `ENG-006`; domain: **Engineering Excellence**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun matriks client support dan handshake prototype isolated untuk TEAP tanpa mengganti koordinator EAP.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **riset kompatibilitas teap terbatas untuk milestone m11**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Hasil kompatibilitas minimal dua implementasi klien; jika tidak memadai, defer terdokumentasi.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Bangun matriks client support dan handshake prototype isolated untuk TEAP tanpa mengganti koordinator EAP.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/roadmap.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun matriks client support dan handshake prototype isolated untuk TEAP tanpa mengganti koordinator EAP.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Hasil kompatibilitas minimal dua implementasi klien; jika tidak memadai, defer terdokumentasi.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Hasil kompatibilitas minimal dua implementasi klien; jika tidak memadai, defer terdokumentasi.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Hasil kompatibilitas minimal dua implementasi klien; jika tidak memadai, defer terdokumentasi. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Bangun matriks client support dan handshake prototype isolated untuk TEAP tanpa mengganti koordinator EAP.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Hasil kompatibilitas minimal dua implementasi klien; jika tidak memadai, defer terdokumentasi.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-007 — Riset kebutuhan EAP-PWD milestone M12

## 01 Identity
- ID: `ENG-007`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P3**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p3`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Nilai keamanan, maintainability dan kebutuhan perangkat sebelum implementasi metode EAP-PWD.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **riset kebutuhan eap-pwd milestone m12**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dokumen penggunaan nyata, risiko dan keputusan no-go/go; tidak ada opsi production yang aktif tanpa tes.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Nilai keamanan, maintainability dan kebutuhan perangkat sebelum implementasi metode EAP-PWD.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/roadmap.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Nilai keamanan, maintainability dan kebutuhan perangkat sebelum implementasi metode EAP-PWD.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dokumen penggunaan nyata, risiko dan keputusan no-go/go; tidak ada opsi production yang aktif tanpa tes.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dokumen penggunaan nyata, risiko dan keputusan no-go/go; tidak ada opsi production yang aktif tanpa tes.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dokumen penggunaan nyata, risiko dan keputusan no-go/go; tidak ada opsi production yang aktif tanpa tes. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Nilai keamanan, maintainability dan kebutuhan perangkat sebelum implementasi metode EAP-PWD.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dokumen penggunaan nyata, risiko dan keputusan no-go/go; tidak ada opsi production yang aktif tanpa tes.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-008 — Pastikan EAP-OTP tak dapat diaktifkan tanpa validator

## 01 Identity
- ID: `ENG-008`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit handler OTP yang ada tetapi tidak diregistrasi; kunci pengaturan fail-closed dan tulis kontrak backend jika nanti dibutuhkan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pastikan eap-otp tak dapat diaktifkan tanpa validator**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Konfigurasi invalid/enable tanpa backend selalu ditolak; hasil uji keamanan tercatat.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Audit handler OTP yang ada tetapi tidak diregistrasi; kunci pengaturan fail-closed dan tulis kontrak backend jika nanti dibutuhkan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/feature-checklist.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit handler OTP yang ada tetapi tidak diregistrasi; kunci pengaturan fail-closed dan tulis kontrak backend jika nanti dibutuhkan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Konfigurasi invalid/enable tanpa backend selalu ditolak; hasil uji keamanan tercatat.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Konfigurasi invalid/enable tanpa backend selalu ditolak; hasil uji keamanan tercatat.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Konfigurasi invalid/enable tanpa backend selalu ditolak; hasil uji keamanan tercatat. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Audit handler OTP yang ada tetapi tidak diregistrasi; kunci pengaturan fail-closed dan tulis kontrak backend jika nanti dibutuhkan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Konfigurasi invalid/enable tanpa backend selalu ditolak; hasil uji keamanan tercatat.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-009 — Regression RadSec mTLS sertifikat tidak tepercaya

## 01 Identity
- ID: `ENG-009`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Perluas kontrak tes tentang verifikasi peers, wrong CA dan perubahan TLS config berdasarkan implementasi aktual.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F003` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **regression radsec mtls sertifikat tidak tepercaya**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Handshake negative/positive matrix hijau dan tidak melonggarkan kebijakan sertifikat.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/radsec_server_test.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/radsec_server_test.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F003`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Perluas kontrak tes tentang verifikasi peers, wrong CA dan perubahan TLS config berdasarkan implementasi aktual.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F003` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/radsec_server_test.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Perluas kontrak tes tentang verifikasi peers, wrong CA dan perubahan TLS config berdasarkan implementasi aktual.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Handshake negative/positive matrix hijau dan tidak melonggarkan kebijakan sertifikat.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Handshake negative/positive matrix hijau dan tidak melonggarkan kebijakan sertifikat.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-004.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F003` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Handshake negative/positive matrix hijau dan tidak melonggarkan kebijakan sertifikat. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F003`. Scope only: **Perluas kontrak tes tentang verifikasi peers, wrong CA dan perubahan TLS config berdasarkan implementasi aktual.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Handshake negative/positive matrix hijau dan tidak melonggarkan kebijakan sertifikat.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-010 — Audit duplikasi RADIUS retransmission dan replay

## 01 Identity
- ID: `ENG-010`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P1**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Peta cache replay per NAS/tenant dan determinisme duplicate Access-Request saat timeout/retry.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F001` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit duplikasi radius retransmission dan replay**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Protokol retransmission berbasis fixture memastikan tidak ada lintas tenant atau efek ganda.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/radius_auth.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/radius_auth.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F001`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Peta cache replay per NAS/tenant dan determinisme duplicate Access-Request saat timeout/retry.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F001` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/radius_auth.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Peta cache replay per NAS/tenant dan determinisme duplicate Access-Request saat timeout/retry.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Protokol retransmission berbasis fixture memastikan tidak ada lintas tenant atau efek ganda.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Protokol retransmission berbasis fixture memastikan tidak ada lintas tenant atau efek ganda.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-011.
- **REUSES:** `TR-F001` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Protokol retransmission berbasis fixture memastikan tidak ada lintas tenant atau efek ganda. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F001`. Scope only: **Peta cache replay per NAS/tenant dan determinisme duplicate Access-Request saat timeout/retry.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Protokol retransmission berbasis fixture memastikan tidak ada lintas tenant atau efek ganda.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-011 — Harness fuzz paket RADIUS cacat dan oversized

## 01 Identity
- ID: `ENG-011`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Terapkan fuzz regression pada parsing atribut, length, vendor TLV dan Message-Authenticator.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F001` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **harness fuzz paket radius cacat dan oversized**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Corpus fuzz menghasilkan nol panic dan rejection sesuai kontrak pada anggaran uji tetap.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/packet_format.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/packet_format.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F001`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Terapkan fuzz regression pada parsing atribut, length, vendor TLV dan Message-Authenticator.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F001` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/packet_format.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Terapkan fuzz regression pada parsing atribut, length, vendor TLV dan Message-Authenticator.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Corpus fuzz menghasilkan nol panic dan rejection sesuai kontrak pada anggaran uji tetap.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Corpus fuzz menghasilkan nol panic dan rejection sesuai kontrak pada anggaran uji tetap.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-010.
- **ENABLES:** FUT-011.
- **REUSES:** `TR-F001` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Corpus fuzz menghasilkan nol panic dan rejection sesuai kontrak pada anggaran uji tetap. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F001`. Scope only: **Terapkan fuzz regression pada parsing atribut, length, vendor TLV dan Message-Authenticator.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Corpus fuzz menghasilkan nol panic dan rejection sesuai kontrak pada anggaran uji tetap.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-012 — Test accounting Start/Interim/Stop out-of-order

## 01 Identity
- ID: `ENG-012`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tambahkan fixture untuk accounting tertunda, duplicate dan session-id reuse tanpa merusak histori.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F002` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **test accounting start/interim/stop out-of-order**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Session online/riwayat konvergen benar setelah urutan pesan yang dipermutasi.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/radius_acct.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/radius_acct.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F002`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Tambahkan fixture untuk accounting tertunda, duplicate dan session-id reuse tanpa merusak histori.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F002` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/radius_acct.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tambahkan fixture untuk accounting tertunda, duplicate dan session-id reuse tanpa merusak histori.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Session online/riwayat konvergen benar setelah urutan pesan yang dipermutasi.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Session online/riwayat konvergen benar setelah urutan pesan yang dipermutasi.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F002` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Session online/riwayat konvergen benar setelah urutan pesan yang dipermutasi. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F002`. Scope only: **Tambahkan fixture untuk accounting tertunda, duplicate dan session-id reuse tanpa merusak histori.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Session online/riwayat konvergen benar setelah urutan pesan yang dipermutasi.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-013 — Pilot RFC 5176 CoA/Disconnect pada NAS nyata

## 01 Identity
- ID: `ENG-013`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **87/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Jalankan acceptance nonproduksi untuk ACK, NAK, timeout dan retry pada model NAS operator yang benar-benar dipakai.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F010` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pilot rfc 5176 coa/disconnect pada nas nyata**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Evidence log tersanitasi dan matrix firmware + efek sesi tercatat, tanpa klaim dari fake NAS saja.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/session-action-queue.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/session-action-queue.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F010`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Jalankan acceptance nonproduksi untuk ACK, NAK, timeout dan retry pada model NAS operator yang benar-benar dipakai.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F010` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/session-action-queue.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Jalankan acceptance nonproduksi untuk ACK, NAK, timeout dan retry pada model NAS operator yang benar-benar dipakai.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Evidence log tersanitasi dan matrix firmware + efek sesi tercatat, tanpa klaim dari fake NAS saja.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Evidence log tersanitasi dan matrix firmware + efek sesi tercatat, tanpa klaim dari fake NAS saja.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-014, ENG-050, ENG-064, FUT-008.
- **REUSES:** `TR-F010` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Evidence log tersanitasi dan matrix firmware + efek sesi tercatat, tanpa klaim dari fake NAS saja. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F010`. Scope only: **Jalankan acceptance nonproduksi untuk ACK, NAK, timeout dan retry pada model NAS operator yang benar-benar dipakai.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Evidence log tersanitasi dan matrix firmware + efek sesi tercatat, tanpa klaim dari fake NAS saja.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-014 — Prioritas vendor VSA berbasis model NAS riil

## 01 Identity
- ID: `ENG-014`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Survei vendor yang dipakai operator dan matriks packet capture; jangan tambah enhancer hanya karena daftar vendor panjang.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F005` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **prioritas vendor vsa berbasis model nas riil**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Setiap prioritas memiliki sample atribut, pemilik, outcome dan keputusan implement/skip.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/vendor-vsa-gap-baseline.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/vendor-vsa-gap-baseline.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F005`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Survei vendor yang dipakai operator dan matriks packet capture; jangan tambah enhancer hanya karena daftar vendor panjang.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F005` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/vendor-vsa-gap-baseline.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Survei vendor yang dipakai operator dan matriks packet capture; jangan tambah enhancer hanya karena daftar vendor panjang.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Setiap prioritas memiliki sample atribut, pemilik, outcome dan keputusan implement/skip.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Setiap prioritas memiliki sample atribut, pemilik, outcome dan keputusan implement/skip.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-013.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F005` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Setiap prioritas memiliki sample atribut, pemilik, outcome dan keputusan implement/skip. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F005`. Scope only: **Survei vendor yang dipakai operator dan matriks packet capture; jangan tambah enhancer hanya karena daftar vendor panjang.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Setiap prioritas memiliki sample atribut, pemilik, outcome dan keputusan implement/skip.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-015 — Profil biaya LDAP bind di bawah concurrent load

## 01 Identity
- ID: `ENG-015`; domain: **Engineering Excellence**; primary subdomain: **performance**; type: `performance`.
- Priority: **P2**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:performance`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Ukur overhead koneksi, saturasi dan cancellation sebelum menyentuh pooling M14.5 yang blocked.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F025` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **profil biaya ldap bind di bawah concurrent load**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Benchmark cold/warm yang reproduktif dan ambang keputusan dicatat; belum ada pooling spekulatif.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/ldapauth/ldapauth.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/ldapauth/ldapauth.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F025`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **measurement and benchmarking**. Work only within `Ukur overhead koneksi, saturasi dan cancellation sebelum menyentuh pooling M14.5 yang blocked.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F025` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/ldapauth/ldapauth.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Ukur overhead koneksi, saturasi dan cancellation sebelum menyentuh pooling M14.5 yang blocked.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Benchmark cold/warm yang reproduktif dan ambang keputusan dicatat; belum ada pooling spekulatif.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Benchmark cold/warm yang reproduktif dan ambang keputusan dicatat; belum ada pooling spekulatif.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F025` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Benchmark cold/warm yang reproduktif dan ambang keputusan dicatat; belum ada pooling spekulatif. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F025`. Scope only: **Ukur overhead koneksi, saturasi dan cancellation sebelum menyentuh pooling M14.5 yang blocked.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Benchmark cold/warm yang reproduktif dan ambang keputusan dicatat; belum ada pooling spekulatif.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-016 — Pemeriksaan berkala perbedaan fork layeh dan upstream

## 01 Identity
- ID: `ENG-016`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **71/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Definisikan kriteria perubahan keamanan/encoding yang pantas disinkronkan dengan fork, bukan sync tanpa alasan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F021` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pemeriksaan berkala perbedaan fork layeh dan upstream**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Laporan SHA upstream/fork, matrix relevansi dan keputusan no-change/change yang dapat diaudit.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F021`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Definisikan kriteria perubahan keamanan/encoding yang pantas disinkronkan dengan fork, bukan sync tanpa alasan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F021` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/roadmap.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Definisikan kriteria perubahan keamanan/encoding yang pantas disinkronkan dengan fork, bukan sync tanpa alasan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Laporan SHA upstream/fork, matrix relevansi dan keputusan no-change/change yang dapat diaudit.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Laporan SHA upstream/fork, matrix relevansi dan keputusan no-change/change yang dapat diaudit.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** FUT-007, FUT-023.
- **REUSES:** `TR-F021` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Laporan SHA upstream/fork, matrix relevansi dan keputusan no-change/change yang dapat diaudit. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F021`. Scope only: **Definisikan kriteria perubahan keamanan/encoding yang pantas disinkronkan dengan fork, bukan sync tanpa alasan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Laporan SHA upstream/fork, matrix relevansi dan keputusan no-change/change yang dapat diaudit.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-017 — Audit penurunan hak anggota tenant saat token lama aktif

## 01 Identity
- ID: `ENG-017`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **87/100 provisional** (planning heuristic, not observed performance); recommended wave: **0**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Susun regresi setiap peralihan operator-admin-super dan revoked membership pada API mutasi penting.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit penurunan hak anggota tenant saat token lama aktif**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Peran lama langsung kehilangan akses dan test membuktikan cross-tenant 404/403 tanpa side effect.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/access-control-matrix.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/access-control-matrix.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Susun regresi setiap peralihan operator-admin-super dan revoked membership pada API mutasi penting.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/access-control-matrix.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Susun regresi setiap peralihan operator-admin-super dan revoked membership pada API mutasi penting.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Peran lama langsung kehilangan akses dan test membuktikan cross-tenant 404/403 tanpa side effect.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Peran lama langsung kehilangan akses dan test membuktikan cross-tenant 404/403 tanpa side effect.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-018, ENG-052, FUT-020.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Peran lama langsung kehilangan akses dan test membuktikan cross-tenant 404/403 tanpa side effect. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Susun regresi setiap peralihan operator-admin-super dan revoked membership pada API mutasi penting.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Peran lama langsung kehilangan akses dan test membuktikan cross-tenant 404/403 tanpa side effect.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-018 — Konsistensi tenant untuk NAS ambigu dan username sama

## 01 Identity
- ID: `ENG-018`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **0**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun fixture NAS tak dikenali/ambigu dan nama pelanggan sama lintas tenant pada RADIUS/EAP.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **konsistensi tenant untuk nas ambigu dan username sama**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Setiap NAS ambigu fail-closed; aktivitas tenant A tidak mengubah tenant B.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/radius_auth.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/radius_auth.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Bangun fixture NAS tak dikenali/ambigu dan nama pelanggan sama lintas tenant pada RADIUS/EAP.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/radius_auth.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun fixture NAS tak dikenali/ambigu dan nama pelanggan sama lintas tenant pada RADIUS/EAP.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Setiap NAS ambigu fail-closed; aktivitas tenant A tidak mengubah tenant B.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Setiap NAS ambigu fail-closed; aktivitas tenant A tidak mengubah tenant B.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-017.
- **ENABLES:** FUT-006.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Setiap NAS ambigu fail-closed; aktivitas tenant A tidak mengubah tenant B. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Bangun fixture NAS tak dikenali/ambigu dan nama pelanggan sama lintas tenant pada RADIUS/EAP.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Setiap NAS ambigu fail-closed; aktivitas tenant A tidak mengubah tenant B.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-019 — Ratcheting audit query SQL raw dan joins tenancy

## 01 Identity
- ID: `ENG-019`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **85/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Buat pemeriksaan diff atas titik Table/Raw/Exec/Joins baru agar wajib ada tenant predicate / exception.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **ratcheting audit query sql raw dan joins tenancy**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **CI mendeteksi satu fixture query yang tidak terscope dan melewatkan query valid.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/tenant-isolation.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/tenant-isolation.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Buat pemeriksaan diff atas titik Table/Raw/Exec/Joins baru agar wajib ada tenant predicate / exception.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/tenant-isolation.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat pemeriksaan diff atas titik Table/Raw/Exec/Joins baru agar wajib ada tenant predicate / exception.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **CI mendeteksi satu fixture query yang tidak terscope dan melewatkan query valid.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** CI mendeteksi satu fixture query yang tidak terscope dan melewatkan query valid.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-036, FUT-010.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: CI mendeteksi satu fixture query yang tidak terscope dan melewatkan query valid. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Buat pemeriksaan diff atas titik Table/Raw/Exec/Joins baru agar wajib ada tenant predicate / exception.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **CI mendeteksi satu fixture query yang tidak terscope dan melewatkan query valid.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-020 — Latihan restore lintas tenant setelah perubahan skema

## 01 Identity
- ID: `ENG-020`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Pertahankan cakupan drill H06 dan tambah tes skema baru untuk kepemilikan tenant pasca restore.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **latihan restore lintas tenant setelah perubahan skema**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Row ownership dua tenant konsisten; payload salah tenant ditolak dan rollback atomik.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/backup-restore-drill-2026-10-27.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/backup-restore-drill-2026-10-27.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Pertahankan cakupan drill H06 dan tambah tes skema baru untuk kepemilikan tenant pasca restore.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/backup-restore-drill-2026-10-27.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Pertahankan cakupan drill H06 dan tambah tes skema baru untuk kepemilikan tenant pasca restore.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Row ownership dua tenant konsisten; payload salah tenant ditolak dan rollback atomik.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Row ownership dua tenant konsisten; payload salah tenant ditolak dan rollback atomik.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Row ownership dua tenant konsisten; payload salah tenant ditolak dan rollback atomik. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Pertahankan cakupan drill H06 dan tambah tes skema baru untuk kepemilikan tenant pasca restore.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Row ownership dua tenant konsisten; payload salah tenant ditolak dan rollback atomik.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-021 — Rekonsiliasi landing/portal publik yang faktanya JWT-only

## 01 Identity
- ID: `ENG-021`; domain: **Engineering Excellence**; primary subdomain: **bug**; type: `bug`.
- Priority: **P1**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **0**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:bug`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit tautan publik /home /portal yang mengiklankan aksi yang gagal tanpa JWT dan tandai operator-only secara jujur.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F018` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **rekonsiliasi landing/portal publik yang faktanya jwt-only**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Pengguna anonim tidak melihat klaim checkout/login yang tidak ada; API tetap fail-closed.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F018`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **reproduction + targeted regression**. Work only within `Audit tautan publik /home /portal yang mengiklankan aksi yang gagal tanpa JWT dan tandai operator-only secara jujur.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F018` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/feature-checklist.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit tautan publik /home /portal yang mengiklankan aksi yang gagal tanpa JWT dan tandai operator-only secara jujur.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Pengguna anonim tidak melihat klaim checkout/login yang tidak ada; API tetap fail-closed.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Pengguna anonim tidak melihat klaim checkout/login yang tidak ada; API tetap fail-closed.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-056.
- **REUSES:** `TR-F018` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Pengguna anonim tidak melihat klaim checkout/login yang tidak ada; API tetap fail-closed. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F018`. Scope only: **Audit tautan publik /home /portal yang mengiklankan aksi yang gagal tanpa JWT dan tandai operator-only secara jujur.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Pengguna anonim tidak melihat klaim checkout/login yang tidak ada; API tetap fail-closed.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-022 — Threat-model autentikasi browser dan sesi admin

## 01 Identity
- ID: `ENG-022`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji token refresh/logout, penyimpanan sesi browser, pinning origin dan pilihan mitigasi CSRF bila applicable.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F016` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **threat-model autentikasi browser dan sesi admin**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Threat model dan tes abuse per skenario yang memang relevan; tidak asal menandai CSRF exploitable.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/auth.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/auth.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F016`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Uji token refresh/logout, penyimpanan sesi browser, pinning origin dan pilihan mitigasi CSRF bila applicable.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F016` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/auth.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji token refresh/logout, penyimpanan sesi browser, pinning origin dan pilihan mitigasi CSRF bila applicable.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Threat model dan tes abuse per skenario yang memang relevan; tidak asal menandai CSRF exploitable.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Threat model dan tes abuse per skenario yang memang relevan; tidak asal menandai CSRF exploitable.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F016` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Threat model dan tes abuse per skenario yang memang relevan; tidak asal menandai CSRF exploitable. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F016`. Scope only: **Uji token refresh/logout, penyimpanan sesi browser, pinning origin dan pilihan mitigasi CSRF bila applicable.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Threat model dan tes abuse per skenario yang memang relevan; tidak asal menandai CSRF exploitable.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-023 — Validasi rate limiting saat reverse proxy multi-hop

## 01 Identity
- ID: `ENG-023`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji spoofing X-Forwarded-For dan trusted proxy berdasarkan konfigurasi aktual.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F018` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **validasi rate limiting saat reverse proxy multi-hop**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Rate limit tak dapat dihindari via header palsu; tenant/actor sah tidak saling menghabiskan kuota.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/ratelimit.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/ratelimit.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F018`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji spoofing X-Forwarded-For dan trusted proxy berdasarkan konfigurasi aktual.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F018` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/ratelimit.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji spoofing X-Forwarded-For dan trusted proxy berdasarkan konfigurasi aktual.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Rate limit tak dapat dihindari via header palsu; tenant/actor sah tidak saling menghabiskan kuota.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Rate limit tak dapat dihindari via header palsu; tenant/actor sah tidak saling menghabiskan kuota.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F018` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Rate limit tak dapat dihindari via header palsu; tenant/actor sah tidak saling menghabiskan kuota. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F018`. Scope only: **Uji spoofing X-Forwarded-For dan trusted proxy berdasarkan konfigurasi aktual.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Rate limit tak dapat dihindari via header palsu; tenant/actor sah tidak saling menghabiskan kuota.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-024 — Runbook rotasi secret SNMP dan encrypted community

## 01 Identity
- ID: `ENG-024`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji konsekuensi rotasi stable web secret pada SNMPv2c ciphertext beserta backup dan cutover.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **runbook rotasi secret snmp dan encrypted community**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dry run menunjukkan data dapat dire-encrypt dengan rollback dan tidak bocor ke log.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/networkmonitor/monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/networkmonitor/monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Uji konsekuensi rotasi stable web secret pada SNMPv2c ciphertext beserta backup dan cutover.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/networkmonitor/monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji konsekuensi rotasi stable web secret pada SNMPv2c ciphertext beserta backup dan cutover.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dry run menunjukkan data dapat dire-encrypt dengan rollback dan tidak bocor ke log.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dry run menunjukkan data dapat dire-encrypt dengan rollback dan tidak bocor ke log.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dry run menunjukkan data dapat dire-encrypt dengan rollback dan tidak bocor ke log. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Uji konsekuensi rotasi stable web secret pada SNMPv2c ciphertext beserta backup dan cutover.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dry run menunjukkan data dapat dire-encrypt dengan rollback dan tidak bocor ke log.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-025 — Audit perlindungan kredensial koneksi WhatsApp

## 01 Identity
- ID: `ENG-025`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Inventaris lokasi session metadata whatsmeow, log, backup, file permission dan revoke.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F030` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit perlindungan kredensial koneksi whatsapp**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Prosedur hilangkan akses sesi dicoba di test environment; tidak ada plaintext session dalam log.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/notify/whatsapp.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/notify/whatsapp.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F030`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Inventaris lokasi session metadata whatsmeow, log, backup, file permission dan revoke.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F030` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/notify/whatsapp.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Inventaris lokasi session metadata whatsmeow, log, backup, file permission dan revoke.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Prosedur hilangkan akses sesi dicoba di test environment; tidak ada plaintext session dalam log.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Prosedur hilangkan akses sesi dicoba di test environment; tidak ada plaintext session dalam log.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F030` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Prosedur hilangkan akses sesi dicoba di test environment; tidak ada plaintext session dalam log. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F030`. Scope only: **Inventaris lokasi session metadata whatsmeow, log, backup, file permission dan revoke.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Prosedur hilangkan akses sesi dicoba di test environment; tidak ada plaintext session dalam log.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-026 — Evaluasi parameter KDF dan envelope backup

## 01 Identity
- ID: `ENG-026`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Ukur biaya PBKDF2, versi envelope, kompatibilitas restore lama dan password policy.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **evaluasi parameter kdf dan envelope backup**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Benchmark dan compat tests mendukung pilihan KDF; tidak mengubah format tanpa migration.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/backup_envelope.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/backup_envelope.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Ukur biaya PBKDF2, versi envelope, kompatibilitas restore lama dan password policy.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/backup_envelope.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Ukur biaya PBKDF2, versi envelope, kompatibilitas restore lama dan password policy.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Benchmark dan compat tests mendukung pilihan KDF; tidak mengubah format tanpa migration.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Benchmark dan compat tests mendukung pilihan KDF; tidak mengubah format tanpa migration.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Benchmark dan compat tests mendukung pilihan KDF; tidak mengubah format tanpa migration. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Ukur biaya PBKDF2, versi envelope, kompatibilitas restore lama dan password policy.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Benchmark dan compat tests mendukung pilihan KDF; tidak mengubah format tanpa migration.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-027 — Hardening diagnostik jaringan terhadap SSRF

## 01 Identity
- ID: `ENG-027`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit input host/port dan resolusi DNS target eksplisit terhadap private ranges tak diizinkan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **hardening diagnostik jaringan terhadap ssrf**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Skenario DNS rebinding/malformed URL diputus sesuai kebijakan; tidak ada scanner umum.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/network_monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/network_monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Audit input host/port dan resolusi DNS target eksplisit terhadap private ranges tak diizinkan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/network_monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit input host/port dan resolusi DNS target eksplisit terhadap private ranges tak diizinkan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Skenario DNS rebinding/malformed URL diputus sesuai kebijakan; tidak ada scanner umum.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Skenario DNS rebinding/malformed URL diputus sesuai kebijakan; tidak ada scanner umum.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Skenario DNS rebinding/malformed URL diputus sesuai kebijakan; tidak ada scanner umum. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Audit input host/port dan resolusi DNS target eksplisit terhadap private ranges tak diizinkan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Skenario DNS rebinding/malformed URL diputus sesuai kebijakan; tidak ada scanner umum.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-028 — Penyamaran PII di logs, traces dan support evidence

## 01 Identity
- ID: `ENG-028`; domain: **Engineering Excellence**; primary subdomain: **security**; type: `security`.
- Priority: **P1**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **85/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:security`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tentukan redaction tenant/customer/password/token/IP sesuai kelas data sebelum log diekspor.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F022` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **penyamaran pii di logs, traces dan support evidence**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Fixture request sensitif tidak muncul mentah di output redacted dan audit trail tetap dapat dikaitkan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/system_logs.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/system_logs.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F022`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **threat model + negative-case tests**. Work only within `Tentukan redaction tenant/customer/password/token/IP sesuai kelas data sebelum log diekspor.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F022` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/system_logs.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tentukan redaction tenant/customer/password/token/IP sesuai kelas data sebelum log diekspor.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Fixture request sensitif tidak muncul mentah di output redacted dan audit trail tetap dapat dikaitkan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Fixture request sensitif tidak muncul mentah di output redacted dan audit trail tetap dapat dikaitkan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-060, FUT-002.
- **REUSES:** `TR-F022` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Fixture request sensitif tidak muncul mentah di output redacted dan audit trail tetap dapat dikaitkan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F022`. Scope only: **Tentukan redaction tenant/customer/password/token/IP sesuai kelas data sebelum log diekspor.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Fixture request sensitif tidak muncul mentah di output redacted dan audit trail tetap dapat dikaitkan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-029 — PNG branding parser dan metadata validation

## 01 Identity
- ID: `ENG-029`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji content-type mismatch, oversized image, pixel bombs serta encoding PNG upload administrator.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **png branding parser dan metadata validation**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Upload non-PNG/terlalu besar ditolak; logo valid di-render aman dan owner global benar.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/branding.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/branding.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji content-type mismatch, oversized image, pixel bombs serta encoding PNG upload administrator.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/branding.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji content-type mismatch, oversized image, pixel bombs serta encoding PNG upload administrator.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Upload non-PNG/terlalu besar ditolak; logo valid di-render aman dan owner global benar.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Upload non-PNG/terlalu besar ditolak; logo valid di-render aman dan owner global benar.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Upload non-PNG/terlalu besar ditolak; logo valid di-render aman dan owner global benar. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **Uji content-type mismatch, oversized image, pixel bombs serta encoding PNG upload administrator.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Upload non-PNG/terlalu besar ditolak; logo valid di-render aman dan owner global benar.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-030 — Inventory SBOM backend/frontend serta lisensi

## 01 Identity
- ID: `ENG-030`; domain: **Engineering Excellence**; primary subdomain: **devops**; type: `devops`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **71/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:devops`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun inventaris dependency Go dan npm, scanner advisory dan license notice tanpa klaim CVE baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F022` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **inventory sbom backend/frontend serta lisensi**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **SBOM reproducible dan daftar temuan berstatus investigated bukan otomatis vulnerability.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`go.mod`](https://github.com/bjo163/mwx-isp/blob/main/go.mod); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F022`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **isolated automation verification**. Work only within `Bangun inventaris dependency Go dan npm, scanner advisory dan license notice tanpa klaim CVE baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F022` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `go.mod` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun inventaris dependency Go dan npm, scanner advisory dan license notice tanpa klaim CVE baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **SBOM reproducible dan daftar temuan berstatus investigated bukan otomatis vulnerability.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** SBOM reproducible dan daftar temuan berstatus investigated bukan otomatis vulnerability.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-062.
- **REUSES:** `TR-F022` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: SBOM reproducible dan daftar temuan berstatus investigated bukan otomatis vulnerability. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F022`. Scope only: **Bangun inventaris dependency Go dan npm, scanner advisory dan license notice tanpa klaim CVE baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **SBOM reproducible dan daftar temuan berstatus investigated bukan otomatis vulnerability.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-031 — Invoice numbering tahan retry setelah crash PostgreSQL

## 01 Identity
- ID: `ENG-031`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tambahkan fault-injection pada sequence dan transaction rollback yang sebelumnya telah diperbaiki.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **invoice numbering tahan retry setelah crash postgresql**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Uji concurrency dan restart membuktikan tak ada nomor ganda; nomor gap dicatat kebijakannya.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/billing/service.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/billing/service.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Tambahkan fault-injection pada sequence dan transaction rollback yang sebelumnya telah diperbaiki.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/billing/service.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tambahkan fault-injection pada sequence dan transaction rollback yang sebelumnya telah diperbaiki.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Uji concurrency dan restart membuktikan tak ada nomor ganda; nomor gap dicatat kebijakannya.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Uji concurrency dan restart membuktikan tak ada nomor ganda; nomor gap dicatat kebijakannya.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ECO-009.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Uji concurrency dan restart membuktikan tak ada nomor ganda; nomor gap dicatat kebijakannya. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Tambahkan fault-injection pada sequence dan transaction rollback yang sebelumnya telah diperbaiki.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Uji concurrency dan restart membuktikan tak ada nomor ganda; nomor gap dicatat kebijakannya.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-032 — Invariant suspend manual tidak tertimpa bayar otomatis

## 01 Identity
- ID: `ENG-032`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **87/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Perluas transisi manual-suspend, bill-suspend dan reactivation pada scheduler lintas invoice.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **invariant suspend manual tidak tertimpa bayar otomatis**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Pembayaran menyelesaikan billing suspension saja; manual suspension tetap berlaku.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/session-action-queue.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/session-action-queue.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Perluas transisi manual-suspend, bill-suspend dan reactivation pada scheduler lintas invoice.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/session-action-queue.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Perluas transisi manual-suspend, bill-suspend dan reactivation pada scheduler lintas invoice.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Pembayaran menyelesaikan billing suspension saja; manual suspension tetap berlaku.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Pembayaran menyelesaikan billing suspension saja; manual suspension tetap berlaku.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-035, FUT-009, FUT-022.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Pembayaran menyelesaikan billing suspension saja; manual suspension tetap berlaku. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Perluas transisi manual-suspend, bill-suspend dan reactivation pada scheduler lintas invoice.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Pembayaran menyelesaikan billing suspension saja; manual suspension tetap berlaku.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-033 — Idempotency pembayaran manual dengan request ulang

## 01 Identity
- ID: `ENG-033`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tentukan fingerprint/idempotency yang boleh dipakai untuk pencatatan manual non-gateway.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **idempotency pembayaran manual dengan request ulang**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dua retry identik tidak menambah paid amount; dua pembayaran berbeda tetap terpisah.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/billing/service.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/billing/service.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Tentukan fingerprint/idempotency yang boleh dipakai untuk pencatatan manual non-gateway.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/billing/service.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tentukan fingerprint/idempotency yang boleh dipakai untuk pencatatan manual non-gateway.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dua retry identik tidak menambah paid amount; dua pembayaran berbeda tetap terpisah.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dua retry identik tidak menambah paid amount; dua pembayaran berbeda tetap terpisah.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-034.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dua retry identik tidak menambah paid amount; dua pembayaran berbeda tetap terpisah. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Tentukan fingerprint/idempotency yang boleh dipakai untuk pencatatan manual non-gateway.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dua retry identik tidak menambah paid amount; dua pembayaran berbeda tetap terpisah.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-034 — Pembetulan transaksi payment operator yang salah input

## 01 Identity
- ID: `ENG-034`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tentukan kebijakan koreksi berbukti audit tanpa refund gateway/ERP yang tidak disetujui.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pembetulan transaksi payment operator yang salah input**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Matrix permutasi paid/partial/void terlindungi dengan role check dan jejak koreksi.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/isp.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/isp.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Tentukan kebijakan koreksi berbukti audit tanpa refund gateway/ERP yang tidak disetujui.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/isp.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tentukan kebijakan koreksi berbukti audit tanpa refund gateway/ERP yang tidak disetujui.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Matrix permutasi paid/partial/void terlindungi dengan role check dan jejak koreksi.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Matrix permutasi paid/partial/void terlindungi dengan role check dan jejak koreksi.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-033.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Matrix permutasi paid/partial/void terlindungi dengan role check dan jejak koreksi. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Tentukan kebijakan koreksi berbukti audit tanpa refund gateway/ERP yang tidak disetujui.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Matrix permutasi paid/partial/void terlindungi dengan role check dan jejak koreksi.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-035 — Konsistensi status Subscription dan RadiusUser saat transisi

## 01 Identity
- ID: `ENG-035`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Buat state-machine test untuk activate/suspend/terminate/relink di tengah kegagalan DB/NAS.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F027` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **konsistensi status subscription dan radiususer saat transisi**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Invariant langganan-akun dijaga; permintaan invalid ditolak tanpa sebagian perubahan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/isp_operations.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/isp_operations.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F027`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Buat state-machine test untuk activate/suspend/terminate/relink di tengah kegagalan DB/NAS.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F027` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/isp_operations.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat state-machine test untuk activate/suspend/terminate/relink di tengah kegagalan DB/NAS.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Invariant langganan-akun dijaga; permintaan invalid ditolak tanpa sebagian perubahan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Invariant langganan-akun dijaga; permintaan invalid ditolak tanpa sebagian perubahan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-032.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F027` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Invariant langganan-akun dijaga; permintaan invalid ditolak tanpa sebagian perubahan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F027`. Scope only: **Buat state-machine test untuk activate/suspend/terminate/relink di tengah kegagalan DB/NAS.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Invariant langganan-akun dijaga; permintaan invalid ditolak tanpa sebagian perubahan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-036 — Jejak scheduler FUP terpisah per tenant sesudah perbaikan

## 01 Identity
- ID: `ENG-036`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Instrumentasi ringan berbasis fixture untuk memastikan hasil #46 tak regresi ketika pekerjaan scheduler baru ditambah.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **jejak scheduler fup terpisah per tenant sesudah perbaikan**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Satu tenant gagal tidak memengaruhi tenant lain; metric tenant-tag aman cardinality.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/app/jobs.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/app/jobs.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Instrumentasi ringan berbasis fixture untuk memastikan hasil #46 tak regresi ketika pekerjaan scheduler baru ditambah.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/app/jobs.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Instrumentasi ringan berbasis fixture untuk memastikan hasil #46 tak regresi ketika pekerjaan scheduler baru ditambah.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Satu tenant gagal tidak memengaruhi tenant lain; metric tenant-tag aman cardinality.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Satu tenant gagal tidak memengaruhi tenant lain; metric tenant-tag aman cardinality.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-019.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Satu tenant gagal tidak memengaruhi tenant lain; metric tenant-tag aman cardinality. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Instrumentasi ringan berbasis fixture untuk memastikan hasil #46 tak regresi ketika pekerjaan scheduler baru ditambah.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Satu tenant gagal tidak memengaruhi tenant lain; metric tenant-tag aman cardinality.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-037 — Validasi integer IDR dan batas invoice items

## 01 Identity
- ID: `ENG-037`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit overflow, nilai negatif, pembulatan dan unit transaksi tanpa mengubah produk menjadi ERP.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **validasi integer idr dan batas invoice items**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Kasus over/underflow ditolak dan jumlah tagihan cocok matematis.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/domain/isp.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/domain/isp.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Audit overflow, nilai negatif, pembulatan dan unit transaksi tanpa mengubah produk menjadi ERP.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/domain/isp.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit overflow, nilai negatif, pembulatan dan unit transaksi tanpa mengubah produk menjadi ERP.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Kasus over/underflow ditolak dan jumlah tagihan cocok matematis.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Kasus over/underflow ditolak dan jumlah tagihan cocok matematis.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Kasus over/underflow ditolak dan jumlah tagihan cocok matematis. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Audit overflow, nilai negatif, pembulatan dan unit transaksi tanpa mengubah produk menjadi ERP.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Kasus over/underflow ditolak dan jumlah tagihan cocok matematis.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-038 — Voucher terminal-state concurrency regressions

## 01 Identity
- ID: `ENG-038`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji revoke bersamaan login/expiry setelah lifecycle yang sebelumnya delivered.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F007` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **voucher terminal-state concurrency regressions**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Tidak ada voucher terminal menjadi aktif lagi; credential linkage tidak merusak user lain.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/voucher-lifecycle.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/voucher-lifecycle.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F007`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji revoke bersamaan login/expiry setelah lifecycle yang sebelumnya delivered.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F007` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/voucher-lifecycle.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji revoke bersamaan login/expiry setelah lifecycle yang sebelumnya delivered.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Tidak ada voucher terminal menjadi aktif lagi; credential linkage tidak merusak user lain.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Tidak ada voucher terminal menjadi aktif lagi; credential linkage tidak merusak user lain.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F007` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Tidak ada voucher terminal menjadi aktif lagi; credential linkage tidak merusak user lain. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F007`. Scope only: **Uji revoke bersamaan login/expiry setelah lifecycle yang sebelumnya delivered.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Tidak ada voucher terminal menjadi aktif lagi; credential linkage tidak merusak user lain.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-039 — Perlindungan marker demo data saat clean-up

## 01 Identity
- ID: `ENG-039`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Verifikasi cleanup hanya menghapus row tagged demo pada tenant yang tepat meski terkait data riil.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F032` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **perlindungan marker demo data saat clean-up**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Data riil dan kredensial riil tidak tersentuh; operasi dapat diulang aman.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/demoseed/seeder.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/demoseed/seeder.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F032`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Verifikasi cleanup hanya menghapus row tagged demo pada tenant yang tepat meski terkait data riil.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F032` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/demoseed/seeder.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Verifikasi cleanup hanya menghapus row tagged demo pada tenant yang tepat meski terkait data riil.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Data riil dan kredensial riil tidak tersentuh; operasi dapat diulang aman.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Data riil dan kredensial riil tidak tersentuh; operasi dapat diulang aman.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F032` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Data riil dan kredensial riil tidak tersentuh; operasi dapat diulang aman. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F032`. Scope only: **Verifikasi cleanup hanya menghapus row tagged demo pada tenant yang tepat meski terkait data riil.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Data riil dan kredensial riil tidak tersentuh; operasi dapat diulang aman.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-040 — Studi archival accounting retention dan query plan

## 01 Identity
- ID: `ENG-040`; domain: **Engineering Excellence**; primary subdomain: **performance**; type: `performance`.
- Priority: **P2**; difficulty: Hard; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:performance`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Ambil EXPLAIN dan volume sintetis sebelum memutuskan partisi atau indeks baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F011` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **studi archival accounting retention dan query plan**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Profil latency/memory dan rencana no-change/index dibuat berdasar data.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/accounting.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/accounting.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F011`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **measurement and benchmarking**. Work only within `Ambil EXPLAIN dan volume sintetis sebelum memutuskan partisi atau indeks baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F011` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/accounting.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Ambil EXPLAIN dan volume sintetis sebelum memutuskan partisi atau indeks baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Profil latency/memory dan rencana no-change/index dibuat berdasar data.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Profil latency/memory dan rencana no-change/index dibuat berdasar data.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F011` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Profil latency/memory dan rencana no-change/index dibuat berdasar data. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F011`. Scope only: **Ambil EXPLAIN dan volume sintetis sebelum memutuskan partisi atau indeks baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Profil latency/memory dan rencana no-change/index dibuat berdasar data.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-041 — Uji instalasi fresh pada VPS Ubuntu/Debian bersih

## 01 Identity
- ID: `ENG-041`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **87/100 provisional** (planning heuristic, not observed performance); recommended wave: **0**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Jalankan matrix sistem terisolasi yang secara eksplisit belum dilakukan pada close-out H14.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **uji instalasi fresh pada vps ubuntu/debian bersih**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Install satu perintah berhasil pada dua versi OS yang didukung; bukti uptime dan rollback tersimpan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/installer-update-release-verification-2026-10-27.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/installer-update-release-verification-2026-10-27.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Jalankan matrix sistem terisolasi yang secara eksplisit belum dilakukan pada close-out H14.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/installer-update-release-verification-2026-10-27.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Jalankan matrix sistem terisolasi yang secara eksplisit belum dilakukan pada close-out H14.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Install satu perintah berhasil pada dua versi OS yang didukung; bukti uptime dan rollback tersimpan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Install satu perintah berhasil pada dua versi OS yang didukung; bukti uptime dan rollback tersimpan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-042, ENG-043, ENG-046, ENG-047.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Install satu perintah berhasil pada dua versi OS yang didukung; bukti uptime dan rollback tersimpan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Jalankan matrix sistem terisolasi yang secara eksplisit belum dilakukan pada close-out H14.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Install satu perintah berhasil pada dua versi OS yang didukung; bukti uptime dan rollback tersimpan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-042 — Simulasi kegagalan registry ketika upgrade aplikasi

## 01 Identity
- ID: `ENG-042`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji rollback image/container dan preservasi .env/database ketika pull gagal atau health check gagal.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **simulasi kegagalan registry ketika upgrade aplikasi**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Versi sebelumnya kembali ready dan volume tetap utuh pada fault injection.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`scripts/vps-update.sh`](https://github.com/bjo163/mwx-isp/blob/main/scripts/vps-update.sh); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji rollback image/container dan preservasi .env/database ketika pull gagal atau health check gagal.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `scripts/vps-update.sh` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji rollback image/container dan preservasi .env/database ketika pull gagal atau health check gagal.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Versi sebelumnya kembali ready dan volume tetap utuh pada fault injection.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Versi sebelumnya kembali ready dan volume tetap utuh pada fault injection.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-041.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Versi sebelumnya kembali ready dan volume tetap utuh pada fault injection. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Uji rollback image/container dan preservasi .env/database ketika pull gagal atau health check gagal.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Versi sebelumnya kembali ready dan volume tetap utuh pada fault injection.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-043 — Drill pemulihan seluruh volume VPS dari backup

## 01 Identity
- ID: `ENG-043`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **85/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Lengkapi drill application JSON H06 dengan data volume nyata tiruan dan restore prosedur host.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **drill pemulihan seluruh volume vps dari backup**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Data konfigurasi, postgres, sertifikat dan file dipulihkan dengan manifest; produksi tidak disentuh.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/backup-and-recovery.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/backup-and-recovery.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Lengkapi drill application JSON H06 dengan data volume nyata tiruan dan restore prosedur host.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/backup-and-recovery.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Lengkapi drill application JSON H06 dengan data volume nyata tiruan dan restore prosedur host.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Data konfigurasi, postgres, sertifikat dan file dipulihkan dengan manifest; produksi tidak disentuh.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Data konfigurasi, postgres, sertifikat dan file dipulihkan dengan manifest; produksi tidak disentuh.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-041.
- **ENABLES:** ENG-044, ENG-045.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Data konfigurasi, postgres, sertifikat dan file dipulihkan dengan manifest; produksi tidak disentuh. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Lengkapi drill application JSON H06 dengan data volume nyata tiruan dan restore prosedur host.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Data konfigurasi, postgres, sertifikat dan file dipulihkan dengan manifest; produksi tidak disentuh.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-044 — Studi PostgreSQL WAL PITR untuk RPO terukur

## 01 Identity
- ID: `ENG-044`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bandingkan backup snapshot dan WAL archiving dengan kebutuhan serta biaya ISP kecil.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F017` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **studi postgresql wal pitr untuk rpo terukur**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **ADR go/no-go dan percobaan restore titik waktu pada data dummy bila layak.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/backup-and-recovery.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/backup-and-recovery.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F017`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Bandingkan backup snapshot dan WAL archiving dengan kebutuhan serta biaya ISP kecil.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F017` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/backup-and-recovery.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bandingkan backup snapshot dan WAL archiving dengan kebutuhan serta biaya ISP kecil.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **ADR go/no-go dan percobaan restore titik waktu pada data dummy bila layak.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** ADR go/no-go dan percobaan restore titik waktu pada data dummy bila layak.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-043.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F017` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: ADR go/no-go dan percobaan restore titik waktu pada data dummy bila layak. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F017`. Scope only: **Bandingkan backup snapshot dan WAL archiving dengan kebutuhan serta biaya ISP kecil.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **ADR go/no-go dan percobaan restore titik waktu pada data dummy bila layak.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-045 — Tetapkan RTO/RPO operator berdasarkan rehearsal

## 01 Identity
- ID: `ENG-045`; domain: **Engineering Excellence**; primary subdomain: **documentation**; type: `documentation`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **73/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:documentation`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Pisahkan metrik lab dari janji produksi; buat lembar target dan pengukuran per konfigurasi.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **tetapkan rto/rpo operator berdasarkan rehearsal**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **RTO/RPO pilot berdasarkan hasil 3 drill atau diberi UNKNOWN; tidak menjanjikan SLA tanpa bukti.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/backup-restore-drill-2026-10-27.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/backup-restore-drill-2026-10-27.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **runbook/ADR verified walkthrough**. Work only within `Pisahkan metrik lab dari janji produksi; buat lembar target dan pengukuran per konfigurasi.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/backup-restore-drill-2026-10-27.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Pisahkan metrik lab dari janji produksi; buat lembar target dan pengukuran per konfigurasi.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **RTO/RPO pilot berdasarkan hasil 3 drill atau diberi UNKNOWN; tidak menjanjikan SLA tanpa bukti.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** RTO/RPO pilot berdasarkan hasil 3 drill atau diberi UNKNOWN; tidak menjanjikan SLA tanpa bukti.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-043.
- **ENABLES:** ECO-003, ECO-005.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: RTO/RPO pilot berdasarkan hasil 3 drill atau diberi UNKNOWN; tidak menjanjikan SLA tanpa bukti. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Pisahkan metrik lab dari janji produksi; buat lembar target dan pengukuran per konfigurasi.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **RTO/RPO pilot berdasarkan hasil 3 drill atau diberi UNKNOWN; tidak menjanjikan SLA tanpa bukti.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-046 — Uji akses localhost SSH tunnel pada instalasi tanpa DNS

## 01 Identity
- ID: `ENG-046`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Reproduksi fallback tanpa hostname publik dan error HTTP/TLS yang jujur.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **uji akses localhost ssh tunnel pada instalasi tanpa dns**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Akses admin berhasil via tunnel; tidak ada sertifikat publik fiktif.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`scripts/vps-install.sh`](https://github.com/bjo163/mwx-isp/blob/main/scripts/vps-install.sh); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Reproduksi fallback tanpa hostname publik dan error HTTP/TLS yang jujur.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `scripts/vps-install.sh` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Reproduksi fallback tanpa hostname publik dan error HTTP/TLS yang jujur.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Akses admin berhasil via tunnel; tidak ada sertifikat publik fiktif.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Akses admin berhasil via tunnel; tidak ada sertifikat publik fiktif.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-041.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Akses admin berhasil via tunnel; tidak ada sertifikat publik fiktif. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Reproduksi fallback tanpa hostname publik dan error HTTP/TLS yang jujur.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Akses admin berhasil via tunnel; tidak ada sertifikat publik fiktif.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-047 — Audit health dan peringatan expiry sertifikat deployment

## 01 Identity
- ID: `ENG-047`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Periksa bagaimana operator mendeteksi sertifikat mendekati expiry dan kesiapan TLS reverse proxy.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit health dan peringatan expiry sertifikat deployment**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Checklist expiry/renew/rollback dapat dieksekusi tanpa pengelolaan CA baru yang tidak perlu.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`Caddyfile`](https://github.com/bjo163/mwx-isp/blob/main/Caddyfile); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Periksa bagaimana operator mendeteksi sertifikat mendekati expiry dan kesiapan TLS reverse proxy.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `Caddyfile` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Periksa bagaimana operator mendeteksi sertifikat mendekati expiry dan kesiapan TLS reverse proxy.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Checklist expiry/renew/rollback dapat dieksekusi tanpa pengelolaan CA baru yang tidak perlu.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Checklist expiry/renew/rollback dapat dieksekusi tanpa pengelolaan CA baru yang tidak perlu.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-041.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Checklist expiry/renew/rollback dapat dieksekusi tanpa pengelolaan CA baru yang tidak perlu. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Periksa bagaimana operator mendeteksi sertifikat mendekati expiry dan kesiapan TLS reverse proxy.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Checklist expiry/renew/rollback dapat dieksekusi tanpa pengelolaan CA baru yang tidak perlu.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-048 — UAT WhatsApp pairing ke nomor uji opt-in

## 01 Identity
- ID: `ENG-048`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **71/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Lakukan test manual resmi operator dengan opt-in, tidak mass-message, tanpa data pelanggan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F030` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **uat whatsapp pairing ke nomor uji opt-in**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **QR, send, reconnect, logout dan dedup dibuktikan di sandbox nomor uji; jika tak tersedia tetap BLOCKED.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F030`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Lakukan test manual resmi operator dengan opt-in, tidak mass-message, tanpa data pelanggan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F030` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/feature-checklist.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Lakukan test manual resmi operator dengan opt-in, tidak mass-message, tanpa data pelanggan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **QR, send, reconnect, logout dan dedup dibuktikan di sandbox nomor uji; jika tak tersedia tetap BLOCKED.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** QR, send, reconnect, logout dan dedup dibuktikan di sandbox nomor uji; jika tak tersedia tetap BLOCKED.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ECO-007.
- **REUSES:** `TR-F030` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: QR, send, reconnect, logout dan dedup dibuktikan di sandbox nomor uji; jika tak tersedia tetap BLOCKED. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F030`. Scope only: **Lakukan test manual resmi operator dengan opt-in, tidak mass-message, tanpa data pelanggan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **QR, send, reconnect, logout dan dedup dibuktikan di sandbox nomor uji; jika tak tersedia tetap BLOCKED.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-049 — UAT SNMPv3 authPriv dengan perangkat sungguhan

## 01 Identity
- ID: `ENG-049`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **75/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Uji kontrak polling/interface counters terhadap satu switch/router yang diizinkan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **uat snmpv3 authpriv dengan perangkat sungguhan**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Polling, auth failure, timeout dan freshness dibuktikan; mock tidak dihitung sebagai real device.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/networkmonitor/monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/networkmonitor/monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Uji kontrak polling/interface counters terhadap satu switch/router yang diizinkan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/networkmonitor/monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji kontrak polling/interface counters terhadap satu switch/router yang diizinkan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Polling, auth failure, timeout dan freshness dibuktikan; mock tidak dihitung sebagai real device.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Polling, auth failure, timeout dan freshness dibuktikan; mock tidak dihitung sebagai real device.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ENG-055, FUT-004, FUT-019, FUT-024.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Polling, auth failure, timeout dan freshness dibuktikan; mock tidak dihitung sebagai real device. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Uji kontrak polling/interface counters terhadap satu switch/router yang diizinkan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Polling, auth failure, timeout dan freshness dibuktikan; mock tidak dihitung sebagai real device.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-050 — Matrix NAS/firmware teruji untuk RADIUS dan CoA

## 01 Identity
- ID: `ENG-050`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **73/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun matriks kapabilitas autentikasi, accounting, VLAN dan disconnect dari pilot riil, bukan penambahan vendor parser spekulatif.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F010` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **matrix nas/firmware teruji untuk radius dan coa**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Minimal satu vendor pada firmware teridentifikasi diuji; unsupported behaviors dilabeli.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F010`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Bangun matriks kapabilitas autentikasi, accounting, VLAN dan disconnect dari pilot riil, bukan penambahan vendor parser spekulatif.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F010` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/roadmap.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun matriks kapabilitas autentikasi, accounting, VLAN dan disconnect dari pilot riil, bukan penambahan vendor parser spekulatif.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Minimal satu vendor pada firmware teridentifikasi diuji; unsupported behaviors dilabeli.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Minimal satu vendor pada firmware teridentifikasi diuji; unsupported behaviors dilabeli.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-013.
- **ENABLES:** FUT-021, ECO-012.
- **REUSES:** `TR-F010` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Minimal satu vendor pada firmware teridentifikasi diuji; unsupported behaviors dilabeli. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F010`. Scope only: **Bangun matriks kapabilitas autentikasi, accounting, VLAN dan disconnect dari pilot riil, bukan penambahan vendor parser spekulatif.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Minimal satu vendor pada firmware teridentifikasi diuji; unsupported behaviors dilabeli.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-051 — Test fungsional website editor yang masih kurang coverage

## 01 Identity
- ID: `ENG-051`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Tambah uji save/publish/restore/read-only preview untuk editor yang tercantum tanpa dedicated test.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `VERIFIED_GAP`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **test fungsional website editor yang masih kurang coverage**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Semua mutasi hanya admin, publish tidak meloloskan script berbahaya, state dapat rollback.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/WebsiteEditorPage.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/WebsiteEditorPage.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Tambah uji save/publish/restore/read-only preview untuk editor yang tercantum tanpa dedicated test.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/WebsiteEditorPage.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tambah uji save/publish/restore/read-only preview untuk editor yang tercantum tanpa dedicated test.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Semua mutasi hanya admin, publish tidak meloloskan script berbahaya, state dapat rollback.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Semua mutasi hanya admin, publish tidak meloloskan script berbahaya, state dapat rollback.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Semua mutasi hanya admin, publish tidak meloloskan script berbahaya, state dapat rollback. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **Tambah uji save/publish/restore/read-only preview untuk editor yang tercantum tanpa dedicated test.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Semua mutasi hanya admin, publish tidak meloloskan script berbahaya, state dapat rollback.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-052 — E2E matriks tombol per role operator/admin/super

## 01 Identity
- ID: `ENG-052`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **83/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Turunkan browser spec dari server access control matrix; jangan mengganti enforcement backend dengan hide tombol.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F016` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **e2e matriks tombol per role operator/admin/super**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Role rendah tidak melihat aksi terlarang dan API langsung tetap menolak.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/access-control-matrix.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/access-control-matrix.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F016`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Turunkan browser spec dari server access control matrix; jangan mengganti enforcement backend dengan hide tombol.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F016` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/access-control-matrix.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Turunkan browser spec dari server access control matrix; jangan mengganti enforcement backend dengan hide tombol.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Role rendah tidak melihat aksi terlarang dan API langsung tetap menolak.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Role rendah tidak melihat aksi terlarang dan API langsung tetap menolak.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-017.
- **ENABLES:** FUT-001.
- **REUSES:** `TR-F016` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Role rendah tidak melihat aksi terlarang dan API langsung tetap menolak. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F016`. Scope only: **Turunkan browser spec dari server access control matrix; jangan mengganti enforcement backend dengan hide tombol.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Role rendah tidak melihat aksi terlarang dan API langsung tetap menolak.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-053 — Navigasi keyboard pada data-grid dan dialog kritis

## 01 Identity
- ID: `ENG-053`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **75/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit keyboard/focus pada customer, billing, NAS, confirmation dialog terhadap WCAG 2.2.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F013` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **navigasi keyboard pada data-grid dan dialog kritis**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Skenario tanpa mouse berhasil, focus tidak hilang, dan dialog kembali ke trigger.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/App.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/App.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F013`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Audit keyboard/focus pada customer, billing, NAS, confirmation dialog terhadap WCAG 2.2.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F013` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/App.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit keyboard/focus pada customer, billing, NAS, confirmation dialog terhadap WCAG 2.2.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Skenario tanpa mouse berhasil, focus tidak hilang, dan dialog kembali ke trigger.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Skenario tanpa mouse berhasil, focus tidak hilang, dan dialog kembali ke trigger.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** FUT-015, FUT-016, FUT-017.
- **REUSES:** `TR-F013` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Skenario tanpa mouse berhasil, focus tidak hilang, dan dialog kembali ke trigger. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F013`. Scope only: **Audit keyboard/focus pada customer, billing, NAS, confirmation dialog terhadap WCAG 2.2.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Skenario tanpa mouse berhasil, focus tidak hilang, dan dialog kembali ke trigger.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-054 — Rekonsiliasi ID/EN seluruh status tagihan dan insiden

## 01 Identity
- ID: `ENG-054`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Audit string dinamis backend, jam WIB dan angka IDR pada layar operator.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F023` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **rekonsiliasi id/en seluruh status tagihan dan insiden**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Tidak ada placeholder string kunci; switch bahasa tidak mengubah semantik status uang.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/i18n`](https://github.com/bjo163/mwx-isp/blob/main/web/src/i18n); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F023`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Audit string dinamis backend, jam WIB dan angka IDR pada layar operator.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F023` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/i18n` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Audit string dinamis backend, jam WIB dan angka IDR pada layar operator.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Tidak ada placeholder string kunci; switch bahasa tidak mengubah semantik status uang.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Tidak ada placeholder string kunci; switch bahasa tidak mengubah semantik status uang.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F023` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Tidak ada placeholder string kunci; switch bahasa tidak mengubah semantik status uang. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F023`. Scope only: **Audit string dinamis backend, jam WIB dan angka IDR pada layar operator.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Tidak ada placeholder string kunci; switch bahasa tidak mengubah semantik status uang.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-055 — State loading, stale-data dan retry pada OperationsPage

## 01 Identity
- ID: `ENG-055`; domain: **Engineering Excellence**; primary subdomain: **ui**; type: `ui`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **85/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:ui`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bedakan unknown/stale/down, dan hindari rendering red-alert dari data tidak tersedia.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **state loading, stale-data dan retry pada operationspage**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Simulasi timeout/offline memunculkan stale/unknown, bukan subscriber offline.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/OperationsPage.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/OperationsPage.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **frontend focused interaction audit**. Work only within `Bedakan unknown/stale/down, dan hindari rendering red-alert dari data tidak tersedia.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/OperationsPage.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bedakan unknown/stale/down, dan hindari rendering red-alert dari data tidak tersedia.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Simulasi timeout/offline memunculkan stale/unknown, bukan subscriber offline.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Simulasi timeout/offline memunculkan stale/unknown, bukan subscriber offline.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-049.
- **ENABLES:** ENG-058, FUT-013.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Simulasi timeout/offline memunculkan stale/unknown, bukan subscriber offline. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Bedakan unknown/stale/down, dan hindari rendering red-alert dari data tidak tersedia.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Simulasi timeout/offline memunculkan stale/unknown, bukan subscriber offline.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-056 — Informasi jujur untuk action portal/payment yang disabled

## 01 Identity
- ID: `ENG-056`; domain: **Engineering Excellence**; primary subdomain: **ui**; type: `ui`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:ui`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Ubah affordance yang menyesatkan, tampilkan boundary operator-only tanpa menciptakan customer-login baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **informasi jujur untuk action portal/payment yang disabled**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Klik anonim diarahkan ke info, bukan ke error tanpa konteks; API tidak dibuka publik.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/LandingPage.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/LandingPage.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **frontend focused interaction audit**. Work only within `Ubah affordance yang menyesatkan, tampilkan boundary operator-only tanpa menciptakan customer-login baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/LandingPage.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Ubah affordance yang menyesatkan, tampilkan boundary operator-only tanpa menciptakan customer-login baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Klik anonim diarahkan ke info, bukan ke error tanpa konteks; API tidak dibuka publik.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Klik anonim diarahkan ke info, bukan ke error tanpa konteks; API tidak dibuka publik.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-021.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Klik anonim diarahkan ke info, bukan ke error tanpa konteks; API tidak dibuka publik. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **Ubah affordance yang menyesatkan, tampilkan boundary operator-only tanpa menciptakan customer-login baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Klik anonim diarahkan ke info, bukan ke error tanpa konteks; API tidak dibuka publik.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-057 — Audit grafik dashboard konsistensi timezone dan nominal

## 01 Identity
- ID: `ENG-057`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bandingkan API/source query terhadap chart harian/bulanan dan mata uang integer.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F015` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit grafik dashboard konsistensi timezone dan nominal**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Tiga fixture date-boundary sesuai total laporan; missing data bukan angka nol palsu.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/Dashboard.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/Dashboard.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F015`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Bandingkan API/source query terhadap chart harian/bulanan dan mata uang integer.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F015` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/Dashboard.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bandingkan API/source query terhadap chart harian/bulanan dan mata uang integer.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Tiga fixture date-boundary sesuai total laporan; missing data bukan angka nol palsu.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Tiga fixture date-boundary sesuai total laporan; missing data bukan angka nol palsu.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F015` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Tiga fixture date-boundary sesuai total laporan; missing data bukan angka nol palsu. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F015`. Scope only: **Bandingkan API/source query terhadap chart harian/bulanan dan mata uang integer.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Tiga fixture date-boundary sesuai total laporan; missing data bukan angka nol palsu.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-058 — Label confidence dan mapping pada NOC impact UI

## 01 Identity
- ID: `ENG-058`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **73/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Buktikan tier confirmed/inferred/unknown terlihat per pelanggan tanpa inferensi offline palsu.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **label confidence dan mapping pada noc impact ui**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Kasus mapping lemah menampilkan UNKNOWN; pengujian regression menjaga perbedaan stale/down.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/components/NocImpactPanel.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/components/NocImpactPanel.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Buktikan tier confirmed/inferred/unknown terlihat per pelanggan tanpa inferensi offline palsu.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `web/src/components/NocImpactPanel.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buktikan tier confirmed/inferred/unknown terlihat per pelanggan tanpa inferensi offline palsu.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Kasus mapping lemah menampilkan UNKNOWN; pengujian regression menjaga perbedaan stale/down.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Kasus mapping lemah menampilkan UNKNOWN; pengujian regression menjaga perbedaan stale/down.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-055.
- **ENABLES:** FUT-003, FUT-005.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Kasus mapping lemah menampilkan UNKNOWN; pengujian regression menjaga perbedaan stale/down. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Buktikan tier confirmed/inferred/unknown terlihat per pelanggan tanpa inferensi offline palsu.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Kasus mapping lemah menampilkan UNKNOWN; pengujian regression menjaga perbedaan stale/down.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-059 — Audit import RADIUS user dengan preview dry-run saja

## 01 Identity
- ID: `ENG-059`; domain: **Engineering Excellence**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Periksa error per baris dan rollback di existing import; rancang preview hanya jika termasuk scope.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F007` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit import radius user dengan preview dry-run saja**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **File buruk ditolak tanpa perubahan; preview proposed tidak dideploy tanpa checklist approval.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/users_import.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/users_import.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F007`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Periksa error per baris dan rollback di existing import; rancang preview hanya jika termasuk scope.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F007` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/users_import.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Periksa error per baris dan rollback di existing import; rancang preview hanya jika termasuk scope.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **File buruk ditolak tanpa perubahan; preview proposed tidak dideploy tanpa checklist approval.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** File buruk ditolak tanpa perubahan; preview proposed tidak dideploy tanpa checklist approval.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ECO-006.
- **REUSES:** `TR-F007` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: File buruk ditolak tanpa perubahan; preview proposed tidak dideploy tanpa checklist approval. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F007`. Scope only: **Periksa error per baris dan rollback di existing import; rancang preview hanya jika termasuk scope.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **File buruk ditolak tanpa perubahan; preview proposed tidak dideploy tanpa checklist approval.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-060 — Studi support bundle tersensor untuk insiden operator

## 01 Identity
- ID: `ENG-060`; domain: **Engineering Excellence**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P2**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:experiment`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Rancang export minimal logs/config bebas secrets untuk perbaikan; jangan otomatis membagikan data pelanggan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F022` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **studi support bundle tersensor untuk insiden operator**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **POC lokal menghasilkan checksum/redaction report, user review wajib sebelum ekspor.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/MWX-ISP-blueprint.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/MWX-ISP-blueprint.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F022`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Rancang export minimal logs/config bebas secrets untuk perbaikan; jangan otomatis membagikan data pelanggan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F022` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `docs/MWX-ISP-blueprint.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Rancang export minimal logs/config bebas secrets untuk perbaikan; jangan otomatis membagikan data pelanggan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **POC lokal menghasilkan checksum/redaction report, user review wajib sebelum ekspor.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** POC lokal menghasilkan checksum/redaction report, user review wajib sebelum ekspor.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-028.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F022` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: POC lokal menghasilkan checksum/redaction report, user review wajib sebelum ekspor. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F022`. Scope only: **Rancang export minimal logs/config bebas secrets untuk perbaikan; jangan otomatis membagikan data pelanggan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **POC lokal menghasilkan checksum/redaction report, user review wajib sebelum ekspor.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-061 — Stabilitas test E2E CI branch main dan dev

## 01 Identity
- ID: `ENG-061`; domain: **Engineering Excellence**; primary subdomain: **devops**; type: `devops`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:devops`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Periksa flaky test, timeout, test DB isolation dan bukti lulus yang relevan pada tiap branch.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F022` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **stabilitas test e2e ci branch main dan dev**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Run >=3 kali per suite tanpa data cross-fixture dan report kegagalan dibagikan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`.github/workflows/ci.yml`](https://github.com/bjo163/mwx-isp/blob/main/.github/workflows/ci.yml); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F022`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **isolated automation verification**. Work only within `Periksa flaky test, timeout, test DB isolation dan bukti lulus yang relevan pada tiap branch.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F022` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `.github/workflows/ci.yml` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Periksa flaky test, timeout, test DB isolation dan bukti lulus yang relevan pada tiap branch.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Run >=3 kali per suite tanpa data cross-fixture dan report kegagalan dibagikan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Run >=3 kali per suite tanpa data cross-fixture dan report kegagalan dibagikan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F022` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Run >=3 kali per suite tanpa data cross-fixture dan report kegagalan dibagikan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F022`. Scope only: **Periksa flaky test, timeout, test DB isolation dan bukti lulus yang relevan pada tiap branch.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Run >=3 kali per suite tanpa data cross-fixture dan report kegagalan dibagikan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-062 — Sistem upgrade dependency dengan ratchet keamanan

## 01 Identity
- ID: `ENG-062`; domain: **Engineering Excellence**; primary subdomain: **devops**; type: `devops`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:devops`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Usulkan update cadence Go/npm + PR risk tiers setelah audit SBOM, tanpa mengulang issue #45 patch dependency.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F022` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **sistem upgrade dependency dengan ratchet keamanan**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Script gagal untuk advisory benar-benar applicable dan memberi jalur pengecualian beralasan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`.github/workflows/ci.yml`](https://github.com/bjo163/mwx-isp/blob/main/.github/workflows/ci.yml); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F022`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **isolated automation verification**. Work only within `Usulkan update cadence Go/npm + PR risk tiers setelah audit SBOM, tanpa mengulang issue #45 patch dependency.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F022` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `.github/workflows/ci.yml` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Usulkan update cadence Go/npm + PR risk tiers setelah audit SBOM, tanpa mengulang issue #45 patch dependency.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Script gagal untuk advisory benar-benar applicable dan memberi jalur pengecualian beralasan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Script gagal untuk advisory benar-benar applicable dan memberi jalur pengecualian beralasan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-030.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F022` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Script gagal untuk advisory benar-benar applicable dan memberi jalur pengecualian beralasan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F022`. Scope only: **Usulkan update cadence Go/npm + PR risk tiers setelah audit SBOM, tanpa mengulang issue #45 patch dependency.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Script gagal untuk advisory benar-benar applicable dan memberi jalur pengecualian beralasan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-063 — Contract testing respons API, paginasi dan error

## 01 Identity
- ID: `ENG-063`; domain: **Engineering Excellence**; primary subdomain: **test**; type: `test`.
- Priority: **P2**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p2`, `type:test`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun fixture contract untuk filter/sort/pagination dan error response yang React Admin pakai.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F012` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **contract testing respons api, paginasi dan error**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Contract lint menangkap breaking payload dan UI consumer tests lolos.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/responses.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/responses.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F012`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **fixture/test harness**. Work only within `Bangun fixture contract untuk filter/sort/pagination dan error response yang React Admin pakai.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F012` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/responses.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun fixture contract untuk filter/sort/pagination dan error response yang React Admin pakai.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Contract lint menangkap breaking payload dan UI consumer tests lolos.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Contract lint menangkap breaking payload dan UI consumer tests lolos.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Automated regression + controlled negative test. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F012` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Contract lint menangkap breaking payload dan UI consumer tests lolos. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F012`. Scope only: **Bangun fixture contract untuk filter/sort/pagination dan error response yang React Admin pakai.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Contract lint menangkap breaking payload dan UI consumer tests lolos.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ENG-064 — Pemeriksaan session-action queue dead-letter dan alert

## 01 Identity
- ID: `ENG-064`; domain: **Engineering Excellence**; primary subdomain: **observability**; type: `observability`.
- Priority: **P1**; difficulty: Medium; effort estimate: **M** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **High**.
- Execution score: **81/100 provisional** (planning heuristic, not observed performance); recommended wave: **1**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:engineering`, `priority:p1`, `type:observability`.
- Canonical owner: **MWX-ISP platform/engineering workstream**, specific accountable person not assigned.

## 02 Executive description
Fokus operasional observabilitas queue yang sudah ada (#52), bukan membangunnya ulang.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F010` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pemeriksaan session-action queue dead-letter dan alert**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Queue exhaust retry tampak dengan tenant-safe alert; replay manual tetap idempotent.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/sessionqueue`](https://github.com/bjo163/mwx-isp/blob/main/internal/sessionqueue); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F010`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **metrics/logs cardinality budget**. Work only within `Fokus operasional observabilitas queue yang sudah ada (#52), bukan membangunnya ulang.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F010` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **IN-SCOPE verification/hardening; preserve current TR-F boundary**. Must preserve existing checklist acceptance boundary.

## 09 Ordered microtasks
- [ ] Inspect `internal/sessionqueue` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Fokus operasional observabilitas queue yang sudah ada (#52), bukan membangunnya ulang.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Queue exhaust retry tampak dengan tenant-safe alert; replay manual tetap idempotent.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Queue exhaust retry tampak dengan tenant-safe alert; replay manual tetap idempotent.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-013.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F010` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: high possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Queue exhaust retry tampak dengan tenant-safe alert; replay manual tetap idempotent. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `M`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F010`. Scope only: **Fokus operasional observabilitas queue yang sudah ada (#52), bukan membangunnya ulang.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Queue exhaust retry tampak dengan tenant-safe alert; replay manual tetap idempotent.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.



---

# 04 Frontier 24

# FUT-001 — Proof-of-concept intent search untuk perintah operator read-only

## 01 Identity
- ID: `FUT-001`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Rancang parser intent terbatas untuk menuju menu/menyusun filter tanpa menjalankan mutasi NAS atau billing.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F013` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **proof-of-concept intent search untuk perintah operator read-only**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **20 prompt fixture mapped dengan alasan, salah tafsir tidak memicu side effect.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/App.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/App.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F013`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Rancang parser intent terbatas untuk menuju menu/menyusun filter tanpa menjalankan mutasi NAS atau billing.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F013` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/App.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Rancang parser intent terbatas untuk menuju menu/menyusun filter tanpa menjalankan mutasi NAS atau billing.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **20 prompt fixture mapped dengan alasan, salah tafsir tidak memicu side effect.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** 20 prompt fixture mapped dengan alasan, salah tafsir tidak memicu side effect.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-052.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F013` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: 20 prompt fixture mapped dengan alasan, salah tafsir tidak memicu side effect. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F013`. Scope only: **Rancang parser intent terbatas untuk menuju menu/menyusun filter tanpa menjalankan mutasi NAS atau billing.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **20 prompt fixture mapped dengan alasan, salah tafsir tidak memicu side effect.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-002 — AI incident triage berbasis log tersensor

## 01 Identity
- ID: `FUT-002`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Buat eksperimen klasifikasi akar masalah dari data sintetis dengan evidence backlinks dan opsi non-AI.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **ai incident triage berbasis log tersensor**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dataset label dan hallucination rate dilaporkan, no automatic remediate.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/system_logs.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/system_logs.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Buat eksperimen klasifikasi akar masalah dari data sintetis dengan evidence backlinks dan opsi non-AI.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/system_logs.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat eksperimen klasifikasi akar masalah dari data sintetis dengan evidence backlinks dan opsi non-AI.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dataset label dan hallucination rate dilaporkan, no automatic remediate.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dataset label dan hallucination rate dilaporkan, no automatic remediate.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-028.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dataset label dan hallucination rate dilaporkan, no automatic remediate. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Buat eksperimen klasifikasi akar masalah dari data sintetis dengan evidence backlinks dan opsi non-AI.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dataset label dan hallucination rate dilaporkan, no automatic remediate.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-003 — Ringkasan insiden berbukti tanpa mengklaim pelanggan offline

## 01 Identity
- ID: `FUT-003`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun template generator narasi deterministik per status freshness/mapping sebelum uji LLM.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **ringkasan insiden berbukti tanpa mengklaim pelanggan offline**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Setiap kalimat dipetakan ke API field; unknown tetap unknown.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/noc_impact.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/noc_impact.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Bangun template generator narasi deterministik per status freshness/mapping sebelum uji LLM.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/noc_impact.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun template generator narasi deterministik per status freshness/mapping sebelum uji LLM.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Setiap kalimat dipetakan ke API field; unknown tetap unknown.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Setiap kalimat dipetakan ke API field; unknown tetap unknown.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-058.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Setiap kalimat dipetakan ke API field; unknown tetap unknown. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Bangun template generator narasi deterministik per status freshness/mapping sebelum uji LLM.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Setiap kalimat dipetakan ke API field; unknown tetap unknown.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-004 — Deteksi pola anomali latency dari metrik historis bounded

## 01 Identity
- ID: `FUT-004`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Uji metode robust statistik pada data 30 hari dengan read-only flags tanpa NMS generik.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **deteksi pola anomali latency dari metrik historis bounded**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Benchmark precision/false positive pada synthetic labeled data + cost projection.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/networkmonitor/monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/networkmonitor/monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Uji metode robust statistik pada data 30 hari dengan read-only flags tanpa NMS generik.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/networkmonitor/monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Uji metode robust statistik pada data 30 hari dengan read-only flags tanpa NMS generik.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Benchmark precision/false positive pada synthetic labeled data + cost projection.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Benchmark precision/false positive pada synthetic labeled data + cost projection.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-049.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Benchmark precision/false positive pada synthetic labeled data + cost projection. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Uji metode robust statistik pada data 30 hari dengan read-only flags tanpa NMS generik.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Benchmark precision/false positive pada synthetic labeled data + cost projection.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-005 — Graph hipotesis penyebab insiden dengan uncertainty

## 01 Identity
- ID: `FUT-005`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Riset struktur edge NAS-monitor-subscriber dari relasi existing, bukan topologi scan aktif.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **graph hipotesis penyebab insiden dengan uncertainty**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Panel graf menandai confidence dan punya fallback tabel; tak mengarang link.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/noc_impact.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/noc_impact.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Riset struktur edge NAS-monitor-subscriber dari relasi existing, bukan topologi scan aktif.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/noc_impact.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Riset struktur edge NAS-monitor-subscriber dari relasi existing, bukan topologi scan aktif.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Panel graf menandai confidence dan punya fallback tabel; tak mengarang link.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Panel graf menandai confidence dan punya fallback tabel; tak mengarang link.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-058.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Panel graf menandai confidence dan punya fallback tabel; tak mengarang link. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Riset struktur edge NAS-monitor-subscriber dari relasi existing, bukan topologi scan aktif.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Panel graf menandai confidence dan punya fallback tabel; tak mengarang link.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-006 — Visual graph statis NAS-ODP dari relasi yang telah diisi operator

## 01 Identity
- ID: `FUT-006`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Eksperimen visualisasi read-only hubungan eksplisit yang sudah tersimpan tanpa discovery atau write router.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **visual graph statis nas-odp dari relasi yang telah diisi operator**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Perubahan data tenant B tak muncul ke tenant A dan edge unknown tidak dipaksakan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/domain/network.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/domain/network.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Eksperimen visualisasi read-only hubungan eksplisit yang sudah tersimpan tanpa discovery atau write router.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/domain/network.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Eksperimen visualisasi read-only hubungan eksplisit yang sudah tersimpan tanpa discovery atau write router.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Perubahan data tenant B tak muncul ke tenant A dan edge unknown tidak dipaksakan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Perubahan data tenant B tak muncul ke tenant A dan edge unknown tidak dipaksakan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-018.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Perubahan data tenant B tak muncul ke tenant A dan edge unknown tidak dipaksakan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Eksperimen visualisasi read-only hubungan eksplisit yang sudah tersimpan tanpa discovery atau write router.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Perubahan data tenant B tak muncul ke tenant A dan edge unknown tidak dipaksakan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-007 — Simulasi perubahan profil bandwidth tanpa menulis NAS

## 01 Identity
- ID: `FUT-007`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Bangun what-if estimator untuk akibat profil inheritance pada fixture pengguna, read-only saja.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F008` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **simulasi perubahan profil bandwidth tanpa menulis nas**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Output diff yang dapat direproduksi dan tidak memutasi cache atau profile.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/app/profile_cache.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/app/profile_cache.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F008`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Bangun what-if estimator untuk akibat profil inheritance pada fixture pengguna, read-only saja.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F008` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/app/profile_cache.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bangun what-if estimator untuk akibat profil inheritance pada fixture pengguna, read-only saja.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Output diff yang dapat direproduksi dan tidak memutasi cache atau profile.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Output diff yang dapat direproduksi dan tidak memutasi cache atau profile.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-016.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F008` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Output diff yang dapat direproduksi dan tidak memutasi cache atau profile. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F008`. Scope only: **Bangun what-if estimator untuk akibat profil inheritance pada fixture pengguna, read-only saja.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Output diff yang dapat direproduksi dan tidak memutasi cache atau profile.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-008 — Preview dampak CoA terhadap sesi yang aktif sebelum aksi

## 01 Identity
- ID: `FUT-008`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Buat dry-run kontrak daftar sesi dan policy yang terdampak, terpisah dari CoA sesungguhnya.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F010` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **preview dampak coa terhadap sesi yang aktif sebelum aksi**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dry run nol paket UDP ke NAS dan berbeda jelas dari Execute.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/coa_service.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/coa_service.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F010`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Buat dry-run kontrak daftar sesi dan policy yang terdampak, terpisah dari CoA sesungguhnya.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F010` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/coa_service.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat dry-run kontrak daftar sesi dan policy yang terdampak, terpisah dari CoA sesungguhnya.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dry run nol paket UDP ke NAS dan berbeda jelas dari Execute.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dry run nol paket UDP ke NAS dan berbeda jelas dari Execute.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-013.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F010` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dry run nol paket UDP ke NAS dan berbeda jelas dari Execute. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F010`. Scope only: **Buat dry-run kontrak daftar sesi dan policy yang terdampak, terpisah dari CoA sesungguhnya.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dry run nol paket UDP ke NAS dan berbeda jelas dari Execute.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-009 — Explanation trace billing-suspend dari rule engine existing

## 01 Identity
- ID: `FUT-009`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Prototype explainability graph reasons invoice/grace/manual suspension dari state yang ada.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **explanation trace billing-suspend dari rule engine existing**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Reason chain dapat ditautkan ke invoice dan tidak membuka customer lintas tenant.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/app/jobs.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/app/jobs.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Prototype explainability graph reasons invoice/grace/manual suspension dari state yang ada.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/app/jobs.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Prototype explainability graph reasons invoice/grace/manual suspension dari state yang ada.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Reason chain dapat ditautkan ke invoice dan tidak membuka customer lintas tenant.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Reason chain dapat ditautkan ke invoice dan tidak membuka customer lintas tenant.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-032.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Reason chain dapat ditautkan ke invoice dan tidak membuka customer lintas tenant. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Prototype explainability graph reasons invoice/grace/manual suspension dari state yang ada.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Reason chain dapat ditautkan ke invoice dan tidak membuka customer lintas tenant.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-010 — Static tenant-scope policy checker dari query AST sederhana

## 01 Identity
- ID: `FUT-010`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Riset analisis source pada query raw/joins untuk menandai penyimpangan semantik scope.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **static tenant-scope policy checker dari query ast sederhana**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Golden valid/invalid sample 100% dikenali tanpa false claim production guarantee.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/tenancy/scope.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/tenancy/scope.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Riset analisis source pada query raw/joins untuk menandai penyimpangan semantik scope.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/tenancy/scope.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Riset analisis source pada query raw/joins untuk menandai penyimpangan semantik scope.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Golden valid/invalid sample 100% dikenali tanpa false claim production guarantee.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Golden valid/invalid sample 100% dikenali tanpa false claim production guarantee.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-019.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Golden valid/invalid sample 100% dikenali tanpa false claim production guarantee. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Riset analisis source pada query raw/joins untuk menandai penyimpangan semantik scope.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Golden valid/invalid sample 100% dikenali tanpa false claim production guarantee.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-011 — Pembangkitan regression tests untuk paket RADIUS via mutator

## 01 Identity
- ID: `FUT-011`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Mutate length/tlv/authenticator secara offline membangun fixture baru yang mempertahankan oracle RFC.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F001` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pembangkitan regression tests untuk paket radius via mutator**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Minimum 20 kasus mutasi tersimpan sebagai reproducer, seluruhnya tidak panic.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/packet_format.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/packet_format.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F001`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Mutate length/tlv/authenticator secara offline membangun fixture baru yang mempertahankan oracle RFC.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F001` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/packet_format.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Mutate length/tlv/authenticator secara offline membangun fixture baru yang mempertahankan oracle RFC.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Minimum 20 kasus mutasi tersimpan sebagai reproducer, seluruhnya tidak panic.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Minimum 20 kasus mutasi tersimpan sebagai reproducer, seluruhnya tidak panic.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-011.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F001` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Minimum 20 kasus mutasi tersimpan sebagai reproducer, seluruhnya tidak panic. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F001`. Scope only: **Mutate length/tlv/authenticator secara offline membangun fixture baru yang mempertahankan oracle RFC.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Minimum 20 kasus mutasi tersimpan sebagai reproducer, seluruhnya tidak panic.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-012 — Eksperimen adversarial EAP TLS fragment scheduler

## 01 Identity
- ID: `FUT-012`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Gunakan state-machine model adversarial untuk duplicate/out-of-order fragment bukan sekadar fuzz byte.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F004` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **eksperimen adversarial eap tls fragment scheduler**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Model trace dapat direplay dan menunjukkan perbedaan accepted/rejected yang dijustifikasi.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/radiusd/plugins/eap/tlsfragment/fragment.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/radiusd/plugins/eap/tlsfragment/fragment.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F004`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Gunakan state-machine model adversarial untuk duplicate/out-of-order fragment bukan sekadar fuzz byte.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F004` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/radiusd/plugins/eap/tlsfragment/fragment.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Gunakan state-machine model adversarial untuk duplicate/out-of-order fragment bukan sekadar fuzz byte.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Model trace dapat direplay dan menunjukkan perbedaan accepted/rejected yang dijustifikasi.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Model trace dapat direplay dan menunjukkan perbedaan accepted/rejected yang dijustifikasi.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-002.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F004` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Model trace dapat direplay dan menunjukkan perbedaan accepted/rejected yang dijustifikasi. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F004`. Scope only: **Gunakan state-machine model adversarial untuk duplicate/out-of-order fragment bukan sekadar fuzz byte.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Model trace dapat direplay dan menunjukkan perbedaan accepted/rejected yang dijustifikasi.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-013 — Offline read-only NOC snapshot dengan TTL dan enkripsi

## 01 Identity
- ID: `FUT-013`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **52/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
POC cache terenkripsi lokal tanpa data pelanggan sensitif dan expiry yang jelas; jangan membuat offline mutation.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **offline read-only noc snapshot dengan ttl dan enkripsi**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Penerima melihat snapshot age, kesalahan freshness, clear cache; zero API mutation saat offline.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/OperationsPage.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/OperationsPage.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `POC cache terenkripsi lokal tanpa data pelanggan sensitif dan expiry yang jelas; jangan membuat offline mutation.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/OperationsPage.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **POC cache terenkripsi lokal tanpa data pelanggan sensitif dan expiry yang jelas; jangan membuat offline mutation.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Penerima melihat snapshot age, kesalahan freshness, clear cache; zero API mutation saat offline.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Penerima melihat snapshot age, kesalahan freshness, clear cache; zero API mutation saat offline.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-055.
- **ENABLES:** FUT-014.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Penerima melihat snapshot age, kesalahan freshness, clear cache; zero API mutation saat offline. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **POC cache terenkripsi lokal tanpa data pelanggan sensitif dan expiry yang jelas; jangan membuat offline mutation.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Penerima melihat snapshot age, kesalahan freshness, clear cache; zero API mutation saat offline.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-014 — Field diagnostics mini-mode untuk teknisi yang offline

## 01 Identity
- ID: `FUT-014`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Prototipe layar mobile untuk checklist terunduh dari target yang diotorisasi, tanpa remote router shell.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **field diagnostics mini-mode untuk teknisi yang offline**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dapat dibuka offline, tanpa penyimpanan secrets dan tanpa klaim live uptime.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/OperationsPage.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/OperationsPage.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Prototipe layar mobile untuk checklist terunduh dari target yang diotorisasi, tanpa remote router shell.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/OperationsPage.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Prototipe layar mobile untuk checklist terunduh dari target yang diotorisasi, tanpa remote router shell.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dapat dibuka offline, tanpa penyimpanan secrets dan tanpa klaim live uptime.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dapat dibuka offline, tanpa penyimpanan secrets dan tanpa klaim live uptime.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** FUT-013.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dapat dibuka offline, tanpa penyimpanan secrets dan tanpa klaim live uptime. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Prototipe layar mobile untuk checklist terunduh dari target yang diotorisasi, tanpa remote router shell.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dapat dibuka offline, tanpa penyimpanan secrets dan tanpa klaim live uptime.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-015 — Adaptive contextual workspace dari tugas operator

## 01 Identity
- ID: `FUT-015`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Research IA adaptive layout dari current customer/NOC tasks tanpa menambah aplikasi terpisah.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F013` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **adaptive contextual workspace dari tugas operator**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Task usability dibanding baseline; keyboard fallback setara.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/App.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/App.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F013`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Research IA adaptive layout dari current customer/NOC tasks tanpa menambah aplikasi terpisah.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F013` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/App.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Research IA adaptive layout dari current customer/NOC tasks tanpa menambah aplikasi terpisah.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Task usability dibanding baseline; keyboard fallback setara.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Task usability dibanding baseline; keyboard fallback setara.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-053.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F013` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Task usability dibanding baseline; keyboard fallback setara. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F013`. Scope only: **Research IA adaptive layout dari current customer/NOC tasks tanpa menambah aplikasi terpisah.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Task usability dibanding baseline; keyboard fallback setara.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-016 — Voice-assisted navigation untuk aksesibilitas internal

## 01 Identity
- ID: `FUT-016`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Eksperimen perintah navigasi tanpa backend listening permanen, tidak mengotomasi pembayaran.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F013` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **voice-assisted navigation untuk aksesibilitas internal**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Voice optional, salah transkripsi tidak membuat mutasi dan teks fallback ada.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/App.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/App.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F013`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Eksperimen perintah navigasi tanpa backend listening permanen, tidak mengotomasi pembayaran.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F013` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/App.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Eksperimen perintah navigasi tanpa backend listening permanen, tidak mengotomasi pembayaran.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Voice optional, salah transkripsi tidak membuat mutasi dan teks fallback ada.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Voice optional, salah transkripsi tidak membuat mutasi dan teks fallback ada.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-053.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F013` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Voice optional, salah transkripsi tidak membuat mutasi dan teks fallback ada. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F013`. Scope only: **Eksperimen perintah navigasi tanpa backend listening permanen, tidak mengotomasi pembayaran.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Voice optional, salah transkripsi tidak membuat mutasi dan teks fallback ada.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-017 — Experimental neo-brutalist manga skin via existing tokens

## 01 Identity
- ID: `FUT-017`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **52/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Buat prototype satu visual direction funky/grunge dengan tokens existing dan semantik status tetap jelas.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **experimental neo-brutalist manga skin via existing tokens**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Contrast WCAG AA pada state penting, toggle rollback ke theme asli tanpa CSS fork.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/theme.ts`](https://github.com/bjo163/mwx-isp/blob/main/web/src/theme.ts); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Buat prototype satu visual direction funky/grunge dengan tokens existing dan semantik status tetap jelas.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/theme.ts` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat prototype satu visual direction funky/grunge dengan tokens existing dan semantik status tetap jelas.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Contrast WCAG AA pada state penting, toggle rollback ke theme asli tanpa CSS fork.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Contrast WCAG AA pada state penting, toggle rollback ke theme asli tanpa CSS fork.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-053.
- **ENABLES:** FUT-018.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Contrast WCAG AA pada state penting, toggle rollback ke theme asli tanpa CSS fork. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **Buat prototype satu visual direction funky/grunge dengan tokens existing dan semantik status tetap jelas.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Contrast WCAG AA pada state penting, toggle rollback ke theme asli tanpa CSS fork.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-018 — Riset visual incident card grunge yang bisa dicetak

## 01 Identity
- ID: `FUT-018`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
PoC ekspor kartu insiden estetika grunge berisi data yang terbukti saja dan tanpa PII.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **riset visual incident card grunge yang bisa dicetak**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Print A4 hitam-putih dapat dibaca; stale/confidence terlihat; tidak membocorkan nama pelanggan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/components/NocImpactPanel.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/components/NocImpactPanel.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `PoC ekspor kartu insiden estetika grunge berisi data yang terbukti saja dan tanpa PII.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/components/NocImpactPanel.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **PoC ekspor kartu insiden estetika grunge berisi data yang terbukti saja dan tanpa PII.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Print A4 hitam-putih dapat dibaca; stale/confidence terlihat; tidak membocorkan nama pelanggan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Print A4 hitam-putih dapat dibaca; stale/confidence terlihat; tidak membocorkan nama pelanggan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** FUT-017.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Print A4 hitam-putih dapat dibaca; stale/confidence terlihat; tidak membocorkan nama pelanggan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **PoC ekspor kartu insiden estetika grunge berisi data yang terbukti saja dan tanpa PII.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Print A4 hitam-putih dapat dibaca; stale/confidence terlihat; tidak membocorkan nama pelanggan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-019 — Edge collector polling read-only pada jaringan terbatas

## 01 Identity
- ID: `FUT-019`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Evaluasi collector kecil dekat NAS untuk target terdaftar, push via signed event tanpa auto discovery.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **edge collector polling read-only pada jaringan terbatas**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Bukti tambahan latensi vs instalasi biasa dan ancaman security; no-go bila tidak menguntungkan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/networkmonitor/monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/networkmonitor/monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Evaluasi collector kecil dekat NAS untuk target terdaftar, push via signed event tanpa auto discovery.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/networkmonitor/monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Evaluasi collector kecil dekat NAS untuk target terdaftar, push via signed event tanpa auto discovery.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Bukti tambahan latensi vs instalasi biasa dan ancaman security; no-go bila tidak menguntungkan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Bukti tambahan latensi vs instalasi biasa dan ancaman security; no-go bila tidak menguntungkan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-049.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Bukti tambahan latensi vs instalasi biasa dan ancaman security; no-go bila tidak menguntungkan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Evaluasi collector kecil dekat NAS untuk target terdaftar, push via signed event tanpa auto discovery.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Bukti tambahan latensi vs instalasi biasa dan ancaman security; no-go bila tidak menguntungkan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-020 — Per-tenant statistik operasional privat berbasis agregat

## 01 Identity
- ID: `FUT-020`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Riset agregasi tanpa PII untuk benchmarking internal yang tidak cross-tenant expose.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F033` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **per-tenant statistik operasional privat berbasis agregat**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Fixture tenant data unik tidak bisa direkonstruksi dari response; publish hanya bila governance jelas.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/dashboard.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/dashboard.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F033`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Riset agregasi tanpa PII untuk benchmarking internal yang tidak cross-tenant expose.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F033` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/dashboard.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Riset agregasi tanpa PII untuk benchmarking internal yang tidak cross-tenant expose.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Fixture tenant data unik tidak bisa direkonstruksi dari response; publish hanya bila governance jelas.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Fixture tenant data unik tidak bisa direkonstruksi dari response; publish hanya bila governance jelas.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-017.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F033` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Fixture tenant data unik tidak bisa direkonstruksi dari response; publish hanya bila governance jelas. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F033`. Scope only: **Riset agregasi tanpa PII untuk benchmarking internal yang tidak cross-tenant expose.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Fixture tenant data unik tidak bisa direkonstruksi dari response; publish hanya bila governance jelas.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-021 — Knowledge base kompatibilitas firmware berbasis bukti uji

## 01 Identity
- ID: `FUT-021`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Tautkan firmware/model/NAS fixture/RFC dan support status sebagai data internal; bukan ensiklopedia tanpa sumber.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F005` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **knowledge base kompatibilitas firmware berbasis bukti uji**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Setiap catatan punya test ID, versi firmware, date dan status UNKNOWN bila tidak teruji.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/vendor-vsa-gap-baseline.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/vendor-vsa-gap-baseline.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F005`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Tautkan firmware/model/NAS fixture/RFC dan support status sebagai data internal; bukan ensiklopedia tanpa sumber.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F005` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `docs/vendor-vsa-gap-baseline.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tautkan firmware/model/NAS fixture/RFC dan support status sebagai data internal; bukan ensiklopedia tanpa sumber.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Setiap catatan punya test ID, versi firmware, date dan status UNKNOWN bila tidak teruji.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Setiap catatan punya test ID, versi firmware, date dan status UNKNOWN bila tidak teruji.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-050.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F005` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Setiap catatan punya test ID, versi firmware, date dan status UNKNOWN bila tidak teruji. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F005`. Scope only: **Tautkan firmware/model/NAS fixture/RFC dan support status sebagai data internal; bukan ensiklopedia tanpa sumber.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Setiap catatan punya test ID, versi firmware, date dan status UNKNOWN bila tidak teruji.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-022 — Event replay simulator untuk reproduksi billing dan NAS

## 01 Identity
- ID: `FUT-022`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Replay event sintetis invoice/session/Disconnect ke sandbox deterministic tanpa data riil.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **event replay simulator untuk reproduksi billing dan nas**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Dua seed identik menghasilkan state hash sama dan nol paket prod.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/app/jobs.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/app/jobs.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Replay event sintetis invoice/session/Disconnect ke sandbox deterministic tanpa data riil.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/app/jobs.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Replay event sintetis invoice/session/Disconnect ke sandbox deterministic tanpa data riil.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Dua seed identik menghasilkan state hash sama dan nol paket prod.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Dua seed identik menghasilkan state hash sama dan nol paket prod.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-032.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Dua seed identik menghasilkan state hash sama dan nol paket prod. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Replay event sintetis invoice/session/Disconnect ke sandbox deterministic tanpa data riil.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Dua seed identik menghasilkan state hash sama dan nol paket prod.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-023 — Graph semantik relasi protokol RADIUS dan feature ID

## 01 Identity
- ID: `FUT-023`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Prototype navigasi RFC→test→feature→file untuk onboarding developer; bukan domain religious ontology.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F021` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **graph semantik relasi protokol radius dan feature id**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Sebuah RFC mengarah ke test dan source nyata, missing link diberi UNKNOWN.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/rfcs`](https://github.com/bjo163/mwx-isp/blob/main/docs/rfcs); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F021`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Prototype navigasi RFC→test→feature→file untuk onboarding developer; bukan domain religious ontology.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F021` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `docs/rfcs` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Prototype navigasi RFC→test→feature→file untuk onboarding developer; bukan domain religious ontology.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Sebuah RFC mengarah ke test dan source nyata, missing link diberi UNKNOWN.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Sebuah RFC mengarah ke test dan source nyata, missing link diberi UNKNOWN.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-016.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F021` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Sebuah RFC mengarah ke test dan source nyata, missing link diberi UNKNOWN. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F021`. Scope only: **Prototype navigasi RFC→test→feature→file untuk onboarding developer; bukan domain religious ontology.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Sebuah RFC mengarah ke test dan source nyata, missing link diberi UNKNOWN.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# FUT-024 — Prediksi kapasitas network yang dibatasi data SNMP nyata

## 01 Identity
- ID: `FUT-024`; domain: **Frontier Innovation**; primary subdomain: **experiment**; type: `experiment`.
- Priority: **P3**; difficulty: Hard; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:frontier`, `priority:p3`, `type:experiment`.
- Canonical owner: **MWX-ISP research/prototype workstream**, specific accountable person not assigned.

## 02 Executive description
Eksperimen forecast sederhana pada timeseries target yang telah diotorisasi dengan confidence bounds.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **prediksi kapasitas network yang dibatasi data snmp nyata**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Backtest menampilkan error, basis sample, dan disables prediction bila data kurang.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/networkmonitor/monitor.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/networkmonitor/monitor.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **bounded opt-in proof of concept**. Work only within `Eksperimen forecast sederhana pada timeseries target yang telah diotorisasi dengan confidence bounds.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **RESEARCH-ONLY until approved TR-F scope change; no production deployment**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/networkmonitor/monitor.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Eksperimen forecast sederhana pada timeseries target yang telah diotorisasi dengan confidence bounds.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Backtest menampilkan error, basis sample, dan disables prediction bila data kurang.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Backtest menampilkan error, basis sample, dan disables prediction bila data kurang.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Reproducible prototype with go/no-go review. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-049.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Backtest menampilkan error, basis sample, dan disables prediction bila data kurang. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Eksperimen forecast sederhana pada timeseries target yang telah diotorisasi dengan confidence bounds.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Backtest menampilkan error, basis sample, dan disables prediction bila data kurang.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.



---

# 05 Economy 12

# ECO-001 — Wawancara willingness-to-pay pada operator ISP kecil

## 01 Identity
- ID: `ECO-001`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **73/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Validasi segmen RT/RW Net dan ISP kecil mengenai pekerjaan penting yang mereka mau bayar, tanpa menambah payment gateway.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F027` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **wawancara willingness-to-pay pada operator isp kecil**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Minimum 5 wawancara consented atau status BLOCKED; quote dan asumsi terpisah.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/MWX-ISP-blueprint.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/MWX-ISP-blueprint.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F027`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Validasi segmen RT/RW Net dan ISP kecil mengenai pekerjaan penting yang mereka mau bayar, tanpa menambah payment gateway.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F027` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `docs/MWX-ISP-blueprint.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Validasi segmen RT/RW Net dan ISP kecil mengenai pekerjaan penting yang mereka mau bayar, tanpa menambah payment gateway.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Minimum 5 wawancara consented atau status BLOCKED; quote dan asumsi terpisah.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Minimum 5 wawancara consented atau status BLOCKED; quote dan asumsi terpisah.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** none (independent after baseline review).
- **ENABLES:** ECO-002, ECO-004, ECO-011.
- **REUSES:** `TR-F027` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Minimum 5 wawancara consented atau status BLOCKED; quote dan asumsi terpisah. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F027`. Scope only: **Validasi segmen RT/RW Net dan ISP kecil mengenai pekerjaan penting yang mereka mau bayar, tanpa menambah payment gateway.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Minimum 5 wawancara consented atau status BLOCKED; quote dan asumsi terpisah.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-002 — Ukur waktu tugas operator sebelum/sesudah MWX-ISP

## 01 Identity
- ID: `ECO-002`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Tentukan benchmark task seperti invoice lookup, NAS session triage, dan restore checklist sebagai value proposition.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F029` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **ukur waktu tugas operator sebelum/sesudah mwx-isp**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Tiga workflow punya waktu baseline nyata dan perubahan diukur pada partisipan berizin.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`web/src/pages/Dashboard.tsx`](https://github.com/bjo163/mwx-isp/blob/main/web/src/pages/Dashboard.tsx); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F029`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Tentukan benchmark task seperti invoice lookup, NAS session triage, dan restore checklist sebagai value proposition.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F029` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `web/src/pages/Dashboard.tsx` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Tentukan benchmark task seperti invoice lookup, NAS session triage, dan restore checklist sebagai value proposition.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Tiga workflow punya waktu baseline nyata dan perubahan diukur pada partisipan berizin.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Tiga workflow punya waktu baseline nyata dan perubahan diukur pada partisipan berizin.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ECO-001.
- **ENABLES:** ECO-010.
- **REUSES:** `TR-F029` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Tiga workflow punya waktu baseline nyata dan perubahan diukur pada partisipan berizin. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F029`. Scope only: **Tentukan benchmark task seperti invoice lookup, NAS session triage, dan restore checklist sebagai value proposition.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Tiga workflow punya waktu baseline nyata dan perubahan diukur pada partisipan berizin.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-003 — Model total cost of ownership deployment ISP

## 01 Identity
- ID: `ECO-003`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **69/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Buat lembar biaya VPS/postgres/backups/ops, lisensi dan maintenance untuk satu tenant vs beberapa tenant.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **model total cost of ownership deployment isp**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Assumptions tagged, 3 scenario + break-even sensitivity terhitung, bukan actual profit.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`Dockerfile`](https://github.com/bjo163/mwx-isp/blob/main/Dockerfile); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Buat lembar biaya VPS/postgres/backups/ops, lisensi dan maintenance untuk satu tenant vs beberapa tenant.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `Dockerfile` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Buat lembar biaya VPS/postgres/backups/ops, lisensi dan maintenance untuk satu tenant vs beberapa tenant.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Assumptions tagged, 3 scenario + break-even sensitivity terhitung, bukan actual profit.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Assumptions tagged, 3 scenario + break-even sensitivity terhitung, bukan actual profit.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-045.
- **ENABLES:** ECO-008.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Assumptions tagged, 3 scenario + break-even sensitivity terhitung, bukan actual profit. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Buat lembar biaya VPS/postgres/backups/ops, lisensi dan maintenance untuk satu tenant vs beberapa tenant.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Assumptions tagged, 3 scenario + break-even sensitivity terhitung, bukan actual profit.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-004 — Evaluasi lisensi managed support vs self-hosted

## 01 Identity
- ID: `ECO-004`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Bandingkan lisensi repo dan opsi kontrak jasa/support tanpa membatasi hak open-source yang sudah diberikan.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **evaluasi lisensi managed support vs self-hosted**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Matriks hak distribusi, maintenance cost dan keputusan legal review, bukan price implementation.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`LICENSE`](https://github.com/bjo163/mwx-isp/blob/main/LICENSE); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Bandingkan lisensi repo dan opsi kontrak jasa/support tanpa membatasi hak open-source yang sudah diberikan.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `LICENSE` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bandingkan lisensi repo dan opsi kontrak jasa/support tanpa membatasi hak open-source yang sudah diberikan.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Matriks hak distribusi, maintenance cost dan keputusan legal review, bukan price implementation.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Matriks hak distribusi, maintenance cost dan keputusan legal review, bukan price implementation.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ECO-001.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Matriks hak distribusi, maintenance cost dan keputusan legal review, bukan price implementation. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Bandingkan lisensi repo dan opsi kontrak jasa/support tanpa membatasi hak open-source yang sudah diberikan.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Matriks hak distribusi, maintenance cost dan keputusan legal review, bukan price implementation.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-005 — Desain paket dukungan & SLA yang realistis

## 01 Identity
- ID: `ECO-005`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Rancang optional service support berbasis kemampuan recovery teruji, tanpa SLA produksi tanpa evidence.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F020` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **desain paket dukungan & sla yang realistis**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Katalog layanan, exclusions dan SLA assumptions berdasarkan RTO/RPO uji.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/backup-and-recovery.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/backup-and-recovery.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F020`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Rancang optional service support berbasis kemampuan recovery teruji, tanpa SLA produksi tanpa evidence.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F020` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `docs/backup-and-recovery.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Rancang optional service support berbasis kemampuan recovery teruji, tanpa SLA produksi tanpa evidence.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Katalog layanan, exclusions dan SLA assumptions berdasarkan RTO/RPO uji.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Katalog layanan, exclusions dan SLA assumptions berdasarkan RTO/RPO uji.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-045.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F020` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Katalog layanan, exclusions dan SLA assumptions berdasarkan RTO/RPO uji. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F020`. Scope only: **Rancang optional service support berbasis kemampuan recovery teruji, tanpa SLA produksi tanpa evidence.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Katalog layanan, exclusions dan SLA assumptions berdasarkan RTO/RPO uji.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-006 — Validasi jasa onboarding dan migrasi pelanggan

## 01 Identity
- ID: `ECO-006`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Riset layanan konsultasi migrasi CSV/config terkontrol untuk operator legal, tanpa CRM baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F027` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **validasi jasa onboarding dan migrasi pelanggan**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Mock migration checklist, test dry-run, estimasi effort dan interview 3 calon klien.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/users_import.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/users_import.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F027`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Riset layanan konsultasi migrasi CSV/config terkontrol untuk operator legal, tanpa CRM baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F027` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/users_import.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Riset layanan konsultasi migrasi CSV/config terkontrol untuk operator legal, tanpa CRM baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Mock migration checklist, test dry-run, estimasi effort dan interview 3 calon klien.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Mock migration checklist, test dry-run, estimasi effort dan interview 3 calon klien.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-059.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F027` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Mock migration checklist, test dry-run, estimasi effort dan interview 3 calon klien. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F027`. Scope only: **Riset layanan konsultasi migrasi CSV/config terkontrol untuk operator legal, tanpa CRM baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Mock migration checklist, test dry-run, estimasi effort dan interview 3 calon klien.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-007 — Pengendalian biaya telemetri dan alert whatsmeow

## 01 Identity
- ID: `ECO-007`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Hitung expense polling/retention/notifikasi per tenant serta risiko ketergantungan WhatsApp unofficial.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F030` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **pengendalian biaya telemetri dan alert whatsmeow**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Cost attribution assumptions dan thresholds di 3 skenario, tanpa SMS marketing.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/notify/dispatcher.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/notify/dispatcher.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F030`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Hitung expense polling/retention/notifikasi per tenant serta risiko ketergantungan WhatsApp unofficial.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F030` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/notify/dispatcher.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Hitung expense polling/retention/notifikasi per tenant serta risiko ketergantungan WhatsApp unofficial.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Cost attribution assumptions dan thresholds di 3 skenario, tanpa SMS marketing.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Cost attribution assumptions dan thresholds di 3 skenario, tanpa SMS marketing.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-048.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F030` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Cost attribution assumptions dan thresholds di 3 skenario, tanpa SMS marketing. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F030`. Scope only: **Hitung expense polling/retention/notifikasi per tenant serta risiko ketergantungan WhatsApp unofficial.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Cost attribution assumptions dan thresholds di 3 skenario, tanpa SMS marketing.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-008 — Cost allocation unit per NAS dan subscriber aktif

## 01 Identity
- ID: `ECO-008`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Gunakan agregat operasional read-only untuk menghitung perkiraan cost-to-serve, bukan menerbitkan billing baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F015` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **cost allocation unit per nas dan subscriber aktif**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Formula transparent, unit expenses dapat direproduksi dan rate tidak dianggap pendapatan.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/dashboard.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/dashboard.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F015`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Gunakan agregat operasional read-only untuk menghitung perkiraan cost-to-serve, bukan menerbitkan billing baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F015` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/dashboard.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Gunakan agregat operasional read-only untuk menghitung perkiraan cost-to-serve, bukan menerbitkan billing baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Formula transparent, unit expenses dapat direproduksi dan rate tidak dianggap pendapatan.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Formula transparent, unit expenses dapat direproduksi dan rate tidak dianggap pendapatan.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ECO-003.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F015` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Formula transparent, unit expenses dapat direproduksi dan rate tidak dianggap pendapatan. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F015`. Scope only: **Gunakan agregat operasional read-only untuk menghitung perkiraan cost-to-serve, bukan menerbitkan billing baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Formula transparent, unit expenses dapat direproduksi dan rate tidak dianggap pendapatan.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-009 — Audit peluang revenue leakage dari invoice yang sudah ada

## 01 Identity
- ID: `ECO-009`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **2**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Analisis exception invoice duplicate/missed grace/paid partial dari data sintetis dan hasil sampel tersanitasi.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F028` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **audit peluang revenue leakage dari invoice yang sudah ada**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang hilang tanpa bukti.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/billing/service.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/billing/service.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F028`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Analisis exception invoice duplicate/missed grace/paid partial dari data sintetis dan hasil sampel tersanitasi.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F028` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/billing/service.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Analisis exception invoice duplicate/missed grace/paid partial dari data sintetis dan hasil sampel tersanitasi.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang hilang tanpa bukti.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang hilang tanpa bukti.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-031.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F028` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang hilang tanpa bukti. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F028`. Scope only: **Analisis exception invoice duplicate/missed grace/paid partial dari data sintetis dan hasil sampel tersanitasi.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Laporan temuan bisa ditelusuri ke rule; belum ada klaim uang hilang tanpa bukti.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-010 — Riset risiko churn berbasis sinyal operasional agregat

## 01 Identity
- ID: `ECO-010`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P3**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Low**.
- Execution score: **50/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p3`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Definisikan indikator operasional complaint/downtime tanpa profiling individu atau marketing otomatis.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F031` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **riset risiko churn berbasis sinyal operasional agregat**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Prototype agregat memiliki consent/data-minimization review dan tidak mempengaruhi layanan subscriber.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`internal/adminapi/noc_impact.go`](https://github.com/bjo163/mwx-isp/blob/main/internal/adminapi/noc_impact.go); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F031`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Definisikan indikator operasional complaint/downtime tanpa profiling individu atau marketing otomatis.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F031` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `internal/adminapi/noc_impact.go` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Definisikan indikator operasional complaint/downtime tanpa profiling individu atau marketing otomatis.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Prototype agregat memiliki consent/data-minimization review dan tidak mempengaruhi layanan subscriber.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Prototype agregat memiliki consent/data-minimization review dan tidak mempengaruhi layanan subscriber.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ECO-002.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F031` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: low possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Prototype agregat memiliki consent/data-minimization review dan tidak mempengaruhi layanan subscriber. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F031`. Scope only: **Definisikan indikator operasional complaint/downtime tanpa profiling individu atau marketing otomatis.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Prototype agregat memiliki consent/data-minimization review dan tidak mempengaruhi layanan subscriber.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-011 — Kajian strategi distribusi community/open-source

## 01 Identity
- ID: `ECO-011`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **3**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Bandingkan komunitas GitHub, dokumentasi bilingual, mitra integrator, layanan konsultasi tanpa marketplace baru.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F023` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **kajian strategi distribusi community/open-source**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Matriks saluran, resource burden dan metrik pilot bukan klaim traction.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`README.md`](https://github.com/bjo163/mwx-isp/blob/main/README.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F023`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Bandingkan komunitas GitHub, dokumentasi bilingual, mitra integrator, layanan konsultasi tanpa marketplace baru.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F023` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `README.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Bandingkan komunitas GitHub, dokumentasi bilingual, mitra integrator, layanan konsultasi tanpa marketplace baru.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Matriks saluran, resource burden dan metrik pilot bukan klaim traction.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Matriks saluran, resource burden dan metrik pilot bukan klaim traction.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ECO-001.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F023` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Matriks saluran, resource burden dan metrik pilot bukan klaim traction. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F023`. Scope only: **Bandingkan komunitas GitHub, dokumentasi bilingual, mitra integrator, layanan konsultasi tanpa marketplace baru.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Matriks saluran, resource burden dan metrik pilot bukan klaim traction.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.


---

# ECO-012 — Partner interoperability kit berbasis fixture NAS

## 01 Identity
- ID: `ECO-012`; domain: **Sustainable Economy**; primary subdomain: **research**; type: `research`.
- Priority: **P2**; difficulty: Medium; effort estimate: **S** (unvalidated); confidence: **medium for planning, unverified execution**; estimated impact/risk: **Medium**.
- Execution score: **67/100 provisional** (planning heuristic, not observed performance); recommended wave: **4**; status: **DRAFT / NOT CREATED ON GITHUB**.
- Suggested labels: `domain:economy`, `priority:p2`, `type:research`.
- Canonical owner: **MWX-ISP product/economic research workstream**, specific accountable person not assigned.

## 02 Executive description
Rancang kit sertifikasi integrator berdasarkan RFC, demo seed dan matriks firmware terbukti.

## 03 Current state
Baseline is GitHub `main` at `1b791f80`. Feature contract `TR-F005` and linked repo documentation are the starting points, **not proof of a defect**. Evidence status: `PROPOSED_VERIFICATION`. Existing implementation must be read before editing.

## 04 Problem/opportunity statement
Without a bounded check of **partner interoperability kit berbasis fixture nas**, the team cannot make a supportable implementation or go/no-go claim. Specific outcome to establish: **Kit uji reproducible tanpa kredensial riil, tidak menyatakan certified tanpa vendor review.**.

## 05 Evidence and source references
- Existing inspected repository path/reference: [`docs/vendor-vsa-gap-baseline.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/vendor-vsa-gap-baseline.md); use it as the investigation anchor and verify relevant functions before edits.
- Scope owner: [`docs/feature-checklist.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/feature-checklist.md), `TR-F005`.
- Roadmap/protocol context: [`docs/roadmap.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/roadmap.md).
- No test execution or observed production defect is claimed solely on the basis of this issue draft.

## 06 Why this matters
Expected benefit: strengthen the reliability, operator usability, operability, or evidence base of MWX-ISP without duplicating historical AUDIT-01..10 deliveries. The benefit remains a hypothesis until demonstrated by the acceptance check.

## 07 Proposed solution and alternatives
Primary technique: **research memo and evidence matrix**. Work only within `Rancang kit sertifikasi integrator berdasarkan RFC, demo seed dan matriks firmware terbukti.`. Consider a no-change outcome if the current implementation already satisfies the criterion, and document why. Prefer reuse to a new service or application.

## 08 Technical architecture and boundaries
- Ownership: `TR-F005` within the existing modular monolith / React Admin / ops flow.
- Read-only research first. Proposed additions to code, schema, endpoint, or UI are *not* existing-file claims.
- Preserve tenant scoping and authenticated operator roles; no unapproved public API/payment self-service/hosted captive portal.
- Scope gate: **MARKET/ECONOMICS RESEARCH-ONLY; no payments or new commercial product without approval**. Requires feature checklist scope decision before implementation.

## 09 Ordered microtasks
- [ ] Inspect `docs/vendor-vsa-gap-baseline.md` and adjacent tests and locate the exact behavior/contracts; record links and HEAD commit.
- [ ] Make an evidence table: existing behavior, missing assertion, reproduction or hypothesis, counterexample, affected tenant roles.
- [ ] Write the smallest change/experiment plan for: **Rancang kit sertifikasi integrator berdasarkan RFC, demo seed dan matriks firmware terbukti.**.
- [ ] Build a failing fixture or explicit measurable hypothesis; include negative/edge cases.
- [ ] Run only isolated test/prototype; preserve data and demonstrate no cross-tenant/secret leakage.
- [ ] Verify the criterion **Kit uji reproducible tanpa kredensial riil, tidak menyatakan certified tanpa vendor review.**; record output, changed files and unresolved limits.
- [ ] Update canonical TR-F docs/ADR or record explicit no-go; don't create duplicate documentation trees.

## 10 Acceptance criteria
- **Given** a clean isolated fixture or documented pilot setup, **when** the targeted scenario is executed, **then** Kit uji reproducible tanpa kredensial riil, tidak menyatakan certified tanpa vendor review.
- No external production side-effect, stale claims or unfounded performance number is accepted.

## 11 Definition of done
Evidence file/trace, reproducible command/steps, relevant automated or human-reviewed validation, owner signoff where needed, updated scope documentation, and an explicit closure decision. Green CI alone is insufficient if this acceptance check remains unperformed.

## 12 Test strategy
Evidence-based comparison, ADR and measurable baseline. Positive and negative cases must be included with failure explanation; for device integration use authorized lab assets and sanitized traces.

## 13 Dependencies & relation types
- **BLOCKED BY:** ENG-050.
- **ENABLES:** none explicitly defined.
- **REUSES:** `TR-F005` current implementation and its documented contract.
- **POTENTIAL CONFLICT:** existing delivered 2026 AUDIT-01..10 tasks or current non-goals; reject duplication on final agent audit.

## 14 Risks & mitigations
- Principal risk: medium possibility of protocol regression, tenant boundary confusion, or scope inflation.
- Mitigation: isolated test, smallest change, verified guardrails, review of licensing/consent/PII where relevant, no surprise write to external systems.

## 15 Validation metrics
Primary success metric is **binary criterion met/not met**: Kit uji reproducible tanpa kredensial riil, tidak menyatakan certified tanpa vendor review. Secondary measures: reproduction rate, test determinism and operator evidence quality. All numerical baselines start **UNKNOWN** until measured.

## 16 Resource estimate
Provisional t-shirt effort `S`; human/device access, vendor lab, infrastructure usage and support costs remain **UNKNOWN** until scoping. No invented dollar estimate.

## 17 Rollback/fallback
If exploration fails, restore baseline state, keep a labeled no-go record, leave the production feature disabled/unmodified. For code changes use small reversible commits, compatible API/schema migration and tested rollback.

## 18 Cross-domain synergy
Any benefit to maintenance cost, user confidence or paid operational support is **conditional and unverified**. This issue owns only its primary outcome; other domains reuse test/evidence rather than reimplementing the same logic.

## 19 Anti-overlap declaration
Canonical owner: `TR-F005`. Scope only: **Rancang kit sertifikasi integrator berdasarkan RFC, demo seed dan matriks firmware terbukti.**. Out of scope: rewriting the RADIUS stack, payment gateway, hosted captive portal, new CRM, generic AI chatbot or unrelated faith/charity components. Existing #45–#54 are treated as closed prior art, not reopened by these drafts.

## 20 Expected outcome
A reviewable artifact or change proving: **Kit uji reproducible tanpa kredensial riil, tidak menyatakan certified tanpa vendor review.**, or a defensible research no-go with the precise blocking evidence.

## 21 Future evolution (not in this issue)
Broader productization requires a separate approved TR-F, user value, security and cost assessment. Do not slip additional features into the present acceptance criterion.

## 22 Evidence required for closure
Link changed code/experiment artifact, exact test invocation/result, redacted logs or screenshots where relevant, updated contract/ADR, reviewer acknowledgement and a final status of delivered, deferred, no-go or verified-already-correct.

