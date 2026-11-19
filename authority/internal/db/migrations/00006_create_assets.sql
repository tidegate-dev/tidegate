-- +goose Up
CREATE TABLE assets (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id),
    name text NOT NULL CHECK (btrim(name) <> ''),
    kind text NOT NULL CHECK (kind = 'ssh'),
    labels jsonb NOT NULL CHECK (jsonb_typeof(labels) = 'object'),
    details jsonb NOT NULL CHECK (jsonb_typeof(details) = 'object')
);

CREATE UNIQUE INDEX assets_project_id_name_key ON assets (project_id, lower(name));
