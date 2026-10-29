-- +goose Up
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Compare names as lower(name), so lookups ignore case and use users_name_key.
    name text NOT NULL,
    password_hash text
);

CREATE UNIQUE INDEX users_name_key ON users (lower(name));
