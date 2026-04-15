-- name: GetCatByID :one
SELECT * FROM cats WHERE cat_id = $1;

-- name: CreateCat :one
INSERT INTO cats (user_id, name, birth_date, breed, weight, habits)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;
