CREATE TABLE series_provider_link (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id INTEGER NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    provider_name TEXT NOT NULL,
    provider_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(folder_id, provider_name)
);

CREATE TABLE series_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    folder_id INTEGER NOT NULL UNIQUE REFERENCES folders(id) ON DELETE CASCADE,
    status TEXT,
    title TEXT,
    summary TEXT,
    publisher TEXT,
    reading_direction TEXT,
    age_rating INTEGER,
    language TEXT,
    total_book_count INTEGER,
    community_score REAL,
    release_year INTEGER,
    release_month INTEGER,
    release_day INTEGER,
    thumbnail_url TEXT,
    status_lock INTEGER NOT NULL DEFAULT 0,
    title_lock INTEGER NOT NULL DEFAULT 0,
    summary_lock INTEGER NOT NULL DEFAULT 0,
    publisher_lock INTEGER NOT NULL DEFAULT 0,
    reading_direction_lock INTEGER NOT NULL DEFAULT 0,
    age_rating_lock INTEGER NOT NULL DEFAULT 0,
    language_lock INTEGER NOT NULL DEFAULT 0,
    total_book_count_lock INTEGER NOT NULL DEFAULT 0,
    community_score_lock INTEGER NOT NULL DEFAULT 0,
    release_date_lock INTEGER NOT NULL DEFAULT 0,
    thumbnail_url_lock INTEGER NOT NULL DEFAULT 0,
    genres_lock INTEGER NOT NULL DEFAULT 0,
    tags_lock INTEGER NOT NULL DEFAULT 0,
    authors_lock INTEGER NOT NULL DEFAULT 0,
    links_lock INTEGER NOT NULL DEFAULT 0,
    titles_lock INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE series_metadata_genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    genre TEXT NOT NULL
);

CREATE TABLE series_metadata_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    tag TEXT NOT NULL
);

CREATE TABLE series_metadata_authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL
);

CREATE TABLE series_metadata_links (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    label TEXT NOT NULL,
    url TEXT NOT NULL
);

CREATE TABLE series_metadata_titles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    metadata_id INTEGER NOT NULL REFERENCES series_metadata(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    type TEXT NOT NULL,
    language TEXT
);

CREATE INDEX idx_series_metadata_folder ON series_metadata(folder_id);
CREATE INDEX idx_series_provider_link_folder ON series_provider_link(folder_id);
CREATE INDEX idx_series_metadata_genres_mid ON series_metadata_genres(metadata_id);
CREATE INDEX idx_series_metadata_tags_mid ON series_metadata_tags(metadata_id);
CREATE INDEX idx_series_metadata_authors_mid ON series_metadata_authors(metadata_id);
CREATE INDEX idx_series_metadata_links_mid ON series_metadata_links(metadata_id);
CREATE INDEX idx_series_metadata_titles_mid ON series_metadata_titles(metadata_id);
