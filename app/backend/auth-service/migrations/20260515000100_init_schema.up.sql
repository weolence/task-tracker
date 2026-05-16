CREATE TABLE users (
    id       SERIAL PRIMARY KEY,
    email    TEXT UNIQUE NOT NULL,
    name     TEXT NOT NULL,
    surname  TEXT NOT NULL,
    password TEXT NOT NULL,
    role     TEXT NOT NULL DEFAULT 'user',
    CONSTRAINT users_role_check CHECK (role IN ('user', 'admin'))
);
