-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: CreateFile :one
INSERT INTO files (user_id, cat_id, post_id, key, url, width, height, size, quality, type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;
