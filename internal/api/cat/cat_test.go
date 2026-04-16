package cat

import (
	"io"
	"net/http"
	"testing"

	"richmond-api/tests"
)

// // setupTestRouter creates a test router with the cat handler
// func setupTestRouter(mock *tests.MockQuerier) *gin.Engine {
// 	gin.SetMode(gin.TestMode)
// 	router := gin.New()

// 	handler := NewCatHandler(mock)

// 	router.POST("/api/v1/cat/new", handler.CreateCat)

// 	return router
// }

// func TestCreateCat_Success(t *testing.T) {
// 	router := SetupTestApi()
// 	req, err := createTestRequest("POST", "/api/v1/cat/new", tests.TestCat, "cat.jpg")
// 	if err != nil {
// 		t.Fatalf("failed to create request: %v", err)
// 	}
// 	req.Header.Set("Authorization", "Bearer test-token")

// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusCreated {
// 		t.Errorf("expected status 201, got %d: %s", w.Code, w.Body.String())
// 	}
// }

func TestCreateCat_Successv2(t *testing.T) {
	res, err := tests.TestReq("POST", "/api/v1/cat/new", tests.TestCat, "cat.jpg")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		bodyText, _ := io.ReadAll(res.Body)
		t.Errorf("expected status 201, got %d: %s", res.StatusCode, bodyText)
	}
}

func TestCreateCat_MissingTitlePhoto(t *testing.T) {
	res, err := tests.TestReq("POST", "/api/v1/cat/new", tests.TestCat, "")
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", res.StatusCode)
	}
}

// func TestCreateCat_MissingData(t *testing.T) {
// 	router := SetupTestApi()
// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)
// 	// Add only a file without the "data" field
// 	part, _ := writer.CreateFormFile("file", "cat.jpg")
// 	// Write valid JPEG magic bytes for content type detection
// 	jpegMagicBytes := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00}
// 	part.Write(jpegMagicBytes)
// 	writer.Close()

// 	req, _ := http.NewRequest("POST", "/api/v1/cat/new", body)
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("Authorization", "Bearer test-token")

// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusBadRequest {
// 		t.Errorf("expected status 400, got %d", w.Code)
// 	}
// }

// func TestCreateCat_InvalidJSON(t *testing.T) {
// 	router := SetupTestApi()
// 	invalidJSON := `{"name": "Whiskers", invalid}`
// 	req, err := createTestRequest("POST", "/api/v1/cat/new", invalidJSON, "cat.jpg")
// 	if err != nil {
// 		t.Fatalf("failed to create request: %v", err)
// 	}
// 	req.Header.Set("Authorization", "Bearer test-token")

// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusBadRequest {
// 		t.Errorf("expected status 400, got %d", w.Code)
// 	}
// }

// func TestCreateCat_InvalidFileType(t *testing.T) {
// 	router := SetupTestApi()
// 	body := &bytes.Buffer{}
// 	writer := multipart.NewWriter(body)
// 	writer.WriteField("data", tests.TestCat)
// 	part, _ := writer.CreateFormFile("file", "document.pdf")
// 	// PDF magic bytes: %PDF-1.4
// 	pdfMagicBytes := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
// 	part.Write(pdfMagicBytes)
// 	writer.Close()

// 	req, _ := http.NewRequest("POST", "/api/v1/cat/new", body)
// 	req.Header.Set("Content-Type", writer.FormDataContentType())
// 	req.Header.Set("Authorization", "Bearer test-token")

// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusBadRequest {
// 		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
// 	}
// }

// func TestCreateCat_Unauthorized(t *testing.T) {
// 	mock := tests.NewMockQuerier()
// 	router := setupTestRouterWithAuth(mock)
// 	req, err := createTestRequest("POST", "/api/v1/cat/new", tests.TestCat, "cat.jpg")
// 	if err != nil {
// 		t.Fatalf("failed to create request: %v", err)
// 	}
// 	// No Authorization header

// 	w := httptest.NewRecorder()
// 	router.ServeHTTP(w, req)

// 	if w.Code != http.StatusUnauthorized {
// 		t.Errorf("expected status 401, got %d", w.Code)
// 	}
// }
