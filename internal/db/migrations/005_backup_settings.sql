CREATE TABLE IF NOT EXISTS backup_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    enabled INTEGER NOT NULL DEFAULT 0,
    schedule_mode TEXT NOT NULL DEFAULT 'interval',
    interval_hours INTEGER NOT NULL DEFAULT 24,
    daily_hour INTEGER NOT NULL DEFAULT 3,
    retention_count INTEGER NOT NULL DEFAULT 3,
    last_run_at TEXT
);

INSERT OR IGNORE INTO backup_settings (id) VALUES (1);
