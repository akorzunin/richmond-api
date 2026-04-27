package fileutil

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/minio/minio-go/v7"
)

type mockUploader struct {
	uploadFunc func(key string, data []byte) (*minio.UploadInfo, error)
}

func (m *mockUploader) Upload(key string, data []byte) (*minio.UploadInfo, error) {
	return m.uploadFunc(key, data)
}

func TestNewFileProcessor(t *testing.T) {
	uploader := &mockUploader{}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	if processor == nil {
		t.Fatal("expected non-nil FileProcessor")
	}
	if processor.bucket != "test-bucket" {
		t.Errorf("expected bucket test-bucket, got %s", processor.bucket)
	}
	if processor.prefix != "cat/" {
		t.Errorf("expected prefix cat/, got %s", processor.prefix)
	}
}

func TestProcess_Success(t *testing.T) {
	uploader := &mockUploader{
		uploadFunc: func(key string, data []byte) (*minio.UploadInfo, error) {
			return &minio.UploadInfo{Key: key}, nil
		},
	}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	fileHeader := createMultipartFile(t, "test.jpg", content)

	metadata, err := processor.Process(fileHeader, 123)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metadata == nil {
		t.Fatal("expected non-nil FileMetadata")
	}
	if metadata.Type != "image/jpeg" {
		t.Errorf("expected type image/jpeg, got %s", metadata.Type)
	}
	if metadata.Size != int64(len(content)) {
		t.Errorf("expected size %d, got %d", len(content), metadata.Size)
	}
	if metadata.Key == "" {
		t.Error("expected non-empty key")
	}
}

func TestProcess_FileTooLarge(t *testing.T) {
	uploader := &mockUploader{}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	content := make([]byte, 11<<20)
	fileHeader := createMultipartFile(t, "large.jpg", content)

	_, err := processor.Process(fileHeader, 123)

	if err == nil {
		t.Fatal("expected error for file too large")
	}
	if err.Error() != "file too large: maximum size is 10MB" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcess_InvalidFileType(t *testing.T) {
	uploader := &mockUploader{}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	content := []byte("not an image")
	fileHeader := createMultipartFile(t, "test.txt", content)

	_, err := processor.Process(fileHeader, 123)

	if err == nil {
		t.Fatal("expected error for invalid file type")
	}
	if err.Error() != "invalid file type: only images allowed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcess_UploadError(t *testing.T) {
	uploader := &mockUploader{
		uploadFunc: func(key string, data []byte) (*minio.UploadInfo, error) {
			return nil, errors.New("upload failed")
		},
	}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	content := []byte{0xFF, 0xD8, 0xFF, 0xE0}
	fileHeader := createMultipartFile(t, "test.jpg", content)

	_, err := processor.Process(fileHeader, 123)

	if err == nil {
		t.Fatal("expected error when upload fails")
	}
	if err.Error() != "failed to upload to S3: upload failed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestProcess_PathTraversal(t *testing.T) {
	uploader := &mockUploader{
		uploadFunc: func(key string, data []byte) (*minio.UploadInfo, error) {
			return &minio.UploadInfo{Key: key}, nil
		},
	}
	processor := NewFileProcessor(uploader, "test-bucket", "cat/")

	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	fileHeader := createMultipartFile(t, "../../../etc/passwd", content)

	metadata, err := processor.Process(fileHeader, 123)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if metadata.Key == "" {
		t.Fatal("expected non-empty key")
	}
	if bytes.Contains([]byte(metadata.Key), []byte("..")) {
		t.Error("key should not contain path traversal")
	}
}

func TestDetectImageType_JPEG(t *testing.T) {
	content := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46}
	fileHeader := createMultipartFile(t, "test.jpg", content)

	contentType, err := detectImageType(fileHeader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contentType != "image/jpeg" {
		t.Errorf("expected image/jpeg, got %s", contentType)
	}
}

func TestDetectImageType_PNG(t *testing.T) {
	content := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	fileHeader := createMultipartFile(t, "test.png", content)

	contentType, err := detectImageType(fileHeader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contentType != "image/png" {
		t.Errorf("expected image/png, got %s", contentType)
	}
}

func TestDetectImageType_GIF(t *testing.T) {
	content := []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}
	fileHeader := createMultipartFile(t, "test.gif", content)

	contentType, err := detectImageType(fileHeader)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if contentType != "image/gif" {
		t.Errorf("expected image/gif, got %s", contentType)
	}
}

func TestDetectImageType_Invalid(t *testing.T) {
	content := []byte("not an image")
	fileHeader := createMultipartFile(t, "test.txt", content)

	_, err := detectImageType(fileHeader)
	if err == nil {
		t.Fatal("expected error for invalid file type")
	}
	if err.Error() != "invalid file type: only images allowed" {
		t.Errorf("unexpected error: %v", err)
	}
}

func createMultipartFile(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", `form-data; name="file"; filename="`+filename+`"`)
	header.Set("Content-Type", "application/octet-stream")

	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}

	if len(form.File["file"]) == 0 {
		t.Fatal("no file in form")
	}

	return form.File["file"][0]
}
