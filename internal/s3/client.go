package s3

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds S3 client configuration settings.
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
	Client    *minio.Client
}

// NewClientFromEnv creates a new S3 client from environment variables.
// Required env vars: S3_ENDPOINT, S3_ACCESS_KEY, S3_SECRET_KEY
// Optional env vars: S3_USE_SSL (default "false"), S3_BUCKET (default "main")
func NewClientFromEnv() (*S3Config, error) {
	endpoint := os.Getenv("S3_ADDRESS")
	if endpoint == "" {
		return nil, fmt.Errorf(
			"S3_ADDRESS environment variable is required",
		)
	}

	accessKey := os.Getenv("ACCESS_KEY")
	if accessKey == "" {
		return nil, fmt.Errorf(
			"ACCESS_KEY environment variable is required",
		)
	}

	secretKey := os.Getenv("SECRET_KEY")
	if secretKey == "" {
		return nil, fmt.Errorf(
			"SECRET_KEY environment variable is required",
		)
	}

	useSSLStr := os.Getenv("S3_USE_SSL")
	useSSL := strings.ToLower(useSSLStr) == "true"

	bucket := os.Getenv("S3_BUCKET")
	if bucket == "" {
		bucket = "main"
	}

	client, err := NewClient(endpoint, accessKey, secretKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	cfg := &S3Config{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		UseSSL:    useSSL,
		Bucket:    bucket,
		Client:    client,
	}

	return cfg, nil
}

// NewClient creates a new MinIO client with the given credentials.
func NewClient(
	endpoint, accessKey, secretKey string,
	useSSL bool,
) (*minio.Client, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}

	return client, nil
}

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

func (s *S3Adapter) Upload(key string, data []byte) error {
	return UploadImage(s.Client, s.Bucket, key, data)
}

func (s *S3Adapter) Endpoint() string {
	return s.Client.EndpointURL().Host
}
