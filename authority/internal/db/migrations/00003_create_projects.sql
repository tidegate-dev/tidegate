-- +goose Up
CREATE TABLE projects (
    id uuid CONSTRAINT projects_pk PRIMARY KEY DEFAULT gen_random_uuid(),
    -- Compare names as lower(name), so lookups ignore case and use projects_name_unique.
    name text NOT NULL CONSTRAINT projects_name_check_not_blank CHECK (btrim(name) <> '')
);

CREATE UNIQUE INDEX projects_name_unique ON projects (lower(name));
