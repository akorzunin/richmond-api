-- name: GetCatByID :one
SELECT * FROM cats WHERE cat_id = $1;

-- name: CreateCat :one
INSERT INTO cats (user_id, name, birth_date, breed, weight, habits)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListCats :many
SELECT * FROM cats ORDER BY created_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateCat :one
UPDATE cats
SET name = COALESCE($2, name),
    birth_date = COALESCE($3, birth_date),
    breed = COALESCE($4, breed),
    weight = COALESCE($5, weight),
    habits = COALESCE($6, habits),
    updated_at = NOW()
WHERE cat_id = $1 AND user_id = $7
RETURNING *;

-- name: DeleteCat :exec
DELETE FROM cats WHERE cat_id = $1 AND user_id = $2;
