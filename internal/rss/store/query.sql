-- name: CheckFeedExists :one
SELECT EXISTS(
    SELECT 1 FROM rss_items WHERE feed_url = ?
);

-- name: GetItemsByStatus :many
SELECT * FROM rss_items 
WHERE status = ? 
ORDER BY published_at DESC NULLS LAST;

-- name: CreateRssItem :exec
INSERT INTO rss_items (
    feed_name, feed_url, guid, link, title, description, published_at, status, categories
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(feed_url, guid) DO NOTHING;

-- name: UpdateItemStatus :exec
UPDATE rss_items
SET status = ?, seen_at = CURRENT_TIMESTAMP
WHERE id = ?;

-- name: ToggleSavedStatus :exec
UPDATE rss_items
SET is_saved = ?
WHERE id = ?;

-- name: DeleteByFeedUR :exec
DELETE FROM rss_items
WHERE feed_url = ?
