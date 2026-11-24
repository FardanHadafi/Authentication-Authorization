package shared

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateCSRFToken(t *testing.T) {
	token1, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(token1) != 64 {
		t.Errorf("expected 64 hex characters, got %d", len(token1))
	}

	token2, err := GenerateCSRFToken()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token1 == token2 {
		t.Errorf("expected unique tokens, got identical tokens")
	}
}

func TestCSRFMiddleware(t *testing.T) {
	cfg := CSRFConfig{
		AllowedOrigins:      []string{"http://localhost:5173", "http://localhost:8080"},
		AllowMissingOrigin:  true,
		EnforceDoubleSubmit: false,
		CookieName:          CSRFCookieName,
		HeaderName:          CSRFHeaderName,
		ExcludedPaths:       []string{"/metrics"},
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := CSRFMiddleware(cfg)(dummyHandler)

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		cookies        map[string]string
		expectedStatus int
	}{
		{
			name:           "GET request passes without headers",
			method:         http.MethodGet,
			path:           "/api/auth/me",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "OPTIONS preflight passes",
			method:         http.MethodOptions,
			path:           "/api/auth/login",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Excluded path passes even with unsafe method",
			method:         http.MethodPost,
			path:           "/metrics",
			headers:        map[string]string{"Sec-Fetch-Site": "cross-site"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Sec-Fetch-Site cross-site is rejected with 403",
			method:         http.MethodPost,
			path:           "/api/auth/reset-password",
			headers:        map[string]string{"Sec-Fetch-Site": "cross-site"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Sec-Fetch-Site same-origin passes",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Sec-Fetch-Site": "same-origin", "Origin": "http://localhost:5173"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Untrusted origin is rejected with 403",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://evil.com"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Allowed origin passes",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://localhost:5173"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Origin matching request Host passes",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://example.com", "Host": "example.com"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Untrusted referer is rejected with 403",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Referer": "http://attacker.com/steal"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Allowed referer passes",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Referer": "http://localhost:5173/login"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Double-submit cookie present but header missing returns 403",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://localhost:5173"},
			cookies:        map[string]string{CSRFCookieName: "validtoken123"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Double-submit cookie present and header mismatched returns 403",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://localhost:5173", CSRFHeaderName: "wrongtoken"},
			cookies:        map[string]string{CSRFCookieName: "validtoken123"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Double-submit cookie present and header matched returns 200",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://localhost:5173", CSRFHeaderName: "validtoken123"},
			cookies:        map[string]string{CSRFCookieName: "validtoken123"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Double-submit cookie matches fallback X-XSRF-TOKEN header returns 200",
			method:         http.MethodPost,
			path:           "/api/auth/login",
			headers:        map[string]string{"Origin": "http://localhost:5173", "X-XSRF-TOKEN": "validtoken123"},
			cookies:        map[string]string{CSRFCookieName: "validtoken123"},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			for k, v := range tt.headers {
				if k == "Host" {
					req.Host = v
				} else {
					req.Header.Set(k, v)
				}
			}
			for k, v := range tt.cookies {
				req.AddCookie(&http.Cookie{Name: k, Value: v})
			}

			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d (body: %s)", tt.expectedStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestCSRFMiddleware_EnforceDoubleSubmit(t *testing.T) {
	cfg := CSRFConfig{
		AllowedOrigins:      []string{"http://localhost:8080"},
		AllowMissingOrigin:  true,
		EnforceDoubleSubmit: true,
		CookieName:          CSRFCookieName,
		HeaderName:          CSRFHeaderName,
	}

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := CSRFMiddleware(cfg)(dummyHandler)

	// Tanpa cookie -> harus 403
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 when cookie missing with EnforceDoubleSubmit, got %d", rec.Code)
	}

	// Dengan cookie dan matching header -> harus 200
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	req2.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token-abc"})
	req2.Header.Set(CSRFHeaderName, "token-abc")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("expected 200 when matching cookie and header, got %d", rec2.Code)
	}
}
