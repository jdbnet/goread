-- +migrate Up
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = WAL;

CREATE TABLE IF NOT EXISTS books (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    title TEXT NOT NULL DEFAULT '',
    author TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    publisher TEXT NOT NULL DEFAULT '',
    pub_date TEXT NOT NULL DEFAULT '',
    isbn TEXT NOT NULL DEFAULT '',
    language TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL,
    file_hash TEXT NOT NULL UNIQUE,
    cover_path TEXT NOT NULL DEFAULT '',
    total_chapters INTEGER NOT NULL DEFAULT 0,
    file_missing INTEGER NOT NULL DEFAULT 0,
    user_set_fields TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_books_author ON books(author);
CREATE INDEX IF NOT EXISTS idx_books_title ON books(title COLLATE NOCASE);
CREATE INDEX IF NOT EXISTS idx_books_created_at ON books(created_at);

CREATE TABLE IF NOT EXISTS series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS book_series (
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    series_id INTEGER NOT NULL REFERENCES series(id) ON DELETE CASCADE,
    sequence_number REAL,
    PRIMARY KEY (book_id, series_id)
);

CREATE INDEX IF NOT EXISTS idx_book_series_series ON book_series(series_id, sequence_number);

CREATE TABLE IF NOT EXISTS reading_progress (
    book_id INTEGER PRIMARY KEY REFERENCES books(id) ON DELETE CASCADE,
    current_cfi TEXT NOT NULL DEFAULT '',
    percent_completed REAL NOT NULL DEFAULT 0,
    total_time_read_seconds INTEGER NOT NULL DEFAULT 0,
    last_read_at TEXT,
    completed_at TEXT
);

CREATE TABLE IF NOT EXISTS reading_stats (
    day TEXT PRIMARY KEY,
    time_read_seconds INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS reader_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    font_size INTEGER NOT NULL DEFAULT 18,
    line_height REAL NOT NULL DEFAULT 1.6,
    theme TEXT NOT NULL DEFAULT 'light'
);

INSERT OR IGNORE INTO reader_settings (id) VALUES (1);
