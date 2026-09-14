package web

import (
	"net/http"

	"bakku.dev/bookist/internal/books"
)

type navItemView struct {
	Href   string
	Label  string
	IconID string
	Active bool
}

type layoutView struct {
	PageTitle  string
	LibraryNav []navItemView
}

type indexPageView struct {
	Layout layoutView
	Books  []books.Book
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	bookList, err := s.books.List(r.Context())
	if err != nil {
		http.Error(w, "failed to list books", http.StatusInternalServerError)
		return
	}

	data := indexPageView{
		Layout: layoutView{
			PageTitle: "Bookist Library",
			LibraryNav: []navItemView{
				{Href: "/", Label: "Books", IconID: "books", Active: true},
			},
		},
		Books: bookList,
	}

	if err := s.templates.BooksIndex.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
