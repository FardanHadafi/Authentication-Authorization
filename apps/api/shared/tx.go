package shared

import (
	"context"
	"fmt"

	"authentication-authorization/shared/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Transactor defines an interface for executing atomic database transactions.
type Transactor interface {
	RunInTx(ctx context.Context, fn func(q db.Querier) error) error
}

type pgxTransactor struct {
	pool *pgxpool.Pool
}

// NewTransactor creates a new Transactor instance backed by a pgxpool.Pool.
func NewTransactor(pool *pgxpool.Pool) Transactor {
	return &pgxTransactor{pool: pool}
}

// RunInTx executes the provided callback within an atomic database transaction.
// If fn returns an error or panics, the transaction is safely rolled back.
// Otherwise, the transaction is committed.
func (t *pgxTransactor) RunInTx(ctx context.Context, fn func(q db.Querier) error) error {
	if t.pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	queries := db.New(tx)
	if err := fn(queries); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
