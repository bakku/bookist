package reads

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO reads (
			book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
	`, bookID, nullString(input.StartedAt), nullString(input.FinishedAt),
		nullString(input.AbandonedAt), nullFloat64(input.Rating), nullString(input.Notes), now, now)

	read, err := scanRead(row)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY constraint failed") {
			return Read{}, ErrBookNotFound
		}
		return Read{}, err
	}

	return read, nil
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Read, error) {
	read, err := scanRead(r.db.QueryRowContext(ctx, `
		SELECT id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		FROM reads
		WHERE id = ?
	`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Read{}, ErrReadNotFound
	}
	return read, err
}

func (r *SQLiteRepository) ListByBookID(ctx context.Context, bookID int64) ([]Read, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT 1 FROM books WHERE id = ?`, bookID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrBookNotFound
		}
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		FROM reads
		WHERE book_id = ?
		ORDER BY updated_at DESC, id ASC
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

func (r *SQLiteRepository) Update(ctx context.Context, id int64, input UpdateReadRequest) (Read, error) {
	set := make([]string, 0, 6)
	args := make([]any, 0, 7)
	if input.StartedAt.Present {
		set = append(set, "started_at = ?")
		args = append(args, nullString(input.StartedAt.Value))
	}
	if input.FinishedAt.Present {
		set = append(set, "finished_at = ?")
		args = append(args, nullString(input.FinishedAt.Value))
	}
	if input.AbandonedAt.Present {
		set = append(set, "abandoned_at = ?")
		args = append(args, nullString(input.AbandonedAt.Value))
	}
	if input.Rating.Present {
		set = append(set, "rating = ?")
		args = append(args, nullFloat64(input.Rating.Value))
	}
	if input.Notes.Present {
		set = append(set, "notes = ?")
		args = append(args, nullString(input.Notes.Value))
	}
	if len(set) == 0 {
		return Read{}, ErrNoFieldsToUpdate
	}

	set = append(set, "updated_at = ?")
	args = append(args, time.Now().UTC().Format(time.RFC3339), id)
	read, err := scanRead(r.db.QueryRowContext(ctx, `
		UPDATE reads SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
	`, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Read{}, ErrReadNotFound
	}
	return read, translateUpdateConstraintError(err)
}

func translateUpdateConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_CHECK {
		return err
	}

	message := sqliteErr.Error()
	switch {
	case strings.Contains(message, "reads_terminal_dates_exclusive"):
		return ErrConflictingTerminalDates
	case strings.Contains(message, "reads_finished_not_before_started"):
		return ErrFinishedBeforeStarted
	case strings.Contains(message, "reads_abandoned_not_before_started"):
		return ErrAbandonedBeforeStarted
	case strings.Contains(message, "reads_started_at_valid"):
		return ErrInvalidStartedAt
	case strings.Contains(message, "reads_finished_at_valid"):
		return ErrInvalidFinishedAt
	case strings.Contains(message, "reads_abandoned_at_valid"):
		return ErrInvalidAbandonedAt
	case strings.Contains(message, "reads_rating_valid"):
		return ErrInvalidRating
	default:
		return err
	}
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM reads WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrReadNotFound
	}

	return nil
}

type readScanner interface {
	Scan(dest ...any) error
}

func scanRead(scanner readScanner) (Read, error) {
	var result Read
	var startedAt sql.NullString
	var finishedAt sql.NullString
	var abandonedAt sql.NullString
	var rating sql.NullFloat64
	var notes sql.NullString
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&result.ID, &result.BookID, &startedAt, &finishedAt, &abandonedAt, &rating, &notes, &createdAt, &updatedAt); err != nil {
		return Read{}, err
	}

	if startedAt.Valid {
		result.StartedAt = &startedAt.String
	}
	if finishedAt.Valid {
		result.FinishedAt = &finishedAt.String
	}
	if abandonedAt.Valid {
		result.AbandonedAt = &abandonedAt.String
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
