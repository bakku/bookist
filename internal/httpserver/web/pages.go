package web

import (
	"fmt"
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
	PageTitle   string
	HeaderTitle string
	BadgeText   string
	BadgeLabel  string
	LibraryNav  []navItemView
	UserListNav []navItemView
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

	bookLabel := "books"
	if len(bookList) == 1 {
		bookLabel = "book"
	}

	data := indexPageView{
		Layout: layoutView{
			PageTitle:   "Bookist Library",
			HeaderTitle: "Library",
			BadgeText:   fmt.Sprintf("%d", len(bookList)),
			BadgeLabel:  fmt.Sprintf("%d %s in library", len(bookList), bookLabel),
			LibraryNav: []navItemView{
				{Href: "/", Label: "Books", IconID: "bookshelf", Active: true},
				{Href: "/authors", Label: "Authors", IconID: "account-outline"},
			},
			UserListNav: []navItemView{
				{Href: "/lists/to-read", Label: "To Read", IconID: "format-list-bulleted"},
				{Href: "/lists/sci-fi-favorites", Label: "Sci-Fi Favorites", IconID: "format-list-bulleted"},
				{Href: "/lists/summer-reading", Label: "Summer Reading", IconID: "format-list-bulleted"},
			},
		},
		Books: bookList,
	}

	if err := s.templates.BooksIndex.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
