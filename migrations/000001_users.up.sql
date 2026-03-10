CREATE TABLE users (
    id            serial       PRIMARY KEY,
    email         text         NOT NULL UNIQUE,
    password_hash text         NOT NULL,
    display_name  text         NOT NULL,
    role          text         NOT NULL DEFAULT 'owner' CHECK (role IN ('owner')),
    active        boolean      NOT NULL DEFAULT true,
    created_at    timestamptz  NOT NULL DEFAULT NOW(),
    updated_at    timestamptz  NOT NULL DEFAULT NOW()
);
