package api

import (
	"net/http"

	"bakku.dev/bookist/internal/reads"
)

func (s *Server) handleAPIListReads(w http.ResponseWriter, r *http.Request) {
	bookID, ok := parsePathID(w, r, "id", "book")
	if !ok {
		return
	}

	result, err := s.reads.ListByBookID(r.Context(), bookID)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPICreateRead(w http.ResponseWriter, r *http.Request) {
	bookID, ok := parsePathID(w, r, "id", "book")
	if !ok {
		return
	}

	var input reads.CreateReadRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	result, err := s.reads.Create(r.Context(), bookID, input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func (s *Server) handleAPIUpdateRead(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "read")
	if !ok {
		return
	}

	var input reads.UpdateReadRequest
	if !decodeJSON(w, r, &input, maxJSONBodySize) {
		return
	}

	result, err := s.reads.Update(r.Context(), id, input)
	if err != nil {
		s.writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPIDeleteRead(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePathID(w, r, "id", "read")
	if !ok {
		return
	}

	if err := s.reads.Delete(r.Context(), id); err != nil {
		s.writeError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
