-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS users (
    id              BIGSERIAL PRIMARY KEY
);

CREATE TABLE IF NOT EXISTS alias_url (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT,
    "url"       VARCHAR NOT NULL,
    short_url   VARCHAR NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL,

    CONSTRAINT alias_url_user_fk          FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT alias_url_user_url_uk      UNIQUE (user_id, "url"),
    CONSTRAINT alias_url_short_url_uk     UNIQUE (short_url)
);

CREATE INDEX IF NOT EXISTS alias_url_user_id_idx ON alias_url(user_id);
CREATE INDEX IF NOT EXISTS alias_url_url_idx ON alias_url("url");
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS alias_url;
DROP TABLE IF EXISTS users CASCADE;
-- +goose StatementEnd
