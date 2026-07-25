package reads

import "context"

type Repository interface {
	Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error)
	ListByBookID(ctx context.Context, bookID int64) ([]Read, error)
	Delete(ctx context.Context, id int64) error
}
