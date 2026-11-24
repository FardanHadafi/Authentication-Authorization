package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"authentication-authorization/shared"
	"authentication-authorization/shared/db"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
)

type mockQuerier struct {
	db.Querier
	auditLogs []db.AuditLog
	count     int64
}

func (m *mockQuerier) CountAuditLogs(ctx context.Context) (int64, error) {
	return m.count, nil
}

func (m *mockQuerier) ListAuditLogs(ctx context.Context, arg db.ListAuditLogsParams) ([]db.AuditLog, error) {
	return m.auditLogs, nil
}

type mockAuditLogger struct {
	mu      sync.Mutex
	entries []shared.AuditEntry
}

func (m *mockAuditLogger) Log(ctx context.Context, entry shared.AuditEntry) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.entries = append(m.entries, entry)
}

func (m *mockAuditLogger) Shutdown(ctx context.Context) error {
	return nil
}

func TestAuthService_ListAuditLogs(t *testing.T) {
	actorID := uuid.New()
	logID := uuid.New()
	sampleLogs := []db.AuditLog{
		{
			ID:           logID,
			ActorID:      pgtype.UUID{Bytes: actorID, Valid: true},
			ActorEmail:   pgtype.Text{String: "admin@example.com", Valid: true},
			ActorRole:    pgtype.Text{String: "admin", Valid: true},
			Action:       "auth.login",
			ResourceType: "session",
			ResourceID:   pgtype.Text{String: "sess-1", Valid: true},
			Status:       "success",
			IpAddress:    "127.0.0.1",
			UserAgent:    "Go-Test",
			Metadata:     []byte(`{"ip":"127.0.0.1"}`),
			CreatedAt:    time.Now(),
		},
	}

	mq := &mockQuerier{
		auditLogs: sampleLogs,
		count:     1,
	}

	svc := NewAuthService(mq, nil, nil, nil, nil, nil, nil)

	ctx := context.Background()
	res, err := svc.ListAuditLogs(ctx, 1, 20)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.TotalCount != 1 {
		t.Errorf("expected TotalCount 1, got %d", res.TotalCount)
	}

	if len(res.Data) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(res.Data))
	}

	entry := res.Data[0]
	if entry.ID != logID.String() {
		t.Errorf("expected log ID %s, got %s", logID.String(), entry.ID)
	}
	if entry.ActorID == nil || *entry.ActorID != actorID.String() {
		t.Errorf("expected actor ID %s, got %v", actorID.String(), entry.ActorID)
	}
	if entry.Action != "auth.login" {
		t.Errorf("expected action auth.login, got %s", entry.Action)
	}
	if entry.Status != "success" {
		t.Errorf("expected status success, got %s", entry.Status)
	}
	if entry.Metadata["ip"] != "127.0.0.1" {
		t.Errorf("expected metadata ip 127.0.0.1, got %v", entry.Metadata["ip"])
	}
}

type mockPermissionCheckerForService struct {
	perms []string
}

func (m *mockPermissionCheckerForService) HasPermission(ctx context.Context, role, permission string) (bool, error) {
	return true, nil
}

func (m *mockPermissionCheckerForService) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	return m.perms, nil
}

func TestAuthService_GetUserPermissions(t *testing.T) {
	pc := &mockPermissionCheckerForService{
		perms: []string{"users:read", "audit_logs:read"},
	}
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, pc)

	res, err := svc.GetUserPermissions(context.Background(), "admin")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if res.Role != "admin" {
		t.Errorf("expected role admin, got %s", res.Role)
	}
	if len(res.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(res.Permissions))
	}
}

type mockQuerierWithUser struct {
	db.Querier
	user       db.GetUserByIDRow
	mfaEnabled bool
	mfaSecret  string
}

func (m *mockQuerierWithUser) GetUserByID(ctx context.Context, id uuid.UUID) (db.GetUserByIDRow, error) {
	return m.user, nil
}

func (m *mockQuerierWithUser) EnableUserMFA(ctx context.Context, arg db.EnableUserMFAParams) error {
	m.mfaEnabled = true
	m.mfaSecret = arg.MfaSecret.String
	return nil
}

func TestAuthService_MFAFlow(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	userID := uuid.New()
	user := db.GetUserByIDRow{
		ID:         userID,
		Email:      "test@example.com",
		FullName:   "Test User",
		Role:       "member",
		Status:     "active",
		MfaEnabled: false,
	}

	mq := &mockQuerierWithUser{
		user: user,
	}

	svc := NewAuthService(mq, nil, nil, rdb, nil, nil, nil)
	ctx := context.Background()

	// 1. Setup MFA
	setupRes, err := svc.SetupMFA(ctx, userID)
	if err != nil {
		t.Fatalf("expected no error on SetupMFA, got %v", err)
	}

	if setupRes.Secret == "" {
		t.Errorf("expected secret to not be empty")
	}
	if len(setupRes.RecoveryCodes) != 8 {
		t.Errorf("expected 8 recovery codes, got %d", len(setupRes.RecoveryCodes))
	}

	// 2. Enable MFA with invalid code should fail
	err = svc.EnableMFA(ctx, userID, "000000")
	if err == nil {
		t.Errorf("expected invalid code to fail EnableMFA")
	}

	// 3. Enable MFA with valid code should succeed
	validCode, err := totp.GenerateCode(setupRes.Secret, time.Now())
	if err != nil {
		t.Fatalf("failed to generate code: %v", err)
	}

	err = svc.EnableMFA(ctx, userID, validCode)
	if err != nil {
		t.Fatalf("expected valid code to enable MFA, got %v", err)
	}

	if !mq.mfaEnabled {
		t.Errorf("expected user in DB to have MFA enabled")
	}
}
