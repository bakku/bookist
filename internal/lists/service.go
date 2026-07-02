package lists

import (
	"context"
	"errors"
	"strings"

	"bakku.dev/bookist/internal/books"
)

var ErrNameRequired = errors.New("name is required")
var ErrListNotFound = errors.New("list not found")
var ErrBookNotFound = errors.New("book not found")
var ErrBookAlreadyInList = errors.New("book is already in this list")

type Repository interface {
	Create(ctx context.Context, input CreateListRequest) (List, error)
	List(ctx context.Context) ([]List, error)
	GetByID(ctx context.Context, id string) (List, error)
	AddBookToList(ctx context.Context, listID, bookID string) error
	ListBooks(ctx context.Context, listID string) ([]books.Book, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateListRequest) (List, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return List{}, ErrNameRequired
	}

	if input.Description != nil {
		desc := strings.TrimSpace(*input.Description)
		if desc == "" {
			input.Description = nil
		} else {
			input.Description = &desc
		}
	}

	return s.repository.Create(ctx, input)
}

func (s *Service) List(ctx context.Context) ([]List, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id string) (List, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) AddBookToList(ctx context.Context, listID, bookID string) error {
	return s.repository.AddBookToList(ctx, listID, bookID)
}

func (s *Service) ListBooks(ctx context.Context, listID string) ([]books.Book, error) {
	return s.repository.ListBooks(ctx, listID)
}
