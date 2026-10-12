package books

import (
	"context"
	"database/sql"
	"strings"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/domain"
)

// bookDataColumns are the books columns written on insert; bookColumns adds the
// generated id and is the column list for every read and RETURNING clause.
const (
	bookDataColumns = `title, isbn, language, publisher, edition, format, purchased_at, purchase_price,
		pages, notes, summary, series_name, series_position, location, condition,
		acquisition_source, published_year, published_month, published_day, cover_image_key,
		created_at, updated_at`
	bookColumns = `id, ` + bookDataColumns
)

type SQLiteRepository struct {
	db *sql.DB
}

// matchesSearch matches a folded query (see database.Fold) as a substring of
// the title, series name, ISBN or the name of any linked author. Each use binds
// the query four times. It expects the books table to be aliased as b.
const matchesSearch = `(
			instr(bookist_fold(b.title), ?) > 0
			OR instr(bookist_fold(b.series_name), ?) > 0
			OR instr(bookist_fold(b.isbn), ?) > 0
			OR EXISTS (
				SELECT 1
				FROM book_authors ba
				JOIN authors a ON a.id = ba.author_id
				WHERE ba.book_id = b.id AND instr(bookist_fold(a.name), ?) > 0
			)
		)`

func NewSQLiteRepository(db *sql.DB) *SQLiteRepository {
	return &SQLiteRepository{db: db}
}

func (r *SQLiteRepository) List(ctx context.Context) ([]Book, error) {
	return r.Search(ctx, "")
}

func (r *SQLiteRepository) GetByID(ctx context.Context, id int64) (Book, error) {
	book, err := scanBook(r.db.QueryRowContext(ctx, `
		SELECT `+bookColumns+`
		FROM books
		WHERE id = ?
	`, id))
	if err == sql.ErrNoRows {
		return Book{}, domain.ErrBookNotFound
	}
	if err != nil {
		return Book{}, err
	}
	return loadBookAuthors(ctx, r.db, book)
}

func (r *SQLiteRepository) Search(ctx context.Context, query string) ([]Book, error) {
	folded := database.Fold(query)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+bookColumns+`
		FROM books b
		WHERE `+matchesSearch+`
		ORDER BY updated_at DESC, id ASC
	`, folded, folded, folded, folded)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	books := make([]Book, 0)
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

	if err := loadAuthors(ctx, r.db, books); err != nil {
		return nil, err
	}

	return books, nil
}

func (r *SQLiteRepository) ListByListID(ctx context.Context, listID int64) ([]Book, error) {
	return r.SearchByListID(ctx, listID, "")
}

func (r *SQLiteRepository) ListExists(ctx context.Context, listID int64) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM lists WHERE id = ?)`, listID).Scan(&exists)
	return exists, err
}

func (r *SQLiteRepository) SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error) {
	folded := database.Fold(query)
	rows, err := r.db.QueryContext(ctx, `
		SELECT `+bookColumns+`
		FROM (
			SELECT b.*, bl.id AS membership_id, bl.updated_at AS membership_updated_at
			FROM books b
			JOIN book_lists bl ON bl.book_id = b.id
			WHERE bl.list_id = ? AND `+matchesSearch+`
		)
		ORDER BY membership_updated_at DESC, membership_id ASC
	`, listID, folded, folded, folded, folded)
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

	if err := loadAuthors(ctx, r.db, bookList); err != nil {
		return nil, err
	}

	return bookList, nil
}

func (r *SQLiteRepository) Create(ctx context.Context, params CreateParams) (Book, error) {
	now := database.Now()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Book{}, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	book, err := scanBook(tx.QueryRowContext(ctx, `
		INSERT INTO books (`+bookDataColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING `+bookColumns+`
	`, params.Title, params.ISBN, params.Language, params.Publisher, params.Edition, params.Format,
		params.PurchasedAt, params.PurchasePrice, params.Pages, params.Notes, params.Summary,
		params.SeriesName, params.SeriesPosition, params.Location, params.Condition,
		params.AcquisitionSource, params.PublishedYear, params.PublishedMonth, params.PublishedDay,
		params.CoverImageKey, now, now))
	if err != nil {
		return Book{}, err
	}

	for position, authorID := range params.AuthorIDs {
		if err := insertBookAuthor(ctx, tx, book.ID, authorID, position, now); err != nil {
			return Book{}, err
		}
	}

	book, err = loadBookAuthors(ctx, tx, book)
	if err != nil {
		return Book{}, err
	}

	if err := tx.Commit(); err != nil {
		return Book{}, err
	}

	return book, nil
}

