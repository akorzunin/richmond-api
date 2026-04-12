package user

import (
	"net/http"

	e "richmond-api/internal/api/errors"
	"richmond-api/internal/db"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware validates Bearer tokens
func AuthMiddleware(queries *db.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(
				http.StatusUnauthorized,
				e.ErrorResponse{Error: "authorization header required"},
			)
			c.Abort()
			return
		}
		var token string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		} else {
			c.JSON(http.StatusUnauthorized, e.ErrorResponse{Error: "invalid authorization header format"})
			c.Abort()
			return
		}
		session, err := queries.GetSessionByToken(c.Request.Context(), token)
		if err != nil {
			c.JSON(
				http.StatusUnauthorized,
				e.ErrorResponse{Error: "invalid or expired token"},
			)
			c.Abort()
			return
		}
		c.Set("user_id", session.UserID)
		c.Next()
	}
}
