package lists

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"bakku.dev/bookist/internal/books"
	"github.com/google/uuid"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateListRequest) (List, error) {
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	updatedAt := createdAt

	var description sql.NullString
	if input.Description != nil {
		description = sql.NullString{String: *input.Description, Valid: true}
	}

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO lists (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, name, description, created_at, updated_at
	`, uuid.NewString(), input.Name, description, createdAt, updatedAt)

	return scanList(row)
}

func (r *SQLiteRepository) List(ctx context.Context) ([]List, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM lists
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	lists := make([]List, 0)
	for rows.Next() {
		list, err := scanList(rows)
		if err != nil {
			return nil, err
		}
		lists = append(lists, list)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return lists, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id string) (List, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM lists
		WHERE id = ?
	`, id)

	list, err := scanList(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return List{}, ErrListNotFound
		}
		return List{}, err
	}

	return list, nil
}

func (r *SQLiteRepository) AddBookToList(ctx context.Context, listID, bookID string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO book_lists (list_id, book_id)
		VALUES (?, ?)
	`, listID, bookID)
	if err != nil {
		if isFKViolation(err) {
			list, listErr := r.GetByID(ctx, listID)
			if listErr != nil {
				return ErrListNotFound
			}
			_ = list
			return ErrBookNotFound
		}
		if isUniqueViolation(err) {
			return ErrBookAlreadyInList
		}
		return err
	}

	return nil
}

func (r *SQLiteRepository) ListBooks(ctx context.Context, listID string) ([]books.Book, error) {
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
	defer rows.Close()

	bookList := make([]books.Book, 0)
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

type listScanner interface {
	Scan(dest ...any) error
}

func scanList(scanner listScanner) (List, error) {
	var list List
	var description sql.NullString
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&list.ID, &list.Name, &description, &createdAt, &updatedAt); err != nil {
		return List{}, err
	}

	if description.Valid {
		list.Description = &description.String
	}

	var err error
	list.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return List{}, fmt.Errorf("parse created_at: %w", err)
	}

	list.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return List{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return list, nil
}

type bookScanner interface {
	Scan(dest ...any) error
}

func scanBook(scanner bookScanner) (books.Book, error) {
	var book books.Book
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
		return books.Book{}, err
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
		f := books.Format(format.String)
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
		return books.Book{}, fmt.Errorf("parse created_at: %w", err)
	}
	book.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return books.Book{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return book, nil
}

func isFKViolation(err error) bool {
	return err != nil && (contains(err.Error(), "FOREIGN KEY constraint failed"))
}

func isUniqueViolation(err error) bool {
	return err != nil && (contains(err.Error(), "UNIQUE constraint failed"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (s[0] == substr[0] && containsAt(s, substr, 0) || contains(s[1:], substr)))
}

func containsAt(s, substr string, start int) bool {
	if start+len(substr) > len(s) {
		return false
	}
	for i := 0; i < len(substr); i++ {
		if s[start+i] != substr[i] {
			return false
		}
	}
	return true
}
