package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"bakku.dev/bookist/internal/authors"
)

func (s *Server) handleAPIListAuthors(w http.ResponseWriter, r *http.Request) {
	authorList, err := s.authors.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		http.Error(w, "failed to list authors", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, authorList)
}

func (s *Server) handleAPICreateAuthor(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) handleAPIDeleteAuthor(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid author ID", http.StatusBadRequest)
		return
	}

	if err := s.authors.Delete(r.Context(), id); err != nil {
		if errors.Is(err, authors.ErrAuthorNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete author", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeCreateAuthorError(w http.ResponseWriter, err error) {
	if errors.Is(err, authors.ErrNameRequired) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "failed to create author", http.StatusInternalServerError)
}
