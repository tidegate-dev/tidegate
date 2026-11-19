-- name: InsertAsset :exec
INSERT INTO assets (id, project_id, name, kind, labels, details)
VALUES (@id, @project_id, @name, @kind, @labels, @details);

-- name: ListAssetsOfProject :many
SELECT id, project_id, name, kind, labels, details
FROM assets
WHERE project_id = @project_id
ORDER BY lower(name);
