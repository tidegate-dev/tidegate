-- +goose Up
-- Postgres only lets a foreign key point at columns that are unique together.
ALTER TABLE roles ADD CONSTRAINT roles_project_id_id_key UNIQUE (project_id, id);

CREATE TABLE bindings (
    id uuid CONSTRAINT bindings_pkey PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    role_id uuid NOT NULL,
    user_id uuid NOT NULL CONSTRAINT bindings_user_id_fkey REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT bindings_project_id_role_id_fkey FOREIGN KEY (project_id, role_id) REFERENCES roles (project_id, id)
);

CREATE INDEX bindings_user_id ON bindings (user_id);
