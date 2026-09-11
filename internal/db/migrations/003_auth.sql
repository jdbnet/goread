CREATE TABLE IF NOT EXISTS auth_credentials (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    username TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT ''
);

INSERT OR IGNORE INTO auth_credentials (id) VALUES (1);

CREATE TABLE IF NOT EXISTS sessions (
    token TEXT PRIMARY KEY,
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);
