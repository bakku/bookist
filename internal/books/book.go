package books

import (
	"time"

	"bakku.dev/bookist/internal/authors"
)

type Book struct {
	ID        string           `json:"id"`
	Title     string           `json:"title"`
	ISBN      *string          `json:"isbn"`
	Authors   []authors.Author `json:"authors"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

type CreateBookRequest struct {
	Title     string   `json:"title"`
	ISBN      *string  `json:"isbn"`
	AuthorIDs []string `json:"author_ids"`
}
