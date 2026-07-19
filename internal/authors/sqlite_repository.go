package authors

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateAuthorRequest) (Author, error) {
	now := time.Now().UTC()
	createdAt := now.Format(time.RFC3339)
	updatedAt := createdAt

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO authors (id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		RETURNING id, name, created_at, updated_at
	`, uuid.NewString(), input.Name, createdAt, updatedAt)

	return scanAuthor(row)
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Author, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, created_at, updated_at
		FROM authors
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		author, err := scanAuthor(rows)
		if err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}

func (r *SQLiteRepository) GetByIDs(ctx context.Context, ids []string) ([]Author, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id, name, created_at, updated_at
		FROM authors
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var authors []Author
	for rows.Next() {
		author, err := scanAuthor(rows)
		if err != nil {
			return nil, err
		}
		authors = append(authors, author)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return authors, nil
}

func (r *SQLiteRepository) ListByBookIDs(ctx context.Context, bookIDs []string) (map[string][]Author, error) {
	if len(bookIDs) == 0 {
		return map[string][]Author{}, nil
	}

	placeholders := make([]string, len(bookIDs))
	args := make([]any, len(bookIDs))
	for i, id := range bookIDs {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT ba.book_id, a.id, a.name, a.created_at, a.updated_at
		FROM book_authors ba
		JOIN authors a ON a.id = ba.author_id
		WHERE ba.book_id IN (%s)
		ORDER BY ba.updated_at DESC, ba.id ASC
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]Author)
	for rows.Next() {
		var bookID string
		var author Author
		var name, createdAt, updatedAt string

		if err := rows.Scan(&bookID, &author.ID, &name, &createdAt, &updatedAt); err != nil {
			return nil, err
		}

		author.Name = name

		author.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}

		author.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			return nil, fmt.Errorf("parse updated_at: %w", err)
		}

		result[bookID] = append(result[bookID], author)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type authorScanner interface {
	Scan(dest ...any) error
}

func scanAuthor(scanner authorScanner) (Author, error) {
	var author Author
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&author.ID, &author.Name, &createdAt, &updatedAt); err != nil {
		return Author{}, err
	}

	var err error

	author.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Author{}, fmt.Errorf("parse created_at: %w", err)
	}

	author.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Author{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return author, nil
}
