-- name: CreatePost :one
INSERT INTO posts (id, created_at, updated_at, title, url, description, published_at, feed_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: GetPostsForUser :many
SELECT
  posts.id,
  posts.created_at,
  posts.updated_at,
  posts.title,
  posts.url,
  posts.description,
  posts.published_at,
  feeds.user_id as feed_user_id
FROM
  posts
  INNER JOIN feeds ON feeds.id = posts.feed_id
WHERE
  feeds.user_id = $1
ORDER BY
  posts.updated_at ASC
LIMIT
  $2;