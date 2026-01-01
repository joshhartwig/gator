-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetFeeds :many
SELECT
  feeds.id,
  feeds.created_at,
  feeds.updated_at,
  feeds.name,
  feeds.url,
  feeds.user_id,
  users.name AS user_name
FROM feeds
JOIN users ON users.id = feeds.user_id;

-- name: GetFeedByUrl :one
SELECT
  feeds.id,
  feeds.created_at,
  feeds.updated_at,
  feeds.name,
  feeds.url,
  feeds.user_id
FROM feeds
WHERE feeds.url = $1;

-- name: MarkFeedFetched :one
UPDATE feeds
SET 
  last_fetched_at = NOW(), 
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetNextFeedToFetch :one
SELECT *
FROM feeds
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;