-- +goose Up
CREATE TABLE tickets (
    id uuid CONSTRAINT tickets_pk PRIMARY KEY,
    asset_id uuid NOT NULL CONSTRAINT tickets_asset_fk REFERENCES assets (id) ON DELETE CASCADE,
    user_id uuid NOT NULL CONSTRAINT tickets_user_fk REFERENCES users (id) ON DELETE CASCADE
);

CREATE INDEX tickets_asset_id_index ON tickets (asset_id);
CREATE INDEX tickets_user_id_index ON tickets (user_id);
