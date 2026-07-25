package httpserver

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"

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
	templates *template.Template
}

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid ID")
	}
	return id, nil
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
	mux.HandleFunc("DELETE /api/lists/{id}/books/{bookID}", s.handleAPIRemoveBookFromList)
	mux.HandleFunc("PATCH /api/reads/{id}", s.handleAPIUpdateRead)
	mux.HandleFunc("DELETE /api/reads/{id}", s.handleAPIDeleteRead)
	mux.HandleFunc("GET /book-covers/{key}", s.handleBookCover)

	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(web.StaticFS()))))

	return mux
}

func (s *Server) handleBookCover(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	file, err := s.covers.Open(key)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", covers.MediaType(key))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, key, info.ModTime(), file)
}
