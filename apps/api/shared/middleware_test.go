package shared

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockTokenManager untuk pengujian RequireAuth
type mockTokenManager struct {
	validateFn func(tokenString string) (*JWTClaims, error)
}

func (m *mockTokenManager) GenerateAccessToken(userID uuid.UUID, email, role string, tokenVersion int) (string, time.Time, error) {
	return "mock-access-token", time.Now().Add(15 * time.Minute), nil
}

type mockRevocationChecker struct {
	isRevokedFn func(ctx context.Context, userID uuid.UUID, tokenVersion int, jti string) (bool, error)
}

func (m *mockRevocationChecker) IsTokenRevoked(ctx context.Context, userID uuid.UUID, tokenVersion int, jti string) (bool, error) {
	if m.isRevokedFn != nil {
		return m.isRevokedFn(ctx, userID, tokenVersion, jti)
	}
	return false, nil
}

func (m *mockRevocationChecker) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, newVersion int) error {
	return nil
}

func (m *mockRevocationChecker) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}

func (m *mockTokenManager) GenerateRefreshToken() (string, error) {
	return "mock-refresh-token", nil
}

func (m *mockTokenManager) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	if m.validateFn != nil {
		return m.validateFn(tokenString)
	}
	return nil, errors.New("invalid token")
}

func TestRequestIDMiddleware(t *testing.T) {
	t.Run("creates new request ID when none provided", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		var capturedID string
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if capturedID == "" {
			t.Fatal("expected request ID in context, got empty string")
		}
		if rec.Header().Get("X-Request-ID") != capturedID {
			t.Fatalf("expected X-Request-ID header %q, got %q", capturedID, rec.Header().Get("X-Request-ID"))
		}
	})

	t.Run("preserves existing X-Request-ID header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("X-Request-ID", "custom-trace-id-123")
		rec := httptest.NewRecorder()

		var capturedID string
		handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedID = GetRequestID(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if capturedID != "custom-trace-id-123" {
			t.Fatalf("expected preserved request ID %q, got %q", "custom-trace-id-123", capturedID)
		}
		if rec.Header().Get("X-Request-ID") != "custom-trace-id-123" {
			t.Fatalf("expected response header %q, got %q", "custom-trace-id-123", rec.Header().Get("X-Request-ID"))
		}
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()

	handler := RecoveryMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("database connection exploded")
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !bytes.Contains([]byte(body), []byte("internal server error")) {
		t.Fatalf("expected 'internal server error' in body, got %s", body)
	}

	if !bytes.Contains(buf.Bytes(), []byte("panic recovered during request processing")) {
		t.Fatalf("expected panic logged, got: %s", buf.String())
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rec := httptest.NewRecorder()

	handler := LoggingMiddleware(logger)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"created":true}`))
	}))

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	logOutput := buf.String()
	if !bytes.Contains([]byte(logOutput), []byte("http request completed")) {
		t.Fatalf("expected log entry, got %s", logOutput)
	}
	if !bytes.Contains([]byte(logOutput), []byte(`"status":201`)) {
		t.Fatalf("expected status 201 in log, got %s", logOutput)
	}
}

func TestChain(t *testing.T) {
	var order []string

	mw1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw1_start")
			next.ServeHTTP(w, r)
			order = append(order, "mw1_end")
		})
	}

	mw2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "mw2_start")
			next.ServeHTTP(w, r)
			order = append(order, "mw2_end")
		})
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(finalHandler, mw1, mw2)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	chained.ServeHTTP(rec, req)

	expected := []string{"mw1_start", "mw2_start", "handler", "mw2_end", "mw1_end"}
	if len(order) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, order)
	}
	for i := range expected {
		if order[i] != expected[i] {
			t.Fatalf("at index %d: expected %s, got %s", i, expected[i], order[i])
		}
	}
}

func TestRequireAuth(t *testing.T) {
	userID := uuid.New()
	mockTM := &mockTokenManager{
		validateFn: func(tokenString string) (*JWTClaims, error) {
			if tokenString == "valid-token" {
				return &JWTClaims{
					UserID: userID,
					Email:  "user@example.com",
					Role:   "admin",
				}, nil
			}
			return nil, errors.New("invalid or expired token")
		},
	}

	t.Run("missing authorization header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		rec := httptest.NewRecorder()

		handler := RequireAuth(mockTM)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("malformed authorization header returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Basic 123456")
		rec := httptest.NewRecorder()

		handler := RequireAuth(mockTM)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("invalid token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		rec := httptest.NewRecorder()

		handler := RequireAuth(mockTM)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("valid token injects claims into context and returns 200", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		var capturedClaims *JWTClaims
		handler := RequireAuth(mockTM)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			capturedClaims = GetUserClaims(r.Context())
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if capturedClaims == nil {
			t.Fatal("expected claims in context, got nil")
		}
		if capturedClaims.Email != "user@example.com" || capturedClaims.Role != "admin" {
			t.Fatalf("unexpected claims: %+v", capturedClaims)
		}
	})

	t.Run("revoked token returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		rec := httptest.NewRecorder()

		mockRevocation := &mockRevocationChecker{
			isRevokedFn: func(ctx context.Context, uID uuid.UUID, v int, jti string) (bool, error) {
				return true, nil
			},
		}

		handler := RequireAuth(mockTM, mockRevocation)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401 for revoked token, got %d", rec.Code)
		}
	})
}

func TestRequireRole(t *testing.T) {
	t.Run("returns 401 when no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		rec := httptest.NewRecorder()

		handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("expected status 401, got %d", rec.Code)
		}
	})

	t.Run("returns 403 when role does not match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		claims := &JWTClaims{UserID: uuid.New(), Email: "user@example.com", Role: "member"}
		ctx := context.WithValue(req.Context(), UserClaimsKey, claims)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler := RequireRole("admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Fatalf("expected status 403, got %d", rec.Code)
		}
	})

	t.Run("allows access when role matches", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/admin", nil)
		claims := &JWTClaims{UserID: uuid.New(), Email: "admin@example.com", Role: "admin"}
		ctx := context.WithValue(req.Context(), UserClaimsKey, claims)
		req = req.WithContext(ctx)
		rec := httptest.NewRecorder()

		handler := RequireRole("admin", "superadmin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "welcome admin")
		}))

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "welcome admin" {
			t.Fatalf("unexpected body: %s", rec.Body.String())
		}
	})
}
