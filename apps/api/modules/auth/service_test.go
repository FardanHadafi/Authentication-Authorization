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

	svc := NewAuthService(mq, nil, nil, nil, nil, nil, nil, nil)

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
	svc := NewAuthService(nil, nil, nil, nil, nil, nil, nil, pc)

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

	svc := NewAuthService(mq, nil, nil, nil, rdb, nil, nil, nil)
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

type mockResetPasswordQuerier struct {
	db.Querier
	user              db.GetUserByIDRow
	updatedPassword   string
	incrementedTimes  int
}

func (m *mockResetPasswordQuerier) GetUserByID(ctx context.Context, id uuid.UUID) (db.GetUserByIDRow, error) {
	return m.user, nil
}

func (m *mockResetPasswordQuerier) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	m.updatedPassword = arg.PasswordHash
	return nil
}

func (m *mockResetPasswordQuerier) IncrementUserTokenVersion(ctx context.Context, id uuid.UUID) (int32, error) {
	m.incrementedTimes++
	return 2, nil
}

type mockTransactor struct {
	txCalled bool
	q        db.Querier
}

func (m *mockTransactor) RunInTx(ctx context.Context, fn func(q db.Querier) error) error {
	m.txCalled = true
	return fn(m.q)
}

func TestAuthService_ResetPassword_AtomicTransactionAndTokenRevocation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	userID := uuid.New()
	user := db.GetUserByIDRow{
		ID:           userID,
		Email:        "reset@example.com",
		Status:       "active",
		TokenVersion: 1,
	}

	mq := &mockResetPasswordQuerier{user: user}
	mtx := &mockTransactor{q: mq}

	svc := NewAuthService(mq, mtx, nil, nil, rdb, nil, nil, nil)
	ctx := context.Background()

	// Simpan reset token di Redis
	resetToken := "valid-reset-token"
	_ = rdb.Set(ctx, "reset:"+resetToken, userID.String(), 15*time.Minute).Err()

	// Simpan 2 refresh token untuk user ini dalam set user:<id>:refresh_tokens
	_ = rdb.Set(ctx, "refresh:token1", userID.String(), 7*24*time.Hour).Err()
	_ = rdb.Set(ctx, "refresh:token2", userID.String(), 7*24*time.Hour).Err()
	_ = rdb.SAdd(ctx, "user:"+userID.String()+":refresh_tokens", "token1", "token2").Err()

	// Eksekusi ResetPassword
	err = svc.ResetPassword(ctx, ResetPasswordRequest{
		Token:       resetToken,
		NewPassword: "newSecurePassword123!",
	})
	if err != nil {
		t.Fatalf("expected ResetPassword to succeed, got %v", err)
	}

	// 1. Verifikasi transaksi database dipanggil
	if !mtx.txCalled {
		t.Errorf("expected ResetPassword to execute inside a database transaction")
	}
	if mq.updatedPassword == "" {
		t.Errorf("expected password to be updated in DB")
	}
	if mq.incrementedTimes != 1 {
		t.Errorf("expected IncrementUserTokenVersion to be called 1 time, got %d", mq.incrementedTimes)
	}

	// 2. Verifikasi reset token langsung terhapus (one-time token enforcement)
	exists, _ := rdb.Exists(ctx, "reset:"+resetToken).Result()
	if exists != 0 {
		t.Errorf("expected reset token to be consumed and deleted from redis")
	}

	// 3. Verifikasi token_version di-cache ke Redis
	versionVal, _ := rdb.Get(ctx, "user:"+userID.String()+":token_version").Result()
	if versionVal != "2" {
		t.Errorf("expected cached token_version to be 2, got %s", versionVal)
	}

	// 4. Verifikasi seluruh refresh token user terhapus secara batch
	exists1, _ := rdb.Exists(ctx, "refresh:token1").Result()
	exists2, _ := rdb.Exists(ctx, "refresh:token2").Result()
	if exists1 != 0 || exists2 != 0 {
		t.Errorf("expected all refresh tokens to be revoked from redis")
	}
	setLen, _ := rdb.SCard(ctx, "user:"+userID.String()+":refresh_tokens").Result()
	if setLen != 0 {
		t.Errorf("expected user refresh_tokens set to be deleted or empty, got count %d", setLen)
	}
}
