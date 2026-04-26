package post

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"richmond-api/internal/db"
	"richmond-api/tests"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	testPostData      = `{"cat_id":"1","title":"Whiskers playing","body":"Look at this cute cat!"}`
	testPostDataBlank = `{"cat_id":"1","title":"  ","body":""}`
	testInvalidJSON   = `{"cat_id": "1", invalid}`
)

// testTime is a consistent time for testing
var testTime = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

func setupTestRouter(_ *PostHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

// setupAuthMiddleware creates a test auth middleware
func setupAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			c.Abort()
			return
		}
		// Test token: "Bearer test-token" sets user_id to 42
		c.Set("user_id", int32(42))
		c.Next()
	}
}

// setupTestRouterWithAuth creates a router with auth middleware and post routes
func setupTestRouterWithAuth(handler *PostHandler) *gin.Engine {
	r := setupTestRouter(handler)
	r.Use(setupAuthMiddleware())
	r.POST("/api/v1/post/new", handler.CreatePost)
	r.GET("/api/v1/post/all", handler.ListPosts)
	r.GET("/api/v1/post/:id", handler.GetPost)
	r.PUT("/api/v1/post/:id", handler.UpdatePost)
	r.DELETE("/api/v1/post/:id", handler.DeletePost)
	return r
}

// setupTestRouterNoAuth creates a router without auth middleware
func setupTestRouterNoAuth(handler *PostHandler) *gin.Engine {
	r := setupTestRouter(handler)
	r.POST("/api/v1/post/new", handler.CreatePost)
	r.GET("/api/v1/post/all", handler.ListPosts)
	r.GET("/api/v1/post/:id", handler.GetPost)
	r.PUT("/api/v1/post/:id", handler.UpdatePost)
	r.DELETE("/api/v1/post/:id", handler.DeletePost)
	return r
}

// createMultipartRequest creates a multipart form request
func createMultipartRequest(
	method, url, data, filename string,
) (*http.Request, error) {
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
		// Write valid JPEG magic bytes
		jpegMagicBytes := []byte{
			0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00,
		}
		part.Write(jpegMagicBytes)
	}

	writer.Close()

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// createJSONRequest creates a JSON request
func createJSONRequest(
	method, url string,
	body interface{},
) (*http.Request, error) {
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return req, nil
}

// addAuthHeader adds authorization header to request
func addAuthHeader(req *http.Request) {
	req.Header.Set("Authorization", "Bearer test-token")
}

// TestCreatePost_Success tests creating a post with valid data
func TestCreatePost_Success(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		testPostData,
		"cat.jpg",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 201, got %d: %s", w.Code, bodyText)
	}
}

// TestCreatePost_NoAuth tests creating a post without auth
func TestCreatePost_NoAuth(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		testPostData,
		"",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestCreatePost_CatNotFound tests creating a post with non-existent cat
func TestCreatePost_CatNotFound(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	// cat_id "999" doesn't exist
	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		`{"cat_id":"999","title":"Test","body":"Body"}`,
		"",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 404, got %d: %s", w.Code, bodyText)
	}
}

// TestCreatePost_CatNotOwned tests creating a post for cat not owned by user
func TestCreatePost_CatNotOwned(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()

	// Add a cat owned by different user (user 99, not user 42)
	mockQuerier.AddCat(db.Cat{
		CatID:  99,
		UserID: 99, // Different from auth user 42
		Name:   "OtherUserCat",
	})

	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	// Try to create post for cat 99 (owned by user 99)
	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		`{"cat_id":"99","title":"Test","body":"Body"}`,
		"",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because user 42 doesn't own cat 99
	if w.Code != http.StatusForbidden {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 403, got %d: %s", w.Code, bodyText)
	}
}

// TestCreatePost_InvalidJSON tests creating a post with malformed JSON
func TestCreatePost_InvalidJSON(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		testInvalidJSON,
		"",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestCreatePost_ValidFiles tests creating a post with files
func TestCreatePost_ValidFiles(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		testPostData,
		"cat.jpg",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 201, got %d: %s", w.Code, bodyText)
	}
}

