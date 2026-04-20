package main

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "richmond-api/docs"
	"richmond-api/internal/api/auth"
	"richmond-api/internal/api/cat"
	h "richmond-api/internal/api/health"
	"richmond-api/internal/api/tx"
	"richmond-api/internal/api/user"
	"richmond-api/internal/db"
	"richmond-api/internal/s3"
)

// @title richmond-api
// @version 0.1.0
// @description Backend for richmond app
func main() {
	r := gin.Default()

	// Connect to database
	pool, err := db.ConnectWithPool()
	if err != nil {
		panic("failed to connect to database")
	}
	queries := db.New(pool)

	// Initialize S3 client
	s3Client, err := s3.NewClientFromEnv()
	if err != nil {
		panic("failed to create S3 client: " + err.Error())
	}

	// Initialize handlers
	userHandler := user.NewUserHandler(queries)
	catHandler := cat.NewCatHandler(
		&tx.QuerierAdapter{Queries: queries},
		&tx.PoolAdapter{Pool: pool},
		&s3.S3Adapter{Client: s3Client.Client, Bucket: s3Client.Bucket},
		s3Client.Bucket,
	)

	r.GET("/health", h.Health)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// User API
	userGroup := r.Group("/api/v1/user")
	userGroup.POST("/new", userHandler.Create)
	userGroup.POST("/login", userHandler.Login)
	userGroup.GET("", auth.Middleware(queries), userHandler.Get)

	// Cat API
	catGroup := r.Group("/api/v1/cat")
	catGroup.POST("/new", auth.Middleware(queries), catHandler.CreateCat)

	r.Run(":8080")
}
