package auth

import (
	"context"
	"richmond-api/internal/api/errors"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
)

// Error messages
const (
	ErrAuthHeaderRequired      = "authorization header required"
	ErrInvalidAuthHeaderFormat = "invalid authorization header format"
	ErrInvalidOrExpiredToken   = "invalid or expired token"
	ErrUnauthorized            = "unauthorized"
)

type AuthQuerier interface {
	GetSessionByToken(ctx context.Context, token string) (db.Session, error)
}

// Middleware validates Bearer tokens and sets user_id in context
func Middleware(queries AuthQuerier) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			errors.Unauthorized(c, ErrAuthHeaderRequired)
			c.Abort()
			return
		}

		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			errors.Unauthorized(c, ErrInvalidAuthHeaderFormat)
			c.Abort()
			return
		}

		session, err := queries.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			errors.Unauthorized(c, ErrInvalidOrExpiredToken)
			c.Abort()
			return
		}

		c.Set("user_id", session.UserID)
		c.Next()
	}
}
