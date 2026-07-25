package authors

import "context"

type Repository interface {
	Create(ctx context.Context, input CreateAuthorRequest) (Author, error)
	List(ctx context.Context) ([]Author, error)
	Search(ctx context.Context, query string) ([]Author, error)
	GetByIDs(ctx context.Context, ids []int64) ([]Author, error)
	ListByBookIDs(ctx context.Context, bookIDs []int64) (map[int64][]Author, error)
	Delete(ctx context.Context, id int64) error
}
