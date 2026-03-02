-- +goose Up
CREATE TABLE IF NOT EXISTS sessions (
    session_id SERIAL PRIMARY KEY,
    user_id INT NOT NULL REFERENCES users(user_id),
    token VARCHAR(255) NOT NULL UNIQUE,
    expires_at TIMESTAMP NOT NULL
);
CREATE INDEX idx_sessions_token ON sessions(token);

-- +goose Down
DROP TABLE IF EXISTS sessions;
