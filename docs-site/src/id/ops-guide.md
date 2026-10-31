# Panduan Operasional

> English version: [Operations Guide](../en/ops-guide.md)

## Deployment yang disarankan

Untuk VPS, gunakan Docker Compose dengan PostgreSQL dan reverse proxy HTTPS.
Instalasi otomatis tersedia melalui:

```bash
sudo bash scripts/vps-install.sh
```

Installer menghasilkan secret dan password unik, mengikat web admin ke
localhost, menyalakan PostgreSQL, serta memasang timer update harian jika
systemd tersedia. Tinjau `.env` dan simpan salinan kredensial di tempat aman.

Untuk mengelola beberapa ISP atau RT/RW Net dalam satu instalasi, konfigurasi
operator platform opsional melalui `MWX_PLATFORM_ADMIN_USERNAME` dan
`MWX_PLATFORM_ADMIN_PASSWORD`. Dari menu **Platform → Organizations**, buat
organisasi dan administrator tenant pertamanya. Operator tenant masuk memakai
slug organisasi, username, dan password. Tenant ID dari request tidak memberi
akses. Data lama dimigrasikan ke organisasi `default` tanpa mengganti ID atau
kredensial pelanggan.
Pada baris organisasi yang sama, administrator platform dapat membuat akses
operator tenant dengan peran `admin` atau `operator`, lalu mencabutnya. Pencabutan
menonaktifkan membership dan token yang sudah terbit ditolak pada request
berikutnya. Kredensial operator tetap lokal per tenant; username dan password
yang sama di organisasi lain adalah akun terpisah yang harus dibuat secara sadar.

RADIUS menentukan tenant dari NAS yang terdaftar sebelum mencari pelanggan.
Pada listener bersama, alamat IP sumber setiap NAS harus unik di seluruh
instalasi; identitas NAS ambigu akan ditolak. Uji setiap tenant dengan NAS dan
kredensialnya sendiri sebelum onboarding pelanggan.

## Port jaringan

| Port | Protokol | Akses |
| --- | --- | --- |
| `1816` | TCP | Web/Admin API; batasi di belakang HTTPS proxy |
| `1812` | UDP | RADIUS authentication dari NAS yang dikenal |
| `1813` | UDP | RADIUS accounting dari NAS yang dikenal |
| `2083` | TCP | RadSec bila dipakai |

Jangan membuka database PostgreSQL ke publik. Terapkan allowlist firewall untuk
NAS dan port yang benar-benar digunakan.

## Metrik dan pemantauan

Semua counter RADIUS dan gauge sistem (CPU/memori host dan proses) disimpan di
memori dan disajikan ulang dalam format teks Prometheus pada `GET /metrics`.
Endpoint ini **nonaktif secara default** dan menjawab 404 sampai token scrape
dikonfigurasi:

```bash
# .env (docker compose memetakan MWX_ISP_METRICS_TOKEN ke TOUGHRADIUS_METRICS_TOKEN)
MWX_ISP_METRICS_TOKEN=$(openssl rand -hex 24)
```

```bash
curl -H "Authorization: Bearer ${MWX_ISP_METRICS_TOKEN}" https://isp.example.com/metrics
```

Token salah atau kosong menghasilkan 401. Namanya tidak berubah dari yang
sudah dikenal di log dan handbook: `radus_accept`, `radus_accounting`,
`radus_online`/`radus_offline` (penghitung accounting Start/Stop yang sukses, bukan
jumlah sesi yang sedang online — untuk itu lihat daftar online session),
`radus_auth_drop`/`radus_acct_drop`, `radus_radsec_saturated`, serta counter
penolakan per penyebab seperti `radus_reject_passwd_error` dan
`radus_reject_ldap_error`. Gauge sistem diekspor sebagai `system_cpuuse`,
`system_memuse`, `mwx_isp_cpuuse`, dan `mwx_isp_memuse` dengan sampling tiap
30 detik.

Tiga series lain berasal dari job latar dan berlabel tetap — tanpa dimensi
tenant maupun pelanggan, jadi jumlahnya tidak bertambah mengikuti banyak tenant:
`fup_scheduler_tenant_attempts_total` dan `fup_scheduler_tenant_failures_total`
menghitung percobaan FUP per tenant dan yang gagal (tenant yang error diisolasi;
tenant setelahnya tetap di-throttle pada run yang sama), sedangkan
`billing_invoice_without_subscription_total` menghitung invoice jatuh tempo yang
dilewati penagihan karena baris subscription-nya tidak ada. Angka terakhir yang
bukan nol perlu diperiksa: utangnya nyata tapi tidak ada yang bisa disuspend,
biasanya karena restore backup membawa invoice tanpa subscription-nya. Tiap
kejadian juga dicatat di log level warn beserta nomor invoice.

Counter ini bersifat per proses dan kembali nol setelah restart; dashboard admin
membaca tabel accounting dan sesi yang tersimpan di database, sehingga tren
7 hari tidak berubah ketika layanan dimulai ulang.

## Perbaikan dan uninstall deployment VPS

Perintah repair online mengunduh installer resmi terbaru dari branch `main`
dan menjalankan alur upgrade aman. Script memeriksa host, memvalidasi
konfigurasi Compose, membuat backup dari instalasi yang berjalan, lalu
memulihkan layanan tanpa mengganti `.env` atau volume data:

