-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE addresses (
    id uuid NOT NULL,
    xpub_id uuid NOT NULL REFERENCES xpubs (id) ON DELETE CASCADE,
    index INT NOT NULL,
    path TEXT NOT NULL,
    address BYTEA NOT NULL,
    change BOOLEAN NOT NULL,
    touched BOOLEAN NOT NULL,
    date_modified TIMESTAMPTZ NOT NULL,
    date_created TIMESTAMPTZ NOT NULL
);

ALTER TABLE ONLY addresses ADD CONSTRAINT addresses_pkey PRIMARY KEY (id);

ALTER TABLE ONLY addresses ADD CONSTRAINT addresses_unique UNIQUE (address);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS addresses CASCADE;
