package api

import (
	"net/http"

	"bakku.dev/bookist/internal/apperr"
	"bakku.dev/bookist/internal/lists"
)

var errBookIDRequired = apperr.Invalid("book_id_required", "book_id is required", "book_id")

func (s *Server) handleAPIListLists(w http.ResponseWriter, r *http.Request) {
	listList, err := s.lists.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, listList)
}

func (s *Server) handleAPICreateList(w http.ResponseWriter, r *http.Request) {
	var input lists.CreateListRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	list, err := s.lists.Create(r.Context(), input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, list)
}

func (s *Server) handleAPIUpdateList(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "list")
	if !ok {
		return
	}

	var input lists.UpdateListRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	list, err := s.lists.Update(r.Context(), id, input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) handleAPIDeleteList(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "list")
	if !ok {
		return
	}

	if err := s.lists.Delete(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIListBooksInList(w http.ResponseWriter, r *http.Request) {
	listID, ok := parsePathID(w, r, "id", "list")
	if !ok {
		return
	}

	listBooks, err := s.books.SearchByListID(r.Context(), listID, r.URL.Query().Get("q"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, listBooks)
}

func (s *Server) handleAPIAddBookToList(w http.ResponseWriter, r *http.Request) {
	listID, ok := parsePathID(w, r, "id", "list")
	if !ok {
		return
	}

	var input lists.AddBookToListRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	if input.BookID <= 0 {
		s.writeError(w, r, errBookIDRequired)
		return
	}

	if err := s.lists.AddBookToList(r.Context(), listID, input.BookID); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAPIRemoveBookFromList(w http.ResponseWriter, r *http.Request) {
	listID, ok := parsePathID(w, r, "id", "list")
	if !ok {
		return
	}

	bookID, ok := parsePathID(w, r, "book_id", "book")
	if !ok {
		return
	}

	if err := s.lists.RemoveBookFromList(r.Context(), listID, bookID); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
