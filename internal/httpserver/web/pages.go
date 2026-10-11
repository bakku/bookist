package web

import (
	"net/http"
	"time"
)

type navItemView struct {
	Href   string
	Label  string
	Active bool
}

type layoutView struct {
	PageTitle  string
	PrimaryNav []navItemView
}

type shelfLinkView struct {
	Href   string
	Label  string
	Count  int
	Active bool
}

type readingStatsView struct {
	Year          int
	BooksFinished int
	PagesRead     string
}

type readStatus string

const (
	readStatusReading   readStatus = "reading"
	readStatusRead      readStatus = "read"
	readStatusUnread    readStatus = "unread"
	readStatusAbandoned readStatus = "abandoned"
)

func (s readStatus) Label() string {
	switch s {
	case readStatusReading:
		return "Reading"
	case readStatusRead:
		return "Read"
	case readStatusAbandoned:
		return "Abandoned"
	default:
		return "Unread"
	}
}

type bookRowView struct {
	Title     string
	Lang      string
	Series    string
	Authors   []string
	CoverURL  string
	CoverTone string
	Meta      string
	Lists     []string
	Rating    float64
	Status    readStatus
	ReadNote  string
	ReadAt    *time.Time
	AddedAt   time.Time
}

type libraryPageView struct {
	Layout  layoutView
	Shelves []shelfLinkView
	Lists   []shelfLinkView
	Stats   readingStatsView
	Books   []bookRowView
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	data := sampleLibrary()
	data.Layout = layoutView{
		PageTitle: "My Books · Bookist",
		PrimaryNav: []navItemView{
			{Href: "/", Label: "My Books", Active: true},
			{Href: "#", Label: "Authors"},
			{Href: "#", Label: "Lists"},
		},
	}

	if err := s.templates.BooksIndex.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
