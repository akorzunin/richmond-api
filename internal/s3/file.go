package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/minio/minio-go/v7"
)

// CreateFile uploads file data to S3 with application/octet-stream content type.
func CreateFile(client *minio.Client, bucket, key string, data []byte) error {
	if client == nil {
		return fmt.Errorf("S3 client is nil")
	}
	if bucket == "" {
		return fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return fmt.Errorf("object key is required")
	}
	if len(data) == 0 {
		return fmt.Errorf("data is empty")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.PutObject(ctx, bucket, key, NewReadSeeker(data), int64(len(data)), minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}

	return nil
}

// GetFile downloads a file from S3 and returns the bytes.
func GetFile(client *minio.Client, bucket, key string) ([]byte, error) {
	if client == nil {
		return nil, fmt.Errorf("S3 client is nil")
	}
	if bucket == "" {
		return nil, fmt.Errorf("bucket name is required")
	}
	if key == "" {
		return nil, fmt.Errorf("object key is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	obj, err := client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to read object data: %w", err)
	}

	return data, nil
}

// NewReadSeeker wraps a byte slice as an io.ReadSeeker.
func NewReadSeeker(data []byte) io.ReadSeeker {
	return &readSeeker{data: data, offset: 0}
}

type readSeeker struct {
	data   []byte
	offset int64
}

func (rs *readSeeker) Read(p []byte) (int, error) {
	if rs.offset >= int64(len(rs.data)) {
		return 0, io.EOF
	}
	n := copy(p, rs.data[rs.offset:])
	rs.offset += int64(n)
	return n, nil
}

func (rs *readSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = rs.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(rs.data)) + offset
	default:
		return 0, fmt.Errorf("readSeeker.Seek: invalid whence")
	}
	if newOffset < 0 {
		return 0, fmt.Errorf("readSeeker.Seek: negative position")
	}
	rs.offset = newOffset
	return newOffset, nil
}
