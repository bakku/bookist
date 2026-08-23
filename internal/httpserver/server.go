package httpserver

import (
	"net/http"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/web"
)

type Server struct {
	books     *books.Service
	authors   *authors.Service
	lists     *lists.Service
	reads     *reads.Service
	covers    *covers.Store
	templates web.TemplateSet
}

func New(books *books.Service, authors *authors.Service, lists *lists.Service, reads *reads.Service, covers *covers.Store) (*Server, error) {
	templates, err := web.Templates()
	if err != nil {
		return nil, err
	}

	return &Server{
		books:     books,
		authors:   authors,
		lists:     lists,
		reads:     reads,
		covers:    covers,
		templates: templates,
	}, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /api/books", s.handleAPIListBooks)
	mux.HandleFunc("POST /api/books", s.handleAPICreateBook)
	mux.HandleFunc("DELETE /api/books/{id}", s.handleAPIDeleteBook)

	mux.HandleFunc("GET /api/books/{id}/reads", s.handleAPIListReads)
	mux.HandleFunc("POST /api/books/{id}/reads", s.handleAPICreateRead)

	mux.HandleFunc("GET /api/authors", s.handleAPIListAuthors)
	mux.HandleFunc("POST /api/authors", s.handleAPICreateAuthor)
	mux.HandleFunc("DELETE /api/authors/{id}", s.handleAPIDeleteAuthor)

	mux.HandleFunc("GET /api/lists", s.handleAPIListLists)
	mux.HandleFunc("POST /api/lists", s.handleAPICreateList)
	mux.HandleFunc("DELETE /api/lists/{id}", s.handleAPIDeleteList)
	mux.HandleFunc("GET /api/lists/{id}/books", s.handleAPIListBooksInList)
	mux.HandleFunc("POST /api/lists/{id}/books", s.handleAPIAddBookToList)
	mux.HandleFunc("DELETE /api/lists/{id}/books/{bookID}", s.handleAPIRemoveBookFromList)

	mux.HandleFunc("DELETE /api/reads/{id}", s.handleAPIDeleteRead)

	mux.HandleFunc("GET /book-covers/{key}", s.handleBookCover)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))

	return mux
}
