package cat

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	e "richmond-api/internal/api/errors"
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

// CreateCatRequest represents the JSON data for creating a cat
type CreateCatRequest struct {
	Name      string  `json:"name"`
	BirthDate string  `json:"birth_date"`
	Breed     string  `json:"breed"`
	Habits    string  `json:"habits"`
	Weight    float64 `json:"weight"`
}

// CreateCatResponse represents the response after creating a cat
type CreateCatResponse struct {
	CatID         string         `json:"cat_id"`
	TitlePhoto    FileMetadata   `json:"title_photo"`
	GalleryPhotos []FileMetadata `json:"gallery_photos"`
}

// Querier interface for future database operations
type Querier interface {
	// Placeholder - add methods as needed
}

// CatHandler handles cat-related requests
type CatHandler struct {
	queries Querier
}

// NewCatHandler creates a new CatHandler
func NewCatHandler(queries Querier) *CatHandler {
	return &CatHandler{queries: queries}
}

// allowedImageTypes contains the allowed MIME types for images
var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
}

// CreateCat handles POST /api/v1/cat/new
// @Summary Create a new cat
// @Description Creates a new cat with photos (multipart/form-data)
// @Tags cat
// @Accept multipart/form-data
// @Produce json
// @Param data formData string true "JSON cat data"
// @Param file formData []file true "Photo files (first is title photo)"
// @Success 201 {object} CreateCatResponse
// @Failure 400 {object} e.ErrorResponse
// @Failure 401 {object} e.ErrorResponse
// @Router /api/v1/cat/new [post]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *CatHandler) CreateCat(c *gin.Context) {
	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)
	if err := c.Request.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "failed to parse multipart form"},
		)
		return
	}
	// Extract JSON from "data" form field
	dataField := c.PostForm("data")
	if dataField == "" {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "missing cat data"},
		)
		return
	}
	var req CreateCatRequest
	if err := json.Unmarshal([]byte(dataField), &req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: fmt.Sprintf("invalid json: %v", err)},
		)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.Name) == "" {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "name is required"},
		)
		return
	}
	if req.Weight < 0 {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "weight cannot be negative"},
		)
		return
	}

	// 4. Get files from multipart form
	form := c.Request.MultipartForm
	if form == nil || form.File == nil {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "title photo required"},
		)
		return
	}

	files := form.File["file"]
	if len(files) == 0 {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "title photo required"},
		)
		return
	}

	// Validate gallery photo count limit (first file is title photo, rest are gallery)
	if len(files) > 21 { // 1 title + 20 gallery max
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{
				Error: "too many files: maximum is 20 gallery photos",
			},
		)
		return
	}

	// 5. Validate and extract title photo (first file)
	titlePhoto, err := h.processFile(files[0], userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: err.Error()})
		return
	}

	// 6. Process gallery photos (remaining files, optional)
	var galleryPhotos []FileMetadata
	for i := 1; i < len(files); i++ {
		photo, err := h.processFile(files[i], userID)
		if err != nil {
			c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: err.Error()})
			return
		}
		galleryPhotos = append(galleryPhotos, *photo)
	}

	// TODO: Save to database (placeholder - user_id: %v, req: %+v)

	// 7. Return success response
	// Generate placeholder cat_id (will be from DB in future)
	catID := fmt.Sprintf("cat_%d_%d", userID, len(files))

	if galleryPhotos == nil {
		galleryPhotos = []FileMetadata{}
	}

	c.JSON(http.StatusCreated, CreateCatResponse{
		CatID:         catID,
		TitlePhoto:    *titlePhoto,
		GalleryPhotos: galleryPhotos,
	})
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

	contentType := http.DetectContentType(buffer[:n])
	if !allowedImageTypes[contentType] {
		return "", fmt.Errorf("invalid file type: only images allowed")
	}
	return contentType, nil
}

// processFile validates and extracts metadata from an uploaded file
func (h *CatHandler) processFile(
	file *multipart.FileHeader,
	userID int32,
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

	// Generate safe key to prevent path traversal
	safeName := filepath.Base(file.Filename)
	if safeName == "." || safeName == "" {
		safeName = "unknown"
	}
	key := fmt.Sprintf("cat/%d/%s_%s", userID, uuid.New().String(), safeName)

	// Extract metadata
	metadata := &FileMetadata{
		Key:    key,
		URL:    "", // Will be S3 URL in future
		Size:   file.Size,
		Type:   contentType,
		Width:  0, // Will be extracted in future
		Height: 0,
	}

	return metadata, nil
}
