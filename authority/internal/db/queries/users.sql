-- name: LockTransaction :exec
SELECT pg_advisory_xact_lock(@lock_id::bigint);

-- name: AnyUserExists :one
SELECT EXISTS (SELECT 1 FROM users);

-- name: GetUserByName :one
SELECT id, name, password_hash, xmin FROM users WHERE lower(name) = lower(@name);

-- name: UserExists :one
SELECT EXISTS (SELECT 1 FROM users WHERE id = @id);

-- name: InsertUser :one
INSERT INTO users (id, name, password_hash) VALUES (@id, @name, @password_hash)
RETURNING xmin;

-- name: UpdateUser :one
UPDATE users SET name = @name, password_hash = @password_hash
WHERE id = @id AND xmin = @version
RETURNING xmin;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = @id AND xmin = @version;
