package httpserver

import (
	"html/template"
	"net/http"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/web"
)

type Server struct {
	books     *books.Service
	authors   *authors.Service
	lists     *lists.Service
	templates *template.Template
}

func New(books *books.Service, authors *authors.Service, lists *lists.Service) (*Server, error) {
	templates, err := web.Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		authors:   authors,
		lists:     lists,
		templates: templates,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/books", s.handleAPIListBooks)
	mux.HandleFunc("POST /api/books", s.handleAPICreateBook)
	mux.HandleFunc("GET /api/authors", s.handleAPIListAuthors)
	mux.HandleFunc("POST /api/authors", s.handleAPICreateAuthor)
	mux.HandleFunc("GET /api/lists", s.handleAPIListLists)
	mux.HandleFunc("POST /api/lists", s.handleAPICreateList)
	mux.HandleFunc("GET /api/lists/{id}/books", s.handleAPIListBooksInList)
	mux.HandleFunc("POST /api/lists/{id}/books", s.handleAPIAddBookToList)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))

	return mux
}
