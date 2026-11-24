-- name: InsertBinding :exec
INSERT INTO bindings (id, project_id, role_id, user_id)
VALUES (@id, @project_id, @role_id, @user_id);

-- name: GetBinding :one
SELECT id, project_id, role_id, user_id FROM bindings WHERE id = @id;

-- name: ListBindingsOfProject :many
SELECT bindings.id, bindings.project_id, roles.name AS role_name, users.name AS user_name
FROM bindings
JOIN roles ON roles.id = bindings.role_id
JOIN users ON users.id = bindings.user_id
WHERE bindings.project_id = @project_id
ORDER BY lower(roles.name), lower(users.name), bindings.id;

-- name: DeleteBinding :exec
DELETE FROM bindings WHERE id = @id;
