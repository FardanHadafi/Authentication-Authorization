# 🛡️ Level 3: Senior Backend / Security Engineer

Pada level **Senior**, fokus utama adalah **keamanan tingkat lanjut (defense-in-depth)**, **skalabilitas akses data**, **concurrency yang aman**, dan **ketahanan sistem (resilience)** saat komponen eksternal mengalami gangguan. Senior Engineer tidak hanya memakai library, tetapi memahami *threat model* dan mekanisme internal di baliknya.

---

## ✅ Bagian Project yang Diekspektasikan untuk Senior

### 1. Security Hardening & Protokol Kriptografi
* **Multi-Factor Authentication (RFC 6238 TOTP & Step-Up Auth)**
  * [`apps/api/shared/mfa.go`](../apps/api/shared/mfa.go) & [`apps/api/migrations/20261124233500_add_mfa_to_users.sql`](../apps/api/migrations/20261124233500_add_mfa_to_users.sql):
    * Generasi Base32 TOTP secret dan QR Code PNG secara *in-memory* (`pquerna/otp`) tanpa membocorkan secret ke layanan pihak ketiga.
    * Pembuatan 8 *recovery codes* sekali pakai (`XXXX-XXXX`) yang disimpan sebagai hash **SHA-256** di PostgreSQL (tidak pernah disimpan dalam bentuk plaintext).
    * Implementasi **Step-Up Authentication** (`Login` $\to$ short-lived `mfa_token` $\to$ `VerifyMFA`, serta proteksi ganda password + TOTP pada `DisableMFA`).
* **CSRF Defense-in-Depth (OWASP Modern Standard)**
  * [`apps/api/shared/csrf.go`](../apps/api/shared/csrf.go): Menggabungkan 3 lapis pertahanan:
    1. Pengecekan *Fetch Metadata* (`Sec-Fetch-Site: cross-site`) sebelum parsing token.
    2. Strict *Origin / Referer* allowlist validation.
    3. *Double-Submit Cookie* yang dibandingkan menggunakan `crypto/subtle.ConstantTimeCompare` untuk mencegah *timing side-channel attack*.
* **Anti-Enumeration & Password Hashing Hardening**
  * Implementasi Argon2id di [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go) serta pencegahan *user enumeration* pada endpoint `Login` dan `ForgotPassword`.

### 2. Otorisasi Granular (PBAC) & Immediate Token Revocation
* **Permission-Based Access Control (PBAC) + O(1) Caching**
  * [`apps/api/migrations/20261124232900_create_rbac_pbac_tables.sql`](../apps/api/migrations/20261124232900_create_rbac_pbac_tables.sql): Merancang skema relasional `roles`, `permissions`, dan `role_permissions`.
  * [`apps/api/shared/rbac.go`](../apps/api/shared/rbac.go): Middleware `RequirePermission("users:read")` yang melakukan lookup ke Redis Set (`role:perms:<role>`) dalam waktu **O(1)** (tanpa berulang kali melakukan JOIN query ke PostgreSQL pada setiap HTTP request).
* **Immediate JWT Revocation (Mengatasi Kelemahan Stateless JWT)**
  * [`apps/api/migrations/20261124230700_add_token_version_to_users.sql`](../apps/api/migrations/20261124230700_add_token_version_to_users.sql) & [`apps/api/shared/revocation.go`](../apps/api/shared/revocation.go):
    * Menambahkan `token_version` di DB yang di-cache di Redis. Saat password di-reset atau user diblokir, `token_version` dinaikkan sehingga Access Token lama langsung tertolak dalam hitungan sub-milidetik sebelum masa `exp`-nya habis.
  * **O(1) Session Indexing & Pipelining**:
    * Di [`apps/api/modules/auth/service.go`](../apps/api/modules/auth/service.go) (`storeRefreshToken`, `revokeAllUserRefreshTokens`), mengganti pola `SCAN` keyspace Redis (yang bermasalah di skala besar) dengan struktur *reverse set index* `user:<id>:refresh_tokens` dan mengeksekusi pembatalan massal dalam satu kali **Redis Pipeline**.

### 3. Concurrency, Compliance, & Resilience Patterns
* **Non-Blocking Async Audit Logger (SOC 2 Type II)**
  * [`apps/api/shared/audit.go`](../apps/api/shared/audit.go) & [`apps/api/migrations/20261124231400_create_audit_logs_table.sql`](../apps/api/migrations/20261124231400_create_audit_logs_table.sql):
    * Menggunakan *bounded Go channel* (`queue chan db.CreateAuditLogParams`), goroutine worker terpisah agar penulisan log audit ke DB tidak menambah latency HTTP response.
    * Menangani *graceful drain* saat server mati menggunakan `sync.WaitGroup`, `sync.Mutex`, dan channel `done`.
* **Distributed Rate Limiter (GCRA)**
  * [`apps/api/shared/limiter.go`](../apps/api/shared/limiter.go): Menerapkan algoritma *Generic Cell Rate Algorithm (GCRA)* berbasis Redis (`go-redis/redis_rate`) untuk melindungi endpoint sensitif dari brute-force dan abuse.
* **Circuit Breaker & Graceful Shutdown**
  * Circuit Breaker (`sony/gobreaker`) pada [`apps/api/shared/mail.go`](../apps/api/shared/mail.go) untuk mencegah *cascading failure* saat server SMTP bermasalah.
  * Penanganan sinyal `SIGINT`/`SIGTERM` dan drain koneksi HTTP + worker + audit logger selama 10 detik di [`apps/api/cmd/main.go`](../apps/api/cmd/main.go).

---

## 🎯 Kompetensi Kunci yang Harus Dikuasai di Level Ini

1. Memahami risiko nyata di balik JWT stateless (tidak bisa di-revoke) dan merancang solusi hybrid yang tetap cepat.
2. Mampu menulis kode concurrent di Go yang bebas *data race* dan bebas *goroutine leak* (memastikan channel/worker di-drain saat shutdown).
3. Memahami *timing attack*, CSRF modern, recovery code hashing, serta menghindari anti-pattern seperti Redis `SCAN` di hot-path.
4. Mampu merancang skema otorisasi yang fleksibel (PBAC) tanpa mengorbankan performa database.
