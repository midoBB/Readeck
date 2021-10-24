CREATE TABLE IF NOT EXISTS bookmark_collection (
    id          SERIAL      PRIMARY KEY,
    uid         varchar(32) UNIQUE NOT NULL,
    user_id     integer     NOT NULL,
    created     timestamptz NOT NULL,
    updated     timestamptz NOT NULL,
    name        text        NOT NULL,
    is_pinned   boolean     NOT NULL DEFAULT false,
    filters     json        NOT NULL DEFAULT '{}',

    CONSTRAINT fk_bookmark_collection_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);
