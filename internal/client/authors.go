package client

import (
	"context"
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/authors"
)

// AuthorsService manages authors.
type AuthorsService struct {
	client *Client
}

// List returns the authors matching query, or all authors when query is blank.
func (s *AuthorsService) List(ctx context.Context, query string) ([]authors.Author, error) {
	var listed []authors.Author
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint(query, "api", "authors"), nil, &listed, http.StatusOK)
	return listed, err
}

// Get returns the author with the given ID.
func (s *AuthorsService) Get(ctx context.Context, id int64) (authors.Author, error) {
	var got authors.Author
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint("", "api", "authors", strconv.FormatInt(id, 10)), nil, &got, http.StatusOK)
	return got, err
}

func (s *AuthorsService) Create(ctx context.Context, request authors.CreateAuthorRequest) (authors.Author, error) {
	var created authors.Author
	err := s.client.do(ctx, http.MethodPost, s.client.endpoint("", "api", "authors"), request, &created, http.StatusCreated)
	return created, err
}

func (s *AuthorsService) Update(ctx context.Context, id int64, request authors.UpdateAuthorRequest) (authors.Author, error) {
	var updated authors.Author
	err := s.client.do(ctx, http.MethodPatch, s.client.endpoint("", "api", "authors", strconv.FormatInt(id, 10)), request, &updated, http.StatusOK)
	return updated, err
}

func (s *AuthorsService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, s.client.endpoint("", "api", "authors", strconv.FormatInt(id, 10)), nil, nil, http.StatusNoContent)
}
