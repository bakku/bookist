package books

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/domain"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/validation"
)

type Service struct {
	repository Repository
	authorRepo authors.Repository
	coverStore CoverStore
	logger     *slog.Logger
}

type CoverStore interface {
	Save(data []byte) (string, error)
	Delete(key string) error
}

func NewService(repository Repository, authorRepo authors.Repository, coverStore CoverStore) *Service {
	return NewServiceWithLogger(repository, authorRepo, coverStore, slog.Default())
}

func NewServiceWithLogger(repository Repository, authorRepo authors.Repository, coverStore CoverStore, logger *slog.Logger) *Service {
	if logger == nil {
		logger = slog.Default()
	}
	return &Service{repository: repository, authorRepo: authorRepo, coverStore: coverStore, logger: logger}
}

type bookValidationState struct {
	format         *Format
	purchasedAt    *string
	pages          *int
	seriesPosition *float64
	condition      *Condition
	publishedYear  *int
	publishedMonth *int
	publishedDay   *int
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	books, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	return withCoverURLs(books), nil
}

func (s *Service) Search(ctx context.Context, query string) ([]Book, error) {
	books, err := s.repository.Search(ctx, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}

	return withCoverURLs(books), nil
}

func (s *Service) ListByListID(ctx context.Context, listID int64) ([]Book, error) {
	books, err := s.repository.ListByListID(ctx, listID)
	if err != nil {
		return nil, err
	}

	return withCoverURLs(books), nil
}

func (s *Service) SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error) {
	exists, err := s.repository.ListExists(ctx, listID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, domain.ErrListNotFound
	}

	books, err := s.repository.SearchByListID(ctx, listID, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}

	return withCoverURLs(books), nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (Book, error) {
	book, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Book{}, err
	}

	setCoverURL(&book)

	return book, nil
}

func (s *Service) Create(ctx context.Context, input CreateBookRequest) (Book, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return Book{}, ErrTitleRequired
	}

	input.ISBN = validation.TrimToNil(input.ISBN)
	input.Language = validation.TrimToNil(input.Language)
	input.Publisher = validation.TrimToNil(input.Publisher)
	input.Edition = validation.TrimToNil(input.Edition)

	input.Format = normalizeBookFormat(input.Format)

	input.PurchasedAt = validation.TrimToNil(input.PurchasedAt)
	input.PurchasePrice = validation.TrimToNil(input.PurchasePrice)
	input.Notes = validation.TrimToNil(input.Notes)
	input.Summary = validation.TrimToNil(input.Summary)
	input.SeriesName = validation.TrimToNil(input.SeriesName)
	input.Location = validation.TrimToNil(input.Location)

	input.Condition = normalizeBookCondition(input.Condition)

	input.AcquisitionSource = validation.TrimToNil(input.AcquisitionSource)

	if err := validateBookState(bookValidationState{
		format: input.Format, purchasedAt: input.PurchasedAt, pages: input.Pages,
		seriesPosition: input.SeriesPosition, condition: input.Condition,
		publishedYear: input.PublishedYear, publishedMonth: input.PublishedMonth, publishedDay: input.PublishedDay,
	}); err != nil {
		return Book{}, err
	}

	dedupedAuthorIDs, err := s.validateAuthorIDs(ctx, input.AuthorIDs)
	if err != nil {
		return Book{}, err
	}
	input.AuthorIDs = dedupedAuthorIDs

	if input.Cover != nil {
		key, err := s.coverStore.Save(*input.Cover)
		if err != nil {
			return Book{}, err
		}
		input.CoverImageKey = &key
	}

	book, err := s.repository.Create(ctx, input)
	if err != nil {
		if input.CoverImageKey != nil {
			if cleanupErr := s.coverStore.Delete(*input.CoverImageKey); cleanupErr != nil {
				return Book{}, errors.Join(err, fmt.Errorf("clean up cover: %w", cleanupErr))
			}
		}
		return Book{}, err
	}

	setCoverURL(&book)

	return book, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateBookRequest) (Book, error) {
	if !input.hasFields() {
		return Book{}, ErrNoFieldsToUpdate
	}

	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Book{}, err
	}

	if input.Title.Present {
		if input.Title.Value == nil {
			return Book{}, ErrTitleRequired
		}
		*input.Title.Value = strings.TrimSpace(*input.Title.Value)
		if *input.Title.Value == "" {
			return Book{}, ErrTitleRequired
		}
	}
	for _, value := range []*optional.Value[string]{
		&input.ISBN, &input.Language, &input.Publisher, &input.Edition, &input.PurchasedAt,
		&input.PurchasePrice, &input.Notes, &input.Summary, &input.SeriesName, &input.Location,
		&input.AcquisitionSource,
	} {
		value.Value = validation.TrimToNil(value.Value)
	}

	if input.Format.Present && input.Format.Value != nil {
		input.Format.Value = normalizeBookFormat(input.Format.Value)
	}
	if input.Condition.Present && input.Condition.Value != nil {
		input.Condition.Value = normalizeBookCondition(input.Condition.Value)
	}

	if err := validateBookState(bookValidationState{
		format:         updateValue(current.Format, input.Format),
		purchasedAt:    updateValue(current.PurchasedAt, input.PurchasedAt),
		pages:          updateValue(current.Pages, input.Pages),
		seriesPosition: updateValue(current.SeriesPosition, input.SeriesPosition),
		condition:      updateValue(current.Condition, input.Condition),
		publishedYear:  updateValue(current.PublishedYear, input.PublishedYear),
		publishedMonth: updateValue(current.PublishedMonth, input.PublishedMonth),
		publishedDay:   updateValue(current.PublishedDay, input.PublishedDay),
	}); err != nil {
		return Book{}, err
	}

	if input.AuthorIDs.Present {
		deduped, err := s.validateAuthorIDs(ctx, valueOrZero(input.AuthorIDs.Value))
		if err != nil {
			return Book{}, err
		}
		input.AuthorIDs.Value = &deduped
	}

	var newCoverKey *string
	if input.Cover.Present {
		input.CoverImageKey.Present = true
		if input.Cover.Value != nil {
			key, err := s.coverStore.Save(*input.Cover.Value)
			if err != nil {
				return Book{}, err
			}
			newCoverKey = &key
			input.CoverImageKey.Value = newCoverKey
		}
	}

	result, err := s.repository.Update(ctx, id, input)
	if err != nil {
		if newCoverKey != nil {
			if cleanupErr := s.coverStore.Delete(*newCoverKey); cleanupErr != nil {
				return Book{}, errors.Join(err, fmt.Errorf("clean up cover: %w", cleanupErr))
			}
		}
		return Book{}, err
	}
	book := result.Book
	priorCoverKey := result.PriorCoverImageKey
	if priorCoverKey != nil && (newCoverKey == nil || *priorCoverKey != *newCoverKey) {
		// The database update has committed, so old-cover cleanup is best effort.
		// Reporting an error here would incorrectly tell clients that the update failed.
		if cleanupErr := s.coverStore.Delete(*priorCoverKey); cleanupErr != nil {
			s.logger.WarnContext(ctx, "failed to delete previous book cover",
				"book_id", id,
				"cover_key", *priorCoverKey,
				"error", cleanupErr,
			)
		}
	}

	setCoverURL(&book)
	return book, nil
}

