package tests

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"richmond-api/internal/db"
)

// MockQuerier implements cat.Querier interface for testing
type MockQuerier struct {
	sessions map[string]db.Session
	cats     []db.Cat
	files    []db.File
}

// newMockQuerier creates a new mockQuerier
func NewMockQuerier() *MockQuerier {
	return &MockQuerier{
		sessions: make(map[string]db.Session),
		cats:     make([]db.Cat, 0),
		files:    make([]db.File, 0),
	}
}

// WithTx is a no-op for testing
func (m *MockQuerier) WithTx(tx pgx.Tx) *db.Queries {
	return &db.Queries{}
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

// DeleteSession implements db.Querier for session management in tests
func (m *MockQuerier) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

// DeleteUserSessions implements db.Querier
func (m *MockQuerier) DeleteUserSessions(ctx context.Context, userID int32) error {
	for token, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}

// GetSessionByToken implements db.Querier
func (m *MockQuerier) GetSessionByToken(ctx context.Context, token string) (db.Session, error) {
	session, exists := m.sessions[token]
	if !exists {
		return db.Session{}, errors.New("session not found")
	}
	return session, nil
}

// CreateFile implements cat.Querier
func (m *MockQuerier) CreateFile(ctx context.Context, params db.CreateFileParams) (db.File, error) {
	if len(m.files) > 0 {
		return m.files[len(m.files)-1], nil
	}
	return db.File{ID: 1}, nil
}
