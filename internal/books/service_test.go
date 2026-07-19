package books_test

import (
	"context"
	"errors"
	"math"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceCreateConvertsBlankISBNToPersistedNull(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	isbn := " "

	created, err := service.Create(context.Background(), books.CreateBookRequest{Title: "Dune", ISBN: &isbn})
	if err != nil {
		t.Fatal(err)
	}
	if created.ISBN != nil {
		t.Fatalf("expected nil ISBN, got %#v", created.ISBN)
	}
	testsupport.AssertBookRow(t, db, created.ID, "Dune", nil)
}

func TestServiceCreateRequiresTitle(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	_, err := service.Create(context.Background(), books.CreateBookRequest{Title: " "})
	if !errors.Is(err, books.ErrTitleRequired) {
		t.Fatalf("expected ErrTitleRequired, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateRejectsUnknownAuthorIDs(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	_, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:     "Test",
		AuthorIDs: []string{uuid.NewString()},
	})
	if !errors.Is(err, books.ErrAuthorNotFound) {
		t.Fatalf("expected ErrAuthorNotFound, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateTrimsAndPersistsInput(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	isbn := " 9783161484100 "
	lang := " EN "
	pub := " O'Reilly "
	ed := " 2nd "
	format := books.FormatPaperback
	purchased := " 2025-06-15 "
	pages := 400
	notes := " Great "
	summary := " A practical Go guide "
	seriesName := " Programming Languages "
	seriesPosition := 1.5
	location := " Office shelf "
	condition := books.Condition(" very_good ")
	acquisitionSource := " Local bookstore "
	year := 2024
	month := 6
	day := 15

	created, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:             " The Go Programming Language ",
		ISBN:              &isbn,
		Language:          &lang,
		Publisher:         &pub,
		Edition:           &ed,
		Format:            &format,
		PurchasedAt:       &purchased,
		Pages:             &pages,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		SeriesPosition:    &seriesPosition,
		Location:          &location,
		Condition:         &condition,
		AcquisitionSource: &acquisitionSource,
		PublishedYear:     &year,
		PublishedMonth:    &month,
		PublishedDay:      &day,
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.Title != "The Go Programming Language" {
		t.Fatalf("expected trimmed title, got %q", created.Title)
	}

	if created.ISBN == nil || *created.ISBN != "9783161484100" {
		t.Fatalf("expected trimmed ISBN, got %#v", created.ISBN)
	}

	if created.Language == nil || *created.Language != "EN" {
		t.Fatalf("expected trimmed Language 'EN', got %#v", created.Language)
	}

	if created.Publisher == nil || *created.Publisher != "O'Reilly" {
		t.Fatalf("expected trimmed Publisher 'O\\'Reilly', got %#v", created.Publisher)
	}

	if created.Edition == nil || *created.Edition != "2nd" {
		t.Fatalf("expected trimmed Edition '2nd', got %#v", created.Edition)
	}

	if created.Format == nil || *created.Format != books.FormatPaperback {
		t.Fatalf("expected Format paperback, got %#v", created.Format)
	}

	if created.PurchasedAt == nil || *created.PurchasedAt != "2025-06-15" {
		t.Fatalf("expected trimmed PurchasedAt '2025-06-15', got %#v", created.PurchasedAt)
	}

	if created.Pages == nil || *created.Pages != 400 {
		t.Fatalf("expected Pages 400, got %#v", created.Pages)
	}

	if created.Notes == nil || *created.Notes != "Great" {
		t.Fatalf("expected trimmed Notes 'Great', got %#v", created.Notes)
	}

	if created.Summary == nil || *created.Summary != "A practical Go guide" {
		t.Fatalf("expected trimmed Summary, got %#v", created.Summary)
	}

	if created.SeriesName == nil || *created.SeriesName != "Programming Languages" {
		t.Fatalf("expected trimmed SeriesName, got %#v", created.SeriesName)
	}

	if created.SeriesPosition == nil || *created.SeriesPosition != 1.5 {
		t.Fatalf("expected fractional SeriesPosition, got %#v", created.SeriesPosition)
	}

	if created.Location == nil || *created.Location != "Office shelf" {
		t.Fatalf("expected trimmed Location, got %#v", created.Location)
	}

	if created.Condition == nil || *created.Condition != books.ConditionVeryGood {
		t.Fatalf("expected trimmed Condition very_good, got %#v", created.Condition)
	}

	if created.AcquisitionSource == nil || *created.AcquisitionSource != "Local bookstore" {
		t.Fatalf("expected trimmed AcquisitionSource, got %#v", created.AcquisitionSource)
	}

	if created.PublishedYear == nil || *created.PublishedYear != 2024 {
		t.Fatalf("expected PublishedYear 2024, got %#v", created.PublishedYear)
	}

	if created.PublishedMonth == nil || *created.PublishedMonth != 6 {
		t.Fatalf("expected PublishedMonth 6, got %#v", created.PublishedMonth)
	}

	if created.PublishedDay == nil || *created.PublishedDay != 15 {
		t.Fatalf("expected PublishedDay 15, got %#v", created.PublishedDay)
	}

	f := string(books.FormatPaperback)
	c := string(books.ConditionVeryGood)
	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title:             "The Go Programming Language",
		ISBN:              new("9783161484100"),
		Language:          new("EN"),
		Publisher:         new("O'Reilly"),
		Edition:           new("2nd"),
		Format:            &f,
		PurchasedAt:       new("2025-06-15"),
		Pages:             new(400),
		Notes:             new("Great"),
		Summary:           new("A practical Go guide"),
		SeriesName:        new("Programming Languages"),
		SeriesPosition:    new(1.5),
		Location:          new("Office shelf"),
		Condition:         &c,
		AcquisitionSource: new("Local bookstore"),
		PublishedYear:     new(2024),
		PublishedMonth:    new(6),
		PublishedDay:      new(15),
	})
}

func TestServiceCreateWithNoAuthorsReturnsEmptyNonNilSlice(t *testing.T) {
	service, _ := testsupport.NewBookService(t)

	created, err := service.Create(context.Background(), books.CreateBookRequest{Title: "No Authors"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Authors == nil {
		t.Fatal("expected non-nil Authors slice")
	}
	if len(created.Authors) != 0 {
		t.Fatalf("expected empty Authors slice, got %d", len(created.Authors))
	}
}

func TestServiceCreateWithValidAuthorIDsReturnsHydratedAuthors(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	author := testsupport.InsertAuthorRow(t, db, "Test Author")
	created, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:     "Test Book",
		AuthorIDs: []string{author},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(created.Authors))
	}
	if created.Authors[0].Name != "Test Author" {
		t.Fatalf("expected author name 'Test Author', got %q", created.Authors[0].Name)
	}
	testsupport.AssertBookAuthors(t, db, created.ID, author)
}

func TestServiceCreateAcceptsValidFormat(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	format := books.FormatPaperback
	created, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:  "Paperback Book",
		Format: &format,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Format == nil || *created.Format != books.FormatPaperback {
		t.Fatalf("expected Format paperback, got %#v", created.Format)
	}
	f := string(books.FormatPaperback)
	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title:  "Paperback Book",
		Format: &f,
	})
}

