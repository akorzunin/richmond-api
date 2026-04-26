package post

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/api/fileutil"
	"richmond-api/internal/db"
)

// Querier defines the database operations needed by post handler
type Querier interface {
	GetCatByID(ctx context.Context, catID int32) (db.Cat, error)
	CreatePost(ctx context.Context, params db.CreatePostParams) (db.Post, error)
	GetPostByID(ctx context.Context, postID int32) (db.Post, error)
	ListPosts(ctx context.Context, params db.ListPostsParams) ([]db.Post, error)
	UpdatePost(ctx context.Context, params db.UpdatePostParams) (db.Post, error)
	DeletePost(ctx context.Context, params db.DeletePostParams) (int32, error)
	CreateFile(ctx context.Context, params db.CreateFileParams) (db.File, error)
	GetFilesByPostID(ctx context.Context, postID pgtype.Int4) ([]db.File, error)
}

// S3Uploader defines the interface for uploading files to S3
// This is an alias to fileutil.Uploader interface
type S3Uploader interface {
	Upload(key string, data []byte) (interface{}, error)
}

// PostHandler handles post-related API endpoints
type PostHandler struct {
	queries       Querier
	s3            S3Uploader
	bucket        string
	fileProcessor *fileutil.FileProcessor
}

// NewPostHandler creates a new PostHandler
func NewPostHandler(
	queries Querier,
	s3 S3Uploader,
	bucket string,
) *PostHandler {
	fp := fileutil.NewFileProcessor(s3, bucket, "post/")
	return &PostHandler{
		queries:       queries,
		s3:            s3,
		bucket:        bucket,
		fileProcessor: fp,
	}
}

// CreatePostRequest represents the JSON data for creating a post
type CreatePostRequest struct {
	CatID string `json:"cat_id"`
	Title string `json:"title"`
	Body  string `json:"body"`
}

// PostResponse represents a post
type PostResponse struct {
	PostID    string                  `json:"post_id"`
	CatID     string                  `json:"cat_id"`
	UserID    int32                   `json:"user_id"`
	Title     string                  `json:"title"`
	Body      string                  `json:"body"`
	Photos    []fileutil.FileMetadata `json:"photos"`
	CreatedAt string                  `json:"created_at"`
}

// FileMetadata is re-exported from fileutil for convenience
type FileMetadata = fileutil.FileMetadata

// UpdatePostRequest represents the JSON data for updating a post
type UpdatePostRequest struct {
	Title *string `json:"title,omitempty"`
	Body  *string `json:"body,omitempty"`
}

