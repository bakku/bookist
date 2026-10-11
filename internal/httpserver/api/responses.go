package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"bakku.dev/bookist/internal/apperr"
	"bakku.dev/bookist/internal/covers"
)

const maxJSONBodySize = 1 << 20

// Errors produced by the HTTP layer itself rather than the domain packages.
var (
	errNotFound             = apperr.NotFound("not_found", "not found")
	errMethodNotAllowed     = apperr.Invalid("method_not_allowed", "method not allowed", "")
	errUnsupportedMediaType = apperr.Invalid("unsupported_media_type", "Content-Type must be application/json", "")
	errBodyTooLarge         = apperr.Invalid("body_too_large", "request body too large", "")
	errInvalidJSON          = apperr.Invalid("invalid_json", "invalid JSON body", "")
	errInternal             = &apperr.Error{Code: "internal_error", Message: "internal server error"}
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// writeError writes err as a structured JSON error. Errors that are not
// apperr.Errors are logged and reported as a generic 500 so internal details
// never reach clients.
func writeError(w http.ResponseWriter, err error) {
	appErr, ok := apperr.As(err)
	if !ok {
		slog.Error("api request failed", "error", err)
		appErr = errInternal
	}

	writeJSON(w, statusFor(appErr), errorResponse{Error: errorBody{
		Code:    appErr.Code,
		Message: appErr.Message,
		Field:   appErr.Field,
	}})
}

// statusFor is the single place that maps errors to HTTP status codes.
func statusFor(err *apperr.Error) int {
	switch {
	case errors.Is(err, covers.ErrTooLarge), errors.Is(err, errBodyTooLarge):
		return http.StatusRequestEntityTooLarge
	case errors.Is(err, covers.ErrUnsupportedMediaType), errors.Is(err, errUnsupportedMediaType):
		return http.StatusUnsupportedMediaType
	case errors.Is(err, errMethodNotAllowed):
		return http.StatusMethodNotAllowed
	}

	switch err.Kind {
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindInvalid:
		return http.StatusBadRequest
	case apperr.KindConflict:
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value any, maxBytes int64) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.EqualFold(mediaType, "application/json") {
		writeError(w, errUnsupportedMediaType)
		return false
	}

	if maxBytes > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return writeDecodeError(w, err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return writeDecodeError(w, err)
	}
	return true
}

func writeDecodeError(w http.ResponseWriter, err error) bool {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, errBodyTooLarge)
		return false
	}
	writeError(w, errInvalidJSON)
	return false
}
