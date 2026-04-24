package db

import (
	"context"
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
