package books

import (
	"time"

	"bakku.dev/bookist/internal/authors"
)

type Format string

const (
	FormatHardback  Format = "hardback"
	FormatPaperback Format = "paperback"
	FormatEpub      Format = "epub"
)

type Book struct {
	ID             string           `json:"id"`
	Title          string           `json:"title"`
	ISBN           *string          `json:"isbn"`
	Authors        []authors.Author `json:"authors"`
	Language       *string          `json:"language"`
	Publisher      *string          `json:"publisher"`
	Edition        *string          `json:"edition"`
	Format         *Format          `json:"format"`
	PurchasedAt    *string          `json:"purchased_at"`
	Pages          *int             `json:"pages"`
	Notes          *string          `json:"notes"`
	PublishedYear  *int             `json:"published_year"`
	PublishedMonth *int             `json:"published_month"`
	PublishedDay   *int             `json:"published_day"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

type CreateBookRequest struct {
	Title          string   `json:"title"`
	ISBN           *string  `json:"isbn"`
	AuthorIDs      []string `json:"author_ids"`
	Language       *string  `json:"language"`
	Publisher      *string  `json:"publisher"`
	Edition        *string  `json:"edition"`
	Format         *Format  `json:"format"`
	PurchasedAt    *string  `json:"purchased_at"`
	Pages          *int     `json:"pages"`
	Notes          *string  `json:"notes"`
	PublishedYear  *int     `json:"published_year"`
	PublishedMonth *int     `json:"published_month"`
	PublishedDay   *int     `json:"published_day"`
}
