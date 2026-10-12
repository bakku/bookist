package authors

import "context"

type Repository interface {
	Create(ctx context.Context, input CreateAuthorRequest) (Author, error)
	Update(ctx context.Context, id int64, input UpdateAuthorRequest) (Author, error)
	List(ctx context.Context) ([]Author, error)
	Search(ctx context.Context, query string) ([]Author, error)
	GetByID(ctx context.Context, id int64) (Author, error)
	GetByIDs(ctx context.Context, ids []int64) ([]Author, error)
	Delete(ctx context.Context, id int64) error
}
