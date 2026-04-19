package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// TxQuerier defines the interface for database operations within a transaction
type TxQuerier interface {
	CreateFile(ctx context.Context, params CreateFileParams) (File, error)
	CreateCat(ctx context.Context, params CreateCatParams) (Cat, error)
}

// QuerierAdapter adapts *Queries to implement cat.Querier interface
type QuerierAdapter struct {
	*Queries
}

func (q *QuerierAdapter) WithTx(tx any) TxQuerier {
	// Accept any type to avoid import cycles - we'll use type assertion
	return q.Queries.WithTx(tx.(pgx.Tx))
}
