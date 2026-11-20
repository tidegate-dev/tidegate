-- +goose Up
CREATE TABLE assets (
    id uuid CONSTRAINT assets_pkey PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL CONSTRAINT assets_project_id_fkey REFERENCES projects (id),
    name text NOT NULL CONSTRAINT assets_name_check CHECK (btrim(name) <> ''),
    kind text NOT NULL CONSTRAINT assets_kind_check CHECK (kind = 'ssh'),
    labels jsonb NOT NULL CONSTRAINT assets_labels_check CHECK (jsonb_typeof(labels) = 'object'),
    details jsonb NOT NULL CONSTRAINT assets_details_check CHECK (jsonb_typeof(details) = 'object')
);

CREATE UNIQUE INDEX assets_project_id_name_key ON assets (project_id, lower(name));
