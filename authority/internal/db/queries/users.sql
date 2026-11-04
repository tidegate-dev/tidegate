-- name: LockTransaction :exec
SELECT pg_advisory_xact_lock(@lock_id::bigint);

-- name: AnyUserExists :one
SELECT EXISTS (SELECT 1 FROM users);

-- name: CreateUser :exec
INSERT INTO users (name, password_hash) VALUES ($1, $2);

-- name: GetUserByName :one
SELECT id, password_hash FROM users WHERE lower(name) = lower(@name);
