# 🚀 Level 2: Mid-Level Software / Fullstack Engineer

Pada level **Mid-Level**, fokus bergeser dari *"yang penting jalan di localhost"* menjadi *"aman, rapi, teruji, dan siap berjalan di production"*. Seorang Mid-Level Engineer diharapkan mampu menangani state session secara aman, menulis automated tests, mengintegrasikan background job, serta bekerja dengan containerization dan CI/CD.

---

## ✅ Bagian Project yang Diekspektasikan untuk Mid-Level

### 1. Backend (Go) — Production Readiness, Transaksi, & Background Job
* **Manajemen Cookie Aman**
  * [`apps/api/shared/cookie.go`](../apps/api/shared/cookie.go): Mengatur `HttpOnly`, `Secure`, `SameSite`, `Path`, dan `MaxAge` untuk Access Token, Refresh Token, dan CSRF Cookie agar token tidak rentan dicuri lewat XSS (`localStorage`).
* **HTTP Middleware Pipeline Dasar**
  * [`apps/api/shared/middleware.go`](../apps/api/shared/middleware.go):
    * `RequestID` middleware untuk melacak setiap request.
    * `Recovery` middleware untuk menangkap *panic* agar server tidak mati total saat terjadi runtime error.
    * Structured Request Logger menggunakan `log/slog` ([`apps/api/shared/logger.go`](../apps/api/shared/logger.go)).
    * `RequireAuth` dasar (ekstrak token dari cookie/header `Authorization` dan injeksi `UserClaims` ke `context.Context`).
    * Ekstraksi `ClientInfo` (IP Address & User-Agent).
* **ACID Database Transactions**
  * [`apps/api/shared/tx.go`](../apps/api/shared/tx.go): Implementasi `pgxTransactor` (`RunInTx`) agar operasi multi-query berjalan atomik (`BEGIN ... COMMIT`, dengan `defer tx.Rollback()`).
  * Contoh penerapan di `ResetPassword` dalam [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go): Mengupdate password dan menaikkan `token_version` dalam satu transaksi database.
* **Background Task Queue & Email Delivery**
  * [`apps/api/shared/task.go`](../apps/api/shared/task.go) & [`apps/api/shared/task_handler.go`](../apps/api/shared/task_handler.go): Menggunakan **Asynq** (berbasis Redis) untuk mengirim email verifikasi dan reset password secara *asynchronous* agar latency HTTP endpoint tetap cepat.
  * [`apps/api/shared/mail.go`](../apps/api/shared/mail.go): Integrasi SMTP menggunakan `go-mail`.
* **Refresh Token Rotation, Logout, & Pagination**
  * `RefreshToken` & `Logout` di [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go): Memahami konsep *token rotation* dan *blacklist* token di Redis.
  * `ListAuditLogs`: Implementasi pagination (`page`, `limit`, `offset`, `total_count`) dan parsing JSON metadata.
* **Automated Unit Testing dengan Mocking**
  * [`apps/api/modules/auth/service_test.go`](../apps/api/modules/auth/service_test.go) & `apps/api/shared/*_test.go`: Menulis unit test terisolasi menggunakan *Dependency Injection* dan manual mock (`mockQuerier`, `mockTransactor`, `mockTokenManager`).
* **Dokumentasi API**
  * [`docs/openapi.yaml`](../docs/openapi.yaml): Mendokumentasikan kontrak API (endpoint, request schema, response codes) sesuai standar OpenAPI 3.x.

### 2. Frontend (SvelteKit) — SSR Auth Guard & Admin Dashboard
* **Server-Side Authentication Hooks**
  * [`apps/web/src/hooks.server.ts`](../apps/web/src/hooks.server.ts): Mencegat setiap request SSR, memanggil backend `/profile` atau melakukan *silent refresh token* otomatis ketika access token expired, lalu meneruskan cookie baru ke browser.
* **Protected Routes & State Synchronization**
  * [`apps/web/src/routes/dashboard/+page.server.ts`](../apps/web/src/routes/dashboard/+page.server.ts) & [`+page.svelte`](../apps/web/src/routes/dashboard/+page.svelte): Guard akses halaman privat, typing `UserProfile` yang sinkron dengan backend, serta tampilan manajemen profil & aktivasi 2FA.
  * [`apps/web/src/routes/admin/+page.server.ts`](../apps/web/src/routes/admin/+page.server.ts) & [`+page.svelte`](../apps/web/src/routes/admin/+page.svelte): Membangun halaman admin dengan proteksi role/permission serta tabel paginated audit logs.

### 3. DevOps, Containerization, & CI/CD
* [`compose.yaml`](../compose.yaml) & [`compose.app.yaml`](../compose.app.yaml): Menjalankan seluruh dependency (PostgreSQL, Redis, Mailpit, API, Web) dalam container dengan healthcheck dan networking yang rapi.
* [`.github/workflows/ci.yml`](../.github/workflows/ci.yml): Mengatur pipeline CI otomatis untuk `go test`, `go build`, `bun run check`, dan `bun run build`.

---

## 🎯 Kompetensi Kunci yang Harus Dikuasai di Level Ini

1. Paham perbedaan menyimpan token di `HttpOnly Cookie` vs `localStorage` dari sisi keamanan (XSS vs CSRF).
2. Paham kapan harus memakai *Database Transaction* (`RunInTx`) agar tidak terjadi data korup/parsial (*split-brain*).
3. Paham alasan memindahkan proses lambat (seperti kirim email SMTP) ke *background worker* (Asynq).
4. Mampu menulis unit test yang menguji *happy path* maupun *error path* menggunakan mock interface.
