-- name: GetFeed :one
SELECT * FROM feeds
WHERE id = ? LIMIT 1;

-- name: ListFeeds :many
SELECT * FROM feeds
ORDER BY created_at;

-- name: CreateFeed :one
INSERT INTO feeds (url, title)
VALUES (?, ?)
    RETURNING *;

-- name: UpdateFeedLastFetched :exec
UPDATE feeds
SET last_fetched_at = ?
WHERE id = ?;

-- name: DeleteFeed :exec
DELETE FROM feeds
WHERE id = ?;