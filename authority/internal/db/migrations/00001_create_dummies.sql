-- +goose Up
CREATE TABLE dummies (
    id bigint GENERATED ALWAYS AS IDENTITY CONSTRAINT dummies_pk PRIMARY KEY,
    text text NOT NULL
);
