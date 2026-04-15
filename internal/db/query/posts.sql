-- name: GetPostByID :one
SELECT * FROM posts WHERE post_id = $1;

-- name: CreatePost :one
INSERT INTO posts (user_id, cat_id, title, body)
VALUES ($1, $2, $3, $4)
RETURNING *;
