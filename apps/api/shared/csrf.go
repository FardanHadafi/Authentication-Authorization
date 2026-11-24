package shared

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
)

type CSRFConfig struct {
	AllowedOrigins      []string
	AllowMissingOrigin  bool
	EnforceDoubleSubmit bool
	CookieName          string
	HeaderName          string
	ExcludedPaths       []string
}

func DefaultCSRFConfig() CSRFConfig {
	origins := []string{
		"http://localhost:5173",
		"http://localhost:8080",
		"http://127.0.0.1:5173",
		"http://127.0.0.1:8080",
	}

	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		origins = append(origins, strings.TrimRight(frontendURL, "/"))
	}
	if appURL := os.Getenv("APP_URL"); appURL != "" {
		origins = append(origins, strings.TrimRight(appURL, "/"))
	}
	if customOrigins := os.Getenv("CSRF_ALLOWED_ORIGINS"); customOrigins != "" {
		for o := range strings.SplitSeq(customOrigins, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				origins = append(origins, strings.TrimRight(trimmed, "/"))
			}
		}
	}

	enforceDoubleSubmit := os.Getenv("CSRF_ENFORCE_DOUBLE_SUBMIT") == "true"

	return CSRFConfig{
		AllowedOrigins:      origins,
		AllowMissingOrigin:  true,
		EnforceDoubleSubmit: enforceDoubleSubmit,
		CookieName:          CSRFCookieName,
		HeaderName:          CSRFHeaderName,
		ExcludedPaths:       []string{"/metrics"},
	}
}

func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

var safeHTTPMethods = []string{
	http.MethodGet,
	http.MethodHead,
	http.MethodOptions,
	http.MethodTrace,
}

func isSafeMethod(method string) bool {
	return slices.Contains(safeHTTPMethods, method)
}

func isOriginAllowed(rawOrigin, host string, allowedOrigins []string) bool {
	if rawOrigin == "" {
		return false
	}
	rawOrigin = strings.ToLower(strings.TrimRight(rawOrigin, "/"))

	// 1. Cek kecocokan dengan host server (same-origin default)
	host = strings.ToLower(host)
	if rawOrigin == "http://"+host || rawOrigin == "https://"+host {
		return true
	}

	// 2. Cek terhadap daftar whitelist menggunakan slices.ContainsFunc
	return slices.ContainsFunc(allowedOrigins, func(allowed string) bool {
		allowed = strings.ToLower(strings.TrimRight(allowed, "/"))
		return allowed == "*" || allowed == rawOrigin
	})
}

// CSRFMiddleware menyediakan proteksi CSRF Defense-in-Depth multi-lapis:
// Lapis 1: Validasi W3C Fetch Metadata (Sec-Fetch-Site: cross-site langsung di-reject)
// Lapis 2: Validasi Origin & Referer terhadap daftar domain/host terpercaya
// Lapis 3: Double-Submit Cookie (pencocokan token cookie _csrf dengan header X-CSRF-Token secara constant-time)
func CSRFMiddleware(cfg CSRFConfig) Middleware {
	if cfg.CookieName == "" {
		cfg.CookieName = CSRFCookieName
	}
	if cfg.HeaderName == "" {
		cfg.HeaderName = CSRFHeaderName
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Bypass untuk HTTP Safe Methods (GET, HEAD, OPTIONS, TRACE)
			if isSafeMethod(r.Method) {
				next.ServeHTTP(w, r)
				return
			}

			// 2. Bypass untuk Excluded Paths (seperti /metrics)
			if slices.Contains(cfg.ExcludedPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// 3. Layer 1: W3C Fetch Metadata Validation (Sec-Fetch-Site)
			secFetchSite := strings.ToLower(strings.TrimSpace(r.Header.Get("Sec-Fetch-Site")))
			if secFetchSite == "cross-site" {
				WriteError(w, http.StatusForbidden, "cross-site request blocked by CSRF defense")
				return
			}

			// 4. Layer 2: Origin & Referer Header Validation
			origin := strings.TrimSpace(r.Header.Get("Origin"))
			if origin != "" {
				if !isOriginAllowed(origin, r.Host, cfg.AllowedOrigins) {
					WriteError(w, http.StatusForbidden, "untrusted origin blocked by CSRF defense")
					return
				}
			} else {
				referer := strings.TrimSpace(r.Header.Get("Referer"))
				if referer != "" {
					refURL, err := url.Parse(referer)
					if err != nil || !isOriginAllowed(refURL.Scheme+"://"+refURL.Host, r.Host, cfg.AllowedOrigins) {
						WriteError(w, http.StatusForbidden, "untrusted referer blocked by CSRF defense")
						return
					}
				} else if !cfg.AllowMissingOrigin {
					WriteError(w, http.StatusForbidden, "missing origin and referer headers")
					return
				}
			}

			// 5. Layer 3: Double Submit Cookie Token Verification
			csrfCookie, err := r.Cookie(cfg.CookieName)
			if err == nil && csrfCookie.Value != "" {
				csrfHeader := strings.TrimSpace(r.Header.Get(cfg.HeaderName))
				if csrfHeader == "" {
					csrfHeader = strings.TrimSpace(r.Header.Get("X-XSRF-TOKEN"))
				}

				if csrfHeader == "" || subtle.ConstantTimeCompare([]byte(csrfCookie.Value), []byte(csrfHeader)) != 1 {
					WriteError(w, http.StatusForbidden, "invalid or missing CSRF token")
					return
				}
			} else if cfg.EnforceDoubleSubmit {
				WriteError(w, http.StatusForbidden, "CSRF cookie not found")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
