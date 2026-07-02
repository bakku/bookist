package authors_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/testsupport"
)

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

func TestServiceCreateRequiresName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := authors.NewService(authors.NewSQLiteRepository(db))

	_, err := service.Create(context.Background(), authors.CreateAuthorRequest{Name: " "})
	if !errors.Is(err, authors.ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}
	testsupport.AssertAuthorCount(t, db, 0)
}

func TestServiceListDelegates(t *testing.T) {
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
