-- +goose Up
CREATE TABLE sessions (
    id uuid CONSTRAINT sessions_pkey PRIMARY KEY DEFAULT gen_random_uuid(),
    -- A hash, so a copy of the table can't sign anyone in.
    token_hash bytea NOT NULL
        CONSTRAINT sessions_token_hash_key UNIQUE
        CONSTRAINT sessions_token_hash_check CHECK (length(token_hash) = 32),
    user_id uuid NOT NULL CONSTRAINT sessions_user_id_fkey REFERENCES users (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL,
    expires_at timestamptz NOT NULL
);

CREATE INDEX sessions_user_id ON sessions (user_id);
