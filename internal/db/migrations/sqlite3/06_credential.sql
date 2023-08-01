CREATE TABLE IF NOT EXISTS credential (
    id          integer  PRIMARY KEY AUTOINCREMENT,
    uid         text     UNIQUE NOT NULL,
    user_id     integer  NOT NULL,
    created     datetime NOT NULL,
    is_enabled  integer  NOT NULL DEFAULT 1,
    name        text     NOT NULL,
    password    text     NOT NULL,
    roles       json     NOT NULL DEFAULT "",

    CONSTRAINT fk_app_password_user FOREIGN KEY (user_id) REFERENCES user(id) ON DELETE CASCADE
);
