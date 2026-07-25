package books

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/validation"
)

type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Search(ctx context.Context, query string) ([]Book, error)
	ListByListID(ctx context.Context, listID int64) ([]Book, error)
	SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error)
	Create(ctx context.Context, input CreateBookRequest) (Book, error)
	Delete(ctx context.Context, id int64) (*string, error)
}

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
	return s.withAuthors(ctx, books, err)
}

func (s *Service) Search(ctx context.Context, query string) ([]Book, error) {
	books, err := s.repository.Search(ctx, strings.TrimSpace(query))
	return s.withAuthors(ctx, books, err)
}

func (s *Service) ListByListID(ctx context.Context, listID int64) ([]Book, error) {
	books, err := s.repository.ListByListID(ctx, listID)
	return s.withAuthors(ctx, books, err)
}

func (s *Service) SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error) {
	books, err := s.repository.SearchByListID(ctx, listID, strings.TrimSpace(query))
	return s.withAuthors(ctx, books, err)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	coverKey, err := s.repository.Delete(ctx, id)
	if err != nil {
		return err
	}
	if coverKey != nil {
		if err := s.coverStore.Delete(*coverKey); err != nil {
			return fmt.Errorf("delete cover: %w", err)
		}
	}
	return nil
}

func (s *Service) withAuthors(ctx context.Context, books []Book, err error) ([]Book, error) {
	if err != nil {
		return nil, err
	}

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

func setCoverURL(book *Book) {
	if book.CoverImageKey == nil {
		book.CoverURL = nil
		return
	}
	url := "/book-covers/" + *book.CoverImageKey
	book.CoverURL = &url
}
