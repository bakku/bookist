package api

import (
	"net/http"

	"bakku.dev/bookist/internal/authors"
)

func (s *Server) handleAPIListAuthors(w http.ResponseWriter, r *http.Request) {
	authorList, err := s.authors.Search(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, authorList)
}

func (s *Server) handleAPIGetAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "author")
	if !ok {
		return
	}

	author, err := s.authors.GetByID(r.Context(), id)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, author)
}

func (s *Server) handleAPICreateAuthor(w http.ResponseWriter, r *http.Request) {
	var input authors.CreateAuthorRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	author, err := s.authors.Create(r.Context(), input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, author)
}

func (s *Server) handleAPIUpdateAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "author")
	if !ok {
		return
	}

	var input authors.UpdateAuthorRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	author, err := s.authors.Update(r.Context(), id, input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, author)
}

func (s *Server) handleAPIDeleteAuthor(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "author")
	if !ok {
		return
	}

	if err := s.authors.Delete(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
