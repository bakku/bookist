package reads

import (
	"context"
	"database/sql"
	"errors"
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

func (r *SQLiteRepository) Create(ctx context.Context, bookID string, input CreateReadRequest) (Read, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO reads (
			id, book_id, started_at, finished_at, rating, notes, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, book_id, started_at, finished_at, rating, notes, created_at, updated_at
	`, uuid.NewString(), bookID, nullString(input.StartedAt), nullString(input.FinishedAt),
		nullFloat64(input.Rating), nullString(input.Notes), now, now)

	read, err := scanRead(row)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return Read{}, ErrBookNotFound
		}
		return Read{}, err
	}

	return read, nil
}

func (r *SQLiteRepository) ListByBookID(ctx context.Context, bookID string) ([]Read, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id = ?`, bookID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookNotFound
		}
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, book_id, started_at, finished_at, rating, notes, created_at, updated_at
		FROM reads
		WHERE book_id = ?
		ORDER BY finished_at DESC, started_at DESC, created_at DESC
	`, bookID)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = rows.Close()
	}()

	result := make([]Read, 0)
	for rows.Next() {
		read, err := scanRead(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, read)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type readScanner interface {
	Scan(dest ...any) error
}

func scanRead(scanner readScanner) (Read, error) {
	var result Read
	var startedAt sql.NullString
	var finishedAt sql.NullString
	var rating sql.NullFloat64
	var notes sql.NullString
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&result.ID, &result.BookID, &startedAt, &finishedAt, &rating, &notes, &createdAt, &updatedAt); err != nil {
		return Read{}, err
	}

	if startedAt.Valid {
		result.StartedAt = &startedAt.String
	}
	if finishedAt.Valid {
		result.FinishedAt = &finishedAt.String
	}
	if rating.Valid {
		result.Rating = &rating.Float64
	}
	if notes.Valid {
		result.Notes = &notes.String
	}

	var err error
	result.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return Read{}, fmt.Errorf("parse created_at: %w", err)
	}
	result.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
	if err != nil {
		return Read{}, fmt.Errorf("parse updated_at: %w", err)
	}

	return result, nil
}

func nullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func nullFloat64(value *float64) sql.NullFloat64 {
	if value == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *value, Valid: true}
}
