-- name: LockTransaction :exec
SELECT pg_advisory_xact_lock(@lock_id::bigint);

-- name: AnyUserExists :one
SELECT EXISTS (SELECT 1 FROM users);

-- name: GetUserByName :one
SELECT id, name, password_hash, xmin FROM users WHERE lower(name) = lower(@name);

-- name: InsertUser :one
INSERT INTO users (id, name, password_hash) VALUES (@id, @name, @password_hash)
RETURNING xmin;

-- name: UpdateUser :one
WITH updated AS (
    UPDATE users SET name = @name, password_hash = @password_hash
    WHERE id = @id AND xmin = @version
    RETURNING xmin
)
SELECT
    (SELECT xmin FROM updated) AS version,
    EXISTS (SELECT 1 FROM users WHERE id = @id) AS found;

-- name: DeleteUser :one
WITH deleted AS (
    DELETE FROM users WHERE id = @id AND xmin = @version
    RETURNING id
)
SELECT
    EXISTS (SELECT 1 FROM deleted) AS deleted,
    EXISTS (SELECT 1 FROM users WHERE id = @id) AS found;
