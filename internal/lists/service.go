package lists

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

func (s *Service) Update(ctx context.Context, id int64, input UpdateListRequest) (List, error) {
	if !input.Name.Present && !input.Description.Present {
		return List{}, ErrNoFieldsToUpdate
	}
	if _, err := s.repository.GetByID(ctx, id); err != nil {
		return List{}, err
	}

	if input.Name.Present {
		if input.Name.Value == nil {
			return List{}, ErrNameRequired
		}
		name := strings.TrimSpace(*input.Name.Value)
		if name == "" {
			return List{}, ErrNameRequired
		}
		input.Name.Value = &name

		exists, err := s.repository.NameExistsExcludingID(ctx, name, id)
		if err != nil {
			return List{}, err
		}
		if exists {
			return List{}, ErrNameConflict
		}
	}

	if input.Description.Present && input.Description.Value != nil {
		description := strings.TrimSpace(*input.Description.Value)
		if description == "" {
			return List{}, ErrDescriptionRequired
		}
		input.Description.Value = &description
	}

	return s.repository.Update(ctx, id, input)
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
