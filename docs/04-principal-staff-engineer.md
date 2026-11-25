# 🏛️ Level 4: Staff / Principal Engineer

Pada level **Staff / Principal Engineer**, tanggung jawab tidak lagi hanya pada penulisan satu fitur, melainkan **arsitektur lintas sistem**, **observabilitas menyeluruh**, **keputusan trade-off teknis jangka panjang**, serta **standarisasi kualitas engineering** agar seluruh tim dapat bekerja cepat tanpa menurunkan reliabilitas dan keamanan.

---

## ✅ Bagian Project yang Diekspektasikan untuk Staff / Principal

### 1. Triple-Pillar Observability Architecture (Metrics, Traces, Logs)
Merancang sistem telemetri terintegrasi sehingga insiden di production dapat dilacak dari *edge* hingga *database query*:
* **Distributed Tracing (OpenTelemetry + Grafana Tempo)**
  * [`apps/api/shared/tracer.go`](../apps/api/shared/tracer.go) & [`infra/observability/tempo/tempo.yml`](../infra/observability/tempo/tempo.yml):
    * Menyiapkan W3C TraceContext propagation dan OTLP gRPC exporter (`:4317`) agar setiap request menghasilkan *trace span* yang terhubung antar-layer.
* **Metrics Instrumentation (Prometheus)**
  * [`apps/api/shared/metrics.go`](../apps/api/shared/metrics.go) & [`infra/observability/prometheus/prometheus.yml`](../infra/observability/prometheus/prometheus.yml):
    * Instrumentasi metrik RED (*Rate, Errors, Duration*) pada HTTP middleware menggunakan custom `responseWriter` wrapper tanpa mengganggu kontrak `http.ResponseWriter`.
* **Centralized Log Aggregation (Loki + Promtail + Structured `slog`)**
  * [`infra/observability/loki/loki.yml`](../infra/observability/loki/loki.yml) & [`infra/observability/promtail/promtail.yml`](../infra/observability/promtail/promtail.yml):
    * Memastikan log JSON terstruktur menyertakan `trace_id` dan `request_id` sehingga engineer bisa melompat langsung dari grafik spike latency di Prometheus $\to$ Trace di Tempo $\to$ Log spesifik di Loki melalui satu dashboard Grafana (`:3000`).

### 2. Keputusan Arsitektur & Trade-Off Strategis
* **Pragmatic Clean Modularity (Anti Over-Engineering)**
  * Struktur `cmd/` $\to$ `modules/auth/` $\to$ `shared/` $\to$ `shared/db/` memisahkan transport HTTP, domain rules, infrastruktur shared, dan persistence secara jelas tanpa menciptakan puluhan lapisan *abstraction* yang memperlambat pengembangan.
* **Compile-Time Type-Safe SQL Governance (`sqlc`)**
  * Memilih `sqlc` di [`apps/api/shared/db/`](../apps/api/shared/db/) daripada ORM berat (seperti GORM):
    * Menghilangkan risiko runtime SQL typo dan *N+1 query* tersembunyi akibat *lazy loading* ORM.
    * Memberikan performa eksekusi query mendekati *raw `pgx`* sekaligus menjaga keamanan tipe data Go saat kompilasi.
* **Strategi *Fail-Open* vs *Fail-Closed***
  * Menentukan kebijakan degradasi sistem saat Redis mengalami gangguan:
    * **Rate Limiter ([`apps/api/shared/limiter.go`](../apps/api/shared/limiter.go))**: Didesain **Fail-Open** — jika Redis down, user yang sah tetap bisa login agar bisnis tidak lumpuh total.
    * **Token Revocation & Auth**: Didesain **Fail-Closed / DB Fallback** agar celah keamanan tidak terbuka saat cache bermasalah.
* **Batas Keamanan BFF/SSR (`apps/web`) ↔ Go API (`apps/api`)**
  * Merancang arsitektur di mana SvelteKit server ([`apps/web/src/hooks.server.ts`](../apps/web/src/hooks.server.ts)) bertindak sebagai *Backend-for-Frontend (BFF)* yang aman: token sensitif tetap berada dalam `HttpOnly` cookies, *silent refresh* terjadi di sisi server sebelum rendering halaman, dan browser tidak pernah mengekspos rahasia token ke JavaScript client.

### 3. Ecosystem Tooling & Engineering Governance
* **Audit Library vs Custom Code**: Menilai kapan harus menulis kode sendiri demi kontrol keamanan/zero-allocation (seperti CSRF middleware & audit logger), dan kapan harus mengadopsi library teruji (`caarlos0/env`, `go-mail`, `pquerna/otp`, `sqlc`, `asynq`, `gobreaker`) agar tim fokus pada *business value* dan mengurangi beban *maintenance*.
* **Performance & Memory Awareness**: Memastikan keputusan kecil (seperti mengembalikan DTO *by value* untuk memanfaatkan Stack allocation dan menghindari tekanan Garbage Collector yang tidak perlu, serta *pre-allocation* slice di `ListUsers`/`ListAuditLogs`) menjadi standar kode tim.

---

## 🎯 Kompetensi Kunci yang Harus Dikuasai di Level Ini

1. Mampu merancang arsitektur observabilitas (*metrics + traces + logs*) yang mempercepat *Mean Time to Resolution (MTTR)* saat terjadi insiden di production.
2. Mampu mengambil keputusan trade-off yang terukur (contoh: *ORM vs `sqlc`*, *Fail-Open vs Fail-Closed*, *Stateless JWT vs Redis Hybrid Revocation*) dan menjelaskan risikonya ke manajemen maupun tim.
3. Mampu menjaga keseimbangan antara **keamanan enterprise**, **performa**, dan **kecepatan development (maintainability)** dalam satu codebase.
