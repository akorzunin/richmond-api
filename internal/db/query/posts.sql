-- name: GetPostByID :one
SELECT * FROM posts WHERE post_id = $1;

-- name: CreatePost :one
INSERT INTO posts (user_id, cat_id, title, body)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListPosts :many
SELECT * FROM posts ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: ListPostsByCatID :many
SELECT * FROM posts WHERE cat_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: ListPostsByUserID :many
SELECT * FROM posts WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3;

-- name: UpdatePost :one
UPDATE posts
SET title = COALESCE($2, title),
    body = COALESCE($3, body),
    updated_at = NOW()
WHERE post_id = $1 AND user_id = $4
RETURNING *;

-- name: DeletePost :one
DELETE FROM posts WHERE post_id = $1 AND user_id = $2 RETURNING post_id;
