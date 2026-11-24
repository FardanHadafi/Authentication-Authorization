package shared

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis_rate/v10"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expectedIP string
	}{
		{
			name:       "CF-Connecting-IP takes highest precedence",
			headers:    map[string]string{"CF-Connecting-IP": "203.0.113.1", "X-Real-IP": "198.51.100.1"},
			remoteAddr: "192.0.2.1:12345",
			expectedIP: "203.0.113.1",
		},
		{
			name:       "X-Real-IP takes precedence over X-Forwarded-For and RemoteAddr",
			headers:    map[string]string{"X-Real-IP": "198.51.100.1", "X-Forwarded-For": "192.0.2.1, 10.0.0.1"},
			remoteAddr: "127.0.0.1:54321",
			expectedIP: "198.51.100.1",
		},
		{
			name:       "X-Forwarded-For takes first IP before comma",
			headers:    map[string]string{"X-Forwarded-For": " 203.0.113.50 , 10.0.0.2 "},
			remoteAddr: "127.0.0.1:54321",
			expectedIP: "203.0.113.50",
		},
		{
			name:       "Fallback to RemoteAddr stripping port",
			headers:    map[string]string{},
			remoteAddr: "192.168.1.100:8080",
			expectedIP: "192.168.1.100",
		},
		{
			name:       "Fallback to RemoteAddr when invalid format",
			headers:    map[string]string{},
			remoteAddr: "invalid-address-without-port",
			expectedIP: "invalid-address-without-port",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			for k, v := range tc.headers {
				req.Header.Set(k, v)
			}
			req.RemoteAddr = tc.remoteAddr

			ip := GetClientIP(req)
			if ip != tc.expectedIP {
				t.Fatalf("expected IP %s, got %s", tc.expectedIP, ip)
			}
		})
	}
}

func TestKeys(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/auth/me", nil)
	req.RemoteAddr = "192.168.1.50:4321"

	// 1. IPKey
	ipKeyFn := IPKey("login")
	if key := ipKeyFn(req); key != "rate:login:ip:192.168.1.50" {
		t.Fatalf("expected rate:login:ip:192.168.1.50, got %s", key)
	}

	// 2. UserKey without claims -> empty
	userKeyFn := UserKey("me")
	if key := userKeyFn(req); key != "" {
		t.Fatalf("expected empty key without claims, got %s", key)
	}

	// 3. UserKey with claims -> rate:me:user:<uuid>
	targetUUID := uuid.MustParse("01934f8a-6b5d-7a32-9a3b-2834b9d0e123")
	claims := &JWTClaims{UserID: targetUUID, Role: "member"}
	ctx := context.WithValue(req.Context(), UserClaimsKey, claims)
	reqWithClaims := req.WithContext(ctx)

	expectedUserKey := "rate:me:user:" + targetUUID.String()
	if key := userKeyFn(reqWithClaims); key != expectedUserKey {
		t.Fatalf("expected %s, got %s", expectedUserKey, key)
	}

	// 4. AutoKey with and without claims
	autoKeyFn := AutoKey("general")
	if key := autoKeyFn(req); key != "rate:general:ip:192.168.1.50" {
		t.Fatalf("expected rate:general:ip:192.168.1.50, got %s", key)
	}
	expectedAutoKey := "rate:general:user:" + targetUUID.String()
	if key := autoKeyFn(reqWithClaims); key != expectedAutoKey {
		t.Fatalf("expected %s, got %s", expectedAutoKey, key)
	}
}

func TestRateLimitMiddleware_NilLimiter(t *testing.T) {
	handler := RateLimitMiddleware(nil, IPKey("test"), 5, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 with nil limiter, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_EmptyKey(t *testing.T) {
	emptyKeyFn := func(r *http.Request) string { return "" }
	handler := RateLimitMiddleware(nil, emptyKeyFn, 5, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 with empty key, got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_FailOpenOnRedisError(t *testing.T) {
	// Koneksikan ke port redis yang mati
	deadClient := redis.NewClient(&redis.Options{Addr: "127.0.0.1:65530"})
	limiter := redis_rate.NewLimiter(deadClient)

	handler := RateLimitMiddleware(limiter, IPKey("test"), 5, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	handler.ServeHTTP(rec, req)

	// Harus fail-open (200 OK) dan tidak melempar panic / 500
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 on redis failure (fail-open), got %d", rec.Code)
	}
}

func TestRateLimitMiddleware_LimitAndExceeded(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	limiter := redis_rate.NewLimiter(rdb)

	limit := 3
	window := time.Minute
	handler := RateLimitMiddleware(limiter, IPKey("login"), limit, window)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest("POST", "/api/auth/login", nil)
	req.RemoteAddr = "192.0.2.10:9876"

	// Request 1: Allowed, Remaining = 2
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req)
	if rec1.Code != http.StatusOK {
		t.Fatalf("request 1: expected 200, got %d", rec1.Code)
	}
	if rec1.Header().Get("X-RateLimit-Limit") != "3" {
		t.Fatalf("request 1: expected X-RateLimit-Limit 3, got %s", rec1.Header().Get("X-RateLimit-Limit"))
	}
	if rec1.Header().Get("X-RateLimit-Remaining") != "2" {
		t.Fatalf("request 1: expected X-RateLimit-Remaining 2, got %s", rec1.Header().Get("X-RateLimit-Remaining"))
	}

	// Request 2: Allowed, Remaining = 1
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("request 2: expected 200, got %d", rec2.Code)
	}
	if rec2.Header().Get("X-RateLimit-Remaining") != "1" {
		t.Fatalf("request 2: expected X-RateLimit-Remaining 1, got %s", rec2.Header().Get("X-RateLimit-Remaining"))
	}

	// Request 3: Allowed, Remaining = 0
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req)
	if rec3.Code != http.StatusOK {
		t.Fatalf("request 3: expected 200, got %d", rec3.Code)
	}
	if rec3.Header().Get("X-RateLimit-Remaining") != "0" {
		t.Fatalf("request 3: expected X-RateLimit-Remaining 0, got %s", rec3.Header().Get("X-RateLimit-Remaining"))
	}

	// Request 4: Blocked (429 Too Many Requests)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req)
	if rec4.Code != http.StatusTooManyRequests {
		t.Fatalf("request 4: expected 429 Too Many Requests, got %d", rec4.Code)
	}
	if rec4.Header().Get("Retry-After") == "" {
		t.Fatalf("request 4: expected Retry-After header, got empty")
	}

	var errResp map[string]string
	if err := json.Unmarshal(rec4.Body.Bytes(), &errResp); err != nil {
		t.Fatalf("failed to parse error response: %v", err)
	}
	if errResp["error"] == "" {
		t.Fatalf("expected error message in response body, got %+v", errResp)
	}
}
