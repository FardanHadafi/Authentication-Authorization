package shared

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type TokenRevocationChecker interface {
	IsTokenRevoked(ctx context.Context, userID uuid.UUID, tokenVersion int, jti string) (bool, error)
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, newVersion int) error
	BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error
}

type RedisRevocationChecker struct {
	redis *redis.Client
}

func NewRevocationChecker(redisClient *redis.Client) TokenRevocationChecker {
	return &RedisRevocationChecker{
		redis: redisClient,
	}
}

func (r *RedisRevocationChecker) IsTokenRevoked(ctx context.Context, userID uuid.UUID, tokenVersion int, jti string) (bool, error) {
	if r.redis == nil {
		return false, nil // Fail-open jika Redis tidak diinisialisasi
	}

	// 1. Cek apakah JTI (JWT ID) spesifik dari token ini ada di blacklist
	if jti != "" {
		blacklisted, err := r.redis.Exists(ctx, fmt.Sprintf("blacklist:jti:%s", jti)).Result()
		if err == nil && blacklisted > 0 {
			return true, nil
		}
	}

	// 2. Cek apakah token_version token lebih lama dari versi aktif di Redis
	versionKey := fmt.Sprintf("user:%s:token_version", userID.String())
	cachedVersionStr, err := r.redis.Get(ctx, versionKey).Result()
	if err == nil {
		if cachedVersion, err := strconv.Atoi(cachedVersionStr); err == nil {
			if tokenVersion < cachedVersion {
				return true, nil // Token stale/revoked!
			}
		}
	}

	return false, nil
}

func (r *RedisRevocationChecker) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID, newVersion int) error {
	if r.redis == nil {
		return nil
	}
	versionKey := fmt.Sprintf("user:%s:token_version", userID.String())
	return r.redis.Set(ctx, versionKey, strconv.Itoa(newVersion), 30*24*time.Hour).Err()
}

func (r *RedisRevocationChecker) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if r.redis == nil || jti == "" {
		return nil
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	blacklistKey := fmt.Sprintf("blacklist:jti:%s", jti)
	return r.redis.Set(ctx, blacklistKey, "revoked", ttl).Err()
}
