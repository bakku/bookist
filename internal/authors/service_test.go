package authors_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceCreateRequiresName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))

	_, err := service.Create(context.Background(), authors.CreateAuthorRequest{Name: " "})
	if !errors.Is(err, authors.ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM authors`)
}

func TestServiceCreateTrimsAndPersistsName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))

	created, err := service.Create(context.Background(), authors.CreateAuthorRequest{Name: "  Jane Austen  "})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Jane Austen" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}
	testsupport.AssertAuthorRow(t, db, created.ID, "Jane Austen")
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestServiceUpdateTrimsAndPersistsName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	id := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	name := "  Octavia Butler  "

	updated, err := service.Update(context.Background(), id, authors.UpdateAuthorRequest{
		Name: optional.Value[string]{Present: true, Value: &name},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Octavia Butler" {
		t.Fatalf("expected trimmed name, got %q", updated.Name)
	}
	testsupport.AssertAuthorRow(t, db, id, "Octavia Butler")
}

func TestServiceUpdateValidatesFields(t *testing.T) {
	tests := []struct {
		name  string
		input authors.UpdateAuthorRequest
		want  error
	}{
		{name: "empty request", want: authors.ErrNoFieldsToUpdate},
		{name: "null name", input: authors.UpdateAuthorRequest{Name: optional.Value[string]{Present: true}}, want: authors.ErrNameRequired},
		{name: "blank name", input: authors.UpdateAuthorRequest{Name: optional.Value[string]{Present: true, Value: stringPointer(" ")}}, want: authors.ErrNameRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			service := authors.NewService(authors.NewSQLiteRepository(db))
			id := testsupport.InsertAuthorRow(t, db, "Jane Austen")

			_, err := service.Update(context.Background(), id, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			testsupport.AssertAuthorRow(t, db, id, "Jane Austen")
		})
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestServiceListReturnsAuthor(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	testsupport.InsertAuthorRow(t, db, "Jane Austen")

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 author, got %d", len(listed))
	}
	if listed[0].Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", listed[0].Name)
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestServiceSearchTrimsQuery(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	testsupport.InsertAuthorRow(t, db, "Jane Austen")
	testsupport.InsertAuthorRow(t, db, "Octavia Butler")

	matched, err := service.Search(context.Background(), "  AUST  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || matched[0].Name != "Jane Austen" {
		t.Fatalf("expected only Jane Austen, got %#v", matched)
	}
}

// ── GetByIDs ──────────────────────────────────────────────────────────────────

func TestServiceGetByIDsReturnsSpecifiedAuthors(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	janeId := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	testsupport.InsertAuthorRow(t, db, "Octavia Butler")
	dumasId := testsupport.InsertAuthorRow(t, db, "Alexandre Dumas")

	result, err := service.GetByIDs(context.Background(), []int64{janeId, dumasId})
	if err != nil {
		t.Fatal(err)
	}

	if len(result) != 2 {
		t.Fatalf("expected '2' authors, got %d", len(result))
	}

	if result[0].Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", result[0].Name)
	}

	if result[1].Name != "Alexandre Dumas" {
		t.Fatalf("expected Alexandre Dumas, got %q", result[1].Name)
	}
}

// ── ListByBookIDs ─────────────────────────────────────────────────────────────

func TestServiceListByBookIDsReturnsAuthorsGroupedByBook(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	bookID1 := testsupport.InsertBookRow(t, db, "Pride and Prejudice", nil)
	bookID2 := testsupport.InsertBookRow(t, db, "Kindred", nil)
	janeID := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	octaviaID := testsupport.InsertAuthorRow(t, db, "Octavia Butler")
	testsupport.InsertBookAuthorRow(t, db, bookID1, janeID)
	testsupport.InsertBookAuthorRow(t, db, bookID2, octaviaID)

	result, err := service.ListByBookIDs(context.Background(), []int64{bookID1, bookID2})
	if err != nil {
		t.Fatal(err)
	}

	if len(result[bookID1]) != 1 || result[bookID1][0].Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen for Pride and Prejudice, got %#v", result[bookID1])
	}

	if len(result[bookID2]) != 1 || result[bookID2][0].Name != "Octavia Butler" {
		t.Fatalf("expected Octavia Butler for Kindred, got %#v", result[bookID2])
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestServiceDeletePersistsDeletion(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))
	id := testsupport.InsertAuthorRow(t, db, "Jane Austen")

	if err := service.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM authors`)
}

func stringPointer(value string) *string {
	return &value
}
