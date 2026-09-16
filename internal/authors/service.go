package authors

import (
	"context"
	"strings"
)

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

	return s.repository.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateAuthorRequest) (Author, error) {
	if !input.Name.Present {
		return Author{}, ErrNoFieldsToUpdate
	}
	if input.Name.Value == nil {
		return Author{}, ErrNameRequired
	}

	name := strings.TrimSpace(*input.Name.Value)
	if name == "" {
		return Author{}, ErrNameRequired
	}
	input.Name.Value = &name

	return s.repository.Update(ctx, id, input)
}

func (s *Service) List(ctx context.Context) ([]Author, error) {
	return s.repository.List(ctx)
}

func (s *Service) Search(ctx context.Context, query string) ([]Author, error) {
	return s.repository.Search(ctx, strings.TrimSpace(query))
}

func (s *Service) GetByIDs(ctx context.Context, ids []int64) ([]Author, error) {
	return s.repository.GetByIDs(ctx, ids)
}

func (s *Service) ListByBookIDs(ctx context.Context, bookIDs []int64) (map[int64][]Author, error) {
	return s.repository.ListByBookIDs(ctx, bookIDs)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}
