-- name: InsertProject :exec
INSERT INTO projects (id, name) VALUES (@id, @name);

-- name: GetProjectByName :one
SELECT id, name FROM projects WHERE lower(name) = lower(@name);

-- name: ListProjects :many
SELECT id, name FROM projects ORDER BY lower(name);
