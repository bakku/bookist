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

-- +goose Down
DROP TABLE book_authors;
DROP TABLE authors;
DROP TABLE books;
