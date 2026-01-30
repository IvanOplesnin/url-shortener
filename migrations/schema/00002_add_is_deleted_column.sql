-- +goose Up
-- +goose StatementBegin
ALTER TABLE alias_url
ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE alias_url
DROP COLUMN IF EXISTS is_deleted;
-- +goose StatementEnd
