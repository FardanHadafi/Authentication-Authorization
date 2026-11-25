# 🛠️ Checklist Minimum Wajib untuk Project *Production-Ready*

Dokumen ini adalah panduan belajar dan daftar periksa praktis (*checklist*) untuk membangun aplikasi web/backend yang layak dirilis ke lingkungan **production** (digunakan oleh pengguna nyata dengan data sungguhan).

Agar sebuah aplikasi aman dan stabil saat live, setidaknya **6 pilar minimum** berikut harus terpenuhi.

---

## 1. Keamanan Dasar (*Non-Negotiable*)

Tanpa poin-poin ini, aplikasi sangat berisiko diretas atau bocor datanya begitu online:

- [ ] **Password Hashing yang Kuat:** Wajib menggunakan algoritma lambat dan *salted* seperti **Argon2id** atau **Bcrypt**. Jangan pernah menyimpan password dalam bentuk *plaintext*, MD5, atau SHA-256 biasa.
  - *Referensi di repo ini:* [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go) (menggunakan Argon2id).
- [ ] **Secret Management:** Tidak ada API key, JWT secret, atau password database yang di-*hardcode* di source code. Semua masuk melalui Environment Variables (`.env` / Secret Manager) yang tidak di-commit ke Git.
  - *Referensi di repo ini:* [`apps/api/shared/config.go`](../apps/api/shared/config.go) & [`.env.example`](../.env.example).
- [ ] **Proteksi Injection & Validasi Backend:** Validasi input wajib ada di sisi server (jangan hanya mengandalkan validasi form di frontend), dan query SQL wajib menggunakan *parameterized queries* (`sqlc`, `pgx`, atau ORM yang aman) untuk menutup celah SQL Injection.
  - *Referensi di repo ini:* [`apps/api/modules/auth/dto.go`](../apps/api/modules/auth/dto.go) (`go-playground/validator`) & [`apps/api/shared/db/`](../apps/api/shared/db/) (`sqlc`).
- [ ] **Transport & Session Aman:** Wajib berjalan di atas **HTTPS** di production, cookie session/token memakai atribut `HttpOnly`, `Secure`, dan `SameSite`, serta konfigurasi **CORS** hanya mengizinkan domain frontend resmi (bukan wildcard `*` sembarangan).
  - *Referensi di repo ini:* [`apps/api/shared/cookie.go`](../apps/api/shared/cookie.go).
- [ ] **Basic Rate Limiting:** Minimal membatasi percobaan login, register, atau reset password agar tidak mudah di-brute-force atau di-spam bot.
  - *Referensi di repo ini:* [`apps/api/shared/limiter.go`](../apps/api/shared/limiter.go).

---

## 2. Integritas Data & Database

Di localhost, database jarang bermasalah karena hanya Anda yang mengaksesnya. Di production, ratusan request dapat datang bersamaan:

- [ ] **Database Migrations Terversi:** Perubahan tabel wajib melalui file migrasi (SQL terurut), bukan mengubah struktur tabel manual lewat GUI di server production.
  - *Referensi di repo ini:* [`apps/api/migrations/`](../apps/api/migrations/).
- [ ] **ACID Database Transactions:** Wajib ada untuk setiap operasi yang mengubah lebih dari satu tabel atau bersifat kritis (misal: membuat order + mengurangi stok, atau mengubah password + menaikkan `token_version`). Jika langkah kedua gagal, langkah pertama harus di-rollback agar data tidak korup.
  - *Referensi di repo ini:* [`apps/api/shared/tx.go`](../apps/api/shared/tx.go) (`RunInTx`).
- [ ] **Connection Pooling & Query Timeout:** Koneksi database harus dibatasi (*pool limit*) dan setiap query wajib dibungkus `context.WithTimeout` supaya satu query yang lambat/macet tidak menghabiskan seluruh koneksi dan melumpuhkan server.
  - *Referensi di repo ini:* [`apps/api/shared/database.go`](../apps/api/shared/database.go).

---

## 3. Reliabilitas & Penanganan Error

Aplikasi production tidak boleh mati total hanya karena satu input aneh dari user:

- [ ] **Panic / Global Error Recovery:** Harus ada middleware penangkap *panic* / unhandled exception. Jika satu request error, hanya request itu yang mendapat status `500`, sementara server tetap hidup melayani user lain.
  - *Referensi di repo ini:* [`apps/api/shared/middleware.go`](../apps/api/shared/middleware.go).
- [ ] **Error Message yang Aman:** Jangan pernah mengirimkan detail internal (seperti raw SQL error, nama tabel, atau stack trace) ke browser user. Catat detail lengkapnya di log server, dan kirim pesan generik yang aman ke client.
  - *Referensi di repo ini:* [`apps/api/shared/writer.go`](../apps/api/shared/writer.go) & [`apps/api/modules/auth/handler.go`](../apps/api/modules/auth/handler.go).
