-- +goose Up
CREATE TABLE projects (
    id uuid CONSTRAINT projects_pkey PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Compare names as lower(name), so lookups ignore case and use projects_name_key.
    name text NOT NULL CONSTRAINT projects_name_check CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX projects_name_key ON projects (lower(name));
