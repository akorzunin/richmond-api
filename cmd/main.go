package main

import (
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "richmond-api/docs"
	"richmond-api/internal/api/auth"
	"richmond-api/internal/api/cat"
	"richmond-api/internal/api/cors"
	"richmond-api/internal/api/file"
	h "richmond-api/internal/api/health"
	"richmond-api/internal/api/post"
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
		&catS3Adapter{
			Adapter: &s3.S3Adapter{
				Client: s3Client.Client,
				Bucket: s3Client.Bucket,
			},
		},
		s3Client.Bucket,
	)
	postHandler := post.NewPostHandler(
		&tx.QuerierAdapter{Queries: queries},
		&postS3Adapter{
			Adapter: &s3.S3Adapter{
				Client: s3Client.Client,
				Bucket: s3Client.Bucket,
			},
		},
		s3Client.Bucket,
	)
	fileHandler := file.NewFileHandler(s3Client.Client, s3Client.Bucket)

	r := gin.Default()
	r.Use(cors.Middleware(cfg.AllowedOrigins))
	r.GET("/health", h.Health)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// User API
	userGroup := r.Group("/api/v1/user")
	userGroup.POST("/new", userHandler.Create)
	userGroup.POST("/login", userHandler.Login)
	userGroup.GET("", auth.Middleware(queries), userHandler.Get)

	// File API
	r.GET("/api/v1/file/*key", fileHandler.Download)

	// Cat API
	catGroup := r.Group("/api/v1/cat")
	catGroup.GET("/all", catHandler.ListCats)
	catGroup.GET("/:id", catHandler.GetCat)
	catGroup.POST("/new", auth.Middleware(queries), catHandler.CreateCat)
	catGroup.PUT("/:id", auth.Middleware(queries), catHandler.UpdateCat)
	catGroup.DELETE("/:id", auth.Middleware(queries), catHandler.DeleteCat)

	// Post API
	postGroup := r.Group("/api/v1/post")
	postGroup.GET("/all", postHandler.ListPosts)
	postGroup.GET("/:id", postHandler.GetPost)
	postGroup.POST("/new", auth.Middleware(queries), postHandler.CreatePost)
	postGroup.PUT("/:id", auth.Middleware(queries), postHandler.UpdatePost)
	postGroup.DELETE("/:id", auth.Middleware(queries), postHandler.DeletePost)

	r.Run(":8080")
}

// postS3Adapter adapts *s3.S3Adapter to post.S3Uploader
// Returns *minio.UploadInfo to satisfy fileutil.Uploader interface
type postS3Adapter struct {
	Adapter *s3.S3Adapter
}

func (a *postS3Adapter) Upload(key string, data []byte) (*minio.UploadInfo, error) {
	return a.Adapter.Upload(key, data)
}

// catS3Adapter adapts *s3.S3Adapter to cat.S3Uploader
// Returns *minio.UploadInfo to satisfy fileutil.Uploader interface
type catS3Adapter struct {
	Adapter *s3.S3Adapter
}

func (a *catS3Adapter) Upload(key string, data []byte) (*minio.UploadInfo, error) {
	return a.Adapter.Upload(key, data)
}

func (a *catS3Adapter) Endpoint() string {
	return a.Adapter.Endpoint()
}
