package config

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config holds S3 client configuration settings.
type S3Config struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Bucket    string
	Client    *minio.Client
}

type PgConfig struct {
	Address  string
	User     string
	Password string
	Database string
	Pool     *pgxpool.Pool
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

type AppConfig struct {
	S3 *S3Config
	Pg *PgConfig
}

func NewS3Config() (*S3Config, error) {
	godotenv.Load("../.env")
	_s3 := &S3Config{
		Endpoint:  getEnv("S3_ENDPOINT", "localhost:9900"),
		AccessKey: getEnv("S3_ACCESS_KEY", "admin"),
		SecretKey: getEnv("S3_SECRET_KEY", "admin"),
		UseSSL:    getEnv("S3_USE_SSL", "false") == "true",
		Bucket:    getEnv("S3_BUCKET", "main"),
		Client:    nil,
	}
	s3c, err := minio.New(_s3.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(_s3.AccessKey, _s3.SecretKey, ""),
		Secure: _s3.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize MinIO client: %w", err)
	}
	_s3.Client = s3c
	return _s3, nil
}

func NewPgConfig() (*PgConfig, error) {
	godotenv.Load("../.env")
	_pg := &PgConfig{
		Address:  getEnv("PG_ADDRESS", "localhost:9903"),
		User:     getEnv("PG_USER", "admin"),
		Password: getEnv("PG_PASS", "admin"),
		Database: getEnv("PG_DB", "main"),
		Pool:     nil,
	}
	pool, err := pgxpool.New(
		context.Background(),
		fmt.Sprintf(
			"postgres://%s:%s@%s/%s?sslmode=disable",
			_pg.User,
			_pg.Password,
			_pg.Address,
			_pg.Database,
		),
	)
	if err != nil {
		return nil, err
	}
	_pg.Pool = pool
	return _pg, nil
}

func NewAppConfig() (*AppConfig, error) {
	_s3, err := NewS3Config()
	if err != nil {
		return nil, err
	}
	_pg, err := NewPgConfig()
	if err != nil {
		return nil, err
	}
	return &AppConfig{
		S3: _s3,
		Pg: _pg,
	}, nil
}
