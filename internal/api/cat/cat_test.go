package cat

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
)

const TestCat string = `{"name": "Whiskers", "birth_date": "2023-01-15", "breed": "Tabby", "habits": "Sleeping", "weight": 4.5}`

// mockQuerier implements cat.Querier interface for testing
type mockQuerier struct {
	sessions map[string]db.Session
}

// newMockQuerier creates a new mockQuerier
func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		sessions: make(map[string]db.Session),
	}
}

// AddSession adds a session for testing auth
func (m *mockQuerier) AddSession(token string, session db.Session) {
	m.sessions[token] = session
}

// DeleteSession implements db.Querier for session management in tests
func (m *mockQuerier) DeleteSession(ctx context.Context, token string) error {
	delete(m.sessions, token)
	return nil
}

// DeleteUserSessions implements db.Querier
func (m *mockQuerier) DeleteUserSessions(ctx context.Context, userID int32) error {
	for token, session := range m.sessions {
		if session.UserID == userID {
			delete(m.sessions, token)
		}
	}
	return nil
}

// GetSessionByToken implements db.Querier
func (m *mockQuerier) GetSessionByToken(ctx context.Context, token string) (db.Session, error) {
	session, exists := m.sessions[token]
	if !exists {
		return db.Session{}, errors.New("session not found")
	}
	return session, nil
}

// createTestRequest creates a multipart form request with optional file
func createTestRequest(method, url string, data string, filename string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON data field
	if data != "" {
		err := writer.WriteField("data", data)
		if err != nil {
			return nil, err
		}
	}

	// Add file if provided
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}
		// Write valid JPEG magic bytes for content type detection
		// JPEG magic bytes: FF D8 FF followed by segment marker
		jpegMagicBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
		_, err = part.Write(jpegMagicBytes)
		if err != nil {
			return nil, err
		}
	}

	// Add additional files for gallery tests
	writer.Close()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// setupTestRouter creates a test router with the cat handler
func setupTestRouter(mock *mockQuerier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewCatHandler(mock)

	router.POST("/api/v1/cat/new", handler.CreateCat)

	return router
}

// setupTestRouterWithAuth creates a test router with auth middleware
func setupTestRouterWithAuth(mock *mockQuerier) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	handler := NewCatHandler(mock)

	// Create a simple auth middleware for testing
	authMiddleware := func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "authorization header required"})
			c.Abort()
			return
		}

		// Extract Bearer token
		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		session, err := mock.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "invalid or expired token"})
			c.Abort()
			return
		}
		c.Set("user_id", session.UserID)
		c.Next()
	}

	router.POST("/api/v1/cat/new", authMiddleware, handler.CreateCat)

	return router
}

func SetupTestApi() *gin.Engine {
	mock := newMockQuerier()
	router := setupTestRouterWithAuth(mock)
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	return router
}

func TestCreateCat_Success(t *testing.T) {
	router := SetupTestApi()
	req, err := createTestRequest("POST", "/api/v1/cat/new", TestCat, "cat.jpg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCat_MissingTitlePhoto(t *testing.T) {
	router := SetupTestApi()
	req, err := createTestRequest("POST", "/api/v1/cat/new", TestCat, "")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateCat_MissingData(t *testing.T) {
	router := SetupTestApi()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	// Add only a file without the "data" field
	part, _ := writer.CreateFormFile("file", "cat.jpg")
	// Write valid JPEG magic bytes for content type detection
	jpegMagicBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
	part.Write(jpegMagicBytes)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/cat/new", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateCat_InvalidJSON(t *testing.T) {
	router := SetupTestApi()
	invalidJSON := `{"name": "Whiskers", invalid}`
	req, err := createTestRequest("POST", "/api/v1/cat/new", invalidJSON, "cat.jpg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestCreateCat_InvalidFileType(t *testing.T) {
	router := SetupTestApi()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	writer.WriteField("data", TestCat)
	part, _ := writer.CreateFormFile("file", "document.pdf")
	// PDF magic bytes: %PDF-1.4
	pdfMagicBytes := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	part.Write(pdfMagicBytes)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/v1/cat/new", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestCreateCat_Unauthorized(t *testing.T) {
	mock := newMockQuerier()
	router := setupTestRouterWithAuth(mock)
	req, err := createTestRequest("POST", "/api/v1/cat/new", TestCat, "cat.jpg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	// No Authorization header

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}
