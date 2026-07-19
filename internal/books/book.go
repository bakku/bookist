package books

import (
	"time"

	"bakku.dev/bookist/internal/authors"
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
}
