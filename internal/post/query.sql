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

