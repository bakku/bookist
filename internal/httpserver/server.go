package httpserver

import (
	"html/template"
	"net/http"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/web"
)

type Server struct {
	books     *books.Service
	templates *template.Template
}

func New(books *books.Service) (*Server, error) {
	templates, err := web.Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		templates: templates,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/books", s.handleAPIListBooks)
	mux.HandleFunc("POST /api/books", s.handleAPICreateBook)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))
	return mux
}
