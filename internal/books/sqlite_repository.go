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
		SELECT id, title, isbn, language, publisher, edition, format, 
		    purchased_at, pages, notes, published_year, published_month, 
		    published_day, created_at, updated_at
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

func (r *SQLiteRepository) ListByListID(ctx context.Context, listID string) ([]Book, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.isbn, b.language, b.publisher, b.edition, b.format,
		       b.purchased_at, b.pages, b.notes, b.published_year, b.published_month,
		       b.published_day, b.created_at, b.updated_at
		FROM books b
		JOIN book_lists bl ON bl.book_id = b.id
		WHERE bl.list_id = ?
		ORDER BY b.title ASC
	`, listID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	bookList := make([]Book, 0)

	for rows.Next() {
		book, err := scanBook(rows)
		if err != nil {
			return nil, err
		}

		bookList = append(bookList, book)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bookList, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateBookRequest) (Book, error) {
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	updatedAt := createdAt

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Book{}, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	isbn := sql.NullString{}
	if input.ISBN != nil {
		isbn = sql.NullString{String: *input.ISBN, Valid: true}
	}

	language := sql.NullString{}
	if input.Language != nil {
		language = sql.NullString{String: *input.Language, Valid: true}
	}

	publisher := sql.NullString{}
	if input.Publisher != nil {
		publisher = sql.NullString{String: *input.Publisher, Valid: true}
	}

	edition := sql.NullString{}
	if input.Edition != nil {
		edition = sql.NullString{String: *input.Edition, Valid: true}
	}

	var format sql.NullString
	if input.Format != nil {
		format = sql.NullString{String: string(*input.Format), Valid: true}
	}

	purchasedAt := sql.NullString{}
	if input.PurchasedAt != nil {
		purchasedAt = sql.NullString{String: *input.PurchasedAt, Valid: true}
	}

	notes := sql.NullString{}
	if input.Notes != nil {
		notes = sql.NullString{String: *input.Notes, Valid: true}
	}

	pages := sql.NullInt64{}
	if input.Pages != nil {
		pages = sql.NullInt64{Int64: int64(*input.Pages), Valid: true}
	}

	publishedYear := sql.NullInt64{}
	if input.PublishedYear != nil {
		publishedYear = sql.NullInt64{Int64: int64(*input.PublishedYear), Valid: true}
	}

	publishedMonth := sql.NullInt64{}
	if input.PublishedMonth != nil {
		publishedMonth = sql.NullInt64{Int64: int64(*input.PublishedMonth), Valid: true}
	}

	publishedDay := sql.NullInt64{}
	if input.PublishedDay != nil {
		publishedDay = sql.NullInt64{Int64: int64(*input.PublishedDay), Valid: true}
	}

	bookID := uuid.NewString()

	row := tx.QueryRowContext(ctx, `
		INSERT INTO books (id, title, isbn, language, publisher, edition, format, 
		                   purchased_at, pages, notes, published_year, published_month, 
		                   published_day, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, title, isbn, language, publisher, edition, format, purchased_at, 
			pages, notes, published_year, published_month, published_day, 
			created_at, updated_at
	`, bookID, input.Title, isbn, language, publisher, edition, format, purchasedAt,
		pages, notes, publishedYear, publishedMonth, publishedDay, createdAt, updatedAt)

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
	var language sql.NullString
	var publisher sql.NullString
	var edition sql.NullString
	var format sql.NullString
	var purchasedAt sql.NullString
	var pages sql.NullInt64
	var notes sql.NullString
	var publishedYear sql.NullInt64
	var publishedMonth sql.NullInt64
	var publishedDay sql.NullInt64
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&book.ID, &book.Title, &isbn, &language, &publisher, &edition, &format, &purchasedAt, &pages, &notes, &publishedYear, &publishedMonth, &publishedDay, &createdAt, &updatedAt); err != nil {
		return Book{}, err
	}

	if isbn.Valid {
		book.ISBN = &isbn.String
	}

	if language.Valid {
		book.Language = &language.String
	}

	if publisher.Valid {
		book.Publisher = &publisher.String
	}

	if edition.Valid {
		book.Edition = &edition.String
	}

	if format.Valid {
		f := Format(format.String)
		book.Format = &f
	}

	if purchasedAt.Valid {
		book.PurchasedAt = &purchasedAt.String
	}

	if pages.Valid {
		p := int(pages.Int64)
		book.Pages = &p
	}

	if notes.Valid {
		book.Notes = &notes.String
	}

	if publishedYear.Valid {
		y := int(publishedYear.Int64)
		book.PublishedYear = &y
	}

	if publishedMonth.Valid {
		m := int(publishedMonth.Int64)
		book.PublishedMonth = &m
	}

	if publishedDay.Valid {
		d := int(publishedDay.Int64)
		book.PublishedDay = &d
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
