-- name: GetByURLs :many
SELECT id, short_url, "url"
FROM alias_url
WHERE "url" = ANY(sqlc.arg(urls)::text[]);


-- name: AddMany :many
WITH input AS (
  SELECT s.short_url, u.url
  FROM unnest(sqlc.arg(short_urls)::text[]) WITH ORDINALITY AS s(short_url, ord)
  JOIN unnest(sqlc.arg(urls)::text[])      WITH ORDINALITY AS u(url, ord)
    USING (ord)
),
inserted AS (
  INSERT INTO alias_url (short_url, "url")
  SELECT short_url, url
  FROM input
  ON CONFLICT DO NOTHING
  RETURNING id, short_url, "url"
)
SELECT id, short_url, "url"
FROM inserted;