// ListPostsResponse represents the response for listing posts
type ListPostsResponse struct {
	Posts  []PostResponse `json:"posts"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
	Total  int32          `json:"total"`
}

// postResponseFromDB converts a db.Post to PostResponse
func postResponseFromDB(post db.Post, files []db.File) PostResponse {
	var createdAt string
	if post.CreatedAt.Valid {
		createdAt = post.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	}

	var photos []fileutil.FileMetadata
	for _, f := range files {
		photos = append(photos, fileutil.FileMetadata{
			Key:    f.Key,
			URL:    f.Url,
			Width:  int(f.Width),
			Height: int(f.Height),
			Size:   f.Size,
			Type:   f.Type,
		})
	}
	// Initialize empty slice instead of nil for JSON array
	if photos == nil {
		photos = []fileutil.FileMetadata{}
	}

	return PostResponse{
		PostID:    strconv.FormatInt(int64(post.PostID), 10),
		CatID:     strconv.FormatInt(int64(post.CatID), 10),
		UserID:    post.UserID,
		Title:     post.Title,
		Body:      post.Body,
		Photos:    photos,
		CreatedAt: createdAt,
	}
}

// CreatePost handles POST /api/v1/post/new
// @Summary Create a new post
// @Description Creates a new post for a cat with photos (multipart/form-data)
// @Tags post
// @Accept multipart/form-data
// @Produce json
// @Param data formData string true "JSON post data"
// @Param file formData []file true "Photo files"
// @Success 201 {object} PostResponse
// @Failure 400 {object} e.ErrorResponse
// @Failure 401 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Failure 403 {object} e.ErrorResponse
// @Router /api/v1/post/new [post]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *PostHandler) CreatePost(c *gin.Context) {
	ctx := c.Request.Context()

	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)

	if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
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
			e.ErrorResponse{Error: "missing post data"},
		)
		return
	}
	var req CreatePostRequest
	if err := json.Unmarshal([]byte(dataField), &req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: fmt.Sprintf("invalid json: %v", err)},
		)
		return
	}

	// Validate required fields
	if strings.TrimSpace(req.CatID) == "" {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "cat_id is required"},
		)
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "title is required"},
		)
		return
	}

	// Parse cat_id
	catID, err := strconv.ParseInt(req.CatID, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid cat_id"})
		return
	}

	// Fetch cat to validate ownership
	cat, err := h.queries.GetCatByID(ctx, int32(catID))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "cat not found"})
		return
	}

	// Validate cat ownership
	if cat.UserID != userID {
		c.JSON(
			http.StatusForbidden,
			e.ErrorResponse{Error: "you do not own this cat"},
		)
		return
	}

	// Guard clause: ensure required services are available before operations
	if h.s3 == nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "S3 client not configured"},
		)
		return
	}

	// Create post first
	post, err := h.queries.CreatePost(ctx, db.CreatePostParams{
		UserID: userID,
		CatID:  cat.CatID,
		Title:  strings.TrimSpace(req.Title),
		Body:   req.Body,
	})
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to create post"},
		)
		return
	}

	// Process and save files (if any)
	var postFiles []fileutil.FileMetadata
	form := c.Request.MultipartForm
	if form != nil && form.File != nil {
		files := form.File["file"]
		for _, file := range files {
			processedFile, err := h.fileProcessor.Process(file, userID)
			if err != nil {
				// Clean up: delete the post if file processing fails
				h.queries.DeletePost(ctx, db.DeletePostParams{
					PostID: post.PostID,
					UserID: userID,
				})
				c.JSON(
					http.StatusBadRequest,
					e.ErrorResponse{Error: err.Error()},
				)
				return
			}

			fileRecord, err := h.queries.CreateFile(ctx, db.CreateFileParams{
				UserID:  userID,
				CatID:   pgtype.Int4{Int32: cat.CatID, Valid: true},
				PostID:  pgtype.Int4{Int32: post.PostID, Valid: true},
				Key:     processedFile.Key,
				Url:     processedFile.URL,
				Width:   int32(processedFile.Width),
				Height:  int32(processedFile.Height),
				Size:    processedFile.Size,
				Quality: "original",
				Type:    processedFile.Type,
			})
			if err != nil {
				// Clean up: delete the post and uploaded files
				h.queries.DeletePost(ctx, db.DeletePostParams{
					PostID: post.PostID,
					UserID: userID,
				})
				c.JSON(
					http.StatusInternalServerError,
					e.ErrorResponse{
						Error: fmt.Sprintf("failed to save file: %v", err),
					},
				)
				return
			}

			postFiles = append(postFiles, fileutil.FileMetadata{
				Key:    fileRecord.Key,
				URL:    fileRecord.Url,
				Width:  int(fileRecord.Width),
				Height: int(fileRecord.Height),
				Size:   fileRecord.Size,
				Type:   fileRecord.Type,
			})
		}
	}

	c.JSON(http.StatusCreated, PostResponse{
		PostID:    strconv.FormatInt(int64(post.PostID), 10),
		CatID:     strconv.FormatInt(int64(post.CatID), 10),
		UserID:    post.UserID,
		Title:     post.Title,
		Body:      post.Body,
		Photos:    postFiles,
		CreatedAt: post.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	})
}

// ListPosts handles GET /api/v1/post/all
// @Summary List all posts
// @Description Lists posts with pagination (no auth required)
// @Tags post
// @Produce json
// @Param limit query int false "Limit (default 20, max 100)"
// @Param offset query int false "Offset (default 0)"
// @Success 200 {object} ListPostsResponse
// @Router /api/v1/post/all [get]
func (h *PostHandler) ListPosts(c *gin.Context) {
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

	// Fetch posts with pagination
	posts, err := h.queries.ListPosts(ctx, db.ListPostsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		e.InternalError(c, "failed to fetch posts: "+err.Error())
		return
	}

	// Build response
	var postResponses []PostResponse
	for _, post := range posts {
		files, err := h.queries.GetFilesByPostID(
			ctx,
			pgtype.Int4{Int32: post.PostID, Valid: true},
		)
		if err != nil {
			e.InternalError(c, "failed to fetch files: "+err.Error())
			return
		}
		postResponses = append(postResponses, postResponseFromDB(post, files))
	}

	if postResponses == nil {
		postResponses = []PostResponse{}
	}

	c.JSON(http.StatusOK, ListPostsResponse{
		Posts:  postResponses,
		Limit:  int32(limit),
		Offset: int32(offset),
		Total:  int32(len(posts)),
	})
}

// GetPost handles GET /api/v1/post/:id
// @Summary Get a post by ID
// @Description Gets a post by ID
// @Tags post
// @Produce json
// @Param id path int true "Post ID"
// @Success 200 {object} PostResponse
// @Failure 400 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/post/{id} [get]
func (h *PostHandler) GetPost(c *gin.Context) {
	ctx := c.Request.Context()

	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid post id"})
		return
	}

	post, err := h.queries.GetPostByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "post not found"})
		return
	}

	files, err := h.queries.GetFilesByPostID(
		ctx,
		pgtype.Int4{Int32: post.PostID, Valid: true},
	)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to fetch files"},
		)
		return
	}

	c.JSON(http.StatusOK, postResponseFromDB(post, files))
}

// UpdatePost handles PUT /api/v1/post/:id
// @Summary Update a post
// @Description Updates a post by ID (auth required, must be owner)
// @Tags post
// @Accept json
// @Produce json
// @Param id path int true "Post ID"
// @Param data body UpdatePostRequest true "Update data"
// @Success 200 {object} PostResponse
// @Failure 400 {object} e.ErrorResponse
// @Failure 401 {object} e.ErrorResponse
// @Failure 403 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/post/{id} [put]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *PostHandler) UpdatePost(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth check
	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)

	// Parse post ID from path
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid post id"})
		return
	}

	// Fetch post to check ownership
	post, err := h.queries.GetPostByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "post not found"})
		return
	}

	// Ownership check
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, e.ErrorResponse{Error: "forbidden"})
		return
	}

	// Bind JSON
	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "invalid request body"},
		)
		return
	}

	// Validate title if provided
	if req.Title != nil && strings.TrimSpace(*req.Title) == "" {
		c.JSON(
			http.StatusBadRequest,
			e.ErrorResponse{Error: "title cannot be empty"},
		)
		return
	}

	// Build update params
	updateParams := db.UpdatePostParams{
		PostID: post.PostID,
		Title:  post.Title,
		Body:   post.Body,
		UserID: userID,
	}

	if req.Title != nil {
		updateParams.Title = strings.TrimSpace(*req.Title)
	}
	if req.Body != nil {
		updateParams.Body = *req.Body
	}

	// Perform update
	updatedPost, err := h.queries.UpdatePost(ctx, updateParams)
	if err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to update post"},
		)
		return
	}

	c.JSON(http.StatusOK, postResponseFromDB(updatedPost, []db.File{}))
}

// DeletePost handles DELETE /api/v1/post/:id
// @Summary Delete a post
// @Description Deletes a post by ID (auth required, must be owner)
// @Tags post
// @Produce json
// @Param id path int true "Post ID"
// @Success 204
// @Failure 401 {object} e.ErrorResponse
// @Failure 403 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Router /api/v1/post/{id} [delete]
// @Security BearerAuth
// @Param Authorization header string true "Insert your access token" default(Bearer <Add access token here>)
func (h *PostHandler) DeletePost(c *gin.Context) {
	ctx := c.Request.Context()

	// Auth check
	anyUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "unauthorized"})
		return
	}
	userID := anyUserID.(int32)

	// Parse post ID from path
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, e.ErrorResponse{Error: "invalid post id"})
		return
	}

	// Fetch post to check ownership
	post, err := h.queries.GetPostByID(ctx, int32(id))
	if err != nil {
		c.JSON(http.StatusNotFound, e.ErrorResponse{Error: "post not found"})
		return
	}

	// Ownership check
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, e.ErrorResponse{Error: "forbidden"})
		return
	}

	// Delete post
	if _, err := h.queries.DeletePost(ctx, db.DeletePostParams{
		PostID: post.PostID,
		UserID: userID,
	}); err != nil {
		c.JSON(
			http.StatusInternalServerError,
			e.ErrorResponse{Error: "failed to delete post"},
		)
		return
	}

	c.Status(http.StatusNoContent)
}
