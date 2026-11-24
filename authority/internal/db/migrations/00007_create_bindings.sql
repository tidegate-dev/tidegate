-- +goose Up
-- Postgres only lets a foreign key point at columns that are unique together.
ALTER TABLE roles ADD CONSTRAINT roles_id_unique_in_project UNIQUE (project_id, id);

CREATE TABLE bindings (
    id uuid CONSTRAINT bindings_pk PRIMARY KEY,
    project_id uuid NOT NULL,
    role_id uuid NOT NULL,
    user_id uuid NOT NULL CONSTRAINT bindings_user_fk REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT bindings_role_in_project_fk FOREIGN KEY (project_id, role_id) REFERENCES roles (project_id, id),
    CONSTRAINT bindings_role_and_user_unique UNIQUE (project_id, role_id, user_id)
);

CREATE INDEX bindings_user_id_index ON bindings (user_id);
