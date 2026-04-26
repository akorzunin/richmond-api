package cat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/api/fileutil"
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
	ListCats(ctx context.Context, arg db.ListCatsParams) ([]db.Cat, error)
	UpdateCat(ctx context.Context, arg db.UpdateCatParams) (db.Cat, error)
	DeleteCat(ctx context.Context, arg db.DeleteCatParams) error
	GetFilesByCatID(ctx context.Context, catID pgtype.Int4) ([]db.File, error)
	WithTx(tx tx.TxRunner) db.TxQuerier
}

// FileMetadata is re-exported from fileutil for convenience
type FileMetadata = fileutil.FileMetadata

// CreateCatRequest represents the JSON data for creating a cat
type CreateCatRequest struct {
	Name      string  `json:"name"`
	BirthDate string  `json:"birthDate"`
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

// CatResponse represents a cat with photos
type CatResponse struct {
	CatID         int32          `json:"cat_id"`
	UserID        int32          `json:"user_id"`
	Name          string         `json:"name"`
	BirthDate     string         `json:"birth_date"`
	Breed         string         `json:"breed"`
	Weight        float64        `json:"weight"`
	Habits        string         `json:"habits"`
	CreatedAt     string         `json:"created_at"`
	TitlePhoto    FileMetadata   `json:"title_photo"`
	GalleryPhotos []FileMetadata `json:"gallery_photos"`
}

// UpdateCatRequest represents the JSON data for updating a cat (all fields optional)
type UpdateCatRequest struct {
	Name      *string  `json:"name,omitempty"`
	BirthDate *string  `json:"birth_date,omitempty"`
	Breed     *string  `json:"breed,omitempty"`
	Habits    *string  `json:"habits,omitempty"`
	Weight    *float64 `json:"weight,omitempty"`
}

// ListCatsResponse represents the response for listing cats
type ListCatsResponse struct {
	Cats   []CatResponse `json:"cats"`
	Limit  int32         `json:"limit"`
	Offset int32         `json:"offset"`
	Total  int32         `json:"total"`
}

// S3Uploader defines the interface for uploading files to S3 (original cat.go interface)
// Note: cat.go uses a different return type (*minio.UploadInfo)
type S3Uploader interface {
	Upload(key string, data []byte) (interface{}, error)
	Endpoint() string
}

// s3Adapter wraps the original S3Uploader to implement fileutil.Uploader
type s3Adapter struct {
	catS3 S3Uploader
}

func (a *s3Adapter) Upload(key string, data []byte) (interface{}, error) {
	return a.catS3.Upload(key, data)
}

type CatHandler struct {
	queries       Querier
	db            tx.Pool
	s3            S3Uploader
	bucket        string
	fileProcessor *fileutil.FileProcessor
}

func NewCatHandler(
	queries Querier,
	db tx.Pool,
	s3Adapter S3Uploader,
	bucket string,
) *CatHandler {
	fp := fileutil.NewFileProcessor(
		s3Adapter,
		bucket,
		"cat/",
	)
	return &CatHandler{
		queries:       queries,
		db:            db,
		s3:            s3Adapter,
		bucket:        bucket,
		fileProcessor: fp,
	}
}

// catResponseFromDB converts a db.Cat to CatResponse with optional file loading
func catResponseFromDB(cat db.Cat, files []db.File) CatResponse {
	var titlePhoto FileMetadata
	var galleryPhotos []FileMetadata

	if len(files) > 0 {
		titlePhoto = FileMetadata{
			Key:    files[0].Key,
			URL:    files[0].Url,
			Width:  int(files[0].Width),
			Height: int(files[0].Height),
			Size:   files[0].Size,
			Type:   files[0].Type,
		}
		if len(files) > 1 {
			galleryPhotos = make([]FileMetadata, len(files)-1)
			for i, f := range files[1:] {
				galleryPhotos[i] = FileMetadata{
					Key:    f.Key,
					URL:    f.Url,
					Width:  int(f.Width),
					Height: int(f.Height),
					Size:   f.Size,
					Type:   f.Type,
				}
			}
		}
	}

	var birthDate string
	if cat.BirthDate.Valid {
		birthDate = cat.BirthDate.Time.Format("2006-01-02")
	}

	var createdAt string
	if cat.CreatedAt.Valid {
		createdAt = cat.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	return CatResponse{
		CatID:         cat.CatID,
		UserID:        cat.UserID,
		Name:          cat.Name,
		BirthDate:     birthDate,
		Breed:         cat.Breed,
		Weight:        cat.Weight,
		Habits:        cat.Habits,
		CreatedAt:     createdAt,
		TitlePhoto:    titlePhoto,
		GalleryPhotos: galleryPhotos,
	}
}

// CreateCat handles POST /api/v1/cat/new
// @Summary Create a new cat
// @Description Creates a new cat with photos (multipart/form-data)
// @Tags cat
// @Accept multipart/form-data
// @Produce json
// @Param data formData string true "JSON cat data"
// @Param file formData []file true "Photo files (first is title photo)"
// @Success 100 {object} CreateCatRequest "Cat data model"
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
	titlePhoto, err := h.fileProcessor.Process(files[0], userID)
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
	var galleryPhotos []fileutil.FileMetadata
	var galleryFileIDs []int32
	for i := 1; i < len(files); i++ {
		photo, err := h.fileProcessor.Process(files[i], userID)
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
	if !fileutil.AllowedImageTypes[contentType] {
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

// ListCats handles GET /api/v1/cat/all
// @Summary List all cats
// @Description Lists cats with pagination (no auth required)
// @Tags cat
// @Produce json
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} ListCatsResponse
// @Router /api/v1/cat/all [get]
func (h *CatHandler) ListCats(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse query params with defaults
	limitStr := c.DefaultQuery("limit", "20")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.ParseInt(limitStr, 10, 32)
	if err != nil || limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.ParseInt(offsetStr, 10, 32)
	if err != nil || offset < 0 {
		offset = 0
	}

	// Fetch cats with pagination
	cats, err := h.queries.ListCats(ctx, db.ListCatsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		e.InternalError(c, "failed to fetch cats: "+err.Error())
		return
	}

	// Build response with files for each cat
	var catResponses []CatResponse
	for _, cat := range cats {
		files, err := h.queries.GetFilesByCatID(
			ctx,
			pgtype.Int4{Int32: cat.CatID, Valid: true},
		)
		if err != nil {
			e.InternalError(c, "failed to fetch cat files: "+err.Error())
			return
		}
		catResponses = append(catResponses, catResponseFromDB(cat, files))
	}

	if catResponses == nil {
		catResponses = []CatResponse{}
	}

	c.JSON(http.StatusOK, ListCatsResponse{
		Cats:   catResponses,
		Limit:  int32(limit),
		Offset: int32(offset),
		Total:  int32(len(cats)),
	})
}

// GetCat handles GET /api/v1/cat/:id
// @Summary Get a cat by ID
// @Description Gets a cat with photos by ID
// @Tags cat
// @Produce json
// @Param id path int true "Cat ID"
// @Success 200 {object} CatResponse
// @Failure 401 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/cat/{id} [get]
func (h *CatHandler) GetCat(c *gin.Context) {
	ctx := c.Request.Context()
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid cat id"})
		return
	}
	cat, err := h.queries.GetCatByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "cat not found"})
		return
	}
	files, err := h.queries.GetFilesByCatID(
		ctx,
		pgtype.Int4{Int32: cat.CatID, Valid: true},
	)
	if err != nil {
		e.InternalError(c, "failed to fetch cat files: "+err.Error())
		return
	}
	c.JSON(http.StatusOK, catResponseFromDB(cat, files))
}