```bash
curl -fsSL https://raw.githubusercontent.com/bjo163/mwx-isp/main/scripts/vps-repair.sh | sudo bash
```

Jika checkout di `/opt/mwx-isp` memiliki perubahan lokal yang belum di-commit,
installer dan updater membuat salinan lengkap terlebih dahulu di direktori
sejajar seperti `/opt/mwx-isp.local-changes.<waktu-UTC>`, kemudian memperbarui
checkout resmi. File `.env` tetap dipertahankan dan database/aplikasi tetap
melewati backup pra-update. Tinjau atau ambil file yang dibutuhkan dari salinan
tersebut setelah layanan pulih; jangan menghapusnya sebelum memastikan isinya
tidak diperlukan.

Jika uninstall lama menghapus direktori yang sedang dipakai sesi SSH, jalankan
`cd /opt` pada sesi tersebut lalu ulangi installer. Installer terbaru otomatis
memulihkan working directory yang sudah tidak valid. Uninstall berikutnya
membiarkan direktori instalasi kosong agar shell aktif tetap bisa dipakai dan
installer dapat melakukan clone ulang ke sana.

Tambahkan `--check` untuk preflight tanpa perubahan. Repair membutuhkan
direktori instalasi dan `.env` yang cocok jika volume data sudah ada. Jika
PostgreSQL dan aplikasi berhenti, nyalakan keduanya dan buat backup terlebih
dahulu agar installer dapat memeriksa dan melindungi data.

Uninstall layanan dan checkout sambil mempertahankan data PostgreSQL/aplikasi:

```bash
curl -fsSL https://raw.githubusercontent.com/bjo163/mwx-isp/main/scripts/vps-uninstall.sh | sudo bash
```

Script menonaktifkan timer update dan backup MWX-ISP. File `.env` disimpan
sebagai `/opt/mwx-isp.env.uninstalled` untuk pemulihan; lindungi file tersebut
karena berisi kredensial database. Backup di `/var/backups/mwx-isp` tetap ada.
Untuk menghapus permanen volume database dan aplikasi, tambahkan `-s -- --purge`
pada perintah remote. Tambahkan `--remove-backups` bersama `--purge` untuk
menghapus backup lokal juga. Kedua mode meminta konfirmasi dengan mengetik
`uninstall`; gunakan `--yes` hanya untuk otomasi.

Setelah uninstall yang mempertahankan data, jalankan ulang installer. Installer
melakukan clone checkout terlebih dahulu, lalu memulihkan
`/opt/mwx-isp.env.uninstalled` sebagai `/opt/mwx-isp/.env`, sehingga Compose
tersambung kembali ke volume PostgreSQL dan data aplikasi yang dipertahankan:

```bash
curl -fsSL https://raw.githubusercontent.com/bjo163/mwx-isp/main/scripts/vps-install.sh | sudo bash
```

Jaga kerahasiaan konfigurasi tersimpan karena berisi kredensial database. Jika
`.env` yang cocok sudah ada, installer tidak mengubahnya.

## Backup dan pemulihan

**System Config → Backup** hanya tersedia bagi administrator platform dan
mengunduh snapshot JSON installation-wide skema 9.3. Snapshot berisi semua
organisasi beserta metadata kepemilikan tenant tersembunyi, konfigurasi node
dan NAS, profil/pelanggan RADIUS, accounting dan audit operator, customer,
paket, subscription, invoice, pembayaran, sequence dokumen, target/sampel/
insiden monitoring, pengaturan dan outbox notifikasi, membership operator,
operator, sertifikat
beserta private key, serta kredensial SNMP terenkripsi. Sesi online sengaja
tidak disalin karena statusnya menjadi usang; NAS membangunnya kembali saat
tersambung.

Restore memulihkan data semua organisasi dalam satu transaksi. Hak administrator
platform yang sudah ada dipertahankan, dan file backup tidak dapat memberikan
hak tersebut. File snapshot tetap berisi password pelanggan, hash password
operator, private key sertifikat, dan rahasia produksi; lindungi seperti secret
produksi. Simpan salinan backup di luar server dan lakukan uji pemulihan berkala.
Untuk disaster recovery penuh, buat juga dump PostgreSQL terenkripsi:

```bash
sudo /opt/mwx-isp/scripts/backup-db.sh
sudo /opt/mwx-isp/scripts/restore-db.sh /path/ke/backup.sql.gz
```

Backup lokal bukan salinan off-site. Salin backup terenkripsi ke lokasi terpisah
dan lakukan uji restore berkala. Sebelum update manual:

```bash
cd /opt/mwx-isp
sudo scripts/vps-update.sh
```

## Kompatibilitas upgrade

Nama variabel `TOUGHRADIUS_*`, direktori `/var/toughradius`, dan beberapa nama
file lama dipertahankan agar instalasi yang ada tetap dapat membaca konfigurasi,
database, log, serta sertifikatnya. Instalasi VPS baru menggunakan PostgreSQL.

Untuk daftar lengkap pengaturan, sertifikat, monitoring, log, dan prosedur
pemulihan, lihat [Operations Guide berbahasa English](../en/ops-guide.md).
