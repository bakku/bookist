package reads

import (
	"context"
	"errors"
	"math"
	"strings"

	"bakku.dev/bookist/internal/validation"
)

var ErrBookNotFound = errors.New("book not found")
var ErrInvalidStartedAt = errors.New("started_at must be a date in YYYY-MM-DD format")
var ErrInvalidFinishedAt = errors.New("finished_at must be a date in YYYY-MM-DD format")
var ErrInvalidAbandonedAt = errors.New("abandoned_at must be a date in YYYY-MM-DD format")
var ErrConflictingTerminalDates = errors.New("finished_at and abandoned_at must not both be set")
var ErrFinishedBeforeStarted = errors.New("finished_at must not be before started_at")
var ErrAbandonedBeforeStarted = errors.New("abandoned_at must not be before started_at")
var ErrInvalidRating = errors.New("rating must be between 1 and 5 in increments of 0.5")

type Repository interface {
	Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error)
	ListByBookID(ctx context.Context, bookID int64) ([]Read, error)
}

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

	if input.FinishedAt != nil && input.AbandonedAt != nil {
		return Read{}, ErrConflictingTerminalDates
	}

	if input.StartedAt != nil && input.FinishedAt != nil && *input.FinishedAt < *input.StartedAt {
		return Read{}, ErrFinishedBeforeStarted
	}
	if input.StartedAt != nil && input.AbandonedAt != nil && *input.AbandonedAt < *input.StartedAt {
		return Read{}, ErrAbandonedBeforeStarted
	}

	if input.Rating != nil {
		rating := *input.Rating
		if math.IsNaN(rating) || math.IsInf(rating, 0) || rating < 1 || rating > 5 || math.Mod(rating*2, 1) != 0 {
			return Read{}, ErrInvalidRating
		}
	}

	if input.Notes != nil {
		notes := strings.TrimSpace(*input.Notes)
		if notes == "" {
			input.Notes = nil
		} else {
			input.Notes = &notes
		}
	}

	return s.repository.Create(ctx, bookID, input)
}

func (s *Service) ListByBookID(ctx context.Context, bookID int64) ([]Read, error) {
	return s.repository.ListByBookID(ctx, bookID)
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