func (r *SQLiteRepository) Update(ctx context.Context, id int64, params UpdateParams) (UpdateResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return UpdateResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var priorCover *string
	if err := tx.QueryRowContext(ctx, `SELECT cover_image_key FROM books WHERE id = ?`, id).Scan(&priorCover); err != nil {
		if err == sql.ErrNoRows {
			return UpdateResult{}, domain.ErrBookNotFound
		}
		return UpdateResult{}, err
	}

	updatedAt := database.Now()
	sets := []string{"updated_at = ?"}
	args := []any{updatedAt}
	add := func(column string, present bool, value any) {
		if present {
			sets = append(sets, column+" = ?")
			args = append(args, value)
		}
	}
	add("title", params.Title.Present, params.Title.Value)
	add("isbn", params.ISBN.Present, params.ISBN.Value)
	add("language", params.Language.Present, params.Language.Value)
	add("publisher", params.Publisher.Present, params.Publisher.Value)
	add("edition", params.Edition.Present, params.Edition.Value)
	add("format", params.Format.Present, params.Format.Value)
	add("purchased_at", params.PurchasedAt.Present, params.PurchasedAt.Value)
	add("purchase_price", params.PurchasePrice.Present, params.PurchasePrice.Value)
	add("pages", params.Pages.Present, params.Pages.Value)
	add("notes", params.Notes.Present, params.Notes.Value)
	add("summary", params.Summary.Present, params.Summary.Value)
	add("series_name", params.SeriesName.Present, params.SeriesName.Value)
	add("series_position", params.SeriesPosition.Present, params.SeriesPosition.Value)
	add("location", params.Location.Present, params.Location.Value)
	add("condition", params.Condition.Present, params.Condition.Value)
	add("acquisition_source", params.AcquisitionSource.Present, params.AcquisitionSource.Value)
	add("published_year", params.PublishedYear.Present, params.PublishedYear.Value)
	add("published_month", params.PublishedMonth.Present, params.PublishedMonth.Value)
	add("published_day", params.PublishedDay.Present, params.PublishedDay.Value)
	add("cover_image_key", params.CoverImageKey.Present, params.CoverImageKey.Value)

	args = append(args, id)
	book, err := scanBook(tx.QueryRowContext(ctx, `
		UPDATE books SET `+strings.Join(sets, ", ")+`
		WHERE id = ?
		RETURNING `+bookColumns+`
	`, args...))
	if err != nil {
		return UpdateResult{}, err
	}

	if params.AuthorIDs.Present {
		if err := replaceBookAuthors(ctx, tx, id, valueOrZero(params.AuthorIDs.Value), updatedAt); err != nil {
			return UpdateResult{}, err
		}
	}

	book, err = loadBookAuthors(ctx, tx, book)
	if err != nil {
		return UpdateResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return UpdateResult{}, err
	}

	result := UpdateResult{Book: book}
	if params.CoverImageKey.Present {
		result.PriorCoverImageKey = priorCover
	}
	return result, nil
}

// Delete removes the book and returns the cover image key it had, if any, so
// the caller can clean up the file from the same statement that deleted the row.
func (r *SQLiteRepository) Delete(ctx context.Context, id int64) (*string, error) {
	var coverImageKey *string
	err := r.db.QueryRowContext(ctx, `DELETE FROM books WHERE id = ? RETURNING cover_image_key`, id).Scan(&coverImageKey)
	if err == sql.ErrNoRows {
		return nil, domain.ErrBookNotFound
	}
	if err != nil {
		return nil, err
	}
	return coverImageKey, nil
}

// insertBookAuthor reports a missing author, including one deleted after
// service validation, as ErrAuthorNotFound.
func insertBookAuthor(ctx context.Context, tx *sql.Tx, bookID, authorID int64, position int, timestamp string) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO book_authors (book_id, author_id, position, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, bookID, authorID, position, timestamp, timestamp)
	if database.IsForeignKeyViolation(err) {
		return ErrAuthorNotFound
	}
	return err
}

