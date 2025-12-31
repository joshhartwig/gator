-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
  INSERT INTO feed_follows (id, created_at, updated_at, user_id, feed_id)
  VALUES ($1,$2,$3,$4,$5)
  RETURNING *
) 
SELECT
  inserted_feed_follow.*,
  feeds.name as feed_name,
  users.name as user_name
FROM inserted_feed_follow
INNER JOIN users ON users.id = inserted_feed_follow.user_id
INNER JOIN feeds ON feeds.id = inserted_feed_follow.feed_id;

-- name: GetFeedFollows :many
SELECT
  feed_follows.id,
  feed_follows.created_at,
  feed_follows.updated_at,
  feed_follows.user_id,
  feed_follows.feed_id
FROM feed_follows;

-- name: GetFeedFollowsForUser :many
SELECT
  feed_follows.id,
  feed_follows.created_at,
  feed_follows.updated_at,
  feed_follows.user_id,
  feed_follows.feed_id,
  users.name as user_name,
  feeds.name as feed_name
FROM feed_follows
INNER JOIN users ON users.id = feed_follows.user_id
INNER JOIN feeds ON feeds.id = feed_follows.feed_id
WHERE feed_follows.user_id = $1;
