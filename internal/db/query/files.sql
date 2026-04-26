-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: CreateFile :one
INSERT INTO files (user_id, cat_id, post_id, key, url, width, height, size, quality, type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetFilesByCatID :many
SELECT * FROM files WHERE cat_id = $1 ORDER BY created_at;

-- name: GetFilesByPostID :many
SELECT * FROM files WHERE post_id = $1 ORDER BY created_at;
