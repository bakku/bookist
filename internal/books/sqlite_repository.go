package books

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/domain"
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
		SELECT id, title, isbn, language, publisher, edition, format,
		       purchased_at, purchase_price, pages, notes, summary, series_name, series_position,
		       location, condition, acquisition_source, published_year,
		       published_month, published_day, cover_image_key, created_at, updated_at
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
		SELECT id, title, isbn, language, publisher, edition, format, 
		    purchased_at, purchase_price, pages, notes, summary, series_name, series_position,
		    location, condition, acquisition_source, published_year,
		    published_month, published_day, cover_image_key, created_at, updated_at
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
		SELECT b.id, b.title, b.isbn, b.language, b.publisher, b.edition, b.format,
		       b.purchased_at, b.purchase_price, b.pages, b.notes, b.summary, b.series_name,
		       b.series_position, b.location, b.condition, b.acquisition_source,
		       b.published_year, b.published_month, b.published_day,
		       b.cover_image_key, b.created_at, b.updated_at
		FROM books b
		JOIN book_lists bl ON bl.book_id = b.id
		WHERE bl.list_id = ? AND `+matchesSearch+`
		ORDER BY bl.updated_at DESC, bl.id ASC
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

	purchasePrice := sql.NullString{}
	if input.PurchasePrice != nil {
		purchasePrice = sql.NullString{String: *input.PurchasePrice, Valid: true}
	}

	notes := sql.NullString{}
	if input.Notes != nil {
		notes = sql.NullString{String: *input.Notes, Valid: true}
	}

	summary := sql.NullString{}
	if input.Summary != nil {
		summary = sql.NullString{String: *input.Summary, Valid: true}
	}

	seriesName := sql.NullString{}
	if input.SeriesName != nil {
		seriesName = sql.NullString{String: *input.SeriesName, Valid: true}
	}

	seriesPosition := sql.NullFloat64{}
	if input.SeriesPosition != nil {
		seriesPosition = sql.NullFloat64{Float64: *input.SeriesPosition, Valid: true}
	}

	location := sql.NullString{}
	if input.Location != nil {
		location = sql.NullString{String: *input.Location, Valid: true}
	}

	condition := sql.NullString{}
	if input.Condition != nil {
		condition = sql.NullString{String: string(*input.Condition), Valid: true}
	}

	acquisitionSource := sql.NullString{}
	if input.AcquisitionSource != nil {
		acquisitionSource = sql.NullString{String: *input.AcquisitionSource, Valid: true}
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

	coverImageKey := sql.NullString{}
	if input.CoverImageKey != nil {
		coverImageKey = sql.NullString{String: *input.CoverImageKey, Valid: true}
	}

	row := tx.QueryRowContext(ctx, `
		INSERT INTO books (title, isbn, language, publisher, edition, format,
		                   purchased_at, purchase_price, pages, notes, summary, series_name,
		                   series_position, location, condition, acquisition_source,
		                   published_year, published_month, published_day, cover_image_key,
		                   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, title, isbn, language, publisher, edition, format, purchased_at,
			purchase_price,
			pages, notes, summary, series_name, series_position, location, condition,
			acquisition_source, published_year, published_month, published_day, cover_image_key,
			created_at, updated_at
	`, input.Title, isbn, language, publisher, edition, format, purchasedAt, purchasePrice,
		pages, notes, summary, seriesName, seriesPosition, location, condition,
		acquisitionSource, publishedYear, publishedMonth, publishedDay, coverImageKey, createdAt, updatedAt)

	book, err := scanBook(row)
	if err != nil {
		return Book{}, err
	}

	for position, authorID := range input.AuthorIDs {
		if err := insertBookAuthor(ctx, tx, book.ID, authorID, position, createdAt); err != nil {
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

func (r *SQLiteRepository) Update(ctx context.Context, id int64, input UpdateBookRequest) (UpdateResult, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return UpdateResult{}, err
	}
	defer func() { _ = tx.Rollback() }()

	var priorCover sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT cover_image_key FROM books WHERE id = ?`, id).Scan(&priorCover); err != nil {
		if err == sql.ErrNoRows {
			return UpdateResult{}, domain.ErrBookNotFound
		}
		return UpdateResult{}, err
	}

	updatedAt := time.Now().UTC().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []any{updatedAt}
	add := func(column string, present bool, value any) {
		if present {
			sets = append(sets, column+" = ?")
			args = append(args, value)
		}
	}
	add("title", input.Title.Present, input.Title.Value)
	add("isbn", input.ISBN.Present, input.ISBN.Value)
	add("language", input.Language.Present, input.Language.Value)
	add("publisher", input.Publisher.Present, input.Publisher.Value)
	add("edition", input.Edition.Present, input.Edition.Value)
	var format *string
	if input.Format.Value != nil {
		format = new(string(*input.Format.Value))
	}
	add("format", input.Format.Present, format)
	add("purchased_at", input.PurchasedAt.Present, input.PurchasedAt.Value)
	add("purchase_price", input.PurchasePrice.Present, input.PurchasePrice.Value)
	add("pages", input.Pages.Present, input.Pages.Value)
	add("notes", input.Notes.Present, input.Notes.Value)
	add("summary", input.Summary.Present, input.Summary.Value)
	add("series_name", input.SeriesName.Present, input.SeriesName.Value)
	add("series_position", input.SeriesPosition.Present, input.SeriesPosition.Value)
	add("location", input.Location.Present, input.Location.Value)
	var condition *string
	if input.Condition.Value != nil {
		condition = new(string(*input.Condition.Value))
	}
	add("condition", input.Condition.Present, condition)
	add("acquisition_source", input.AcquisitionSource.Present, input.AcquisitionSource.Value)
	add("published_year", input.PublishedYear.Present, input.PublishedYear.Value)
	add("published_month", input.PublishedMonth.Present, input.PublishedMonth.Value)
	add("published_day", input.PublishedDay.Present, input.PublishedDay.Value)
	add("cover_image_key", input.CoverImageKey.Present, input.CoverImageKey.Value)

	args = append(args, id)
	book, err := scanBook(tx.QueryRowContext(ctx, `
		UPDATE books SET `+strings.Join(sets, ", ")+`
		WHERE id = ?
		RETURNING id, title, isbn, language, publisher, edition, format, purchased_at,
			purchase_price, pages, notes, summary, series_name, series_position, location,
			condition, acquisition_source, published_year, published_month, published_day,
			cover_image_key, created_at, updated_at
	`, args...))
	if err != nil {
		return UpdateResult{}, err
	}

	if input.AuthorIDs.Present {
		if err := replaceBookAuthors(ctx, tx, id, valueOrZero(input.AuthorIDs.Value), updatedAt); err != nil {
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
	if input.CoverImageKey.Present && priorCover.Valid {
		result.PriorCoverImageKey = &priorCover.String
	}
	return result, nil
}

// Delete removes the book and returns the cover image key it had, if any, so
// the caller can clean up the file from the same statement that deleted the row.
func (r *SQLiteRepository) Delete(ctx context.Context, id int64) (*string, error) {
	var coverImageKey sql.NullString
	err := r.db.QueryRowContext(ctx, `DELETE FROM books WHERE id = ? RETURNING cover_image_key`, id).Scan(&coverImageKey)
	if err == sql.ErrNoRows {
		return nil, domain.ErrBookNotFound
	}
	if err != nil {
		return nil, err
	}
	if !coverImageKey.Valid {
		return nil, nil
	}
	return &coverImageKey.String, nil
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
	var isbn sql.NullString
	var language sql.NullString
	var publisher sql.NullString
	var edition sql.NullString
	var format sql.NullString
	var purchasedAt sql.NullString
	var purchasePrice sql.NullString
	var pages sql.NullInt64
	var notes sql.NullString
	var summary sql.NullString
	var seriesName sql.NullString
	var seriesPosition sql.NullFloat64
	var location sql.NullString
	var condition sql.NullString
	var acquisitionSource sql.NullString
	var publishedYear sql.NullInt64
	var publishedMonth sql.NullInt64
	var publishedDay sql.NullInt64
	var coverImageKey sql.NullString
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(&book.ID, &book.Title, &isbn, &language, &publisher, &edition,
		&format, &purchasedAt, &purchasePrice, &pages, &notes, &summary, &seriesName, &seriesPosition,
		&location, &condition, &acquisitionSource, &publishedYear, &publishedMonth,
		&publishedDay, &coverImageKey, &createdAt, &updatedAt); err != nil {
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

	if purchasePrice.Valid {
		book.PurchasePrice = &purchasePrice.String
	}

	if pages.Valid {
		p := int(pages.Int64)
		book.Pages = &p
	}

	if notes.Valid {
		book.Notes = &notes.String
	}

	if summary.Valid {
		book.Summary = &summary.String
	}

	if seriesName.Valid {
		book.SeriesName = &seriesName.String
	}

	if seriesPosition.Valid {
		book.SeriesPosition = &seriesPosition.Float64
	}

	if location.Valid {
		book.Location = &location.String
	}

	if condition.Valid {
		c := Condition(condition.String)
		book.Condition = &c
	}

	if acquisitionSource.Valid {
		book.AcquisitionSource = &acquisitionSource.String
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

	if coverImageKey.Valid {
		book.CoverImageKey = &coverImageKey.String
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
