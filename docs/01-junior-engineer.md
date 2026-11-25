# 🌱 Level 1: Junior Software / Fullstack Engineer

Pada level **Junior**, industri **tidak** menuntut Anda membangun sistem terdistribusi, *distributed tracing*, atau protokol kriptografi kompleks dari nol. Ekspektasi utama terhadap Junior adalah **penguasaan fundamental web**, kemampuan membaca dan menulis kode yang bersih untuk fitur standar, serta pemahaman alur data *end-to-end* dari Frontend ke Database.

---

## ✅ Bagian Project yang Diekspektasikan untuk Junior

### 1. Backend (Go) — Fundamental HTTP, Validasi, & CRUD
* **Konfigurasi & Koneksi Database Dasar**
  * [`apps/api/shared/config.go`](../apps/api/shared/config.go): Memuat variabel environment (`.env`) menggunakan struct `Config` dan library `caarlos0/env`.
  * [`apps/api/shared/database.go`](../apps/api/shared/database.go): Membuka koneksi pool ke PostgreSQL (`pgxpool`).
  * [`apps/api/shared/writer.go`](../apps/api/shared/writer.go): Helper standar untuk menulis JSON response & `ErrorResponse` dengan HTTP status code yang tepat (`200`, `201`, `400`, `401`, `404`, `500`).
* **DTO & Validasi Input**
  * [`apps/api/modules/auth/dto.go`](../apps/api/modules/auth/dto.go): Mendefinisikan struct request/response (`CreateUserRequest`, `LoginUserRequest`, `UserResponse`) beserta tag validasi (`validate:"required,email,min=8"`).
* **Routing & HTTP Handler Dasar**
  * [`apps/api/modules/auth/route.go`](../apps/api/modules/auth/route.go): Mendaftarkan endpoint HTTP (`POST /register`, `POST /login`, `GET /profile`, dll.).
  * [`apps/api/modules/auth/handler.go`](../apps/api/modules/auth/handler.go): Decode JSON request body, menjalankan validator, memanggil method di service layer, dan mengembalikan JSON response.
* **Skema Database Dasar**
  * [`apps/api/migrations/20261121143853_create_users_table.sql`](../apps/api/migrations/20261121143853_create_users_table.sql): Membuat tabel `users` dasar (`id`, `full_name`, `email`, `password_hash`, `role`, `status`, `created_at`, `updated_at`).
* **Business Logic Dasar di [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go)**
  * **Register dasar**: Cek apakah email sudah terdaftar $\to$ hash password $\to$ simpan user ke database.
  * **Login dasar**: Ambil user berdasarkan email $\to$ verifikasi password hash $\to$ buat token JWT.
  * **GetProfile & ListUsers**: Mengambil data user dari database dan memetakannya ke `UserResponse`.
  * **Token JWT Dasar** ([`apps/api/shared/token.go`](../apps/api/shared/token.go)): Memahami cara generate dan mem-parse JWT claims (`UserID`, `Email`, `Role`, `exp`).

### 2. Frontend (SvelteKit) — Form, Validasi Client, & Konsumsi API
* **Validasi Schema**
  * [`apps/web/src/lib/schemas/auth.ts`](../apps/web/src/lib/schemas/auth.ts): Mendefinisikan aturan validasi form menggunakan `valibot`.
* **Halaman & Form Actions Standar**
  * [`apps/web/src/routes/register/+page.svelte`](../apps/web/src/routes/register/+page.svelte) & [`+page.server.ts`](../apps/web/src/routes/register/+page.server.ts)
  * [`apps/web/src/routes/login/+page.svelte`](../apps/web/src/routes/login/+page.svelte) & [`+page.server.ts`](../apps/web/src/routes/login/+page.server.ts) *(untuk alur login email + password standar, di luar alur step-up MFA)*.
  * [`apps/web/src/routes/forgot-password/`](../apps/web/src/routes/forgot-password/+page.svelte), [`reset-password/`](../apps/web/src/routes/reset-password/+page.svelte), [`resend-verification/`](../apps/web/src/routes/resend-verification/+page.svelte).
* **Layout & TypeScript Dasar**
  * [`apps/web/src/routes/+layout.svelte`](../apps/web/src/routes/+layout.svelte) & [`apps/web/src/app.d.ts`](../apps/web/src/app.d.ts): Menata UI layout, menampilkan notifikasi error/sukses, dan menuliskan tipe dasar `App.Locals`.

---

## 🎯 Kompetensi Kunci yang Harus Dikuasai di Level Ini

1. Bisa menjelaskan perjalanan satu HTTP Request dari tombol **Submit** di browser hingga tersimpan di PostgreSQL dan kembali ke layar.
2. Paham perbedaan HTTP Status Code (`400 Bad Request` vs `401 Unauthorized` vs `403 Forbidden` vs `500 Internal Server Error`).
3. Paham kenapa password **wajib** di-hash sebelum disimpan ke database (tidak boleh *plaintext*).
4. Bisa membaca pesan error compiler (seperti error TypeScript `svelte-check` atau error compile Go) dan memperbaikinya.

---

## 🚫 Yang Belum Diekspektasikan dari Junior
* Merancang O(1) Redis Set caching, GCRA rate limiter, atau atomic `token_version` revocation.
* Mengonfigurasi OpenTelemetry, Grafana Tempo, Prometheus, atau Loki.
* Membangun mekanisme TOTP 2FA RFC 6238 atau CSRF *Fetch Metadata* (*Sec-Fetch-Site*).
