-- +goose Up
CREATE TABLE projects (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Compare names as lower(name), so lookups ignore case and use projects_name_key.
    name text NOT NULL CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX projects_name_key ON projects (lower(name));
