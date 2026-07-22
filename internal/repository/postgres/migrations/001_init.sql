CREATE TABLE IF NOT EXISTS posts (
    id                UUID PRIMARY KEY,
    title             TEXT        NOT NULL,
    content           TEXT        NOT NULL,
    author            TEXT        NOT NULL,
    comments_disabled BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ NOT NULL
);

CREATE TABLE IF NOT EXISTS comments (
    id         UUID PRIMARY KEY,
    post_id    UUID          NOT NULL REFERENCES posts (id) ON DELETE CASCADE,
    parent_id  UUID          REFERENCES comments (id) ON DELETE CASCADE,
    author     TEXT          NOT NULL,
    text       VARCHAR(2000) NOT NULL,
    created_at TIMESTAMPTZ   NOT NULL
);

-- Лента постов: новые первыми, keyset-пагинация по (created_at, id).
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts (created_at, id);

-- Корневые комментарии поста: частичный индекс только по parent_id IS NULL.
CREATE INDEX IF NOT EXISTS idx_comments_top_level
    ON comments (post_id, created_at, id) WHERE parent_id IS NULL;

-- Ответы на комментарий.
CREATE INDEX IF NOT EXISTS idx_comments_replies
    ON comments (parent_id, created_at, id) WHERE parent_id IS NOT NULL;
