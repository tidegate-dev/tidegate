-- name: InsertTicket :one
INSERT INTO tickets (id, asset_id, user_id) VALUES (@id, @asset_id, @user_id)
RETURNING xmin;
