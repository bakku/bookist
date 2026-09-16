package books

import (
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/optional"
)

type Format string

type Condition string

const (
	FormatHardback  Format = "hardback"
	FormatPaperback Format = "paperback"
	FormatEpub      Format = "epub"

	ConditionNew        Condition = "new"
	ConditionVeryGood   Condition = "very_good"
	ConditionGood       Condition = "good"
	ConditionAcceptable Condition = "acceptable"
	ConditionPoor       Condition = "poor"
)

type Book struct {
	ID                int64            `json:"id"`
	Title             string           `json:"title"`
	ISBN              *string          `json:"isbn"`
	Authors           []authors.Author `json:"authors"`
	Language          *string          `json:"language"`
	Publisher         *string          `json:"publisher"`
	Edition           *string          `json:"edition"`
	Format            *Format          `json:"format"`
	PurchasedAt       *string          `json:"purchased_at"`
	PurchasePrice     *string          `json:"purchase_price"`
	Pages             *int             `json:"pages"`
	Notes             *string          `json:"notes"`
	Summary           *string          `json:"summary"`
	SeriesName        *string          `json:"series_name"`
	SeriesPosition    *float64         `json:"series_position"`
	Location          *string          `json:"location"`
	Condition         *Condition       `json:"condition"`
	AcquisitionSource *string          `json:"acquisition_source"`
	// Separate nullable components preserve partial dates such as a publication year alone.
	PublishedYear  *int      `json:"published_year"`
	PublishedMonth *int      `json:"published_month"`
	PublishedDay   *int      `json:"published_day"`
	CoverImageKey  *string   `json:"-"`
	CoverURL       *string   `json:"cover_url"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateBookRequest struct {
	Title             string     `json:"title"`
	ISBN              *string    `json:"isbn"`
	AuthorIDs         []int64    `json:"author_ids"`
	Language          *string    `json:"language"`
	Publisher         *string    `json:"publisher"`
	Edition           *string    `json:"edition"`
	Format            *Format    `json:"format"`
	PurchasedAt       *string    `json:"purchased_at"`
	PurchasePrice     *string    `json:"purchase_price"`
	Pages             *int       `json:"pages"`
	Notes             *string    `json:"notes"`
	Summary           *string    `json:"summary"`
	SeriesName        *string    `json:"series_name"`
	SeriesPosition    *float64   `json:"series_position"`
	Location          *string    `json:"location"`
	Condition         *Condition `json:"condition"`
	AcquisitionSource *string    `json:"acquisition_source"`
	PublishedYear     *int       `json:"published_year"`
	PublishedMonth    *int       `json:"published_month"`
	PublishedDay      *int       `json:"published_day"`
	Cover             *[]byte    `json:"cover"`
	CoverImageKey     *string    `json:"-"`
}

type UpdateBookRequest struct {
	Title             optional.Value[string]    `json:"title"`
	ISBN              optional.Value[string]    `json:"isbn"`
	AuthorIDs         optional.Value[[]int64]   `json:"author_ids"`
	Language          optional.Value[string]    `json:"language"`
	Publisher         optional.Value[string]    `json:"publisher"`
	Edition           optional.Value[string]    `json:"edition"`
	Format            optional.Value[Format]    `json:"format"`
	PurchasedAt       optional.Value[string]    `json:"purchased_at"`
	PurchasePrice     optional.Value[string]    `json:"purchase_price"`
	Pages             optional.Value[int]       `json:"pages"`
	Notes             optional.Value[string]    `json:"notes"`
	Summary           optional.Value[string]    `json:"summary"`
	SeriesName        optional.Value[string]    `json:"series_name"`
	SeriesPosition    optional.Value[float64]   `json:"series_position"`
	Location          optional.Value[string]    `json:"location"`
	Condition         optional.Value[Condition] `json:"condition"`
	AcquisitionSource optional.Value[string]    `json:"acquisition_source"`
	PublishedYear     optional.Value[int]       `json:"published_year"`
	PublishedMonth    optional.Value[int]       `json:"published_month"`
	PublishedDay      optional.Value[int]       `json:"published_day"`
	Cover             optional.Value[[]byte]    `json:"cover"`
	CoverImageKey     optional.Value[string]    `json:"-"`
}

func (r UpdateBookRequest) hasFields() bool {
	return r.Title.Present || r.ISBN.Present || r.AuthorIDs.Present || r.Language.Present ||
		r.Publisher.Present || r.Edition.Present || r.Format.Present || r.PurchasedAt.Present ||
		r.PurchasePrice.Present || r.Pages.Present || r.Notes.Present || r.Summary.Present ||
		r.SeriesName.Present || r.SeriesPosition.Present || r.Location.Present ||
		r.Condition.Present || r.AcquisitionSource.Present || r.PublishedYear.Present ||
		r.PublishedMonth.Present || r.PublishedDay.Present || r.Cover.Present
}