// UpdateCat handles PUT /api/v1/cat/:id
// @Summary Update a cat
// @Description Updates a cat by ID (auth required, must be owner)
// @Tags cat
// @Accept json
// @Produce json
// @Param id path int true "Cat ID"
// @Param data body UpdateCatRequest true "Update data"
// @Success 200 {object} CatResponse
// @Failure 400 {object} e.ErrorResponse
// @Failure 401 {object} e.ErrorResponse
// @Failure 403 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/cat/{id} [put]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *CatHandler) UpdateCat(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth check
	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)

	// Parse cat ID from path
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid cat id"})
		return
	}

	// Fetch cat to check ownership
	cat, err := h.queries.GetCatByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "cat not found"})
		return
	}

	// Ownership check
	if cat.UserID != userID {
		c.JSON(http.StatusForbidden, e.ErrorResponse{Error: "forbidden"})
		return
	}

	// Bind JSON to request
	var req UpdateCatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "invalid request body"},
		)
		return
	}

	// Parse birth_date if provided
	var birthDate pgtype.Date
	birthDate.Valid = false
	if req.BirthDate != nil && *req.BirthDate != "" {
		parsed, err := time.Parse("2006-01-02", *req.BirthDate)
		if err != nil {
			c.JSON(
				http.StatusBadRequest,
				e.ErrorResponse{
					Error: "invalid birth_date format, use YYYY-MM-DD",
				},
			)
			return
		}
		birthDate = pgtype.Date{Time: parsed, Valid: true}
	}

	// Validate weight if provided
	if req.Weight != nil && *req.Weight < 0 {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "weight cannot be negative"},
		)
		return
	}

	// Build update params - always set all fields, COALESCE in SQL keeps existing values for nulls
	updateParams := db.UpdateCatParams{
		CatID:     cat.CatID,
		UserID:    cat.UserID,
		Name:      cat.Name,
		BirthDate: cat.BirthDate,
		Breed:     cat.Breed,
		Weight:    cat.Weight,
		Habits:    cat.Habits,
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			c.JSON(
				http.StatusBadRequest,
				e.ErrorResponse{Error: "name cannot be empty"},
			)
			return
		}
		updateParams.Name = name
	}
	if req.BirthDate != nil {
		updateParams.BirthDate = birthDate
	}
	if req.Breed != nil {
		updateParams.Breed = *req.Breed
	}
	if req.Habits != nil {
		updateParams.Habits = *req.Habits
	}
	if req.Weight != nil {
		updateParams.Weight = *req.Weight
	}

	// Perform update
	updatedCat, err := h.queries.UpdateCat(ctx, updateParams)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to update cat"},
		)
		return
	}

	// Fetch updated files
	files, err := h.queries.GetFilesByCatID(
		ctx,
		pgtype.Int4{Int32: updatedCat.CatID, Valid: true},
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to fetch cat files"},
		)
		return
	}

	c.JSON(http.StatusOK, catResponseFromDB(updatedCat, files))
}

// DeleteCat handles DELETE /api/v1/cat/:id
// @Summary Delete a cat
// @Description Deletes a cat by ID (auth required, must be owner)
// @Tags cat
// @Produce json
// @Param id path int true "Cat ID"
// @Success 204
// @Failure 401 {object} e.ErrorResponse
// @Failure 403 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/cat/{id} [delete]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *CatHandler) DeleteCat(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth check
	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)

	// Parse cat ID from path
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid cat id"})
		return
	}

	// Fetch cat to check ownership
	cat, err := h.queries.GetCatByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "cat not found"})
		return
	}

	// Ownership check
	if cat.UserID != userID {
		c.JSON(http.StatusForbidden, e.ErrorResponse{Error: "forbidden"})
		return
	}

	// Delete cat
	if err := h.queries.DeleteCat(ctx, db.DeleteCatParams{
		CatID:  cat.CatID,
		UserID: userID,
	}); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to delete cat"},
		)
		return
	}

	c.Status(http.StatusNoContent)
}
