DELETE FROM post_tags
WHERE post_id = (SELECT id FROM posts WHERE slug = 'my-first-post')
  AND tag_id = (SELECT id FROM tags WHERE name = 'general');

DELETE FROM tags WHERE name = 'general';
DELETE FROM posts WHERE slug = 'my-first-post';
