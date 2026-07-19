-- +goose Up
CREATE TABLE books (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    isbn TEXT NULL,
    language TEXT NULL,
    publisher TEXT NULL,
    edition TEXT NULL,
    format TEXT NULL,
    purchased_at TEXT NULL,
    pages INTEGER NULL,
    notes TEXT NULL,
    published_year INTEGER NULL,
    published_month INTEGER NULL,
    published_day INTEGER NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE authors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE book_authors (
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    author_id TEXT NOT NULL REFERENCES authors(id) ON DELETE CASCADE,
    PRIMARY KEY (book_id, author_id)
);
CREATE TABLE lists (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE book_lists (
    list_id TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    PRIMARY KEY (list_id, book_id)
);
CREATE TABLE reads (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id) ON DELETE CASCADE,
    started_at TEXT NULL,
    finished_at TEXT NULL,
    rating REAL NULL CHECK (
        rating IS NULL OR
        (rating >= 1 AND rating <= 5 AND rating * 2 = CAST(rating * 2 AS INTEGER))
    ),
    notes TEXT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    CHECK (started_at IS NULL OR finished_at IS NULL OR finished_at >= started_at)
);
CREATE INDEX reads_book_id_idx ON reads(book_id);

-- +goose Down
DROP TABLE reads;
DROP TABLE book_lists;
DROP TABLE lists;
DROP TABLE book_authors;
DROP TABLE authors;
DROP TABLE books;
