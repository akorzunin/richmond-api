package cors

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// Middleware returns a Gin middleware handler that adds CORS headers.
// allowedOrigins should be comma-separated list (e.g., "https://a.com,https://b.com") or "*" for all.
func Middleware(allowedOrigins string) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Determine allowed origin value
		allowOrigin := allowedOrigins // default to config value
		if allowedOrigins == "*" {
			allowOrigin = "*"
		} else if origin != "" {
			// Check if origin is in allowed list
			origins := strings.Split(allowedOrigins, ",")
			for _, o := range origins {
				if strings.TrimSpace(o) == origin {
					allowOrigin = origin
					break
				}
			}
		}

		if allowOrigin != "" {
			c.Header("Access-Control-Allow-Origin", allowOrigin)
		}

		// Handle preflight
		if c.Request.Method == "OPTIONS" {
			c.Header(
				"Access-Control-Allow-Methods",
				"GET, POST, PUT, DELETE, OPTIONS",
			)
			c.Header(
				"Access-Control-Allow-Headers",
				"Content-Type, Authorization",
			)
			c.Header("Access-Control-Max-Age", "86400")
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
