package cat

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"richmond-api/tests"

	"github.com/gin-gonic/gin"
)

// createTestRequest creates a multipart form request with optional file
func createTestRequest(
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

	writer.Close()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return req, nil
}

// testHandler executes a handler with auth middleware
func testHandler(handlerFunc gin.HandlerFunc) *http.Response {
	gin.SetMode(gin.TestMode)
	router := gin.New()

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

		// For testing, we accept "test-token" as valid
		if token != "test-token" {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"},
			)
			c.Abort()
			return
		}

		c.Set("user_id", int32(42))
		c.Next()
	})

	router.Handle("POST", "/api/v1/cat/new", handlerFunc)

	req, _ := createTestRequest(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
		"cat.jpg",
	)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result()
}

// testReq executes a request with the given handler and auth
func testReq(
	method, path, data, filename string,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	return testReqWithAuth(
		method,
		path,
		data,
		filename,
		handlerFunc,
		"test-token",
	)
}

// testReqNoAuth executes a request without authorization
func testReqNoAuth(
	method, path, data, filename string,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Handle(method, path, handlerFunc)

	req, _ := createTestRequest(method, path, data, filename)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// testReqWithAuth executes a request with specific auth token
func testReqWithAuth(
	method, path, data, filename string,
	handlerFunc gin.HandlerFunc,
	token string,
) (*http.Response, error) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

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

		var t string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			t = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		if t != token {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"},
			)
			c.Abort()
			return
		}

		c.Set("user_id", int32(42))
		c.Next()
	})

	router.Handle(method, path, handlerFunc)

	req, _ := createTestRequest(method, path, data, filename)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}

// testReqWithFileContent executes a request with custom file content
func testReqWithFileContent(
	method, path, data, filename string,
	fileContent []byte,
	handlerFunc gin.HandlerFunc,
) (*http.Response, error) {
	return testReqWithAuthAndFile(
		method,
		path,
		data,
		filename,
		fileContent,
		handlerFunc,
		"test-token",
	)
}

// testReqWithAuthAndFile executes a request with auth token and custom file content
func testReqWithAuthAndFile(
	method, path, data, filename string,
	fileContent []byte,
	handlerFunc gin.HandlerFunc,
	token string,
) (*http.Response, error) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

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

		var t string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			t = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			c.Abort()
			return
		}

		if t != token {
			c.JSON(
				http.StatusUnauthorized,
				gin.H{"error": "invalid or expired token"},
			)
			c.Abort()
			return
		}

		c.Set("user_id", int32(42))
		c.Next()
	})

	router.Handle(method, path, handlerFunc)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

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
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w.Result(), nil
}
