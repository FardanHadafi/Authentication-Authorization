<div align="center">

# 🛡️ Enterprise Authentication & Authorization Engine

**A Production-Grade, High-Performance, Tier-1 Security & Identity Platform built in Go.**

[![Go Version](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-18-336791?style=for-the-badge&logo=postgresql)](https://www.postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-8.x-DC382D?style=for-the-badge&logo=redis)](https://redis.io)
[![OpenTelemetry](https://img.shields.io/badge/OpenTelemetry-Tracing-4D56D1?style=for-the-badge&logo=opentelemetry)](https://opentelemetry.io)
[![Prometheus](https://img.shields.io/badge/Prometheus-Metrics-E6522C?style=for-the-badge&logo=prometheus)](https://prometheus.io)
[![Grafana](https://img.shields.io/badge/Grafana-Observability-F46800?style=for-the-badge&logo=grafana)](https://grafana.com)
[![Tests Status](https://img.shields.io/badge/Tests-100%25%20Passing-brightgreen?style=for-the-badge)](https://github.com/FardanHadafi/Authentication-Authorization)
[![License](https://img.shields.io/badge/License-MIT-blue?style=for-the-badge)](LICENSE)

<br/>

[Key Highlights](#-why-this-is-not-just-another-tutorial-project) •
[Architecture](#-system-architecture) •
[Core Features](#-core-enterprise-capabilities) •
[Tech Stack](#-technology-stack) •
[API Reference](#-api-endpoints-summary) •
[Getting Started](#-getting-started) •
[Observability](#-observability--monitoring)

---

</div>

## 📌 Executive Summary

Modern backend architectures require more than standard username/password authentication. In fintech, healthcare, and enterprise SaaS environments, security compliance standards (such as **SOC 2, ISO 27001, and GDPR**) mandate strict identity verification, immediate token revocation, granular access control, tamper-evident audit trails, and zero-trust transport defense.

This repository implements a **Tier-1 Production-Ready Identity & Access Management (IAM) Engine** engineered in Go. It moves beyond standard educational tutorials by implementing defense-in-depth security patterns, high-concurrency caching, asynchronous background queues, compile-time type-safe queries, and triple-pillar observability (Metrics, Traces, and Logs).

---

## ⚡ Why This Is Not "Just Another Tutorial Project"

Most web authentication tutorials leave severe architectural security and performance flaws unaddressed. Here is how this enterprise implementation solves them:

| Challenge / Vulnerability | Standard Tutorial Approach | **This Enterprise Architecture** |
| :--- | :--- | :--- |
| **Password Storage** | Bcrypt (GPU-crackable, length limits) | **Argon2id (RFC 9106)**: Memory-hard hashing resistant to GPU/ASIC attacks. |
| **Multi-Factor Auth (MFA)** | None | **RFC 6238 TOTP (Google Authenticator)** + 8 SHA-256 hashed single-use recovery codes. |
| **Stolen JWT Tokens** | Token valid until 15m expiration | **Immediate Revocation**: PostgreSQL `token_version` + Redis JTI blacklist check in middleware. |
| **Authorization Model** | Macro `if role == 'admin'` | **Granular PBAC**: Relational `roles`, `permissions`, and `role_permissions` with O(1) Redis Set caching. |
| **CSRF & Cross-Site Attacks** | Relies solely on `SameSite` cookies | **Defense-in-Depth**: `Sec-Fetch-Site: cross-site` drop + Origin whitelist + Constant-Time Double Submit Cookie. |
| **Security Compliance Logs** | Ephemeral console `fmt.Println` | **Immutable Audit Trail**: Async non-blocking worker queue to PostgreSQL `audit_logs` table. |
| **Database Mutations** | Autocommit multi-statement queries | **ACID Database Transactions (`RunInTx`)**: Atomic rollbacks on multi-table updates. |
| **Redis Session Scaling** | Blocking `SCAN` with N round-trips | **O(1) Token Set Indexing**: Pipelined batch revocation via `user:<id>:refresh_tokens`. |
| **System Observability** | Raw unstructured console logs | **Triple-Pillar Telemetry**: Prometheus metrics, OpenTelemetry traces to Tempo, Loki JSON logs to Grafana. |
| **Email & Network Reliability**| Unhandled SMTP timeouts / crashes | **Circuit Breaker (`gobreaker`)** + Asynq Redis background task queue + 10s graceful shutdown drain. |

---

## 🏗️ System Architecture

The application is structured following Clean Architecture principles and idiomatic Go modularity, separating transport protocols, business domain rules, database persistence, and observability telemetry.

```mermaid
flowchart TD
    Client["Client / Frontend\n(SvelteKit / SPA / Mobile)"]

    subgraph MiddlewareChain ["HTTP Middleware Pipeline (Zero-Allocation Chain)"]
        M1["RequestID Middleware\n(UUID injection & propagation)"]
        M2["OpenTelemetry Tracing\n(W3C TraceContext & Span injection)"]
        M3["Prometheus Metrics\n(Latency, requests_total, status_codes)"]
        M4["Panic Recovery\n(Structured error logging with stacktrace)"]
        M5["Structured Logger\n(slog JSON with duration & client IP)"]
        M6["CSRF Defense-in-Depth\n(Sec-Fetch-Site + Origin + Double Submit)"]
        M7["GCRA Rate Limiter\n(Redis Token Bucket / Fail-Open)"]
        M8["RequireAuth & Immediate Revocation\n(JWT decode + Redis token_version check)"]
        M9["RequirePermission PBAC\n(O(1) Redis Set cache lookup)"]
        M1 --> M2 --> M3 --> M4 --> M5 --> M6 --> M7 --> M8 --> M9
    end

    subgraph ServiceLayer ["Business & Domain Services"]
        AuthHandler["Auth HTTP Handler\n(Input Validation & DTO Mapping)"]
        AuthService["Auth Service Layer\n(Domain Rules & Step-Up Auth)"]
        Transactor["Database Transactor\n(Atomic RunInTx Engine)"]
        AuditWorker["Async Audit Logger\n(Buffer Queue & Batch Flushing)"]
    end

    subgraph StorageLayer ["High-Performance Storage Layer"]
        Postgres[("PostgreSQL 18\n(Users, Roles, Permissions,\nAudit Logs via sqlc)")]
        Redis[("Redis 8 Cache & KV\n(Token Versions, PBAC Sets,\nBlacklist, Session Index)")]
        AsynqQueue["Asynq Task Queue\n(Transactional Email Worker)"]
    end

    subgraph TelemetryStack ["Triple-Pillar Observability Stack"]
        Prometheus["Prometheus TSDB\n(Scrapes :8080/metrics)"]
        Tempo["Grafana Tempo\n(Distributed Traces via OTLP gRPC :4317)"]
        Loki["Grafana Loki\n(Log Ingestion via Promtail)"]
        Grafana["Grafana Dashboards\n(Unified Observability UI :3000)"]
    end

    Client -->|HTTP Request| MiddlewareChain
    MiddlewareChain --> AuthHandler
    AuthHandler --> AuthService
    AuthService --> Transactor
    Transactor --> Postgres
    AuthService --> Redis
    AuthService --> AsynqQueue
    AuthService --> AuditWorker
    AuditWorker -.->|Async Non-Blocking| Postgres

    M2 -.->|Export Spans| Tempo
    M3 -.->|Metrics| Prometheus
    M5 -.->|Structured JSON| Loki
    Prometheus --> Grafana
    Tempo --> Grafana
    Loki --> Grafana
```

---

## 🚀 Core Enterprise Capabilities

### 1. Multi-Factor Authentication (MFA / 2FA)
* **RFC 6238 TOTP Standard**: Generates standard `otpauth://totp/...` URIs and Base32 secrets compatible with **Google Authenticator**, Microsoft Authenticator, Apple Keychain, 1Password, and Bitwarden.
* **8 Emergency Recovery Codes**: Plain codes (`XXXX-XXXX`) generated once during setup; stored in PostgreSQL as cryptographically secure **SHA-256 hashes**. Supports atomic single-use consumption.
* **Step-Up Challenge Flow**: When MFA is active, initial login issues a short-lived (5 min) challenge token (`mfa_required: true`, `mfa_token: "..."`). Tokens are verified at `POST /api/auth/mfa/verify` before issuing access/refresh credentials.
* **Protected Deactivation**: Disabling MFA requires dual-factor confirmation (Argon2id password verification + active 6-digit TOTP passcode).

### 2. Immediate JWT Token Revocation (Zero-Delay Session Invalidation)
* Solves the fundamental security flaw of stateless JWTs where stolen tokens remain valid until expiration.
* Each user record possesses an atomic `token_version` counter in PostgreSQL, cached in Redis with a 30-day TTL.
* When a user changes passwords, triggers logout, or an admin bans the account, `token_version` is bumped. Any existing Access Token with an older `token_version` claim is **immediately rejected in <1 millisecond**.

### 3. Granular PBAC (Permission-Based Access Control)
* Replaces rigid binary roles (`admin` vs `member`) with an enterprise-grade PBAC relational schema:
  - `roles (id, name, description)`
  - `permissions (id, name, description)`
  - `role_permissions (role_id, permission_id)`
* Evaluated dynamically via `shared.RequirePermission("users:read")` middleware.
* **O(1) Redis Caching**: Permissions are cached as Redis Sets (`role:perms:<role>`). Permissions are checked in microsecond timeframes without triggering repetitive database queries.

### 4. CSRF Defense-in-Depth (OWASP Modern Standard)
* **Fetch Metadata Drop**: Immediately drops unsafe cross-site requests by inspecting the modern browser `Sec-Fetch-Site: cross-site` header before token parsing.
* **Origin / Referer Strict Whitelisting**: Verifies against authorized origins and host headers.
* **Double-Submit Cookie Pattern**: Validates anti-CSRF request headers against secure cookies using `crypto/subtle.ConstantTimeCompare` to eliminate side-channel timing attacks.

### 5. Tamper-Evident Security Audit Logging
* Compliant with **SOC 2 Type II** and **GDPR** audit standards.
* Records security-critical events (login successes/failures, password resets, MFA activations, permission checks, and administrative user mutations) in a dedicated PostgreSQL `audit_logs` table.
* **Non-Blocking Asynchronous Processing**: Log entries are buffered via bounded Go channels and written by background worker routines, ensuring HTTP response latencies are completely unaffected.
* Dedicated administrative inspection endpoint with pagination: `GET /api/auth/admin/audit-logs`.

### 6. ACID Database Transactions & O(1) Redis Session Indexing
* **Database Transactor**: Encapsulates `pgxpool.Pool` inside `RunInTx(ctx, func(q db.Querier) error)`. Multi-table mutations (e.g., updating user password + bumping token version) run within atomic `BEGIN ... COMMIT` blocks with automatic `defer tx.Rollback()`.
* **Zero N+1 Query Problem**: Relational queries use compile-time type-safe SQL generated by `sqlc`.
* **Zero N+1 Redis Network Round-Trips**: Replaced blocking `SCAN` keyspace searches with bidirectional Redis Set indexing (`user:<id>:refresh_tokens`). Multi-session revocation executes in **1 single batched Redis Pipeline roundtrip**.

---

## 🛠️ Technology Stack

| Layer | Technology | Key Details & Purpose |
| :--- | :--- | :--- |
| **Language & Runtime** | **Go 1.23+** | High-concurrency, memory-efficient compiled systems language. |
| **Web Server / Routing** | **Standard Library `net/http`** | Zero external framework bloat; uses Go 1.22+ pattern routing. |
| **Primary Database** | **PostgreSQL 18** | Relational data persistence with strict foreign keys & indexes. |
| **Query Generator** | **sqlc** + **pgx/v5** | Compile-time type-safe raw SQL; zero runtime reflection, zero ORM overhead. |
| **Cache & Session Store**| **Redis 8 (Alpine)** | In-memory key-value caching, Redis Sets, and GCRA token rate-limiting. |
| **Cryptography** | **Argon2id (`alexedwards/argon2id`)** | Modern password hashing parameterized to OWASP recommendations. |
| **2FA / MFA** | **TOTP (`pquerna/otp`)** | RFC 6238 time-based one-time passwords for Google Authenticator. |
| **Email Delivery** | **go-mail (`wneessen/go-mail`)** | Modern type-safe MIME email client wrapped in a Circuit Breaker. |
| **Fault Tolerance** | **Circuit Breaker (`sony/gobreaker/v2`)**| Prevents cascading network failures during SMTP server degradation. |
| **Background Tasks** | **Asynq (`hibiken/asynq`)** | Redis-backed asynchronous queue for transactional emails. |
| **Configuration** | **caarlos0/env/v11** | Clean, declarative, type-safe environment variable parsing with validation. |
| **Metrics & Telemetry** | **Prometheus & Grafana** | Real-time time-series telemetry on `/metrics` with pre-built dashboards. |
| **Distributed Tracing** | **OpenTelemetry & Tempo** | End-to-end W3C trace propagation via gRPC (`:4317`) to Grafana Tempo. |
| **Structured Logging** | **Go `log/slog` & Loki** | High-performance JSON logging ingested into Grafana Loki via Promtail. |
| **Local Mail Sandbox** | **Mailpit** | Local SMTP test server (`:1025`) with interactive web UI (`:8025`). |

---

## 📋 API Endpoints Summary

All routes are fully documented in OpenAPI 3.0 specification ([`docs/openapi.yaml`](docs/openapi.yaml)).

### Public & Authentication Endpoints
| Method | Endpoint | Protection / Middleware | Purpose |
| :--- | :--- | :--- | :--- |
| `POST` | `/api/auth/register` | Rate Limited (GCRA) | Register new user account; enqueues verification email. |
| `GET` | `/api/auth/verify` | Public | Verify user account email address from link token. |
| `POST` | `/api/auth/verify/resend` | Rate Limited (GCRA) | Resend email verification link token. |
| `POST` | `/api/auth/login` | Rate Limited (GCRA) | Authenticate user; returns JWTs or triggers MFA challenge. |
| `POST` | `/api/auth/mfa/verify` | Rate Limited (GCRA) | Verify MFA challenge via 6-digit TOTP or backup recovery code. |
| `POST` | `/api/auth/refresh` | Public | Refresh token rotation (RTR) to issue new access & refresh pairs. |
| `POST` | `/api/auth/forgot-password`| Rate Limited (Anti-Enumeration)| Request password reset email. |
| `POST` | `/api/auth/reset-password` | Atomic Database Transaction | Reset password and revoke all active sessions. |

### Authenticated & Security Endpoints
| Method | Endpoint | Protection / Middleware | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/auth/me` | `RequireAuth` (Revocation checked)| Retrieve authenticated user profile. |
| `POST` | `/api/auth/logout` | `RequireAuth` (Revocation checked)| Revoke current session refresh token and delete cache. |
| `POST` | `/api/auth/mfa/setup` | `RequireAuth` | Generate TOTP secret, QR URL, and 8 emergency backup codes. |
| `POST` | `/api/auth/mfa/enable` | `RequireAuth` | Confirm first TOTP code and permanently activate 2FA. |
| `POST` | `/api/auth/mfa/disable` | `RequireAuth` + Password Verify | Disable 2FA with dual password & TOTP confirmation. |
| `GET` | `/api/auth/me/permissions`| `RequireAuth` | Retrieve all granular PBAC permissions for active role. |

### Administration & Operations
| Method | Endpoint | Protection / Middleware | Purpose |
| :--- | :--- | :--- | :--- |
| `GET` | `/api/auth/users` | `RequirePermission("users:read")` | List registered users with pagination. |
| `GET` | `/api/auth/admin/audit-logs`| `RequirePermission("audit_logs:read")`| Query immutable SOC 2 compliance audit logs. |
| `GET` | `/metrics` | Prometheus Scraper | Export real-time Prometheus HTTP & runtime metrics. |

---

## 🚦 Getting Started

### Prerequisites
* **Go 1.23+** installed locally
* **Podman** or **Docker** with Compose support
* **Goose** CLI (for database migrations):
  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

### 1. Clone & Configure Environment
```bash
git clone https://github.com/FardanHadafi/Authentication-Authorization.git
cd Authentication-Authorization

# Setup environment configuration
cp apps/api/.env.example apps/api/.env
```

### 2. Start Infrastructure Containers
Spin up all 8 background services (Postgres, Redis, Mailpit, Prometheus, Tempo, Loki, Promtail, Grafana):
```bash
# Using Podman
podman compose -f compose.yaml up -d

# Or using Docker
docker compose -f compose.yaml up -d
```

Verify all containers are healthy:
```bash
podman ps   # or docker ps
```

### 3. Run Database Migrations
Apply versioned PostgreSQL migrations using Goose:
```bash
export DB_URL="postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable"
goose -dir apps/api/migrations postgres "$DB_URL" up
```

### 4. Run Automated Test Suite
Execute all unit and integration tests (asserting 100% pass rate):
```bash
cd apps/api
go test -v ./...
```

### 5. Launch the Application
```bash
cd apps/api
go run ./cmd
```
The server will start listening on port `:8080`.

---

## 📊 Observability & Monitoring

The repository ships with an enterprise-ready observability stack pre-configured in `compose.yaml`:

| Service | Port | Type & Access | Description |
| :--- | :--- | :--- | :--- |
| **Grafana Dashboard** | [`http://localhost:3000`](http://localhost:3000) | **Interactive Web UI** | Unified visual dashboard for Metrics, Logs, and Traces (login: `admin`/`admin`). |
| **Mailpit Web UI** | [`http://localhost:8025`](http://localhost:8025) | **Interactive Web UI** | Local SMTP mailbox inspector to view verification & password reset emails. |
| **Prometheus TSDB** | [`http://localhost:9090`](http://localhost:9090) | **Web UI & Scraper** | Time-series database scraping `/metrics` directly. |
| **Grafana Tempo** | `:3200` (API) / `:4317` (OTLP) | **Headless Storage Engine** | Distributed tracing backend. *(No HTML UI at root `/`; traces are visualized in Grafana).* |
| **Grafana Loki** | `:3100` (API) | **Headless Log Engine** | High-performance log aggregation engine. *(No HTML UI at root `/`; logs are queried in Grafana).* |

---

## 🏛️ Engineering & Architecture Decisions (ADR)

* **Why `sqlc` instead of an ORM like GORM?**  
  ORMs introduce runtime reflection overhead, complex abstraction leaks, and insidious N+1 query patterns. `sqlc` compiles raw SQL into type-safe Go code with zero runtime cost and 100% query visibility.
* **Why `caarlos0/env/v11` for configuration?**  
  Centralizes environment variables into a single, type-safe Go struct with built-in validation and default values, eliminating fragile manual `os.Getenv` string manipulation throughout the codebase.
* **Why `wneessen/go-mail` with Circuit Breaker?**  
  Provides clean MIME builder ergonomics, native STARTTLS negotiation, and protects backend workers from cascading hanging connections when an upstream SMTP server becomes unresponsive.
* **Why Double-Submit Cookie + `Sec-Fetch-Site` over Legacy `gorilla/csrf`?**  
  `gorilla/csrf` was built a decade ago for monolith SSR web apps using encrypted session cookies. For modern SPAs and stateless APIs, combining modern browser Fetch Metadata headers (`Sec-Fetch-Site`) with constant-time Double Submit cookies delivers superior OWASP protection without legacy session baggage.

---

## 👨‍💻 Author

**Fardan Hadafi**
* **GitHub**: [@FardanHadafi](https://github.com/FardanHadafi)
* **Email**: [fardan.hadafi@yahoo.com](mailto:fardan.hadafi@yahoo.com)

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
