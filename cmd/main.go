package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "richmond-api/docs"
	"richmond-api/internal/api/cat"
	h "richmond-api/internal/api/health"
	"richmond-api/internal/api/user"
	"richmond-api/internal/db"
)

// @title richmond-api
// @version 0.1.0
// @description Backend for richmond app
func main() {
	r := gin.Default()

	// Connect to database
	conn, err := db.Connect()
	if err != nil {
		panic("failed to connect to database")
	}
	queries := db.New(conn)

	// Initialize handlers
	userHandler := user.NewUserHandler(queries)
	catHandler := cat.NewCatHandler(queries)

	r.GET("/health", h.Health)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// User API
	v1 := r.Group("/api/v1/user")
	v1.POST("/new", userHandler.Create)
	v1.POST("/login", userHandler.Login)
	v1.GET("", user.AuthMiddleware(queries), userHandler.Get)

	// Cat API
	catGroup := r.Group("/api/v1/cat")
	catGroup.POST("/new", user.AuthMiddleware(queries), catHandler.CreateCat)

	r.Run(":8080")
}
