-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id BIGINT PRIMARY KEY GENERATED ALWAYS AS IDENTITY
);

CREATE TABLE IF NOT EXISTS alias_url (
    id          INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id     BIGINT,
    "url"       VARCHAR(2000) NOT NULL,
    short_url   VARCHAR(50) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT alias_url_user_url_uk  UNIQUE (user_id, "url"),
    CONSTRAINT alias_url_short_url_uk UNIQUE (short_url)
);

CREATE INDEX IF NOT EXISTS alias_url_user_id_idx ON alias_url(user_id);
CREATE INDEX IF NOT EXISTS alias_url_url_idx ON alias_url("url");

CREATE UNIQUE INDEX IF NOT EXISTS alias_url_anon_url_uk
ON alias_url ("url")
WHERE user_id IS NULL;
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS alias_url;
DROP TABLE IF EXISTS users CASCADE;
-- +goose StatementEnd
