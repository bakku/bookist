package client

import (
	"context"
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
)

// ListsService manages book lists and their books.
type ListsService struct {
	client *Client
}

// List returns the lists matching query, or all lists when query is blank.
func (s *ListsService) List(ctx context.Context, query string) ([]lists.List, error) {
	var listed []lists.List
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint(query, "api", "lists"), nil, &listed, http.StatusOK)
	return listed, err
}

func (s *ListsService) Create(ctx context.Context, request lists.CreateListRequest) (lists.List, error) {
	var created lists.List
	err := s.client.do(ctx, http.MethodPost, s.client.endpoint("", "api", "lists"), request, &created, http.StatusCreated)
	return created, err
}

func (s *ListsService) Update(ctx context.Context, id int64, request lists.UpdateListRequest) (lists.List, error) {
	var updated lists.List
	err := s.client.do(ctx, http.MethodPatch, s.client.endpoint("", "api", "lists", strconv.FormatInt(id, 10)), request, &updated, http.StatusOK)
	return updated, err
}

func (s *ListsService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, s.client.endpoint("", "api", "lists", strconv.FormatInt(id, 10)), nil, nil, http.StatusNoContent)
}

// ListBooks returns the books in a list matching query, or all of the list's
// books when query is blank.
func (s *ListsService) ListBooks(ctx context.Context, listID int64, query string) ([]books.Book, error) {
	var listed []books.Book
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint(query, "api", "lists", strconv.FormatInt(listID, 10), "books"), nil, &listed, http.StatusOK)
	return listed, err
}

func (s *ListsService) AddBook(ctx context.Context, listID, bookID int64) error {
	request := lists.AddBookToListRequest{BookID: bookID}
	return s.client.do(ctx, http.MethodPost, s.client.endpoint("", "api", "lists", strconv.FormatInt(listID, 10), "books"), request, nil, http.StatusNoContent)
}

func (s *ListsService) RemoveBook(ctx context.Context, listID, bookID int64) error {
	endpoint := s.client.endpoint("", "api", "lists", strconv.FormatInt(listID, 10), "books", strconv.FormatInt(bookID, 10))
	return s.client.do(ctx, http.MethodDelete, endpoint, nil, nil, http.StatusNoContent)
}
