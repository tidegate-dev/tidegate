-- +goose Up
CREATE TABLE dummies (
    id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    text text NOT NULL
);
