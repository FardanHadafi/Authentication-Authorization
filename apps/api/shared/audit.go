package shared

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"authentication-authorization/shared/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type AuditEntry struct {
	ActorID      *uuid.UUID
	ActorEmail   string
	ActorRole    string
	Action       string
	ResourceType string
	ResourceID   string
	Status       string // "success" or "failure"
	IPAddress    string
	UserAgent    string
	Metadata     map[string]any
}

type AuditLogger interface {
	Log(ctx context.Context, entry AuditEntry)
	Shutdown(ctx context.Context) error
}

type asyncAuditLogger struct {
	querier db.Querier
	logger  *slog.Logger
	queue   chan db.CreateAuditLogParams
	wg      sync.WaitGroup
	done    chan struct{}
	closed  bool
	mu      sync.Mutex
}

func NewAuditLogger(querier db.Querier, logger *slog.Logger, bufferSize int) AuditLogger {
	if bufferSize <= 0 {
		bufferSize = 1000
	}

	al := &asyncAuditLogger{
		querier: querier,
		logger:  logger,
		queue:   make(chan db.CreateAuditLogParams, bufferSize),
		done:    make(chan struct{}),
	}

	al.wg.Add(1)
	go al.worker()

	return al
}

func (al *asyncAuditLogger) worker() {
	defer al.wg.Done()

	for {
		select {
		case params, ok := <-al.queue:
			if !ok {
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			_, err := al.querier.CreateAuditLog(ctx, params)
			cancel()
			if err != nil && al.logger != nil {
				al.logger.Error("failed to write audit log to database", "action", params.Action, "error", err)
			}
		case <-al.done:
			// Drain sisa antrean sebelum selesai
			for {
				select {
				case params, ok := <-al.queue:
					if !ok {
						return
					}
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					_, _ = al.querier.CreateAuditLog(ctx, params)
					cancel()
				default:
					return
				}
			}
		}
	}
}

func (al *asyncAuditLogger) Log(ctx context.Context, entry AuditEntry) {
	if al == nil || al.querier == nil {
		return
	}

	al.mu.Lock()
	if al.closed {
		al.mu.Unlock()
		return
	}
	al.mu.Unlock()

	if ctx != nil {
		clientInfo := GetClientInfo(ctx)
		if entry.IPAddress == "" {
			entry.IPAddress = clientInfo.IP
		}
		if entry.UserAgent == "" {
			entry.UserAgent = clientInfo.UserAgent
		}
		if claims := GetUserClaims(ctx); claims != nil {
			if entry.ActorID == nil {
				entry.ActorID = &claims.UserID
			}
			if entry.ActorEmail == "" {
				entry.ActorEmail = claims.Email
			}
			if entry.ActorRole == "" {
				entry.ActorRole = claims.Role
			}
		}
	}

	if entry.IPAddress == "" {
		entry.IPAddress = "127.0.0.1"
	}
	if entry.UserAgent == "" {
		entry.UserAgent = "unknown"
	}

	var actorUUID pgtype.UUID
	if entry.ActorID != nil {
		actorUUID = pgtype.UUID{Bytes: *entry.ActorID, Valid: true}
	}

	status := entry.Status
	if status == "" {
		status = "success"
	}

	metaBytes := []byte("{}")
	if entry.Metadata != nil {
		if b, err := json.Marshal(entry.Metadata); err == nil {
			metaBytes = b
		}
	}

	params := db.CreateAuditLogParams{
		ActorID:      actorUUID,
		ActorEmail:   pgtype.Text{String: entry.ActorEmail, Valid: entry.ActorEmail != ""},
		ActorRole:    pgtype.Text{String: entry.ActorRole, Valid: entry.ActorRole != ""},
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   pgtype.Text{String: entry.ResourceID, Valid: entry.ResourceID != ""},
		Status:       status,
		IpAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		Metadata:     metaBytes,
	}

	select {
	case al.queue <- params:
	default:
		if al.logger != nil {
			al.logger.Warn("audit log queue full, dropping entry", "action", entry.Action)
		}
	}
}

func (al *asyncAuditLogger) Shutdown(ctx context.Context) error {
	al.mu.Lock()
	if al.closed {
		al.mu.Unlock()
		return nil
	}
	al.closed = true
	al.mu.Unlock()

	close(al.done)
	close(al.queue)

	ch := make(chan struct{})
	go func() {
		al.wg.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
