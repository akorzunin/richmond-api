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

// MockS3Adapter wraps s3.S3Adapter for testing with no-op upload
type MockS3Adapter struct {
	*s3.S3Adapter
}

// NewMockS3Adapter creates a new MockS3Adapter for testing
func NewMockS3Adapter() *MockS3Adapter {
	client, _ := minio.New("localhost:9999", &minio.Options{
		Creds:  credentials.NewStaticV4("test", "test", ""),
		Secure: false,
	})
	return &MockS3Adapter{
		S3Adapter: &s3.S3Adapter{
			Client: client,
			Bucket: "test-bucket",
		},
	}
}

// Upload is a no-op for testing - overrides the embedded S3Adapter.Upload
func (m *MockS3Adapter) Upload(
	key string,
	data []byte,
) (*minio.UploadInfo, error) {
	return nil, nil
}
