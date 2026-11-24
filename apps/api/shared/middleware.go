package shared

import (
	"context"
	"log/slog"
	"net/http"
	"runtime/debug"
	"slices"
	"strings"
	"time"
	"uuid"
)

type contextKey string

const RequestIDKey contextKey = "request_id"
const UserClaimsKey contextKey = "user_claims"

// Helper untuk mengambil request ID dari context manapun
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

func GetUserClaims(ctx context.Context) *JWTClaims {
	if claims, ok := ctx.Value(UserClaimsKey).(*JWTClaims); ok {
		return claims
	}
	return nil
}

// Menginjeksi / meneruskan X-Request-ID
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Ambil dari header request jika ada
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))

		// Jika kosong buat UUID baru
		if requestID == "" {
			requestID = uuid.NewV7().String()
		}

		// Set ke header response agar client menerima ID yang sama
		w.Header().Set("X-Request-ID", requestID)

		// Masukan ke context
		ctx := context.WithValue(r.Context(), RequestIDKey, requestID)

		// Teruskan request dengan context baru
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Ambil request ID dari context
					reqID := GetRequestID(r.Context())

					// Ambil stack trace lengkap
					stack := string(debug.Stack())

					// Catat ke server log dengan level ERROR terstruktur
					logger.Error("panic recovered during request processing",
						"request_id", reqID,
						"method", r.Method,
						"path", r.URL.Path,
						"client_ip", r.RemoteAddr,
						"panic", rec,
						"stack_trace", stack,
					)

					// Minta client / proxy menutup koneksi ini
					w.Header().Set("Connection", "close")

					// Kembalikan format JSON 500 yang aman ke client
					WriteError(w, http.StatusInternalServerError, "internal server error")
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	wroteHeader  bool
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	if rw, ok := w.(*responseWriter); ok {
		return rw
	}

	// Default status code di Go adalah 200 OK jika handler tidak memanggil WriteHeader secara eksplisit
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.wroteHeader {
		rw.statusCode = code
		rw.wroteHeader = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.wroteHeader {
		rw.WriteHeader(http.StatusOK)
	}

	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// Unwrap mendukung http.ResponseController (Standard Go 1.20+)
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// Mencatat log terstruktur untuk setiap HTTP Request
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Ambil request ID dari context
			reqID := GetRequestID(r.Context())

			// Ambil IP Client (cek proxy header jika ada)
			clientIP := r.Header.Get("X-Forwarded-For")
			if clientIP == "" {
				clientIP = r.RemoteAddr
			} else {
				clientIP = strings.Split(clientIP, ",")[0]
			}

			// Bungkus response writer untuk menyadap status code & ukuran response
			wrapped := newResponseWriter(w)

			// Teruskan ke handler berikutnya
			next.ServeHTTP(wrapped, r)

			// Hitung durasi request
			duration := time.Since(start)

			// Tentukan log level secara dinamis sesuai HTTP status
			msg := "http request completed"
			attrs := []any{
				"request_id", reqID,
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"duration_ms", duration.Milliseconds(),
				"bytes", wrapped.bytesWritten,
				"client_ip", clientIP,
				"user_agent", r.UserAgent(),
			}

			switch {
			case wrapped.statusCode >= 500:
				logger.Error(msg, attrs...)
			case wrapped.statusCode >= 400:
				logger.Warn(msg, attrs...)
			default:
				logger.Info(msg, attrs...)
			}
		})
	}
}

type Middleware func(http.Handler) http.Handler

// Chain menggabungkan beberapa middleware secara berurutan
func Chain(h http.Handler, middlewares ...Middleware) http.Handler {
	for _, mw := range slices.Backward(middlewares) {
		h = mw(h)
	}
	return h
}

// RequireAuth memvalidasi JWT Access Token dari header Authorization: Bearer <token>
func RequireAuth(tokenManager TokenManager) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
			if authHeader == "" {
				WriteError(w, http.StatusUnauthorized, "authorization header is required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				WriteError(w, http.StatusUnauthorized, "invalid authorization format, expected 'Bearer <token>'")
				return
			}

			tokenString := strings.TrimSpace(parts[1])
			claims, err := tokenManager.ValidateAccessToken(tokenString)
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "invalid or expired access token")
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole membatasi akses endpoint hanya untuk role tertentu (RBAC)
func RequireRole(allowedRoles ...string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetUserClaims(r.Context())
			if claims == nil {
				WriteError(w, http.StatusUnauthorized, "unauthorized: no user identity found")
				return
			}

			if !slices.Contains(allowedRoles, claims.Role) {
				WriteError(w, http.StatusForbidden, "forbidden: insufficient permissions")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
