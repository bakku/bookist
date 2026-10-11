package web

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
)

type Server struct {
	books     *books.Service
	covers    *covers.Store
	templates TemplateSet
	static    *staticHandler
	logger    *slog.Logger
}

func New(books *books.Service, covers *covers.Store, logger *slog.Logger) (*Server, error) {
	templates, err := Templates()
	if err != nil {
		return nil, err
	}

	static, err := newStaticHandler(StaticFS())
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		covers:    covers,
		templates: templates,
		static:    static,
		logger:    logger,
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /book-covers/{key}", s.handleBookCover)
	mux.Handle("GET /static/", http.StripPrefix("/static/", s.static))
}

// serverError logs err with the request details and answers with a generic 500
// so internal details never reach clients.
func (s *Server) serverError(w http.ResponseWriter, r *http.Request, err error, message string) {
	s.logger.ErrorContext(r.Context(), message, "method", r.Method, "path", r.URL.Path, "error", err)
	http.Error(w, message, http.StatusInternalServerError)
}

// render executes the named template into a buffer before writing anything, so
// a template failure produces a clean 500 instead of a half-written 200 page.
func (s *Server) render(w http.ResponseWriter, r *http.Request, tmpl *template.Template, name string, data any) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
		s.serverError(w, r, err, "failed to render page")
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
