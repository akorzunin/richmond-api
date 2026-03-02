package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"richmond-api/internal/db"
)

// AuthMiddleware returns a gin middleware that validates Bearer tokens
func AuthMiddleware(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "authorization header required"})
			c.Abort()
			return
		}

		// Extract Bearer token
		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		session, err := queries.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired token"})
			c.Abort()
			return
		}

		// Set user_id in context
		c.Set("user_id", session.UserID)
		c.Next()
	}
}
