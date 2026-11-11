-- name: CreateSession :exec
INSERT INTO sessions (token_hash, user_id, expires_at)
VALUES (@token_hash, @user_id, now() + @lifetime::interval);

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE user_id = @user_id AND expires_at <= now();

-- name: GetSession :one
SELECT id, user_id FROM sessions WHERE token_hash = @token_hash AND expires_at > now();

-- name: RevokeSession :exec
DELETE FROM sessions WHERE id = @id;

-- name: RevokeUserSessions :exec
DELETE FROM sessions WHERE user_id = @user_id;
