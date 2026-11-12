CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    account       VARCHAR(64)  NOT NULL,
    nickname      VARCHAR(64)  NOT NULL,
    password_hash TEXT         NOT NULL,
    avatar        TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX users_account_key ON users (account);