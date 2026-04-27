package tests

import (
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"richmond-api/internal/s3"
)

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

// Upload is a no-op for testing - returns to satisfy fileutil.Uploader
func (m *MockS3Adapter) Upload(
	key string,
	data []byte,
) (*minio.UploadInfo, error) {
	return nil, nil
}
