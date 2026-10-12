package reads

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/domain"
)

const (
	terminalDatesConstraint  = "reads_terminal_dates_exclusive"
	finishedOrderConstraint  = "reads_finished_not_before_started"
	abandonedOrderConstraint = "reads_abandoned_not_before_started"
	startedAtConstraint      = "reads_started_at_valid"
	finishedAtConstraint     = "reads_finished_at_valid"
	abandonedAtConstraint    = "reads_abandoned_at_valid"
	ratingConstraint         = "reads_rating_valid"
)

const readColumns = `id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at`

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error) {
	now := database.Now()

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO reads (
			book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING `+readColumns+`
	`, bookID, input.StartedAt, input.FinishedAt, input.AbandonedAt, input.Rating, input.Notes, now, now)

	read, err := scanRead(row)
	return read, translateConstraints(err)
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Read, error) {
	read, err := scanRead(r.db.QueryRowContext(ctx, `
		SELECT `+readColumns+`
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
			return nil, domain.ErrBookNotFound
		}
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT `+readColumns+`
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
		args = append(args, input.StartedAt.Value)
	}
	if input.FinishedAt.Present {
		set = append(set, "finished_at = ?")
		args = append(args, input.FinishedAt.Value)
	}
	if input.AbandonedAt.Present {
		set = append(set, "abandoned_at = ?")
		args = append(args, input.AbandonedAt.Value)
	}
	if input.Rating.Present {
		set = append(set, "rating = ?")
		args = append(args, input.Rating.Value)
	}
	if input.Notes.Present {
		set = append(set, "notes = ?")
		args = append(args, input.Notes.Value)
	}
	if len(set) == 0 {
		return Read{}, ErrNoFieldsToUpdate
	}

	set = append(set, "updated_at = ?")
	args = append(args, database.Now(), id)
	read, err := scanRead(r.db.QueryRowContext(ctx, `
		UPDATE reads SET `+strings.Join(set, ", ")+`
		WHERE id = ?
		RETURNING `+readColumns+`
	`, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return Read{}, ErrReadNotFound
	}
	return read, translateConstraints(err)
}

func translateConstraints(err error) error {
	if err == nil {
		return nil
	}

	if database.IsForeignKeyViolation(err) {
		return domain.ErrBookNotFound
	}

	// Create and update share the same durable read invariants.
	switch {
	case database.IsCheckViolation(err, terminalDatesConstraint):
		return ErrConflictingTerminalDates
	case database.IsCheckViolation(err, finishedOrderConstraint):
		return ErrFinishedBeforeStarted
	case database.IsCheckViolation(err, abandonedOrderConstraint):
		return ErrAbandonedBeforeStarted
	case database.IsCheckViolation(err, startedAtConstraint):
		return ErrInvalidStartedAt
	case database.IsCheckViolation(err, finishedAtConstraint):
		return ErrInvalidFinishedAt
	case database.IsCheckViolation(err, abandonedAtConstraint):
		return ErrInvalidAbandonedAt
	case database.IsCheckViolation(err, ratingConstraint):
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
	err := scanner.Scan(&result.ID, &result.BookID, &result.StartedAt, &result.FinishedAt, &result.AbandonedAt,
		&result.Rating, &result.Notes, database.Time(&result.CreatedAt), database.Time(&result.UpdatedAt))
	if err != nil {
		return Read{}, err
	}

	return result, nil
}
