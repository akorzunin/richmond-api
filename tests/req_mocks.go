package tests

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
)

// CreateTestRequest creates a multipart form request with optional file
func CreateTestRequest(
	method, url string,
	data string,
	filename string,
) (*http.Request, error) {
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
		jpegMagicBytes := []byte{
			0xFF,
			0xD8,
			0xFF,
			0xE0,
			0x00,
			0x10,
			0x4A,
			0x46,
			0x49,
			0x46,
			0x00,
		}
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

// SetupTestApi creates a gin router with auth middleware and registers the provided handlers
func SetupTestApi(
	mock *MockQuerier,
	registerHandlers func(router *gin.Engine),
) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add session for test-token
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})

	// Auth middleware
	router.Use(func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "authorization header required"},
			)
			c.Abort()
			return
		}

		// Extract Bearer token
		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		session, err := mock.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"},
			)
			c.Abort()
			return
		}
		c.Set("user_id", session.UserID)
		c.Next()
	})

	// Register provided handlers
	registerHandlers(router)

	return router
}

func TestReq(
	method, path, data, filename string,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	mock := NewMockQuerier()
	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(method, path, handlerFunc)
	})
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})

	req, err := CreateTestRequest(method, path, data, filename)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// TestReqNoAuth is a convenience function that executes a request without authorization
func TestReqNoAuth(
	method, path, data, filename string,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	mock := NewMockQuerier()
	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(method, path, handlerFunc)
	})
	req, err := CreateTestRequest(method, path, data, filename)
	if err != nil {
		return nil, err
	}
	// NO Authorization header for unauthorized tests
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// TestReqWithFileContent is a convenience function that executes a request with custom file content
func TestReqWithFileContent(
	method, path, data, filename string,
	fileContent []byte,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	mock := NewMockQuerier()
	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(method, path, handlerFunc)
	})
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON data field
	if data != "" {
		writer.WriteField("data", data)
	}
	if filename != "" {
		part, err := writer.CreateFormFile("file", filename)
		if err != nil {
			return nil, err
		}
		if len(fileContent) > 0 {
			part.Write(fileContent)
		}
	}
	writer.Close()
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}
