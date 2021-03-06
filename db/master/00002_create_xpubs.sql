-- +goose Up
-- SQL in this section is executed when the migration is applied.
CREATE TABLE xpubs (
    id uuid NOT NULL,
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    xpub BYTEA NOT NULL,
    required_signers INT NOT NULL DEFAULT 1,
    date_created TIMESTAMPTZ NOT NULL
);

ALTER TABLE ONLY xpubs ADD CONSTRAINT xpubs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY xpubs ADD CONSTRAINT xpubs_unique UNIQUE (user_id, xpub);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
DROP TABLE IF EXISTS xpubs CASCADE;
