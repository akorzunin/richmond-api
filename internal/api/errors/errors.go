package errors

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

// BadRequest sends a 400 response
func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, ErrorResponse{Error: msg})
}

// Unauthorized sends a 401 response
func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, ErrorResponse{Error: msg})
}

// NotFound sends a 404 response
func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, ErrorResponse{Error: msg})
}

// Conflict sends a 409 response
func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, ErrorResponse{Error: msg})
}

// InternalError sends a 500 response
func InternalError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: msg})
}
