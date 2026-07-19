package lists

import "time"

type List struct {
	ID          int64     `json:"id"`
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
	BookID int64 `json:"book_id"`
}
