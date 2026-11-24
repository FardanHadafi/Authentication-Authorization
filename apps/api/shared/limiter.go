package shared

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis_rate/v10"
)

type KeyFunc func(r *http.Request) string

func GetClientIP(r *http.Request) string {
	if cfIP := r.Header.Get("CF-Connecting-IP"); cfIP != "" {
		return cfIP
	}

	if xRealIP := r.Header.Get("X-Real-IP"); xRealIP != "" {
		return xRealIP
	}

	if xForwarded := r.Header.Get("X-Forwarded-For"); xForwarded != "" {
		return strings.TrimSpace(strings.Split(xForwarded, ",")[0])
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// IPKey menghasilkan kunci rate limit berdasarkan IP murni
func IPKey(prefix string) KeyFunc {
	return func(r *http.Request) string {
		return fmt.Sprintf("rate:%s:ip:%s", prefix, GetClientIP(r))
	}
}

// UserKey menghasilkan kunci rate limit berdasarkan User ID dari JWT claims
func UserKey(prefix string) KeyFunc {
	return func(r *http.Request) string {
		if claims := GetUserClaims(r.Context()); claims != nil {
			return fmt.Sprintf("rate:%s:user:%s", prefix, claims.UserID.String())
		}
		return ""
	}
}

// AutoKey memilih User ID jika sudah login, atau Client IP jika belum login
func AutoKey(prefix string) KeyFunc {
	return func(r *http.Request) string {
		if claims := GetUserClaims(r.Context()); claims != nil {
			return fmt.Sprintf("rate:%s:user:%s", prefix, claims.UserID.String())
		}

		return fmt.Sprintf("rate:%s:ip:%s", prefix, GetClientIP(r))
	}
}

// RateLimitMiddleware membatasi jumlah request menggunakan algoritma GCRA Redis (redis_rate)
func RateLimitMiddleware(limiter *redis_rate.Limiter, keyFunc KeyFunc, limit int, window time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if limiter == nil {
				next.ServeHTTP(w, r)
				return
			}

			key := keyFunc(r)
			if key == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Evaluasi kuota di Redis via GCRA
			res, err := limiter.Allow(r.Context(), key, redis_rate.Limit{
				Rate:   limit,
				Period: window,
				Burst:  limit,
			})
			if err != nil {
				// Fail-open: Izinkan request jika koneksi Redis bermasalah
				next.ServeHTTP(w, r)
				return
			}

			// Pasang header standar IETF RateLimit
			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(res.ResetAfter).Unix(), 10))

			// Jika kuota habis, tolak dengan 429 Too Many Requests
			if res.Allowed == 0 {
				retryAfterSecs := int(res.RetryAfter.Seconds())
				if retryAfterSecs < 1 {
					retryAfterSecs = 1
				}
				w.Header().Set("Retry-After", strconv.Itoa(retryAfterSecs))

				WriteError(w, http.StatusTooManyRequests, fmt.Sprintf("too many requests, please try again in %d seconds", retryAfterSecs))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
