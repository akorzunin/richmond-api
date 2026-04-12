package user

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// mockQuerier implements db.Querier interface for testing
type mockQuerier struct {
	users     map[string]db.User
	sessions  map[string]db.Session
	sessionID int32
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		users:    make(map[string]db.User),
		sessions: make(map[string]db.Session),
	}
}

func (m *mockQuerier) GetUserByName(
	ctx context.Context,
	userName string,
) (db.User, error) {
	user, exists := m.users[userName]
	if !exists {
		return db.User{}, errors.New("user not found")
	}
	return user, nil
}

func (m *mockQuerier) GetUserByID(
	ctx context.Context,
	userID int32,
) (db.User, error) {
	for _, user := range m.users {
		if user.UserID == userID {
			return user, nil
		}
	}
	return db.User{}, errors.New("user not found")
}

func (m *mockQuerier) CreateUser(
	ctx context.Context,
	params db.CreateUserParams,
) (db.User, error) {
	user := db.User{
		UserID:   1,
		UserName: params.UserName,
		UserPass: params.UserPass,
	}
	m.users[params.UserName] = user
	return user, nil
}

func (m *mockQuerier) CreateSession(
	ctx context.Context,
	params db.CreateSessionParams,
) (db.Session, error) {
	m.sessionID++
	session := db.Session{
		SessionID: m.sessionID,
		UserID:    params.UserID,
		Token:     params.Token,
		ExpiresAt: params.ExpiresAt,
	}
	m.sessions[params.Token] = session
	return session, nil
}

func (m *mockQuerier) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

func (m *mockQuerier) DeleteUserSessions(
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

func (m *mockQuerier) GetSessionByToken(
	ctx context.Context,
	token string,
) (db.Session, error) {
	session, exists := m.sessions[token]
	if !exists {
		return db.Session{}, errors.New("session not found")
	}
	return session, nil
}

// Helper function to setup test router
func setupTestRouter(mock *mockQuerier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewUserHandler(mock)

	router.POST("/api/v1/user/new", handler.Create)
	router.POST("/api/v1/user/login", handler.Login)

	return router
}

func TestCreateUser_Success(t *testing.T) {
	mock := newMockQuerier()
	router := setupTestRouter(mock)

	body := CreateRequest{
		Login:    "newuser",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/api/v1/user/new",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d", w.Code)
	}

	var response UserResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Login != "newuser" {
		t.Errorf("expected login 'newuser', got '%s'", response.Login)
	}
}

func TestCreateUser_UserExists(t *testing.T) {
	mock := newMockQuerier()
	// Pre-add a user
	hashedPassword, _ := bcrypt.GenerateFromPassword(
		[]byte("password123"),
		bcrypt.DefaultCost,
	)
	mock.users["existinguser"] = db.User{
		UserID:   1,
		UserName: "existinguser",
		UserPass: string(hashedPassword),
	}

	router := setupTestRouter(mock)

	body := CreateRequest{
		Login:    "existinguser",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/api/v1/user/new",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusConflict {
		t.Errorf("expected status 409, got %d", w.Code)
	}

	var response e.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "user already exists" {
		t.Errorf(
			"expected error 'user already exists', got '%s'",
			response.Error,
		)
	}
}

func TestCreateUser_InvalidRequest(t *testing.T) {
	mock := newMockQuerier()
	router := setupTestRouter(mock)

	tests := []struct {
		name string
		body CreateRequest
	}{
		{
			name: "missing password",
			body: CreateRequest{Login: "user"},
		},
		{
			name: "missing login",
			body: CreateRequest{Password: "password"},
		},
		{
			name: "empty request",
			body: CreateRequest{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tt.body)
			req, _ := http.NewRequest(
				"POST",
				"/api/v1/user/new",
				bytes.NewBuffer(jsonBody),
			)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Errorf("expected status 400, got %d", w.Code)
			}
		})
	}
}

func TestLogin_Success(t *testing.T) {
	mock := newMockQuerier()
	// Pre-add a user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword(
		[]byte("correctpassword"),
		bcrypt.DefaultCost,
	)
	mock.users["testuser"] = db.User{
		UserID:   1,
		UserName: "testuser",
		UserPass: string(hashedPassword),
	}

	router := setupTestRouter(mock)

	body := LoginRequest{
		Login:    "testuser",
		Password: "correctpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/api/v1/user/login",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response TokenResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Token == "" {
		t.Errorf("expected non-empty token")
	}
}

func TestLogin_InvalidCredentials(t *testing.T) {
	mock := newMockQuerier()
	// Pre-add a user with hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword(
		[]byte("correctpassword"),
		bcrypt.DefaultCost,
	)
	mock.users["testuser"] = db.User{
		UserID:   1,
		UserName: "testuser",
		UserPass: string(hashedPassword),
	}

	router := setupTestRouter(mock)

	body := LoginRequest{
		Login:    "testuser",
		Password: "wrongpassword",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/api/v1/user/login",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var response e.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "invalid credentials" {
		t.Errorf(
			"expected error 'invalid credentials', got '%s'",
			response.Error,
		)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	mock := newMockQuerier()
	router := setupTestRouter(mock)

	body := LoginRequest{
		Login:    "nonexistent",
		Password: "password123",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest(
		"POST",
		"/api/v1/user/login",
		bytes.NewBuffer(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var response e.ErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	if response.Error != "invalid credentials" {
		t.Errorf(
			"expected error 'invalid credentials', got '%s'",
			response.Error,
		)
	}
}
