package httpserver

import (
	"net/http"

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
