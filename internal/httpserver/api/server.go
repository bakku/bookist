package api

import (
	"log/slog"
	"net/http"
	"strings"

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
	logger  *slog.Logger
}

func New(books *books.Service, authors *authors.Service, lists *lists.Service, reads *reads.Service, logger *slog.Logger) *Server {
	return &Server{
		books:   books,
		authors: authors,
		lists:   lists,
		reads:   reads,
		logger:  logger,
	}
}

func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/books", s.handleAPIListBooks)
	mux.HandleFunc("POST /api/books", s.handleAPICreateBook)
	mux.HandleFunc("PATCH /api/books/{id}", s.handleAPIUpdateBook)
	mux.HandleFunc("DELETE /api/books/{id}", s.handleAPIDeleteBook)

	mux.HandleFunc("GET /api/books/{id}/reads", s.handleAPIListReads)
	mux.HandleFunc("POST /api/books/{id}/reads", s.handleAPICreateRead)

	mux.HandleFunc("GET /api/authors", s.handleAPIListAuthors)
	mux.HandleFunc("POST /api/authors", s.handleAPICreateAuthor)
	mux.HandleFunc("PATCH /api/authors/{id}", s.handleAPIUpdateAuthor)
	mux.HandleFunc("DELETE /api/authors/{id}", s.handleAPIDeleteAuthor)

	mux.HandleFunc("GET /api/lists", s.handleAPIListLists)
	mux.HandleFunc("POST /api/lists", s.handleAPICreateList)
	mux.HandleFunc("PATCH /api/lists/{id}", s.handleAPIUpdateList)
	mux.HandleFunc("DELETE /api/lists/{id}", s.handleAPIDeleteList)
	mux.HandleFunc("GET /api/lists/{id}/books", s.handleAPIListBooksInList)
	mux.HandleFunc("POST /api/lists/{id}/books", s.handleAPIAddBookToList)
	mux.HandleFunc("DELETE /api/lists/{id}/books/{book_id}", s.handleAPIRemoveBookFromList)

	mux.HandleFunc("PATCH /api/reads/{id}", s.handleAPIUpdateRead)
	mux.HandleFunc("DELETE /api/reads/{id}", s.handleAPIDeleteRead)

	mux.HandleFunc("/api/", handleAPIFallback(mux))
}

// handleAPIFallback answers requests that no API route claims. It returns 405
// when the path exists for other methods and a JSON 404 otherwise.
func handleAPIFallback(mux *http.ServeMux) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var allowed []string
		for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete} {
			probe := r.Clone(r.Context())
			probe.Method = method
			if _, pattern := mux.Handler(probe); pattern != "" && pattern != "/api/" {
				allowed = append(allowed, method)
			}
		}

		if len(allowed) > 0 {
			w.Header().Set("Allow", strings.Join(allowed, ", "))
			writeAppError(w, errMethodNotAllowed)
			return
		}

		writeAppError(w, errNotFound)
	}
}
