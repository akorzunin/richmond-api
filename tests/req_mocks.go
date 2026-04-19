package tests

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	"richmond-api/internal/db"
)

// TestOptions provides flexible configuration for test requests
type TestOptions struct {
	Method      string
	Path        string
	Data        string
	Filename    string
	FileContent []byte
	Auth        bool
	Headers     map[string]string
	QueryParams map[string]string
}

// CreateTestRequest creates a generic multipart form request with flexible options
func CreateTestRequest(options TestOptions) (*http.Request, error) {
	// Guard clause: validate required options
	if options.Method == "" || options.Path == "" {
		return nil, errors.New("method and path are required for test requests")
	}

	// Create request body
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add JSON data field
	if options.Data != "" {
		err := writer.WriteField("data", options.Data)
		if err != nil {
			return nil, err
		}
	}

	// Add file if provided
	if options.Filename != "" {
		part, err := writer.CreateFormFile("file", options.Filename)
		if err != nil {
			return nil, err
		}
		if len(options.FileContent) > 0 {
			_, err = part.Write(options.FileContent)
			if err != nil {
				return nil, err
			}
		}
	}

	writer.Close()

	// Create HTTP request
	req, err := http.NewRequest(options.Method, options.Path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Add additional headers
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	if len(options.QueryParams) > 0 {
		query := req.URL.Query()
		for key, value := range options.QueryParams {
			query.Add(key, value)
		}
		req.URL.RawQuery = query.Encode()
	}

	return req, nil
}

// AuthMiddleware provides reusable authentication middleware
func AuthMiddleware(mock *MockQuerier) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Guard clause: check for Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
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
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user context for downstream handlers
		c.Set("user_id", session.UserID)
		c.Next()
	}
}

// SetupTestApi creates a gin router with auth middleware and registers the provided handlers
func SetupTestApi(mock *MockQuerier, registerHandlers func(router *gin.Engine)) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Add auth middleware
	router.Use(AuthMiddleware(mock))

	// Register provided handlers
	registerHandlers(router)

	return router
}

// TestReq executes a test request with flexible options
func TestReq(options TestOptions, handlerFunc gin.HandlerFunc) (*http.Response, error) {
	mock := NewMockQuerier()
	options.Auth = true // Default to authenticated requests

	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(options.Method, options.Path, handlerFunc)
	})

	// Add test session
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})

	req, err := CreateTestRequest(options)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// TestReqNoAuth executes a test request without authorization
func TestReqNoAuth(options TestOptions, handlerFunc gin.HandlerFunc) (*http.Response, error) {
	mock := NewMockQuerier()
	options.Auth = false

	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(options.Method, options.Path, handlerFunc)
	})

	req, err := CreateTestRequest(options)
	if err != nil {
		return nil, err
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// TestReqWithCustomOptions executes a test request with custom options
func TestReqWithCustomOptions(options TestOptions, handlerFunc gin.HandlerFunc) (*http.Response, error) {
	mock := NewMockQuerier()
	if options.Auth {
		mock.AddSession("test-token", db.Session{
			SessionID: 1,
			UserID:    42,
			Token:     "test-token",
		})
	}

	router := SetupTestApi(mock, func(r *gin.Engine) {
		r.Handle(options.Method, options.Path, handlerFunc)
	})

	req, err := CreateTestRequest(options)
	if err != nil {
		return nil, err
	}
	if options.Auth {
		req.Header.Set("Authorization", "Bearer test-token")
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}
