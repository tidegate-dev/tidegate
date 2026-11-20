-- +goose Up
CREATE TABLE dummies (
    id bigint GENERATED ALWAYS AS IDENTITY CONSTRAINT dummies_pkey PRIMARY KEY,
    text text NOT NULL
);
