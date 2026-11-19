-- name: InsertRole :exec
INSERT INTO roles (id, project_id, name, create_roles, escalate, create_bindings, bind)
VALUES (@id, @project_id, @name, @create_roles, @escalate, @create_bindings, @bind);
