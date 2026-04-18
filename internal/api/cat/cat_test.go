package cat

import (
	"io"
	"net/http"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"richmond-api/internal/s3"
	"richmond-api/tests"
)

// mockS3Config creates a minimal S3 config for testing without real S3
func mockS3Config() (*s3.S3Config, error) {
	// Create a client with fake credentials - won't connect until used
	client, err := minio.New("localhost:9999", &minio.Options{
		Creds:  credentials.NewStaticV4("test", "test", ""),
		Secure: false,
	})
	if err != nil {
		return nil, err
	}
	return &s3.S3Config{
		Endpoint:  "localhost:9999",
		AccessKey: "test",
		SecretKey: "test",
		UseSSL:    false,
		Bucket:    "test-bucket",
		Client:    client,
	}, nil
}

func TestCreateCat_Success(t *testing.T) {
	handler := NewCatHandler(
		tests.NewMockQuerier(),
		tests.NewMockPool(),
		tests.NewMockS3Client(),
	).CreateCat
	res, err := tests.TestReq(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
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
	handler := NewCatHandler(tests.NewMockQuerier(), nil, nil).CreateCat
	res, err := tests.TestReq(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
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
	handler := NewCatHandler(tests.NewMockQuerier(), nil, nil).CreateCat
	res, err := tests.TestReq("POST", "/api/v1/cat/new", "", "cat.jpg", handler)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", res.StatusCode)
	}
}

func TestCreateCat_InvalidJSON(t *testing.T) {
	handler := NewCatHandler(tests.NewMockQuerier(), nil, nil).CreateCat
	res, err := tests.TestReq(
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
		tests.NewMockS3Client(),
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
	handler := NewCatHandler(tests.NewMockQuerier(), nil, nil).CreateCat
	res, err := tests.TestReqNoAuth(
		"POST",
		"/api/v1/cat/new",
		tests.TestCat,
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
