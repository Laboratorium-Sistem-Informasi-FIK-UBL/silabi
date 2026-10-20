# Silabi Web (silabi.ubl.ac.id)

Frontend situs Laboratorium Sistem Informasi FIK UBL — repo & deployment
terpisah dari Tangkas, tapi **tidak punya data sendiri**. Semua konten (hero,
tentang, fokus riset, tim, galeri, kontak) dikelola lewat dashboard Tangkas
(template "Silabi Lab") dan dibaca server ini dari API publik Tangkas:

```
GET {TANGKAS_API_URL}/api/public/site/{SITE_SLUG}
```

## Kenapa server kecil, bukan SPA

- **Ringan**: cuma Go standard library (`net/http`, `html/template`,
  `encoding/json`) — tanpa framework, tanpa `go.sum` pihak ketiga.
- **Aman**: permukaan serangan kecil (tidak ada dependency npm/Go pihak
  ketiga untuk di-audit atau rawan supply-chain attack).
- **SEO-friendly**: HTML sudah lengkap berisi konten saat pertama kali
  dikirim ke browser/crawler (server-rendered), bukan di-fetch belakangan di
  browser seperti SPA.
- **Tanpa CORS**: fetch ke API Tangkas dilakukan di server (Go), bukan di
  browser, jadi tidak perlu whitelist origin di backend Tangkas untuk
  render halaman (`EXTRA_ALLOWED_ORIGINS` di `be` tetap dipakai kalau nanti
  ada fitur client-side yang fetch langsung dari browser).

## Menjalankan lokal

```bash
cp .env.example .env
export $(grep -v '^#' .env | xargs)
go run .
```

Buka `http://localhost:4000`.

## Cara kerja

1. Saat boot, dan setiap `CACHE_TTL_SECONDS` detik, server fetch
   `GET /api/public/site/{SITE_SLUG}` ke `TANGKAS_API_URL` dan menyimpan
   `content` (JSON) di memori.
2. Setiap request `GET /` dirender dari cache itu memakai
   `templates/site.html` (sama shape placeholder `{{.hero.title}}` dsb.
   dengan template "Silabi Lab" di `be/templates/company-profile-silabi-lab.html`).
3. Kalau fetch ke Tangkas gagal, server tetap menyajikan konten cache
   terakhir yang berhasil (tidak langsung down), dan baru menampilkan error
   503 kalau belum pernah berhasil fetch sama sekali (mis. baru pertama
   boot dan Tangkas API sedang down).
4. Form kontak di halaman mem-POST langsung ke
   `{TANGKAS_API_URL}/api/public/leads` dari browser pengunjung (butuh
   `EXTRA_ALLOWED_ORIGINS=https://silabi.ubl.ac.id` di `be/.env` production).

## Update konten

Tidak ada deploy ulang yang dibutuhkan untuk ganti konten — edit lewat
dashboard Tangkas (`SiteEditor`, site dengan slug `SITE_SLUG`), server ini
otomatis ambil versi terbaru di refresh cache berikutnya (maks.
`CACHE_TTL_SECONDS` detik).

## Deployment production

Lihat panduan lengkap (DNS, Nginx, SSL) di `be/docs/custom-domain-silabi-ubl.md`
pada repo backend Tangkas — server ini yang dimaksud sebagai "app repo baru"
di panduan tersebut.

Untuk menjalankan proses binary-nya, panduan lengkap Nginx + PM2 (build,
`ecosystem.config.js`, operasional `pm2 restart`/`pm2 logs`, dst) ada di
[`docs/deploy-nginx-pm2.md`](docs/deploy-nginx-pm2.md).