// replaceBookAuthors makes authorIDs, which must not contain duplicates, the
// book's authors in that order. Rows for retained authors are kept so their ID
// and created_at survive; only rows whose position changes get a new updated_at.
func replaceBookAuthors(ctx context.Context, tx *sql.Tx, bookID int64, authorIDs []int64, timestamp string) error {
	rows, err := tx.QueryContext(ctx, `SELECT author_id, position FROM book_authors WHERE book_id = ?`, bookID)
	if err != nil {
		return err
	}
	existing := make(map[int64]int)
	maxPosition := -1
	for rows.Next() {
		var authorID int64
		var position int
		if err := rows.Scan(&authorID, &position); err != nil {
			_ = rows.Close()
			return err
		}
		existing[authorID] = position
		maxPosition = max(maxPosition, position)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
		return err
	}

	desired := make(map[int64]bool, len(authorIDs))
	for _, authorID := range authorIDs {
		desired[authorID] = true
	}
	for authorID := range existing {
		if !desired[authorID] {
			if _, err := tx.ExecContext(ctx, `DELETE FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, authorID); err != nil {
				return err
			}
		}
	}

	// UNIQUE (book_id, position) is checked per row, so moving retained rows
	// straight to their new positions could collide with rows not yet moved.
	// Park them above every old and new position first.
	if _, err := tx.ExecContext(ctx, `UPDATE book_authors SET position = position + ? WHERE book_id = ?`,
		maxPosition+1+len(authorIDs), bookID); err != nil {
		return err
	}

	for position, authorID := range authorIDs {
		oldPosition, retained := existing[authorID]
		if !retained {
			if err := insertBookAuthor(ctx, tx, bookID, authorID, position, timestamp); err != nil {
				return err
			}
			continue
		}

		updatedAt := "updated_at"
		args := []any{position}
		if oldPosition != position {
			updatedAt = "?"
			args = append(args, timestamp)
		}
		args = append(args, bookID, authorID)
		if _, err := tx.ExecContext(ctx, `UPDATE book_authors SET position = ?, updated_at = `+updatedAt+` WHERE book_id = ? AND author_id = ?`, args...); err != nil {
			return err
		}
	}

	return nil
}

type authorQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// bookAuthorScanner prefixes an author row scan with its book ID.
type bookAuthorScanner struct {
	bookID *int64
	rows   *sql.Rows
}

func (s bookAuthorScanner) Scan(dest ...any) error {
	return s.rows.Scan(append([]any{s.bookID}, dest...)...)
}

// loadBookAuthors returns book with its authors loaded.
func loadBookAuthors(ctx context.Context, q authorQuerier, book Book) (Book, error) {
	loaded := []Book{book}
	if err := loadAuthors(ctx, q, loaded); err != nil {
		return Book{}, err
	}
	return loaded[0], nil
}

// loadAuthors sets Authors on every book, ordered by book_authors.position.
// This is deliberately not the updated_at DESC collection order: author order
// is part of the book's data, not edit history. Books without authors get an
// empty, non-nil slice.
func loadAuthors(ctx context.Context, q authorQuerier, books []Book) error {
	if len(books) == 0 {
		return nil
	}

	placeholders := make([]string, len(books))
	args := make([]any, len(books))
	indexByID := make(map[int64]int, len(books))
	for i := range books {
		placeholders[i] = "?"
		args[i] = books[i].ID
		indexByID[books[i].ID] = i
		books[i].Authors = []authors.Author{}
	}

	rows, err := q.QueryContext(ctx, `
		SELECT ba.book_id, a.id, a.name, a.created_at, a.updated_at
		FROM book_authors ba
		JOIN authors a ON a.id = ba.author_id
		WHERE ba.book_id IN (`+strings.Join(placeholders, ",")+`)
		ORDER BY ba.book_id ASC, ba.position ASC
	`, args...)
	if err != nil {
		return err
	}
	defer func() {
		_ = rows.Close()
	}()

	for rows.Next() {
		var bookID int64
		author, err := authors.ScanAuthor(bookAuthorScanner{bookID: &bookID, rows: rows})
		if err != nil {
			return err
		}
		i := indexByID[bookID]
		books[i].Authors = append(books[i].Authors, author)
	}

	return rows.Err()
}

type bookScanner interface {
	Scan(dest ...any) error
}

func scanBook(scanner bookScanner) (Book, error) {
	var book Book
	err := scanner.Scan(&book.ID, &book.Title, &book.ISBN, &book.Language, &book.Publisher, &book.Edition,
		&book.Format, &book.PurchasedAt, &book.PurchasePrice, &book.Pages, &book.Notes, &book.Summary,
		&book.SeriesName, &book.SeriesPosition, &book.Location, &book.Condition, &book.AcquisitionSource,
		&book.PublishedYear, &book.PublishedMonth, &book.PublishedDay, &book.CoverImageKey,
		database.Time(&book.CreatedAt), database.Time(&book.UpdatedAt))
	if err != nil {
		return Book{}, err
	}

	return book, nil
}
