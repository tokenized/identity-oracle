-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE xpubs (
    id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES user (id) ON DELETE CASCADE,
    xpub BYTEA NOT NULL,
    path TEXT NOT NULL,
    date_created TIMESTAMPTZ NOT NULL
);

ALTER TABLE ONLY xpubs ADD CONSTRAINT xpubs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY xpubs ADD CONSTRAINT xpubs_unique UNIQUE (xpub);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS xpubs CASCADE;
