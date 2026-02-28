package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Health check
// @Description Returns health status of the API
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{Message: "ok"})
}

type HealthResponse struct {
	Message string `json:"message"`
}
