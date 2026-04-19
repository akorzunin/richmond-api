package cat

import (
	"io"
	"net/http"
	"richmond-api/tests"
	"testing"
)

const testCatJSON = `{
	"name": "Whiskers",
	"birth_date": "2023-01-15",
	"breed": "Tabby",
	"habits": "Sleeping",
	"weight": 4.5
}`

func TestCreateCat_Success(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		tests.NewMockPool(),
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	res, err := testReq(
		"POST",
		"/api/v1/cat/new",
		testCatJSON,
		"cat.jpg",
		handler,
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
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	res, err := testReq(
		"POST",
		"/api/v1/cat/new",
		testCatJSON,
		"",
		handler,
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
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	res, err := testReq("POST", "/api/v1/cat/new", "", "cat.jpg", handler)
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
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	res, err := testReq(
		"POST",
		"/api/v1/cat/new",
		`{"name": "Whiskers", invalid}`,
		"cat.jpg",
		handler,
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
		tests.NewMockPool(),
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	pdfMagicBytes := []byte{0x25, 0x50, 0x44, 0x46, 0x2D, 0x31, 0x2E, 0x34}
	res, err := testReqWithFileContent(
		"POST",
		"/api/v1/cat/new",
		testCatJSON,
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
		&tests.MockS3Uploader{},
		"test-bucket",
	).CreateCat
	res, err := testReqNoAuth(
		"POST",
		"/api/v1/cat/new",
		testCatJSON,
		"cat.jpg",
		handler,
	)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", res.StatusCode)
	}
}
