package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	"bakku.dev/bookist/internal/reads"
)

func (s *Server) handleAPIListReads(w http.ResponseWriter, r *http.Request) {
	bookID, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	result, err := s.reads.ListByBookID(r.Context(), bookID)
	if err != nil {
		if errors.Is(err, reads.ErrBookNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "failed to list reads", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPICreateRead(w http.ResponseWriter, r *http.Request) {
	bookID, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid book ID", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var input reads.CreateReadRequest
	if err := decoder.Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	result, err := s.reads.Create(r.Context(), bookID, input)
	if err != nil {
		writeCreateReadError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, result)
}

func writeCreateReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, reads.ErrBookNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if errors.Is(err, reads.ErrInvalidStartedAt) ||
		errors.Is(err, reads.ErrInvalidFinishedAt) ||
		errors.Is(err, reads.ErrInvalidAbandonedAt) ||
		errors.Is(err, reads.ErrConflictingTerminalDates) ||
		errors.Is(err, reads.ErrFinishedBeforeStarted) ||
		errors.Is(err, reads.ErrAbandonedBeforeStarted) ||
		errors.Is(err, reads.ErrInvalidRating) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	http.Error(w, "failed to create read", http.StatusInternalServerError)
}
