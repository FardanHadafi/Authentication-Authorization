package shared

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"authentication-authorization/shared/db"

	"github.com/redis/go-redis/v9"
)

type PermissionChecker interface {
	HasPermission(ctx context.Context, role, permission string) (bool, error)
	GetRolePermissions(ctx context.Context, role string) ([]string, error)
}

type permissionChecker struct {
	querier db.Querier
	redis   *redis.Client
}

func NewPermissionChecker(querier db.Querier, redisClient *redis.Client) PermissionChecker {
	return &permissionChecker{
		querier: querier,
		redis:   redisClient,
	}
}

func (p *permissionChecker) GetRolePermissions(ctx context.Context, role string) ([]string, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	if role == "" {
		return []string{}, nil
	}

	cacheKey := fmt.Sprintf("role:%s:permissions", role)

	// 1. Coba ambil dari Redis cache
	if p.redis != nil {
		cached, err := p.redis.SMembers(ctx, cacheKey).Result()
		if err == nil && len(cached) > 0 {
			slices.Sort(cached)
			return cached, nil
		}
	}

	// 2. Jika cache miss, query ke PostgreSQL
	perms, err := p.querier.GetRolePermissions(ctx, role)
	if err != nil {
		return nil, fmt.Errorf("failed to get role permissions: %w", err)
	}

	// 3. Simpan ke Redis cache dengan TTL 1 jam
	if p.redis != nil && len(perms) > 0 {
		members := make([]any, len(perms))
		for i, v := range perms {
			members[i] = v
		}
		_ = p.redis.SAdd(ctx, cacheKey, members...).Err()
		_ = p.redis.Expire(ctx, cacheKey, time.Hour).Err()
	}

	return perms, nil
}

func (p *permissionChecker) HasPermission(ctx context.Context, role, permission string) (bool, error) {
	role = strings.ToLower(strings.TrimSpace(role))
	permission = strings.ToLower(strings.TrimSpace(permission))
	if role == "" || permission == "" {
		return false, nil
	}

	cacheKey := fmt.Sprintf("role:%s:permissions", role)

	// 1. Coba cek via Redis Set membership jika key ada
	if p.redis != nil {
		exists, err := p.redis.Exists(ctx, cacheKey).Result()
		if err == nil && exists > 0 {
			isMember, errMember := p.redis.SIsMember(ctx, cacheKey, permission).Result()
			if errMember == nil {
				return isMember, nil
			}
		}
	}

	// 2. Jika cache miss, ambil seluruh permission role (sekaligus memanaskan cache)
	perms, err := p.GetRolePermissions(ctx, role)
	if err != nil {
		// Fallback langsung ke query jika GetRolePermissions gagal
		return p.querier.CheckRoleHasPermission(ctx, db.CheckRoleHasPermissionParams{
			RoleID:       role,
			PermissionID: permission,
		})
	}

	return slices.Contains(perms, permission), nil
}
