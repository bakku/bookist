package books

import (
	"context"
	"errors"
	"strings"

	"bakku.dev/bookist/internal/authors"
)

var ErrTitleRequired = errors.New("title is required")
var ErrAuthorNotFound = errors.New("author not found")

type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Create(ctx context.Context, input CreateBookRequest) (Book, error)
}

type Service struct {
	repository Repository
	authorRepo authors.Repository
}

func NewService(repository Repository, authorRepo authors.Repository) *Service {
	return &Service{repository: repository, authorRepo: authorRepo}
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	books, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	if len(books) == 0 {
		return books, nil
	}

	ids := make([]string, len(books))
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

	seen := make(map[string]bool)
	var deduped []string
	for _, id := range input.AuthorIDs {
		id = strings.TrimSpace(id)
		if id != "" && !seen[id] {
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

		foundMap := make(map[string]bool)
		for _, a := range found {
			foundMap[a.ID] = true
		}
		for _, id := range input.AuthorIDs {
			if !foundMap[id] {
				return Book{}, ErrAuthorNotFound
			}
		}
	}

	book, err := s.repository.Create(ctx, input)
	if err != nil {
		return Book{}, err
	}

	if len(input.AuthorIDs) > 0 {
		book.Authors = found
	} else {
		book.Authors = []authors.Author{}
	}

	return book, nil
}
