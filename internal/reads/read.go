package reads

import "time"

type Read struct {
	ID         string    `json:"id"`
	BookID     string    `json:"book_id"`
	StartedAt  *string   `json:"started_at"`
	FinishedAt *string   `json:"finished_at"`
	Rating     *float64  `json:"rating"`
	Notes      *string   `json:"notes"`
	CreatedAt  time.Time `json:"-"`
	UpdatedAt  time.Time `json:"-"`
}

type CreateReadRequest struct {
	StartedAt  *string  `json:"started_at"`
	FinishedAt *string  `json:"finished_at"`
	Rating     *float64 `json:"rating"`
	Notes      *string  `json:"notes"`
}

type ListItem struct {
	ID         string   `json:"id"`
	StartedAt  *string  `json:"started_at"`
	FinishedAt *string  `json:"finished_at"`
	Rating     *float64 `json:"rating"`
	Notes      *string  `json:"notes"`
}
