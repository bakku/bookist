package lists_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ── Constraint Errors ─────────────────────────────────────────────────────────

func TestCreateNameConflict(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	testsupport.InsertListRow(t, db, "Favorites")

	_, err := lists.NewSQLiteRepository(db).Create(context.Background(), lists.CreateListRequest{Name: "favorites"})
	if !errors.Is(err, lists.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 1, `SELECT count(*) FROM lists`)
}

func TestAddBookLookupError(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	listID := testsupport.InsertListRow(t, db, "Favorites")
	if _, err := db.Exec(`UPDATE lists SET created_at = 'invalid' WHERE id = ?`, listID); err != nil {
		t.Fatal(err)
	}
	const missingBookID = int64(999999)

	err := lists.NewSQLiteRepository(db).AddBookToList(context.Background(), listID, missingBookID)
	var parseErr *time.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected lookup parse error, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT count(*) FROM book_lists`)
}

func TestAddBookDatabaseError(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	listID := testsupport.InsertListRow(t, db, "Favorites")
	if _, err := db.Exec(`ALTER TABLE lists RENAME COLUMN description TO unavailable_description`); err != nil {
		t.Fatal(err)
	}
	const missingBookID = int64(999999)

	err := lists.NewSQLiteRepository(db).AddBookToList(context.Background(), listID, missingBookID)
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_ERROR {
		t.Fatalf("expected lookup database error, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT count(*) FROM book_lists`)
}

func TestListConstraintLookalikes(t *testing.T) {
	type operation int
	const (
		create operation = iota
		update
		addBook
	)

	for _, test := range []struct {
		name    string
		op      operation
		trigger string
	}{
		{name: "create", op: create, trigger: `CREATE TRIGGER blocked BEFORE INSERT ON lists BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed'); END`},
		{name: "update", op: update, trigger: `CREATE TRIGGER blocked BEFORE UPDATE ON lists BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed'); END`},
		{name: "membership unique", op: addBook, trigger: `CREATE TRIGGER blocked BEFORE INSERT ON book_lists BEGIN SELECT RAISE(ABORT, 'UNIQUE constraint failed'); END`},
		{name: "membership foreign key", op: addBook, trigger: `CREATE TRIGGER blocked BEFORE INSERT ON book_lists BEGIN SELECT RAISE(ABORT, 'FOREIGN KEY constraint failed'); END`},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			listID := testsupport.InsertListRow(t, db, "Favorites")
			bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
			if _, err := db.Exec(test.trigger); err != nil {
				t.Fatal(err)
			}

			repository := lists.NewSQLiteRepository(db)
			var err error
			switch test.op {
			case create:
				_, err = repository.Create(context.Background(), lists.CreateListRequest{Name: "New"})
			case update:
				_, err = repository.Update(context.Background(), listID, lists.UpdateListRequest{
					Name: optional.Value[string]{Present: true, Value: new("Changed")},
				})
			case addBook:
				err = repository.AddBookToList(context.Background(), listID, bookID)
			}

			var sqliteErr *sqlite.Error
			if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_TRIGGER {
				t.Fatalf("expected unchanged trigger error, got %v", err)
			}
		})
	}
}
