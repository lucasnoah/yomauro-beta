CREATE TABLE sessions (
    id         text         PRIMARY KEY,
    user_id    integer      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz  NOT NULL DEFAULT NOW(),
    expires_at timestamptz  NOT NULL
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
