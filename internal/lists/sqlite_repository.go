package lists

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
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
		INSERT INTO lists (name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		RETURNING id, name, description, created_at, updated_at
	`, input.Name, description, createdAt, updatedAt)

	list, err := scanList(row)
	if err != nil && isUniqueViolation(err) {
		return List{}, ErrNameConflict
	}
	return list, err
}

func (r *SQLiteRepository) NameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM lists WHERE name = ? COLLATE NOCASE)`, name).Scan(&exists)
	return exists, err
}

func (r *SQLiteRepository) List(ctx context.Context) ([]List, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM lists
		ORDER BY updated_at DESC, id ASC
	`)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

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

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (List, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, created_at, updated_at
		FROM lists
		WHERE id = ?
	`, id)

	list, err := scanList(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return List{}, ErrListNotFound
		}
		return List{}, err
	}

	return list, nil
}

func (r *SQLiteRepository) AddBookToList(ctx context.Context, listID, bookID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO book_lists (list_id, book_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, listID, bookID, now, now)

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
