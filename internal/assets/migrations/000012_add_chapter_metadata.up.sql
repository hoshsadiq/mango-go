CREATE TABLE chapter_metadata (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_id INTEGER NOT NULL UNIQUE REFERENCES chapters(id) ON DELETE CASCADE,
    title TEXT,
    number TEXT,
    sort_number REAL,
    volume TEXT,
    summary TEXT,
    notes TEXT,
    release_date TEXT,
    language TEXT,
    chapter_type TEXT DEFAULT 'Regular',
    age_rating TEXT,
    web TEXT,
    characters TEXT,
    teams TEXT,
    locations TEXT,
    scanlation_group TEXT,
    story_arc TEXT,
    story_arc_number TEXT,
    title_lock INTEGER NOT NULL DEFAULT 0,
    number_lock INTEGER NOT NULL DEFAULT 0,
    sort_number_lock INTEGER NOT NULL DEFAULT 0,
    volume_lock INTEGER NOT NULL DEFAULT 0,
    summary_lock INTEGER NOT NULL DEFAULT 0,
    notes_lock INTEGER NOT NULL DEFAULT 0,
    release_date_lock INTEGER NOT NULL DEFAULT 0,
    language_lock INTEGER NOT NULL DEFAULT 0,
    chapter_type_lock INTEGER NOT NULL DEFAULT 0,
    age_rating_lock INTEGER NOT NULL DEFAULT 0,
    web_lock INTEGER NOT NULL DEFAULT 0,
    characters_lock INTEGER NOT NULL DEFAULT 0,
    teams_lock INTEGER NOT NULL DEFAULT 0,
    locations_lock INTEGER NOT NULL DEFAULT 0,
    scanlation_group_lock INTEGER NOT NULL DEFAULT 0,
    story_arc_lock INTEGER NOT NULL DEFAULT 0,
    story_arc_number_lock INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE chapter_metadata_authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_metadata_id INTEGER NOT NULL REFERENCES chapter_metadata(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    UNIQUE(chapter_metadata_id, name, role)
);

CREATE TABLE chapter_metadata_genres (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_metadata_id INTEGER NOT NULL REFERENCES chapter_metadata(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    UNIQUE(chapter_metadata_id, name)
);

CREATE TABLE chapter_metadata_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    chapter_metadata_id INTEGER NOT NULL REFERENCES chapter_metadata(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    UNIQUE(chapter_metadata_id, name)
);

CREATE INDEX idx_chapter_metadata_chapter ON chapter_metadata(chapter_id);
CREATE INDEX idx_chapter_metadata_authors_metadata ON chapter_metadata_authors(chapter_metadata_id);
CREATE INDEX idx_chapter_metadata_genres_metadata ON chapter_metadata_genres(chapter_metadata_id);
CREATE INDEX idx_chapter_metadata_tags_metadata ON chapter_metadata_tags(chapter_metadata_id);
