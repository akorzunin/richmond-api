package tests

import (
	"context"
	"errors"

	"richmond-api/internal/api/tx"
	"richmond-api/internal/db"
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
