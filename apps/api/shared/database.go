package shared

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func InitDatabase(ctx context.Context, connectionString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	if config.MinConns == 0 {
		config.MinConns = 5
	}

	if config.MaxConns == 0 {
		config.MaxConns = 25
	}

	config.MaxConnLifetime = 1 * time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = 1 * time.Minute
	config.MaxConnLifetimeJitter = 10 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err = pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
