-- name: ListPosts :many
SELECT *
FROM posts
WHERE (
    published_at IS NOT NULL 
    OR CAST(@include_all AS BOOLEAN) = TRUE
)
ORDER BY published_at DESC;

-- name: ListPostsByTag :many
SELECT DISTINCT posts.*
FROM posts
JOIN post_tags post_tags ON posts.id = post_tags.post_id
JOIN tags ON post_tags.tag_id = tags.id
WHERE tags.name IN (sqlc.slice(tag_names))
AND (
    posts.published_at IS NOT NULL 
    OR CAST(@include_all AS BOOLEAN) = TRUE
)
ORDER BY posts.published_at DESC;

-- name: ListTags :many
SELECT * FROM tags;

-- name: PostByID :one
SELECT * FROM posts 
WHERE id = ?
AND (
    published_at IS NOT NULL 
    OR CAST(@include_all AS BOOLEAN) = TRUE
);


-- name: PostBySlug :one
SELECT * FROM posts
WHERE slug = ?
AND (
    published_at IS NOT NULL 
    OR CAST(@include_all AS BOOLEAN) = TRUE
);

-- name: TagsByPostID :many
SELECT tags.*
FROM tags
JOIN post_tags ON post_tags.tag_id = tags.id
WHERE post_tags.post_id = ?
ORDER BY tags.name;

-- name: UpsertPost :one
INSERT INTO posts (
  filename, title, slug, cover_image, language, published_at, content_hash, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(filename) DO UPDATE SET
  title = excluded.title,
  slug = excluded.slug,
  cover_image = excluded.cover_image,
  language = excluded.language,
  published_at = excluded.published_at,
  content_hash = excluded.content_hash,
  version = posts.version + 1,
  updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeletePostByFilename :exec
DELETE FROM posts WHERE filename = ?;

-- name: UpsertTag :one
INSERT INTO tags (name)
VALUES (?)
ON CONFLICT(name) DO UPDATE SET name = excluded.name
RETURNING *;

-- name: DeletePostTags :exec
DELETE FROM post_tags WHERE post_id = ?;

-- name: AddPostTag :exec
INSERT OR IGNORE INTO post_tags (post_id, tag_id)
VALUES (?, ?);
