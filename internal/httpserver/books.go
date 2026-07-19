package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"bakku.dev/bookist/internal/books"
)

func (s *Server) handleAPIListBooks(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bookList)
}

func (s *Server) handleAPICreateBook(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input books.CreateBookRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	book, err := s.books.Create(r.Context(), input)
	if err != nil {
		writeCreateBookError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, book)
}

func writeCreateBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, books.ErrTitleRequired) ||
		errors.Is(err, books.ErrAuthorNotFound) ||
		errors.Is(err, books.ErrInvalidFormat) ||
		errors.Is(err, books.ErrInvalidPurchasedAt) ||
		errors.Is(err, books.ErrInvalidPages) ||
		errors.Is(err, books.ErrInvalidPublishedYear) ||
		errors.Is(err, books.ErrInvalidPublishedMonth) ||
		errors.Is(err, books.ErrInvalidPublishedDay) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, books.ErrTitleConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	http.Error(w, "failed to create book", http.StatusInternalServerError)
}
