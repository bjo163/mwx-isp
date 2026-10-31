# Panduan Admin

> English version: [Admin UI Manual](../en/admin-manual.md)

## Alur onboarding

1. Login menggunakan akun admin yang dibuat saat instalasi, lalu ganti password.
2. Tinjau **Network & Alerts** dan daftarkan perangkat yang memang boleh dipantau.
3. Buat **RADIUS Profile** untuk atribut dan kecepatan akses.
4. Daftarkan **NAS** menggunakan alamat sumber dan shared secret yang sesuai.
5. Buat **Customer** dan **Package**, lalu hubungkan melalui **Subscription**.
6. Periksa invoice dan catat pembayaran melalui menu **Billing**.
7. Verifikasi autentikasi dan accounting dari NAS sebelum layanan pelanggan.

## Menu utama

| Menu | Kegunaan |
| --- | --- |
| Dashboard | Ringkasan pelanggan, sesi, layanan, dan billing |
| Customers | Profil, kontak, subscription, invoice, dan saldo pelanggan |
| Packages | Paket komersial dan relasinya dengan RADIUS Profile |
| Subscriptions | Aktivasi layanan pelanggan dan pengaturan siklus billing |
| Billing | Invoice, pembayaran manual, dan status jatuh tempo |
| Users / Profiles | Akun RADIUS dan kebijakan akses jaringan |
| Online Sessions / Accounting | Sesi aktif dan riwayat autentikasi / accounting |
| NAS / Network Nodes | Perangkat jaringan yang terdaftar |
| Network & Alerts | Status pemeriksaan jaringan dan notifikasi operator |
| Product Branding | Nama, logo, tagline, dan warna instalasi |
| System Configuration | Konfigurasi layanan dan operasional |

Nomor customer, subscription, invoice, dan payment dihasilkan oleh sistem.
Jangan membuat nomor secara manual atau memakai ulang sample user pada jaringan
produksi. Data contoh hanya dibuat saat database benar-benar kosong.

## Studio Brand, Website, dan Paket Tenant

Administrator tenant bekerja di ruang kerjanya sendiri; halaman berikut tidak
dapat membaca atau menulis data tenant lain.

- **Brand Studio tenant** (`#/tenant/branding`) — nama brand, nama pendek,
  varian logo, favicon, social image, token warna, tema, dan tipografi.
- **Website Studio tenant** (`#/tenant/website`) — situs publik berbasis
  section: draf, publikasi, riwayat versi, dan restore. Draf tetap privat
  sampai dipublikasikan.
- **Package Studio** (`#/isp/package-studio`) — nama, deskripsi, kategori,
  badge, highlight, dan urutan tampil paket yang sudah dimiliki tenant; harga
  dan kecepatan tetap berada pada paket dan RADIUS Profile.

Lihat [Pengalaman Brand Tenant](./tenant-brand-experience.md) untuk aturan
peran, hostname terverifikasi, semantik publikasi, dan daftar API. Operator
platform mengelola branding seluruh instalasi di halaman **Product Branding**;
branding tenant adalah state terpisah yang dibatasi per tenant.

Untuk detail field, hak akses, dan tampilan setiap halaman, lihat
[manual lengkap berbahasa English](../en/admin-manual.md).
