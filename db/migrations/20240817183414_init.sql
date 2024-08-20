-- migrate:up
CREATE TABLE feeds
(
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    url             TEXT      NOT NULL UNIQUE,
    title           TEXT,
    last_fetched_at TIMESTAMP,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feeds_url ON feeds (url);
CREATE INDEX idx_feeds_last_fetched_at ON feeds (last_fetched_at);

-- migrate:down

