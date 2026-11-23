-- +goose Up
CREATE TABLE roles (
    id uuid CONSTRAINT roles_pk PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL CONSTRAINT roles_project_fk REFERENCES projects (id),
    name text NOT NULL CONSTRAINT roles_name_check_not_blank CHECK (btrim(name) <> ''),
    create_roles boolean NOT NULL,
    escalate boolean NOT NULL,
    create_bindings boolean NOT NULL,
    bind boolean NOT NULL
);

CREATE UNIQUE INDEX roles_name_unique_in_project ON roles (project_id, lower(name));
