-- name: CreateSession :one
INSERT INTO sessions (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING session_id, user_id, token, expires_at;

-- name: GetSessionByToken :one
SELECT session_id, user_id, token, expires_at FROM sessions
WHERE token = $1 AND expires_at > NOW();

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = $1;

-- name: DeleteUserSessions :exec
DELETE FROM sessions WHERE user_id = $1;
