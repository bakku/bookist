package web

import (
	"net/http"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
)

type Server struct {
	books     *books.Service
	covers    *covers.Store
	templates TemplateSet
}

func New(books *books.Service, covers *covers.Store) (*Server, error) {
	templates, err := Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		covers:    covers,
		templates: templates,
	}, nil
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /book-covers/{key}", s.handleBookCover)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(StaticFS()))))
}
