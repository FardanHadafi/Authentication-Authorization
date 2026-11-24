package shared

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"authentication-authorization/shared/db"

	"github.com/google/uuid"
)

type mockQuerierForRBAC struct {
	db.Querier
	rolePermissions map[string][]string
}

func (m *mockQuerierForRBAC) GetRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	if perms, ok := m.rolePermissions[roleID]; ok {
		return perms, nil
	}
	return []string{}, nil
}

func (m *mockQuerierForRBAC) CheckRoleHasPermission(ctx context.Context, arg db.CheckRoleHasPermissionParams) (bool, error) {
	if perms, ok := m.rolePermissions[arg.RoleID]; ok {
		for _, p := range perms {
			if p == arg.PermissionID {
				return true, nil
			}
		}
	}
	return false, nil
}

func TestPermissionChecker_HasPermission(t *testing.T) {
	mockQ := &mockQuerierForRBAC{
		rolePermissions: map[string][]string{
			"admin":  {"users:read", "users:write", "audit_logs:read"},
			"member": {"profile:read", "profile:write"},
		},
	}

	checker := NewPermissionChecker(mockQ, nil)
	ctx := context.Background()

	// Admin should have users:read
	hasPerm, err := checker.HasPermission(ctx, "admin", "users:read")
	if err != nil || !hasPerm {
		t.Errorf("expected admin to have users:read, got %v, err: %v", hasPerm, err)
	}

	// Member should NOT have users:read
	hasPerm, err = checker.HasPermission(ctx, "member", "users:read")
	if err != nil || hasPerm {
		t.Errorf("expected member NOT to have users:read, got %v, err: %v", hasPerm, err)
	}

	// Member should have profile:read
	hasPerm, err = checker.HasPermission(ctx, "member", "profile:read")
	if err != nil || !hasPerm {
		t.Errorf("expected member to have profile:read, got %v, err: %v", hasPerm, err)
	}

	// Empty role or permission should return false
	hasPerm, _ = checker.HasPermission(ctx, "", "profile:read")
	if hasPerm {
		t.Errorf("expected empty role to return false")
	}
}

func TestRequirePermission_Middleware(t *testing.T) {
	mockQ := &mockQuerierForRBAC{
		rolePermissions: map[string][]string{
			"admin":  {"users:read", "audit_logs:read"},
			"member": {"profile:read"},
		},
	}

	checker := NewPermissionChecker(mockQ, nil)

	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("access granted"))
	})

	protectedHandler := RequirePermission(checker, "users:read")(dummyHandler)

	t.Run("returns 401 when no claims in context", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("returns 403 when user lacks permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		claims := &JWTClaims{
			UserID: uuid.New(),
			Email:  "member@example.com",
			Role:   "member",
		}
		ctx := context.WithValue(req.Context(), UserClaimsKey, claims)
		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})

	t.Run("allows access when user has permission", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		claims := &JWTClaims{
			UserID: uuid.New(),
			Email:  "admin@example.com",
			Role:   "admin",
		}
		ctx := context.WithValue(req.Context(), UserClaimsKey, claims)
		rec := httptest.NewRecorder()
		protectedHandler.ServeHTTP(rec, req.WithContext(ctx))

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
