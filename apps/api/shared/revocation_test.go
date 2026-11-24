package shared

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func TestTokenRevocationChecker(t *testing.T) {
	s := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	checker := NewRevocationChecker(rdb)
	ctx := context.Background()
	userID := uuid.New()
	jti := uuid.New().String()

	// 1. Initial state: token version 1, not revoked
	revoked, err := checker.IsTokenRevoked(ctx, userID, 1, jti)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revoked {
		t.Fatalf("expected token not revoked initially")
	}

	// 2. Blacklist JTI
	if err := checker.BlacklistJTI(ctx, jti, time.Hour); err != nil {
		t.Fatalf("failed to blacklist JTI: %v", err)
	}

	revoked, err = checker.IsTokenRevoked(ctx, userID, 1, jti)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revoked {
		t.Fatalf("expected token with blacklisted JTI to be revoked")
	}

	// 3. New token with different JTI should not be revoked yet
	newJTI := uuid.New().String()
	revoked, err = checker.IsTokenRevoked(ctx, userID, 1, newJTI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revoked {
		t.Fatalf("expected new token not to be revoked")
	}

	// 4. Global revocation: bump token_version to 2
	if err := checker.RevokeAllUserTokens(ctx, userID, 2); err != nil {
		t.Fatalf("failed to revoke all user tokens: %v", err)
	}

	// Token with version 1 is now revoked!
	revoked, err = checker.IsTokenRevoked(ctx, userID, 1, newJTI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !revoked {
		t.Fatalf("expected version 1 token to be revoked after bump to 2")
	}

	// Token with version 2 is valid!
	revoked, err = checker.IsTokenRevoked(ctx, userID, 2, newJTI)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if revoked {
		t.Fatalf("expected version 2 token to be valid")
	}
}
