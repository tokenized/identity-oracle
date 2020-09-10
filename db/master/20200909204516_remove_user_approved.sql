-- +goose Up
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN approved;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN approved boolean NOT NULL DEFAULT false;
-- +goose StatementEnd
