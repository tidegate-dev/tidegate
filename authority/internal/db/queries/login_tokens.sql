-- name: CreateLoginToken :exec
INSERT INTO login_tokens (token_hash, user_id, expires_at)
VALUES (@token_hash, @user_id, now() + @lifetime::interval);

-- name: DeleteExpiredLoginTokens :exec
DELETE FROM login_tokens WHERE user_id = @user_id AND expires_at <= now();
