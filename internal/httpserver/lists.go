package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"bakku.dev/bookist/internal/lists"
)

func (s *Server) handleAPIListLists(w http.ResponseWriter, r *http.Request) {
	listList, err := s.lists.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list lists", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, listList)
}

func (s *Server) handleAPICreateList(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input lists.CreateListRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	list, err := s.lists.Create(r.Context(), input)
	if err != nil {
		writeCreateListError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, list)
}

func (s *Server) handleAPIListBooksInList(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("id")

	_, err := s.lists.GetByID(r.Context(), listID)
	if err != nil {
		if errors.Is(err, lists.ErrListNotFound) {
			http.Error(w, "list not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to get list", http.StatusInternalServerError)
		return
	}

	listBooks, err := s.books.ListByListID(r.Context(), listID)
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, listBooks)
}

func (s *Server) handleAPIAddBookToList(w http.ResponseWriter, r *http.Request) {
	listID := r.PathValue("id")

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input lists.AddBookToListRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.BookID) == "" {
		http.Error(w, "book_id is required", http.StatusBadRequest)
		return
	}

	err := s.lists.AddBookToList(r.Context(), listID, input.BookID)
	if err != nil {
		writeAddBookToListError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCreateListError(w http.ResponseWriter, err error) {
	if errors.Is(err, lists.ErrNameRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if errors.Is(err, lists.ErrNameConflict) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	http.Error(w, "failed to create list", http.StatusInternalServerError)
}

func writeAddBookToListError(w http.ResponseWriter, err error) {
	if errors.Is(err, lists.ErrListNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if errors.Is(err, lists.ErrBookNotFound) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if errors.Is(err, lists.ErrBookAlreadyInList) {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}

	http.Error(w, "failed to add book to list", http.StatusInternalServerError)
}
