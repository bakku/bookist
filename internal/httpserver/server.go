package httpserver

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/web"
)

type Server struct {
	books     *books.Service
	authors   *authors.Service
	lists     *lists.Service
	reads     *reads.Service
	templates *template.Template
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid ID")
	}
	return id, nil
}

func New(books *books.Service, authors *authors.Service, lists *lists.Service, reads *reads.Service) (*Server, error) {
	templates, err := web.Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		authors:   authors,
		lists:     lists,
		reads:     reads,
		templates: templates,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/books", s.handleAPIListBooks)
	mux.HandleFunc("POST /api/books", s.handleAPICreateBook)
	mux.HandleFunc("GET /api/books/{id}/reads", s.handleAPIListReads)
	mux.HandleFunc("POST /api/books/{id}/reads", s.handleAPICreateRead)
	mux.HandleFunc("GET /api/authors", s.handleAPIListAuthors)
	mux.HandleFunc("POST /api/authors", s.handleAPICreateAuthor)
	mux.HandleFunc("GET /api/lists", s.handleAPIListLists)
	mux.HandleFunc("POST /api/lists", s.handleAPICreateList)
	mux.HandleFunc("GET /api/lists/{id}/books", s.handleAPIListBooksInList)
	mux.HandleFunc("POST /api/lists/{id}/books", s.handleAPIAddBookToList)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))

	return mux
}
