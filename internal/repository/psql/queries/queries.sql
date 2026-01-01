-- name: Search :one
SELECT short_url 
FROM alias_url
WHERE "url" = $1
LIMIT 1;

-- name: Get :one
SELECT "url"
FROM alias_url
WHERE short_url = $1;

-- name: Add :exec
INSERT INTO alias_url (
    short_url, "url", created_at
) VALUES (
    $1, $2, $3
);

-- name: GetAllRecords :many
SELECT * 
FROM alias_url
ORDER BY id;