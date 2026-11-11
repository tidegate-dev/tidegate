-- +goose Up
CREATE TABLE sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- A hash, so a copy of the table can't sign anyone in.
    token_hash bytea NOT NULL UNIQUE CHECK (length(token_hash) = 32),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id ON sessions (user_id);
