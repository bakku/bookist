package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
)

const maxCreateBookBodySize = 15 << 20

func (s *Server) handleAPIListBooks(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bookList)
}

func (s *Server) handleAPICreateBook(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCreateBookBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input books.CreateBookRequest
	if err := decoder.Decode(&input); err != nil {
		var tooLarge *http.MaxBytesError

		if errors.As(err, &tooLarge) {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

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

func (s *Server) handleAPIUpdateBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	var input books.UpdateBookRequest
	if !decodePatchJSON(w, r, &input, maxCreateBookBodySize) {
		return
	}

	book, err := s.books.Update(r.Context(), id, input)
	if err != nil {
		writeUpdateBookError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (s *Server) handleAPIDeleteBook(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	if err := s.books.Delete(r.Context(), id); err != nil {
		if errors.Is(err, books.ErrBookNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete book", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCreateBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, books.ErrTitleRequired) ||
		errors.Is(err, books.ErrAuthorNotFound) ||
		errors.Is(err, books.ErrInvalidFormat) ||
		errors.Is(err, books.ErrInvalidPurchasedAt) ||
		errors.Is(err, books.ErrInvalidPages) ||
		errors.Is(err, books.ErrInvalidCondition) ||
		errors.Is(err, books.ErrInvalidSeriesPosition) ||
		errors.Is(err, books.ErrInvalidPublishedYear) ||
		errors.Is(err, books.ErrInvalidPublishedMonth) ||
		errors.Is(err, books.ErrInvalidPublishedDay) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errors.Is(err, covers.ErrTooLarge) {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}

	if errors.Is(err, covers.ErrUnsupportedMediaType) {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	http.Error(w, "failed to create book", http.StatusInternalServerError)
}

func writeUpdateBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, books.ErrBookNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, books.ErrNoFieldsToUpdate) || errors.Is(err, books.ErrBlankOptionalString) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, covers.ErrTooLarge) {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}
	if errors.Is(err, covers.ErrUnsupportedMediaType) {
		http.Error(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	if errors.Is(err, books.ErrTitleRequired) ||
		errors.Is(err, books.ErrAuthorNotFound) ||
		errors.Is(err, books.ErrInvalidFormat) ||
		errors.Is(err, books.ErrInvalidPurchasedAt) ||
		errors.Is(err, books.ErrInvalidPages) ||
		errors.Is(err, books.ErrInvalidCondition) ||
		errors.Is(err, books.ErrInvalidSeriesPosition) ||
		errors.Is(err, books.ErrInvalidPublishedYear) ||
		errors.Is(err, books.ErrInvalidPublishedMonth) ||
		errors.Is(err, books.ErrInvalidPublishedDay) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "failed to update book", http.StatusInternalServerError)
}
