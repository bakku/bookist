package reads

import "context"

type Repository interface {
	Create(ctx context.Context, bookID int64, input CreateReadRequest) (Read, error)
	GetByID(ctx context.Context, id int64) (Read, error)
	ListByBookID(ctx context.Context, bookID int64) ([]Read, error)
	Update(ctx context.Context, id int64, input UpdateReadRequest) (Read, error)
	Delete(ctx context.Context, id int64) error
}