func updateValue[T any](current *T, update optional.Value[T]) *T {
	if update.Present {
		return update.Value
	}
	return current
}

func valueOrZero[T any](value *T) T {
	if value == nil {
		var zero T
		return zero
	}
	return *value
}

func normalizeBookFormat(value *Format) *Format {
	if value == nil {
		return nil
	}
	format := Format(strings.TrimSpace(string(*value)))
	return &format
}

func normalizeBookCondition(value *Condition) *Condition {
	if value == nil {
		return nil
	}
	condition := Condition(strings.TrimSpace(string(*value)))
	return &condition
}

func validateBookState(state bookValidationState) error {
	if state.format != nil {
		switch *state.format {
		case FormatHardback, FormatPaperback, FormatEpub:
		default:
			return ErrInvalidFormat
		}
	}
	if state.condition != nil {
		switch *state.condition {
		case ConditionNew, ConditionVeryGood, ConditionGood, ConditionAcceptable, ConditionPoor:
		default:
			return ErrInvalidCondition
		}
	}
	if state.purchasedAt != nil && !validation.IsCalendarDate(*state.purchasedAt) {
		return ErrInvalidPurchasedAt
	}
	if state.seriesPosition != nil &&
		(!(*state.seriesPosition > 0) || math.IsInf(*state.seriesPosition, 0) || math.IsNaN(*state.seriesPosition)) {
		return ErrInvalidSeriesPosition
	}
	if state.pages != nil && *state.pages < 1 {
		return ErrInvalidPages
	}
	if state.publishedYear != nil && *state.publishedYear < 1 {
		return ErrInvalidPublishedYear
	}
	if state.publishedMonth != nil &&
		(state.publishedYear == nil || *state.publishedMonth < 1 || *state.publishedMonth > 12) {
		return ErrInvalidPublishedMonth
	}
	if state.publishedDay != nil &&
		(state.publishedYear == nil || state.publishedMonth == nil || *state.publishedDay < 1 ||
			*state.publishedDay > time.Date(*state.publishedYear, time.Month(*state.publishedMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()) {
		return ErrInvalidPublishedDay
	}
	return nil
}

// validateAuthorIDs drops repeated IDs, keeping each author's first position,
// and reports ErrAuthorNotFound unless every author exists.
func (s *Service) validateAuthorIDs(ctx context.Context, authorIDs []int64) ([]int64, error) {
	deduped := make([]int64, 0, len(authorIDs))
	seen := make(map[int64]bool, len(authorIDs))
	for _, authorID := range authorIDs {
		if authorID <= 0 {
			return nil, ErrAuthorNotFound
		}
		if !seen[authorID] {
			seen[authorID] = true
			deduped = append(deduped, authorID)
		}
	}

	if len(deduped) == 0 {
		return deduped, nil
	}
	found, err := s.authorRepo.GetByIDs(ctx, deduped)
	if err != nil {
		return nil, err
	}
	foundIDs := make(map[int64]bool, len(found))
	for _, author := range found {
		foundIDs[author.ID] = true
	}
	for _, authorID := range deduped {
		if !foundIDs[authorID] {
			return nil, ErrAuthorNotFound
		}
	}
	return deduped, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	coverKey, err := s.repository.Delete(ctx, id)
	if err != nil {
		return err
	}

	if coverKey != nil {
		// The book row is already gone, so cover cleanup is best effort.
		// Reporting an error here would incorrectly tell clients that the delete failed.
		if cleanupErr := s.coverStore.Delete(*coverKey); cleanupErr != nil {
			s.logger.WarnContext(ctx, "failed to delete book cover",
				"book_id", id,
				"cover_key", *coverKey,
				"error", cleanupErr,
			)
		}
	}

	return nil
}

func withCoverURLs(books []Book) []Book {
	for i := range books {
		setCoverURL(&books[i])
	}

	return books
}

func setCoverURL(book *Book) {
	if book.CoverImageKey == nil {
		book.CoverURL = nil
		return
	}

	url := "/book-covers/" + *book.CoverImageKey

	book.CoverURL = &url
}
