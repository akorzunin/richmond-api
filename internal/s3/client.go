package s3

import (
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
)

// EnsureBucketExists creates the bucket if it doesn't exist.
func EnsureBucketExists(client *minio.Client, bucketName string) error {
	ctx := context.Background()

	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("failed to check bucket existence: %w", err)
	}

	if exists {
		fmt.Printf("Bucket '%s' already exists\n", bucketName)
		return nil
	}

	err = client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
	if err != nil {
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	return nil
}

type S3Adapter struct {
	Client *minio.Client
	Bucket string
}

func (s *S3Adapter) Upload(key string, data []byte) (*minio.UploadInfo, error) {
	return UploadImage(s.Client, s.Bucket, key, data)
}

func (s *S3Adapter) Download(key string) ([]byte, error) {
	return GetFile(s.Client, s.Bucket, key)
}

func (s *S3Adapter) Endpoint() string {
	return s.Client.EndpointURL().Host
}
