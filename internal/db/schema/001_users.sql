-- +goose Up
CREATE TABLE IF NOT EXISTS users (
    user_id SERIAL PRIMARY KEY,
    user_name VARCHAR(255) NOT NULL UNIQUE,
    user_pass VARCHAR(255) NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS users;
