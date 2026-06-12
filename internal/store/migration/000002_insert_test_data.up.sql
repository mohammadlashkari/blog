INSERT INTO posts (
    title,
    filename,
    slug,
    language,
    published_at
) VALUES 
    (
        'My first post',
        'my-first-post.md',
        'my-first-post',
        'en',
        datetime('now')
    );


INSERT INTO tags (name, slug) VALUES ('general', 'general');

INSERT INTO post_tags (post_id, tag_id) VALUES (
    (SELECT id FROM posts WHERE slug = 'my-first-post'),
    (SELECT id FROM tags WHERE slug = 'general')
);
