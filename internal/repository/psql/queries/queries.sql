-- name: Search :one
SELECT short_url, is_deleted
FROM alias_url
WHERE "url" = $1 AND user_id = $2
LIMIT 1;

-- name: Get :one
SELECT "url", is_deleted
FROM alias_url
WHERE short_url = $1;

-- name: Add :exec
INSERT INTO alias_url (
    short_url, "url", created_at, user_id
) VALUES (
    $1, $2, $3, $4
);

-- name: GetAllRecords :many
SELECT  id, "url", short_url, created_at
FROM alias_url
WHERE user_id = $1
ORDER BY id;


-- name: AddUser :one
INSERT INTO users DEFAULT VALUES
RETURNING id;


-- name: GetUser :one
SELECT id
FROM users
WHERE id = $1
LIMIT 1;


-- name: UserURLs :many
SELECT id, short_url, "url"
FROM alias_url
WHERE user_id = $1;


-- name: SetDeletedBatch :exec
UPDATE alias_url
SET is_deleted = true
WHERE user_id = $1
  AND short_url = ANY(sqlc.arg(short_urls)::text[]);

-- name: SetDeletedFalse :exec
UPDATE alias_url
SET is_deleted = false
WHERE user_id = $1
  AND short_url = $2;