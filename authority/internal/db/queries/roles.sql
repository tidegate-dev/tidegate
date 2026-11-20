-- name: InsertRole :exec
INSERT INTO roles (id, project_id, name, create_roles, escalate, create_bindings, bind)
VALUES (@id, @project_id, @name, @create_roles, @escalate, @create_bindings, @bind);

-- name: GetRoleByName :one
SELECT id, project_id, name, create_roles, escalate, create_bindings, bind
FROM roles
WHERE project_id = @project_id AND lower(name) = lower(@name);
