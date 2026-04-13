-- name: GetUserByID :one
SELECT user_id, user_name, user_pass, created_at, updated_at FROM users WHERE user_id = $1;

-- name: GetUserByName :one
SELECT user_id, user_name, user_pass, created_at, updated_at FROM users WHERE user_name = $1;

-- name: CreateUser :one
INSERT INTO users (user_name, user_pass)
VALUES ($1, $2)
RETURNING user_id, user_name, user_pass, created_at, updated_at;
