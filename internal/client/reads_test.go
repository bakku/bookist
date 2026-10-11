package client_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Reads List ────────────────────────────────────────────────────────────────

func TestReadsListReturnsReadsForBook(t *testing.T) {
	c, db := newTestClient(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, db, "Kindred", nil)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 1, BookID: bookID, FinishedAt: new("2026-03-01"), Rating: new(4.5), CreatedAt: "2026-03-01T00:00:00Z"})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 2, BookID: otherBookID, CreatedAt: "2026-03-02T00:00:00Z"})

	listed, err := c.Reads.List(context.Background(), bookID)
	if err != nil {
		t.Fatal(err)
	}

	if len(listed) != 1 || listed[0].ID != 1 || listed[0].BookID != bookID ||
		listed[0].FinishedAt == nil || *listed[0].FinishedAt != "2026-03-01" ||
		listed[0].Rating == nil || *listed[0].Rating != 4.5 {
		t.Fatalf("expected only the read of book %d, got %#v", bookID, listed)
	}
}

// ── Reads Get ─────────────────────────────────────────────────────────────────

func TestReadsGetReturnsRead(t *testing.T) {
	c, db := newTestClient(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 8, BookID: bookID, CreatedAt: "2026-02-01T00:00:00Z"})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: 9, BookID: bookID, StartedAt: new("2026-03-01"), Rating: new(4.5), Notes: new("Great"),
		CreatedAt: "2026-03-01T00:00:00Z",
	})

	got, err := c.Reads.Get(context.Background(), 9)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 9 || got.BookID != bookID || got.StartedAt == nil || *got.StartedAt != "2026-03-01" ||
		got.Rating == nil || *got.Rating != 4.5 || got.Notes == nil || *got.Notes != "Great" {
		t.Fatalf("unexpected read %#v", got)
	}
}

func TestReadsGetMissingReadReturnsNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	_, err := c.Reads.Get(context.Background(), 999)

	assertClientError(t, err, 404, "read_not_found", "")
}

// ── Reads Create ──────────────────────────────────────────────────────────────

func TestReadsCreatePersistsRead(t *testing.T) {
	c, db := newTestClient(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	created, err := c.Reads.Create(context.Background(), bookID, reads.CreateReadRequest{
		StartedAt: new("2026-02-01"),
		Rating:    new(4.5),
		Notes:     new("Loved it"),
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == 0 || created.BookID != bookID {
		t.Fatalf("unexpected created read %#v", created)
	}
	testsupport.AssertReadRow(t, db, created.ID, testsupport.ReadRowAssertion{
		BookID:    bookID,
		StartedAt: new("2026-02-01"),
		Rating:    new(4.5),
		Notes:     new("Loved it"),
	})
}

func TestReadsCreateForMissingBookReturnsNotFound(t *testing.T) {
	c, db := newTestClient(t)

	_, err := c.Reads.Create(context.Background(), 999, reads.CreateReadRequest{})

	assertClientError(t, err, 404, "book_not_found", "")
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM reads`)
}

// ── Reads Update ──────────────────────────────────────────────────────────────

func TestReadsUpdateSetsAndClearsFields(t *testing.T) {
	c, db := newTestClient(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 9, BookID: bookID, StartedAt: new("2026-02-01"), Rating: new(3.0), Notes: new("Slow start"), CreatedAt: "2026-03-01T00:00:00Z"})

	updated, err := c.Reads.Update(context.Background(), 9, reads.UpdateReadRequest{
		Rating: optional.Set(4.5),
		Notes:  optional.Null[string](),
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.ID != 9 || updated.Rating == nil || *updated.Rating != 4.5 || updated.Notes != nil {
		t.Fatalf("unexpected updated read %#v", updated)
	}
	testsupport.AssertReadRow(t, db, 9, testsupport.ReadRowAssertion{
		BookID:    bookID,
		StartedAt: new("2026-02-01"),
		Rating:    new(4.5),
	})
}

// ── Reads Delete ──────────────────────────────────────────────────────────────

func TestReadsDeleteRemovesRead(t *testing.T) {
	c, db := newTestClient(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 9, BookID: bookID, CreatedAt: "2026-03-01T00:00:00Z"})

	if err := c.Reads.Delete(context.Background(), 9); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM reads`)
}

func TestReadsDeleteMissingReadReturnsNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	err := c.Reads.Delete(context.Background(), 999)

	assertClientError(t, err, 404, "read_not_found", "")
}
