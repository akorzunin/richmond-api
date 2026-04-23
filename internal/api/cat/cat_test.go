package cat

import (
	"io"
	"net/http"
	"richmond-api/tests"
	"testing"
)

func TestCreateCat_Success(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	res, err := tests.TestReq(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
		"cat.jpg",
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		bodyText, _ := io.ReadAll(res.Body)
		t.Errorf("expected status 201, got %d: %s", res.StatusCode, bodyText)
	}
}

func TestCreateCat_MissingTitlePhoto(t *testing.T) {
	handler := NewCatHandler(
		&tests.MockQuerier{},
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	res, err := tests.TestReq(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
		"",
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateCat_MissingData(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	res, err := tests.TestReq("POST", "/api/v1/cat/new", "", "cat.jpg", handler, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateCat_InvalidJSON(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	res, err := tests.TestReq(
		"POST",
		"/api/v1/cat/new",
		`{"name": "Whiskers", invalid}`,
		"cat.jpg",
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateCat_InvalidFileType(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	pdfMagicBytes := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	res, err := tests.TestReqWithFileContent(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
		"document.pdf",
		pdfMagicBytes,
		handler,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		bodyText, _ := io.ReadAll(res.Body)
		t.Errorf("expected status 400, got %d: %s", res.StatusCode, bodyText)
	}
}

func TestCreateCat_Unauthorized(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		&tests.MockPool{},
		tests.NewMockS3Adapter(),
		"test-bucket",
	).CreateCat
	res, err := tests.TestReqNoAuth(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
		"cat.jpg",
		handler,
		nil,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", res.StatusCode)
	}
}
