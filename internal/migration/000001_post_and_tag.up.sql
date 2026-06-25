CREATE TABLE IF NOT EXISTS posts (
    id            INTEGER PRIMARY KEY,
    filename      TEXT NOT NULL UNIQUE,
    title         TEXT NOT NULL,
    slug          TEXT NOT NULL UNIQUE,
    cover_image   TEXT,
    language      TEXT NOT NULL CHECK (language IN ('en', 'fa')),
    version       INTEGER NOT NULL DEFAULT 1,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at  DATETIME,
    updated_at    DATETIME
);


CREATE TABLE IF NOT EXISTS tags (
    id    INTEGER PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE
);


CREATE TABLE IF NOT EXISTS post_tags (
    post_id INTEGER NOT NULL,
    tag_id  INTEGER NOT NULL,

    PRIMARY KEY (post_id, tag_id),
    FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags(id) ON DELETE CASCADE
);
