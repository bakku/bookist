package authors

import (
	"context"
	"errors"
	"strings"
)

var ErrNameRequired = errors.New("name is required")
var ErrNameConflict = errors.New("an author with this name already exists")

type Repository interface {
	Create(ctx context.Context, input CreateAuthorRequest) (Author, error)
	List(ctx context.Context) ([]Author, error)
	NameExists(ctx context.Context, name string) (bool, error)
	GetByIDs(ctx context.Context, ids []string) ([]Author, error)
	ListByBookIDs(ctx context.Context, bookIDs []string) (map[string][]Author, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, input CreateAuthorRequest) (Author, error) {
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return Author{}, ErrNameRequired
	}

	exists, err := s.repository.NameExists(ctx, input.Name)
	if err != nil {
		return Author{}, err
	}

	if exists {
		return Author{}, ErrNameConflict
	}

	return s.repository.Create(ctx, input)
}

func (s *Service) List(ctx context.Context) ([]Author, error) {
	return s.repository.List(ctx)
}

func (s *Service) GetByIDs(ctx context.Context, ids []string) ([]Author, error) {
	return s.repository.GetByIDs(ctx, ids)
}

func (s *Service) ListByBookIDs(ctx context.Context, bookIDs []string) (map[string][]Author, error) {
	return s.repository.ListByBookIDs(ctx, bookIDs)
}
