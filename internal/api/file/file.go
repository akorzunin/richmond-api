package file

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/s3"
)

// getFileFunc is the function type for fetching files from S3.
// It's a variable to allow mocking in tests.
var getFileFunc = s3.GetFile

// FileHandler handles file download requests
type FileHandler struct {
	client *minio.Client
	bucket string
}

// NewFileHandler creates a new FileHandler
func NewFileHandler(client *minio.Client, bucket string) *FileHandler {
	return &FileHandler{
		client: client,
		bucket: bucket,
	}
}

// Download handles GET /api/v1/file/:key
// @Summary Download a file from S3
// @Description Downloads a file by key from S3 storage
// @Tags file
// @Produce octet-stream
// @Param key path string true "File key"
// @Param quality query string false "Quality (original, thumbnail, or preview)" Enums(original, thumbnail, preview)
// @Success 200 {file} binary
// @Failure 400 {object} e.ErrorResponse
// @Failure 404 {object} e.ErrorResponse
// @Failure 500 {object} e.ErrorResponse
// @Router /api/v1/file/{key} [get]
func (h *FileHandler) Download(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		e.BadRequest(c, "key is required")
		return
	}
	quality := c.Query("quality")
	if quality != "" && quality != "original" && quality != "thumbnail" &&
		quality != "preview" {
		e.BadRequest(
			c,
			"invalid quality: must be original, thumbnail, or preview",
		)
		return
	}
	data, err := getFileFunc(h.client, h.bucket, key)
	if err != nil {
		if strings.HasPrefix(err.Error(), "No such key") {
			e.NotFound(c, "file not found")
			return
		}
		e.InternalError(c, "failed to download file: "+err.Error())
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "inline")
	c.Data(http.StatusOK, "application/octet-stream", data)
}
