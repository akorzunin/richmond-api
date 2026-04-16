package tests

import (
	"context"
	"errors"
	"richmond-api/internal/db"
)

// MockQuerier implements cat.Querier interface for testing
type MockQuerier struct {
	sessions map[string]db.Session
}

// newMockQuerier creates a new mockQuerier
func NewMockQuerier() *MockQuerier {
	return &MockQuerier{
		sessions: make(map[string]db.Session),
	}
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
