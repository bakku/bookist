package api

import (
	"net/http"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

type Server struct {
	books   *books.Service
	authors *authors.Service
	lists   *lists.Service
	reads   *reads.Service
}

func New(books *books.Service, authors *authors.Service, lists *lists.Service, reads *reads.Service) *Server {
	return &Server{
		books:   books,
		authors: authors,
		lists:   lists,
		reads:   reads,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
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
}
