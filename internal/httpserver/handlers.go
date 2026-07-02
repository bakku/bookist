package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	data := struct {
		Books []books.Book
	}{Books: bookList}

	if err := s.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}

func (s *Server) handleAPIListBooks(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, bookList)
}

func (s *Server) handleAPICreateBook(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

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

func (s *Server) handleAPIListAuthors(w http.ResponseWriter, r *http.Request) {
	authorList, err := s.authors.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list authors", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authorList)
}

func (s *Server) handleAPICreateAuthor(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input authors.CreateAuthorRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	author, err := s.authors.Create(r.Context(), input)
	if err != nil {
		writeCreateAuthorError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, author)
}

func writeCreateBookError(w http.ResponseWriter, err error) {
	if errors.Is(err, books.ErrTitleRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, books.ErrAuthorNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "failed to create book", http.StatusInternalServerError)
}

func writeCreateAuthorError(w http.ResponseWriter, err error) {
	if errors.Is(err, authors.ErrNameRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "failed to create author", http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
