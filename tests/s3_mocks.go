package tests

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"richmond-api/internal/s3"
)

// NewMockS3Client creates a mock S3Config for testing without real S3 connections.
func NewMockS3Client() *s3.S3Config {
	// Create a client with fake credentials - won't connect until used
	client, _ := minio.New("localhost:9999", &minio.Options{
		Creds:  credentials.NewStaticV4("test", "test", ""),
		Secure: false,
	})

	return &s3.S3Config{
		Endpoint:  "localhost:9999",
		AccessKey: "test",
		SecretKey: "test",
		UseSSL:    false,
		Bucket:    "test-bucket",
		Client:    client,
	}
}

// MockS3Uploader implements S3Uploader for testing
type MockS3Uploader struct{}

func (m *MockS3Uploader) Upload(key string, data []byte) error {
	return nil
}

func (m *MockS3Uploader) Endpoint() string {
	return "localhost:9999"
}
