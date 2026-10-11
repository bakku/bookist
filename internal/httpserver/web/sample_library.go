package web

import (
	"fmt"
	"time"
)

// sampleLibrary is placeholder content for the My Books page. It stands in
// for real data until the page is wired to the books, reads, and lists
// backends.
func sampleLibrary() libraryPageView {
	return libraryPageView{
		Shelves: []shelfLinkView{
			{Href: "#", Label: "All", Count: 14, Active: true},
			{Href: "#", Label: "Reading", Count: 2},
			{Href: "#", Label: "Read", Count: 7},
			{Href: "#", Label: "Unread", Count: 4},
			{Href: "#", Label: "Abandoned", Count: 1},
		},
		Lists: []shelfLinkView{
			{Href: "#", Label: "Favourites", Count: 3},
			{Href: "#", Label: "Book club 2026", Count: 3},
			{Href: "#", Label: "To re-read", Count: 2},
			{Href: "#", Label: "Lent out", Count: 1},
		},
		Stats: readingStatsView{Year: 2026, BooksFinished: 4, PagesRead: "1,792"},
		Books: []bookRowView{
			{
				Title:    "Dune",
				Series:   "Dune #1",
				Authors:  []string{"Frank Herbert"},
				CoverURL: openLibraryCover(14565843),
				Meta:     "Paperback · 658 pages · 1965",
				Lists:    []string{"Book club 2026"},
				Status:   readStatusReading,
				ReadNote: "Started",
				ReadAt:   sampleDatePtr(2026, time.September, 28),
				AddedAt:  sampleDate(2026, time.September, 25),
			},
			{
				Title:    "The Left Hand of Darkness",
				Series:   "Hainish Cycle #6",
				Authors:  []string{"Ursula K. Le Guin"},
				CoverURL: openLibraryCover(284550),
				Meta:     "Paperback · 304 pages · 1969",
				Status:   readStatusReading,
				ReadNote: "Started",
				ReadAt:   sampleDatePtr(2026, time.October, 2),
				AddedAt:  sampleDate(2026, time.August, 30),
			},
			{
				Title:    "Project Hail Mary",
				Authors:  []string{"Andy Weir"},
				CoverURL: openLibraryCover(12455001),
				Meta:     "Hardback · 476 pages · 2021",
				Lists:    []string{"Book club 2026"},
				Rating:   4.5,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2026, time.August, 2),
				AddedAt:  sampleDate(2026, time.July, 20),
			},
			{
				Title:    "Klara and the Sun",
				Authors:  []string{"Kazuo Ishiguro"},
				CoverURL: openLibraryCover(10673548),
				Meta:     "Hardback · 303 pages · 2021",
				Status:   readStatusUnread,
				AddedAt:  sampleDate(2026, time.June, 12),
			},
			{
				Title:    "The Catcher in the Rye",
				Authors:  []string{"J.D. Salinger"},
				CoverURL: openLibraryCover(15172531),
				Meta:     "Paperback · 277 pages · 1951",
				Status:   readStatusUnread,
				AddedAt:  sampleDate(2026, time.May, 30),
			},
			{
				Title:    "Sapiens: A Brief History of Humankind",
				Authors:  []string{"Yuval Noah Harari"},
				CoverURL: openLibraryCover(8594920),
				Meta:     "Paperback · 443 pages · 2015",
				Status:   readStatusAbandoned,
				ReadNote: "Stopped",
				ReadAt:   sampleDatePtr(2026, time.May, 9),
				AddedAt:  sampleDate(2026, time.April, 18),
			},
			{
				Title:    "Die Vermessung der Welt",
				Lang:     "de",
				Authors:  []string{"Daniel Kehlmann"},
				CoverURL: openLibraryCover(10252320),
				Meta:     "Paperback · 302 pages · 2005",
				Lists:    []string{"Book club 2026"},
				Rating:   4,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2026, time.April, 9),
				AddedAt:  sampleDate(2026, time.March, 1),
			},
			{
				Title:    "The Name of the Wind",
				Series:   "The Kingkiller Chronicle #1",
				Authors:  []string{"Patrick Rothfuss"},
				CoverURL: openLibraryCover(8259445),
				Meta:     "Hardback · 662 pages · 2007",
				Lists:    []string{"Favourites", "To re-read"},
				Rating:   5,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2026, time.March, 14),
				AddedAt:  sampleDate(2026, time.January, 3),
			},
			{
				Title:    "The Pragmatic Programmer",
				Authors:  []string{"David Thomas", "Andrew Hunt"},
				CoverURL: openLibraryCover(10143650),
				Meta:     "Hardback · 352 pages · 2019 · 20th Anniversary Edition",
				Lists:    []string{"To re-read"},
				Rating:   4.5,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2026, time.February, 2),
				AddedAt:  sampleDate(2025, time.December, 10),
			},
			{
				Title:    "Piranesi",
				Authors:  []string{"Susanna Clarke"},
				CoverURL: openLibraryCover(10226290),
				Meta:     "Hardback · 272 pages · 2020",
				Lists:    []string{"Favourites", "Lent out"},
				Rating:   5,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2025, time.November, 20),
				AddedAt:  sampleDate(2025, time.October, 5),
			},
			{
				Title:     "Faust. Eine Tragödie",
				Lang:      "de",
				Authors:   []string{"Johann Wolfgang von Goethe"},
				CoverTone: "oxblood",
				Meta:      "Hardback · 1932 · Living room, shelf 2",
				Status:    readStatusUnread,
				AddedAt:   sampleDate(2025, time.September, 1),
			},
			{
				Title:     "Buddenbrooks",
				Lang:      "de",
				Authors:   []string{"Thomas Mann"},
				CoverTone: "green",
				Meta:      "Hardback · 759 pages · 1960 · Living room, shelf 2",
				Status:    readStatusUnread,
				AddedAt:   sampleDate(2025, time.September, 1),
			},
			{
				Title:    "Thinking, Fast and Slow",
				Authors:  []string{"Daniel Kahneman"},
				CoverURL: openLibraryCover(7889800),
				Meta:     "Paperback · 499 pages · 2011",
				Rating:   4,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2025, time.June, 30),
				AddedAt:  sampleDate(2025, time.June, 2),
			},
			{
				Title:    "The Hobbit",
				Authors:  []string{"J.R.R. Tolkien"},
				CoverURL: openLibraryCover(11065702),
				Meta:     "Paperback · 300 pages · 1937",
				Lists:    []string{"Favourites"},
				Rating:   4,
				Status:   readStatusRead,
				ReadAt:   sampleDatePtr(2024, time.December, 24),
				AddedAt:  sampleDate(2024, time.November, 30),
			},
		},
	}
}

func openLibraryCover(id int) string {
	return fmt.Sprintf("https://covers.openlibrary.org/b/id/%d-M.jpg", id)
}

func sampleDate(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func sampleDatePtr(year int, month time.Month, day int) *time.Time {
	date := sampleDate(year, month, day)
	return &date
}
