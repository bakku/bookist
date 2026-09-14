package books

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/validation"
)

type Service struct {
	repository Repository
	authorRepo authors.Repository
	coverStore CoverStore
}

type CoverStore interface {
	Save(data []byte) (string, error)
	Delete(key string) error
}

func NewService(repository Repository, authorRepo authors.Repository, coverStore CoverStore) *Service {
	return &Service{repository: repository, authorRepo: authorRepo, coverStore: coverStore}
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	books, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	return s.withAuthorsAndCover(ctx, books)
}

func (s *Service) Search(ctx context.Context, query string) ([]Book, error) {
	books, err := s.repository.Search(ctx, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}

	return s.withAuthorsAndCover(ctx, books)
}

func (s *Service) ListByListID(ctx context.Context, listID int64) ([]Book, error) {
	books, err := s.repository.ListByListID(ctx, listID)
	if err != nil {
		return nil, err
	}

	return s.withAuthorsAndCover(ctx, books)
}

func (s *Service) SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error) {
	books, err := s.repository.SearchByListID(ctx, listID, strings.TrimSpace(query))
	if err != nil {
		return nil, err
	}

	return s.withAuthorsAndCover(ctx, books)
}

func (s *Service) Create(ctx context.Context, input CreateBookRequest) (Book, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" {
		return Book{}, ErrTitleRequired
	}

	if input.ISBN != nil {
		isbn := strings.TrimSpace(*input.ISBN)
		if isbn == "" {
			input.ISBN = nil
		} else {
			input.ISBN = &isbn
		}
	}

	if input.Language != nil {
		v := strings.TrimSpace(*input.Language)
		if v == "" {
			input.Language = nil
		} else {
			input.Language = &v
		}
	}

	if input.Publisher != nil {
		v := strings.TrimSpace(*input.Publisher)
		if v == "" {
			input.Publisher = nil
		} else {
			input.Publisher = &v
		}
	}

	if input.Edition != nil {
		v := strings.TrimSpace(*input.Edition)
		if v == "" {
			input.Edition = nil
		} else {
			input.Edition = &v
		}
	}

	if input.Format != nil {
		f := Format(strings.TrimSpace(string(*input.Format)))
		switch f {
		case FormatHardback, FormatPaperback, FormatEpub:
			input.Format = &f
		default:
			return Book{}, ErrInvalidFormat
		}
	}

	if input.PurchasedAt != nil {
		v := strings.TrimSpace(*input.PurchasedAt)
		if v == "" {
			input.PurchasedAt = nil
		} else {
			if !validation.IsCalendarDate(v) {
				return Book{}, ErrInvalidPurchasedAt
			}

			input.PurchasedAt = &v
		}
	}

	if input.PurchasePrice != nil {
		v := strings.TrimSpace(*input.PurchasePrice)
		if v == "" {
			input.PurchasePrice = nil
		} else {
			input.PurchasePrice = &v
		}
	}

	if input.Notes != nil {
		v := strings.TrimSpace(*input.Notes)
		if v == "" {
			input.Notes = nil
		} else {
			input.Notes = &v
		}
	}

	if input.Summary != nil {
		v := strings.TrimSpace(*input.Summary)
		if v == "" {
			input.Summary = nil
		} else {
			input.Summary = &v
		}
	}

	if input.SeriesName != nil {
		v := strings.TrimSpace(*input.SeriesName)
		if v == "" {
			input.SeriesName = nil
		} else {
			input.SeriesName = &v
		}
	}

	if input.Location != nil {
		v := strings.TrimSpace(*input.Location)
		if v == "" {
			input.Location = nil
		} else {
			input.Location = &v
		}
	}

	if input.Condition != nil {
		condition := Condition(strings.TrimSpace(string(*input.Condition)))
		switch condition {
		case ConditionNew, ConditionVeryGood, ConditionGood, ConditionAcceptable, ConditionPoor:
			input.Condition = &condition
		default:
			return Book{}, ErrInvalidCondition
		}
	}

	if input.AcquisitionSource != nil {
		v := strings.TrimSpace(*input.AcquisitionSource)
		if v == "" {
			input.AcquisitionSource = nil
		} else {
			input.AcquisitionSource = &v
		}
	}

	if input.SeriesPosition != nil &&
		(!(*input.SeriesPosition > 0) || math.IsInf(*input.SeriesPosition, 0) || math.IsNaN(*input.SeriesPosition)) {
		return Book{}, ErrInvalidSeriesPosition
	}

	if input.Pages != nil && *input.Pages < 1 {
		return Book{}, ErrInvalidPages
	}

	if input.PublishedYear != nil && *input.PublishedYear < 1 {
		return Book{}, ErrInvalidPublishedYear
	}

	if input.PublishedMonth != nil && (input.PublishedYear == nil || *input.PublishedMonth < 1 || *input.PublishedMonth > 12) {
		return Book{}, ErrInvalidPublishedMonth
	}

	if input.PublishedDay != nil {
		if input.PublishedYear == nil || input.PublishedMonth == nil || *input.PublishedDay < 1 ||
			*input.PublishedDay > time.Date(*input.PublishedYear, time.Month(*input.PublishedMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day() {
			return Book{}, ErrInvalidPublishedDay
		}
	}

	seen := make(map[int64]bool)
	var deduped []int64
	for _, id := range input.AuthorIDs {
		if id <= 0 {
			return Book{}, ErrAuthorNotFound
		}
		if !seen[id] {
			seen[id] = true
			deduped = append(deduped, id)
		}
	}
	input.AuthorIDs = deduped

	var found []authors.Author
	if len(input.AuthorIDs) > 0 {
		var err error
		found, err = s.authorRepo.GetByIDs(ctx, input.AuthorIDs)
		if err != nil {
			return Book{}, err
		}

		foundMap := make(map[int64]bool)
		for _, a := range found {
			foundMap[a.ID] = true
		}
		for _, id := range input.AuthorIDs {
			if !foundMap[id] {
				return Book{}, ErrAuthorNotFound
			}
		}
	}

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

	if len(input.AuthorIDs) > 0 {
		book.Authors = found
	} else {
		book.Authors = []authors.Author{}
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
		if value.Present && value.Value != nil {
			*value.Value = strings.TrimSpace(*value.Value)
			if *value.Value == "" {
				return Book{}, ErrBlankOptionalString
			}
		}
	}

	if input.Format.Present && input.Format.Value != nil {
		format := Format(strings.TrimSpace(string(*input.Format.Value)))
		switch format {
		case FormatHardback, FormatPaperback, FormatEpub:
			input.Format.Value = &format
		default:
			return Book{}, ErrInvalidFormat
		}
	}
	if input.Condition.Present && input.Condition.Value != nil {
		condition := Condition(strings.TrimSpace(string(*input.Condition.Value)))
		switch condition {
		case ConditionNew, ConditionVeryGood, ConditionGood, ConditionAcceptable, ConditionPoor:
			input.Condition.Value = &condition
		default:
			return Book{}, ErrInvalidCondition
		}
	}

	finalPurchasedAt := updateValue(current.PurchasedAt, input.PurchasedAt)
	if finalPurchasedAt != nil && !validation.IsCalendarDate(*finalPurchasedAt) {
		return Book{}, ErrInvalidPurchasedAt
	}
	finalPages := updateValue(current.Pages, input.Pages)
	if finalPages != nil && *finalPages < 1 {
		return Book{}, ErrInvalidPages
	}
	finalSeriesPosition := updateValue(current.SeriesPosition, input.SeriesPosition)
	if finalSeriesPosition != nil && (!(*finalSeriesPosition > 0) || math.IsInf(*finalSeriesPosition, 0) || math.IsNaN(*finalSeriesPosition)) {
		return Book{}, ErrInvalidSeriesPosition
	}
	finalYear := updateValue(current.PublishedYear, input.PublishedYear)
	finalMonth := updateValue(current.PublishedMonth, input.PublishedMonth)
	finalDay := updateValue(current.PublishedDay, input.PublishedDay)
	if finalYear != nil && *finalYear < 1 {
		return Book{}, ErrInvalidPublishedYear
	}
	if finalMonth != nil && (finalYear == nil || *finalMonth < 1 || *finalMonth > 12) {
		return Book{}, ErrInvalidPublishedMonth
	}
	if finalDay != nil && (finalYear == nil || finalMonth == nil || *finalDay < 1 ||
		*finalDay > time.Date(*finalYear, time.Month(*finalMonth)+1, 0, 0, 0, 0, 0, time.UTC).Day()) {
		return Book{}, ErrInvalidPublishedDay
	}

	if input.AuthorIDs.Present {
		var deduped []int64
		seen := make(map[int64]bool)
		if input.AuthorIDs.Value != nil {
			for _, authorID := range *input.AuthorIDs.Value {
				if authorID <= 0 {
					return Book{}, ErrAuthorNotFound
				}
				if !seen[authorID] {
					seen[authorID] = true
					deduped = append(deduped, authorID)
				}
			}
		}
		input.AuthorIDs.Value = &deduped
		if len(deduped) > 0 {
			found, err := s.authorRepo.GetByIDs(ctx, deduped)
			if err != nil {
				return Book{}, err
			}
			foundIDs := make(map[int64]bool)
			for _, author := range found {
				foundIDs[author.ID] = true
			}
			for _, authorID := range deduped {
				if !foundIDs[authorID] {
					return Book{}, ErrAuthorNotFound
				}
			}
		}
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
		_ = s.coverStore.Delete(*priorCoverKey)
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

func (s *Service) Delete(ctx context.Context, id int64) error {
	book, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repository.Delete(ctx, id); err != nil {
		return err
	}

	if book.CoverImageKey != nil {
		if err := s.coverStore.Delete(*book.CoverImageKey); err != nil {
			return fmt.Errorf("delete cover: %w", err)
		}
	}

	return nil
}

func (s *Service) withAuthorsAndCover(ctx context.Context, books []Book) ([]Book, error) {
	if len(books) == 0 {
		return books, nil
	}

	ids := make([]int64, len(books))
	for i, b := range books {
		ids[i] = b.ID
	}

	authorsByBook, err := s.authorRepo.ListByBookIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	for i, b := range books {
		if aa, ok := authorsByBook[b.ID]; ok {
			books[i].Authors = aa
		} else {
			books[i].Authors = []authors.Author{}
		}

		setCoverURL(&books[i])
	}

	return books, nil
}

func setCoverURL(book *Book) {
	if book.CoverImageKey == nil {
		book.CoverURL = nil
		return
	}

	url := "/book-covers/" + *book.CoverImageKey

	book.CoverURL = &url
}
