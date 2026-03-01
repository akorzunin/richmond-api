package db

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func Connect() (*pgx.Conn, error) {
	// Load .env if exists
	_ = godotenv.Load()

	// Get env vars
	addr := getEnv("PG_ADDRESS", "localhost:9903")
	user := getEnv("PG_USER", "admin")
	pass := getEnv("PG_PASS", "admin")
	dbname := getEnv("PG_DB", "main")

	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, addr, dbname)

	return pgx.Connect(context.Background(), connStr)
}

func ConnectWithPool() (*pgxpool.Pool, error) {
	// Load .env if exists
	_ = godotenv.Load()

	// Get env vars
	addr := getEnv("PG_ADDRESS", "localhost:9903")
	user := getEnv("PG_USER", "admin")
	pass := getEnv("PG_PASS", "admin")
	dbname := getEnv("PG_DB", "main")

	connStr := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", user, pass, addr, dbname)

	return pgxpool.New(context.Background(), connStr)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
