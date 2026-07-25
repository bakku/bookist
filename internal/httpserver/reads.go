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

func (s *Server) handleAPIUpdateRead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid read ID", http.StatusBadRequest)
		return
	}

	var input reads.UpdateReadRequest
	if !decodePatchJSON(w, r, &input, maxPatchBodySize) {
		return
	}

	result, err := s.reads.Update(r.Context(), id, input)
	if err != nil {
		writeUpdateReadError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleAPIDeleteRead(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid read ID", http.StatusBadRequest)
		return
	}

	if err := s.reads.Delete(r.Context(), id); err != nil {
		if errors.Is(err, reads.ErrReadNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete read", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
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

func writeUpdateReadError(w http.ResponseWriter, err error) {
	if errors.Is(err, reads.ErrReadNotFound) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if errors.Is(err, reads.ErrNoFieldsToUpdate) ||
		errors.Is(err, reads.ErrBlankOptionalString) ||
		errors.Is(err, reads.ErrInvalidStartedAt) ||
		errors.Is(err, reads.ErrInvalidFinishedAt) ||
		errors.Is(err, reads.ErrInvalidAbandonedAt) ||
		errors.Is(err, reads.ErrConflictingTerminalDates) ||
		errors.Is(err, reads.ErrFinishedBeforeStarted) ||
		errors.Is(err, reads.ErrAbandonedBeforeStarted) ||
		errors.Is(err, reads.ErrInvalidRating) {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Error(w, "failed to update read", http.StatusInternalServerError)
}
