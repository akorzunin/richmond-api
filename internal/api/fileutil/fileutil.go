package fileutil

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

// FileMetadata represents metadata for an uploaded file
type FileMetadata struct {
	Key    string `json:"key"`
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Size   int64  `json:"size"`
	Type   string `json:"type"`
}

// AllowedImageTypes contains the allowed MIME types for images
var AllowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// Uploader defines the interface for uploading files to S3
// The return type is interface{} to support both cat.go (returns *minio.UploadInfo)
// and post.go (returns string, error)
type Uploader interface {
	Upload(key string, data []byte) (interface{}, error)
}

// FileProcessor handles file processing with configurable S3 key prefix
type FileProcessor struct {
	uploader Uploader
	bucket   string
	prefix   string // "cat/" or "post/"
}

// NewFileProcessor creates a new FileProcessor
func NewFileProcessor(uploader Uploader, bucket, prefix string) *FileProcessor {
	return &FileProcessor{
		uploader: uploader,
		bucket:   bucket,
		prefix:   prefix,
	}
}

// Process processes a multipart file and uploads it to S3
func (p *FileProcessor) Process(
	file *multipart.FileHeader,
	userID int32,
) (*FileMetadata, error) {
	// Guard clause: validate file size first (Early Exit)
	const maxFileSize = 10 << 20 // 10MB
	if file.Size > maxFileSize {
		return nil, fmt.Errorf("file too large: maximum size is 10MB")
	}

	// Detect content type using magic bytes
	contentType, err := detectImageType(file)
	if err != nil {
		return nil, err
	}

	// Generate safe name to prevent path traversal
	safeName := filepath.Base(file.Filename)
	if safeName == "." || safeName == "" {
		safeName = "unknown"
	}

	// Generate S3 key with configurable prefix
	key := fmt.Sprintf(
		"%s%d/%s_%s",
		p.prefix,
		userID,
		uuid.New().String(),
		safeName,
	)

	// Open and read file data
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}

	// Upload to S3
	if _, err := p.uploader.Upload(key, data); err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	url := fmt.Sprintf("http://rustfs:9000/%s/%s", p.bucket, key)

	return &FileMetadata{
		Key:    key,
		URL:    url,
		Size:   file.Size,
		Type:   contentType,
		Width:  0, // Will be extracted in future
		Height: 0, // Will be extracted in future
	}, nil
}

// detectImageType reads magic bytes from file header to detect content type
func detectImageType(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read first 512 bytes for magic byte detection
	buffer := make([]byte, 512)
	n, err := src.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	// Detect content type using Go's standard library
	contentType := http.DetectContentType(buffer[:n])
	if !AllowedImageTypes[contentType] {
		return "", fmt.Errorf("invalid file type: only images allowed")
	}
	return contentType, nil
}

// S3Uploader defines the interface for uploading files to S3
type S3Uploader interface {
	Upload(key string, data []byte) (*minio.UploadInfo, error)
	Endpoint() string
}

func ProcessFile(
	file *multipart.FileHeader,
	userID int32,
	uploader S3Uploader,
	bucket string,
) (*FileMetadata, error) {
	// Validate file size (10MB max)
	const maxFileSize = 10 << 20 // 10MB
	if file.Size > maxFileSize {
		return nil, fmt.Errorf("file too large: maximum size is 10MB")
	}

	// Detect content type using magic bytes
	contentType, err := detectImageType(file)
	if err != nil {
		return nil, err
	}

	// Generate safe name to prevent path traversal
	safeName := filepath.Base(file.Filename)
	if safeName == "." || safeName == "" {
		safeName = "unknown"
	}

	// Generate S3 key
	key := fmt.Sprintf("cat/%d/%s_%s", userID, uuid.New().String(), safeName)

	// Open and read file data
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file data: %w", err)
	}
	if _, err := uploader.Upload(key, data); err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}
	url := fmt.Sprintf(
		"http://rustfs:9000/%s/%s",
		bucket,
		key,
	)
	return &FileMetadata{
		Key:    key,
		URL:    url,
		Size:   file.Size,
		Type:   contentType,
		Width:  0, // Will be extracted in future
		Height: 0,
	}, nil
}
