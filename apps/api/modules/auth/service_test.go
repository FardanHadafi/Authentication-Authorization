package auth

import (
	"context"
	"sync"
	"testing"
	"time"

	"authentication-authorization/shared"
	"authentication-authorization/shared/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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

	svc := NewAuthService(mq, nil, nil, nil, nil, nil)

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
