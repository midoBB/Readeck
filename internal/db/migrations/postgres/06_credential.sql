CREATE TABLE IF NOT EXISTS "credential" (
    id         SERIAL       PRIMARY KEY,
    uid        varchar(32)  UNIQUE NOT NULL,
    user_id    integer      NOT NULL,
    created    timestamptz  NOT NULL,
    is_enabled boolean      NOT NULL DEFAULT true,
    name       varchar(128) NOT NULL,
    password   varchar(256) NOT NULL,
    roles      jsonb        NOT NULL DEFAULT '[]',

    CONSTRAINT fk_app_password_user FOREIGN KEY (user_id) REFERENCES "user"(id) ON DELETE CASCADE
);
