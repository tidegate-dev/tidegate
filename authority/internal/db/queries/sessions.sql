-- name: GetActiveSessionByTokenHash :one
SELECT id, user_id, token_hash, created_at, expires_at, xmin FROM sessions
WHERE token_hash = @token_hash AND expires_at > @now;

-- name: SessionExists :one
SELECT EXISTS (SELECT 1 FROM sessions WHERE id = @id);

-- name: InsertSession :one
INSERT INTO sessions (id, user_id, token_hash, created_at, expires_at)
VALUES (@id, @user_id, @token_hash, @created_at, @expires_at)
RETURNING xmin;

-- name: DeleteSession :execrows
DELETE FROM sessions WHERE id = @id AND xmin = @version;

-- name: DeleteSessionsOfUser :exec
DELETE FROM sessions WHERE user_id = @user_id;

-- name: DeleteExpiredSessionsOfUser :exec
DELETE FROM sessions WHERE user_id = @user_id AND expires_at <= @now;