// TestCreatePost_S3UploadFails tests creating a post when S3 upload fails
func TestCreatePost_S3UploadFails(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.MockS3UploaderWithError(errors.New("S3 upload failed"))
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	// Request with file should fail because S3 upload fails
	req, err := createMultipartRequest(
		"POST",
		"/api/v1/post/new",
		testPostData,
		"cat.jpg",
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should fail because S3 upload failed
	if w.Code != http.StatusBadRequest {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 400, got %d: %s", w.Code, bodyText)
	}

	// Verify no post was created
	posts, err := mockQuerier.ListPosts(
		context.Background(),
		db.ListPostsParams{},
	)
	if err != nil {
		t.Fatalf("failed to list posts: %v", err)
	}
	if len(posts) != 0 {
		t.Errorf("expected no posts, got %d", len(posts))
	}
}

// TestListPosts_Success tests listing posts
func TestListPosts_Success(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := http.NewRequest("GET", "/api/v1/post/all", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}
}

// TestListPosts_WithQueryParams tests listing posts with query params
func TestListPosts_WithQueryParams(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	// Note: The current ListPosts implementation doesn't support cat_id filter
	// This test verifies the endpoint works with query params
	req, err := http.NewRequest("GET", "/api/v1/post/all?cat_id=1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}
}

// TestListPosts_Pagination tests listing posts with pagination
func TestListPosts_Pagination(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	// Add multiple posts
	for i := 1; i <= 25; i++ {
		mockQuerier.AddPost(db.Post{
			PostID: int32(i),
			UserID: 42,
			CatID:  1,
			Title:  "Test Post",
			Body:   "Body",
		})
	}
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := http.NewRequest(
		"GET",
		"/api/v1/post/all?limit=10&offset=0",
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}

	// Verify response contains pagination info
	var resp ListPostsResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp.Limit != 10 {
		t.Errorf("expected limit 10, got %d", resp.Limit)
	}
	if resp.Offset != 0 {
		t.Errorf("expected offset 0, got %d", resp.Offset)
	}
}

// TestGetPost_Success tests getting a post by ID
func TestGetPost_Success(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := http.NewRequest("GET", "/api/v1/post/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}
}

// TestGetPost_NotFound tests getting a non-existent post
func TestGetPost_NotFound(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := http.NewRequest("GET", "/api/v1/post/999", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestUpdatePost_Success tests updating a post
func TestUpdatePost_Success(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	updateData := UpdatePostRequest{
		Title: strPtr("Updated Title"),
		Body:  strPtr("Updated Body"),
	}
	req, err := createJSONRequest("PUT", "/api/v1/post/1", updateData)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}
}

// TestUpdatePost_NoAuth tests updating a post without auth
func TestUpdatePost_NoAuth(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	updateData := UpdatePostRequest{
		Title: strPtr("Updated Title"),
	}
	req, err := createJSONRequest("PUT", "/api/v1/post/1", updateData)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestUpdatePost_NotFound tests updating a non-existent post
func TestUpdatePost_NotFound(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	updateData := UpdatePostRequest{
		Title: strPtr("Updated Title"),
	}
	req, err := createJSONRequest("PUT", "/api/v1/post/999", updateData)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestUpdatePost_NotOwner tests updating a post owned by another user
func TestUpdatePost_NotOwner(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	// Add a post owned by a different user (user 99)
	otherUserPost := db.Post{
		PostID:    1,
		UserID:    99, // Different from auth user 42
		CatID:     1,
		Title:     "Other user's post",
		Body:      "Body",
		CreatedAt: pgtype.Timestamp{Time: testTime, Valid: true},
		UpdatedAt: pgtype.Timestamp{Valid: false},
	}
	mockQuerier.AddPost(otherUserPost)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	updateData := UpdatePostRequest{
		Title: strPtr("Hacked Title"),
	}
	req, err := createJSONRequest("PUT", "/api/v1/post/1", updateData)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

// TestUpdatePost_Partial tests partial update (only title)
func TestUpdatePost_Partial(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	// Only update title, leave body unchanged
	updateData := UpdatePostRequest{
		Title: strPtr("New Title"),
	}
	req, err := createJSONRequest("PUT", "/api/v1/post/1", updateData)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, bodyText)
	}

	// Verify title was updated
	var resp PostResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Title != "New Title" {
		t.Errorf("expected title 'New Title', got '%s'", resp.Title)
	}
}

// TestDeletePost_Success tests deleting a post
func TestDeletePost_Success(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := http.NewRequest("DELETE", "/api/v1/post/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		bodyText, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 204, got %d: %s", w.Code, bodyText)
	}
}

// TestDeletePost_NoAuth tests deleting a post without auth
func TestDeletePost_NoAuth(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockQuerier.AddPost(tests.TestPostWhiskers)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterNoAuth(handler)

	req, err := http.NewRequest("DELETE", "/api/v1/post/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

// TestDeletePost_NotFound tests deleting a non-existent post
func TestDeletePost_NotFound(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := http.NewRequest("DELETE", "/api/v1/post/999", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

// TestDeletePost_NotOwner tests deleting a post owned by another user
func TestDeletePost_NotOwner(t *testing.T) {
	mockQuerier := tests.NewMockPostQuerier()
	// Add a post owned by a different user (user 99)
	otherUserPost := db.Post{
		PostID:    1,
		UserID:    99, // Different from auth user 42
		CatID:     1,
		Title:     "Other user's post",
		Body:      "Body",
		CreatedAt: pgtype.Timestamp{Time: testTime, Valid: true},
		UpdatedAt: pgtype.Timestamp{Valid: false},
	}
	mockQuerier.AddPost(otherUserPost)
	mockS3 := tests.NewMockS3Uploader()
	handler := NewPostHandler(mockQuerier, mockS3, "test-bucket")

	router := setupTestRouterWithAuth(handler)

	req, err := http.NewRequest("DELETE", "/api/v1/post/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	addAuthHeader(req)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

// strPtr is a helper to create string pointers
func strPtr(s string) *string {
	return &s
}
