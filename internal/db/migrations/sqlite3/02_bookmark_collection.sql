CREATE TABLE IF NOT EXISTS bookmark_collection (
    id          integer  PRIMARY KEY AUTOINCREMENT,
    uid         text     UNIQUE NOT NULL,
    user_id     integer  NOT NULL,
    created     datetime NOT NULL,
    updated     datetime NOT NULL,
    name        text     NOT NULL,
    is_pinned   integer  NOT NULL DEFAULT 0,
    filters     json     NOT NULL DEFAULT "{}",

    CONSTRAINT fk_bookmark_collection_user FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);