- [ ] **Graceful Shutdown:** Saat server dimatikan atau di-restart untuk deploy versi baru, server harus berhenti menerima request baru, lalu menyelesaikan request yang sedang berjalan terlebih dahulu (*drain*) sebelum proses dimatikan.
  - *Referensi di repo ini:* [`apps/api/cmd/main.go`](../apps/api/cmd/main.go).
- [ ] **Health Check Endpoint (`/health` atau `/ready`):** Endpoint sederhana untuk mengecek apakah aplikasi dan koneksi databasenya sehat, agar load balancer/Docker tahu kapan server siap menerima traffic.

---

## 4. Observabilitas Minimum (Agar Bisa Debugging Saat Live)

Anda **tidak wajib** langsung memasang Grafana Tempo, Prometheus, dan Loki di hari pertama. Minimum yang wajib ada:

- [ ] **Structured Logging + Request ID:** Log dalam format terstruktur (misal JSON dengan `slog`) yang mencatat `timestamp`, `level`, `endpoint`, `status_code`, `latency`, dan `request_id`. Tanpa ini, saat user lapor *"tadi siang saya error pas bayar"*, Anda akan buta dan tidak tahu apa yang terjadi di server.
  - *Referensi di repo ini:* [`apps/api/shared/logger.go`](../apps/api/shared/logger.go) & [`apps/api/shared/middleware.go`](../apps/api/shared/middleware.go).

---

## 5. Pemrosesan Tugas Eksternal Secara *Asynchronous*

- [ ] **Non-Blocking Background Tasks:** Operasi yang bergantung pada jaringan luar yang lambat/rentan gagal (seperti kirim email SMTP, kirim webhook, generate dokumen) tidak boleh menahan HTTP response user terlalu lama. Minimal dijalankan di background worker/goroutine dengan batas timeout (atau antrean seperti Redis/Asynq jika butuh jaminan retry).
  - *Referensi di repo ini:* [`apps/api/shared/task.go`](../apps/api/shared/task.go), [`apps/api/shared/task_handler.go`](../apps/api/shared/task_handler.go), & [`apps/api/shared/mail.go`](../apps/api/shared/mail.go).

---

## 6. Deployment & Backup Minimum

- [ ] **Reproducible Build & Basic CI:** Minimal ada `Dockerfile`/`compose` yang konsisten dan automated check (`test` + `build` lolos) sebelum kode naik ke production.
  - *Referensi di repo ini:* [`compose.yaml`](../compose.yaml), [`compose.app.yaml`](../compose.app.yaml), & [`.github/workflows/ci.yml`](../.github/workflows/ci.yml).
- [ ] **Automated Database Backup:** Sering dilupakan, padahal jika database terhapus atau korup tanpa backup, seluruh bisnis dan data pengguna bisa hilang permanen. Pastikan database memiliki jadwal backup otomatis (harian/per-jam) dan pernah diuji proses *restore*-nya.

---

## ⚖️ Peta Posisi: *Minimum Production-Ready* vs *Enterprise-Grade*

Gunakan tabel ini untuk membedakan mana yang **wajib ada sejak awal (hari pertama live)** dan mana yang baru dibutuhkan saat aplikasi berkembang ke skala *enterprise*:

| Komponen | Wajib di *Minimum Production* (Mid-Level) | Tambahan *Enterprise-Grade* (Senior / Staff) |
| :--- | :--- | :--- |
| **Autentikasi** | Register, Login (Bcrypt/Argon2), Session/JWT, Reset Password | + TOTP 2FA (RFC 6238), QR Code, Recovery Codes, Step-Up Auth |
| **Otorisasi** | Role dasar (`admin` vs `user`) | + Granular PBAC relasional + O(1) Redis Set caching |
| **Session Revocation**| Logout / Expiry token biasa | + Immediate JWT Revocation via `token_version` + Redis Pipeline |
| **Keamanan Request** | HTTPS, CORS, Basic Rate Limit | + OWASP CSRF Defense-in-Depth (`Sec-Fetch-Site`, constant-time), GCRA Limiter |
| **Database** | Migrations, Pool, Timeout, ACID Transactions | + `sqlc` compile-time type-safe SQL + zero N+1 Redis indexing |
| **Audit & Compliance** | Log error & akses standar | + Tamper-evident SOC 2 Audit Logs via async buffered worker |
| **Observabilitas** | Structured JSON Logs + Request ID + `/health` | + OpenTelemetry Tracing (Tempo), Prometheus Metrics, Loki, Grafana |
