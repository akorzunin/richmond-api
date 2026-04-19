// Package tx defines transaction interfaces for database operations.
// These interfaces are used by the application layer to avoid depending on pgx directly.
package tx

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TxRunner defines the minimal transaction interface used by the application.
// Only Exec, Commit, and Rollback are needed - avoiding the full pgx.Tx interface.
type TxRunner interface {
	Exec(ctx context.Context, sql string, args ...any) (any, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// Pool defines the interface for beginning transactions.
type Pool interface {
	Begin(ctx context.Context) (TxRunner, error)
}

// TxRunnerAdapter wraps pgx.Tx to implement tx.TxRunner
type TxRunnerAdapter struct {
	Tx pgx.Tx
}

func (t *TxRunnerAdapter) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (any, error) {
	return t.Tx.Exec(ctx, sql, args...)
}

func (t *TxRunnerAdapter) Commit(ctx context.Context) error {
	return t.Tx.Commit(ctx)
}

func (t *TxRunnerAdapter) Rollback(ctx context.Context) error {
	return t.Tx.Rollback(ctx)
}

// PoolAdapter wraps *pgxpool.Pool to implement tx.Pool
type PoolAdapter struct {
	Pool *pgxpool.Pool
}

func (p *PoolAdapter) Begin(ctx context.Context) (TxRunner, error) {
	tx, err := p.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return &TxRunnerAdapter{Tx: tx}, nil
}
