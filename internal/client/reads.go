package client

import (
	"context"
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/reads"
)

// ReadsService manages reads of books.
type ReadsService struct {
	client *Client
}

// List returns the reads recorded for a book.
func (s *ReadsService) List(ctx context.Context, bookID int64) ([]reads.Read, error) {
	var listed []reads.Read
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint("", "api", "books", strconv.FormatInt(bookID, 10), "reads"), nil, &listed, http.StatusOK)
	return listed, err
}

// Create records a read for a book.
func (s *ReadsService) Create(ctx context.Context, bookID int64, request reads.CreateReadRequest) (reads.Read, error) {
	var created reads.Read
	err := s.client.do(ctx, http.MethodPost, s.client.endpoint("", "api", "books", strconv.FormatInt(bookID, 10), "reads"), request, &created, http.StatusCreated)
	return created, err
}

func (s *ReadsService) Update(ctx context.Context, id int64, request reads.UpdateReadRequest) (reads.Read, error) {
	var updated reads.Read
	err := s.client.do(ctx, http.MethodPatch, s.client.endpoint("", "api", "reads", strconv.FormatInt(id, 10)), request, &updated, http.StatusOK)
	return updated, err
}

func (s *ReadsService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, s.client.endpoint("", "api", "reads", strconv.FormatInt(id, 10)), nil, nil, http.StatusNoContent)
}
