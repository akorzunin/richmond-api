package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "richmond-api/docs"
	h "richmond-api/internal/api/health"
)

// @title richmond-api
// @version 0.1.0
// @description Backend for richmond app
func main() {
	r := gin.Default()

	r.GET("/health", h.Health)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.Run(":8080")
}
