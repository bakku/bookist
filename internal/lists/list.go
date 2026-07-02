package lists

import "time"

type List struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type CreateListRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

type AddBookToListRequest struct {
	BookID string `json:"book_id"`
}
