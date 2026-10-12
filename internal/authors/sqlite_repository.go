package authors

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"bakku.dev/bookist/internal/database"
)

const authorColumns = `id, name, created_at, updated_at`

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateAuthorRequest) (Author, error) {
	now := database.Now()

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO authors (name, created_at, updated_at)
		VALUES (?, ?, ?)
		RETURNING `+authorColumns+`
	`, input.Name, now, now)

	return ScanAuthor(row)
}

func (r *SQLiteRepository) Update(ctx context.Context, id int64, input UpdateAuthorRequest) (Author, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE authors
		SET name = CASE WHEN ? THEN ? ELSE name END,
			updated_at = ?
		WHERE id = ?
		RETURNING `+authorColumns+`
	`, input.Name.Present, input.Name.Value, database.Now(), id)

	author, err := ScanAuthor(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Author{}, ErrAuthorNotFound
	}
	return author, err
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Author, error) {
	return r.Search(ctx, "")
}

func (r *SQLiteRepository) Search(ctx context.Context, query string) ([]Author, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+authorColumns+`
		FROM authors
		WHERE instr(bookist_fold(name), ?) > 0
		ORDER BY updated_at DESC, id ASC
	`, database.Fold(query))
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	authors := make([]Author, 0)
	for rows.Next() {
		author, err := ScanAuthor(rows)
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

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Author, error) {
	author, err := ScanAuthor(r.db.QueryRowContext(ctx, `
		SELECT `+authorColumns+`
		FROM authors
		WHERE id = ?
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Author{}, ErrAuthorNotFound
	}
	return author, err
}

func (r *SQLiteRepository) GetByIDs(ctx context.Context, ids []int64) ([]Author, error) {
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
		SELECT `+authorColumns+`
		FROM authors
		WHERE id IN (%s)
	`, strings.Join(placeholders, ","))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	var authors []Author
	for rows.Next() {
		author, err := ScanAuthor(rows)
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

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM authors WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrAuthorNotFound
	}

	return nil
}

// RowScanner is satisfied by *sql.Row and *sql.Rows.
type RowScanner interface {
	Scan(dest ...any) error
}

// ScanAuthor scans the columns id, name, created_at, updated_at, in that order.
func ScanAuthor(scanner RowScanner) (Author, error) {
	var author Author
	err := scanner.Scan(&author.ID, &author.Name, database.Time(&author.CreatedAt), database.Time(&author.UpdatedAt))
	if err != nil {
		return Author{}, err
	}

	return author, nil
}
