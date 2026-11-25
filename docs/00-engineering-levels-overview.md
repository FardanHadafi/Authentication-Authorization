# 🧭 Engineering Levels Breakdown — Project Overview

Dokumen ini memetakan seluruh komponen, modul, dan keputusan arsitektur dalam project **Authentication & Authorization** ini ke dalam empat tingkatan kompetensi industri: **Junior**, **Mid-Level**, **Senior**, dan **Principal / Staff Engineer**.

Tujuan pemetaan ini adalah memberikan gambaran **objektif dan realistis** mengenai ekspektasi industri terhadap masing-masing level, sekaligus menjadi peta jalan belajar (*learning roadmap*) bertahap.

---

## 📊 Ringkasan Pemetaan Komponen

| Tingkatan | Fokus Utama | Komponen & Fitur di Project Ini | Dokumen Rincian |
| :--- | :--- | :--- | :--- |
| **Junior Engineer** | Fundamental HTTP, CRUD, Validasi, Auth Dasar, & UI Form | Config `.env`, DTO & Validation tags, Handler & Routing dasar, Tabel `users` dasar, Register/Login/Profile dasar, Form & Zod schema SvelteKit | [01-junior-engineer.md](./01-junior-engineer.md) |
| **Mid-Level Engineer** | Production Readiness, Session Management, Testing, & Queue | HttpOnly Cookie management, Middleware pipeline dasar, Email worker (`Asynq` + `go-mail`), ACID Transaction (`RunInTx`), Unit Tests + Mocks, SSR `hooks.server.ts`, Admin Panel, Compose & CI | [02-mid-level-engineer.md](./02-mid-level-engineer.md) |
| **Senior Engineer** | Security Hardening, Kriptografi, Concurrency, & O(1) Caching | TOTP 2FA + QR Code + SHA-256 Recovery Codes, OWASP CSRF Defense-in-Depth, PBAC + O(1) Redis Set, Immediate JWT Revocation (`token_version`), Async Audit Worker (`channel` + `WaitGroup`), GCRA Rate Limiter, Circuit Breaker | [03-senior-engineer.md](./03-senior-engineer.md) |
| **Principal / Staff** | System Architecture, Observability, & Trade-Off Governance | Triple-Pillar Telemetry (OTel Tempo, Prometheus, Loki/Promtail), Clean Modularity + `sqlc` governance, Strategi *Fail-Open vs Fail-Closed*, Eliminasi Redis `SCAN` N+1, Batas Keamanan BFF SvelteKit ↔ Go API | [04-principal-staff-engineer.md](./04-principal-staff-engineer.md) |

---

## 💡 Cara Menggunakan Panduan Ini

1. **Jangan mencoba menguasai semuanya sekaligus.** Jika Anda berada di level Junior atau baru lulus, fokuslah pada `01-junior-engineer.md` terlebih dahulu sampai Anda bisa menjelaskan setiap barisnya tanpa melihat catatan.
2. **Gunakan sebagai bahan *Reverse Engineering*.** Buka file kode yang tercantum di setiap dokumen, baca alurnya, lalu coba tulis ulang versi sederhananya secara mandiri.
3. **Untuk Persiapan Interview:** Pastikan fitur yang Anda cantumkan di CV sesuai dengan kedalaman pemahaman Anda. Lebih baik menguasai penuh fitur level Junior–Mid dan memahaminya 100%, daripada mengklaim seluruh fitur Senior–Principal tetapi tidak bisa menjawab pertanyaan *deep-dive* saat sesi wawancara teknis.
