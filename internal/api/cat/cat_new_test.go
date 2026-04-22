package cat

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"richmond-api/internal/db"
	"richmond-api/tests"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestListCats_Success(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 1,
		Name:   "Whiskers",
		Breed:  "Tabby",
		BirthDate: pgtype.Date{
			Time:  time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		Weight: 4.5,
		Habits: "Sleeping",
	})
	mock.AddCat(db.Cat{
		CatID:  2,
		UserID: 1,
		Name:   "Mittens",
		Breed:  "Siamese",
		BirthDate: pgtype.Date{
			Time:  time.Date(2022, 6, 20, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		Weight: 3.8,
		Habits: "Playing",
	})
	handler := NewCatHandler(mock, nil, nil, "test-bucket").ListCats
	res, err := testReq("GET", "/api/v1/cat/all", "", "", handler)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Errorf("expected status 200, got %d: %s", res.StatusCode, body)
	}

	var resp ListCatsResponse
	b, _ := io.ReadAll(res.Body)
	if err := json.Unmarshal(b, &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp.Cats) != 2 {
		t.Errorf("expected 2 cats, got %d", len(resp.Cats))
	}
}

func TestListCatsPagination(t *testing.T) {
	// Create 5 test cats
	mock := tests.NewMockQuerier()
	for i := int32(1); i <= 5; i++ {
		mock.AddCat(db.Cat{
			CatID:  i,
			UserID: 1,
			Name:   "Cat" + strconv.Itoa(int(i)),
			Breed:  "Tabby",
		})
	}

	handler := NewCatHandler(mock, nil, nil, "test-bucket").ListCats

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/cat", handler)

	// First page: limit=2, offset=0
	req1, err := http.NewRequest("GET", "/api/v1/cat?limit=2&offset=0", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w1.Code)
	}

	var resp1 ListCatsResponse
	if err := json.Unmarshal(w1.Body.Bytes(), &resp1); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp1.Cats) != 2 {
		t.Errorf("expected 2 cats on first page, got %d", len(resp1.Cats))
	}

	// Second page: limit=2, offset=2
	req2, err := http.NewRequest("GET", "/api/v1/cat?limit=2&offset=2", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w2.Code)
	}

	var resp2 ListCatsResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp2); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp2.Cats) != 2 {
		t.Errorf("expected 2 cats on second page, got %d", len(resp2.Cats))
	}
}

func TestGetCat_Success(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 42,
		Name:   "Whiskers",
		Breed:  "Tabby",
		BirthDate: pgtype.Date{
			Time:  time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		Weight: 4.5,
		Habits: "Sleeping",
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").GetCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.GET("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("GET", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, body)
	}

	var resp CatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Name != "Whiskers" {
		t.Errorf("expected cat name Whiskers, got %s", resp.Name)
	}
}

func TestGetCat_NotFound(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 42,
		Name:   "Whiskers",
		Breed:  "Tabby",
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").GetCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.GET("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("GET", "/api/v1/cat/999", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 404, got %d: %s", w.Code, body)
	}
}

func TestGetCat_Unauthorized(t *testing.T) {
	mock := tests.NewMockQuerier()

	handler := NewCatHandler(mock, nil, nil, "test-bucket").GetCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.GET("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("GET", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 401, got %d: %s", w.Code, body)
	}
}

func TestUpdateCat_Success(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 42,
		Name:   "Whiskers",
		Breed:  "Tabby",
		Weight: 4.5,
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").UpdateCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.PUT("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("PUT", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewBufferString(`{"name": "NewName"}`))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 200, got %d: %s", w.Code, body)
	}

	var resp CatResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if resp.Name != "NewName" {
		t.Errorf("expected cat name NewName, got %s", resp.Name)
	}
}

func TestUpdateCat_NotOwner(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	// Cat owned by user 1, but token is for user 42
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 1,
		Name:   "Whiskers",
		Breed:  "Tabby",
		Weight: 4.5,
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").UpdateCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.PUT("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("PUT", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	req.Body = io.NopCloser(bytes.NewBufferString(`{"name": "NewName"}`))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 403, got %d: %s", w.Code, body)
	}
}

func TestDeleteCat_Success(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 42,
		Name:   "Whiskers",
		Breed:  "Tabby",
		Weight: 4.5,
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").DeleteCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.DELETE("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("DELETE", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 204, got %d: %s", w.Code, body)
	}
}

func TestDeleteCat_NotOwner(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddSession("test-token", db.Session{
		SessionID: 1,
		UserID:    42,
		Token:     "test-token",
	})
	// Cat owned by user 1, but token is for user 42
	mock.AddCat(db.Cat{
		CatID:  1,
		UserID: 1,
		Name:   "Whiskers",
		Breed:  "Tabby",
		Weight: 4.5,
	})

	handler := NewCatHandler(mock, nil, nil, "test-bucket").DeleteCat

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(tests.AuthMiddleware(mock))
	router.DELETE("/api/v1/cat/:id", handler)

	req, err := http.NewRequest("DELETE", "/api/v1/cat/1", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		body, _ := io.ReadAll(w.Body)
		t.Errorf("expected status 403, got %d: %s", w.Code, body)
	}
}
