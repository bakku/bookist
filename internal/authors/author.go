package authors

import (
	"time"

	"bakku.dev/bookist/internal/optional"
)

type Author struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateAuthorRequest struct {
	Name string `json:"name"`
}

type UpdateAuthorRequest struct {
	Name optional.Value[string] `json:"name"`
}
