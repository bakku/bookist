package client

import (
	"context"
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/books"
)

// BooksService manages books.
type BooksService struct {
	client *Client
}

// List returns the books matching query, or all books when query is blank.
func (s *BooksService) List(ctx context.Context, query string) ([]books.Book, error) {
	var listed []books.Book
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint(query, "api", "books"), nil, &listed, http.StatusOK)
	return listed, err
}

// Get returns the book with the given ID.
func (s *BooksService) Get(ctx context.Context, id int64) (books.Book, error) {
	var got books.Book
	err := s.client.do(ctx, http.MethodGet, s.client.endpoint("", "api", "books", strconv.FormatInt(id, 10)), nil, &got, http.StatusOK)
	return got, err
}

func (s *BooksService) Create(ctx context.Context, request books.CreateBookRequest) (books.Book, error) {
	var created books.Book
	err := s.client.do(ctx, http.MethodPost, s.client.endpoint("", "api", "books"), request, &created, http.StatusCreated)
	return created, err
}

func (s *BooksService) Update(ctx context.Context, id int64, request books.UpdateBookRequest) (books.Book, error) {
	var updated books.Book
	err := s.client.do(ctx, http.MethodPatch, s.client.endpoint("", "api", "books", strconv.FormatInt(id, 10)), request, &updated, http.StatusOK)
	return updated, err
}

func (s *BooksService) Delete(ctx context.Context, id int64) error {
	return s.client.do(ctx, http.MethodDelete, s.client.endpoint("", "api", "books", strconv.FormatInt(id, 10)), nil, nil, http.StatusNoContent)
}
