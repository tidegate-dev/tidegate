-- name: CreateDummy :one
INSERT INTO dummies (text) VALUES ($1) RETURNING id;

-- name: GetDummy :one
SELECT text FROM dummies WHERE id = $1;
