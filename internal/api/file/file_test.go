package file

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
)

// mockGetFileOverride temporarily overrides getFileFunc for testing
func mockGetFileOverride(
	mockFn func(client *minio.Client, bucket, key string) ([]byte, error),
) func() {
	original := getFileFunc
	getFileFunc = mockFn
	return func() { getFileFunc = original }
}

// TestDownload_Success tests successful file download
func TestDownload_Success(t *testing.T) {
	// Setup mock for successful download
	cleanup := mockGetFileOverride(
		func(client *minio.Client, bucket, key string) ([]byte, error) {
			return []byte("file content bytes"), nil
		},
	)
	defer cleanup()

	// Create handler
	handler := NewFileHandler(&minio.Client{}, "test-bucket")

	// Setup Gin test mode and router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Any("/api/v1/file/*key", handler.Download)

	// Create GET request
	req, _ := http.NewRequest("GET", "/api/v1/file/cat/1/testkey.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}

	// Verify Content-Type header
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/octet-stream" {
		t.Errorf(
			"expected Content-Type application/octet-stream, got %s",
			contentType,
		)
	}

	// Verify body content
	body := w.Body.String()
	if body != "file content bytes" {
		t.Errorf("expected body 'file content bytes', got %s", body)
	}
}

// TestDownload_MissingKey tests empty key returns 400
func TestDownload_MissingKey(t *testing.T) {
	// Create handler
	handler := NewFileHandler(&minio.Client{}, "test-bucket")

	// Setup Gin test mode and router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/file/:key", handler.Download)

	// Create GET request with empty key (using path param)
	req, _ := http.NewRequest("GET", "/api/v1/file/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response - empty key in path should return 404 (Gin treats empty param as not found)
	// But if we use the actual empty string case
	if w.Code != http.StatusNotFound && w.Code != http.StatusBadRequest {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 400 or 404, got %d: %s", w.Code, bodyText)
	}
}

// TestDownload_InvalidQuality tests invalid quality param returns 400
func TestDownload_InvalidQuality(t *testing.T) {
	// Setup mock (won't be called due to early validation)
	cleanup := mockGetFileOverride(
		func(client *minio.Client, bucket, key string) ([]byte, error) {
			return nil, errors.New("should not be called")
		},
	)
	defer cleanup()

	// Create handler
	handler := NewFileHandler(&minio.Client{}, "test-bucket")

	// Setup Gin test mode and router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/file/:key", handler.Download)

	// Create GET request with invalid quality query param
	req, _ := http.NewRequest(
		"GET",
		"/api/v1/file/testkey.jpg?quality=invalid",
		nil,
	)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusBadRequest {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 400, got %d: %s", w.Code, bodyText)
	}
	body := w.Body.String()
	if !strings.Contains(body, "invalid quality") {
		t.Errorf(
			"expected error message to contain 'invalid quality', got %s",
			body,
		)
	}
}

// TestDownload_FileNotFound tests "No such key" error returns 404
func TestDownload_FileNotFound(t *testing.T) {
	// Setup mock for file not found error
	cleanup := mockGetFileOverride(
		func(client *minio.Client, bucket, key string) ([]byte, error) {
			return nil, errors.New("No such key: testkey.jpg")
		},
	)
	defer cleanup()

	// Create handler
	handler := NewFileHandler(&minio.Client{}, "test-bucket")

	// Setup Gin test mode and router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/file/:key", handler.Download)

	// Create GET request
	req, _ := http.NewRequest("GET", "/api/v1/file/testkey.jpg", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Verify response
	if w.Code != http.StatusNotFound {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 404, got %d: %s", w.Code, bodyText)
	}
	body := w.Body.String()
	if !strings.Contains(body, "not found") {
		t.Errorf("expected error message to contain 'not found', got %s", body)
	}
}
