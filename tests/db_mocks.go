package tests

import (
	"context"
	"errors"

	"richmond-api/internal/api/tx"
	"richmond-api/internal/db"

	"github.com/jackc/pgx/v5/pgtype"
)

// MockQuerier implements the Querier interface for testing
type MockQuerier struct {
	sessions  map[string]db.Session
	cats      []db.Cat
	files     []db.File
	txQuerier interface {
		CreateFile(ctx context.Context, params db.CreateFileParams) (db.File, error)
		CreateCat(ctx context.Context, params db.CreateCatParams) (db.Cat, error)
	}
}

// ListCats implements cat.Querier.
func (m *MockQuerier) ListCats(
	ctx context.Context,
	arg db.ListCatsParams,
) ([]db.Cat, error) {
	if len(m.cats) == 0 {
		return []db.Cat{}, nil
	}
	limit := arg.Limit
	if limit <= 0 || int(limit) > len(m.cats) {
		limit = int32(len(m.cats))
	}
	offset := arg.Offset
	if offset < 0 || int(offset) > len(m.cats) {
		offset = 0
	}
	end := int(offset) + int(limit)
	if end > len(m.cats) {
		end = len(m.cats)
	}
	return m.cats[offset:end], nil
}

// UpdateCat implements cat.Querier.
func (m *MockQuerier) UpdateCat(
	ctx context.Context,
	arg db.UpdateCatParams,
) (db.Cat, error) {
	for i, cat := range m.cats {
		if cat.CatID == arg.CatID {
			// Return updated cat - in mock we just update the fields
			m.cats[i].Name = arg.Name
			m.cats[i].Breed = arg.Breed
			m.cats[i].Weight = arg.Weight
			m.cats[i].Habits = arg.Habits
			return m.cats[i], nil
		}
	}
	return db.Cat{}, errors.New("cat not found")
}

// DeleteCat implements cat.Querier.
func (m *MockQuerier) DeleteCat(
	ctx context.Context,
	arg db.DeleteCatParams,
) error {
	for i, cat := range m.cats {
		if cat.CatID == arg.CatID {
			// Remove cat from slice
			m.cats = append(m.cats[:i], m.cats[i+1:]...)
			return nil
		}
	}
	return errors.New("cat not found")
}

// GetFilesByCatID implements cat.Querier.
func (m *MockQuerier) GetFilesByCatID(
	ctx context.Context,
	catID pgtype.Int4,
) ([]db.File, error) {
	if !catID.Valid {
		return []db.File{}, nil
	}
	var files []db.File
	for _, f := range m.files {
		if f.CatID.Valid && f.CatID.Int32 == catID.Int32 {
			files = append(files, f)
		}
	}
	return files, nil
}

// NewMockQuerier creates a new mockQuerier
func NewMockQuerier() *MockQuerier {
	return &MockQuerier{
		sessions:  make(map[string]db.Session),
		cats:      make([]db.Cat, 0),
		files:     make([]db.File, 0),
		txQuerier: &MockTxQuerier{},
	}
}

// WithTx returns a mock transaction querier (accepts tx.TxRunner to satisfy tx.Querier)
func (m *MockQuerier) WithTx(tx tx.TxRunner) db.TxQuerier {
	return m.txQuerier
}

// AddCat adds a cat for testing
func (m *MockQuerier) AddCat(cat db.Cat) {
	m.cats = append(m.cats, cat)
}

// AddFile adds a file for testing
func (m *MockQuerier) AddFile(file db.File) {
	m.files = append(m.files, file)
}

// AddSession adds a session for testing auth
func (m *MockQuerier) AddSession(token string, session db.Session) {
	m.sessions[token] = session
}

// DeleteSession implements Querier for session management in tests
func (m *MockQuerier) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

// DeleteUserSessions implements Querier
func (m *MockQuerier) DeleteUserSessions(
	ctx context.Context,
	userID int32,
) error {
	for token, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}

// GetSessionByToken implements Querier
func (m *MockQuerier) GetSessionByToken(
	ctx context.Context,
	token string,
) (db.Session, error) {
	session, exists := m.sessions[token]
	if !exists {
		return db.Session{}, errors.New("session not found")
	}
	return session, nil
}

// CreateFile implements Querier
func (m *MockQuerier) CreateFile(
	ctx context.Context,
	params db.CreateFileParams,
) (db.File, error) {
	if len(m.files) > 0 {
		return m.files[len(m.files)-1], nil
	}
	return db.File{ID: 1}, nil
}

// MockTxQuerier implements TxQuerierIF interface for testing
type MockTxQuerier struct {
	files []db.File
	cats  []db.Cat
}

// CreateFile implements TxQuerierIF
func (m *MockTxQuerier) CreateFile(
	ctx context.Context,
	params db.CreateFileParams,
) (db.File, error) {
	if len(m.files) > 0 {
		return m.files[len(m.files)-1], nil
	}
	return db.File{ID: 1}, nil
}

// CreateCat implements TxQuerierIF
func (m *MockTxQuerier) CreateCat(
	ctx context.Context,
	params db.CreateCatParams,
) (db.Cat, error) {
	if len(m.cats) > 0 {
		return m.cats[len(m.cats)-1], nil
	}
	return db.Cat{CatID: 1}, nil
}

// MockPool implements the tx.Pool interface for testing
type MockPool struct {
	tx *MockTx
}

type MockTx struct{}

// Begin implements Pool interface - returns a tx.TxRunner
func (m *MockPool) Begin(ctx context.Context) (tx.TxRunner, error) {
	return &MockTxRunner{}, nil
}

// MockTxRunner implements the tx.TxRunner interface for testing
type MockTxRunner struct{}

// Exec implements tx.TxRunner - returns nil for simplicity
func (m *MockTxRunner) Exec(
	ctx context.Context,
	sql string,
	args ...any,
) (any, error) {
	return nil, nil
}

// Commit implements tx.TxRunner - no-op for tests
func (m *MockTxRunner) Commit(ctx context.Context) error {
	return nil
}

// Rollback implements tx.TxRunner - no-op for tests
func (m *MockTxRunner) Rollback(ctx context.Context) error {
	return nil
}
