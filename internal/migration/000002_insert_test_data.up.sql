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
        datetime('now', '-2 years')
    ),
    (
        'Understanding SQLC with Go',
        'sqlc-with-go.md',
        'sqlc-with-go',
        'en',
        datetime('now', '-18 months')
    ),
    (
        'Why I switched to Linux',
        'switch-to-linux.md',
        'switch-to-linux',
        'en',
        NULL
    ),
    (
        'Building a simple blog engine',
        'simple-blog-engine.md',
        'simple-blog-engine',
        'en',
        datetime('now', '-8 months')
    ),
    (
        'Working with Redis in Go',
        'redis-go.md',
        'redis-go',
        'en',
        datetime('now', '-4 months')
    ),
    (
        'اولین نوشته من',
        'first-farsi-post.md',
        'first-farsi-post',
        'fa',
        NULL
    ),
    (
        'چرا به لینوکس مهاجرت کردم',
        'why-linux-fa.md',
        'why-linux-fa',
        'fa',
        datetime('now', '-1 year', '-6 months')
    ),
    (
        'آموزش ساده SQLC در Go',
        'sqlc-go-fa.md',
        'sqlc-go-fa',
        'fa',
        datetime('now', '-10 months')
    ),
    (
        'ساخت یک وبلاگ ساده',
        'simple-blog-fa.md',
        'simple-blog-fa',
        'fa',
        datetime('now', '-6 months')
    ),
    (
        'یادداشت‌های کوتاه برنامه‌نویسی',
        'dev-notes-fa.md',
        'dev-notes-fa',
        'fa',
        datetime('now', '-2 months')
    );


INSERT INTO tags (name) VALUES
('general'),
('go'),
('database'),
('sqlc'),
('linux'),
('life'),
('blog'),
('farsi'),
('programming');

INSERT INTO post_tags (post_id, tag_id) VALUES (
    (SELECT id FROM posts WHERE slug = 'my-first-post'),
    (SELECT id FROM tags WHERE name = 'general')
);

INSERT INTO post_tags (post_id, tag_id) VALUES
((SELECT id FROM posts WHERE slug = 'sqlc-with-go'),
 (SELECT id FROM tags WHERE name = 'go')),
((SELECT id FROM posts WHERE slug = 'sqlc-with-go'),
 (SELECT id FROM tags WHERE name = 'sqlc')),
((SELECT id FROM posts WHERE slug = 'sqlc-with-go'),
 (SELECT id FROM tags WHERE name = 'database'));

INSERT INTO post_tags (post_id, tag_id) VALUES
((SELECT id FROM posts WHERE slug = 'switch-to-linux'),
 (SELECT id FROM tags WHERE name = 'linux')),
((SELECT id FROM posts WHERE slug = 'switch-to-linux'),
 (SELECT id FROM tags WHERE name = 'life'));

INSERT INTO post_tags (post_id, tag_id) VALUES
((SELECT id FROM posts WHERE slug = 'simple-blog-engine'),
 (SELECT id FROM tags WHERE name = 'go')),
((SELECT id FROM posts WHERE slug = 'simple-blog-engine'),
 (SELECT id FROM tags WHERE name = 'blog')),
((SELECT id FROM posts WHERE slug = 'simple-blog-engine'),
 (SELECT id FROM tags WHERE name = 'programming'));

INSERT INTO post_tags (post_id, tag_id) VALUES
((SELECT id FROM posts WHERE slug = 'first-farsi-post'),
 (SELECT id FROM tags WHERE name = 'farsi')),
((SELECT id FROM posts WHERE slug = 'why-linux-fa'),
 (SELECT id FROM tags WHERE name = 'linux')),
((SELECT id FROM posts WHERE slug = 'why-linux-fa'),
 (SELECT id FROM tags WHERE name = 'life')),
((SELECT id FROM posts WHERE slug = 'sqlc-go-fa'),
 (SELECT id FROM tags WHERE name = 'go')),
((SELECT id FROM posts WHERE slug = 'sqlc-go-fa'),
 (SELECT id FROM tags WHERE name = 'sqlc')),
((SELECT id FROM posts WHERE slug = 'simple-blog-fa'),
 (SELECT id FROM tags WHERE name = 'blog')),
((SELECT id FROM posts WHERE slug = 'simple-blog-fa'),
 (SELECT id FROM tags WHERE name = 'farsi'));
