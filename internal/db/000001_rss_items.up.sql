CREATE TABLE IF NOT EXISTS rss_items (
    id INTEGER PRIMARY KEY,
    feed_name TEXT NOT NULL,
    feed_url TEXT NOT NULL,
    guid TEXT NOT NULL, -- fallback to link if feed lacks guid
    link TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    published_at DATETIME,
    fetched_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'archived')),
    is_saved BOOLEAN NOT NULL DEFAULT FALSE,
    seen_at DATETIME,

    UNIQUE(feed_url, guid)
);

CREATE INDEX idx_status ON rss_items(status);
CREATE INDEX idx_feed_url ON rss_items(feed_url);
CREATE INDEX idx_published_at ON rss_items(published_at DESC);
