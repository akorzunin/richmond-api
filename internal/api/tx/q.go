package tx

import (
	"context"
	"richmond-api/internal/db"
)

// QuerierAdapter wraps *db.Queries to implement cat.Querier
type QuerierAdapter struct {
	*db.Queries
}

func (q *QuerierAdapter) WithTx(tx TxRunner) db.TxQuerier {
	// Extract the underlying pgx.Tx from TxRunnerAdapter if possible
	if adapted, ok := tx.(*TxRunnerAdapter); ok {
		return q.Queries.WithTx(adapted.Tx)
	}
	// Fallback: pass nil (will fail at runtime if reached incorrectly)
	panic("WithTx requires *TxRunnerAdapter")
}

func (q *QuerierAdapter) CreateCat(
	ctx context.Context,
	params db.CreateCatParams,
) (db.Cat, error) {
	return q.Queries.CreateCat(ctx, params)
}

func (q *QuerierAdapter) GetCatByID(
	ctx context.Context,
	catID int32,
) (db.Cat, error) {
	return q.Queries.GetCatByID(ctx, catID)
}

func (q *QuerierAdapter) CreateFile(
	ctx context.Context,
	params db.CreateFileParams,
) (db.File, error) {
	return q.Queries.CreateFile(ctx, params)
}

func (q *QuerierAdapter) GetSessionByToken(
	ctx context.Context,
	token string,
) (db.Session, error) {
	return q.Queries.GetSessionByToken(ctx, token)
}

func (q *QuerierAdapter) DeleteSession(
	ctx context.Context,
	token string,
) error {
	return q.Queries.DeleteSession(ctx, token)
}

func (q *QuerierAdapter) DeleteUserSessions(
	ctx context.Context,
	userID int32,
) error {
	return q.Queries.DeleteUserSessions(ctx, userID)
}
