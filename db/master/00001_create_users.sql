-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE users (
    id uuid NOT NULL,
    entity bytes NOT NULL,
    name TEXT,
    jurisdiction TEXT,
    date_created TIMESTAMPTZ NOT NULL,
    date_modified TIMESTAMPTZ NOT NULL,
    is_deleted boolean NOT NULL DEFAULT false
);

ALTER TABLE ONLY users ADD CONSTRAINT users_pkey PRIMARY KEY (id);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS users CASCADE;
