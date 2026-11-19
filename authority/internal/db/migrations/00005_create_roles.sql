-- +goose Up
CREATE TABLE roles (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL REFERENCES projects (id),
    name text NOT NULL CHECK (btrim(name) <> ''),
    create_roles boolean NOT NULL,
    escalate boolean NOT NULL,
    create_bindings boolean NOT NULL,
    bind boolean NOT NULL
);

CREATE UNIQUE INDEX roles_project_id_name_key ON roles (project_id, lower(name));
