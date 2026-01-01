-- +goose Up
-- +goose StatementBegin
CREATE TABLE alias_url (
    id BIGSERIAL PRIMARY KEY,
    "url" VARCHAR NOT NULL,
    short_url VARCHAR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    
    CONSTRAINT alias_url_url_uk UNIQUE ("url"),
    CONSTRAINT alias_url_short_url_uk UNIQUE (short_url)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE alias_url;
-- +goose StatementEnd

