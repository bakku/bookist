package books

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bakku.dev/bookist/internal/authors"
	"github.com/google/uuid"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Book, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, isbn, created_at, updated_at
		FROM books
		ORDER BY id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}
		books = append(books, book)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateBookRequest) (Book, error) {
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	updatedAt := createdAt

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Book{}, err
	}
	defer tx.Rollback()

	isbn := sql.NullString{}
	if input.ISBN != nil {
		isbn = sql.NullString{String: *input.ISBN, Valid: true}
	}

	bookID := uuid.NewString()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO books (id, title, isbn, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, title, isbn, created_at, updated_at
	`, bookID, input.Title, isbn, createdAt, updatedAt)

	book, err := scanBook(row)
	if err != nil {
		return Book{}, err
	}

	for _, authorID := range input.AuthorIDs {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO book_authors (book_id, author_id)
			VALUES (?, ?)
		`, book.ID, authorID)
		if err != nil {
			return Book{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Book{}, err
	}

	book.Authors = []authors.Author{}

	return book, nil
}

type bookScanner interface {
	Scan(dest ...any) error
}

func scanBook(scanner bookScanner) (Book, error) {
	var book Book
	var isbn sql.NullString
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&book.ID, &book.Title, &isbn, &createdAt, &updatedAt); err != nil {
		return Book{}, err
	}

	if isbn.Valid {
		book.ISBN = &isbn.String
	}

	var err error
	book.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Book{}, fmt.Errorf("parse created_at: %w", err)
	}
	book.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Book{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return book, nil
}
