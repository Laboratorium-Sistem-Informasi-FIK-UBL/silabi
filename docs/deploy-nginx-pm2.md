# Deploy `silabi-web` — Nginx + PM2

Panduan jalankan binary Go `silabi-web` di server production (server yang
sama dengan Tangkas BE/FE — lihat `be/docs/custom-domain-silabi-ubl.md` untuk
konteks DNS & arsitektur domainnya) pakai **PM2** sebagai process manager
(bukan systemd) dan **Nginx** sebagai reverse proxy.

PM2 biasanya dipakai untuk proses Node.js, tapi bisa menjalankan binary apa
saja asal dipanggil dengan `--interpreter none` — jadi tetap bisa dipakai
untuk binary Go ini tanpa masalah.

## 1. Build binary di server

```bash
cd /var/www/silabi-web   # clone/pull repo ini ke sini
go build -ldflags="-s -w" -o bin/silabi-web .
```

## 2. Siapkan `.env`

```bash
cp .env.example .env
nano .env
```

Isi minimal:
```
TANGKAS_API_URL=https://api.tangkas.sta.my.id
SITE_SLUG=silabi
PORT=4000
CACHE_TTL_SECONDS=60
```

## 3. Install PM2 (kalau belum ada)

```bash
sudo npm install -g pm2
```

## 4. Ecosystem file PM2

Buat `ecosystem.config.js` di root repo ini:

```js
module.exports = {
  apps: [
    {
      name: "silabi-web",
      script: "./bin/silabi-web",
      interpreter: "none",
      cwd: "/var/www/silabi-web",
      env_file: "/var/www/silabi-web/.env",
      autorestart: true,
      watch: false,
      max_memory_restart: "200M",
    },
  ],
};
```

> Catatan: PM2 versi lama tidak selalu membaca `env_file` otomatis
> tergantung versi. Kalau env tidak terbaca, pakai field `env` langsung di
> ecosystem file sebagai gantinya:
> ```js
> env: {
>   TANGKAS_API_URL: "https://api.tangkas.sta.my.id",
>   SITE_SLUG: "silabi",
>   PORT: "4000",
>   CACHE_TTL_SECONDS: "60",
> }
> ```

## 5. Jalankan dengan PM2

```bash
pm2 start ecosystem.config.js
pm2 save
pm2 startup   # ikuti instruksi yang muncul (generate systemd unit untuk PM2 sendiri, sekali saja per server)
```

Perintah operasional sehari-hari:

```bash
pm2 status
pm2 logs silabi-web
pm2 restart silabi-web   # setelah `git pull` + build ulang binary
pm2 stop silabi-web
```

## 6. Blok Nginx

```nginx
server {
    listen 80;
    server_name silabi.ubl.ac.id;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name silabi.ubl.ac.id;

    ssl_certificate     /etc/letsencrypt/live/silabi.ubl.ac.id/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/silabi.ubl.ac.id/privkey.pem;

    location / {
        proxy_pass http://127.0.0.1:4000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/silabi-web /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx
```

## 7. SSL (single domain, HTTP-01 cukup)

```bash
sudo certbot --nginx -d silabi.ubl.ac.id
```

## 8. Verifikasi

```bash
pm2 status                              # pastikan silabi-web "online"
curl -I http://127.0.0.1:4000           # langsung ke app, lewati Nginx
curl -I https://silabi.ubl.ac.id        # lewat Nginx + SSL
curl -s https://silabi.ubl.ac.id/health # -> "ok"
```

## Update setelah ganti kode (bukan ganti konten)

Ganti konten (hero, tim, galeri, dst) **tidak perlu** langkah ini — cukup
edit lewat dashboard Tangkas, server refresh otomatis tiap `CACHE_TTL_SECONDS`
detik. Langkah di bawah ini hanya untuk update kode `main.go`/`templates/site.html`:

```bash
cd /var/www/silabi-web
git pull
go build -ldflags="-s -w" -o bin/silabi-web .
pm2 restart silabi-web
```
