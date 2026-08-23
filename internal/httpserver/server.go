package httpserver

import (
	"net/http"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver/api"
	httpweb "bakku.dev/bookist/internal/httpserver/web"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

type Server struct {
	handler http.Handler
}

func New(bookService *books.Service, authorService *authors.Service, listService *lists.Service, readService *reads.Service, coverStore *covers.Store) (*Server, error) {
	webServer, err := httpweb.New(bookService, coverStore)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	api.New(bookService, authorService, listService, readService).RegisterRoutes(mux)
	webServer.RegisterRoutes(mux)

	return &Server{handler: mux}, nil
}

func (s *Server) Handler() http.Handler {
	return s.handler
}
