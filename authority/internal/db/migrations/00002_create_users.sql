-- +goose Up
CREATE TABLE users (
    id uuid CONSTRAINT users_pk PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Compare names as lower(name), so lookups ignore case and use users_name_unique.
    name text NOT NULL CONSTRAINT users_name_check_not_blank CHECK (btrim(name) <> ''),
    password_hash text
);

CREATE UNIQUE INDEX users_name_unique ON users (lower(name));
