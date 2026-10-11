package api

import (
	"net/http"

	"bakku.dev/bookist/internal/books"
)

const maxCreateBookBodySize = 15 << 20

func (s *Server) handleAPIListBooks(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, bookList)
}

func (s *Server) handleAPIGetBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "book")
	if !ok {
		return
	}

	book, err := s.books.GetByID(r.Context(), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, book)
}

func (s *Server) handleAPICreateBook(w http.ResponseWriter, r *http.Request) {
	var input books.CreateBookRequest
	if !decodeJSON(w, r, &input, maxCreateBookBodySize) {
		return
	}

	book, err := s.books.Create(r.Context(), input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, book)
}

func (s *Server) handleAPIUpdateBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "book")
	if !ok {
		return
	}

	var input books.UpdateBookRequest
	if !decodeJSON(w, r, &input, maxCreateBookBodySize) {
		return
	}

	book, err := s.books.Update(r.Context(), id, input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, book)
}

func (s *Server) handleAPIDeleteBook(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "book")
	if !ok {
		return
	}

	if err := s.books.Delete(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
