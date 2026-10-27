# Batas Integrasi Portal / Hotspot

> English version: [Portal / Hotspot Integration Boundary](../en/portal-hotspot-boundary.md)

MWX-ISP menyediakan backend AAA RADIUS. Produk ini tidak menghosting captive
portal atau halaman login pelanggan. Batas integrasi dan tanggung jawab sistem
eksternal dijelaskan dalam [panduan English](../en/portal-hotspot-boundary.md).

## Surface yang sudah ada (rekonsiliasi scope, 2026-10-27)

Beberapa tool operator sudah ada sebelum batas ini ditulis: pembuatan dan cetak
batch voucher hotspot, halaman lookup pelanggan (`/portal`), tiket gangguan,
catatan IPAM/ODP, serta landing page publik dengan katalog paket dan formulir
registrasi. Semuanya dicatat di inventory A20 pada feature checklist dan
berstatus **grandfathered**:

- hanya boleh diperbaiki untuk containment, kebenaran data, isolasi tenant, dan
  pengujian;
- bukan persetujuan untuk portal hosting, login pelanggan, onboarding guest,
  onboarding voucher, CRM, ekspansi ticketing, atau payment gateway;
- pencarian tagihan, registrasi, dan cek voucher publik bukan layanan
  self-service anonim: API-nya memerlukan JWT operator, sehingga pengunjung
  anonim mendapat error autentikasi;
- kanal pembayaran online tetap nonaktif dengan penolakan `503`/`410` sampai ada
  provider terverifikasi dan keputusan scope.

MWX-ISP tetap backend AAA RADIUS. Setiap alur pelanggan atau endpoint publik
baru harus lebih dulu memperbarui feature checklist; keberadaan route atau menu
tidak berarti fitur tersebut disetujui untuk diperluas.
