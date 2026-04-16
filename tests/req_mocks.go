package tests

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
)

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

func TestReq(method, url, data, filename string) (*http.Response, error) {
	mock := NewMockQuerier()
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// handler := new_handler(mock)
	// NewCatHandler(mock)

	// Create a simple auth middleware for testing
	// authMiddleware := func(c *gin.Context) {
	// 	authHeader := c.GetHeader("Authorization")
	// 	if authHeader == "" {
	// 		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "authorization header required"})
	// 		c.Abort()
	// 		return
	// 	}

	// 	// Extract Bearer token
	// 	var token string
	// 	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
	// 		token = authHeader[7:]
	// 	} else {
	// 		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "invalid authorization header format"})
	// 		c.Abort()
	// 		return
	// 	}

	// 	// Validate token
	// 	session, err := mock.GetSessionByToken(c.Request.Context(), token)
	// 	if err != nil {
	// 		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "invalid or expired token"})
	// 		c.Abort()
	// 		return
	// 	}
	// 	c.Set("user_id", session.UserID)
	// 	c.Next()
	// }

	// router.POST("/api/v1/cat/new", authMiddleware, handler.CreateCat)

	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	req, err := createTestRequest(method, url, data, filename)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}
