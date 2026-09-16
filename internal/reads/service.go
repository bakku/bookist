package reads

import (
	"context"
	"math"
	"strings"

	"bakku.dev/bookist/internal/validation"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error) {
	startedAt, err := normalizeDate(input.StartedAt, ErrInvalidStartedAt)
	if err != nil {
		return Read{}, err
	}
	input.StartedAt = startedAt

	finishedAt, err := normalizeDate(input.FinishedAt, ErrInvalidFinishedAt)
	if err != nil {
		return Read{}, err
	}
	input.FinishedAt = finishedAt

	abandonedAt, err := normalizeDate(input.AbandonedAt, ErrInvalidAbandonedAt)
	if err != nil {
		return Read{}, err
	}
	input.AbandonedAt = abandonedAt

	if input.Notes != nil {
		notes := strings.TrimSpace(*input.Notes)
		if notes == "" {
			input.Notes = nil
		} else {
			input.Notes = &notes
		}
	}
	if err := validateRead(input); err != nil {
		return Read{}, err
	}

	return s.repository.Create(ctx, bookID, input)
}

func (s *Service) GetByID(ctx context.Context, id int64) (Read, error) {
	return s.repository.GetByID(ctx, id)
}

func (s *Service) ListByBookID(ctx context.Context, bookID int64) ([]Read, error) {
	return s.repository.ListByBookID(ctx, bookID)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateReadRequest) (Read, error) {
	if !input.StartedAt.Present && !input.FinishedAt.Present && !input.AbandonedAt.Present &&
		!input.Rating.Present && !input.Notes.Present {
		return Read{}, ErrNoFieldsToUpdate
	}

	current, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return Read{}, err
	}

	if input.StartedAt.Present {
		input.StartedAt.Value, err = normalizeUpdateString(input.StartedAt.Value)
		if err != nil {
			return Read{}, err
		}
	}
	if input.FinishedAt.Present {
		input.FinishedAt.Value, err = normalizeUpdateString(input.FinishedAt.Value)
		if err != nil {
			return Read{}, err
		}
	}
	if input.AbandonedAt.Present {
		input.AbandonedAt.Value, err = normalizeUpdateString(input.AbandonedAt.Value)
		if err != nil {
			return Read{}, err
		}
	}
	if input.Notes.Present {
		input.Notes.Value, err = normalizeUpdateString(input.Notes.Value)
		if err != nil {
			return Read{}, err
		}
	}

	merged := CreateReadRequest{
		StartedAt: current.StartedAt, FinishedAt: current.FinishedAt, AbandonedAt: current.AbandonedAt,
		Rating: current.Rating, Notes: current.Notes,
	}
	if input.StartedAt.Present {
		merged.StartedAt = input.StartedAt.Value
	}
	if input.FinishedAt.Present {
		merged.FinishedAt = input.FinishedAt.Value
	}
	if input.AbandonedAt.Present {
		merged.AbandonedAt = input.AbandonedAt.Value
	}
	if input.Rating.Present {
		merged.Rating = input.Rating.Value
	}
	if input.Notes.Present {
		merged.Notes = input.Notes.Value
	}
	if err := validateRead(merged); err != nil {
		return Read{}, err
	}

	return s.repository.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repository.Delete(ctx, id)
}

func normalizeDate(value *string, invalidError error) (*string, error) {
	if value == nil {
		return nil, nil
	}

	date := strings.TrimSpace(*value)
	if date == "" {
		return nil, nil
	}
	if !validation.IsCalendarDate(date) {
		return nil, invalidError
	}

	return &date, nil
}

func normalizeUpdateString(value *string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, ErrBlankOptionalString
	}
	return &normalized, nil
}

func validateRead(input CreateReadRequest) error {
	if input.StartedAt != nil && !validation.IsCalendarDate(*input.StartedAt) {
		return ErrInvalidStartedAt
	}
	if input.FinishedAt != nil && !validation.IsCalendarDate(*input.FinishedAt) {
		return ErrInvalidFinishedAt
	}
	if input.AbandonedAt != nil && !validation.IsCalendarDate(*input.AbandonedAt) {
		return ErrInvalidAbandonedAt
	}
	if input.FinishedAt != nil && input.AbandonedAt != nil {
		return ErrConflictingTerminalDates
	}
	if input.StartedAt != nil && input.FinishedAt != nil && *input.FinishedAt < *input.StartedAt {
		return ErrFinishedBeforeStarted
	}
	if input.StartedAt != nil && input.AbandonedAt != nil && *input.AbandonedAt < *input.StartedAt {
		return ErrAbandonedBeforeStarted
	}
	if input.Rating != nil {
		rating := *input.Rating
		if math.IsNaN(rating) || math.IsInf(rating, 0) || rating < 1 || rating > 5 || math.Mod(rating*2, 1) != 0 {
			return ErrInvalidRating
		}
	}
	return nil
}
