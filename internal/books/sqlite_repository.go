package books

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
)

type SQLiteRepository struct {
	db *sql.DB
}

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
		return Book{}, ErrBookNotFound
	}
	return book, err
}

func (r *SQLiteRepository) Search(ctx context.Context, query string) ([]Book, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, title, isbn, language, publisher, edition, format, 
		    purchased_at, purchase_price, pages, notes, summary, series_name, series_position,
		    location, condition, acquisition_source, published_year,
		    published_month, published_day, cover_image_key, created_at, updated_at
		FROM books
		WHERE instr(lower(title), lower(?)) > 0
		ORDER BY updated_at DESC, id ASC
	`, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book
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

	return books, nil
}

func (r *SQLiteRepository) ListByListID(ctx context.Context, listID int64) ([]Book, error) {
	return r.SearchByListID(ctx, listID, "")
}

func (r *SQLiteRepository) SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT b.id, b.title, b.isbn, b.language, b.publisher, b.edition, b.format,
		       b.purchased_at, b.purchase_price, b.pages, b.notes, b.summary, b.series_name,
		       b.series_position, b.location, b.condition, b.acquisition_source,
		       b.published_year, b.published_month, b.published_day,
		       b.cover_image_key, b.created_at, b.updated_at
		FROM books b
		JOIN book_lists bl ON bl.book_id = b.id
		WHERE bl.list_id = ? AND instr(lower(b.title), lower(?)) > 0
		ORDER BY bl.updated_at DESC, bl.id ASC
	`, listID, query)
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

	for _, authorID := range input.AuthorIDs {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO book_authors (book_id, author_id, created_at, updated_at)
			VALUES (?, ?, ?, ?)
		`, book.ID, authorID, createdAt, updatedAt)
		if err != nil {
			return Book{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return Book{}, err
	}

	book.Authors = []authors.Author{}

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
			return UpdateResult{}, ErrBookNotFound
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
		desired := make(map[int64]bool)
		if input.AuthorIDs.Value != nil {
			for _, authorID := range *input.AuthorIDs.Value {
				desired[authorID] = true
			}
		}

		rows, err := tx.QueryContext(ctx, `SELECT author_id FROM book_authors WHERE book_id = ?`, id)
		if err != nil {
			return UpdateResult{}, err
		}
		existing := make(map[int64]bool)
		for rows.Next() {
			var authorID int64
			if err := rows.Scan(&authorID); err != nil {
				_ = rows.Close()
				return UpdateResult{}, err
			}
			existing[authorID] = true
		}
		if err := rows.Close(); err != nil {
			return UpdateResult{}, err
		}
		if err := rows.Err(); err != nil {
			return UpdateResult{}, err
		}

		for authorID := range existing {
			if !desired[authorID] {
				if _, err := tx.ExecContext(ctx, `DELETE FROM book_authors WHERE book_id = ? AND author_id = ?`, id, authorID); err != nil {
					return UpdateResult{}, err
				}
			}
		}
		inserted := make(map[int64]bool)
		if input.AuthorIDs.Value != nil {
			for _, authorID := range *input.AuthorIDs.Value {
				if existing[authorID] || inserted[authorID] {
					continue
				}
				if _, err := tx.ExecContext(ctx, `
					INSERT INTO book_authors (book_id, author_id, created_at, updated_at)
					VALUES (?, ?, ?, ?)
				`, id, authorID, updatedAt, updatedAt); err != nil {
					return UpdateResult{}, err
				}
				inserted[authorID] = true
			}
		}
	}

	authorRows, err := tx.QueryContext(ctx, `
		SELECT a.id, a.name, a.created_at, a.updated_at
		FROM book_authors ba
		JOIN authors a ON a.id = ba.author_id
		WHERE ba.book_id = ?
		ORDER BY ba.updated_at DESC, ba.id ASC
	`, id)
	if err != nil {
		return UpdateResult{}, err
	}
	book.Authors = make([]authors.Author, 0)
	for authorRows.Next() {
		var author authors.Author
		var createdAt, updatedAt string
		if err := authorRows.Scan(&author.ID, &author.Name, &createdAt, &updatedAt); err != nil {
			_ = authorRows.Close()
			return UpdateResult{}, err
		}
		author.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			_ = authorRows.Close()
			return UpdateResult{}, fmt.Errorf("parse author created_at: %w", err)
		}
		author.UpdatedAt, err = time.Parse(time.RFC3339, updatedAt)
		if err != nil {
			_ = authorRows.Close()
			return UpdateResult{}, fmt.Errorf("parse author updated_at: %w", err)
		}
		book.Authors = append(book.Authors, author)
	}
	if err := authorRows.Close(); err != nil {
		return UpdateResult{}, err
	}
	if err := authorRows.Err(); err != nil {
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

func (r *SQLiteRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM books WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrBookNotFound
	}
	return nil
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
