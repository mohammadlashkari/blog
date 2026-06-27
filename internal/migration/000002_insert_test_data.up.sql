INSERT INTO posts (
    title,
    slug,
    language,
    published_at
) VALUES 
    (
        'My first post',
        'my-first-post',
        'en',
        datetime('now', '-2 years')
    ),
    (
        'GIT DIFF',
        'git-diff',
        'en',
        datetime('now', '-18 months')
    ),
    (
        'Why I switched to Linux',
        'switch-to-linux',
        'en',
        NULL
    ),
    (
        'Building a simple blog engine',
        'simple-blog-engine',
        'en',
        datetime('now', '-8 months')
    ),
    (
        'Working with Redis in Go',
        'redis-go',
        'en',
        datetime('now', '-4 months')
    ),
    (
        'اولین نوشته من',
        'first-farsi-post',
        'fa',
        datetime('now', '-20 months')
    ),
    (
        'چرا به لینوکس مهاجرت کردم',
        'why-linux-fa',
        'fa',
        NULL
    ),
    (
        'آموزش ساده SQLC در Go',
        'sqlc-go-fa',
        'fa',
        datetime('now', '-10 months')
    ),
    (
        'ساخت یک وبلاگ ساده',
        'simple-blog-fa',
        'fa',
        datetime('now', '-6 months')
    ),
    (
        'یادداشت‌های کوتاه برنامه‌نویسی',
        'dev-notes-fa',
        'fa',
        datetime('now', '-2 months')
    );


INSERT INTO tags (name) VALUES
('general'),
('go'),
('git'),
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

INSERT INTO post_tags (post_id, tag_id) VALUES(
 (SELECT id FROM posts WHERE slug = 'git-diff'),
 (SELECT id FROM tags WHERE name = 'git')
);

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
