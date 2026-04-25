package s3

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"

	"github.com/minio/minio-go/v7"
)

// DetectContentType maps file extension to MIME type.
func DetectContentType(key string) string {
	ext := filepath.Ext(key)
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".bmp":
		return "image/bmp"
	case ".tiff", ".tif":
		return "image/tiff"
	default:
		return "application/octet-stream"
	}
}

func ValidateParams(client *minio.Client, bucket, key string) error {
	if client == nil {
		return fmt.Errorf("S3 client is nil")
	}
	if bucket == "" {
		return fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	return nil
}

// UploadImage uploads image data to S3 with auto-detected content type.
func UploadImage(
	client *minio.Client,
	bucket, key string,
	data []byte,
) (*minio.UploadInfo, error) {
	if err := ValidateParams(client, bucket, key); err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("image data is empty")
	}

	ctx := context.Background()
	contentType := DetectContentType(key)

	i, err := client.PutObject(
		ctx,
		bucket,
		key,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to upload image: %w", err)
	}

	return &i, nil
}
