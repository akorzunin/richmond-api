package tests

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

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

// NewMockPool creates a new MockPool for testing
func NewMockPool() *MockPool {
	return &MockPool{
		tx: &MockTx{},
	}
}

// Begin implements Pool interface - returns a tx.TxRunner
func (m *MockPool) Begin(ctx context.Context) (tx.TxRunner, error) {
	return &MockTxRunner{}, nil
}

// MockTxRunner implements the tx.TxRunner interface for testing
type MockTxRunner struct{}

// Exec implements tx.TxRunner - returns nil for simplicity
func (m *MockTxRunner) Exec(ctx context.Context, sql string, args ...any) (any, error) {
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

// MockTx implements the full pgx.Tx interface for tests that need it.
// Only the methods actually used by the application are fully mocked:
// - Exec: Used for UPDATE queries
// - Commit: Used to finalize transaction
// - Rollback: Used on error paths
// Unused pgx.Tx methods return zero values to satisfy the interface.
type MockTx struct{}

// Begin implements pgx.Tx - satisfies interface, returns self
func (m *MockTx) Begin(ctx context.Context) (pgx.Tx, error) {
	return m, nil
}

// Commit implements pgx.Tx - commits the transaction (no-op for tests)
func (m *MockTx) Commit(ctx context.Context) error {
	return nil
}

// Rollback implements pgx.Tx - rolls back the transaction (no-op for tests)
func (m *MockTx) Rollback(ctx context.Context) error {
	return nil
}

// Exec implements pgx.Tx - executes a query (no-op for tests)
func (m *MockTx) Exec(
	ctx context.Context,
	sql string,
	arguments ...any,
) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

// Prepare implements pgx.Tx - satisfies interface, returns empty statement
func (m *MockTx) Prepare(
	ctx context.Context,
	name string,
	sql string,
) (*pgconn.StatementDescription, error) {
	return &pgconn.StatementDescription{}, nil
}

// Query implements pgx.Tx - satisfies interface, returns nil rows
func (m *MockTx) Query(
	ctx context.Context,
	sql string,
	args ...any,
) (pgx.Rows, error) {
	return nil, nil
}

// QueryRow implements pgx.Tx - satisfies interface, returns nil row
func (m *MockTx) QueryRow(
	ctx context.Context,
	sql string,
	args ...any,
) pgx.Row {
	return nil
}

// CopyFrom implements pgx.Tx - satisfies interface, returns 0
func (m *MockTx) CopyFrom(
	ctx context.Context,
	tableName pgx.Identifier,
	columnNames []string,
	rowSrc pgx.CopyFromSource,
) (int64, error) {
	return 0, nil
}

// SendBatch implements pgx.Tx - satisfies interface, returns nil
func (m *MockTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults {
	return nil
}

// LargeObjects implements pgx.Tx - satisfies interface, returns empty struct
func (m *MockTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

// Reset implements pgx.Tx - satisfies interface, returns nil
func (m *MockTx) Reset(ctx context.Context) error {
	return nil
}

// Conn implements pgx.Tx - satisfies interface, returns nil
func (m *MockTx) Conn() *pgx.Conn {
	return nil
}
