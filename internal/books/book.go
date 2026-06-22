package books

import "time"

type Book struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	ISBN      *string   `json:"isbn"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateBookInput struct {
	Title string  `json:"title"`
	ISBN  *string `json:"isbn"`
}
