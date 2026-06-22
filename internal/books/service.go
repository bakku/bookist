package books

import (
	"context"
	"errors"
	"strings"
)

var ErrTitleRequired = errors.New("title is required")

type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Create(ctx context.Context, input CreateBookInput) (Book, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) List(ctx context.Context) ([]Book, error) {
	return s.repository.List(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateBookInput) (Book, error) {
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

	return s.repository.Create(ctx, input)
}
