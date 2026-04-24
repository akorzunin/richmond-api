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
	"richmond-api/internal/config"
	"richmond-api/internal/db"
	"richmond-api/internal/s3"
)

// @title richmond-api
// @version 0.1.0
// @description Backend for richmond app
func main() {
	cfg, err := config.NewAppConfig()
	if err != nil {
		panic(err)
	}
	pool := cfg.Pg.Pool
	s3Client := cfg.S3
	queries := db.New(pool)
	userHandler := user.NewUserHandler(queries)
	catHandler := cat.NewCatHandler(
		&tx.QuerierAdapter{Queries: queries},
		&tx.PoolAdapter{Pool: pool},
		&s3.S3Adapter{Client: s3Client.Client, Bucket: s3Client.Bucket},
		s3Client.Bucket,
	)

	r := gin.Default()
	r.GET("/health", h.Health)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// User API
	userGroup := r.Group("/api/v1/user")
	userGroup.POST("/new", userHandler.Create)
	userGroup.POST("/login", userHandler.Login)
	userGroup.GET("", auth.Middleware(queries), userHandler.Get)

	// Cat API
	catGroup := r.Group("/api/v1/cat")
	catGroup.GET("/all", catHandler.ListCats)
	catGroup.GET("/:id", catHandler.GetCat)
	catGroup.POST("/new", auth.Middleware(queries), catHandler.CreateCat)
	catGroup.PUT("/:id", auth.Middleware(queries), catHandler.UpdateCat)
	catGroup.DELETE("/:id", auth.Middleware(queries), catHandler.DeleteCat)

	r.Run(":8080")
}
