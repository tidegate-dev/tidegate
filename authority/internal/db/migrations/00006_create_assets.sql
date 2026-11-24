-- +goose Up
CREATE TABLE assets (
    id uuid CONSTRAINT assets_pk PRIMARY KEY,
    project_id uuid NOT NULL CONSTRAINT assets_project_fk REFERENCES projects (id),
    name text NOT NULL CONSTRAINT assets_name_check_not_blank CHECK (btrim(name) <> ''),
    kind text NOT NULL CONSTRAINT assets_kind_check_known CHECK (kind = 'ssh'),
    labels jsonb NOT NULL CONSTRAINT assets_labels_check_is_object CHECK (jsonb_typeof(labels) = 'object'),
    details jsonb NOT NULL CONSTRAINT assets_details_check_is_object CHECK (jsonb_typeof(details) = 'object')
);

CREATE UNIQUE INDEX assets_name_unique_in_project ON assets (project_id, lower(name));
