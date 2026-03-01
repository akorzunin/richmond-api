package s3

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

func UploadCommand(
	client *minio.Client,
	bucket, filePath, objectKey string,
) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	return UploadImage(client, bucket, objectKey, data)
}

func DownloadCommand(
	client *minio.Client,
	bucket, objectKey, filePath string,
) error {
	data, err := DownloadImage(client, bucket, objectKey)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

func ListBucketsCommand(client *minio.Client) error {
	ctx := context.Background()
	buckets, err := client.ListBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to list buckets: %w", err)
	}

	fmt.Println("Buckets:")
	for _, bucket := range buckets {
		fmt.Printf("  - %s\n", bucket.Name)
	}

	return nil
}

func ListObjectsCommand(client *minio.Client, bucket string) error {
	ctx := context.Background()

	objectCh := client.ListObjects(ctx, bucket, minio.ListObjectsOptions{})

	fmt.Printf("Objects in bucket '%s':\n", bucket)
	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("error listing object: %w", object.Err)
		}
		fmt.Printf("  - %s (%d bytes)\n", object.Key, object.Size)
	}

	return nil
}
