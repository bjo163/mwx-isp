# Pengalaman Brand Tenant

> English version: [Tenant Brand Experience](../en/tenant-brand-experience.md)

MWX-ISP melayani beberapa ISP dan RT/RW Net dari satu instalasi. Setiap tenant
menampilkan brand, situs web, dan katalog paketnya sendiri tanpa bisa mengakses
data tenant lain. Halaman ini menjelaskan tiga studio yang dipakai operator,
permukaan publik yang dihasilkannya, dan batas yang memisahkannya.

Batasan domain ada di
[`docs/tenant-brand-experience.md`](https://github.com/bjo163/mwx-isp/blob/main/docs/tenant-brand-experience.md);
halaman ini adalah panduan operator.

## Peran

| Peran | Brand Studio | Website Studio | Package Studio | Situs publik |
| --- | --- | --- | --- | --- |
| `admin` / `super` tenant | Baca dan tulis | Baca, simpan, terbitkan, pulihkan | Baca dan tulis presentasi | Baca |
| `operator` tenant | Hanya baca | Hanya baca | Hanya baca | Baca |
| `platform_admin` | Tidak berlaku | Tidak berlaku | Tidak berlaku | Baca |

Setiap penulisan diperiksa di server. Menyembunyikan tombol di browser tidak
memberi atau mencabut izin; endpoint-nya sendiri yang menolak permintaan.

## Brand Studio

Rute: `#/tenant/branding`.

Brand Studio menyimpan identitas pemasaran tenant yang sedang masuk:

- nama brand, nama pendek, tagline, dan nama tampilan lama;
- varian logo (utama, ringkas, terang, gelap), favicon, dan gambar sosial;
- palet tetap (primary, secondary, accent, background, surface, text, link);
- tema (`light`, `dark`, `system`) dan preset tipografi;
- email, telepon, WhatsApp, situs web, dan alamat dukungan.

Aturan yang ditegakkan API:

- Warna harus hex enam digit; CSS bebas tidak pernah disimpan.
- Referensi gambar harus jalur relatif same-origin atau URL HTTPS absolut; URL
  protocol-relative ditolak.
- Aset yang diunggah harus PNG, dikodekan ulang di server, dan disimpan dengan
  nama berlingkup tenant. Hapus dan ganti menolak menghapus berkas yang masih
  direferensikan versi situs terbit, presentasi paket, atau snapshot invoice.
- Identitas legal issuer (nama perusahaan, NPWP, alamat tagihan) tetap ada di
  record `Tenant` dan tidak diedit di sini.

## Website Studio

Rute: `#/tenant/website`.

Situs web adalah konten seksi terstruktur, bukan HTML bebas. Sepuluh seksi:
`header`, `hero`, `announcement`, `about`, `services`, `coverage`, `sla`,
`contact`, `social`, dan `footer`; masing-masing membawa judul, subjudul, isi,
gambar, label dan URL CTA, visibilitas, serta urutan.

Alur kerja:

1. Edit seksi di **Section editor**. Pratinjau langsung menampilkan seksi
   terpilih sebagai teks.
2. **Save draft** hanya menyimpan draft. Situs publik tetap memakai versi terbit.
3. **Publish website** membuat snapshot versi yang tidak bisa diubah dan
   memperbarui proyeksi publik tenant itu saja.
4. **Restore** menerbitkan ulang versi lama sebagai versi baru. Riwayat bersifat
   append-only; pemulihan tidak pernah menghapus snapshot.

Draft dibatasi 96 KiB, kunci seksi tak dikenal dan duplikat ditolak, dan URL CTA
hanya menerima HTTPS, `mailto:`, `tel:`, atau jalur relatif same-origin.

## Hostname terverifikasi

Situs publik dan katalog menentukan tenant dari hostname permintaan, bukan dari
ID tenant yang dikirim klien.

- Subdomain terkelola (`<tenant>.e-isp.id`) dicocokkan ke tenant aktif.
- Domain kustom harus lolos verifikasi kepemilikan DNS: MWX-ISP menulis token
  acak, tenant menerbitkan `_mwx-isp-verification.<hostname>` berisi
  `mwx-isp-verification=<token>`, dan baru setelah itu pemetaan aktif.
- Pemetaan yang belum terverifikasi, duplikat, atau tidak aktif ditolak.
  Hostname di bawah `e-isp.id` tidak bisa didaftarkan sebagai domain kustom.
- Header forwarded hanya dipercaya bila reverse proxy dideklarasikan trusted.

## Package Studio

Rute: `#/isp/package-studio`.

Package Studio mengedit *presentasi* paket operasional. Studio dapat menambah
nama tampilan, deskripsi singkat, kategori, ikon, gambar, badge, warna sorotan,
highlight, label CTA, syarat, urutan tampil, tanda unggulan, dan visibilitas
publik.

Studio tidak bisa mengubah kebenaran operasional:

| Field | Sumber kebenaran |
| --- | --- |
| Harga | `InternetPackage.Price` |
| Siklus tagihan | `InternetPackage.BillingCycle` |
| Status | `InternetPackage.Status` |
| Kecepatan | `RadiusProfile.UpRate` / `DownRate` dari profil yang dirujuk |
| FUP | `InternetPackage.FupLimitGB` |

Studio menampilkan harga operasional sebagai teks hanya-baca beserta sumbernya
di atas editor. Payload yang mencoba mengirim `price` diabaikan: server hanya
menulis field presentasi. Setiap penulisan paket membuktikan paket milik tenant
yang terautentikasi, sehingga paket tenant lain menjawab `404`.

## Permukaan publik

- Situs web: `/#/site/<slug-atau-host-terverifikasi>` menampilkan seksi terbit
  yang terlihat sesuai urutan. Tenant tanpa versi terbit melihat pemberitahuan;
  konten tenant lain tidak pernah menggantikannya.
- Branding: `GET /api/v1/tenant-branding/<identifier>` hanya mengembalikan field
  tampilan yang aman.
- Katalog: `GET /api/v1/public/packages` mengembalikan paket aktif tenant yang
  teresolusi, menghormati `public_visible`, dan menampilkan `price`, kecepatan
  turunan, serta FUP sebagai data operasional hanya-baca.

Ketiganya hanya-baca. Endpoint pencarian pelanggan, cek voucher, pendaftaran,
pembayaran, dan swalayan tetap fail-closed dan tidak dibuka oleh fitur ini.

## Permukaan API

Semua rute administratif memerlukan JWT berlingkup tenant dan memakai envelope
sukses `{"data": …}` serta envelope gagal `{"error", "message"}` standar pada
[API contract](https://github.com/bjo163/mwx-isp/blob/main/docs/api-contract.md).

| Metode | Path | Kegunaan |
| --- | --- | --- |
| `GET` / `PUT` | `/api/v1/branding` | Baca atau ubah brand tenant |
| `POST` | `/api/v1/branding/assets` | Unggah aset PNG brand |
| `DELETE` | `/api/v1/branding/assets/:filename` | Hapus aset yang tak lagi dirujuk |
| `GET` / `PUT` | `/api/v1/tenant-website/draft` | Baca atau simpan draft situs |
| `POST` | `/api/v1/tenant-website/publish` | Terbitkan draft sebagai versi baru |
| `GET` | `/api/v1/tenant-website/versions` | Daftar versi terbit |
| `POST` | `/api/v1/tenant-website/versions/:version/rollback` | Pulihkan versi |
| `GET` / `POST` | `/api/v1/tenant-domains` | Daftar atau ajukan domain kustom |
| `POST` | `/api/v1/tenant-domains/:id/verify` | Verifikasi tantangan DNS |
| `DELETE` | `/api/v1/tenant-domains/:id` | Hapus domain kustom |
| `GET` | `/api/v1/isp/package-presentations` | Daftar presentasi tenant ini |
| `GET` / `PUT` | `/api/v1/isp/packages/:id/presentation` | Baca atau ubah satu presentasi |
| `GET` | `/api/v1/public/packages` | Katalog publik tenant teresolusi |
| `GET` | `/api/v1/tenant-website/public/:identifier` | Situs web publik terbit |
| `GET` | `/api/v1/tenant-branding/:identifier` | Branding tampilan publik |

## Bukti verifikasi

- `test/integration/tenant_brand_experience_test.go` menjalankan dua admin tenant
  sungguhan lewat HTTP terhadap PostgreSQL: isolasi branding, draft yang belum
  terbit, riwayat publikasi/pemulihan, katalog berbasis hostname, paket
  tersembunyi, penulisan presentasi lintas tenant, dan integritas harga.
- `web/e2e/tenant-brand-experience.spec.ts` menjalankan kedua tenant melalui
  Brand Studio, Website Studio, Package Studio, dan situs publik di browser.
- Suite unit `branding_test.go`, `tenant_branding_assets_test.go`,
  `branding_editor_test.go`, `tenant_website_test.go`, `tenant_domains_test.go`,
  dan `package_presentation_test.go` mengunci aturan validasi di level handler.
