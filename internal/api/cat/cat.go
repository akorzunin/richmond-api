package cat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/api/tx"
	"richmond-api/internal/db"
)

type Querier interface {
	CreateCat(ctx context.Context, params db.CreateCatParams) (db.Cat, error)
	GetCatByID(ctx context.Context, catID int32) (db.Cat, error)
	CreateFile(ctx context.Context, params db.CreateFileParams) (db.File, error)
	GetSessionByToken(ctx context.Context, token string) (db.Session, error)
	DeleteSession(ctx context.Context, token string) error
	DeleteUserSessions(ctx context.Context, userID int32) error
	WithTx(tx tx.TxRunner) db.TxQuerier
}

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

// S3Uploader defines the interface for uploading files to S3
type S3Uploader interface {
	Upload(key string, data []byte) (*minio.UploadInfo, error)
	Endpoint() string
}

type CatHandler struct {
	queries Querier
	db      tx.Pool
	s3      S3Uploader
	bucket  string
}

func NewCatHandler(
	queries Querier,
	db tx.Pool,
	s3Adapter S3Uploader,
	bucket string,
) *CatHandler {
	return &CatHandler{
		queries: queries,
		db:      db,
		s3:      s3Adapter,
		bucket:  bucket,
	}
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
	ctx := c.Request.Context()

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

	// Guard clause: ensure required services are available before DB operations
	if h.db == nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "database not configured"},
		)
		return
	}
	if h.s3 == nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "S3 client not configured"},
		)
		return
	}

	// Begin transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to begin transaction"},
		)
		return
	}
	defer tx.Rollback(ctx)

	txQueries := h.queries.WithTx(tx)

	// Process title photo (first file)
	titlePhoto, err := processFile(files[0], userID, h.s3, h.bucket)
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: err.Error()})
		return
	}

	// Save title photo to DB with null cat_id
	titleFile, err := txQueries.CreateFile(ctx, db.CreateFileParams{
		UserID:  userID,
		CatID:   pgtype.Int4{Valid: false},
		PostID:  pgtype.Int4{Valid: false},
		Key:     titlePhoto.Key,
		Url:     titlePhoto.URL,
		Width:   int32(titlePhoto.Width),
		Height:  int32(titlePhoto.Height),
		Size:    titlePhoto.Size,
		Quality: "original",
		Type:    titlePhoto.Type,
	})
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{
				Error: fmt.Sprintf("failed to save title photo: %v", err),
			},
		)
		return
	}

	// Process gallery photos (remaining files, optional)
	var galleryPhotos []FileMetadata
	var galleryFileIDs []int32
	for i := 1; i < len(files); i++ {
		photo, err := processFile(files[i], userID, h.s3, h.bucket)
		if err != nil {
			tx.Rollback(ctx)
			c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: err.Error()})
			return
		}

		galleryFile, err := txQueries.CreateFile(ctx, db.CreateFileParams{
			UserID:  userID,
			CatID:   pgtype.Int4{Valid: false},
			PostID:  pgtype.Int4{Valid: false},
			Key:     photo.Key,
			Url:     photo.URL,
			Width:   int32(photo.Width),
			Height:  int32(photo.Height),
			Size:    photo.Size,
			Quality: "original",
			Type:    photo.Type,
		})
		if err != nil {
			tx.Rollback(ctx)
			c.JSON(
				http.StatusInternalServerError,
				e.ErrorResponse{
					Error: fmt.Sprintf("failed to save gallery photo: %v", err),
				},
			)
			return
		}

		galleryPhotos = append(galleryPhotos, *photo)
		galleryFileIDs = append(galleryFileIDs, galleryFile.ID)
	}

	// Parse birth_date if provided
	var birthDate pgtype.Date
	if req.BirthDate != "" {
		parsed, err := time.Parse("2006-01-02", req.BirthDate)
		if err != nil {
			tx.Rollback(ctx)
			c.JSON(
				http.StatusBadRequest,
				e.ErrorResponse{
					Error: "invalid birth_date format, use YYYY-MM-DD",
				},
			)
			return
		}
		birthDate = pgtype.Date{Time: parsed, Valid: true}
	} else {
		birthDate = pgtype.Date{Valid: false}
	}

	// Create cat record
	cat, err := txQueries.CreateCat(ctx, db.CreateCatParams{
		UserID:    userID,
		Name:      req.Name,
		BirthDate: birthDate,
		Breed:     req.Breed,
		Weight:    req.Weight,
		Habits:    req.Habits,
	})
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{
				Error: fmt.Sprintf("failed to create cat: %v", err),
			},
		)
		return
	}

	// Update title photo with cat_id
	_, err = tx.Exec(
		ctx,
		"UPDATE files SET cat_id = $1 WHERE id = $2",
		cat.CatID,
		titleFile.ID,
	)
	if err != nil {
		tx.Rollback(ctx)
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{
				Error: fmt.Sprintf("failed to update title photo: %v", err),
			},
		)
		return
	}

	// Update gallery photos with cat_id
	for _, fileID := range galleryFileIDs {
		_, err = tx.Exec(
			ctx,
			"UPDATE files SET cat_id = $1 WHERE id = $2",
			cat.CatID,
			fileID,
		)
		if err != nil {
			tx.Rollback(ctx)
			c.JSON(
				http.StatusInternalServerError,
				e.ErrorResponse{
					Error: fmt.Sprintf(
						"failed to update gallery photo: %v",
						err,
					),
				},
			)
			return
		}
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		tx.Rollback(ctx)
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to commit transaction"},
		)
		return
	}

	if galleryPhotos == nil {
		galleryPhotos = []FileMetadata{}
	}

	c.JSON(http.StatusCreated, CreateCatResponse{
		CatID:         fmt.Sprintf("%d", cat.CatID),
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

func processFile(
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

	// Upload to S3
	if _, err := uploader.Upload(key, data); err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Build public URL
	url := fmt.Sprintf(
		"https://%s/%s/%s",
		uploader.Endpoint(),
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
