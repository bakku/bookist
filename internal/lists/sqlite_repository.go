package lists

import (
	"context"
	"database/sql"
	"errors"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/domain"
)

const listColumns = `id, name, description, created_at, updated_at`

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) Create(ctx context.Context, input CreateListRequest) (List, error) {
	now := database.Now()

	row := r.db.QueryRowContext(ctx, `
		INSERT INTO lists (name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		RETURNING `+listColumns+`
	`, input.Name, input.Description, now, now)

	list, err := scanList(row)
	if database.IsUniqueViolation(err) {
		return List{}, ErrNameConflict
	}
	return list, err
}

func (r *SQLiteRepository) Update(ctx context.Context, id int64, input UpdateListRequest) (List, error) {
	row := r.db.QueryRowContext(ctx, `
		UPDATE lists
		SET name = CASE WHEN ? THEN ? ELSE name END,
			description = CASE WHEN ? THEN ? ELSE description END,
			updated_at = ?
		WHERE id = ?
		RETURNING `+listColumns+`
	`, input.Name.Present, input.Name.Value, input.Description.Present, input.Description.Value, database.Now(), id)

	list, err := scanList(row)
	if database.IsUniqueViolation(err) {
		return List{}, ErrNameConflict
	}
	if errors.Is(err, sql.ErrNoRows) {
		return List{}, domain.ErrListNotFound
	}
	return list, err
}

func (r *SQLiteRepository) NameExists(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM lists WHERE name = ? COLLATE NOCASE)`, name).Scan(&exists)
	return exists, err
}

func (r *SQLiteRepository) NameExistsExcludingID(ctx context.Context, name string, id int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM lists WHERE name = ? COLLATE NOCASE AND id <> ?)`, name, id).Scan(&exists)
	return exists, err
}

func (r *SQLiteRepository) List(ctx context.Context) ([]List, error) {
	return r.Search(ctx, "")
}

func (r *SQLiteRepository) Search(ctx context.Context, query string) ([]List, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+listColumns+`
		FROM lists
		WHERE instr(bookist_fold(name), ?) > 0
		ORDER BY updated_at DESC, id ASC
	`, database.Fold(query))
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
		SELECT `+listColumns+`
		FROM lists
		WHERE id = ?
	`, id)

	list, err := scanList(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return List{}, domain.ErrListNotFound
		}
		return List{}, err
	}

	return list, nil
}

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM lists WHERE id = ?`, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrListNotFound
	}

	return nil
}

func (r *SQLiteRepository) AddBookToList(ctx context.Context, listID, bookID int64) error {
	now := database.Now()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO book_lists (list_id, book_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, listID, bookID, now, now)

	if database.IsForeignKeyViolation(err) {
		// Distinguish the missing parent without hiding lookup failures.
		_, listErr := r.GetByID(ctx, listID)
		if listErr != nil {
			return listErr
		}

		return domain.ErrBookNotFound
	}

	if database.IsUniqueViolation(err) {
		return ErrBookAlreadyInList
	}

	return err
}

func (r *SQLiteRepository) RemoveBookFromList(ctx context.Context, listID, bookID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	result, err := tx.ExecContext(ctx, `DELETE FROM book_lists WHERE list_id = ? AND book_id = ?`, listID, bookID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected > 0 {
		return tx.Commit()
	}

	var listExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM lists WHERE id = ?)`, listID).Scan(&listExists); err != nil {
		return err
	}
	if !listExists {
		return domain.ErrListNotFound
	}

	var bookExists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM books WHERE id = ?)`, bookID).Scan(&bookExists); err != nil {
		return err
	}
	if !bookExists {
		return domain.ErrBookNotFound
	}

	return ErrBookNotInList
}

type listScanner interface {
	Scan(dest ...any) error
}

func scanList(scanner listScanner) (List, error) {
	var list List
	err := scanner.Scan(&list.ID, &list.Name, &list.Description, database.Time(&list.CreatedAt), database.Time(&list.UpdatedAt))
	if err != nil {
		return List{}, err
	}

	return list, nil
}
