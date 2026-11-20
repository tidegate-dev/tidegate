-- +goose Up
-- Postgres only lets a foreign key point at columns that are unique together.
ALTER TABLE roles ADD CONSTRAINT roles_project_id_id_key UNIQUE (project_id, id);

CREATE TABLE bindings (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id uuid NOT NULL,
    role_id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    FOREIGN KEY (project_id, role_id) REFERENCES roles (project_id, id)
);

CREATE INDEX bindings_user_id ON bindings (user_id);
