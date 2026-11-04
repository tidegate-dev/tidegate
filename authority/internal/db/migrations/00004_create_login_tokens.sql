-- +goose Up
CREATE TABLE login_tokens (
    -- The SHA-256 hash of the token, so a copy of the table can't sign anyone in.
    token_hash bytea PRIMARY KEY CHECK (length(token_hash) = 32),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX login_tokens_user_id ON login_tokens (user_id);
