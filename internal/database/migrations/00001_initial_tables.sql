-- +goose Up
CREATE TABLE books (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL COLLATE NOCASE CHECK (trim(title) <> ''),
    isbn TEXT NULL,
    language TEXT NULL,
    publisher TEXT NULL,
    edition TEXT NULL,
    format TEXT NULL CHECK (format IS NULL OR format IN ('hardback', 'paperback', 'epub')),
    purchased_at TEXT NULL CHECK (
        purchased_at IS NULL OR (
            length(purchased_at) = 10 AND
            purchased_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
            CAST(substr(purchased_at, 1, 4) AS INTEGER) >= 1 AND
            CAST(substr(purchased_at, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
            CAST(substr(purchased_at, 9, 2) AS INTEGER) BETWEEN 1 AND CASE CAST(substr(purchased_at, 6, 2) AS INTEGER)
                WHEN 2 THEN 28 + (CAST(substr(purchased_at, 1, 4) AS INTEGER) % 4 = 0 AND
                    (CAST(substr(purchased_at, 1, 4) AS INTEGER) % 100 <> 0 OR CAST(substr(purchased_at, 1, 4) AS INTEGER) % 400 = 0))
                WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31
            END
        )
    ),
    pages INTEGER NULL CHECK (pages IS NULL OR (typeof(pages) = 'integer' AND pages >= 1)),
    notes TEXT NULL,
    summary TEXT NULL,
    series_name TEXT NULL,
    series_position REAL NULL CHECK (series_position IS NULL OR (typeof(series_position) IN ('integer', 'real') AND series_position > 0)),
    location TEXT NULL,
    condition TEXT NULL CHECK (condition IS NULL OR condition IN ('new', 'very_good', 'good', 'acceptable', 'poor')),
    acquisition_source TEXT NULL,
    published_year INTEGER NULL CHECK (published_year IS NULL OR (typeof(published_year) = 'integer' AND published_year >= 1)),
    published_month INTEGER NULL CHECK (
        published_month IS NULL OR (typeof(published_month) = 'integer' AND published_year IS NOT NULL AND published_month BETWEEN 1 AND 12)
    ),
    published_day INTEGER NULL CHECK (
        published_day IS NULL OR (
            typeof(published_day) = 'integer' AND published_year IS NOT NULL AND published_month IS NOT NULL AND
            published_day BETWEEN 1 AND CASE published_month
                WHEN 2 THEN 28 + (published_year % 4 = 0 AND (published_year % 100 <> 0 OR published_year % 400 = 0))
                WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31
            END
        )
    ),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE authors (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE CHECK (trim(name) <> ''),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE book_authors (
    id INTEGER PRIMARY KEY,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author_id INTEGER NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (book_id, author_id)
);
CREATE TABLE lists (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL COLLATE NOCASE UNIQUE CHECK (trim(name) <> ''),
    description TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE book_lists (
    id INTEGER PRIMARY KEY,
    list_id INTEGER NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE (list_id, book_id)
);
CREATE TABLE reads (
    id INTEGER PRIMARY KEY,
    book_id INTEGER NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    started_at TEXT NULL CHECK (
        started_at IS NULL OR (
            length(started_at) = 10 AND started_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
            CAST(substr(started_at, 1, 4) AS INTEGER) >= 1 AND
            CAST(substr(started_at, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
            CAST(substr(started_at, 9, 2) AS INTEGER) BETWEEN 1 AND CASE CAST(substr(started_at, 6, 2) AS INTEGER)
                WHEN 2 THEN 28 + (CAST(substr(started_at, 1, 4) AS INTEGER) % 4 = 0 AND
                    (CAST(substr(started_at, 1, 4) AS INTEGER) % 100 <> 0 OR CAST(substr(started_at, 1, 4) AS INTEGER) % 400 = 0))
                WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31
            END
        )
    ),
    finished_at TEXT NULL CHECK (
        finished_at IS NULL OR (
            length(finished_at) = 10 AND finished_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
            CAST(substr(finished_at, 1, 4) AS INTEGER) >= 1 AND
            CAST(substr(finished_at, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
            CAST(substr(finished_at, 9, 2) AS INTEGER) BETWEEN 1 AND CASE CAST(substr(finished_at, 6, 2) AS INTEGER)
                WHEN 2 THEN 28 + (CAST(substr(finished_at, 1, 4) AS INTEGER) % 4 = 0 AND
                    (CAST(substr(finished_at, 1, 4) AS INTEGER) % 100 <> 0 OR CAST(substr(finished_at, 1, 4) AS INTEGER) % 400 = 0))
                WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31
            END
        )
    ),
    abandoned_at TEXT NULL CHECK (
        abandoned_at IS NULL OR (
            length(abandoned_at) = 10 AND abandoned_at GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]' AND
            CAST(substr(abandoned_at, 1, 4) AS INTEGER) >= 1 AND
            CAST(substr(abandoned_at, 6, 2) AS INTEGER) BETWEEN 1 AND 12 AND
            CAST(substr(abandoned_at, 9, 2) AS INTEGER) BETWEEN 1 AND CASE CAST(substr(abandoned_at, 6, 2) AS INTEGER)
                WHEN 2 THEN 28 + (CAST(substr(abandoned_at, 1, 4) AS INTEGER) % 4 = 0 AND
                    (CAST(substr(abandoned_at, 1, 4) AS INTEGER) % 100 <> 0 OR CAST(substr(abandoned_at, 1, 4) AS INTEGER) % 400 = 0))
                WHEN 4 THEN 30 WHEN 6 THEN 30 WHEN 9 THEN 30 WHEN 11 THEN 30 ELSE 31
            END
        )
    ),
    rating REAL NULL CHECK (
        rating IS NULL OR
        (rating >= 1 AND rating <= 5 AND rating * 2 = CAST(rating * 2 AS INTEGER))
    ),
    notes TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (finished_at IS NULL OR abandoned_at IS NULL),
    CHECK (started_at IS NULL OR finished_at IS NULL OR finished_at >= started_at),
    CHECK (started_at IS NULL OR abandoned_at IS NULL OR abandoned_at >= started_at)
);
CREATE INDEX reads_book_id_idx ON reads(book_id);

-- +goose Down
DROP TABLE reads;
DROP TABLE book_lists;
DROP TABLE lists;
DROP TABLE book_authors;
DROP TABLE authors;
DROP TABLE books;
