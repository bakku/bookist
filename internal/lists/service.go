package lists

import (
	"context"
	"strings"
)

type Repository interface {
	Create(ctx context.Context, input CreateListRequest) (List, error)
	List(ctx context.Context) ([]List, error)
	Search(ctx context.Context, query string) ([]List, error)
	NameExists(ctx context.Context, name string) (bool, error)
	GetByID(ctx context.Context, id int64) (List, error)
	Delete(ctx context.Context, id int64) error
	AddBookToList(ctx context.Context, listID, bookID int64) error
	RemoveBookFromList(ctx context.Context, listID, bookID int64) error
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
	exists, err := s.repository.NameExists(ctx, input.Name)
	if err != nil {
		return List{}, err
	}
	if exists {
		return List{}, ErrNameConflict
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

func (s *Service) Search(ctx context.Context, query string) ([]List, error) {
	return s.repository.Search(ctx, strings.TrimSpace(query))
}

func (s *Service) GetByID(ctx context.Context, id int64) (List, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}

func (s *Service) AddBookToList(ctx context.Context, listID, bookID int64) error {
	return s.repository.AddBookToList(ctx, listID, bookID)
}

func (s *Service) RemoveBookFromList(ctx context.Context, listID, bookID int64) error {
	return s.repository.RemoveBookFromList(ctx, listID, bookID)
}
