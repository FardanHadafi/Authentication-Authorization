package shared

import (
	"context"
	"sync"
	"testing"
	"time"

	"authentication-authorization/shared/db"

	"github.com/google/uuid"
)

type mockQuerierForAudit struct {
	db.Querier
	mu      sync.Mutex
	created []db.CreateAuditLogParams
}

func (m *mockQuerierForAudit) CreateAuditLog(ctx context.Context, arg db.CreateAuditLogParams) (db.AuditLog, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.created = append(m.created, arg)
	return db.AuditLog{}, nil
}

func TestAuditLogger_AsyncLoggingAndShutdown(t *testing.T) {
	mockQ := &mockQuerierForAudit{}
	auditLogger := NewAuditLogger(mockQ, nil, 100)

	ctx := context.Background()
	actorID := uuid.New()

	auditLogger.Log(ctx, AuditEntry{
		ActorID:      &actorID,
		ActorEmail:   "admin@example.com",
		ActorRole:    "admin",
		Action:       "auth.login",
		ResourceType: "session",
		ResourceID:   "session-1",
		Status:       "success",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Go-Test",
		Metadata:     map[string]any{"method": "password"},
	})

	auditLogger.Log(ctx, AuditEntry{
		Action:       "auth.logout",
		ResourceType: "session",
		Status:       "success",
		IPAddress:    "127.0.0.1",
		UserAgent:    "Go-Test",
	})

	// Panggil shutdown yang menunggu antrean selesai di-flush
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := auditLogger.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("expected clean shutdown, got %v", err)
	}

	mockQ.mu.Lock()
	defer mockQ.mu.Unlock()

	if len(mockQ.created) != 2 {
		t.Fatalf("expected 2 audit logs recorded, got %d", len(mockQ.created))
	}

	if mockQ.created[0].Action != "auth.login" {
		t.Fatalf("expected action auth.login, got %s", mockQ.created[0].Action)
	}

	if mockQ.created[1].Action != "auth.logout" {
		t.Fatalf("expected action auth.logout, got %s", mockQ.created[1].Action)
	}
}
