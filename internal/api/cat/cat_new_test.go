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

	"richmond-api/internal/api/auth"
	"richmond-api/internal/db"
	"richmond-api/tests"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestListCats_Success(t *testing.T) {
	mock := tests.NewMockQuerier()
	mock.AddCat(tests.TestCatWhiskers)
	mock.AddCat(tests.TestCatMittens)
	handler := NewCatHandler(mock, nil, nil, "test-bucket").ListCats
	res, err := tests.TestReq("GET", "/api/v1/cat/all", "", "", handler, nil)
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
	const testLen = 5
	mock := tests.NewMockQuerier()
	for i := int32(1); i <= testLen; i++ {
		mock.AddCat(db.Cat{
			CatID:  i,
			UserID: 1,
			Name:   "Cat" + strconv.Itoa(int(i)),
			Breed:  "Tabby",
		})
	}
	handler := NewCatHandler(mock, nil, nil, "test-bucket").ListCats

	_tests := []struct {
		name   string
		limit  int
		offset int
		expect int
	}{
		{
			name:   "one page",
			limit:  2,
			offset: 0,
			expect: 2,
		},
		{
			name:   "page w offset",
			limit:  2,
			offset: 2,
			expect: 2,
		},
		{
			name:   "last page",
			limit:  50,
			offset: 4,
			expect: 1,
		},
		{
			name:   "more than one page",
			limit:  50,
			offset: 0,
			expect: testLen,
		},
		{
			name:   "more than one page w offset",
			limit:  50,
			offset: 50,
			expect: testLen,
		},
	}

	for _, tt := range _tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tests.TestReq(
				"GET",
				"/api/v1/cat/all",
				"",
				"",
				handler,
				tests.UrlQueryParams{
					"limit":  {strconv.Itoa(int(tt.limit))},
					"offset": {strconv.Itoa(int(tt.offset))},
				},
			)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			if res.StatusCode != http.StatusOK {
				t.Errorf("expected status 200, got %d", res.StatusCode)
			}
			var resp1 ListCatsResponse
			b, _ := io.ReadAll(res.Body)
			if err := json.Unmarshal(b, &resp1); err != nil {
				t.Fatalf("failed to unmarshal response: %v: %s", err, string(b))
			}
			if len(resp1.Cats) != tt.expect {
				t.Errorf("expected %d cats on first page, got %d", tt.expect, len(resp1.Cats))
			}
		})
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
	router.Use(auth.Middleware(mock))
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
