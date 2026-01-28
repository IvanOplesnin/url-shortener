-- name: GetByURLs :many
SELECT id, short_url, "url"
FROM alias_url
WHERE "url" = ANY(sqlc.arg(urls)::text[])
  AND user_id = $1;


-- name: AddMany :many
WITH input AS (
  SELECT
    s.short_url,
    u.url,
    c.created_at,
    sqlc.narg(user_id)::bigint AS user_id
  FROM unnest(sqlc.arg(short_urls)::text[])        WITH ORDINALITY AS s(short_url, ord)
  JOIN unnest(sqlc.arg(urls)::text[])             WITH ORDINALITY AS u(url, ord)
    USING (ord)
  JOIN unnest(sqlc.arg(created_ats)::timestamptz[]) WITH ORDINALITY AS c(created_at, ord)
    USING (ord)
),
inserted AS (
  INSERT INTO alias_url (short_url, "url", created_at, user_id)
  SELECT short_url, url, created_at, user_id
  FROM input
  ON CONFLICT DO NOTHING
  RETURNING id, short_url, "url", created_at, user_id
)
SELECT id, short_url, "url", created_at, user_id
FROM inserted;
