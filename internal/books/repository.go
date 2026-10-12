package books

import (
	"context"

	"bakku.dev/bookist/internal/optional"
)

type Repository interface {
	List(ctx context.Context) ([]Book, error)
	Search(ctx context.Context, query string) ([]Book, error)
	ListByListID(ctx context.Context, listID int64) ([]Book, error)
	SearchByListID(ctx context.Context, listID int64, query string) ([]Book, error)
	ListExists(ctx context.Context, listID int64) (bool, error)
	GetByID(ctx context.Context, id int64) (Book, error)
	Create(ctx context.Context, params CreateParams) (Book, error)
	Update(ctx context.Context, id int64, params UpdateParams) (UpdateResult, error)
	Delete(ctx context.Context, id int64) (*string, error)
}

type UpdateResult struct {
	Book               Book
	PriorCoverImageKey *string
}

// CreateParams is the repository input for a new book. Unlike
// CreateBookRequest it holds the stored cover's key rather than the cover bytes.
type CreateParams struct {
	Title             string
	ISBN              *string
	AuthorIDs         []int64
	Language          *string
	Publisher         *string
	Edition           *string
	Format            *Format
	PurchasedAt       *string
	PurchasePrice     *string
	Pages             *int
	Notes             *string
	Summary           *string
	SeriesName        *string
	SeriesPosition    *float64
	Location          *string
	Condition         *Condition
	AcquisitionSource *string
	PublishedYear     *int
	PublishedMonth    *int
	PublishedDay      *int
	CoverImageKey     *string
}

// UpdateParams is the repository input for changing a book. Only present
// fields are written. Unlike UpdateBookRequest it holds the stored cover's key
// rather than the cover bytes.
type UpdateParams struct {
	Title             optional.Value[string]
	ISBN              optional.Value[string]
	AuthorIDs         optional.Value[[]int64]
	Language          optional.Value[string]
	Publisher         optional.Value[string]
	Edition           optional.Value[string]
	Format            optional.Value[Format]
	PurchasedAt       optional.Value[string]
	PurchasePrice     optional.Value[string]
	Pages             optional.Value[int]
	Notes             optional.Value[string]
	Summary           optional.Value[string]
	SeriesName        optional.Value[string]
	SeriesPosition    optional.Value[float64]
	Location          optional.Value[string]
	Condition         optional.Value[Condition]
	AcquisitionSource optional.Value[string]
	PublishedYear     optional.Value[int]
	PublishedMonth    optional.Value[int]
	PublishedDay      optional.Value[int]
	CoverImageKey     optional.Value[string]
}
