package web

import (
	"log/slog"
	"net/http"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
)

type Server struct {
	books     *books.Service
	covers    *covers.Store
	templates TemplateSet
	logger    *slog.Logger
}

func New(books *books.Service, covers *covers.Store, logger *slog.Logger) (*Server, error) {
	templates, err := Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		covers:    covers,
		templates: templates,
		logger:    logger,
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /book-covers/{key}", s.handleBookCover)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS()))))
}

// serverError logs err with the request details and answers with a generic 500
// so internal details never reach clients.
func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error, message string) {
	s.logger.ErrorContext(r.Context(), message, "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, message, http.StatusInternalServerError)
}