func TestServiceCreateRejectsInvalidFormat(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	format := books.Format("invalid")
	_, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:  "Bad Format",
		Format: &format,
	})
	if !errors.Is(err, books.ErrInvalidFormat) {
		t.Fatalf("expected ErrInvalidFormat, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateConvertsBlankStringFieldsToNull(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	lang := " "
	pub := " "
	ed := " "
	notes := " "
	summary := " "
	seriesName := " "
	location := " "
	acquisitionSource := " "
	purchased := " "

	created, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:             "Blank Fields",
		Language:          &lang,
		Publisher:         &pub,
		Edition:           &ed,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		Location:          &location,
		AcquisitionSource: &acquisitionSource,
		PurchasedAt:       &purchased,
	})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title: "Blank Fields",
	})
}

func TestServiceCreateRejectsInvalidCondition(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	for _, condition := range []books.Condition{"", " ", "like_new"} {
		_, err := service.Create(context.Background(), books.CreateBookRequest{
			Title:     "Bad Condition " + string(condition),
			Condition: &condition,
		})
		if !errors.Is(err, books.ErrInvalidCondition) {
			t.Fatalf("expected ErrInvalidCondition for %q, got %v", condition, err)
		}
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateAcceptsValidConditions(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	for _, condition := range []books.Condition{
		books.ConditionNew,
		books.ConditionVeryGood,
		books.ConditionGood,
		books.ConditionAcceptable,
		books.ConditionPoor,
	} {
		created, err := service.Create(context.Background(), books.CreateBookRequest{
			Title:     "Condition " + string(condition),
			Condition: &condition,
		})
		if err != nil {
			t.Fatalf("expected condition %q to be accepted: %v", condition, err)
		}
		if created.Condition == nil || *created.Condition != condition {
			t.Fatalf("expected condition %q, got %#v", condition, created.Condition)
		}
	}
	testsupport.AssertBookCount(t, db, 5)
}

func TestServiceCreateRejectsInvalidNumericFields(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	for _, test := range []struct {
		name  string
		input books.CreateBookRequest
		want  error
	}{
		{name: "pages", input: books.CreateBookRequest{Title: "Bad Pages", Pages: new(0)}, want: books.ErrInvalidPages},
		{name: "zero series position", input: books.CreateBookRequest{Title: "Bad Position", SeriesPosition: new(0.0)}, want: books.ErrInvalidSeriesPosition},
		{name: "negative series position", input: books.CreateBookRequest{Title: "Bad Position", SeriesPosition: new(-1.5)}, want: books.ErrInvalidSeriesPosition},
		{name: "non-finite series position", input: books.CreateBookRequest{Title: "Bad Position", SeriesPosition: new(math.Inf(1))}, want: books.ErrInvalidSeriesPosition},
		{name: "year", input: books.CreateBookRequest{Title: "Bad Year", PublishedYear: new(0)}, want: books.ErrInvalidPublishedYear},
		{name: "month without year", input: books.CreateBookRequest{Title: "Bad Month", PublishedMonth: new(1)}, want: books.ErrInvalidPublishedMonth},
		{name: "invalid month", input: books.CreateBookRequest{Title: "Bad Month", PublishedYear: new(2024), PublishedMonth: new(13)}, want: books.ErrInvalidPublishedMonth},
		{name: "day without month", input: books.CreateBookRequest{Title: "Bad Day", PublishedYear: new(2024), PublishedDay: new(1)}, want: books.ErrInvalidPublishedDay},
		{name: "invalid calendar day", input: books.CreateBookRequest{Title: "Bad Day", PublishedYear: new(2023), PublishedMonth: new(2), PublishedDay: new(29)}, want: books.ErrInvalidPublishedDay},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := service.Create(context.Background(), test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}
		})
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateRejectsInvalidPurchasedAt(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	_, err := service.Create(context.Background(), books.CreateBookRequest{Title: "Bad Date", PurchasedAt: new("2025-02-29")})
	if !errors.Is(err, books.ErrInvalidPurchasedAt) {
		t.Fatalf("expected ErrInvalidPurchasedAt, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestServiceListHydratesAuthors(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	bookID := testsupport.InsertBookRow(t, db, "Book With Author", nil)
	author := testsupport.InsertAuthorRow(t, db, "Author One")
	testsupport.InsertBookAuthorRow(t, db, bookID, author)

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if len(listed[0].Authors) != 1 {
		t.Fatalf("expected 1 author on listed book, got %d", len(listed[0].Authors))
	}
	if listed[0].Authors[0].Name != "Author One" {
		t.Fatalf("expected Author One, got %q", listed[0].Authors[0].Name)
	}
}

func TestServiceListReturnsPersistedBooks(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	repository := books.NewSQLiteRepository(db)

	isbn := "9783161484100"
	lang := "en"
	pub := "O'Reilly"
	ed := "2nd"
	format := books.FormatPaperback
	purchased := "2025-06-15"
	pages := 400
	notes := "Great book"
	year := 2024
	month := 6
	day := 15

	created, err := repository.Create(context.Background(), books.CreateBookRequest{
		Title:          "Kindred",
		ISBN:           &isbn,
		Language:       &lang,
		Publisher:      &pub,
		Edition:        &ed,
		Format:         &format,
		PurchasedAt:    &purchased,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &year,
		PublishedMonth: &month,
		PublishedDay:   &day,
	})
	if err != nil {
		t.Fatal(err)
	}

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}

	got := listed[0]

	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	if got.Title != "Kindred" {
		t.Fatalf("expected Kindred, got %q", got.Title)
	}

	if got.ISBN == nil || *got.ISBN != isbn {
		t.Fatalf("expected ISBN %q, got %#v", isbn, got.ISBN)
	}

	if got.Language == nil || *got.Language != lang {
		t.Fatalf("expected Language %q, got %#v", lang, got.Language)
	}

	if got.Publisher == nil || *got.Publisher != pub {
		t.Fatalf("expected Publisher %q, got %#v", pub, got.Publisher)
	}

	if got.Edition == nil || *got.Edition != ed {
		t.Fatalf("expected Edition %q, got %#v", ed, got.Edition)
	}

	if got.Format == nil || *got.Format != format {
		t.Fatalf("expected Format %q, got %#v", format, got.Format)
	}

	if got.PurchasedAt == nil || *got.PurchasedAt != purchased {
		t.Fatalf("expected PurchasedAt %q, got %#v", purchased, got.PurchasedAt)
	}

	if got.Pages == nil || *got.Pages != pages {
		t.Fatalf("expected Pages %d, got %#v", pages, got.Pages)
	}

	if got.Notes == nil || *got.Notes != notes {
		t.Fatalf("expected Notes %q, got %#v", notes, got.Notes)
	}

	if got.PublishedYear == nil || *got.PublishedYear != year {
		t.Fatalf("expected PublishedYear %d, got %#v", year, got.PublishedYear)
	}

	if got.PublishedMonth == nil || *got.PublishedMonth != month {
		t.Fatalf("expected PublishedMonth %d, got %#v", month, got.PublishedMonth)
	}

	if got.PublishedDay == nil || *got.PublishedDay != day {
		t.Fatalf("expected PublishedDay %d, got %#v", day, got.PublishedDay)
	}
}
